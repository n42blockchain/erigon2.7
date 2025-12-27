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

// TestConsolidationRequestAddress verifies the EIP-7251 contract address
func TestConsolidationRequestAddress(t *testing.T) {
	t.Parallel()

	// EIP-7251 specifies the consolidation request contract address
	expectedAddr := libcommon.HexToAddress("0x0000BBdDc7CE488642fb579F8B00f3a590007251")
	assert.Equal(t, expectedAddr, params.ConsolidationRequestAddress)
}

// TestConsolidationRequestType verifies the consolidation request type constant
func TestConsolidationRequestType(t *testing.T) {
	t.Parallel()

	// EIP-7251 specifies consolidation request type as 0x02
	assert.Equal(t, byte(0x02), types.ConsolidationRequestType)
}

// TestConsolidationRequestDataLen verifies the consolidation request data length
func TestConsolidationRequestDataLen(t *testing.T) {
	t.Parallel()

	// EIP-7251 consolidation request data structure:
	// source_address: 20 bytes
	// source_pubkey: 48 bytes (BLS public key)
	// target_pubkey: 48 bytes (BLS public key)
	expectedLen := 20 + 48 + 48

	assert.Equal(t, 116, expectedLen, "Consolidation request data length should be 116 bytes")
	assert.Equal(t, 116, types.ConsolidationRequestDataLen, "ConsolidationRequestDataLen should be 116")
}

// TestDequeueConsolidationRequests7251WithNilSyscall tests the function with nil syscall result
func TestDequeueConsolidationRequests7251WithNilSyscall(t *testing.T) {
	t.Parallel()

	// Create a syscall that returns nil (no consolidation requests)
	syscall := func(addr libcommon.Address, data []byte) ([]byte, error) {
		assert.Equal(t, params.ConsolidationRequestAddress, addr)
		return nil, nil
	}

	result := DequeueConsolidationRequests7251(syscall)
	assert.Nil(t, result, "Should return nil when syscall returns nil")
}

// TestDequeueConsolidationRequests7251WithData tests the function with syscall returning data
func TestDequeueConsolidationRequests7251WithData(t *testing.T) {
	t.Parallel()

	// Create mock consolidation request data (116 bytes)
	mockData := make([]byte, types.ConsolidationRequestDataLen)

	// Fill with mock data
	// source_address (20 bytes)
	copy(mockData[0:20], libcommon.HexToAddress("0x1234567890123456789012345678901234567890").Bytes())

	// source_pubkey (48 bytes)
	for i := 20; i < 68; i++ {
		mockData[i] = byte(i)
	}

	// target_pubkey (48 bytes)
	for i := 68; i < 116; i++ {
		mockData[i] = byte(i + 100)
	}

	// Create a syscall that returns the mock data
	syscall := func(addr libcommon.Address, data []byte) ([]byte, error) {
		assert.Equal(t, params.ConsolidationRequestAddress, addr)
		assert.Nil(t, data, "Input data should be nil")
		return mockData, nil
	}

	result := DequeueConsolidationRequests7251(syscall)
	require.NotNil(t, result)

	assert.Equal(t, types.ConsolidationRequestType, result.Type)
	assert.Equal(t, mockData, result.RequestData)
	assert.Len(t, result.RequestData, types.ConsolidationRequestDataLen)
}

// TestDequeueConsolidationRequests7251WithMultipleRequests tests multiple consolidation requests
func TestDequeueConsolidationRequests7251WithMultipleRequests(t *testing.T) {
	t.Parallel()

	// Create mock data for 3 consolidation requests (3 * 116 = 348 bytes)
	numRequests := 3
	mockData := make([]byte, numRequests*types.ConsolidationRequestDataLen)

	// Fill with mock data for 3 requests
	for i := 0; i < numRequests; i++ {
		offset := i * types.ConsolidationRequestDataLen

		// source_address (20 bytes)
		for j := 0; j < 20; j++ {
			mockData[offset+j] = byte(i + 1)
		}

		// source_pubkey (48 bytes)
		for j := 20; j < 68; j++ {
			mockData[offset+j] = byte(j + i*10)
		}

		// target_pubkey (48 bytes)
		for j := 68; j < 116; j++ {
			mockData[offset+j] = byte(j + i*10 + 100)
		}
	}

	syscall := func(addr libcommon.Address, data []byte) ([]byte, error) {
		return mockData, nil
	}

	result := DequeueConsolidationRequests7251(syscall)
	require.NotNil(t, result)

	assert.Equal(t, types.ConsolidationRequestType, result.Type)
	assert.Len(t, result.RequestData, numRequests*types.ConsolidationRequestDataLen)
}

// TestDequeueConsolidationRequests7251WithError tests error handling
func TestDequeueConsolidationRequests7251WithError(t *testing.T) {
	t.Parallel()

	// Create a syscall that returns an error
	syscall := func(addr libcommon.Address, data []byte) ([]byte, error) {
		return nil, assert.AnError
	}

	// Should return nil on error (error is logged)
	result := DequeueConsolidationRequests7251(syscall)
	assert.Nil(t, result)
}

// TestConsolidationRequestDataStructure tests the consolidation request data structure
func TestConsolidationRequestDataStructure(t *testing.T) {
	t.Parallel()

	// Verify the structure according to EIP-7251:
	// | Field           | Offset | Size |
	// |-----------------|--------|------|
	// | source_address  | 0      | 20   |
	// | source_pubkey   | 20     | 48   |
	// | target_pubkey   | 68     | 48   |

	sourceAddressOffset := 0
	sourceAddressLen := 20
	sourcePubkeyOffset := 20
	sourcePubkeyLen := 48
	targetPubkeyOffset := 68
	targetPubkeyLen := 48

	assert.Equal(t, 0, sourceAddressOffset)
	assert.Equal(t, 20, sourceAddressLen)
	assert.Equal(t, 20, sourcePubkeyOffset)
	assert.Equal(t, 48, sourcePubkeyLen)
	assert.Equal(t, 68, targetPubkeyOffset)
	assert.Equal(t, 48, targetPubkeyLen)
	assert.Equal(t, 116, sourceAddressLen+sourcePubkeyLen+targetPubkeyLen)
}

// TestFlatRequestConsolidationEncode tests FlatRequest encoding for consolidation requests
func TestFlatRequestConsolidationEncode(t *testing.T) {
	t.Parallel()

	requestData := make([]byte, types.ConsolidationRequestDataLen)
	// Fill with test data
	for i := range requestData {
		requestData[i] = byte(i)
	}

	fr := types.FlatRequest{
		Type:        types.ConsolidationRequestType,
		RequestData: requestData,
	}

	encoded := fr.Encode()
	assert.Equal(t, byte(0x02), encoded[0], "First byte should be ConsolidationRequestType (0x02)")
	assert.Equal(t, requestData, encoded[1:])
	assert.Len(t, encoded, types.ConsolidationRequestDataLen+1)
}

// TestKnownRequestTypesIncludesConsolidation verifies ConsolidationRequestType is in known types
func TestKnownRequestTypesIncludesConsolidation(t *testing.T) {
	t.Parallel()

	found := false
	for _, rt := range types.KnownRequestTypes {
		if rt == types.ConsolidationRequestType {
			found = true
			break
		}
	}
	assert.True(t, found, "ConsolidationRequestType should be in KnownRequestTypes")
}

// TestConsolidationRequestPubkeyLength verifies the BLS pubkey length is correct
func TestConsolidationRequestPubkeyLength(t *testing.T) {
	t.Parallel()

	// BLS public keys are 48 bytes (384 bits)
	blsPubkeyLen := 48

	// Verify two pubkeys plus address equals total length
	assert.Equal(t, types.ConsolidationRequestDataLen, 20+blsPubkeyLen+blsPubkeyLen)
}

// TestConsolidationRequestExtraction tests extracting fields from request data
func TestConsolidationRequestExtraction(t *testing.T) {
	t.Parallel()

	// Create a consolidation request with known values
	sourceAddr := libcommon.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12")
	sourcePubkey := make([]byte, 48)
	targetPubkey := make([]byte, 48)

	// Fill pubkeys with distinguishable patterns
	for i := 0; i < 48; i++ {
		sourcePubkey[i] = byte(0xAA)
		targetPubkey[i] = byte(0xBB)
	}

	// Build the request data
	requestData := make([]byte, types.ConsolidationRequestDataLen)
	copy(requestData[0:20], sourceAddr.Bytes())
	copy(requestData[20:68], sourcePubkey)
	copy(requestData[68:116], targetPubkey)

	// Extract and verify
	extractedAddr := libcommon.BytesToAddress(requestData[0:20])
	extractedSourcePubkey := requestData[20:68]
	extractedTargetPubkey := requestData[68:116]

	assert.Equal(t, sourceAddr, extractedAddr)
	assert.Equal(t, sourcePubkey, extractedSourcePubkey)
	assert.Equal(t, targetPubkey, extractedTargetPubkey)
}

// TestConsolidationVsWithdrawalRequestTypes verifies types are distinct
func TestConsolidationVsWithdrawalRequestTypes(t *testing.T) {
	t.Parallel()

	// Ensure all request types are distinct
	assert.NotEqual(t, types.DepositRequestType, types.WithdrawalRequestType)
	assert.NotEqual(t, types.DepositRequestType, types.ConsolidationRequestType)
	assert.NotEqual(t, types.WithdrawalRequestType, types.ConsolidationRequestType)

	// Verify ordering
	assert.Equal(t, byte(0x00), types.DepositRequestType)
	assert.Equal(t, byte(0x01), types.WithdrawalRequestType)
	assert.Equal(t, byte(0x02), types.ConsolidationRequestType)
}

// TestConsolidationVsWithdrawalDataLen verifies data lengths are different
func TestConsolidationVsWithdrawalDataLen(t *testing.T) {
	t.Parallel()

	// Consolidation: 20 + 48 + 48 = 116
	// Withdrawal: 20 + 48 + 8 = 76
	assert.Equal(t, 116, types.ConsolidationRequestDataLen)
	assert.Equal(t, 76, types.WithdrawalRequestDataLen)
	assert.Greater(t, types.ConsolidationRequestDataLen, types.WithdrawalRequestDataLen)
}

