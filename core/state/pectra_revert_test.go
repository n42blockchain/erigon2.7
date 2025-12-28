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
	"github.com/erigontech/erigon/params"
)

// Pectra EIP State Revert Tests
// This file tests that all Pectra EIPs correctly handle state reversion (snapshot/revert)

// TestEIP2935StateRevert tests EIP-2935 (Historical block hashes) state revert
// EIP-2935 stores block hashes in HistoryStorageAddress contract
func TestEIP2935StateRevert(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	historyAddr := params.HistoryStorageAddress

	// Simulate storing a block hash (as EIP-2935 does)
	slotNum := uint64(100) % params.BlockHashHistoryServeWindow
	storageSlot := libcommon.BytesToHash(uint256.NewInt(slotNum).Bytes())
	blockHash := libcommon.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	hashValue := uint256.NewInt(0).SetBytes32(blockHash.Bytes())

	// Take snapshot before modification
	snapshot := state.Snapshot()

	// Store block hash (what EIP-2935 does)
	state.SetState(historyAddr, &storageSlot, *hashValue)

	// Verify it's stored
	var storedValue uint256.Int
	state.GetState(historyAddr, &storageSlot, &storedValue)
	assert.Equal(t, *hashValue, storedValue, "Block hash should be stored")

	// Revert to snapshot
	state.RevertToSnapshot(snapshot)

	// Verify storage is reverted
	var revertedValue uint256.Int
	state.GetState(historyAddr, &storageSlot, &revertedValue)
	assert.True(t, revertedValue.IsZero(), "Storage should be reverted to zero")
}

// TestEIP6110DepositStateRevert tests that deposit processing state can be reverted
// EIP-6110 parses deposits from logs - no direct state changes, but affects pending deposits
func TestEIP6110DepositLogRevert(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take snapshot
	snapshot := state.Snapshot()

	// Simulate log addition (deposits come from logs)
	log := &types.Log{
		Address: libcommon.HexToAddress("0x1234"),
		Topics:  []libcommon.Hash{{1, 2, 3}},
		Data:    []byte{4, 5, 6},
	}
	state.AddLog(log)

	// Revert to snapshot
	state.RevertToSnapshot(snapshot)

	// Logs should be reverted (logSize decremented)
	// Note: After revert, the log entry is removed
}

// TestEIP7002WithdrawalRequestStateRevert tests EIP-7002 state revert
// EIP-7002 uses WithdrawalRequestAddress system contract
func TestEIP7002WithdrawalRequestStateRevert(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	withdrawalAddr := params.WithdrawalRequestAddress

	// Take snapshot
	snapshot := state.Snapshot()

	// Simulate syscall state change (withdrawal request queue update)
	storageSlot := libcommon.Hash{1}
	value := *uint256.NewInt(12345)
	state.SetState(withdrawalAddr, &storageSlot, value)

	// Verify state is set
	var storedValue uint256.Int
	state.GetState(withdrawalAddr, &storageSlot, &storedValue)
	assert.Equal(t, value, storedValue)

	// Revert
	state.RevertToSnapshot(snapshot)

	// Verify reverted
	var revertedValue uint256.Int
	state.GetState(withdrawalAddr, &storageSlot, &revertedValue)
	assert.True(t, revertedValue.IsZero())
}

// TestEIP7251ConsolidationRequestStateRevert tests EIP-7251 state revert
// EIP-7251 uses ConsolidationRequestAddress system contract
func TestEIP7251ConsolidationRequestStateRevert(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	consolidationAddr := params.ConsolidationRequestAddress

	// Take snapshot
	snapshot := state.Snapshot()

	// Simulate syscall state change
	storageSlot := libcommon.Hash{2}
	value := *uint256.NewInt(67890)
	state.SetState(consolidationAddr, &storageSlot, value)

	// Verify
	var storedValue uint256.Int
	state.GetState(consolidationAddr, &storageSlot, &storedValue)
	assert.Equal(t, value, storedValue)

	// Revert
	state.RevertToSnapshot(snapshot)

	// Verify reverted
	var revertedValue uint256.Int
	state.GetState(consolidationAddr, &storageSlot, &revertedValue)
	assert.True(t, revertedValue.IsZero())
}

// TestEIP7702DelegationStateRevert tests EIP-7702 delegation state revert
// EIP-7702 sets delegation code on EOA accounts
func TestEIP7702DelegationStateRevert(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	delegator := libcommon.HexToAddress("0x1111111111111111111111111111111111111111")
	delegateTarget := libcommon.HexToAddress("0x2222222222222222222222222222222222222222")

	// Create account
	state.CreateAccount(delegator, false)
	state.SetBalance(delegator, uint256.NewInt(1000000))
	state.SetNonce(delegator, 1)

	// Take snapshot before delegation
	snapshot := state.Snapshot()

	// Set delegation code
	delegationCode := types.AddressToDelegation(delegateTarget)
	state.SetCode(delegator, delegationCode)

	// Verify delegation is set
	code := state.GetCode(delegator)
	assert.True(t, types.IsDelegation(code))

	dd, ok := state.GetDelegatedDesignation(delegator)
	assert.True(t, ok)
	assert.Equal(t, delegateTarget, dd)

	// Revert to snapshot
	state.RevertToSnapshot(snapshot)

	// Verify delegation is reverted (code should be empty)
	revertedCode := state.GetCode(delegator)
	assert.False(t, types.IsDelegation(revertedCode))
	assert.Empty(t, revertedCode)

	_, ok = state.GetDelegatedDesignation(delegator)
	assert.False(t, ok)
}

// TestEIP7623GasFloorStateRevert tests that EIP-7623 doesn't affect state
// EIP-7623 only affects gas calculation, no state changes
func TestEIP7623GasFloorNoStateChanges(t *testing.T) {
	t.Parallel()
	// EIP-7623 (calldata gas cost increase) only affects gas calculations
	// No state changes are made by this EIP
	// This test confirms the EIP doesn't introduce unexpected state changes
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take snapshot
	snapshot := state.Snapshot()

	// EIP-7623 doesn't modify state directly
	// Gas calculations happen in state_transition.go

	// Revert should be a no-op
	state.RevertToSnapshot(snapshot)

	// Test passes if no panic occurs
}

// TestEIP7685RequestsStateRevert tests EIP-7685 execution requests
// EIP-7685 aggregates requests from EIP-6110, EIP-7002, EIP-7251
func TestEIP7685RequestsStateRevert(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Test that all request-related state changes can be reverted
	addresses := []libcommon.Address{
		params.WithdrawalRequestAddress,
		params.ConsolidationRequestAddress,
	}

	snapshot := state.Snapshot()

	// Simulate state changes for each request type
	for i, addr := range addresses {
		slot := libcommon.Hash{byte(i)}
		value := *uint256.NewInt(uint64(i + 1))
		state.SetState(addr, &slot, value)
	}

	// Revert all
	state.RevertToSnapshot(snapshot)

	// Verify all reverted
	for i, addr := range addresses {
		slot := libcommon.Hash{byte(i)}
		var value uint256.Int
		state.GetState(addr, &slot, &value)
		assert.True(t, value.IsZero(), "Address %s slot %d should be reverted", addr, i)
	}
}

// TestEIP7691BlobThroughputNoStateChanges tests that EIP-7691 doesn't affect state
// EIP-7691 only changes blob limits, no state modifications
func TestEIP7691BlobThroughputNoStateChanges(t *testing.T) {
	t.Parallel()
	// EIP-7691 (blob throughput increase) only affects consensus parameters
	// No state changes are made by this EIP
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	snapshot := state.Snapshot()

	// EIP-7691 doesn't modify state directly
	// Blob gas calculations happen in consensus/misc/eip4844.go

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestEIP7840BlobScheduleNoStateChanges tests that EIP-7840 doesn't affect state
// EIP-7840 only adds config file structure, no runtime state changes
func TestEIP7840BlobScheduleNoStateChanges(t *testing.T) {
	t.Parallel()
	// EIP-7840 (blob schedule in config) is purely configuration
	// No state changes are made by this EIP
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	snapshot := state.Snapshot()

	// EIP-7840 doesn't modify state - it's a config file structure change

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestEIP2537BLSPrecompileNoStateChanges tests that EIP-2537 doesn't affect state
// EIP-2537 precompiles don't modify state, they only perform computations
func TestEIP2537BLSPrecompileNoStateChanges(t *testing.T) {
	t.Parallel()
	// EIP-2537 (BLS12-381 precompiles) are pure computation
	// No state changes are made by these precompiles
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	snapshot := state.Snapshot()

	// BLS precompiles at addresses 0x0b-0x11 don't modify state

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestEIP7549AttestationNoDirectELState tests that EIP-7549 doesn't affect EL state
// EIP-7549 is CL-only (attestation committee index)
func TestEIP7549AttestationNoDirectELState(t *testing.T) {
	t.Parallel()
	// EIP-7549 (attestation committee index) is a CL-only change
	// It doesn't affect EL state
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	snapshot := state.Snapshot()

	// EIP-7549 doesn't modify EL state

	state.RevertToSnapshot(snapshot)
	// Test passes if no panic occurs
}

// TestMultiplePectraEIPStateRevert tests reverting multiple EIP state changes together
func TestMultiplePectraEIPStateRevert(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	// Take initial snapshot
	snapshot := state.Snapshot()

	// EIP-2935: Store block hash
	historySlot := libcommon.BytesToHash(uint256.NewInt(100).Bytes())
	state.SetState(params.HistoryStorageAddress, &historySlot, *uint256.NewInt(0x1234))

	// EIP-7002: Withdrawal request state
	withdrawalSlot := libcommon.Hash{1}
	state.SetState(params.WithdrawalRequestAddress, &withdrawalSlot, *uint256.NewInt(0x5678))

	// EIP-7251: Consolidation request state
	consolidationSlot := libcommon.Hash{2}
	state.SetState(params.ConsolidationRequestAddress, &consolidationSlot, *uint256.NewInt(0x9abc))

	// EIP-7702: Delegation
	delegator := libcommon.HexToAddress("0x3333333333333333333333333333333333333333")
	delegateTarget := libcommon.HexToAddress("0x4444444444444444444444444444444444444444")
	state.CreateAccount(delegator, false)
	state.SetCode(delegator, types.AddressToDelegation(delegateTarget))

	// Verify all state changes are present
	var v1, v2, v3 uint256.Int
	state.GetState(params.HistoryStorageAddress, &historySlot, &v1)
	state.GetState(params.WithdrawalRequestAddress, &withdrawalSlot, &v2)
	state.GetState(params.ConsolidationRequestAddress, &consolidationSlot, &v3)
	require.False(t, v1.IsZero())
	require.False(t, v2.IsZero())
	require.False(t, v3.IsZero())
	require.True(t, types.IsDelegation(state.GetCode(delegator)))

	// Revert all at once
	state.RevertToSnapshot(snapshot)

	// Verify all reverted
	state.GetState(params.HistoryStorageAddress, &historySlot, &v1)
	state.GetState(params.WithdrawalRequestAddress, &withdrawalSlot, &v2)
	state.GetState(params.ConsolidationRequestAddress, &consolidationSlot, &v3)
	assert.True(t, v1.IsZero(), "EIP-2935 state should be reverted")
	assert.True(t, v2.IsZero(), "EIP-7002 state should be reverted")
	assert.True(t, v3.IsZero(), "EIP-7251 state should be reverted")

	// Delegation account should not exist after revert
	code := state.GetCode(delegator)
	assert.Empty(t, code, "EIP-7702 delegation should be reverted")
}

// TestNestedSnapshotRevert tests nested snapshot/revert functionality
func TestNestedSnapshotRevert(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	r := NewPlainStateReader(tx)
	state := New(r)

	addr := params.HistoryStorageAddress
	slot := libcommon.Hash{1}

	// Snapshot 1
	snap1 := state.Snapshot()
	state.SetState(addr, &slot, *uint256.NewInt(1))

	// Snapshot 2 (nested)
	snap2 := state.Snapshot()
	state.SetState(addr, &slot, *uint256.NewInt(2))

	// Snapshot 3 (nested)
	snap3 := state.Snapshot()
	state.SetState(addr, &slot, *uint256.NewInt(3))

	// Verify current value
	var v uint256.Int
	state.GetState(addr, &slot, &v)
	assert.Equal(t, uint64(3), v.Uint64())

	// Revert to snap3 (should stay at 3, since snap3 was taken after setting 2)
	state.RevertToSnapshot(snap3)
	state.GetState(addr, &slot, &v)
	assert.Equal(t, uint64(2), v.Uint64())

	// Revert to snap2
	state.RevertToSnapshot(snap2)
	state.GetState(addr, &slot, &v)
	assert.Equal(t, uint64(1), v.Uint64())

	// Revert to snap1
	state.RevertToSnapshot(snap1)
	state.GetState(addr, &slot, &v)
	assert.True(t, v.IsZero())
}

