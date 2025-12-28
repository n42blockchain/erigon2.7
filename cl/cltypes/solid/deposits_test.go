package solid

import (
	"testing"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDepositRequestEncodeDecode tests SSZ encoding and decoding of DepositRequest
func TestDepositRequestEncodeDecode(t *testing.T) {
	t.Parallel()

	// Create a sample DepositRequest (EIP-6110)
	pubkey := libcommon.Bytes48{}
	for i := 0; i < 48; i++ {
		pubkey[i] = byte(i)
	}

	withdrawalCredentials := libcommon.Hash{}
	for i := 0; i < 32; i++ {
		withdrawalCredentials[i] = byte(i + 100)
	}

	signature := libcommon.Bytes96{}
	for i := 0; i < 96; i++ {
		signature[i] = byte(i + 50)
	}

	depositRequest := &DepositRequest{
		PubKey:                pubkey,
		WithdrawalCredentials: withdrawalCredentials,
		Amount:                32000000000, // 32 ETH in Gwei
		Signature:             signature,
		Index:                 12345,
	}

	// Verify encoding size
	assert.Equal(t, SizeDepositRequest, depositRequest.EncodingSizeSSZ())
	assert.Equal(t, 192, SizeDepositRequest, "SizeDepositRequest should be 192 bytes (48+32+8+96+8)")

	// Encode to SSZ
	encoded, err := depositRequest.EncodeSSZ(nil)
	require.NoError(t, err)
	assert.Len(t, encoded, SizeDepositRequest)

	// Decode from SSZ
	decoded := &DepositRequest{}
	err = decoded.DecodeSSZ(encoded, 0)
	require.NoError(t, err)

	// Verify decoded values
	assert.Equal(t, depositRequest.PubKey, decoded.PubKey)
	assert.Equal(t, depositRequest.WithdrawalCredentials, decoded.WithdrawalCredentials)
	assert.Equal(t, depositRequest.Amount, decoded.Amount)
	assert.Equal(t, depositRequest.Signature, decoded.Signature)
	assert.Equal(t, depositRequest.Index, decoded.Index)
}

// TestDepositRequestHashSSZ tests HashSSZ of DepositRequest
func TestDepositRequestHashSSZ(t *testing.T) {
	t.Parallel()

	depositRequest := &DepositRequest{
		PubKey:                libcommon.Bytes48{1, 2, 3},
		WithdrawalCredentials: libcommon.Hash{4, 5, 6},
		Amount:                32000000000,
		Signature:             libcommon.Bytes96{7, 8, 9},
		Index:                 100,
	}

	hash1, err := depositRequest.HashSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, hash1)

	// Same data should produce same hash
	hash2, err := depositRequest.HashSSZ()
	require.NoError(t, err)
	assert.Equal(t, hash1, hash2)

	// Different data should produce different hash
	depositRequest2 := &DepositRequest{
		PubKey:                libcommon.Bytes48{1, 2, 4}, // Changed
		WithdrawalCredentials: libcommon.Hash{4, 5, 6},
		Amount:                32000000000,
		Signature:             libcommon.Bytes96{7, 8, 9},
		Index:                 100,
	}
	hash3, err := depositRequest2.HashSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash3)
}

// TestDepositRequestClone tests Clone of DepositRequest
func TestDepositRequestClone(t *testing.T) {
	t.Parallel()

	depositRequest := &DepositRequest{
		PubKey:                libcommon.Bytes48{1, 2, 3},
		WithdrawalCredentials: libcommon.Hash{4, 5, 6},
		Amount:                32000000000,
		Signature:             libcommon.Bytes96{7, 8, 9},
		Index:                 100,
	}

	cloned := depositRequest.Clone().(*DepositRequest)
	assert.NotNil(t, cloned)
	// Clone returns empty struct, not a copy
	assert.Equal(t, &DepositRequest{}, cloned)
}

// TestDepositRequestStatic tests Static method
func TestDepositRequestStatic(t *testing.T) {
	t.Parallel()

	depositRequest := &DepositRequest{}
	assert.True(t, depositRequest.Static())
}

// TestPendingDepositEncodeDecode tests SSZ encoding and decoding of PendingDeposit
func TestPendingDepositEncodeDecode(t *testing.T) {
	t.Parallel()

	pubkey := libcommon.Bytes48{}
	for i := 0; i < 48; i++ {
		pubkey[i] = byte(i)
	}

	withdrawalCredentials := libcommon.Hash{}
	for i := 0; i < 32; i++ {
		withdrawalCredentials[i] = byte(i + 100)
	}

	signature := libcommon.Bytes96{}
	for i := 0; i < 96; i++ {
		signature[i] = byte(i + 50)
	}

	pendingDeposit := &PendingDeposit{
		PubKey:                pubkey,
		WithdrawalCredentials: withdrawalCredentials,
		Amount:                32000000000,
		Signature:             signature,
		Slot:                  1000,
	}

	// Verify encoding size
	assert.Equal(t, SizePendingDeposit, pendingDeposit.EncodingSizeSSZ())
	assert.Equal(t, 192, SizePendingDeposit, "SizePendingDeposit should be 192 bytes")

	// Encode to SSZ
	encoded, err := pendingDeposit.EncodeSSZ(nil)
	require.NoError(t, err)
	assert.Len(t, encoded, SizePendingDeposit)

	// Decode from SSZ
	decoded := &PendingDeposit{}
	err = decoded.DecodeSSZ(encoded, 0)
	require.NoError(t, err)

	// Verify decoded values
	assert.Equal(t, pendingDeposit.PubKey, decoded.PubKey)
	assert.Equal(t, pendingDeposit.WithdrawalCredentials, decoded.WithdrawalCredentials)
	assert.Equal(t, pendingDeposit.Amount, decoded.Amount)
	assert.Equal(t, pendingDeposit.Signature, decoded.Signature)
	assert.Equal(t, pendingDeposit.Slot, decoded.Slot)
}

// TestPendingDepositHashSSZ tests HashSSZ of PendingDeposit
func TestPendingDepositHashSSZ(t *testing.T) {
	t.Parallel()

	pendingDeposit := &PendingDeposit{
		PubKey:                libcommon.Bytes48{1, 2, 3},
		WithdrawalCredentials: libcommon.Hash{4, 5, 6},
		Amount:                32000000000,
		Signature:             libcommon.Bytes96{7, 8, 9},
		Slot:                  1000,
	}

	hash, err := pendingDeposit.HashSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, hash)
}

// TestDepositRequestDataLength verifies deposit request data matches EIP-6110 specification
func TestDepositRequestDataLength(t *testing.T) {
	t.Parallel()

	// EIP-6110 deposit request data structure:
	// pubkey: 48 bytes
	// withdrawal_credentials: 32 bytes
	// amount: 8 bytes (little-endian uint64)
	// signature: 96 bytes
	// index: 8 bytes (little-endian uint64)
	expectedLen := 48 + 32 + 8 + 96 + 8

	assert.Equal(t, expectedLen, SizeDepositRequest)
	assert.Equal(t, 192, SizeDepositRequest)

	// Verify PendingDeposit has same size (slot replaces index)
	assert.Equal(t, SizeDepositRequest, SizePendingDeposit)
}

// TestDepositRequestInterfaceCompliance verifies DepositRequest implements required interfaces
func TestDepositRequestInterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ EncodableHashableSSZ = (*DepositRequest)(nil)
	var _ EncodableHashableSSZ = (*PendingDeposit)(nil)
}

