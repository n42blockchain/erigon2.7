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

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// EIP-7685: General purpose execution layer requests

func TestEmptyRequestsHashCalculation(t *testing.T) {
	reqs := make(FlatRequests, 0)
	h := reqs.Hash()
	testH := EmptyRequestsHash
	if *h != testH {
		t.Errorf("Requests Hash calculation error for empty hash, expected: %v, got: %v", testH, h)
	}
}

func TestRequestTypeConstants(t *testing.T) {
	// EIP-7685 defines request types
	assert.Equal(t, byte(0x00), DepositRequestType)
	assert.Equal(t, byte(0x01), WithdrawalRequestType)
	assert.Equal(t, byte(0x02), ConsolidationRequestType)

	// Verify KnownRequestTypes contains all types
	assert.Len(t, KnownRequestTypes, 3)
	assert.Contains(t, KnownRequestTypes, DepositRequestType)
	assert.Contains(t, KnownRequestTypes, WithdrawalRequestType)
	assert.Contains(t, KnownRequestTypes, ConsolidationRequestType)
}

func TestRequestDataLengths(t *testing.T) {
	// EIP-7685/EIP-6110: Deposit request data length
	// BLSPubKeyLen(48) + WithdrawalCredentialsLen(32) + Amount(8) + BLSSigLen(96) + Index(8) = 192
	assert.Equal(t, 192, DepositRequestDataLen)

	// EIP-7685/EIP-7002: Withdrawal request data length
	// Address(20) + Pubkey(48) + Amount(8) = 76
	assert.Equal(t, 76, WithdrawalRequestDataLen)

	// EIP-7685/EIP-7251: Consolidation request data length
	// Address(20) + SourcePubkey(48) + TargetPubkey(48) = 116
	assert.Equal(t, 116, ConsolidationRequestDataLen)
}

func TestFlatRequestEncode(t *testing.T) {
	// Test encoding a deposit request
	req := FlatRequest{
		Type:        DepositRequestType,
		RequestData: []byte{0x01, 0x02, 0x03, 0x04},
	}

	encoded := req.Encode()
	require.Len(t, encoded, 5) // type byte + 4 data bytes
	assert.Equal(t, DepositRequestType, encoded[0])
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, encoded[1:])
}

func TestFlatRequestRequestType(t *testing.T) {
	req := FlatRequest{
		Type:        WithdrawalRequestType,
		RequestData: []byte{0x01},
	}

	assert.Equal(t, WithdrawalRequestType, req.RequestType())
}

func TestFlatRequestCopy(t *testing.T) {
	req := FlatRequest{
		Type:        ConsolidationRequestType,
		RequestData: []byte{0x01, 0x02, 0x03},
	}

	copied := req.copy()
	require.NotNil(t, copied)

	// Verify copy is equal
	assert.Equal(t, req.Type, copied.Type)
	assert.Equal(t, req.RequestData, copied.RequestData)

	// Verify deep copy (modifying original doesn't affect copy)
	req.RequestData[0] = 0xFF
	assert.NotEqual(t, req.RequestData[0], copied.RequestData[0])
}

func TestFlatRequestsHash(t *testing.T) {
	// Test nil requests
	var nilReqs FlatRequests
	assert.Nil(t, nilReqs.Hash())

	// Test empty requests
	emptyReqs := make(FlatRequests, 0)
	h := emptyReqs.Hash()
	require.NotNil(t, h)
	assert.Equal(t, EmptyRequestsHash, *h)

	// Test with requests
	reqs := FlatRequests{
		{Type: DepositRequestType, RequestData: []byte{0x01, 0x02}},
		{Type: WithdrawalRequestType, RequestData: []byte{0x03, 0x04}},
	}
	h1 := reqs.Hash()
	require.NotNil(t, h1)
	assert.NotEqual(t, EmptyRequestsHash, *h1)

	// Same requests should produce same hash
	reqs2 := FlatRequests{
		{Type: DepositRequestType, RequestData: []byte{0x01, 0x02}},
		{Type: WithdrawalRequestType, RequestData: []byte{0x03, 0x04}},
	}
	h2 := reqs2.Hash()
	assert.Equal(t, *h1, *h2)

	// Different order should produce different hash
	reqs3 := FlatRequests{
		{Type: WithdrawalRequestType, RequestData: []byte{0x03, 0x04}},
		{Type: DepositRequestType, RequestData: []byte{0x01, 0x02}},
	}
	h3 := reqs3.Hash()
	assert.NotEqual(t, *h1, *h3)
}

func TestFlatRequestsLen(t *testing.T) {
	var nilReqs FlatRequests
	assert.Equal(t, 0, nilReqs.Len())

	emptyReqs := make(FlatRequests, 0)
	assert.Equal(t, 0, emptyReqs.Len())

	reqs := FlatRequests{
		{Type: DepositRequestType, RequestData: []byte{0x01}},
		{Type: WithdrawalRequestType, RequestData: []byte{0x02}},
		{Type: ConsolidationRequestType, RequestData: []byte{0x03}},
	}
	assert.Equal(t, 3, reqs.Len())
}

func TestEmptyRequestsHashValue(t *testing.T) {
	// EIP-7685: Empty requests hash should be SHA256 of empty input
	// sha256.Sum256([]byte("")) = e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	expectedHash := "0xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	assert.Equal(t, expectedHash, EmptyRequestsHash.Hex())
}

func TestFlatRequestsHashDeterministic(t *testing.T) {
	// Test that hash is deterministic for the same requests
	for i := 0; i < 10; i++ {
		reqs := FlatRequests{
			{Type: DepositRequestType, RequestData: []byte{0x01, 0x02, 0x03}},
		}
		h := reqs.Hash()
		require.NotNil(t, h)

		// All iterations should produce the same hash
		expectedHash := reqs.Hash()
		assert.Equal(t, *expectedHash, *h)
	}
}
