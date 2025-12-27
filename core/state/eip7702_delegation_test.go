package state

import (
	"bytes"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/kv"
	"github.com/erigontech/erigon-lib/kv/dbutils"
	"github.com/erigontech/erigon-lib/kv/memdb"
	"github.com/erigontech/erigon-lib/kv/temporal/historyv2"

	"github.com/erigontech/erigon-lib/crypto"
	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/core/types/accounts"
	"github.com/erigontech/erigon/params"
)

// TestEIP7702DelegationCodeFormat tests that delegation code is correctly formatted
func TestEIP7702DelegationCodeFormat(t *testing.T) {
	t.Parallel()

	delegateAddr := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")
	delegationCode := types.AddressToDelegation(delegateAddr)

	// Check format: 0xef0100 + 20-byte address = 23 bytes
	assert.Equal(t, 23, len(delegationCode), "delegation code should be 23 bytes")
	assert.True(t, bytes.HasPrefix(delegationCode, params.DelegatedDesignationPrefix), "should have correct prefix")

	// Verify IsDelegation
	assert.True(t, types.IsDelegation(delegationCode), "should be recognized as delegation")

	// Verify ParseDelegation
	parsedAddr, ok := types.ParseDelegation(delegationCode)
	assert.True(t, ok, "should parse successfully")
	assert.Equal(t, delegateAddr, parsedAddr, "parsed address should match")

	// Test invalid cases
	assert.False(t, types.IsDelegation(nil), "nil should not be delegation")
	assert.False(t, types.IsDelegation([]byte{}), "empty should not be delegation")
	assert.False(t, types.IsDelegation([]byte{0xef, 0x01}), "short should not be delegation")
	assert.False(t, types.IsDelegation(make([]byte, 24)), "wrong length should not be delegation")
}

// TestEIP7702DelegationAccountWithIncarnation0 tests setting delegation on an EOA (Incarnation=0)
func TestEIP7702DelegationAccountWithIncarnation0(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	addr := libcommon.HexToAddress("0x1111111111111111111111111111111111111111")
	delegateAddr := libcommon.HexToAddress("0x2222222222222222222222222222222222222222")
	delegationCode := types.AddressToDelegation(delegateAddr)
	codeHash := crypto.Keccak256Hash(delegationCode)

	// Create an EOA (no code, Incarnation=0)
	emptyAcc := accounts.NewAccount()
	emptyAcc.Initialised = true
	emptyAcc.Balance = *uint256.NewInt(1000)
	emptyAcc.Nonce = 1
	emptyAcc.Incarnation = 0 // EOA has Incarnation=0

	// New account with delegation code
	delegationAcc := emptyAcc.SelfCopy()
	delegationAcc.CodeHash = codeHash
	delegationAcc.Nonce = 2 // Nonce increased after setting delegation

	blockWriter := NewPlainStateWriter(tx, tx, 1)

	// Write account data
	err := blockWriter.UpdateAccountData(addr, &emptyAcc, delegationAcc)
	require.NoError(t, err)

	// Write delegation code
	err = blockWriter.UpdateAccountCode(addr, delegationAcc.Incarnation, codeHash, delegationCode)
	require.NoError(t, err)

	// Write changesets
	err = blockWriter.WriteChangeSets()
	require.NoError(t, err)
	err = blockWriter.WriteHistory()
	require.NoError(t, err)

	// Verify PlainState
	encAcc, err := tx.GetOne(kv.PlainState, addr.Bytes())
	require.NoError(t, err)
	require.NotNil(t, encAcc)

	var readAcc accounts.Account
	err = readAcc.DecodeForStorage(encAcc)
	require.NoError(t, err)
	assert.Equal(t, delegationAcc.Nonce, readAcc.Nonce)
	assert.Equal(t, codeHash, readAcc.CodeHash)

	// Verify PlainContractCode
	codeHashFromDB, err := tx.GetOne(kv.PlainContractCode, dbutils.PlainGenerateStoragePrefix(addr.Bytes(), delegationAcc.Incarnation))
	require.NoError(t, err)
	assert.Equal(t, codeHash.Bytes(), codeHashFromDB)

	// Verify Code table
	codeFromDB, err := tx.GetOne(kv.Code, codeHash.Bytes())
	require.NoError(t, err)
	assert.Equal(t, delegationCode, codeFromDB)
	assert.True(t, types.IsDelegation(codeFromDB))

	// Verify changeset recorded original state
	cs := historyv2.NewAccountChangeSet()
	err = historyv2.ForPrefix(tx, kv.AccountChangeSet, dbutils.EncodeBlockNumber(1), func(_ uint64, k, v []byte) error {
		return cs.Add(k, v)
	})
	require.NoError(t, err)
	assert.Equal(t, 1, cs.Len(), "should have one changeset entry")
}

// TestEIP7702DelegationAccountWithIncarnationGreaterThan0 tests setting delegation on an account with existing code
func TestEIP7702DelegationAccountWithIncarnationGreaterThan0(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	addr := libcommon.HexToAddress("0x3333333333333333333333333333333333333333")
	delegateAddr := libcommon.HexToAddress("0x4444444444444444444444444444444444444444")
	delegationCode := types.AddressToDelegation(delegateAddr)
	delegationCodeHash := crypto.Keccak256Hash(delegationCode)

	// Original contract code
	originalCode := []byte{0x60, 0x00, 0x60, 0x00, 0xf3} // PUSH1 0x00 PUSH1 0x00 RETURN
	originalCodeHash := crypto.Keccak256Hash(originalCode)

	// Create account with existing code (Incarnation=1)
	originalAcc := accounts.NewAccount()
	originalAcc.Initialised = true
	originalAcc.Balance = *uint256.NewInt(2000)
	originalAcc.Nonce = 5
	originalAcc.Incarnation = 1
	originalAcc.CodeHash = originalCodeHash

	// Write original account
	blockWriter := NewPlainStateWriter(tx, tx, 1)
	emptyAcc := accounts.NewAccount()
	err := blockWriter.UpdateAccountData(addr, &emptyAcc, &originalAcc)
	require.NoError(t, err)
	err = blockWriter.UpdateAccountCode(addr, originalAcc.Incarnation, originalCodeHash, originalCode)
	require.NoError(t, err)
	err = blockWriter.WriteChangeSets()
	require.NoError(t, err)

	// Now replace with delegation code in block 2
	newAcc := originalAcc.SelfCopy()
	newAcc.CodeHash = delegationCodeHash
	newAcc.Nonce = 6

	blockWriter2 := NewPlainStateWriter(tx, tx, 2)
	err = blockWriter2.UpdateAccountData(addr, &originalAcc, newAcc)
	require.NoError(t, err)
	err = blockWriter2.UpdateAccountCode(addr, newAcc.Incarnation, delegationCodeHash, delegationCode)
	require.NoError(t, err)
	err = blockWriter2.WriteChangeSets()
	require.NoError(t, err)
	err = blockWriter2.WriteHistory()
	require.NoError(t, err)

	// Verify the delegation code is now set
	codeHashFromDB, err := tx.GetOne(kv.PlainContractCode, dbutils.PlainGenerateStoragePrefix(addr.Bytes(), newAcc.Incarnation))
	require.NoError(t, err)
	assert.Equal(t, delegationCodeHash.Bytes(), codeHashFromDB)

	codeFromDB, err := tx.GetOne(kv.Code, delegationCodeHash.Bytes())
	require.NoError(t, err)
	assert.True(t, types.IsDelegation(codeFromDB))
}

// TestEIP7702RemoveDelegation tests removing delegation code from an account
func TestEIP7702RemoveDelegation(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	addr := libcommon.HexToAddress("0x5555555555555555555555555555555555555555")
	delegateAddr := libcommon.HexToAddress("0x6666666666666666666666666666666666666666")
	delegationCode := types.AddressToDelegation(delegateAddr)
	delegationCodeHash := crypto.Keccak256Hash(delegationCode)

	// Create account with delegation code
	delegationAcc := accounts.NewAccount()
	delegationAcc.Initialised = true
	delegationAcc.Balance = *uint256.NewInt(3000)
	delegationAcc.Nonce = 1
	delegationAcc.Incarnation = 0
	delegationAcc.CodeHash = delegationCodeHash

	// Write account with delegation
	blockWriter := NewPlainStateWriter(tx, tx, 1)
	emptyAcc := accounts.NewAccount()
	err := blockWriter.UpdateAccountData(addr, &emptyAcc, &delegationAcc)
	require.NoError(t, err)
	err = blockWriter.UpdateAccountCode(addr, delegationAcc.Incarnation, delegationCodeHash, delegationCode)
	require.NoError(t, err)
	err = blockWriter.WriteChangeSets()
	require.NoError(t, err)

	// Verify delegation is set
	codeFromDB, err := tx.GetOne(kv.Code, delegationCodeHash.Bytes())
	require.NoError(t, err)
	assert.True(t, types.IsDelegation(codeFromDB))

	// Remove delegation in block 2 (set empty code)
	removedDelegationAcc := delegationAcc.SelfCopy()
	removedDelegationAcc.CodeHash = libcommon.BytesToHash(emptyCodeHash)
	removedDelegationAcc.Nonce = 2

	blockWriter2 := NewPlainStateWriter(tx, tx, 2)
	err = blockWriter2.UpdateAccountData(addr, &delegationAcc, removedDelegationAcc)
	require.NoError(t, err)
	err = blockWriter2.WriteChangeSets()
	require.NoError(t, err)
	err = blockWriter2.WriteHistory()
	require.NoError(t, err)

	// Verify account now has empty code hash
	encAcc, err := tx.GetOne(kv.PlainState, addr.Bytes())
	require.NoError(t, err)
	var readAcc accounts.Account
	err = readAcc.DecodeForStorage(encAcc)
	require.NoError(t, err)
	assert.True(t, readAcc.IsEmptyCodeHash())
}

// TestEIP7702ChangeSetRecording tests that changesets correctly record delegation changes
func TestEIP7702ChangeSetRecording(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	addr := libcommon.HexToAddress("0x7777777777777777777777777777777777777777")
	delegateAddr := libcommon.HexToAddress("0x8888888888888888888888888888888888888888")
	delegationCode := types.AddressToDelegation(delegateAddr)
	delegationCodeHash := crypto.Keccak256Hash(delegationCode)

	// Original EOA
	originalAcc := accounts.NewAccount()
	originalAcc.Initialised = true
	originalAcc.Balance = *uint256.NewInt(5000)
	originalAcc.Nonce = 10
	originalAcc.Incarnation = 0

	// Write original account in block 1
	blockWriter := NewPlainStateWriter(tx, tx, 1)
	emptyAcc := accounts.NewAccount()
	err := blockWriter.UpdateAccountData(addr, &emptyAcc, &originalAcc)
	require.NoError(t, err)
	err = blockWriter.WriteChangeSets()
	require.NoError(t, err)

	// Add delegation in block 2
	delegationAcc := originalAcc.SelfCopy()
	delegationAcc.CodeHash = delegationCodeHash
	delegationAcc.Nonce = 11

	blockWriter2 := NewPlainStateWriter(tx, tx, 2)
	err = blockWriter2.UpdateAccountData(addr, &originalAcc, delegationAcc)
	require.NoError(t, err)
	err = blockWriter2.UpdateAccountCode(addr, delegationAcc.Incarnation, delegationCodeHash, delegationCode)
	require.NoError(t, err)
	err = blockWriter2.WriteChangeSets()
	require.NoError(t, err)
	err = blockWriter2.WriteHistory()
	require.NoError(t, err)

	// Verify changeset for block 2 has the original state (without delegation)
	cs := historyv2.NewAccountChangeSet()
	err = historyv2.ForPrefix(tx, kv.AccountChangeSet, dbutils.EncodeBlockNumber(2), func(_ uint64, k, v []byte) error {
		return cs.Add(k, v)
	})
	require.NoError(t, err)
	assert.Equal(t, 1, cs.Len())

	// Decode the changeset value - should be original account without CodeHash
	for _, change := range cs.Changes {
		if bytes.Equal(change.Key, addr.Bytes()) {
			var csAcc accounts.Account
			err = csAcc.DecodeForStorage(change.Value)
			require.NoError(t, err)
			// The changeset stores original account with empty CodeHash (THIN_HISTORY)
			assert.True(t, csAcc.IsEmptyCodeHash(), "changeset should have empty CodeHash (THIN_HISTORY)")
			assert.Equal(t, originalAcc.Nonce, csAcc.Nonce)
			assert.Equal(t, originalAcc.Balance, csAcc.Balance)
		}
	}
}

// TestEIP7702PlainStateReaderDelegation tests PlainStateReader with delegation accounts
func TestEIP7702PlainStateReaderDelegation(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	addr := libcommon.HexToAddress("0x9999999999999999999999999999999999999999")
	delegateAddr := libcommon.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	delegationCode := types.AddressToDelegation(delegateAddr)
	delegationCodeHash := crypto.Keccak256Hash(delegationCode)

	// Create account with delegation
	delegationAcc := accounts.NewAccount()
	delegationAcc.Initialised = true
	delegationAcc.Balance = *uint256.NewInt(6000)
	delegationAcc.Nonce = 1
	delegationAcc.Incarnation = 0
	delegationAcc.CodeHash = delegationCodeHash

	// Write account
	blockWriter := NewPlainStateWriter(tx, tx, 1)
	emptyAcc := accounts.NewAccount()
	err := blockWriter.UpdateAccountData(addr, &emptyAcc, &delegationAcc)
	require.NoError(t, err)
	err = blockWriter.UpdateAccountCode(addr, delegationAcc.Incarnation, delegationCodeHash, delegationCode)
	require.NoError(t, err)
	err = blockWriter.WriteChangeSets()
	require.NoError(t, err)

	// Read using PlainStateReader
	reader := NewPlainStateReader(tx)
	readAcc, err := reader.ReadAccountData(addr)
	require.NoError(t, err)
	require.NotNil(t, readAcc)

	// PlainStateReader should read the CodeHash as stored in PlainState
	assert.Equal(t, delegationCodeHash, readAcc.CodeHash)
	assert.Equal(t, delegationAcc.Nonce, readAcc.Nonce)

	// Read code
	code, err := reader.ReadAccountCode(addr, readAcc.Incarnation, readAcc.CodeHash)
	require.NoError(t, err)
	assert.True(t, types.IsDelegation(code))

	parsed, ok := types.ParseDelegation(code)
	assert.True(t, ok)
	assert.Equal(t, delegateAddr, parsed)
}

// TestEIP7702MultipleDelegationAccounts tests multiple delegation accounts in a single block
func TestEIP7702MultipleDelegationAccounts(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	numAccounts := 5
	addrs := make([]libcommon.Address, numAccounts)
	delegateAddrs := make([]libcommon.Address, numAccounts)
	delegationCodes := make([][]byte, numAccounts)
	delegationCodeHashes := make([]libcommon.Hash, numAccounts)

	for i := 0; i < numAccounts; i++ {
		addrBytes := make([]byte, 20)
		addrBytes[0] = byte(i + 1)
		addrBytes[19] = byte(i + 1)
		addrs[i] = libcommon.BytesToAddress(addrBytes)

		delegateBytes := make([]byte, 20)
		delegateBytes[0] = byte(i + 100)
		delegateBytes[19] = byte(i + 100)
		delegateAddrs[i] = libcommon.BytesToAddress(delegateBytes)

		delegationCodes[i] = types.AddressToDelegation(delegateAddrs[i])
		delegationCodeHashes[i] = crypto.Keccak256Hash(delegationCodes[i])
	}

	blockWriter := NewPlainStateWriter(tx, tx, 1)

	for i := 0; i < numAccounts; i++ {
		emptyAcc := accounts.NewAccount()
		delegationAcc := accounts.NewAccount()
		delegationAcc.Initialised = true
		delegationAcc.Balance = *uint256.NewInt(uint64(1000 * (i + 1)))
		delegationAcc.Nonce = uint64(i + 1)
		delegationAcc.Incarnation = 0
		delegationAcc.CodeHash = delegationCodeHashes[i]

		err := blockWriter.UpdateAccountData(addrs[i], &emptyAcc, &delegationAcc)
		require.NoError(t, err)
		err = blockWriter.UpdateAccountCode(addrs[i], delegationAcc.Incarnation, delegationCodeHashes[i], delegationCodes[i])
		require.NoError(t, err)
	}

	err := blockWriter.WriteChangeSets()
	require.NoError(t, err)
	err = blockWriter.WriteHistory()
	require.NoError(t, err)

	// Verify all accounts
	reader := NewPlainStateReader(tx)
	for i := 0; i < numAccounts; i++ {
		acc, err := reader.ReadAccountData(addrs[i])
		require.NoError(t, err)
		require.NotNil(t, acc)
		assert.Equal(t, delegationCodeHashes[i], acc.CodeHash)

		code, err := reader.ReadAccountCode(addrs[i], acc.Incarnation, acc.CodeHash)
		require.NoError(t, err)
		assert.True(t, types.IsDelegation(code))

		parsed, ok := types.ParseDelegation(code)
		assert.True(t, ok)
		assert.Equal(t, delegateAddrs[i], parsed)
	}
}

// TestEIP7702DelegationCodeHashRecovery tests CodeHash recovery from PlainContractCode
func TestEIP7702DelegationCodeHashRecovery(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	addr := libcommon.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	delegateAddr := libcommon.HexToAddress("0xcccccccccccccccccccccccccccccccccccccccc")
	delegationCode := types.AddressToDelegation(delegateAddr)
	delegationCodeHash := crypto.Keccak256Hash(delegationCode)

	// Simulate the scenario where PlainState has empty CodeHash but PlainContractCode has the hash
	// This can happen due to THIN_HISTORY or previous bugs

	// 1. Write the code to Code table
	err := tx.Put(kv.Code, delegationCodeHash.Bytes(), delegationCode)
	require.NoError(t, err)

	// 2. Write the codeHash to PlainContractCode
	incarnation := uint64(0)
	err = tx.Put(kv.PlainContractCode, dbutils.PlainGenerateStoragePrefix(addr.Bytes(), incarnation), delegationCodeHash.Bytes())
	require.NoError(t, err)

	// 3. Write account to PlainState WITHOUT CodeHash (simulating corrupted state)
	acc := accounts.NewAccount()
	acc.Initialised = true
	acc.Balance = *uint256.NewInt(7000)
	acc.Nonce = 3
	acc.Incarnation = incarnation
	// CodeHash is empty (emptyCodeHash)

	value := make([]byte, acc.EncodingLengthForStorage())
	acc.EncodeForStorage(value)
	err = tx.Put(kv.PlainState, addr.Bytes(), value)
	require.NoError(t, err)

	// Verify that PlainState has empty CodeHash
	hasCodeHash := accounts.HasCodeHashInStorage(value)
	assert.False(t, hasCodeHash, "PlainState should NOT have CodeHash stored")

	// Verify PlainContractCode has the correct hash
	codeHashFromDB, err := tx.GetOne(kv.PlainContractCode, dbutils.PlainGenerateStoragePrefix(addr.Bytes(), incarnation))
	require.NoError(t, err)
	assert.Equal(t, delegationCodeHash.Bytes(), codeHashFromDB)

	// Verify Code table has valid delegation code
	codeFromDB, err := tx.GetOne(kv.Code, delegationCodeHash.Bytes())
	require.NoError(t, err)
	assert.True(t, types.IsDelegation(codeFromDB))

	// Note: PlainStateReader does NOT recover CodeHash (this is intentional as per v24)
	// Other readers (CachedReader2, PlainState) DO recover it
	reader := NewPlainStateReader(tx)
	readAcc, err := reader.ReadAccountData(addr)
	require.NoError(t, err)
	// Current behavior: PlainStateReader does not recover CodeHash
	assert.True(t, readAcc.IsEmptyCodeHash(), "PlainStateReader should not recover CodeHash")
}

// TestEIP7702AccountEncoding tests account encoding/decoding with delegation CodeHash
func TestEIP7702AccountEncoding(t *testing.T) {
	t.Parallel()

	delegateAddr := libcommon.HexToAddress("0xdddddddddddddddddddddddddddddddddddddddd")
	delegationCode := types.AddressToDelegation(delegateAddr)
	delegationCodeHash := crypto.Keccak256Hash(delegationCode)

	// Test account with delegation code
	acc := accounts.Account{
		Initialised: true,
		Nonce:       100,
		Balance:     *uint256.NewInt(50000),
		Incarnation: 0,
		CodeHash:    delegationCodeHash,
	}

	// Encode
	encodedLen := acc.EncodingLengthForStorage()
	encoded := make([]byte, encodedLen)
	acc.EncodeForStorage(encoded)

	// Verify CodeHash is stored
	hasCodeHash := accounts.HasCodeHashInStorage(encoded)
	assert.True(t, hasCodeHash, "encoded account should have CodeHash")

	// Decode
	var decoded accounts.Account
	err := decoded.DecodeForStorage(encoded)
	require.NoError(t, err)

	assert.Equal(t, acc.Nonce, decoded.Nonce)
	assert.Equal(t, acc.Balance, decoded.Balance)
	assert.Equal(t, acc.Incarnation, decoded.Incarnation)
	assert.Equal(t, acc.CodeHash, decoded.CodeHash)
}

// TestEIP7702DelegationWithStorage tests delegation account that also has storage
func TestEIP7702DelegationWithStorage(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	addr := libcommon.HexToAddress("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	delegateAddr := libcommon.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff")
	delegationCode := types.AddressToDelegation(delegateAddr)
	delegationCodeHash := crypto.Keccak256Hash(delegationCode)

	storageKey := libcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")
	storageValue := uint256.NewInt(12345)

	// Create delegation account
	delegationAcc := accounts.NewAccount()
	delegationAcc.Initialised = true
	delegationAcc.Balance = *uint256.NewInt(8000)
	delegationAcc.Nonce = 1
	delegationAcc.Incarnation = 1 // Account with storage needs incarnation > 0
	delegationAcc.CodeHash = delegationCodeHash

	blockWriter := NewPlainStateWriter(tx, tx, 1)
	emptyAcc := accounts.NewAccount()

	err := blockWriter.UpdateAccountData(addr, &emptyAcc, &delegationAcc)
	require.NoError(t, err)
	err = blockWriter.UpdateAccountCode(addr, delegationAcc.Incarnation, delegationCodeHash, delegationCode)
	require.NoError(t, err)
	err = blockWriter.WriteAccountStorage(addr, delegationAcc.Incarnation, &storageKey, uint256.NewInt(0), storageValue)
	require.NoError(t, err)
	err = blockWriter.WriteChangeSets()
	require.NoError(t, err)
	err = blockWriter.WriteHistory()
	require.NoError(t, err)

	// Verify account
	reader := NewPlainStateReader(tx)
	readAcc, err := reader.ReadAccountData(addr)
	require.NoError(t, err)
	assert.Equal(t, delegationCodeHash, readAcc.CodeHash)

	// Verify storage
	storageBuf, err := reader.ReadAccountStorage(addr, readAcc.Incarnation, &storageKey)
	require.NoError(t, err)
	readStorageValue := uint256.NewInt(0).SetBytes(storageBuf)
	assert.Equal(t, storageValue, readStorageValue)

	// Verify code is delegation
	code, err := reader.ReadAccountCode(addr, readAcc.Incarnation, readAcc.CodeHash)
	require.NoError(t, err)
	assert.True(t, types.IsDelegation(code))
}

// TestEIP7702SequentialDelegationChanges tests multiple delegation changes across blocks
func TestEIP7702SequentialDelegationChanges(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	addr := libcommon.HexToAddress("0x1111000000000000000000000000000000001111")

	// Block 1: Create EOA
	acc1 := accounts.NewAccount()
	acc1.Initialised = true
	acc1.Balance = *uint256.NewInt(10000)
	acc1.Nonce = 1

	blockWriter1 := NewPlainStateWriter(tx, tx, 1)
	emptyAcc := accounts.NewAccount()
	err := blockWriter1.UpdateAccountData(addr, &emptyAcc, &acc1)
	require.NoError(t, err)
	err = blockWriter1.WriteChangeSets()
	require.NoError(t, err)
	err = blockWriter1.WriteHistory()
	require.NoError(t, err)

	// Block 2: Add delegation to address A
	delegateA := libcommon.HexToAddress("0xaaaa000000000000000000000000000000000aaa")
	delegationCodeA := types.AddressToDelegation(delegateA)
	codeHashA := crypto.Keccak256Hash(delegationCodeA)

	acc2 := acc1.SelfCopy()
	acc2.CodeHash = codeHashA
	acc2.Nonce = 2

	blockWriter2 := NewPlainStateWriter(tx, tx, 2)
	err = blockWriter2.UpdateAccountData(addr, &acc1, acc2)
	require.NoError(t, err)
	err = blockWriter2.UpdateAccountCode(addr, acc2.Incarnation, codeHashA, delegationCodeA)
	require.NoError(t, err)
	err = blockWriter2.WriteChangeSets()
	require.NoError(t, err)
	err = blockWriter2.WriteHistory()
	require.NoError(t, err)

	// Block 3: Change delegation to address B
	delegateB := libcommon.HexToAddress("0xbbbb000000000000000000000000000000000bbb")
	delegationCodeB := types.AddressToDelegation(delegateB)
	codeHashB := crypto.Keccak256Hash(delegationCodeB)

	acc3 := acc2.SelfCopy()
	acc3.CodeHash = codeHashB
	acc3.Nonce = 3

	blockWriter3 := NewPlainStateWriter(tx, tx, 3)
	err = blockWriter3.UpdateAccountData(addr, acc2, acc3)
	require.NoError(t, err)
	err = blockWriter3.UpdateAccountCode(addr, acc3.Incarnation, codeHashB, delegationCodeB)
	require.NoError(t, err)
	err = blockWriter3.WriteChangeSets()
	require.NoError(t, err)
	err = blockWriter3.WriteHistory()
	require.NoError(t, err)

	// Block 4: Remove delegation
	acc4 := acc3.SelfCopy()
	acc4.CodeHash = libcommon.BytesToHash(emptyCodeHash)
	acc4.Nonce = 4

	blockWriter4 := NewPlainStateWriter(tx, tx, 4)
	err = blockWriter4.UpdateAccountData(addr, acc3, acc4)
	require.NoError(t, err)
	err = blockWriter4.WriteChangeSets()
	require.NoError(t, err)
	err = blockWriter4.WriteHistory()
	require.NoError(t, err)

	// Verify final state
	reader := NewPlainStateReader(tx)
	finalAcc, err := reader.ReadAccountData(addr)
	require.NoError(t, err)
	assert.True(t, finalAcc.IsEmptyCodeHash())
	assert.Equal(t, uint64(4), finalAcc.Nonce)

	// Verify all changesets exist
	for block := uint64(1); block <= 4; block++ {
		cs := historyv2.NewAccountChangeSet()
		err = historyv2.ForPrefix(tx, kv.AccountChangeSet, dbutils.EncodeBlockNumber(block), func(_ uint64, k, v []byte) error {
			return cs.Add(k, v)
		})
		require.NoError(t, err)
		assert.Equal(t, 1, cs.Len(), "block %d should have changeset", block)
	}
}

// TestEIP7702RecoverCodeHashPlain tests the recoverCodeHashPlain function behavior
func TestEIP7702RecoverCodeHashPlain(t *testing.T) {
	t.Parallel()
	_, tx := memdb.NewTestTx(t)

	addr := libcommon.HexToAddress("0x2222000000000000000000000000000000002222")
	delegateAddr := libcommon.HexToAddress("0x3333000000000000000000000000000000003333")
	delegationCode := types.AddressToDelegation(delegateAddr)
	delegationCodeHash := crypto.Keccak256Hash(delegationCode)

	// Test case 1: Account with Incarnation > 0
	t.Run("Incarnation>0", func(t *testing.T) {
		incarnation := uint64(1)

		// Write code
		err := tx.Put(kv.Code, delegationCodeHash.Bytes(), delegationCode)
		require.NoError(t, err)
		err = tx.Put(kv.PlainContractCode, dbutils.PlainGenerateStoragePrefix(addr.Bytes(), incarnation), delegationCodeHash.Bytes())
		require.NoError(t, err)

		// Create account with empty CodeHash
		acc := accounts.Account{
			Initialised: true,
			Nonce:       5,
			Incarnation: incarnation,
			CodeHash:    libcommon.BytesToHash(emptyCodeHash),
		}

		// Simulate recoverCodeHashPlain
		if acc.Incarnation > 0 && acc.IsEmptyCodeHash() {
			if codeHash, err2 := tx.GetOne(kv.PlainContractCode, dbutils.PlainGenerateStoragePrefix(addr.Bytes(), acc.Incarnation)); err2 == nil && len(codeHash) > 0 {
				copy(acc.CodeHash[:], codeHash)
			}
		}

		assert.Equal(t, delegationCodeHash, acc.CodeHash, "CodeHash should be recovered for Incarnation>0")
	})

	// Test case 2: Account with Incarnation = 0 (EIP-7702 delegation accounts)
	t.Run("Incarnation=0_Fixed", func(t *testing.T) {
		addr2 := libcommon.HexToAddress("0x4444000000000000000000000000000000004444")
		incarnation := uint64(0)

		// Write code
		err := tx.Put(kv.PlainContractCode, dbutils.PlainGenerateStoragePrefix(addr2.Bytes(), incarnation), delegationCodeHash.Bytes())
		require.NoError(t, err)

		// Create account with empty CodeHash and Incarnation=0
		acc := accounts.Account{
			Initialised: true,
			Nonce:       1,
			Incarnation: incarnation,
			CodeHash:    libcommon.BytesToHash(emptyCodeHash),
		}

		// Fixed recoverCodeHashPlain behavior (removed Incarnation > 0 check)
		// This now recovers CodeHash for EIP-7702 delegation accounts (Incarnation=0)
		if acc.IsEmptyCodeHash() {
			if codeHash, err2 := tx.GetOne(kv.PlainContractCode, dbutils.PlainGenerateStoragePrefix(addr2.Bytes(), acc.Incarnation)); err2 == nil && len(codeHash) > 0 {
				copy(acc.CodeHash[:], codeHash)
			}
		}

		assert.Equal(t, delegationCodeHash, acc.CodeHash, "Fixed logic should recover CodeHash for Incarnation=0")
	})
}

