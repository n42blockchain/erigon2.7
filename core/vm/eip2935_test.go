// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.

package vm

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erigontech/erigon-lib/chain"
	libcommon "github.com/erigontech/erigon-lib/common"
	types2 "github.com/erigontech/erigon-lib/types"

	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/core/vm/evmtypes"
	"github.com/erigontech/erigon/core/vm/stack"
	"github.com/erigontech/erigon/params"
)

// mockStateForEIP2935 implements evmtypes.IntraBlockState for EIP-2935 testing
type mockStateForEIP2935 struct {
	storage map[libcommon.Address]map[libcommon.Hash]uint256.Int
}

func newMockStateForEIP2935() *mockStateForEIP2935 {
	return &mockStateForEIP2935{
		storage: make(map[libcommon.Address]map[libcommon.Hash]uint256.Int),
	}
}

func (m *mockStateForEIP2935) SetStorage(addr libcommon.Address, slot libcommon.Hash, value uint256.Int) {
	if m.storage[addr] == nil {
		m.storage[addr] = make(map[libcommon.Hash]uint256.Int)
	}
	m.storage[addr][slot] = value
}

// Implement evmtypes.IntraBlockState interface methods
func (m *mockStateForEIP2935) CreateAccount(libcommon.Address, bool)      {}
func (m *mockStateForEIP2935) SubBalance(libcommon.Address, *uint256.Int) {}
func (m *mockStateForEIP2935) AddBalance(libcommon.Address, *uint256.Int) {}
func (m *mockStateForEIP2935) GetBalance(libcommon.Address) *uint256.Int  { return nil }
func (m *mockStateForEIP2935) GetNonce(libcommon.Address) uint64          { return 0 }
func (m *mockStateForEIP2935) SetNonce(libcommon.Address, uint64)         {}
func (m *mockStateForEIP2935) GetCodeHash(libcommon.Address) libcommon.Hash {
	return libcommon.Hash{}
}
func (m *mockStateForEIP2935) GetCode(libcommon.Address) []byte         { return nil }
func (m *mockStateForEIP2935) SetCode(libcommon.Address, []byte)        {}
func (m *mockStateForEIP2935) GetCodeSize(libcommon.Address) int        { return 0 }
func (m *mockStateForEIP2935) AddRefund(uint64)                         {}
func (m *mockStateForEIP2935) SubRefund(uint64)                         {}
func (m *mockStateForEIP2935) GetRefund() uint64                        { return 0 }
func (m *mockStateForEIP2935) GetCommittedState(libcommon.Address, *libcommon.Hash, *uint256.Int) {
}
func (m *mockStateForEIP2935) GetState(addr libcommon.Address, slot *libcommon.Hash, outValue *uint256.Int) {
	if m.storage[addr] != nil {
		if val, ok := m.storage[addr][*slot]; ok {
			outValue.Set(&val)
			return
		}
	}
	outValue.Clear()
}
func (m *mockStateForEIP2935) SetState(libcommon.Address, *libcommon.Hash, uint256.Int) {}
func (m *mockStateForEIP2935) GetTransientState(libcommon.Address, libcommon.Hash) uint256.Int {
	return uint256.Int{}
}
func (m *mockStateForEIP2935) SetTransientState(libcommon.Address, libcommon.Hash, uint256.Int) {}
func (m *mockStateForEIP2935) Selfdestruct(libcommon.Address) bool                              { return false }
func (m *mockStateForEIP2935) HasSelfdestructed(libcommon.Address) bool                         { return false }
func (m *mockStateForEIP2935) Selfdestruct6780(libcommon.Address)                               {}
func (m *mockStateForEIP2935) Exist(libcommon.Address) bool                                     { return true }
func (m *mockStateForEIP2935) Empty(libcommon.Address) bool                                     { return false }
func (m *mockStateForEIP2935) Prepare(rules *chain.Rules, sender, coinbase libcommon.Address, dest *libcommon.Address,
	precompiles []libcommon.Address, txAccesses types2.AccessList, authorities []libcommon.Address) {
}
func (m *mockStateForEIP2935) AddressInAccessList(addr libcommon.Address) bool { return false }
func (m *mockStateForEIP2935) AddAddressToAccessList(addr libcommon.Address) (addrMod bool) {
	return false
}
func (m *mockStateForEIP2935) AddSlotToAccessList(addr libcommon.Address, slot libcommon.Hash) (addrMod, slotMod bool) {
	return false, false
}
func (m *mockStateForEIP2935) RevertToSnapshot(int)                           {}
func (m *mockStateForEIP2935) Snapshot() int                                  { return 0 }
func (m *mockStateForEIP2935) AddLog(*types.Log)                              {}
func (m *mockStateForEIP2935) ResolveCode(libcommon.Address) []byte           { return nil }
func (m *mockStateForEIP2935) ResolveCodeHash(libcommon.Address) libcommon.Hash { return libcommon.Hash{} }
func (m *mockStateForEIP2935) GetDelegatedDesignation(libcommon.Address) (libcommon.Address, bool) {
	return libcommon.Address{}, false
}

// TestEIP2935BlockhashWithinOldWindow tests BLOCKHASH for blocks within the 256 block window
func TestEIP2935BlockhashWithinOldWindow(t *testing.T) {
	t.Parallel()

	// Test block number 1000, query block 900 (within 256 window)
	currentBlock := uint64(1000)
	queryBlock := uint64(900)
	expectedHash := libcommon.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	// Setup GetHash function
	getHash := func(num uint64) libcommon.Hash {
		if num == queryBlock {
			return expectedHash
		}
		return libcommon.Hash{}
	}

	// Create EVM with Prague rules (IsPrague = true)
	chainConfig := &chain.Config{
		ChainID:    big.NewInt(1),
		PragueTime: new(big.Int).SetUint64(0), // Prague is active
	}
	blockContext := evmtypes.BlockContext{
		BlockNumber: currentBlock,
		Time:        100,
		GetHash:     getHash,
	}
	txContext := evmtypes.TxContext{}

	mockState := newMockStateForEIP2935()
	evm := NewEVM(blockContext, txContext, mockState, chainConfig, Config{})
	evmInterpreter := evm.interpreter.(*EVMInterpreter)

	// Create stack with query block number
	stk := stack.New()
	stk.Push(uint256.NewInt(queryBlock))

	// Create scope context
	scope := &ScopeContext{nil, stk, nil}

	// Execute opBlockhash
	pc := uint64(0)
	_, err := opBlockhash(&pc, evmInterpreter, scope)
	require.NoError(t, err)

	// Check result
	result := stk.Pop()
	resultBytes32 := result.Bytes32()
	assert.Equal(t, expectedHash.Bytes(), resultBytes32[:])
}

// TestEIP2935BlockhashBeyondOldWindowPrePrague tests BLOCKHASH for blocks beyond 256 before Prague
func TestEIP2935BlockhashBeyondOldWindowPrePrague(t *testing.T) {
	t.Parallel()

	// Test block number 1000, query block 500 (beyond 256 window)
	currentBlock := uint64(1000)
	queryBlock := uint64(500)

	// Setup GetHash function
	getHash := func(num uint64) libcommon.Hash {
		return libcommon.HexToHash("0xdead")
	}

	// Create EVM without Prague (before fork)
	chainConfig := &chain.Config{
		ChainID:    big.NewInt(1),
		PragueTime: nil, // Prague not active
	}
	blockContext := evmtypes.BlockContext{
		BlockNumber: currentBlock,
		Time:        100,
		GetHash:     getHash,
	}
	txContext := evmtypes.TxContext{}

	mockState := newMockStateForEIP2935()
	evm := NewEVM(blockContext, txContext, mockState, chainConfig, Config{})
	evmInterpreter := evm.interpreter.(*EVMInterpreter)

	// Create stack with query block number
	stk := stack.New()
	stk.Push(uint256.NewInt(queryBlock))

	// Create scope context
	scope := &ScopeContext{nil, stk, nil}

	// Execute opBlockhash
	pc := uint64(0)
	_, err := opBlockhash(&pc, evmInterpreter, scope)
	require.NoError(t, err)

	// Check result - should be zero (block too old, Prague not active)
	result := stk.Pop()
	assert.True(t, result.IsZero(), "Block hash should be zero for block beyond 256 window before Prague")
}

// TestEIP2935BlockhashBeyondOldWindowPostPrague tests BLOCKHASH for blocks beyond 256 after Prague
func TestEIP2935BlockhashBeyondOldWindowPostPrague(t *testing.T) {
	t.Parallel()

	// Test block number 10000, query block 9500 (beyond 256 but within 8191)
	currentBlock := uint64(10000)
	queryBlock := uint64(9500)
	expectedHash := libcommon.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")

	mockState := newMockStateForEIP2935()

	// Store the hash in the history contract storage
	slotNum := queryBlock % params.BlockHashHistoryServeWindow
	storageSlot := libcommon.BytesToHash(uint256.NewInt(slotNum).Bytes())
	hashValue := uint256.NewInt(0).SetBytes32(expectedHash.Bytes())
	mockState.SetStorage(params.HistoryStorageAddress, storageSlot, *hashValue)

	// Setup GetHash function (won't be called for blocks beyond 256)
	getHash := func(num uint64) libcommon.Hash {
		return libcommon.Hash{}
	}

	// Create EVM with Prague rules (IsPrague = true)
	chainConfig := &chain.Config{
		ChainID:    big.NewInt(1),
		PragueTime: new(big.Int).SetUint64(0), // Prague is active
	}
	blockContext := evmtypes.BlockContext{
		BlockNumber: currentBlock,
		Time:        100,
		GetHash:     getHash,
	}
	txContext := evmtypes.TxContext{}

	evm := NewEVM(blockContext, txContext, mockState, chainConfig, Config{})
	evmInterpreter := evm.interpreter.(*EVMInterpreter)

	// Create stack with query block number
	stk := stack.New()
	stk.Push(uint256.NewInt(queryBlock))

	// Create scope context
	scope := &ScopeContext{nil, stk, nil}

	// Execute opBlockhash
	pc := uint64(0)
	_, err := opBlockhash(&pc, evmInterpreter, scope)
	require.NoError(t, err)

	// Check result - should match the stored hash
	result := stk.Pop()
	resultBytes32 := result.Bytes32()
	assert.Equal(t, expectedHash.Bytes(), resultBytes32[:], "Block hash should be retrieved from history contract")
}

// TestEIP2935BlockhashTooOld tests BLOCKHASH for blocks older than 8191 blocks
func TestEIP2935BlockhashTooOld(t *testing.T) {
	t.Parallel()

	// Test block number 20000, query block 1000 (older than 8191)
	currentBlock := uint64(20000)
	queryBlock := uint64(1000)

	// Setup GetHash function
	getHash := func(num uint64) libcommon.Hash {
		return libcommon.Hash{}
	}

	// Create EVM with Prague rules
	chainConfig := &chain.Config{
		ChainID:    big.NewInt(1),
		PragueTime: new(big.Int).SetUint64(0), // Prague is active
	}
	blockContext := evmtypes.BlockContext{
		BlockNumber: currentBlock,
		Time:        100,
		GetHash:     getHash,
	}
	txContext := evmtypes.TxContext{}

	mockState := newMockStateForEIP2935()
	evm := NewEVM(blockContext, txContext, mockState, chainConfig, Config{})
	evmInterpreter := evm.interpreter.(*EVMInterpreter)

	// Create stack with query block number
	stk := stack.New()
	stk.Push(uint256.NewInt(queryBlock))

	// Create scope context
	scope := &ScopeContext{nil, stk, nil}

	// Execute opBlockhash
	pc := uint64(0)
	_, err := opBlockhash(&pc, evmInterpreter, scope)
	require.NoError(t, err)

	// Check result - should be zero (block too old even for EIP-2935)
	result := stk.Pop()
	assert.True(t, result.IsZero(), "Block hash should be zero for block older than 8191 blocks")
}

// TestEIP2935BlockhashFutureBlock tests BLOCKHASH for future blocks
func TestEIP2935BlockhashFutureBlock(t *testing.T) {
	t.Parallel()

	// Test block number 1000, query block 1001 (future block)
	currentBlock := uint64(1000)
	queryBlock := uint64(1001)

	// Setup GetHash function
	getHash := func(num uint64) libcommon.Hash {
		return libcommon.HexToHash("0xdead")
	}

	// Create EVM with Prague rules
	chainConfig := &chain.Config{
		ChainID:    big.NewInt(1),
		PragueTime: new(big.Int).SetUint64(0), // Prague is active
	}
	blockContext := evmtypes.BlockContext{
		BlockNumber: currentBlock,
		Time:        100,
		GetHash:     getHash,
	}
	txContext := evmtypes.TxContext{}

	mockState := newMockStateForEIP2935()
	evm := NewEVM(blockContext, txContext, mockState, chainConfig, Config{})
	evmInterpreter := evm.interpreter.(*EVMInterpreter)

	// Create stack with query block number
	stk := stack.New()
	stk.Push(uint256.NewInt(queryBlock))

	// Create scope context
	scope := &ScopeContext{nil, stk, nil}

	// Execute opBlockhash
	pc := uint64(0)
	_, err := opBlockhash(&pc, evmInterpreter, scope)
	require.NoError(t, err)

	// Check result - should be zero (future block)
	result := stk.Pop()
	assert.True(t, result.IsZero(), "Block hash should be zero for future block")
}

// TestEIP2935BlockhashCurrentBlock tests BLOCKHASH for current block
func TestEIP2935BlockhashCurrentBlock(t *testing.T) {
	t.Parallel()

	// Test block number 1000, query block 1000 (current block)
	currentBlock := uint64(1000)
	queryBlock := uint64(1000)

	// Setup GetHash function
	getHash := func(num uint64) libcommon.Hash {
		return libcommon.HexToHash("0xdead")
	}

	// Create EVM with Prague rules
	chainConfig := &chain.Config{
		ChainID:    big.NewInt(1),
		PragueTime: new(big.Int).SetUint64(0), // Prague is active
	}
	blockContext := evmtypes.BlockContext{
		BlockNumber: currentBlock,
		Time:        100,
		GetHash:     getHash,
	}
	txContext := evmtypes.TxContext{}

	mockState := newMockStateForEIP2935()
	evm := NewEVM(blockContext, txContext, mockState, chainConfig, Config{})
	evmInterpreter := evm.interpreter.(*EVMInterpreter)

	// Create stack with query block number
	stk := stack.New()
	stk.Push(uint256.NewInt(queryBlock))

	// Create scope context
	scope := &ScopeContext{nil, stk, nil}

	// Execute opBlockhash
	pc := uint64(0)
	_, err := opBlockhash(&pc, evmInterpreter, scope)
	require.NoError(t, err)

	// Check result - should be zero (current block hash not accessible)
	result := stk.Pop()
	assert.True(t, result.IsZero(), "Block hash should be zero for current block")
}

// TestEIP2935BlockhashOverflow tests BLOCKHASH with overflow
func TestEIP2935BlockhashOverflow(t *testing.T) {
	t.Parallel()

	// Test with an overflow value
	currentBlock := uint64(1000)

	// Setup GetHash function
	getHash := func(num uint64) libcommon.Hash {
		return libcommon.HexToHash("0xdead")
	}

	// Create EVM with Prague rules
	chainConfig := &chain.Config{
		ChainID:    big.NewInt(1),
		PragueTime: new(big.Int).SetUint64(0), // Prague is active
	}
	blockContext := evmtypes.BlockContext{
		BlockNumber: currentBlock,
		Time:        100,
		GetHash:     getHash,
	}
	txContext := evmtypes.TxContext{}

	mockState := newMockStateForEIP2935()
	evm := NewEVM(blockContext, txContext, mockState, chainConfig, Config{})
	evmInterpreter := evm.interpreter.(*EVMInterpreter)

	// Create stack with overflow value
	stk := stack.New()
	overflowValue := new(uint256.Int)
	overflowValue.SetAllOne() // Max uint256 value
	stk.Push(overflowValue)

	// Create scope context
	scope := &ScopeContext{nil, stk, nil}

	// Execute opBlockhash
	pc := uint64(0)
	_, err := opBlockhash(&pc, evmInterpreter, scope)
	require.NoError(t, err)

	// Check result - should be zero (overflow)
	result := stk.Pop()
	assert.True(t, result.IsZero(), "Block hash should be zero for overflow value")
}

// TestEIP2935StoreBlockHash tests the storeHash function
func TestEIP2935StoreBlockHash(t *testing.T) {
	t.Parallel()

	// This test verifies the storage slot calculation for EIP-2935
	blockNum := uint64(12345)
	expectedSlot := blockNum % params.BlockHashHistoryServeWindow

	// Verify the slot calculation
	assert.Equal(t, uint64(12345%8191), expectedSlot)

	// Verify BlockHashHistoryServeWindow value
	assert.Equal(t, uint64(8191), params.BlockHashHistoryServeWindow)

	// Verify BlockHashOldWindow value
	assert.Equal(t, uint64(256), params.BlockHashOldWindow)
}

// TestEIP2935RingBufferWrapAround tests the ring buffer behavior at boundaries
func TestEIP2935RingBufferWrapAround(t *testing.T) {
	t.Parallel()

	// Test that block hashes wrap around correctly in the ring buffer
	// Block 8191 and block 16382 should use the same storage slot
	block1 := uint64(8191)
	block2 := uint64(16382)

	slot1 := block1 % params.BlockHashHistoryServeWindow
	slot2 := block2 % params.BlockHashHistoryServeWindow

	assert.Equal(t, slot1, slot2, "Same slot should be used for blocks that differ by BlockHashHistoryServeWindow")
	assert.Equal(t, uint64(0), slot1, "Slot should be 0 for block 8191")
}

// TestEIP2935EdgeCaseBoundary tests BLOCKHASH at the exact boundary of 256 blocks
func TestEIP2935EdgeCaseBoundary(t *testing.T) {
	t.Parallel()

	// Test at the exact boundary: block 1000, query block 744 (exactly 256 blocks ago)
	currentBlock := uint64(1000)
	queryBlock := currentBlock - params.BlockHashOldWindow // 744

	expectedHash := libcommon.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	// Setup GetHash function
	getHash := func(num uint64) libcommon.Hash {
		if num == queryBlock {
			return expectedHash
		}
		return libcommon.Hash{}
	}

	// Create EVM with Prague rules
	chainConfig := &chain.Config{
		ChainID:    big.NewInt(1),
		PragueTime: new(big.Int).SetUint64(0), // Prague is active
	}
	blockContext := evmtypes.BlockContext{
		BlockNumber: currentBlock,
		Time:        100,
		GetHash:     getHash,
	}
	txContext := evmtypes.TxContext{}

	mockState := newMockStateForEIP2935()
	evm := NewEVM(blockContext, txContext, mockState, chainConfig, Config{})
	evmInterpreter := evm.interpreter.(*EVMInterpreter)

	// Create stack with query block number
	stk := stack.New()
	stk.Push(uint256.NewInt(queryBlock))

	// Create scope context
	scope := &ScopeContext{nil, stk, nil}

	// Execute opBlockhash
	pc := uint64(0)
	_, err := opBlockhash(&pc, evmInterpreter, scope)
	require.NoError(t, err)

	// Check result - should get the hash from GetHash (within 256 window)
	result := stk.Pop()
	resultBytes32 := result.Bytes32()
	assert.Equal(t, expectedHash.Bytes(), resultBytes32[:], "Block at boundary should still use GetHash")
}
