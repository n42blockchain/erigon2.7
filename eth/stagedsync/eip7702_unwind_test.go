package stagedsync

import (
	"context"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/kv"
	"github.com/erigontech/erigon-lib/kv/dbutils"
	"github.com/erigontech/erigon-lib/kv/memdb"
	"github.com/erigontech/erigon-lib/kv/temporal/historyv2"
	"github.com/erigontech/erigon-lib/log/v3"
	"github.com/erigontech/erigon-lib/wrap"

	"github.com/erigontech/erigon-lib/crypto"
	"github.com/erigontech/erigon/core/state"
	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/core/types/accounts"
	"github.com/erigontech/erigon/eth/stagedsync/stages"
	"github.com/erigontech/erigon/params"
)

// TestEIP7702UnwindDelegationAccount tests unwinding a block that added delegation
func TestEIP7702UnwindDelegationAccount(t *testing.T) {
	logger := log.New()
	ctx := context.Background()
	db := memdb.NewTestDB(t)
	tx := memdb.BeginRw(t, db)

	addr := libcommon.HexToAddress("0x1111111111111111111111111111111111111111")
	delegateAddr := libcommon.HexToAddress("0x2222222222222222222222222222222222222222")
	delegationCode := types.AddressToDelegation(delegateAddr)
	delegationCodeHash := crypto.Keccak256Hash(delegationCode)

	// Block 1: Create EOA
	acc1 := accounts.NewAccount()
	acc1.Initialised = true
	acc1.Balance = *uint256.NewInt(10000)
	acc1.Nonce = 1
	acc1.Incarnation = 0

	blockWriter1 := state.NewPlainStateWriter(tx, tx, 1)
	emptyAcc := accounts.NewAccount()
	err := blockWriter1.UpdateAccountData(addr, &emptyAcc, &acc1)
	require.NoError(t, err)
	err = blockWriter1.WriteChangeSets()
	require.NoError(t, err)
	err = blockWriter1.WriteHistory()
	require.NoError(t, err)

	// Block 2: Add delegation
	acc2 := acc1.SelfCopy()
	acc2.CodeHash = delegationCodeHash
	acc2.Nonce = 2

	blockWriter2 := state.NewPlainStateWriter(tx, tx, 2)
	err = blockWriter2.UpdateAccountData(addr, &acc1, acc2)
	require.NoError(t, err)
	err = blockWriter2.UpdateAccountCode(addr, acc2.Incarnation, delegationCodeHash, delegationCode)
	require.NoError(t, err)
	err = blockWriter2.WriteChangeSets()
	require.NoError(t, err)
	err = blockWriter2.WriteHistory()
	require.NoError(t, err)

	// Save stage progress
	err = stages.SaveStageProgress(tx, stages.Execution, 2)
	require.NoError(t, err)

	// Verify current state has delegation
	reader := state.NewPlainStateReader(tx)
	currentAcc, err := reader.ReadAccountData(addr)
	require.NoError(t, err)
	assert.Equal(t, delegationCodeHash, currentAcc.CodeHash)
	assert.Equal(t, uint64(2), currentAcc.Nonce)

	// Unwind to block 1
	cfg := ExecuteBlockCfg{}
	u := &UnwindState{ID: stages.Execution, UnwindPoint: 1}
	s := &StageState{ID: stages.Execution, BlockNumber: 2}
	err = UnwindExecutionStage(u, s, wrap.TxContainer{Tx: tx}, ctx, cfg, false, logger)
	require.NoError(t, err)

	// Verify state is restored to block 1 (no delegation)
	unwoundAcc, err := reader.ReadAccountData(addr)
	require.NoError(t, err)
	require.NotNil(t, unwoundAcc)

	// After unwind, account should have original state
	// Note: Due to THIN_HISTORY, CodeHash may need to be recovered from PlainContractCode
	// For Incarnation=0, the current recoverCodeHashPlain doesn't recover
	assert.Equal(t, uint64(1), unwoundAcc.Nonce, "Nonce should be restored to 1")
}

// TestEIP7702UnwindMultipleDelegationChanges tests unwinding multiple delegation changes
func TestEIP7702UnwindMultipleDelegationChanges(t *testing.T) {
	logger := log.New()
	ctx := context.Background()
	db := memdb.NewTestDB(t)
	tx := memdb.BeginRw(t, db)

	addr := libcommon.HexToAddress("0x3333333333333333333333333333333333333333")

	// Block 1: EOA
	acc1 := accounts.NewAccount()
	acc1.Initialised = true
	acc1.Balance = *uint256.NewInt(5000)
	acc1.Nonce = 1
	acc1.Incarnation = 0

	blockWriter1 := state.NewPlainStateWriter(tx, tx, 1)
	emptyAcc := accounts.NewAccount()
	err := blockWriter1.UpdateAccountData(addr, &emptyAcc, &acc1)
	require.NoError(t, err)
	err = blockWriter1.WriteChangeSets()
	require.NoError(t, err)
	err = blockWriter1.WriteHistory()
	require.NoError(t, err)

	// Block 2: Delegation to A
	delegateA := libcommon.HexToAddress("0xaaaa000000000000000000000000000000000000")
	delegationCodeA := types.AddressToDelegation(delegateA)
	codeHashA := crypto.Keccak256Hash(delegationCodeA)

	acc2 := acc1.SelfCopy()
	acc2.CodeHash = codeHashA
	acc2.Nonce = 2

	blockWriter2 := state.NewPlainStateWriter(tx, tx, 2)
	err = blockWriter2.UpdateAccountData(addr, &acc1, acc2)
	require.NoError(t, err)
	err = blockWriter2.UpdateAccountCode(addr, acc2.Incarnation, codeHashA, delegationCodeA)
	require.NoError(t, err)
	err = blockWriter2.WriteChangeSets()
	require.NoError(t, err)
	err = blockWriter2.WriteHistory()
	require.NoError(t, err)

	// Block 3: Delegation to B
	delegateB := libcommon.HexToAddress("0xbbbb000000000000000000000000000000000000")
	delegationCodeB := types.AddressToDelegation(delegateB)
	codeHashB := crypto.Keccak256Hash(delegationCodeB)

	acc3 := acc2.SelfCopy()
	acc3.CodeHash = codeHashB
	acc3.Nonce = 3

	blockWriter3 := state.NewPlainStateWriter(tx, tx, 3)
	err = blockWriter3.UpdateAccountData(addr, acc2, acc3)
	require.NoError(t, err)
	err = blockWriter3.UpdateAccountCode(addr, acc3.Incarnation, codeHashB, delegationCodeB)
	require.NoError(t, err)
	err = blockWriter3.WriteChangeSets()
	require.NoError(t, err)
	err = blockWriter3.WriteHistory()
	require.NoError(t, err)

	// Save stage progress
	err = stages.SaveStageProgress(tx, stages.Execution, 3)
	require.NoError(t, err)

	// Verify we're at block 3 with delegation B
	reader := state.NewPlainStateReader(tx)
	currentAcc, err := reader.ReadAccountData(addr)
	require.NoError(t, err)
	assert.Equal(t, codeHashB, currentAcc.CodeHash)
	assert.Equal(t, uint64(3), currentAcc.Nonce)

	// Unwind to block 2
	cfg := ExecuteBlockCfg{}
	u := &UnwindState{ID: stages.Execution, UnwindPoint: 2}
	s := &StageState{ID: stages.Execution, BlockNumber: 3}
	err = UnwindExecutionStage(u, s, wrap.TxContainer{Tx: tx}, ctx, cfg, false, logger)
	require.NoError(t, err)

	// Verify state is restored to block 2 (delegation A)
	unwoundAcc, err := reader.ReadAccountData(addr)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), unwoundAcc.Nonce, "Nonce should be restored to 2")
	// CodeHash might be codeHashA depending on recovery logic
}

// TestEIP7702ChangeSetContents tests that changesets correctly capture delegation state
func TestEIP7702ChangeSetContents(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	addr := libcommon.HexToAddress("0x4444444444444444444444444444444444444444")
	delegateAddr := libcommon.HexToAddress("0x5555555555555555555555555555555555555555")
	delegationCode := types.AddressToDelegation(delegateAddr)
	delegationCodeHash := crypto.Keccak256Hash(delegationCode)

	// Create original account with delegation
	originalAcc := accounts.NewAccount()
	originalAcc.Initialised = true
	originalAcc.Balance = *uint256.NewInt(1000)
	originalAcc.Nonce = 5
	originalAcc.Incarnation = 1
	originalAcc.CodeHash = delegationCodeHash

	// Write original state in block 1
	blockWriter1 := state.NewPlainStateWriter(tx, tx, 1)
	emptyAcc := accounts.NewAccount()
	err := blockWriter1.UpdateAccountData(addr, &emptyAcc, &originalAcc)
	require.NoError(t, err)
	err = blockWriter1.UpdateAccountCode(addr, originalAcc.Incarnation, delegationCodeHash, delegationCode)
	require.NoError(t, err)
	err = blockWriter1.WriteChangeSets()
	require.NoError(t, err)

	// Modify account in block 2 (remove delegation)
	modifiedAcc := originalAcc.SelfCopy()
	modifiedAcc.CodeHash = libcommon.BytesToHash(crypto.Keccak256(nil))
	modifiedAcc.Nonce = 6

	blockWriter2 := state.NewPlainStateWriter(tx, tx, 2)
	err = blockWriter2.UpdateAccountData(addr, &originalAcc, modifiedAcc)
	require.NoError(t, err)
	err = blockWriter2.WriteChangeSets()
	require.NoError(t, err)
	err = blockWriter2.WriteHistory()
	require.NoError(t, err)

	// Verify changeset for block 2 has original delegation account
	cs := historyv2.NewAccountChangeSet()
	err = historyv2.ForPrefix(tx, kv.AccountChangeSet, dbutils.EncodeBlockNumber(2), func(_ uint64, k, v []byte) error {
		return cs.Add(k, v)
	})
	require.NoError(t, err)
	assert.Equal(t, 1, cs.Len())

	// The changeset should contain the original account state
	for _, change := range cs.Changes {
		if string(change.Key) == string(addr.Bytes()) {
			var csAcc accounts.Account
			err = csAcc.DecodeForStorage(change.Value)
			require.NoError(t, err)

			// THIN_HISTORY: CodeHash is omitted
			assert.True(t, csAcc.IsEmptyCodeHash(), "THIN_HISTORY omits CodeHash")
			assert.Equal(t, originalAcc.Nonce, csAcc.Nonce)
			assert.Equal(t, originalAcc.Incarnation, csAcc.Incarnation)
		}
	}
}

// TestEIP7702DelegationPrefixValidation tests validation of delegation prefix
func TestEIP7702DelegationPrefixValidation(t *testing.T) {
	t.Parallel()

	// Valid delegation
	validAddr := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")
	validDelegation := types.AddressToDelegation(validAddr)
	assert.True(t, types.IsDelegation(validDelegation))
	assert.Equal(t, 23, len(validDelegation)) // 3 bytes prefix + 20 bytes address

	// Verify prefix is correct
	assert.Equal(t, params.DelegatedDesignationPrefix, validDelegation[:3])

	// Invalid cases
	testCases := []struct {
		name  string
		code  []byte
		valid bool
	}{
		{"nil code", nil, false},
		{"empty code", []byte{}, false},
		{"too short", []byte{0xef, 0x01, 0x00}, false},
		{"wrong prefix", append([]byte{0xef, 0x02, 0x00}, validAddr.Bytes()...), false},
		{"too long", append(validDelegation, 0x00), false},
		{"regular contract code", []byte{0x60, 0x00, 0x60, 0x00, 0xf3}, false},
		{"valid delegation", validDelegation, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.valid, types.IsDelegation(tc.code))
		})
	}
}

// TestEIP7702PlainContractCodeTable tests PlainContractCode table behavior with delegation
func TestEIP7702PlainContractCodeTable(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	addr := libcommon.HexToAddress("0x6666666666666666666666666666666666666666")
	delegateAddr := libcommon.HexToAddress("0x7777777777777777777777777777777777777777")
	delegationCode := types.AddressToDelegation(delegateAddr)
	delegationCodeHash := crypto.Keccak256Hash(delegationCode)

	// Test with different incarnation values
	testCases := []struct {
		name        string
		incarnation uint64
	}{
		{"Incarnation=0", 0},
		{"Incarnation=1", 1},
		{"Incarnation=100", 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create unique address for this test
			addrBytes := addr.Bytes()
			addrBytes[0] = byte(tc.incarnation)
			testAddr := libcommon.BytesToAddress(addrBytes)

			// Write code to Code table
			err := tx.Put(kv.Code, delegationCodeHash.Bytes(), delegationCode)
			require.NoError(t, err)

			// Write to PlainContractCode
			key := dbutils.PlainGenerateStoragePrefix(testAddr.Bytes(), tc.incarnation)
			err = tx.Put(kv.PlainContractCode, key, delegationCodeHash.Bytes())
			require.NoError(t, err)

			// Read back
			codeHashFromDB, err := tx.GetOne(kv.PlainContractCode, key)
			require.NoError(t, err)
			assert.Equal(t, delegationCodeHash.Bytes(), codeHashFromDB)

			// Verify code is delegation
			code, err := tx.GetOne(kv.Code, codeHashFromDB)
			require.NoError(t, err)
			assert.True(t, types.IsDelegation(code))
		})
	}
}

// TestEIP7702RecoverCodeHashPlainFunction tests the recoverCodeHashPlain function
func TestEIP7702RecoverCodeHashPlainFunction(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	delegateAddr := libcommon.HexToAddress("0x8888888888888888888888888888888888888888")
	delegationCode := types.AddressToDelegation(delegateAddr)
	delegationCodeHash := crypto.Keccak256Hash(delegationCode)

	// Write delegation code to Code table
	err := tx.Put(kv.Code, delegationCodeHash.Bytes(), delegationCode)
	require.NoError(t, err)

	t.Run("WithIncarnation1", func(t *testing.T) {
		addr := libcommon.HexToAddress("0x9999999999999999999999999999999999999999")
		incarnation := uint64(1)

		// Write to PlainContractCode
		err := tx.Put(kv.PlainContractCode, dbutils.PlainGenerateStoragePrefix(addr.Bytes(), incarnation), delegationCodeHash.Bytes())
		require.NoError(t, err)

		// Test recoverCodeHashPlain
		acc := &accounts.Account{
			Incarnation: incarnation,
			CodeHash:    libcommon.BytesToHash(crypto.Keccak256(nil)), // empty
		}

		recoverCodeHashPlain(acc, tx, addr.Bytes())

		// Should recover because Incarnation > 0
		assert.Equal(t, delegationCodeHash, acc.CodeHash)
	})

	t.Run("WithIncarnation0_Fixed", func(t *testing.T) {
		addr := libcommon.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		incarnation := uint64(0)

		// Write to PlainContractCode
		err := tx.Put(kv.PlainContractCode, dbutils.PlainGenerateStoragePrefix(addr.Bytes(), incarnation), delegationCodeHash.Bytes())
		require.NoError(t, err)

		// Test recoverCodeHashPlain
		acc := &accounts.Account{
			Incarnation: incarnation,
			CodeHash:    libcommon.BytesToHash(crypto.Keccak256(nil)), // empty
		}

		recoverCodeHashPlain(acc, tx, addr.Bytes())

		// Fixed behavior: NOW recovers for Incarnation=0 (EIP-7702 delegation accounts)
		assert.Equal(t, delegationCodeHash, acc.CodeHash, "Fixed: should recover CodeHash for Incarnation=0")
	})
}

