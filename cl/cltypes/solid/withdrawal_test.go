package solid

import (
	"testing"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWithdrawalRequestEncodeDecode tests SSZ encoding and decoding of WithdrawalRequest
func TestWithdrawalRequestEncodeDecode(t *testing.T) {
	t.Parallel()

	// Create a sample WithdrawalRequest (EIP-7002)
	sourceAddress := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")

	validatorPubKey := libcommon.Bytes48{}
	for i := 0; i < 48; i++ {
		validatorPubKey[i] = byte(i)
	}

	withdrawalRequest := &WithdrawalRequest{
		SourceAddress:   sourceAddress,
		ValidatorPubKey: validatorPubKey,
		Amount:          32000000000, // 32 ETH in Gwei
	}

	// Verify encoding size
	assert.Equal(t, SizeWithdrawalRequest, withdrawalRequest.EncodingSizeSSZ())
	assert.Equal(t, 76, SizeWithdrawalRequest, "SizeWithdrawalRequest should be 76 bytes (20+48+8)")

	// Encode to SSZ
	encoded, err := withdrawalRequest.EncodeSSZ(nil)
	require.NoError(t, err)
	assert.Len(t, encoded, SizeWithdrawalRequest)

	// Decode from SSZ
	decoded := &WithdrawalRequest{}
	err = decoded.DecodeSSZ(encoded, 0)
	require.NoError(t, err)

	// Verify decoded values
	assert.Equal(t, withdrawalRequest.SourceAddress, decoded.SourceAddress)
	assert.Equal(t, withdrawalRequest.ValidatorPubKey, decoded.ValidatorPubKey)
	assert.Equal(t, withdrawalRequest.Amount, decoded.Amount)
}

// TestWithdrawalRequestHashSSZ tests HashSSZ of WithdrawalRequest
func TestWithdrawalRequestHashSSZ(t *testing.T) {
	t.Parallel()

	withdrawalRequest := &WithdrawalRequest{
		SourceAddress:   libcommon.HexToAddress("0x1234567890123456789012345678901234567890"),
		ValidatorPubKey: libcommon.Bytes48{1, 2, 3},
		Amount:          32000000000,
	}

	hash1, err := withdrawalRequest.HashSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, hash1)

	// Same data should produce same hash
	hash2, err := withdrawalRequest.HashSSZ()
	require.NoError(t, err)
	assert.Equal(t, hash1, hash2)

	// Different data should produce different hash
	withdrawalRequest2 := &WithdrawalRequest{
		SourceAddress:   libcommon.HexToAddress("0x0987654321098765432109876543210987654321"), // Changed
		ValidatorPubKey: libcommon.Bytes48{1, 2, 3},
		Amount:          32000000000,
	}
	hash3, err := withdrawalRequest2.HashSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash3)
}

// TestWithdrawalRequestClone tests Clone of WithdrawalRequest
func TestWithdrawalRequestClone(t *testing.T) {
	t.Parallel()

	withdrawalRequest := &WithdrawalRequest{
		SourceAddress:   libcommon.HexToAddress("0x1234567890123456789012345678901234567890"),
		ValidatorPubKey: libcommon.Bytes48{1, 2, 3},
		Amount:          32000000000,
	}

	cloned := withdrawalRequest.Clone().(*WithdrawalRequest)
	assert.NotNil(t, cloned)
	// Clone returns empty struct, not a copy
	assert.Equal(t, &WithdrawalRequest{}, cloned)
}

// TestWithdrawalRequestStatic tests Static method
func TestWithdrawalRequestStatic(t *testing.T) {
	t.Parallel()

	withdrawalRequest := &WithdrawalRequest{}
	assert.True(t, withdrawalRequest.Static())
}

// TestPendingPartialWithdrawalEncodeDecode tests SSZ encoding and decoding of PendingPartialWithdrawal
func TestPendingPartialWithdrawalEncodeDecode(t *testing.T) {
	t.Parallel()

	pendingWithdrawal := &PendingPartialWithdrawal{
		ValidatorIndex:    12345,
		Amount:            32000000000, // 32 ETH in Gwei
		WithdrawableEpoch: 100,
	}

	// Verify encoding size
	assert.Equal(t, SizePendingPartialWithdrawal, pendingWithdrawal.EncodingSizeSSZ())
	assert.Equal(t, 24, SizePendingPartialWithdrawal, "SizePendingPartialWithdrawal should be 24 bytes (8+8+8)")

	// Encode to SSZ
	encoded, err := pendingWithdrawal.EncodeSSZ(nil)
	require.NoError(t, err)
	assert.Len(t, encoded, SizePendingPartialWithdrawal)

	// Decode from SSZ
	decoded := &PendingPartialWithdrawal{}
	err = decoded.DecodeSSZ(encoded, 0)
	require.NoError(t, err)

	// Verify decoded values
	assert.Equal(t, pendingWithdrawal.ValidatorIndex, decoded.ValidatorIndex)
	assert.Equal(t, pendingWithdrawal.Amount, decoded.Amount)
	assert.Equal(t, pendingWithdrawal.WithdrawableEpoch, decoded.WithdrawableEpoch)
}

// TestPendingPartialWithdrawalHashSSZ tests HashSSZ of PendingPartialWithdrawal
func TestPendingPartialWithdrawalHashSSZ(t *testing.T) {
	t.Parallel()

	pendingWithdrawal := &PendingPartialWithdrawal{
		ValidatorIndex:    12345,
		Amount:            32000000000,
		WithdrawableEpoch: 100,
	}

	hash, err := pendingWithdrawal.HashSSZ()
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, hash)
}

// TestWithdrawalRequestDataLength verifies withdrawal request data matches EIP-7002 specification
func TestWithdrawalRequestDataLength(t *testing.T) {
	t.Parallel()

	// EIP-7002 withdrawal request data structure:
	// source_address: 20 bytes
	// validator_pubkey: 48 bytes
	// amount: 8 bytes (little-endian uint64, in Gwei)
	expectedLen := 20 + 48 + 8

	assert.Equal(t, expectedLen, SizeWithdrawalRequest)
	assert.Equal(t, 76, SizeWithdrawalRequest)
}

// TestWithdrawalRequestInterfaceCompliance verifies WithdrawalRequest implements required interfaces
func TestWithdrawalRequestInterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ EncodableHashableSSZ = (*WithdrawalRequest)(nil)
	var _ EncodableHashableSSZ = (*PendingPartialWithdrawal)(nil)
}

// TestWithdrawalRequestFullExit tests full exit request (amount = 0)
func TestWithdrawalRequestFullExit(t *testing.T) {
	t.Parallel()

	// EIP-7002: Amount = 0 indicates a full exit request
	fullExitRequest := &WithdrawalRequest{
		SourceAddress:   libcommon.HexToAddress("0x1234567890123456789012345678901234567890"),
		ValidatorPubKey: libcommon.Bytes48{1, 2, 3},
		Amount:          0, // Full exit
	}

	encoded, err := fullExitRequest.EncodeSSZ(nil)
	require.NoError(t, err)

	decoded := &WithdrawalRequest{}
	err = decoded.DecodeSSZ(encoded, 0)
	require.NoError(t, err)

	assert.Equal(t, uint64(0), decoded.Amount, "Full exit should have amount = 0")
}

// TestWithdrawalRequestPartialWithdrawal tests partial withdrawal request
func TestWithdrawalRequestPartialWithdrawal(t *testing.T) {
	t.Parallel()

	// EIP-7002: Amount > 0 indicates a partial withdrawal request
	partialWithdrawalRequest := &WithdrawalRequest{
		SourceAddress:   libcommon.HexToAddress("0x1234567890123456789012345678901234567890"),
		ValidatorPubKey: libcommon.Bytes48{1, 2, 3},
		Amount:          1000000000, // 1 ETH in Gwei
	}

	encoded, err := partialWithdrawalRequest.EncodeSSZ(nil)
	require.NoError(t, err)

	decoded := &WithdrawalRequest{}
	err = decoded.DecodeSSZ(encoded, 0)
	require.NoError(t, err)

	assert.Equal(t, uint64(1000000000), decoded.Amount, "Partial withdrawal should have specified amount")
}

