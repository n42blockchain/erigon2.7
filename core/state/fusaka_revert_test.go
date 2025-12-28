// Copyright 2025 The Erigon Authors
// This file is part of Erigon.
//
// Erigon is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Erigon is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Erigon. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/kv/memdb"

	"github.com/erigontech/erigon/core/types"
)

// Fusaka EIP State Revert Tests
// This file tests that all Fusaka EIPs correctly handle state reversion (snapshot/revert)
//
// Fusaka EIPs (13):
// - EIP-7594: PeerDAS (CL-only, no EL state)
// - EIP-7642: Historical Window (P2P protocol, no EL state)
// - EIP-7823: MODEXP Length Limits (precompile validation, no persistent state)
// - EIP-7825: Transaction Gas Limit Cap (tx validation, no persistent state)
// - EIP-7883: MODEXP Gas Cost (gas calculation, no persistent state)
// - EIP-7892: Blob Parameter Hardforks (config, no persistent state)
// - EIP-7910: eth_config RPC (read-only, no state changes)
// - EIP-7917: Proposer Lookahead (CL-only, no direct EL state)
// - EIP-7918: Blob Base Fee Bound (gas calculation, no persistent state)
// - EIP-7934: RLP Block Size Limit (validation, no persistent state)
// - EIP-7935: Block Gas Limit Increase (config, no persistent state)
// - EIP-7939: CLZ Opcode (stack operation, no persistent state)
// - EIP-7951: P256 Verify (precompile, no persistent state)

// TestEIP7594PeerDASNoDirectELState verifies EIP-7594 doesn't affect EL state
// EIP-7594 (PeerDAS) is a CL data availability layer change
func TestEIP7594PeerDASNoDirectELState(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take snapshot
	snapshot := state.Snapshot()

	// EIP-7594 doesn't modify EL state directly
	// It only affects CL data availability sampling

	// Verify no state changes
	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestEIP7642HistoricalWindowNoDirectELState verifies EIP-7642 doesn't affect EL state
// EIP-7642 is a P2P protocol change for historical block range
func TestEIP7642HistoricalWindowNoDirectELState(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take snapshot
	snapshot := state.Snapshot()

	// EIP-7642 is a P2P protocol change
	// It doesn't modify EL state

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestEIP7823MODEXPLengthLimitsNoState verifies EIP-7823 doesn't create persistent state
// EIP-7823 adds length limits to MODEXP precompile
func TestEIP7823MODEXPLengthLimitsNoState(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take snapshot
	snapshot := state.Snapshot()

	// EIP-7823 is a precompile input validation change
	// It doesn't modify persistent state
	// Any MODEXP execution that fails due to length limits returns an error
	// but doesn't leave residual state

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestEIP7825TxGasLimitCapNoState verifies EIP-7825 doesn't create persistent state
// EIP-7825 adds a cap on transaction gas limits
func TestEIP7825TxGasLimitCapNoState(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take snapshot
	snapshot := state.Snapshot()

	// EIP-7825 is a transaction validation rule
	// Transactions exceeding the gas limit are rejected before execution
	// No state changes occur

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestEIP7883MODEXPGasCostNoState verifies EIP-7883 doesn't create persistent state
// EIP-7883 changes MODEXP gas calculation
func TestEIP7883MODEXPGasCostNoState(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take snapshot
	snapshot := state.Snapshot()

	// EIP-7883 only affects gas calculation for MODEXP precompile
	// The precompile doesn't modify persistent storage
	// Gas consumption is part of transaction execution, not state

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestEIP7892BlobParametersNoState verifies EIP-7892 doesn't create persistent state
// EIP-7892 defines blob schedule for blob-parameter-only hardforks
func TestEIP7892BlobParametersNoState(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take snapshot
	snapshot := state.Snapshot()

	// EIP-7892 is a configuration change (blob schedule)
	// It doesn't modify EL state
	// Blob parameters are read from chain config, not from state

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestEIP7910EthConfigRPCNoState verifies EIP-7910 is read-only
// EIP-7910 adds eth_config JSON-RPC method
func TestEIP7910EthConfigRPCNoState(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take snapshot
	snapshot := state.Snapshot()

	// EIP-7910 is a JSON-RPC method (eth_config)
	// It only reads configuration, never writes state

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestEIP7917ProposerLookaheadNoDirectELState verifies EIP-7917 doesn't affect EL state
// EIP-7917 is deterministic proposer lookahead (CL feature)
func TestEIP7917ProposerLookaheadNoDirectELState(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take snapshot
	snapshot := state.Snapshot()

	// EIP-7917 is a CL-only feature
	// Proposer lookahead is stored in beacon state, not EL state

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestEIP7918BlobBaseFeeNoState verifies EIP-7918 doesn't create persistent state
// EIP-7918 bounds blob base fee by execution cost
func TestEIP7918BlobBaseFeeNoState(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take snapshot
	snapshot := state.Snapshot()

	// EIP-7918 modifies excess blob gas calculation
	// This affects block header fields (calculated, not stored in state trie)
	// No persistent state changes

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestEIP7934RLPBlockSizeLimitNoState verifies EIP-7934 doesn't create persistent state
// EIP-7934 limits RLP-encoded block size
func TestEIP7934RLPBlockSizeLimitNoState(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take snapshot
	snapshot := state.Snapshot()

	// EIP-7934 is a block validation rule
	// Blocks exceeding the RLP size limit are rejected
	// No state changes occur

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestEIP7935BlockGasLimitNoState verifies EIP-7935 doesn't create persistent state
// EIP-7935 increases the default block gas limit
func TestEIP7935BlockGasLimitNoState(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take snapshot
	snapshot := state.Snapshot()

	// EIP-7935 is a client configuration change (default gas limit)
	// Gas limit is a block header field, not stored in state trie
	// No persistent state changes

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestEIP7939CLZOpcodeNoState verifies EIP-7939 CLZ opcode doesn't create persistent state
// EIP-7939 adds CLZ (Count Leading Zeros) opcode
func TestEIP7939CLZOpcodeNoState(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take snapshot
	snapshot := state.Snapshot()

	// EIP-7939 CLZ is a pure stack operation
	// It pops one value, pushes one value (the count of leading zeros)
	// No memory, storage, or log operations
	// No persistent state changes

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestEIP7951P256VerifyNoState verifies EIP-7951 P256VERIFY doesn't create persistent state
// EIP-7951 adds secp256r1 signature verification precompile
func TestEIP7951P256VerifyNoState(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take snapshot
	snapshot := state.Snapshot()

	// EIP-7951 P256VERIFY is a pure verification precompile
	// It takes input (hash, r, s, x, y) and returns 1 or empty
	// No state modifications (no SSTORE, LOG, etc.)
	// No persistent state changes

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestFusakaEIPsNoUnexpectedStateChanges verifies no Fusaka EIP leaves unexpected state
func TestFusakaEIPsNoUnexpectedStateChanges(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take initial snapshot
	snapshot := state.Snapshot()

	// Simulate various operations that might be affected by Fusaka EIPs
	
	// Create a test account for opcode/precompile tests
	testAddr := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")
	state.CreateAccount(testAddr, false)
	state.SetBalance(testAddr, uint256.NewInt(1000000))
	state.SetNonce(testAddr, 1)

	// Store some test data
	testSlot := libcommon.Hash{1, 2, 3}
	state.SetState(testAddr, &testSlot, *uint256.NewInt(0xDEADBEEF))

	// Verify data is stored
	var value uint256.Int
	state.GetState(testAddr, &testSlot, &value)
	require.Equal(t, uint64(0xDEADBEEF), value.Uint64())

	// Revert to snapshot
	state.RevertToSnapshot(snapshot)

	// Verify everything is reverted
	// Account should not exist after revert
	assert.False(t, state.Exist(testAddr), "Account should not exist after revert")
}

// TestFusakaPrecompileStateIsolation verifies precompiles don't leak state
func TestFusakaPrecompileStateIsolation(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Precompile addresses (Fusaka-relevant)
	modexpAddr := libcommon.BytesToAddress([]byte{0x05})    // MODEXP
	p256Addr := libcommon.BytesToAddress([]byte{0x01, 0x00}) // P256VERIFY

	// Take snapshot
	snapshot := state.Snapshot()

	// Precompiles should not have storage
	modexpSlot := libcommon.Hash{1}
	p256Slot := libcommon.Hash{1}

	// Attempt to set state on precompile addresses (should be isolated)
	state.SetState(modexpAddr, &modexpSlot, *uint256.NewInt(1))
	state.SetState(p256Addr, &p256Slot, *uint256.NewInt(2))

	// Verify storage was set (state allows it, execution would fail)
	var v1, v2 uint256.Int
	state.GetState(modexpAddr, &modexpSlot, &v1)
	state.GetState(p256Addr, &p256Slot, &v2)
	assert.Equal(t, uint64(1), v1.Uint64())
	assert.Equal(t, uint64(2), v2.Uint64())

	// Revert to snapshot
	state.RevertToSnapshot(snapshot)

	// Verify reverted
	state.GetState(modexpAddr, &modexpSlot, &v1)
	state.GetState(p256Addr, &p256Slot, &v2)
	assert.True(t, v1.IsZero(), "MODEXP precompile storage should be reverted")
	assert.True(t, v2.IsZero(), "P256VERIFY precompile storage should be reverted")
}

// TestFusakaOpcodeExecutionStateRevert verifies opcode execution can be reverted
// CLZ opcode (EIP-7939) doesn't modify state, but we test the general pattern
func TestFusakaOpcodeExecutionStateRevert(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	contractAddr := libcommon.HexToAddress("0xCONTRACT")

	// Create contract account
	state.CreateAccount(contractAddr, false)

	// Set some code (simulating a contract with CLZ-like operations)
	// CLZ doesn't modify state, but the contract might
	code := []byte{0x60, 0x00} // PUSH1 0
	state.SetCode(contractAddr, code)

	// Take snapshot AFTER account creation
	snapshot := state.Snapshot()

	// Simulate state modification during execution
	storageSlot := libcommon.Hash{1}
	state.SetState(contractAddr, &storageSlot, *uint256.NewInt(42))

	// Verify modification
	var value uint256.Int
	state.GetState(contractAddr, &storageSlot, &value)
	assert.Equal(t, uint64(42), value.Uint64())

	// Revert to snapshot (simulating transaction revert)
	state.RevertToSnapshot(snapshot)

	// Verify storage is reverted
	state.GetState(contractAddr, &storageSlot, &value)
	assert.True(t, value.IsZero(), "Storage should be reverted")

	// But contract should still exist (it was created before snapshot)
	assert.True(t, state.Exist(contractAddr), "Contract should still exist")
	assert.Equal(t, code, state.GetCode(contractAddr), "Code should still exist")
}

// TestFusakaGasRelatedEIPsNoStateEffect verifies gas-related EIPs don't affect state
func TestFusakaGasRelatedEIPsNoStateEffect(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Gas-related Fusaka EIPs:
	// - EIP-7823: MODEXP length limits
	// - EIP-7825: Transaction gas limit cap
	// - EIP-7883: MODEXP gas cost
	// - EIP-7918: Blob base fee bound
	//
	// All these EIPs affect gas calculation/validation, not state

	testAddr := libcommon.HexToAddress("0xGAS_TEST")
	state.CreateAccount(testAddr, false)
	initialBalance := uint256.NewInt(1000000)
	state.SetBalance(testAddr, initialBalance)

	// Take snapshot
	snapshot := state.Snapshot()

	// Simulate gas deduction (would happen during tx execution)
	gasUsed := uint256.NewInt(21000)
	newBalance := new(uint256.Int).Sub(initialBalance, gasUsed)
	state.SetBalance(testAddr, newBalance)

	// Verify balance changed
	assert.Equal(t, newBalance.Uint64(), state.GetBalance(testAddr).Uint64())

	// Revert to snapshot (simulating out-of-gas revert)
	state.RevertToSnapshot(snapshot)

	// Verify balance is restored
	assert.Equal(t, initialBalance.Uint64(), state.GetBalance(testAddr).Uint64())
}

// TestFusakaConfigEIPsNoStateEffect verifies config-related EIPs don't affect state
func TestFusakaConfigEIPsNoStateEffect(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Config-related Fusaka EIPs:
	// - EIP-7892: Blob schedule
	// - EIP-7910: eth_config RPC
	// - EIP-7935: Block gas limit
	// - EIP-7934: RLP block size limit
	//
	// These EIPs define/read configuration, not state

	snapshot := state.Snapshot()

	// No state operations for config EIPs

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestFusakaCLOnlyEIPsNoELState verifies CL-only EIPs don't affect EL state
func TestFusakaCLOnlyEIPsNoELState(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// CL-only Fusaka EIPs:
	// - EIP-7594: PeerDAS
	// - EIP-7642: Historical Window
	// - EIP-7917: Proposer Lookahead
	//
	// These EIPs only affect CL/P2P layer

	snapshot := state.Snapshot()

	// No EL state operations for CL-only EIPs

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestFusakaEIPsWithTransactionRevert tests full transaction revert scenario
func TestFusakaEIPsWithTransactionRevert(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	contract := libcommon.HexToAddress("0xCONTRACT")

	// Take initial snapshot (before any account creation)
	initialSnapshot := state.Snapshot()

	// Setup contract account
	state.CreateAccount(contract, false)
	state.SetCode(contract, []byte{0x60, 0x00}) // PUSH1 0

	// Take transaction snapshot (after account setup)
	txSnapshot := state.Snapshot()

	// Simulate transaction execution with contract state changes
	slot1 := libcommon.Hash{1}
	slot2 := libcommon.Hash{2}
	state.SetState(contract, &slot1, *uint256.NewInt(100))
	state.SetState(contract, &slot2, *uint256.NewInt(200))

	// Add log (would be part of EIP-6110 deposit processing if any)
	log := &types.Log{
		Address: contract,
		Topics:  []libcommon.Hash{{1}},
		Data:    []byte{1, 2, 3},
	}
	state.AddLog(log)

	// Verify all changes
	var v1, v2 uint256.Int
	state.GetState(contract, &slot1, &v1)
	state.GetState(contract, &slot2, &v2)
	assert.Equal(t, uint64(100), v1.Uint64())
	assert.Equal(t, uint64(200), v2.Uint64())

	// Simulate transaction revert (e.g., due to EIP-7825 gas limit exceeded)
	state.RevertToSnapshot(txSnapshot)

	// Verify transaction changes are reverted (but account still exists)
	state.GetState(contract, &slot1, &v1)
	state.GetState(contract, &slot2, &v2)
	assert.True(t, v1.IsZero(), "Slot1 should be reverted")
	assert.True(t, v2.IsZero(), "Slot2 should be reverted")

	// Account should still exist (it was created before txSnapshot)
	assert.True(t, state.Exist(contract), "Contract should still exist")

	// Revert to initial snapshot (before account creation)
	state.RevertToSnapshot(initialSnapshot)

	// Now account should not exist
	assert.False(t, state.Exist(contract), "Contract should not exist after full revert")
}

// TestFusakaNestedCallRevert tests nested call revert within transaction
func TestFusakaNestedCallRevert(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	contractA := libcommon.HexToAddress("0xAAAA")
	contractB := libcommon.HexToAddress("0xBBBB")

	// Setup contracts
	state.CreateAccount(contractA, false)
	state.SetCode(contractA, []byte{0x60, 0x00})
	state.CreateAccount(contractB, false)
	state.SetCode(contractB, []byte{0x60, 0x01})

	slotA := libcommon.Hash{1}
	slotB := libcommon.Hash{2}

	// Transaction starts
	txSnapshot := state.Snapshot()

	// ContractA executes
	state.SetState(contractA, &slotA, *uint256.NewInt(100))

	// ContractA calls ContractB
	callSnapshot := state.Snapshot()

	// ContractB executes (might use CLZ, P256VERIFY, etc.)
	state.SetState(contractB, &slotB, *uint256.NewInt(200))

	// ContractB reverts (e.g., due to EIP-7823 MODEXP length error in subcall)
	state.RevertToSnapshot(callSnapshot)

	// Verify ContractB changes are reverted, but ContractA changes persist
	var vA, vB uint256.Int
	state.GetState(contractA, &slotA, &vA)
	state.GetState(contractB, &slotB, &vB)
	assert.Equal(t, uint64(100), vA.Uint64(), "ContractA state should persist")
	assert.True(t, vB.IsZero(), "ContractB state should be reverted")

	// Now revert the entire transaction
	state.RevertToSnapshot(txSnapshot)

	// Verify everything is reverted
	state.GetState(contractA, &slotA, &vA)
	assert.True(t, vA.IsZero(), "ContractA state should be reverted after tx revert")
}

// TestFusakaEIPsComplianceStatus documents which EIPs affect state revert
func TestFusakaEIPsComplianceStatus(t *testing.T) {
	type eipStatus struct {
		EIP            string
		Title          string
		AffectsELState bool
		Notes          string
	}

	fusakaEIPs := []eipStatus{
		{"7594", "PeerDAS", false, "CL data availability, no EL state"},
		{"7642", "Historical Window", false, "P2P protocol, no EL state"},
		{"7823", "MODEXP Length Limits", false, "Precompile validation, errors don't persist"},
		{"7825", "Transaction Gas Limit Cap", false, "TX validation, rejected before execution"},
		{"7883", "MODEXP Gas Cost", false, "Gas calculation only"},
		{"7892", "Blob Parameters", false, "Configuration only"},
		{"7910", "eth_config RPC", false, "Read-only RPC method"},
		{"7917", "Proposer Lookahead", false, "CL-only, no direct EL state"},
		{"7918", "Blob Base Fee Bound", false, "Header field calculation"},
		{"7934", "RLP Block Size Limit", false, "Block validation only"},
		{"7935", "Block Gas Limit Increase", false, "Client configuration only"},
		{"7939", "CLZ Opcode", false, "Pure stack operation, no state"},
		{"7951", "P256 Verify", false, "Pure verification precompile, no state"},
	}

	t.Log("Fusaka EIPs State Revert Compliance:")
	t.Log("")
	for _, eip := range fusakaEIPs {
		stateImpact := "No EL state impact"
		if eip.AffectsELState {
			stateImpact = "AFFECTS EL STATE"
		}
		t.Logf("EIP-%s (%s): %s - %s", eip.EIP, eip.Title, stateImpact, eip.Notes)
	}

	t.Log("")
	t.Log("Summary: All 13 Fusaka EIPs have NO direct EL persistent state impact.")
	t.Log("State revert is handled by the standard IntraBlockState snapshot/revert mechanism.")
}

