package state

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/kv/memdb"

	"github.com/erigontech/erigon-lib/crypto"
	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/params"
)

// TestIntraBlockStateEIP7702SetDelegation tests setting delegation code on an account
func TestIntraBlockStateEIP7702SetDelegation(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	delegator := libcommon.HexToAddress("0x1111111111111111111111111111111111111111")
	delegateTarget := libcommon.HexToAddress("0x2222222222222222222222222222222222222222")

	r := NewPlainStateReader(tx)
	intraBlockState := New(r)

	// Create the delegator account
	intraBlockState.CreateAccount(delegator, true)
	intraBlockState.AddBalance(delegator, uint256.NewInt(1000000))
	intraBlockState.SetNonce(delegator, 1)

	// Set delegation code
	delegationCode := types.AddressToDelegation(delegateTarget)
	intraBlockState.SetCode(delegator, delegationCode)

	// Verify code is delegation
	code := intraBlockState.GetCode(delegator)
	assert.True(t, types.IsDelegation(code), "code should be delegation")
	assert.Equal(t, delegationCode, code)

	// Verify GetDelegatedDesignation
	dd, ok := intraBlockState.GetDelegatedDesignation(delegator)
	assert.True(t, ok, "should have delegation")
	assert.Equal(t, delegateTarget, dd, "delegation target should match")

	// Verify CodeHash
	expectedCodeHash := crypto.Keccak256Hash(delegationCode)
	assert.Equal(t, expectedCodeHash, intraBlockState.GetCodeHash(delegator))
}

// TestIntraBlockStateEIP7702ResolveCode tests ResolveCode for delegation accounts
func TestIntraBlockStateEIP7702ResolveCode(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	delegator := libcommon.HexToAddress("0x3333333333333333333333333333333333333333")
	delegateTarget := libcommon.HexToAddress("0x4444444444444444444444444444444444444444")
	targetCode := []byte{0x60, 0x00, 0x60, 0x00, 0xf3} // PUSH1 0x00 PUSH1 0x00 RETURN

	r := NewPlainStateReader(tx)
	intraBlockState := New(r)

	// Create the target contract with some code
	intraBlockState.CreateAccount(delegateTarget, true)
	intraBlockState.SetCode(delegateTarget, targetCode)

	// Create the delegator with delegation to target
	intraBlockState.CreateAccount(delegator, true)
	delegationCode := types.AddressToDelegation(delegateTarget)
	intraBlockState.SetCode(delegator, delegationCode)

	// GetCode should return the delegation code
	code := intraBlockState.GetCode(delegator)
	assert.True(t, types.IsDelegation(code))

	// ResolveCode should return the target's code
	resolvedCode := intraBlockState.ResolveCode(delegator)
	assert.Equal(t, targetCode, resolvedCode, "ResolveCode should return target's code")

	// ResolveCodeHash should return target's code hash
	expectedTargetCodeHash := crypto.Keccak256Hash(targetCode)
	resolvedCodeHash := intraBlockState.ResolveCodeHash(delegator)
	assert.Equal(t, expectedTargetCodeHash, resolvedCodeHash)
}

// TestIntraBlockStateEIP7702ChainedDelegation tests that chained delegation is NOT followed
func TestIntraBlockStateEIP7702ChainedDelegation(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	// A -> B -> C (chained delegation)
	addrA := libcommon.HexToAddress("0x5555555555555555555555555555555555555555")
	addrB := libcommon.HexToAddress("0x6666666666666666666666666666666666666666")
	addrC := libcommon.HexToAddress("0x7777777777777777777777777777777777777777")
	targetCode := []byte{0x60, 0x01, 0x60, 0x00, 0x55} // PUSH1 1 PUSH1 0 SSTORE

	r := NewPlainStateReader(tx)
	intraBlockState := New(r)

	// C has actual code
	intraBlockState.CreateAccount(addrC, true)
	intraBlockState.SetCode(addrC, targetCode)

	// B delegates to C
	intraBlockState.CreateAccount(addrB, true)
	intraBlockState.SetCode(addrB, types.AddressToDelegation(addrC))

	// A delegates to B
	intraBlockState.CreateAccount(addrA, true)
	intraBlockState.SetCode(addrA, types.AddressToDelegation(addrB))

	// A's GetCode should return delegation code pointing to B
	codeA := intraBlockState.GetCode(addrA)
	assert.True(t, types.IsDelegation(codeA))
	parsedA, _ := types.ParseDelegation(codeA)
	assert.Equal(t, addrB, parsedA)

	// A's ResolveCode should return B's code (which is also delegation)
	// Note: ResolveCode only resolves ONE level of delegation
	resolvedA := intraBlockState.ResolveCode(addrA)
	assert.True(t, types.IsDelegation(resolvedA), "ResolveCode only resolves one level")

	// B's ResolveCode should return C's code
	resolvedB := intraBlockState.ResolveCode(addrB)
	assert.Equal(t, targetCode, resolvedB)
}

// TestIntraBlockStateEIP7702DelegationToEmptyAccount tests delegation to non-existent account
func TestIntraBlockStateEIP7702DelegationToEmptyAccount(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	delegator := libcommon.HexToAddress("0x8888888888888888888888888888888888888888")
	emptyTarget := libcommon.HexToAddress("0x9999999999999999999999999999999999999999")

	r := NewPlainStateReader(tx)
	intraBlockState := New(r)

	// Create delegator with delegation to empty account
	intraBlockState.CreateAccount(delegator, true)
	delegationCode := types.AddressToDelegation(emptyTarget)
	intraBlockState.SetCode(delegator, delegationCode)

	// Verify delegation exists
	dd, ok := intraBlockState.GetDelegatedDesignation(delegator)
	assert.True(t, ok)
	assert.Equal(t, emptyTarget, dd)

	// ResolveCode should return empty code (target has no code)
	resolvedCode := intraBlockState.ResolveCode(delegator)
	assert.Empty(t, resolvedCode, "ResolveCode to empty account should return empty")

	// ResolveCodeHash for non-existent account returns zero hash (not emptyCodeHash)
	// This is the current behavior - getStateObject returns nil for non-existent accounts
	resolvedCodeHash := intraBlockState.ResolveCodeHash(delegator)
	// The target doesn't exist, so GetCodeHash returns the zero hash
	assert.Equal(t, libcommon.Hash{}, resolvedCodeHash, "non-existent account returns zero hash")
}

// TestIntraBlockStateEIP7702RemoveDelegation tests removing delegation by setting empty code
func TestIntraBlockStateEIP7702RemoveDelegation(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	delegator := libcommon.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	delegateTarget := libcommon.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	r := NewPlainStateReader(tx)
	intraBlockState := New(r)

	// Create and set delegation
	intraBlockState.CreateAccount(delegator, true)
	intraBlockState.SetCode(delegator, types.AddressToDelegation(delegateTarget))

	// Verify delegation
	_, ok := intraBlockState.GetDelegatedDesignation(delegator)
	assert.True(t, ok, "should have delegation")

	// Remove delegation by setting empty code
	intraBlockState.SetCode(delegator, nil)

	// Verify delegation is removed
	_, ok = intraBlockState.GetDelegatedDesignation(delegator)
	assert.False(t, ok, "should not have delegation after removal")

	assert.Empty(t, intraBlockState.GetCode(delegator))
}

// TestIntraBlockStateEIP7702SnapshotRevert tests snapshot/revert with delegation
func TestIntraBlockStateEIP7702SnapshotRevert(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	delegator := libcommon.HexToAddress("0xcccccccccccccccccccccccccccccccccccccccc")
	targetA := libcommon.HexToAddress("0xdddddddddddddddddddddddddddddddddddddddd")
	targetB := libcommon.HexToAddress("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")

	r := NewPlainStateReader(tx)
	intraBlockState := New(r)

	// Create account with delegation to A
	intraBlockState.CreateAccount(delegator, true)
	intraBlockState.SetCode(delegator, types.AddressToDelegation(targetA))
	intraBlockState.AddBalance(delegator, uint256.NewInt(1000))

	// Take snapshot
	snapshot := intraBlockState.Snapshot()

	// Verify current state
	dd, _ := intraBlockState.GetDelegatedDesignation(delegator)
	assert.Equal(t, targetA, dd)

	// Change delegation to B
	intraBlockState.SetCode(delegator, types.AddressToDelegation(targetB))
	intraBlockState.AddBalance(delegator, uint256.NewInt(500))

	// Verify new state
	dd, _ = intraBlockState.GetDelegatedDesignation(delegator)
	assert.Equal(t, targetB, dd)
	assert.Equal(t, uint64(1500), intraBlockState.GetBalance(delegator).Uint64())

	// Revert to snapshot
	intraBlockState.RevertToSnapshot(snapshot)

	// Verify state is reverted
	dd, _ = intraBlockState.GetDelegatedDesignation(delegator)
	assert.Equal(t, targetA, dd, "delegation should be reverted to A")
	assert.Equal(t, uint64(1000), intraBlockState.GetBalance(delegator).Uint64())
}

// TestIntraBlockStateEIP7702DelegationCodeSize tests GetCodeSize with delegation
func TestIntraBlockStateEIP7702DelegationCodeSize(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	delegator := libcommon.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff")
	delegateTarget := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")
	targetCode := make([]byte, 100) // 100 bytes of code

	r := NewPlainStateReader(tx)
	intraBlockState := New(r)

	// Create target with 100 bytes of code
	intraBlockState.CreateAccount(delegateTarget, true)
	intraBlockState.SetCode(delegateTarget, targetCode)

	// Create delegator
	intraBlockState.CreateAccount(delegator, true)
	delegationCode := types.AddressToDelegation(delegateTarget)
	intraBlockState.SetCode(delegator, delegationCode)

	// GetCodeSize should return 23 (delegation code size)
	codeSize := intraBlockState.GetCodeSize(delegator)
	assert.Equal(t, 23, codeSize, "delegation code is exactly 23 bytes")

	// Verify target's code size
	assert.Equal(t, 100, intraBlockState.GetCodeSize(delegateTarget))
}

// TestIntraBlockStateEIP7702DelegationWithBalance tests delegation account with balance
func TestIntraBlockStateEIP7702DelegationWithBalance(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	delegator := libcommon.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")
	delegateTarget := libcommon.HexToAddress("0x1234123412341234123412341234123412341234")

	r := NewPlainStateReader(tx)
	intraBlockState := New(r)

	// Create delegator with balance
	intraBlockState.CreateAccount(delegator, true)
	intraBlockState.AddBalance(delegator, uint256.NewInt(5000000))
	intraBlockState.SetNonce(delegator, 10)
	intraBlockState.SetCode(delegator, types.AddressToDelegation(delegateTarget))

	// Verify balance and nonce are independent of delegation
	assert.Equal(t, uint64(5000000), intraBlockState.GetBalance(delegator).Uint64())
	assert.Equal(t, uint64(10), intraBlockState.GetNonce(delegator))

	// Verify delegation
	dd, ok := intraBlockState.GetDelegatedDesignation(delegator)
	assert.True(t, ok)
	assert.Equal(t, delegateTarget, dd)

	// Add more balance
	intraBlockState.AddBalance(delegator, uint256.NewInt(1000000))
	assert.Equal(t, uint64(6000000), intraBlockState.GetBalance(delegator).Uint64())

	// Delegation should still be valid
	dd, ok = intraBlockState.GetDelegatedDesignation(delegator)
	assert.True(t, ok)
	assert.Equal(t, delegateTarget, dd)
}

// TestIntraBlockStateEIP7702SelfDestruct tests self-destruct with delegation
func TestIntraBlockStateEIP7702SelfDestruct(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	delegator := libcommon.HexToAddress("0x5678567856785678567856785678567856785678")
	delegateTarget := libcommon.HexToAddress("0x9abc9abc9abc9abc9abc9abc9abc9abc9abc9abc")

	r := NewPlainStateReader(tx)
	intraBlockState := New(r)

	// Create delegator with delegation and balance
	intraBlockState.CreateAccount(delegator, true)
	intraBlockState.AddBalance(delegator, uint256.NewInt(1000))
	intraBlockState.SetCode(delegator, types.AddressToDelegation(delegateTarget))

	// Self-destruct
	result := intraBlockState.Selfdestruct(delegator)
	assert.True(t, result)

	// Check state after self-destruct
	assert.True(t, intraBlockState.HasSelfdestructed(delegator))
	assert.Equal(t, uint64(0), intraBlockState.GetBalance(delegator).Uint64())
}

// TestIntraBlockStateEIP7702InvalidDelegationCode tests that invalid delegation code is not recognized
func TestIntraBlockStateEIP7702InvalidDelegationCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		code []byte
	}{
		{"Wrong prefix", append([]byte{0xef, 0x02, 0x00}, make([]byte, 20)...)},
		{"Too short", []byte{0xef, 0x01, 0x00}},
		{"Regular code starting with ef01", []byte{0xef, 0x01, 0x00, 0x60}},
		{"Empty", nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, tx := memdb.NewTestTx(t)

			addr := libcommon.HexToAddress("0x1111111111111111111111111111111111111111")

			r := NewPlainStateReader(tx)
			intraBlockState := New(r)

			intraBlockState.CreateAccount(addr, true)
			intraBlockState.SetCode(addr, tc.code)

			// Should not be recognized as delegation
			_, ok := intraBlockState.GetDelegatedDesignation(addr)
			assert.False(t, ok, "should not be delegation: %s", tc.name)
		})
	}
}

// TestIntraBlockStateEIP7702DelegatedDesignationPrefix verifies the prefix is correct
func TestIntraBlockStateEIP7702DelegatedDesignationPrefix(t *testing.T) {
	t.Parallel()

	// Verify the prefix is exactly 0xef0100
	assert.Equal(t, []byte{0xef, 0x01, 0x00}, params.DelegatedDesignationPrefix)
	assert.Len(t, params.DelegatedDesignationPrefix, 3)
}

// TestIntraBlockStateEIP7702MultipleAccounts tests multiple delegation accounts in same block
func TestIntraBlockStateEIP7702MultipleAccounts(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	intraBlockState := New(r)

	// Create multiple delegators all pointing to same target
	target := libcommon.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	targetCode := []byte{0x60, 0x42, 0x60, 0x00, 0x52} // PUSH1 0x42 PUSH1 0 MSTORE

	intraBlockState.CreateAccount(target, true)
	intraBlockState.SetCode(target, targetCode)

	delegators := make([]libcommon.Address, 5)
	for i := 0; i < 5; i++ {
		addrBytes := make([]byte, 20)
		addrBytes[0] = byte(i + 1)
		addrBytes[19] = byte(i + 1)
		delegators[i] = libcommon.BytesToAddress(addrBytes)

		intraBlockState.CreateAccount(delegators[i], true)
		intraBlockState.SetCode(delegators[i], types.AddressToDelegation(target))
		intraBlockState.AddBalance(delegators[i], uint256.NewInt(uint64(1000*(i+1))))
	}

	// Verify all delegators
	for i, delegator := range delegators {
		dd, ok := intraBlockState.GetDelegatedDesignation(delegator)
		assert.True(t, ok, "delegator %d should have delegation", i)
		assert.Equal(t, target, dd)

		// ResolveCode should return target's code for all
		resolved := intraBlockState.ResolveCode(delegator)
		assert.Equal(t, targetCode, resolved, "delegator %d should resolve to target code", i)

		// Balance should be individual
		assert.Equal(t, uint64(1000*(i+1)), intraBlockState.GetBalance(delegator).Uint64())
	}
}

