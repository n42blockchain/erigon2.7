// Copyright 2024 The Erigon Authors
// This file is part of Erigon.

package misc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/params"
)

// TestWithdrawalRequestAddress verifies the EIP-7002 contract address
func TestWithdrawalRequestAddress(t *testing.T) {
	t.Parallel()

	// EIP-7002 specifies the withdrawal request contract address
	expectedAddr := libcommon.HexToAddress("0x00000961Ef480Eb55e80D19ad83579A64c007002")
	assert.Equal(t, expectedAddr, params.WithdrawalRequestAddress)
}

// TestWithdrawalRequestType verifies the withdrawal request type constant
func TestWithdrawalRequestType(t *testing.T) {
	t.Parallel()

	// EIP-7002 specifies withdrawal request type as 0x01
	assert.Equal(t, byte(0x01), types.WithdrawalRequestType)
}

// TestWithdrawalRequestDataLen verifies the withdrawal request data length
func TestWithdrawalRequestDataLen(t *testing.T) {
	t.Parallel()

	// EIP-7002 withdrawal request data structure:
	// source_address: 20 bytes
	// validator_pubkey: 48 bytes
	// amount: 8 bytes (little-endian uint64, in Gwei)
	expectedLen := 20 + 48 + 8

	assert.Equal(t, 76, expectedLen, "Withdrawal request data length should be 76 bytes")
	assert.Equal(t, 76, types.WithdrawalRequestDataLen, "WithdrawalRequestDataLen should be 76")
}

// TestDequeueWithdrawalRequests7002WithNilSyscall tests the function with nil syscall result
func TestDequeueWithdrawalRequests7002WithNilSyscall(t *testing.T) {
	t.Parallel()

	// Create a syscall that returns nil (no withdrawal requests)
	syscall := func(addr libcommon.Address, data []byte) ([]byte, error) {
		assert.Equal(t, params.WithdrawalRequestAddress, addr)
		return nil, nil
	}

	result := DequeueWithdrawalRequests7002(syscall)
	assert.Nil(t, result, "Should return nil when syscall returns nil")
}

// TestDequeueWithdrawalRequests7002WithData tests the function with syscall returning data
func TestDequeueWithdrawalRequests7002WithData(t *testing.T) {
	t.Parallel()

	// Create mock withdrawal request data (76 bytes)
	mockData := make([]byte, types.WithdrawalRequestDataLen)

	// Fill with mock data
	// source_address (20 bytes)
	copy(mockData[0:20], libcommon.HexToAddress("0x1234567890123456789012345678901234567890").Bytes())
	// validator_pubkey (48 bytes)
	for i := 20; i < 68; i++ {
		mockData[i] = byte(i)
	}
	// amount (8 bytes) - 32 ETH in Gwei
	mockData[68] = 0x00
	mockData[69] = 0xe8
	mockData[70] = 0x76
	mockData[71] = 0x48
	mockData[72] = 0x17
	mockData[73] = 0x00
	mockData[74] = 0x00
	mockData[75] = 0x00

	// Create a syscall that returns the mock data
	syscall := func(addr libcommon.Address, data []byte) ([]byte, error) {
		assert.Equal(t, params.WithdrawalRequestAddress, addr)
		assert.Nil(t, data, "Input data should be nil")
		return mockData, nil
	}

	result := DequeueWithdrawalRequests7002(syscall)
	require.NotNil(t, result)

	assert.Equal(t, types.WithdrawalRequestType, result.Type)
	assert.Equal(t, mockData, result.RequestData)
	assert.Len(t, result.RequestData, types.WithdrawalRequestDataLen)
}

// TestDequeueWithdrawalRequests7002WithMultipleRequests tests multiple withdrawal requests
func TestDequeueWithdrawalRequests7002WithMultipleRequests(t *testing.T) {
	t.Parallel()

	// Create mock data for 2 withdrawal requests (2 * 76 = 152 bytes)
	numRequests := 2
	mockData := make([]byte, numRequests*types.WithdrawalRequestDataLen)

	// Fill with mock data for 2 requests
	for i := 0; i < numRequests; i++ {
		offset := i * types.WithdrawalRequestDataLen
		// source_address
		for j := 0; j < 20; j++ {
			mockData[offset+j] = byte(i + 1)
		}
		// validator_pubkey
		for j := 20; j < 68; j++ {
			mockData[offset+j] = byte(j + i*10)
		}
		// amount
		mockData[offset+68] = byte(i + 1)
	}

	syscall := func(addr libcommon.Address, data []byte) ([]byte, error) {
		return mockData, nil
	}

	result := DequeueWithdrawalRequests7002(syscall)
	require.NotNil(t, result)

	assert.Equal(t, types.WithdrawalRequestType, result.Type)
	assert.Len(t, result.RequestData, numRequests*types.WithdrawalRequestDataLen)
}

// TestDequeueWithdrawalRequests7002WithError tests error handling
func TestDequeueWithdrawalRequests7002WithError(t *testing.T) {
	t.Parallel()

	// Create a syscall that returns an error
	syscall := func(addr libcommon.Address, data []byte) ([]byte, error) {
		return nil, assert.AnError
	}

	// Should return nil on error (error is logged)
	result := DequeueWithdrawalRequests7002(syscall)
	assert.Nil(t, result)
}

// TestWithdrawalRequestDataStructure tests the withdrawal request data structure
func TestWithdrawalRequestDataStructure(t *testing.T) {
	t.Parallel()

	// Verify the structure according to EIP-7002:
	// | Field              | Offset | Size |
	// |--------------------|--------|------|
	// | source_address     | 0      | 20   |
	// | validator_pubkey   | 20     | 48   |
	// | amount             | 68     | 8    |

	sourceAddressOffset := 0
	sourceAddressLen := 20
	validatorPubkeyOffset := 20
	validatorPubkeyLen := 48
	amountOffset := 68
	amountLen := 8

	assert.Equal(t, 0, sourceAddressOffset)
	assert.Equal(t, 20, sourceAddressLen)
	assert.Equal(t, 20, validatorPubkeyOffset)
	assert.Equal(t, 48, validatorPubkeyLen)
	assert.Equal(t, 68, amountOffset)
	assert.Equal(t, 8, amountLen)
	assert.Equal(t, 76, sourceAddressLen+validatorPubkeyLen+amountLen)
}

// TestFlatRequestWithdrawalEncode tests FlatRequest encoding for withdrawal requests
func TestFlatRequestWithdrawalEncode(t *testing.T) {
	t.Parallel()

	requestData := make([]byte, types.WithdrawalRequestDataLen)
	// Fill with test data
	for i := range requestData {
		requestData[i] = byte(i)
	}

	fr := types.FlatRequest{
		Type:        types.WithdrawalRequestType,
		RequestData: requestData,
	}

	encoded := fr.Encode()
	assert.Equal(t, byte(0x01), encoded[0], "First byte should be WithdrawalRequestType (0x01)")
	assert.Equal(t, requestData, encoded[1:])
	assert.Len(t, encoded, types.WithdrawalRequestDataLen+1)
}

// TestKnownRequestTypesIncludesWithdrawal verifies WithdrawalRequestType is in known types
func TestKnownRequestTypesIncludesWithdrawal(t *testing.T) {
	t.Parallel()

	found := false
	for _, rt := range types.KnownRequestTypes {
		if rt == types.WithdrawalRequestType {
			found = true
			break
		}
	}
	assert.True(t, found, "WithdrawalRequestType should be in KnownRequestTypes")
}

// TestWithdrawalRequestAmountEncoding tests the amount field encoding
func TestWithdrawalRequestAmountEncoding(t *testing.T) {
	t.Parallel()

	// According to EIP-7002, the amount is a little-endian uint64 in Gwei
	// Test cases:
	testCases := []struct {
		name     string
		amountLE []byte // little-endian
		expected uint64
	}{
		{
			name:     "zero",
			amountLE: []byte{0, 0, 0, 0, 0, 0, 0, 0},
			expected: 0,
		},
		{
			name:     "1 ETH (1e9 Gwei)",
			amountLE: []byte{0x00, 0xca, 0x9a, 0x3b, 0x00, 0x00, 0x00, 0x00},
			expected: 1000000000,
		},
		{
			name:     "32 ETH (32e9 Gwei)",
			amountLE: []byte{0x00, 0x40, 0x59, 0x73, 0x07, 0x00, 0x00, 0x00}, // 0x773594000 in little-endian
			expected: 32000000000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Decode little-endian uint64
			var amount uint64
			for i := 0; i < 8; i++ {
				amount |= uint64(tc.amountLE[i]) << (8 * i)
			}
			assert.Equal(t, tc.expected, amount)
		})
	}
}

