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

package clparams

import (
	"testing"

	"github.com/erigontech/erigon-lib/chain"
	"github.com/stretchr/testify/assert"
)

// EIP-7691: Blob throughput increase
// Tests for CL blob configuration parameters

func TestEIP7691MaxBlobsPerBlockByVersion(t *testing.T) {
	cfg := MainnetBeaconConfig

	// Pre-Electra versions should use MaxBlobsPerBlock (6)
	assert.Equal(t, uint64(6), cfg.MaxBlobsPerBlockByVersion(Phase0Version))
	assert.Equal(t, uint64(6), cfg.MaxBlobsPerBlockByVersion(AltairVersion))
	assert.Equal(t, uint64(6), cfg.MaxBlobsPerBlockByVersion(BellatrixVersion))
	assert.Equal(t, uint64(6), cfg.MaxBlobsPerBlockByVersion(CapellaVersion))
	assert.Equal(t, uint64(6), cfg.MaxBlobsPerBlockByVersion(DenebVersion))

	// EIP-7691: Electra and Fulu use MaxBlobsPerBlockElectra (9)
	assert.Equal(t, uint64(9), cfg.MaxBlobsPerBlockByVersion(ElectraVersion))
	assert.Equal(t, uint64(9), cfg.MaxBlobsPerBlockByVersion(FuluVersion))
}

func TestEIP7691MaxBlobsConstants(t *testing.T) {
	cfg := MainnetBeaconConfig

	// EIP-7691 specifies:
	// Deneb: MAX_BLOBS_PER_BLOCK = 6
	// Electra: MAX_BLOBS_PER_BLOCK = 9
	assert.Equal(t, uint64(6), cfg.MaxBlobsPerBlock)
	assert.Equal(t, uint64(9), cfg.MaxBlobsPerBlockElectra)
}

func TestEIP7691MaxBlobGasPerBlock(t *testing.T) {
	cfg := MainnetBeaconConfig

	// MaxBlobGasPerBlock = MAX_BLOBS_PER_BLOCK * 131072 (BlobGasPerBlob)
	// Deneb: 6 * 131072 = 786432
	assert.Equal(t, uint64(786432), cfg.MaxBlobGasPerBlock)
}

func TestEIP7691MaxRequestBlobSidecarsByVersion(t *testing.T) {
	cfg := MainnetBeaconConfig

	// MaxRequestBlobSidecars = MAX_REQUEST_BLOCKS_DENEB * MAX_BLOBS_PER_BLOCK
	// Deneb: 128 * 6 = 768
	assert.Equal(t, 768, cfg.MaxRequestBlobSidecarsByVersion(DenebVersion))

	// Electra: 128 * 9 = 1152
	assert.Equal(t, 1152, cfg.MaxRequestBlobSidecarsByVersion(ElectraVersion))
	assert.Equal(t, 1152, cfg.MaxRequestBlobSidecarsByVersion(FuluVersion))
}

func TestEIP7691BlobCommitmentsPerBlock(t *testing.T) {
	cfg := MainnetBeaconConfig

	// MAX_BLOB_COMMITMENTS_PER_BLOCK = 4096 (maximum for SSZ list)
	assert.Equal(t, uint64(4096), cfg.MaxBlobCommittmentsPerBlock)
}

func TestEIP7691BlobSidecarSubnetCount(t *testing.T) {
	cfg := MainnetBeaconConfig

	// BlobSidecarSubnetCount for different versions
	// Deneb uses BlobSidecarSubnetCount (6)
	assert.Equal(t, uint64(6), cfg.BlobSidecarSubnetCountByVersion(DenebVersion))
	// EIP-7691: Electra and Fulu use BlobSidecarSubnetCountElectra (9) to match MaxBlobsPerBlockElectra
	assert.Equal(t, uint64(9), cfg.BlobSidecarSubnetCountByVersion(ElectraVersion))
	assert.Equal(t, uint64(9), cfg.BlobSidecarSubnetCountByVersion(FuluVersion))
}

func TestEIP7691SepoliaConfig(t *testing.T) {
	cfg := BeaconConfigs[NetworkType(chain.SepoliaChainID)]

	// Sepolia uses same blob values as mainnet
	assert.Equal(t, uint64(6), cfg.MaxBlobsPerBlock)
	assert.Equal(t, uint64(9), cfg.MaxBlobsPerBlockElectra)
}

func TestEIP7691GnosisConfig(t *testing.T) {
	cfg := BeaconConfigs[NetworkType(chain.GnosisChainID)]

	// Gnosis uses smaller blob limits
	assert.Equal(t, uint64(2), cfg.MaxBlobsPerBlock)
	assert.Equal(t, uint64(2), cfg.MaxBlobsPerBlockElectra)
}

func TestEIP7691BlobParametersMainnet(t *testing.T) {
	cfg := MainnetBeaconConfig

	// GetBlobParameters returns blob config for a given epoch
	// It iterates through BlobSchedule in descending order and returns matching entry
	// If no match, it returns default (ElectraForkEpoch, MaxBlobsPerBlockElectra)

	// For epoch 0, should return default (ElectraForkEpoch, MaxBlobsPerBlockElectra)
	// since no BlobSchedule entry matches epoch 0
	preElectraParams := cfg.GetBlobParameters(0)
	assert.Equal(t, cfg.MaxBlobsPerBlockElectra, preElectraParams.MaxBlobsPerBlock)

	// If BlobSchedule has entries, check them
	if len(cfg.BlobSchedule) > 0 {
		// Get params for the first scheduled epoch
		firstEntry := cfg.BlobSchedule[0]
		params := cfg.GetBlobParameters(firstEntry.Epoch)
		assert.Equal(t, firstEntry.MaxBlobsPerBlock, params.MaxBlobsPerBlock)
	}
}

func TestEIP7691BlobScheduleOrdering(t *testing.T) {
	cfg := MainnetBeaconConfig

	// Verify BlobSchedule entries are properly ordered for GetBlobParameters to work
	if len(cfg.BlobSchedule) > 1 {
		for i := 1; i < len(cfg.BlobSchedule); i++ {
			// Later entries should have higher or equal epochs
			assert.GreaterOrEqual(t, cfg.BlobSchedule[i].Epoch, cfg.BlobSchedule[i-1].Epoch)
		}
	}
}

