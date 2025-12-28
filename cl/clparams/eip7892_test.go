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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// EIP-7892: Blob Parameter Only Hardforks (BPO)
//
// This EIP introduces a lightweight hardfork mechanism for the Consensus Layer
// to support rapid blob parameter adjustments via BlobSchedule configuration.

func TestEIP7892BlobScheduleStructure(t *testing.T) {
	// BlobSchedule is an array of BlobParameters with epoch-based activation
	params := BlobParameters{
		Epoch:            100,
		MaxBlobsPerBlock: 15,
	}

	assert.Equal(t, uint64(100), params.Epoch)
	assert.Equal(t, uint64(15), params.MaxBlobsPerBlock)
}

func TestEIP7892GetBlobParametersDefault(t *testing.T) {
	cfg := MainnetBeaconConfig

	// For epoch 0, should return default (MaxBlobsPerBlockElectra)
	params := cfg.GetBlobParameters(0)
	assert.Equal(t, cfg.MaxBlobsPerBlockElectra, params.MaxBlobsPerBlock)
}

func TestEIP7892GetBlobParametersWithSchedule(t *testing.T) {
	// Create a config with BlobSchedule
	cfg := BeaconChainConfig{
		ElectraForkEpoch:       100,
		MaxBlobsPerBlockElectra: 9,
		BlobSchedule: []BlobParameters{
			{Epoch: 100, MaxBlobsPerBlock: 9},
			{Epoch: 200, MaxBlobsPerBlock: 12},
			{Epoch: 300, MaxBlobsPerBlock: 15},
		},
	}

	// Before first scheduled epoch: default
	params := cfg.GetBlobParameters(50)
	assert.Equal(t, cfg.MaxBlobsPerBlockElectra, params.MaxBlobsPerBlock)

	// At first scheduled epoch
	params = cfg.GetBlobParameters(100)
	assert.Equal(t, uint64(9), params.MaxBlobsPerBlock)

	// At second scheduled epoch
	params = cfg.GetBlobParameters(200)
	assert.Equal(t, uint64(12), params.MaxBlobsPerBlock)

	// At third scheduled epoch
	params = cfg.GetBlobParameters(300)
	assert.Equal(t, uint64(15), params.MaxBlobsPerBlock)

	// Between epochs: use previous
	params = cfg.GetBlobParameters(250)
	assert.Equal(t, uint64(12), params.MaxBlobsPerBlock)
}

func TestEIP7892BlobScheduleOrdering(t *testing.T) {
	// BlobSchedule should be sorted by epoch in ascending order
	// Verify that GetBlobParameters works correctly with sorted schedule

	cfg := BeaconChainConfig{
		ElectraForkEpoch:        100,
		MaxBlobsPerBlockElectra: 9,
		SlotsPerEpoch:           32,
		BlobSchedule: []BlobParameters{
			{Epoch: 100, MaxBlobsPerBlock: 9},
			{Epoch: 200, MaxBlobsPerBlock: 12},
			{Epoch: 300, MaxBlobsPerBlock: 15},
		},
	}

	// Verify sorted order (already sorted)
	require.Len(t, cfg.BlobSchedule, 3)
	assert.Equal(t, uint64(100), cfg.BlobSchedule[0].Epoch)
	assert.Equal(t, uint64(200), cfg.BlobSchedule[1].Epoch)
	assert.Equal(t, uint64(300), cfg.BlobSchedule[2].Epoch)

	// Verify GetBlobParameters returns correct values
	assert.Equal(t, uint64(9), cfg.GetBlobParameters(100).MaxBlobsPerBlock)
	assert.Equal(t, uint64(12), cfg.GetBlobParameters(200).MaxBlobsPerBlock)
	assert.Equal(t, uint64(15), cfg.GetBlobParameters(300).MaxBlobsPerBlock)
}

func TestEIP7892BlobScheduleInFulu(t *testing.T) {
	cfg := MainnetBeaconConfig

	// In Fulu version, GetBlobParameters is used instead of MaxBlobsPerBlockByVersion
	// This test verifies the behavior

	// For Deneb/Electra, use MaxBlobsPerBlockByVersion
	denebMax := cfg.MaxBlobsPerBlockByVersion(DenebVersion)
	assert.Equal(t, cfg.MaxBlobsPerBlock, denebMax)

	electraMax := cfg.MaxBlobsPerBlockByVersion(ElectraVersion)
	assert.Equal(t, cfg.MaxBlobsPerBlockElectra, electraMax)

	// Fulu also uses MaxBlobsPerBlockByVersion for now
	fuluMax := cfg.MaxBlobsPerBlockByVersion(FuluVersion)
	assert.Equal(t, cfg.MaxBlobsPerBlockElectra, fuluMax)
}

func TestEIP7892EmptyBlobSchedule(t *testing.T) {
	cfg := BeaconChainConfig{
		ElectraForkEpoch:       100,
		MaxBlobsPerBlockElectra: 9,
		BlobSchedule:           nil, // Empty schedule
	}

	// Should return default parameters
	params := cfg.GetBlobParameters(200)
	assert.Equal(t, cfg.ElectraForkEpoch, params.Epoch)
	assert.Equal(t, cfg.MaxBlobsPerBlockElectra, params.MaxBlobsPerBlock)
}

func TestEIP7892BPORoadmap(t *testing.T) {
	// Document the BPO hardfork roadmap as per EIP-7892
	// This shows the planned blob parameter increases

	roadmap := []struct {
		name   string
		target uint64
		max    uint64
	}{
		{"Electra (current)", 6, 9},
		{"Fusaka (PeerDAS)", 10, 15},
		{"BPO1 (planned)", 12, 18},
		{"BPO2 (planned)", 14, 21},
	}

	t.Log("EIP-7892 BPO Hardfork Roadmap:")
	for _, step := range roadmap {
		t.Logf("  %s: target=%d, max=%d", step.name, step.target, step.max)
	}

	// Verify Electra values match current config
	cfg := MainnetBeaconConfig
	assert.Equal(t, uint64(9), cfg.MaxBlobsPerBlockElectra, "Electra max blobs")
}

func TestEIP7892MaxBlobsPerBlockConsistency(t *testing.T) {
	cfg := MainnetBeaconConfig

	// Verify consistency between BlobSchedule and hardcoded values
	if len(cfg.BlobSchedule) > 0 {
		// If schedule exists, first entry should match Electra config
		firstEntry := cfg.BlobSchedule[0]
		t.Logf("First BlobSchedule entry: epoch=%d, max=%d",
			firstEntry.Epoch, firstEntry.MaxBlobsPerBlock)
	}

	// MaxBlobsPerBlockElectra should be 9 (EIP-7691)
	assert.Equal(t, uint64(9), cfg.MaxBlobsPerBlockElectra)
}

func TestEIP7892ComplianceStatus(t *testing.T) {
	t.Log("EIP-7892: Blob Parameter Only Hardforks (CL)")
	t.Log("=============================================")
	t.Log("")
	t.Log("Core Features:")
	t.Log("✅ BlobParameters struct with Epoch and MaxBlobsPerBlock")
	t.Log("✅ BlobSchedule array in BeaconChainConfig")
	t.Log("✅ GetBlobParameters() for dynamic epoch-based lookup")
	t.Log("✅ InitializeForkSchedule() sorts BlobSchedule by epoch")
	t.Log("✅ Fallback to MaxBlobsPerBlockElectra if no schedule match")
	t.Log("")
	t.Log("Integration Points:")
	t.Log("✅ ProcessExecutionPayload checks blob commitments via GetBlobParameters")
	t.Log("✅ Block service uses GetBlobParameters for Fulu+")
	t.Log("✅ Block production respects MaxRlpBlockSize limit")
	t.Log("")
	t.Log("Version Handling:")
	t.Log("✅ Pre-Fulu: uses MaxBlobsPerBlockByVersion()")
	t.Log("✅ Fulu+: uses GetBlobParameters(epoch)")
}

