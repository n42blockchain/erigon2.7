// Copyright 2024 The Erigon Authors
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

package services

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/erigontech/erigon/cl/abstract"
	"github.com/erigontech/erigon/cl/antiquary/tests"
	"github.com/erigontech/erigon/cl/beacon/beaconevents"
	"github.com/erigontech/erigon/cl/beacon/synced_data"
	"github.com/erigontech/erigon/cl/clparams"
	"github.com/erigontech/erigon/cl/cltypes"
	"github.com/erigontech/erigon/cl/cltypes/solid"
	"github.com/erigontech/erigon/cl/phase1/forkchoice/mock_services"
	"github.com/erigontech/erigon/cl/utils/eth_clock"
	mockCommittee "github.com/erigontech/erigon/cl/validator/committee_subscription/mock_services"
	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/types/ssz"
)

var (
	mockSlot          = uint64(321)
	mockEpoch         = uint64(10)
	mockSlotsPerEpoch = uint64(32)
	attData           = &solid.AttestationData{
		Slot:            mockSlot,
		CommitteeIndex:  2,
		BeaconBlockRoot: [32]byte{0, 4, 2, 6},
		Source:          solid.Checkpoint{Epoch: mockEpoch, Root: [32]byte{1, 0}},
		Target:          solid.Checkpoint{Epoch: mockEpoch, Root: [32]byte{1, 0}},
	}

	att = &solid.Attestation{
		AggregationBits: solid.BitlistFromBytes([]byte{0b00000001, 1}, 2048),
		Data:            attData,
		Signature:       [96]byte{'a', 'b', 'c', 'd', 'e', 'f'},
	}
)

type attestationTestSuite struct {
	suite.Suite
	gomockCtrl        *gomock.Controller
	mockForkChoice    *mock_services.ForkChoiceStorageMock
	syncedData        synced_data.SyncedData
	committeeSubscibe *mockCommittee.MockCommitteeSubscribe
	ethClock          *eth_clock.MockEthereumClock
	attService        AttestationService
	beaconConfig      *clparams.BeaconChainConfig
}

func (t *attestationTestSuite) SetupTest() {
	t.gomockCtrl = gomock.NewController(t.T())
	t.mockForkChoice = &mock_services.ForkChoiceStorageMock{}
	_, st, _ := tests.GetBellatrixRandom()
	t.syncedData = synced_data.NewSyncedDataManager(&clparams.MainnetBeaconConfig, true)
	t.syncedData.OnHeadState(st)
	t.committeeSubscibe = mockCommittee.NewMockCommitteeSubscribe(t.gomockCtrl)
	t.ethClock = eth_clock.NewMockEthereumClock(t.gomockCtrl)
	t.beaconConfig = &clparams.BeaconChainConfig{
		SlotsPerEpoch:    mockSlotsPerEpoch,
		ElectraForkEpoch: 100000,
	}
	netConfig := &clparams.NetworkConfig{}
	emitters := beaconevents.NewEventEmitter()
	computeSigningRoot = func(obj ssz.HashableSSZ, domain []byte) ([32]byte, error) { return [32]byte{}, nil }
	batchSignatureVerifier := NewBatchSignatureVerifier(context.TODO(), nil)
	go batchSignatureVerifier.Start()
	ctx, cn := context.WithCancel(context.Background())
	cn()
	t.attService = NewAttestationService(ctx, t.mockForkChoice, t.committeeSubscibe, t.ethClock, t.syncedData, t.beaconConfig, netConfig, emitters, batchSignatureVerifier)
}

func (t *attestationTestSuite) TearDownTest() {
	t.gomockCtrl.Finish()
}

func (t *attestationTestSuite) TestAttestationProcessMessage() {
	type args struct {
		ctx    context.Context
		subnet *uint64
		msg    *solid.Attestation
	}
	tests := []struct {
		name    string
		wantErr bool
		mock    func()
		args    args
	}{
		{
			name: "Test attestation with committee index out of range",
			mock: func() {
				computeCommitteeCountPerSlot = func(_ abstract.BeaconStateReader, _, _ uint64) uint64 {
					return 1
				}
				t.ethClock.EXPECT().GetEpochAtSlot(mockSlot).Return(mockEpoch).Times(1)
				t.ethClock.EXPECT().GetCurrentSlot().Return(mockSlot).Times(1)
			},
			args: args{
				ctx:    context.Background(),
				subnet: nil,
				msg:    att,
			},
			wantErr: true,
		},
		{
			name: "Test attestation with wrong subnet",
			mock: func() {
				computeCommitteeCountPerSlot = func(_ abstract.BeaconStateReader, _, _ uint64) uint64 {
					return 5
				}
				computeSubnetForAttestation = func(_, _, _, _, _ uint64) uint64 {
					return 2
				}
				t.ethClock.EXPECT().GetCurrentSlot().Return(mockSlot).Times(1)
				t.ethClock.EXPECT().GetEpochAtSlot(mockSlot).Return(mockEpoch).Times(1)
			},
			args: args{
				ctx:    context.Background(),
				subnet: uint64Ptr(1),
				msg:    att,
			},
			wantErr: true,
		},
		{
			name: "Test attestation with wrong slot (current_slot < slot)",
			mock: func() {
				computeCommitteeCountPerSlot = func(_ abstract.BeaconStateReader, _, _ uint64) uint64 {
					return 5
				}
				computeSubnetForAttestation = func(_, _, _, _, _ uint64) uint64 {
					return 1
				}
				t.ethClock.EXPECT().GetEpochAtSlot(mockSlot).Return(mockEpoch).Times(1)
				t.ethClock.EXPECT().GetCurrentSlot().Return(uint64(1)).Times(1)
			},
			args: args{
				ctx:    context.Background(),
				subnet: uint64Ptr(1),
				msg:    att,
			},
			wantErr: true,
		},
		{
			name: "Attestation is aggregated",
			mock: func() {
				computeCommitteeCountPerSlot = func(_ abstract.BeaconStateReader, _, _ uint64) uint64 {
					return 5
				}
				computeSubnetForAttestation = func(_, _, _, _, _ uint64) uint64 {
					return 1
				}
				t.ethClock.EXPECT().GetEpochAtSlot(mockSlot).Return(mockEpoch).Times(1)
				t.ethClock.EXPECT().GetCurrentSlot().Return(mockSlot).Times(1)
			},
			args: args{
				ctx:    context.Background(),
				subnet: uint64Ptr(1),
				msg: &solid.Attestation{
					AggregationBits: solid.BitlistFromBytes([]byte{0b10000001, 1}, 2048),
					Data:            attData,
					Signature:       [96]byte{0, 1, 2, 3, 4, 5},
				},
			},
			wantErr: true,
		},
		{
			name: "Attestation is empty",
			mock: func() {
				computeCommitteeCountPerSlot = func(_ abstract.BeaconStateReader, _, _ uint64) uint64 {
					return 5
				}
				computeSubnetForAttestation = func(_, _, _, _, _ uint64) uint64 {
					return 1
				}
				t.ethClock.EXPECT().GetEpochAtSlot(mockSlot).Return(mockEpoch).Times(1)
				t.ethClock.EXPECT().GetCurrentSlot().Return(mockSlot).Times(1)
			},
			args: args{
				ctx:    context.Background(),
				subnet: uint64Ptr(1),
				msg: &solid.Attestation{
					AggregationBits: solid.BitlistFromBytes([]byte{0b0, 1}, 2048),
					Data:            attData,
					Signature:       [96]byte{0, 1, 2, 3, 4, 5},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid signature",
			mock: func() {
				computeCommitteeCountPerSlot = func(_ abstract.BeaconStateReader, _, _ uint64) uint64 {
					return 5
				}
				computeSubnetForAttestation = func(_, _, _, _, _ uint64) uint64 {
					return 1
				}
				t.ethClock.EXPECT().GetEpochAtSlot(mockSlot).Return(mockEpoch).Times(1)
				t.ethClock.EXPECT().GetCurrentSlot().Return(mockSlot).Times(1)
				computeSigningRoot = func(obj ssz.HashableSSZ, domain []byte) ([32]byte, error) {
					return [32]byte{}, nil
				}
			},
			args: args{
				ctx:    context.Background(),
				subnet: uint64Ptr(1),
				msg:    att,
			},
			wantErr: true,
		},
		{
			name: "block header not found",
			mock: func() {
				computeCommitteeCountPerSlot = func(_ abstract.BeaconStateReader, _, _ uint64) uint64 {
					return 8
				}
				computeSubnetForAttestation = func(_, _, _, _, _ uint64) uint64 {
					return 1
				}
				t.ethClock.EXPECT().GetEpochAtSlot(mockSlot).Return(mockEpoch).Times(1)
				t.ethClock.EXPECT().GetCurrentSlot().Return(mockSlot).Times(1)
				computeSigningRoot = func(obj ssz.HashableSSZ, domain []byte) ([32]byte, error) {
					return [32]byte{}, nil
				}
			},
			args: args{
				ctx:    context.Background(),
				subnet: uint64Ptr(1),
				msg:    att,
			},
			wantErr: true,
		},
		{
			name: "invalid target block",
			mock: func() {
				computeCommitteeCountPerSlot = func(_ abstract.BeaconStateReader, _, _ uint64) uint64 {
					return 8
				}
				computeSubnetForAttestation = func(_, _, _, _, _ uint64) uint64 {
					return 1
				}
				t.ethClock.EXPECT().GetEpochAtSlot(mockSlot).Return(mockEpoch).Times(1)
				t.ethClock.EXPECT().GetCurrentSlot().Return(mockSlot).Times(1)
				computeSigningRoot = func(obj ssz.HashableSSZ, domain []byte) ([32]byte, error) {
					return [32]byte{}, nil
				}
				t.mockForkChoice.Headers = map[libcommon.Hash]*cltypes.BeaconBlockHeader{
					att.Data.BeaconBlockRoot: {}, // wrong block root
				}
			},
			args: args{
				ctx:    context.Background(),
				subnet: uint64Ptr(1),
				msg:    att,
			},
			wantErr: true,
		},
		{
			name: "invalid finality checkpoint",
			mock: func() {
				computeCommitteeCountPerSlot = func(_ abstract.BeaconStateReader, _, _ uint64) uint64 {
					return 8
				}
				computeSubnetForAttestation = func(_, _, _, _, _ uint64) uint64 {
					return 1
				}
				t.ethClock.EXPECT().GetEpochAtSlot(mockSlot).Return(mockEpoch).Times(1)
				t.ethClock.EXPECT().GetCurrentSlot().Return(mockSlot).Times(1)
				computeSigningRoot = func(obj ssz.HashableSSZ, domain []byte) ([32]byte, error) {
					return [32]byte{}, nil
				}
				t.mockForkChoice.Headers = map[libcommon.Hash]*cltypes.BeaconBlockHeader{
					att.Data.BeaconBlockRoot: {},
				}
				mockFinalizedCheckPoint := &solid.Checkpoint{Root: [32]byte{1, 0}, Epoch: 1}
				t.mockForkChoice.Ancestors = map[uint64]libcommon.Hash{
					mockEpoch * mockSlotsPerEpoch:                     att.Data.Target.Root,
					mockFinalizedCheckPoint.Epoch * mockSlotsPerEpoch: {}, // wrong block root
				}
				t.mockForkChoice.FinalizedCheckpointVal = *mockFinalizedCheckPoint
			},
			args: args{
				ctx:    context.Background(),
				subnet: uint64Ptr(1),
				msg:    att,
			},
			wantErr: true,
		},
		{
			name: "success",
			mock: func() {
				computeCommitteeCountPerSlot = func(_ abstract.BeaconStateReader, _, _ uint64) uint64 {
					return 8
				}
				computeSubnetForAttestation = func(_, _, _, _, _ uint64) uint64 {
					return 1
				}
				t.ethClock.EXPECT().GetEpochAtSlot(mockSlot).Return(mockEpoch).Times(1)
				t.ethClock.EXPECT().GetCurrentSlot().Return(mockSlot).Times(1)
				computeSigningRoot = func(obj ssz.HashableSSZ, domain []byte) ([32]byte, error) {
					return [32]byte{}, nil
				}
				blsVerifyMultipleSignatures = func(signatures [][]byte, signRoots [][]byte, pks [][]byte) (bool, error) {
					return true, nil
				}
				t.mockForkChoice.Headers = map[libcommon.Hash]*cltypes.BeaconBlockHeader{
					att.Data.BeaconBlockRoot: {},
				}

				mockFinalizedCheckPoint := &solid.Checkpoint{Root: [32]byte{1, 0}, Epoch: 1}
				t.mockForkChoice.Ancestors = map[uint64]libcommon.Hash{
					mockEpoch * mockSlotsPerEpoch:                     att.Data.Target.Root,
					mockFinalizedCheckPoint.Epoch * mockSlotsPerEpoch: mockFinalizedCheckPoint.Root,
				}
				t.mockForkChoice.FinalizedCheckpointVal = *mockFinalizedCheckPoint
				//t.committeeSubscibe.EXPECT().NeedToAggregate(att).Return(true).Times(1)
				t.committeeSubscibe.EXPECT().AggregateAttestation(att).Return(nil).Times(1)
			},
			args: args{
				ctx:    context.Background(),
				subnet: uint64Ptr(1),
				msg:    att,
			},
		},
	}

	for _, tt := range tests {
		log.Printf("test case: %s", tt.name)
		t.SetupTest()
		tt.mock()
		err := t.attService.ProcessMessage(tt.args.ctx, tt.args.subnet, &AttestationForGossip{
			Attestation:      tt.args.msg,
			ImmediateProcess: true,
		})
		time.Sleep(time.Millisecond * 60)
		if tt.wantErr {
			t.Require().Error(err)
		} else {
			t.Require().NoError(err)
		}

		t.True(t.gomockCtrl.Satisfied())
	}
}

func TestAttestation(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	suite.Run(t, &attestationTestSuite{})
}

func uint64Ptr(i uint64) *uint64 {
	return &i
}

// EIP-7549: Electra SingleAttestation Tests
type electraAttestationTestSuite struct {
	suite.Suite
	gomockCtrl        *gomock.Controller
	mockForkChoice    *mock_services.ForkChoiceStorageMock
	syncedData        synced_data.SyncedData
	committeeSubscibe *mockCommittee.MockCommitteeSubscribe
	ethClock          *eth_clock.MockEthereumClock
	attService        AttestationService
	beaconConfig      *clparams.BeaconChainConfig
	netConfig         *clparams.NetworkConfig
}

func (t *electraAttestationTestSuite) SetupTest() {
	t.gomockCtrl = gomock.NewController(t.T())
	t.mockForkChoice = &mock_services.ForkChoiceStorageMock{}
	_, st, _ := tests.GetBellatrixRandom()
	t.syncedData = synced_data.NewSyncedDataManager(&clparams.MainnetBeaconConfig, true)
	t.syncedData.OnHeadState(st)
	t.committeeSubscibe = mockCommittee.NewMockCommitteeSubscribe(t.gomockCtrl)
	t.ethClock = eth_clock.NewMockEthereumClock(t.gomockCtrl)
	t.beaconConfig = &clparams.BeaconChainConfig{
		SlotsPerEpoch:    mockSlotsPerEpoch,
		ElectraForkEpoch: 0, // Electra is active from epoch 0
	}
	t.netConfig = &clparams.NetworkConfig{
		AttestationSubnetCount: 64,
	}
	emitters := beaconevents.NewEventEmitter()
	computeSigningRoot = func(obj ssz.HashableSSZ, domain []byte) ([32]byte, error) { return [32]byte{}, nil }
	batchSignatureVerifier := NewBatchSignatureVerifier(context.TODO(), nil)
	go batchSignatureVerifier.Start()
	ctx, cn := context.WithCancel(context.Background())
	cn()
	t.attService = NewAttestationService(ctx, t.mockForkChoice, t.committeeSubscibe, t.ethClock, t.syncedData, t.beaconConfig, t.netConfig, emitters, batchSignatureVerifier)
}

func (t *electraAttestationTestSuite) TearDownTest() {
	t.gomockCtrl.Finish()
}

// TestElectraSingleAttestationCommitteeIndexMustBeZeroInData tests EIP-7549 requirement:
// For Electra, the attestation.data.index (CommitteeIndex) must be 0
func (t *electraAttestationTestSuite) TestElectraSingleAttestationCommitteeIndexMustBeZeroInData() {
	// Create SingleAttestation with non-zero committee index in data (should fail)
	singleAtt := &solid.SingleAttestation{
		CommitteeIndex: 3,
		AttesterIndex:  42,
		Data: &solid.AttestationData{
			Slot:            mockSlot,
			CommitteeIndex:  1, // EIP-7549: This should be 0 in Electra
			BeaconBlockRoot: [32]byte{0, 4, 2, 6},
			Source:          solid.Checkpoint{Epoch: mockEpoch, Root: [32]byte{1, 0}},
			Target:          solid.Checkpoint{Epoch: mockEpoch, Root: [32]byte{1, 0}},
		},
		Signature: [96]byte{'a', 'b', 'c', 'd', 'e', 'f'},
	}

	computeCommitteeCountPerSlot = func(_ abstract.BeaconStateReader, _, _ uint64) uint64 {
		return 8
	}
	computeSubnetForAttestation = func(_, _, _, _, _ uint64) uint64 {
		return 1
	}
	t.ethClock.EXPECT().GetEpochAtSlot(mockSlot).Return(mockEpoch).Times(1)
	t.ethClock.EXPECT().GetCurrentSlot().Return(mockSlot).Times(1)

	subnet := uint64Ptr(1)
	err := t.attService.ProcessMessage(context.Background(), subnet, &AttestationForGossip{
		SingleAttestation: singleAtt,
		ImmediateProcess:  true,
	})

	// Should fail because data.CommitteeIndex != 0
	t.Require().Error(err)
	t.Contains(err.Error(), "committee index must be 0")
}

// TestElectraSingleAttestationValidCommitteeIndex tests EIP-7549:
// Committee index should be extracted from SingleAttestation.CommitteeIndex field
func (t *electraAttestationTestSuite) TestElectraSingleAttestationValidCommitteeIndex() {
	// Create valid SingleAttestation for Electra
	singleAtt := &solid.SingleAttestation{
		CommitteeIndex: 3, // This is where committee index is in Electra
		AttesterIndex:  0, // First validator in committee
		Data: &solid.AttestationData{
			Slot:            mockSlot,
			CommitteeIndex:  0, // Must be 0 in Electra (EIP-7549)
			BeaconBlockRoot: [32]byte{0, 4, 2, 6},
			Source:          solid.Checkpoint{Epoch: mockEpoch, Root: [32]byte{1, 0}},
			Target:          solid.Checkpoint{Epoch: mockEpoch, Root: [32]byte{1, 0}},
		},
		Signature: [96]byte{'a', 'b', 'c', 'd', 'e', 'f'},
	}

	computeCommitteeCountPerSlot = func(_ abstract.BeaconStateReader, _, _ uint64) uint64 {
		return 8
	}
	computeSubnetForAttestation = func(_, _, _, _, _ uint64) uint64 {
		return 1
	}
	t.ethClock.EXPECT().GetEpochAtSlot(mockSlot).Return(mockEpoch).Times(1)
	t.ethClock.EXPECT().GetCurrentSlot().Return(mockSlot).Times(1)
	computeSigningRoot = func(obj ssz.HashableSSZ, domain []byte) ([32]byte, error) {
		return [32]byte{}, nil
	}
	blsVerifyMultipleSignatures = func(signatures [][]byte, signRoots [][]byte, pks [][]byte) (bool, error) {
		return true, nil
	}
	t.mockForkChoice.Headers = map[libcommon.Hash]*cltypes.BeaconBlockHeader{
		singleAtt.Data.BeaconBlockRoot: {},
	}

	mockFinalizedCheckPoint := &solid.Checkpoint{Root: [32]byte{1, 0}, Epoch: 1}
	t.mockForkChoice.Ancestors = map[uint64]libcommon.Hash{
		mockEpoch * mockSlotsPerEpoch:                     singleAtt.Data.Target.Root,
		mockFinalizedCheckPoint.Epoch * mockSlotsPerEpoch: mockFinalizedCheckPoint.Root,
	}
	t.mockForkChoice.FinalizedCheckpointVal = *mockFinalizedCheckPoint

	subnet := uint64Ptr(1)
	err := t.attService.ProcessMessage(context.Background(), subnet, &AttestationForGossip{
		SingleAttestation: singleAtt,
		ImmediateProcess:  true,
	})

	// Should fail because attester is not in committee (mock state doesn't have the validator)
	// But the committee index validation should pass
	t.Require().Error(err)
	t.Contains(err.Error(), "attester is not a member of the committee")
}

// TestElectraSingleAttestationWithData tests EIP-7549 SingleAttestation with valid data structure
func (t *electraAttestationTestSuite) TestElectraSingleAttestationWithData() {
	// Create a valid SingleAttestation with correct EIP-7549 structure
	singleAtt := &solid.SingleAttestation{
		CommitteeIndex: 0,
		AttesterIndex:  0,
		Data: &solid.AttestationData{
			Slot:            mockSlot,
			CommitteeIndex:  0, // Must be 0 in Electra (EIP-7549)
			BeaconBlockRoot: [32]byte{0, 4, 2, 6},
			Source:          solid.Checkpoint{Epoch: mockEpoch, Root: [32]byte{1, 0}},
			Target:          solid.Checkpoint{Epoch: mockEpoch, Root: [32]byte{1, 0}},
		},
		Signature: [96]byte{'a', 'b', 'c', 'd', 'e', 'f'},
	}

	// Verify the SingleAttestation structure is correct
	t.Require().NotNil(singleAtt.Data)
	t.Require().Equal(uint64(0), singleAtt.Data.CommitteeIndex, "EIP-7549: data.CommitteeIndex must be 0 in Electra")
	t.Require().Equal(uint64(0), singleAtt.CommitteeIndex, "CommitteeIndex in SingleAttestation container")
	t.Require().Equal(uint64(0), singleAtt.AttesterIndex, "AttesterIndex in SingleAttestation")

	// Test ToAttestation conversion (EIP-7549 core functionality)
	attestation := singleAtt.ToAttestation(5, 100) // member index 5 in committee of 100
	t.Require().NotNil(attestation)
	t.Require().NotNil(attestation.CommitteeBits)
	t.Require().NotNil(attestation.AggregationBits)

	// Verify CommitteeBits is set correctly
	idx, err := attestation.GetCommitteeIndexFromBits()
	t.Require().NoError(err)
	t.Require().Equal(uint64(0), idx)

	// Verify AggregationBits has the member bit set
	t.Require().True(attestation.AggregationBits.GetBitAt(5))
}

func TestElectraAttestation(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	suite.Run(t, &electraAttestationTestSuite{})
}




























