// Copyright 2024 The Erigon Authors
// This file is part of Erigon.

package misc

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon/core/types"
)

// TestDepositTopicHash verifies the deposit event topic hash is correctly calculated
func TestDepositTopicHash(t *testing.T) {
	t.Parallel()

	// Calculate the expected topic hash
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte("DepositEvent(bytes,bytes,bytes,bytes,bytes)"))
	expected := libcommon.BytesToHash(h.Sum(nil))

	// Verify against the hardcoded value
	assert.Equal(t, expected, depositTopic, "Deposit topic hash mismatch")
	assert.Equal(t, "0x649bbc62d0e31342afea4e5cd82d4049e7e1ee912fc0889aa790803be39038c5", depositTopic.Hex())
}

// TestDepositRequestDataLen verifies the deposit request data length is correct
func TestDepositRequestDataLen(t *testing.T) {
	t.Parallel()

	// EIP-6110 deposit request data structure:
	// pubkey: 48 bytes
	// withdrawal_credentials: 32 bytes
	// amount: 8 bytes (little-endian uint64)
	// signature: 96 bytes
	// index: 8 bytes (little-endian uint64)
	expectedLen := BLSPubKeyLen + WithdrawalCredentialsLen + 8 + BLSSigLen + 8

	assert.Equal(t, 48, BLSPubKeyLen, "BLS public key length should be 48")
	assert.Equal(t, 32, WithdrawalCredentialsLen, "Withdrawal credentials length should be 32")
	assert.Equal(t, 96, BLSSigLen, "BLS signature length should be 96")
	assert.Equal(t, 192, expectedLen, "Total deposit data length should be 192")
	assert.Equal(t, 192, types.DepositRequestDataLen, "DepositRequestDataLen should be 192")
}

// TestUnpackDepositLog tests unpacking a deposit event log
func TestUnpackDepositLog(t *testing.T) {
	t.Parallel()

	// Create test data
	pubkey := make([]byte, BLSPubKeyLen)
	for i := range pubkey {
		pubkey[i] = byte(i)
	}

	withdrawalCredentials := make([]byte, WithdrawalCredentialsLen)
	for i := range withdrawalCredentials {
		withdrawalCredentials[i] = byte(i + 100)
	}

	amount := make([]byte, 8)
	binary.LittleEndian.PutUint64(amount, 32000000000) // 32 ETH in Gwei

	signature := make([]byte, BLSSigLen)
	for i := range signature {
		signature[i] = byte(i + 50)
	}

	index := make([]byte, 8)
	binary.LittleEndian.PutUint64(index, 12345)

	// Pack using ABI
	packedData, err := DepositABI.Events["DepositEvent"].Inputs.Pack(
		pubkey,
		withdrawalCredentials,
		amount,
		signature,
		index,
	)
	require.NoError(t, err)

	// Unpack the data
	result, err := unpackDepositLog(packedData)
	require.NoError(t, err)
	require.Len(t, result, types.DepositRequestDataLen)

	// Verify each component
	offset := 0

	// Verify pubkey
	assert.Equal(t, pubkey, result[offset:offset+BLSPubKeyLen])
	offset += BLSPubKeyLen

	// Verify withdrawal credentials
	assert.Equal(t, withdrawalCredentials, result[offset:offset+WithdrawalCredentialsLen])
	offset += WithdrawalCredentialsLen

	// Verify amount
	assert.Equal(t, amount, result[offset:offset+8])
	offset += 8

	// Verify signature
	assert.Equal(t, signature, result[offset:offset+BLSSigLen])
	offset += BLSSigLen

	// Verify index
	assert.Equal(t, index, result[offset:offset+8])
}

// TestParseDepositLogs tests parsing deposit logs from receipts
func TestParseDepositLogs(t *testing.T) {
	t.Parallel()

	depositContract := libcommon.HexToAddress("0x00000000219ab540356cBB839Cbe05303d7705Fa")

	// Create test deposit data
	pubkey := make([]byte, BLSPubKeyLen)
	copy(pubkey, []byte("test_pubkey_0123456789012345678901234567"))

	withdrawalCredentials := make([]byte, WithdrawalCredentialsLen)
	copy(withdrawalCredentials, []byte("test_withdrawal_credentials_123"))

	amount := make([]byte, 8)
	binary.LittleEndian.PutUint64(amount, 32000000000)

	signature := make([]byte, BLSSigLen)
	copy(signature, []byte("test_signature_01234567890123456789012345678901234567890123456789012345678901234567890123456789012345"))

	index := make([]byte, 8)
	binary.LittleEndian.PutUint64(index, 0)

	// Pack the deposit data
	packedData, err := DepositABI.Events["DepositEvent"].Inputs.Pack(
		pubkey,
		withdrawalCredentials,
		amount,
		signature,
		index,
	)
	require.NoError(t, err)

	// Create a log with the deposit event
	logs := []*types.Log{
		{
			Address: depositContract,
			Topics:  []libcommon.Hash{depositTopic},
			Data:    packedData,
		},
	}

	// Parse the logs
	result, err := ParseDepositLogs(logs, depositContract)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, types.DepositRequestType, result.Type)
	assert.Len(t, result.RequestData, types.DepositRequestDataLen)
}

// TestParseDepositLogsEmpty tests parsing empty logs
func TestParseDepositLogsEmpty(t *testing.T) {
	t.Parallel()

	depositContract := libcommon.HexToAddress("0x00000000219ab540356cBB839Cbe05303d7705Fa")

	// Empty logs should return nil
	result, err := ParseDepositLogs([]*types.Log{}, depositContract)
	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestParseDepositLogsWrongAddress tests logs from wrong address are ignored
func TestParseDepositLogsWrongAddress(t *testing.T) {
	t.Parallel()

	depositContract := libcommon.HexToAddress("0x00000000219ab540356cBB839Cbe05303d7705Fa")
	wrongAddress := libcommon.HexToAddress("0x1111111111111111111111111111111111111111")

	// Create a log with wrong address
	logs := []*types.Log{
		{
			Address: wrongAddress,
			Topics:  []libcommon.Hash{depositTopic},
			Data:    make([]byte, 100), // Some data
		},
	}

	// Should return nil because address doesn't match
	result, err := ParseDepositLogs(logs, depositContract)
	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestParseDepositLogsWrongTopic tests logs with wrong topic are ignored
func TestParseDepositLogsWrongTopic(t *testing.T) {
	t.Parallel()

	depositContract := libcommon.HexToAddress("0x00000000219ab540356cBB839Cbe05303d7705Fa")
	wrongTopic := libcommon.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")

	// Create a log with wrong topic
	logs := []*types.Log{
		{
			Address: depositContract,
			Topics:  []libcommon.Hash{wrongTopic},
			Data:    make([]byte, 100),
		},
	}

	// Should return nil because topic doesn't match
	result, err := ParseDepositLogs(logs, depositContract)
	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestParseDepositLogsMultiple tests parsing multiple deposit logs
func TestParseDepositLogsMultiple(t *testing.T) {
	t.Parallel()

	depositContract := libcommon.HexToAddress("0x00000000219ab540356cBB839Cbe05303d7705Fa")

	// Create two deposits
	createDepositData := func(idx uint64) []byte {
		pubkey := make([]byte, BLSPubKeyLen)
		binary.BigEndian.PutUint64(pubkey[40:], idx)

		withdrawalCredentials := make([]byte, WithdrawalCredentialsLen)
		binary.BigEndian.PutUint64(withdrawalCredentials[24:], idx)

		amount := make([]byte, 8)
		binary.LittleEndian.PutUint64(amount, 32000000000)

		signature := make([]byte, BLSSigLen)
		binary.BigEndian.PutUint64(signature[88:], idx)

		index := make([]byte, 8)
		binary.LittleEndian.PutUint64(index, idx)

		packedData, _ := DepositABI.Events["DepositEvent"].Inputs.Pack(
			pubkey,
			withdrawalCredentials,
			amount,
			signature,
			index,
		)
		return packedData
	}

	logs := []*types.Log{
		{
			Address: depositContract,
			Topics:  []libcommon.Hash{depositTopic},
			Data:    createDepositData(0),
		},
		{
			Address: depositContract,
			Topics:  []libcommon.Hash{depositTopic},
			Data:    createDepositData(1),
		},
	}

	result, err := ParseDepositLogs(logs, depositContract)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, types.DepositRequestType, result.Type)
	// Two deposits should have 2 * DepositRequestDataLen bytes
	assert.Len(t, result.RequestData, 2*types.DepositRequestDataLen)
}

// TestDepositRequestType verifies the deposit request type constant
func TestDepositRequestType(t *testing.T) {
	t.Parallel()

	// EIP-6110 specifies deposit request type as 0x00
	assert.Equal(t, byte(0x00), types.DepositRequestType)
}

// TestFlatRequestEncode tests FlatRequest encoding
func TestFlatRequestEncode(t *testing.T) {
	t.Parallel()

	requestData := []byte{1, 2, 3, 4, 5}
	fr := types.FlatRequest{
		Type:        types.DepositRequestType,
		RequestData: requestData,
	}

	encoded := fr.Encode()
	assert.Equal(t, byte(0x00), encoded[0])
	assert.Equal(t, requestData, encoded[1:])
}

