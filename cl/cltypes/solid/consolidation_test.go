package solid

import (
	"testing"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConsolidationRequestEncodeDecode tests SSZ encoding and decoding of ConsolidationRequest
func TestConsolidationRequestEncodeDecode(t *testing.T) {
	t.Parallel()

	// Create a sample ConsolidationRequest (EIP-7251)
	sourceAddress := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")

	sourcePubKey := libcommon.Bytes48{}
	for i := 0; i < 48; i++ {
		sourcePubKey[i] = byte(i)
	}

	targetPubKey := libcommon.Bytes48{}
	for i := 0; i < 48; i++ {
		targetPubKey[i] = byte(i + 100)
	}

	consolidationRequest := &ConsolidationRequest{
		SourceAddress: sourceAddress,
		SourcePubKey:  sourcePubKey,
		TargetPubKey:  targetPubKey,
	}

	// Verify encoding size
	assert.Equal(t, SizeConsolidationRequest, consolidationRequest.EncodingSizeSSZ())
	assert.Equal(t, 116, SizeConsolidationRequest, "SizeConsolidationRequest should be 116 bytes (20+48+48)")

	// Encode to SSZ
	encoded, err := consolidationRequest.EncodeSSZ(nil)
	require.NoError(t, err)
	assert.Len(t, encoded, SizeConsolidationRequest)

	// Decode from SSZ
	decoded := &ConsolidationRequest{}
	err = decoded.DecodeSSZ(encoded, 0)
	require.NoError(t, err)

	// Verify decoded values
	assert.Equal(t, consolidationRequest.SourceAddress, decoded.SourceAddress)
	assert.Equal(t, consolidationRequest.SourcePubKey, decoded.SourcePubKey)
	assert.Equal(t, consolidationRequest.TargetPubKey, decoded.TargetPubKey)
}

// TestConsolidationRequestHashSSZ tests HashSSZ of ConsolidationRequest
func TestConsolidationRequestHashSSZ(t *testing.T) {
	t.Parallel()

	consolidationRequest := &ConsolidationRequest{
		SourceAddress: libcommon.HexToAddress("0x1234567890123456789012345678901234567890"),
		SourcePubKey:  libcommon.Bytes48{1, 2, 3},
		TargetPubKey:  libcommon.Bytes48{4, 5, 6},
	}

	hash1, err := consolidationRequest.HashSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, hash1)

	// Same data should produce same hash
	hash2, err := consolidationRequest.HashSSZ()
	require.NoError(t, err)
	assert.Equal(t, hash1, hash2)

	// Different data should produce different hash
	consolidationRequest2 := &ConsolidationRequest{
		SourceAddress: libcommon.HexToAddress("0x0987654321098765432109876543210987654321"), // Changed
		SourcePubKey:  libcommon.Bytes48{1, 2, 3},
		TargetPubKey:  libcommon.Bytes48{4, 5, 6},
	}
	hash3, err := consolidationRequest2.HashSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash3)
}

// TestConsolidationRequestClone tests Clone of ConsolidationRequest
func TestConsolidationRequestClone(t *testing.T) {
	t.Parallel()

	consolidationRequest := &ConsolidationRequest{
		SourceAddress: libcommon.HexToAddress("0x1234567890123456789012345678901234567890"),
		SourcePubKey:  libcommon.Bytes48{1, 2, 3},
		TargetPubKey:  libcommon.Bytes48{4, 5, 6},
	}

	cloned := consolidationRequest.Clone().(*ConsolidationRequest)
	assert.NotNil(t, cloned)
	// Clone returns empty struct, not a copy
	assert.Equal(t, &ConsolidationRequest{}, cloned)
}

// TestConsolidationRequestStatic tests Static method
func TestConsolidationRequestStatic(t *testing.T) {
	t.Parallel()

	consolidationRequest := &ConsolidationRequest{}
	assert.True(t, consolidationRequest.Static())
}

// TestPendingConsolidationEncodeDecode tests SSZ encoding and decoding of PendingConsolidation
func TestPendingConsolidationEncodeDecode(t *testing.T) {
	t.Parallel()

	pendingConsolidation := &PendingConsolidation{
		SourceIndex: 12345,
		TargetIndex: 67890,
	}

	// Verify encoding size
	assert.Equal(t, 16, pendingConsolidation.EncodingSizeSSZ())
	assert.Equal(t, 16, SizePendingConsolidation, "SizePendingConsolidation should be 16 bytes (8+8)")

	// Encode to SSZ
	encoded, err := pendingConsolidation.EncodeSSZ(nil)
	require.NoError(t, err)
	assert.Len(t, encoded, SizePendingConsolidation)

	// Decode from SSZ
	decoded := &PendingConsolidation{}
	err = decoded.DecodeSSZ(encoded, 0)
	require.NoError(t, err)

	// Verify decoded values
	assert.Equal(t, pendingConsolidation.SourceIndex, decoded.SourceIndex)
	assert.Equal(t, pendingConsolidation.TargetIndex, decoded.TargetIndex)
}

// TestPendingConsolidationHashSSZ tests HashSSZ of PendingConsolidation
func TestPendingConsolidationHashSSZ(t *testing.T) {
	t.Parallel()

	pendingConsolidation := &PendingConsolidation{
		SourceIndex: 12345,
		TargetIndex: 67890,
	}

	hash, err := pendingConsolidation.HashSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, hash)
}

// TestConsolidationRequestDataLength verifies consolidation request data matches EIP-7251 specification
func TestConsolidationRequestDataLength(t *testing.T) {
	t.Parallel()

	// EIP-7251 consolidation request data structure:
	// source_address: 20 bytes
	// source_pubkey: 48 bytes (BLS public key)
	// target_pubkey: 48 bytes (BLS public key)
	expectedLen := 20 + 48 + 48

	assert.Equal(t, expectedLen, SizeConsolidationRequest)
	assert.Equal(t, 116, SizeConsolidationRequest)
}

// TestConsolidationRequestInterfaceCompliance verifies ConsolidationRequest implements required interfaces
func TestConsolidationRequestInterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ EncodableHashableSSZ = (*ConsolidationRequest)(nil)
	var _ EncodableHashableSSZ = (*PendingConsolidation)(nil)
}

// TestConsolidationRequestSwitchToCompounding tests consolidation request for switching to compounding
func TestConsolidationRequestSwitchToCompounding(t *testing.T) {
	t.Parallel()

	// EIP-7251: When source_pubkey == target_pubkey, it's a switch-to-compounding request
	pubKey := libcommon.Bytes48{}
	for i := 0; i < 48; i++ {
		pubKey[i] = byte(i)
	}

	switchToCompoundingRequest := &ConsolidationRequest{
		SourceAddress: libcommon.HexToAddress("0x1234567890123456789012345678901234567890"),
		SourcePubKey:  pubKey,
		TargetPubKey:  pubKey, // Same as source - switch to compounding
	}

	// Encode and decode
	encoded, err := switchToCompoundingRequest.EncodeSSZ(nil)
	require.NoError(t, err)

	decoded := &ConsolidationRequest{}
	err = decoded.DecodeSSZ(encoded, 0)
	require.NoError(t, err)

	// Verify source and target are the same
	assert.Equal(t, decoded.SourcePubKey, decoded.TargetPubKey, "Switch to compounding should have same source and target pubkey")
}

// TestConsolidationRequestDifferentPubkeys tests consolidation request with different pubkeys
func TestConsolidationRequestDifferentPubkeys(t *testing.T) {
	t.Parallel()

	sourcePubKey := libcommon.Bytes48{}
	targetPubKey := libcommon.Bytes48{}
	for i := 0; i < 48; i++ {
		sourcePubKey[i] = byte(i)
		targetPubKey[i] = byte(i + 1)
	}

	consolidationRequest := &ConsolidationRequest{
		SourceAddress: libcommon.HexToAddress("0x1234567890123456789012345678901234567890"),
		SourcePubKey:  sourcePubKey,
		TargetPubKey:  targetPubKey, // Different from source - actual consolidation
	}

	encoded, err := consolidationRequest.EncodeSSZ(nil)
	require.NoError(t, err)

	decoded := &ConsolidationRequest{}
	err = decoded.DecodeSSZ(encoded, 0)
	require.NoError(t, err)

	// Verify source and target are different
	assert.NotEqual(t, decoded.SourcePubKey, decoded.TargetPubKey, "Consolidation should have different source and target pubkey")
}

// TestMaxEffectiveBalanceConstants tests EIP-7251 related constants
func TestMaxEffectiveBalanceConstants(t *testing.T) {
	t.Parallel()

	// EIP-7251 increases max effective balance from 32 ETH to 2048 ETH
	// These are defined in clparams but we test the structure sizes here
	assert.Equal(t, 116, SizeConsolidationRequest, "ConsolidationRequest size: 20 + 48 + 48")
	assert.Equal(t, 16, SizePendingConsolidation, "PendingConsolidation size: 8 + 8")
}

