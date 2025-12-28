package solid

import (
	"testing"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/stretchr/testify/assert"
)

func TestAttestationData(t *testing.T) {
	slot := uint64(123)
	committeeIndex := uint64(456)
	beaconBlockRoot := libcommon.HexToHash("0x63426b1ac6f47473ce3386469f2408f992a0a18c52e343d63b6872be45f4e6f2")
	source := Checkpoint{Epoch: 123, Root: libcommon.HexToHash("0x63426b1ac6f47473ce3386469f2408f992a0a18c52e343d63b6872be45f4e6f1")}
	target := Checkpoint{Epoch: 456, Root: libcommon.HexToHash("0x63426b1ac6f47473ce3386469f2408f992a0a18c52e343d63b6872be45f4e6f3")}

	attData := &AttestationData{
		Slot:            slot,
		CommitteeIndex:  committeeIndex,
		BeaconBlockRoot: beaconBlockRoot,
		Source:          source,
		Target:          target,
	}

	// Ensure that the data was set correctly
	assert.Equal(t, slot, attData.Slot)
	assert.Equal(t, committeeIndex, attData.CommitteeIndex)
	assert.Equal(t, beaconBlockRoot, attData.BeaconBlockRoot)
	assert.True(t, attData.Source.Equal(source))
	assert.True(t, attData.Target.Equal(target))

	// Test clone functionality
	clone := attData.Clone().(*AttestationData)
	assert.NotNil(t, clone)

	// Test SSZ encoding and decoding
	encoded, err := attData.EncodeSSZ(nil)
	assert.NoError(t, err)

	decoded := &AttestationData{}
	err = decoded.DecodeSSZ(encoded, 0)
	assert.NoError(t, err)

	assert.True(t, attData.Equal(decoded))

	// Test SSZ Hash
	_, err = attData.HashSSZ()
	assert.NoError(t, err)

	// Test equality
	assert.True(t, attData.Equal(decoded))
	assert.False(t, attData.Equal(&AttestationData{}))
}

func TestAttestation(t *testing.T) {
	aggregationBits := []byte{1, 0, 1, 0, 1, 0, 1, 0}
	data := &AttestationData{
		Slot:            100,
		CommitteeIndex:  1,
		BeaconBlockRoot: libcommon.HexToHash("0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
		Source:          Checkpoint{Epoch: 10, Root: libcommon.HexToHash("0x1102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")},
		Target:          Checkpoint{Epoch: 11, Root: libcommon.HexToHash("0x2102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")},
	}
	signature := libcommon.Bytes96{}
	for i := range signature {
		signature[i] = byte(i)
	}

	// Test NewAttestation creation
	bitlist := NewBitList(len(aggregationBits), len(aggregationBits)*8)
	copy(bitlist.Bytes(), aggregationBits)
	attestation := &Attestation{
		AggregationBits: bitlist,
		Data:            data,
		Signature:       signature,
	}
	assert.NotNil(t, attestation)

	// Test getters
	assert.Equal(t, data, attestation.Data)
	assert.Equal(t, signature, attestation.Signature)

	// Test setters
	newData := &AttestationData{
		Slot:            200,
		CommitteeIndex:  2,
		BeaconBlockRoot: libcommon.HexToHash("0x3102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
		Source:          Checkpoint{Epoch: 20, Root: libcommon.HexToHash("0x4102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")},
		Target:          Checkpoint{Epoch: 21, Root: libcommon.HexToHash("0x5102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")},
	}
	newSignature := libcommon.Bytes96{}
	for i := range newSignature {
		newSignature[i] = byte(95 - i)
	}
	attestation.Data = newData
	attestation.Signature = newSignature
	assert.Equal(t, newData, attestation.Data)
	assert.Equal(t, newSignature, attestation.Signature)

	// Test HashSSZ
	hash, err := attestation.HashSSZ()
	assert.NoError(t, err)
	assert.NotNil(t, hash)

	// Test Clone
	cloned := attestation.Clone()
	assert.NotNil(t, cloned.(*Attestation))
}

func TestCheckpoint(t *testing.T) {
	epoch := uint64(123)
	root := libcommon.HexToHash("0x63426b1ac6f47473ce3386469f2408f992a0a18c52e343d63b6872be45f4e6f1")

	cp := Checkpoint{Epoch: epoch, Root: root}

	// Test values
	assert.Equal(t, epoch, cp.Epoch)
	assert.Equal(t, root, cp.Root)

	// Test SSZ encoding
	encoded, err := cp.EncodeSSZ(nil)
	assert.NoError(t, err)

	// Test SSZ decoding
	decoded := &Checkpoint{}
	err = decoded.DecodeSSZ(encoded, 0)
	assert.NoError(t, err)
	assert.True(t, cp.Equal(*decoded))

	// Test HashSSZ
	_, err = cp.HashSSZ()
	assert.NoError(t, err)

	// Test Copy
	copied := cp.Copy()
	assert.True(t, cp.Equal(*copied))
}
