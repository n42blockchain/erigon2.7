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

// EIP-7549: Attestation Committee Bits Tests (Electra)
func TestElectraAttestation(t *testing.T) {
	// Test Electra attestation with CommitteeBits
	data := &AttestationData{
		Slot:            100,
		CommitteeIndex:  0, // In Electra, committee index should be 0 in data, actual index is in CommitteeBits
		BeaconBlockRoot: libcommon.HexToHash("0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
		Source:          Checkpoint{Epoch: 10, Root: libcommon.HexToHash("0x1102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")},
		Target:          Checkpoint{Epoch: 11, Root: libcommon.HexToHash("0x2102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")},
	}
	signature := libcommon.Bytes96{}
	for i := range signature {
		signature[i] = byte(i)
	}

	// Create Electra attestation with CommitteeBits
	committeeBits := NewBitVector(64) // maxCommitteesPerSlot = 64
	committeeBits.SetBitAt(3, true)   // Set committee index 3

	aggregationBits := NewBitList(2, 2048*64) // Electra aggregation bits size
	// Set bits 0 and 5 (0b00100001 = 0x21)
	aggregationBits.Set(0, 0x21)

	attestation := &Attestation{
		AggregationBits: aggregationBits,
		Data:            data,
		Signature:       signature,
		CommitteeBits:   committeeBits,
	}

	// Test GetCommitteeIndexFromBits
	idx, err := attestation.GetCommitteeIndexFromBits()
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), idx)

	// Test SSZ encoding for Electra
	encoded, err := attestation.EncodeSSZ(nil)
	assert.NoError(t, err)
	assert.NotNil(t, encoded)

	// Test SSZ decoding for Electra
	decoded := &Attestation{}
	err = decoded.DecodeSSZ(encoded, int(7)) // ElectraVersion = 7
	assert.NoError(t, err)
	assert.NotNil(t, decoded.CommitteeBits)

	// Test HashSSZ for Electra
	hash, err := attestation.HashSSZ()
	assert.NoError(t, err)
	assert.NotNil(t, hash)

	// Test Copy for Electra
	copied := attestation.Copy()
	assert.NotNil(t, copied.CommitteeBits)
	copiedIdx, err := copied.GetCommitteeIndexFromBits()
	assert.NoError(t, err)
	assert.Equal(t, idx, copiedIdx)
}

func TestGetCommitteeIndexFromBits_NoCommitteeBitsSet(t *testing.T) {
	committeeBits := NewBitVector(64)
	// Don't set any bits
	attestation := &Attestation{
		CommitteeBits: committeeBits,
	}

	_, err := attestation.GetCommitteeIndexFromBits()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no committee bits set")
}

func TestGetCommitteeIndexFromBits_MultipleCommitteeBits(t *testing.T) {
	committeeBits := NewBitVector(64)
	committeeBits.SetBitAt(2, true)
	committeeBits.SetBitAt(5, true) // Multiple bits set

	attestation := &Attestation{
		CommitteeBits: committeeBits,
	}

	// Should return the first set bit
	idx, err := attestation.GetCommitteeIndexFromBits()
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), idx)
}

// EIP-7549: SingleAttestation Tests (Electra)
func TestSingleAttestation(t *testing.T) {
	data := &AttestationData{
		Slot:            100,
		CommitteeIndex:  0,
		BeaconBlockRoot: libcommon.HexToHash("0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
		Source:          Checkpoint{Epoch: 10, Root: libcommon.HexToHash("0x1102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")},
		Target:          Checkpoint{Epoch: 11, Root: libcommon.HexToHash("0x2102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")},
	}
	signature := libcommon.Bytes96{}
	for i := range signature {
		signature[i] = byte(i)
	}

	singleAtt := &SingleAttestation{
		CommitteeIndex: 3,
		AttesterIndex:  42,
		Data:           data,
		Signature:      signature,
	}

	// Test SSZ encoding
	encoded, err := singleAtt.EncodeSSZ(nil)
	assert.NoError(t, err)
	assert.NotNil(t, encoded)

	// Test SSZ decoding
	decoded := &SingleAttestation{}
	err = decoded.DecodeSSZ(encoded, 0)
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), decoded.CommitteeIndex)
	assert.Equal(t, uint64(42), decoded.AttesterIndex)

	// Test HashSSZ
	hash, err := singleAtt.HashSSZ()
	assert.NoError(t, err)
	assert.NotNil(t, hash)

	// Test Clone
	cloned := singleAtt.Clone().(*SingleAttestation)
	assert.NotNil(t, cloned)

	// Test Static
	assert.True(t, singleAtt.Static())

	// Test AttestationData getter
	assert.Equal(t, data, singleAtt.AttestationData())
}

func TestSingleAttestationToAttestation(t *testing.T) {
	data := &AttestationData{
		Slot:            100,
		CommitteeIndex:  0,
		BeaconBlockRoot: libcommon.HexToHash("0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
		Source:          Checkpoint{Epoch: 10, Root: libcommon.HexToHash("0x1102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")},
		Target:          Checkpoint{Epoch: 11, Root: libcommon.HexToHash("0x2102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")},
	}
	signature := libcommon.Bytes96{}
	for i := range signature {
		signature[i] = byte(i)
	}

	singleAtt := &SingleAttestation{
		CommitteeIndex: 5,
		AttesterIndex:  42,
		Data:           data,
		Signature:      signature,
	}

	// Convert to Attestation with member index 3 in a committee of size 100
	memberIndex := 3
	committeeLen := 100
	attestation := singleAtt.ToAttestation(memberIndex, committeeLen)

	// Verify CommitteeBits
	assert.NotNil(t, attestation.CommitteeBits)
	idx, err := attestation.GetCommitteeIndexFromBits()
	assert.NoError(t, err)
	assert.Equal(t, uint64(5), idx)

	// Verify AggregationBits - the bit at memberIndex should be set
	assert.NotNil(t, attestation.AggregationBits)
	assert.True(t, attestation.AggregationBits.GetBitAt(memberIndex))

	// Verify Data and Signature are preserved
	assert.Equal(t, data, attestation.Data)
	assert.Equal(t, signature, attestation.Signature)
}

func TestElectraAttestationEncodingSizeSSZ(t *testing.T) {
	// Electra attestation
	committeeBits := NewBitVector(64)
	committeeBits.SetBitAt(3, true)
	aggregationBits := NewBitList(10, 2048*64)

	electraAtt := &Attestation{
		AggregationBits: aggregationBits,
		Data:            &AttestationData{},
		Signature:       libcommon.Bytes96{},
		CommitteeBits:   committeeBits,
	}

	electraSize := electraAtt.EncodingSizeSSZ()
	assert.Greater(t, electraSize, 0)

	// Deneb attestation (no CommitteeBits)
	denebAggBits := NewBitList(10, 2048)
	denebAtt := &Attestation{
		AggregationBits: denebAggBits,
		Data:            &AttestationData{},
		Signature:       libcommon.Bytes96{},
		CommitteeBits:   nil,
	}

	denebSize := denebAtt.EncodingSizeSSZ()
	assert.Greater(t, denebSize, 0)

	// Electra attestation should be larger due to CommitteeBits
	assert.Greater(t, electraSize, denebSize)
}

func TestAttestationJSONUnmarshalElectra(t *testing.T) {
	// Electra JSON with committee_bits
	electraJSON := `{
		"aggregation_bits": "0x01",
		"data": {
			"slot": "100",
			"index": "0",
			"beacon_block_root": "0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
			"source": {"epoch": "10", "root": "0x1102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"},
			"target": {"epoch": "11", "root": "0x2102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"}
		},
		"signature": "0x000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f404142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f",
		"committee_bits": "0x08"
	}`

	var att Attestation
	err := att.UnmarshalJSON([]byte(electraJSON))
	assert.NoError(t, err)
	assert.NotNil(t, att.CommitteeBits)

	// Verify committee bits were set correctly (bit 3 should be set based on 0x08)
	idx, err := att.GetCommitteeIndexFromBits()
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), idx)
}

func TestAttestationJSONUnmarshalDeneb(t *testing.T) {
	// Deneb JSON without committee_bits
	denebJSON := `{
		"aggregation_bits": "0x01",
		"data": {
			"slot": "100",
			"index": "1",
			"beacon_block_root": "0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
			"source": {"epoch": "10", "root": "0x1102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"},
			"target": {"epoch": "11", "root": "0x2102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"}
		},
		"signature": "0x000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f404142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f"
	}`

	var att Attestation
	err := att.UnmarshalJSON([]byte(denebJSON))
	assert.NoError(t, err)
	assert.Nil(t, att.CommitteeBits)
	assert.Equal(t, uint64(1), att.Data.CommitteeIndex)
}
