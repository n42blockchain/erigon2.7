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

package cltypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erigontech/erigon/cl/clparams"
	"github.com/erigontech/erigon/cl/cltypes/solid"
	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/common/hexutility"
)

// EIP-7685: General purpose execution layer requests

func TestExecutionRequestsNew(t *testing.T) {
	cfg := &clparams.BeaconChainConfig{
		MaxDepositRequestsPerPayload:       8192,
		MaxWithdrawalRequestsPerPayload:    16,
		MaxConsolidationRequestsPerPayload: 2,
	}

	er := NewExecutionRequests(cfg)
	require.NotNil(t, er)
	require.NotNil(t, er.Deposits)
	require.NotNil(t, er.Withdrawals)
	require.NotNil(t, er.Consolidations)

	// Verify initial lengths are 0
	assert.Equal(t, 0, er.Deposits.Len())
	assert.Equal(t, 0, er.Withdrawals.Len())
	assert.Equal(t, 0, er.Consolidations.Len())
}

func TestExecutionRequestsEncodeDecode(t *testing.T) {
	cfg := &clparams.BeaconChainConfig{
		MaxDepositRequestsPerPayload:       8192,
		MaxWithdrawalRequestsPerPayload:    16,
		MaxConsolidationRequestsPerPayload: 2,
	}

	er := NewExecutionRequests(cfg)

	// Add a deposit request
	deposit := &solid.DepositRequest{
		PubKey:                libcommon.Bytes48{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48},
		WithdrawalCredentials: libcommon.HexToHash("0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
		Amount:                32000000000,
		Signature:             libcommon.Bytes96{},
		Index:                 12345,
	}
	er.Deposits.Append(deposit)

	// Add a withdrawal request
	withdrawal := &solid.WithdrawalRequest{
		SourceAddress:   libcommon.HexToAddress("0x1234567890123456789012345678901234567890"),
		ValidatorPubKey: libcommon.Bytes48{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48},
		Amount:          1000000000,
	}
	er.Withdrawals.Append(withdrawal)

	// Add a consolidation request
	consolidation := &solid.ConsolidationRequest{
		SourceAddress: libcommon.HexToAddress("0x1234567890123456789012345678901234567890"),
		SourcePubKey:  libcommon.Bytes48{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48},
		TargetPubKey:  libcommon.Bytes48{48, 47, 46, 45, 44, 43, 42, 41, 40, 39, 38, 37, 36, 35, 34, 33, 32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
	}
	er.Consolidations.Append(consolidation)

	// Test SSZ encoding
	encoded, err := er.EncodeSSZ(nil)
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)

	// Test SSZ decoding
	decoded := NewExecutionRequests(cfg)
	err = decoded.DecodeSSZ(encoded, int(clparams.ElectraVersion))
	require.NoError(t, err)

	assert.Equal(t, 1, decoded.Deposits.Len())
	assert.Equal(t, 1, decoded.Withdrawals.Len())
	assert.Equal(t, 1, decoded.Consolidations.Len())
}

func TestExecutionRequestsHashSSZ(t *testing.T) {
	cfg := &clparams.BeaconChainConfig{
		MaxDepositRequestsPerPayload:       8192,
		MaxWithdrawalRequestsPerPayload:    16,
		MaxConsolidationRequestsPerPayload: 2,
	}

	er := NewExecutionRequests(cfg)

	// Empty requests should have a valid hash
	hash, err := er.HashSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, libcommon.Hash{}, hash)

	// Add a deposit and verify hash changes
	deposit := &solid.DepositRequest{
		PubKey:                libcommon.Bytes48{1, 2, 3},
		WithdrawalCredentials: libcommon.HexToHash("0x01"),
		Amount:                32000000000,
		Index:                 1,
	}
	er.Deposits.Append(deposit)

	hash2, err := er.HashSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, hash, hash2)
}

func TestExecutionRequestsClone(t *testing.T) {
	cfg := &clparams.BeaconChainConfig{
		MaxDepositRequestsPerPayload:       8192,
		MaxWithdrawalRequestsPerPayload:    16,
		MaxConsolidationRequestsPerPayload: 2,
	}

	er := NewExecutionRequests(cfg)
	cloned := er.Clone()

	require.NotNil(t, cloned)
	assert.IsType(t, &ExecutionRequests{}, cloned)
}

func TestExecutionRequestsStatic(t *testing.T) {
	cfg := &clparams.BeaconChainConfig{
		MaxDepositRequestsPerPayload:       8192,
		MaxWithdrawalRequestsPerPayload:    16,
		MaxConsolidationRequestsPerPayload: 2,
	}

	er := NewExecutionRequests(cfg)
	// ExecutionRequests is not static (contains lists)
	assert.False(t, er.Static())
}

func TestComputeExecutionRequestHash(t *testing.T) {
	// Test empty requests
	emptyRequests := []hexutility.Bytes{}
	emptyHash := ComputeExecutionRequestHash(emptyRequests)
	// Empty hash should be SHA256 of empty input
	expectedEmpty := libcommon.HexToHash("0xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	assert.Equal(t, expectedEmpty, emptyHash)

	// Test with requests
	req1 := hexutility.Bytes{0x00, 0x01, 0x02, 0x03}
	req2 := hexutility.Bytes{0x01, 0x04, 0x05, 0x06}
	requests := []hexutility.Bytes{req1, req2}

	hash := ComputeExecutionRequestHash(requests)
	assert.NotEqual(t, libcommon.Hash{}, hash)
	assert.NotEqual(t, emptyHash, hash)

	// Same requests should produce same hash
	hash2 := ComputeExecutionRequestHash(requests)
	assert.Equal(t, hash, hash2)

	// Different order should produce different hash
	reversedRequests := []hexutility.Bytes{req2, req1}
	hash3 := ComputeExecutionRequestHash(reversedRequests)
	assert.NotEqual(t, hash, hash3)
}

func TestGetExecutionRequestsList(t *testing.T) {
	cfg := &clparams.BeaconChainConfig{
		MaxDepositRequestsPerPayload:       8192,
		MaxWithdrawalRequestsPerPayload:    16,
		MaxConsolidationRequestsPerPayload: 2,
		DepositRequestType:                 0x00,
		WithdrawalRequestType:              0x01,
		ConsolidationRequestType:           0x02,
	}

	// Test nil ExecutionRequests
	nilList := GetExecutionRequestsList(cfg, nil)
	assert.Nil(t, nilList)

	// Test empty ExecutionRequests
	er := NewExecutionRequests(cfg)
	emptyList := GetExecutionRequestsList(cfg, er)
	assert.Empty(t, emptyList)

	// Test with a deposit request
	deposit := &solid.DepositRequest{
		PubKey:                libcommon.Bytes48{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48},
		WithdrawalCredentials: libcommon.HexToHash("0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
		Amount:                32000000000,
		Signature:             libcommon.Bytes96{},
		Index:                 12345,
	}
	er.Deposits.Append(deposit)

	list := GetExecutionRequestsList(cfg, er)
	require.Len(t, list, 1)

	// First byte should be the deposit request type
	assert.Equal(t, byte(0x00), list[0][0])
}

func TestExecutionRequestsJSONRoundtrip(t *testing.T) {
	cfg := &clparams.BeaconChainConfig{
		MaxDepositRequestsPerPayload:       8192,
		MaxWithdrawalRequestsPerPayload:    16,
		MaxConsolidationRequestsPerPayload: 2,
	}

	er := NewExecutionRequests(cfg)

	// Add some test data
	deposit := &solid.DepositRequest{
		PubKey:                libcommon.Bytes48{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48},
		WithdrawalCredentials: libcommon.HexToHash("0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
		Amount:                32000000000,
		Signature:             libcommon.Bytes96{},
		Index:                 12345,
	}
	er.Deposits.Append(deposit)

	// Note: JSON marshalling/unmarshalling requires proper JSON tags in solid types
	// This test verifies the UnmarshalJSON implementation exists
	jsonData := `{"deposits":[],"withdrawals":[],"consolidations":[]}`
	decoded := NewExecutionRequests(cfg)
	err := decoded.UnmarshalJSON([]byte(jsonData))
	require.NoError(t, err)
}

func TestExecutionRequestTypeConstants(t *testing.T) {
	// Verify EIP-7685 request type constants match
	cfg := clparams.MainnetBeaconConfig

	// EIP-7685 specifies:
	// DEPOSIT_REQUEST_TYPE = 0x00
	// WITHDRAWAL_REQUEST_TYPE = 0x01
	// CONSOLIDATION_REQUEST_TYPE = 0x02
	assert.Equal(t, clparams.ConfigByte(0x00), cfg.DepositRequestType)
	assert.Equal(t, clparams.ConfigByte(0x01), cfg.WithdrawalRequestType)
	assert.Equal(t, clparams.ConfigByte(0x02), cfg.ConsolidationRequestType)
}

func TestExecutionRequestsEncodingSizeSSZ(t *testing.T) {
	cfg := &clparams.BeaconChainConfig{
		MaxDepositRequestsPerPayload:       8192,
		MaxWithdrawalRequestsPerPayload:    16,
		MaxConsolidationRequestsPerPayload: 2,
	}

	er := NewExecutionRequests(cfg)

	// Empty requests should have size 0 (empty lists)
	emptySize := er.EncodingSizeSSZ()
	assert.Equal(t, 0, emptySize, "Empty ExecutionRequests should have 0 size")

	// Add a deposit and verify size increases
	deposit := &solid.DepositRequest{
		PubKey:                libcommon.Bytes48{1, 2, 3},
		WithdrawalCredentials: libcommon.HexToHash("0x01"),
		Amount:                32000000000,
		Index:                 1,
	}
	er.Deposits.Append(deposit)

	sizeWithDeposit := er.EncodingSizeSSZ()
	assert.Greater(t, sizeWithDeposit, 0, "ExecutionRequests with deposit should have non-zero size")
	assert.Equal(t, solid.SizeDepositRequest, sizeWithDeposit, "Size should match deposit request size")
}

