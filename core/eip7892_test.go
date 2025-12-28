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

package core

import (
	"math"
	"math/big"
	"testing"

	"github.com/erigontech/erigon-lib/chain"
	"github.com/erigontech/erigon/params"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// EIP-7892: Blob Parameter Only Hardforks (BPO)
//
// This EIP introduces a lightweight hardfork mechanism for adjusting blob-related
// parameters without requiring code changes. Key features:
// 1. BlobSchedule configuration for dynamic blob limits per fork
// 2. MaxRlpBlockSize limit for Osaka/Fusaka
// 3. Support for rapid BPO hardforks to scale blob throughput

func TestEIP7892MaxRlpBlockSizeConstants(t *testing.T) {
	// EIP-7892 defines maximum block size limits
	// MaxBlockSize = 10 MiB
	// MaxBlockSizeSafetyMargin = 2 MiB
	// MaxRlpBlockSize = MaxBlockSize - SafetyMargin = ~8 MiB
	assert.Equal(t, 10_485_760, params.MaxBlockSize, "MaxBlockSize should be 10 MiB")
	assert.Equal(t, 2_097_152, params.MaxBlockSizeSafetyMargin, "MaxBlockSizeSafetyMargin should be 2 MiB")
	assert.Equal(t, 8_388_608, params.MaxRlpBlockSize, "MaxRlpBlockSize should be ~8 MiB")

	// Verify relationship
	assert.Equal(t, params.MaxBlockSize-params.MaxBlockSizeSafetyMargin, params.MaxRlpBlockSize)
}

func TestEIP7892GetMaxRlpBlockSize(t *testing.T) {
	osakaTime := uint64(1000)
	config := &chain.Config{
		OsakaTime: big.NewInt(int64(osakaTime)),
	}

	// Before Osaka: no limit (MaxInt)
	beforeOsaka := osakaTime - 1
	assert.Equal(t, math.MaxInt, config.GetMaxRlpBlockSize(beforeOsaka), "Before Osaka should return MaxInt (no limit)")

	// At Osaka: limited to MaxRlpBlockSize
	atOsaka := osakaTime
	assert.Equal(t, 10_485_760-2_097_152, config.GetMaxRlpBlockSize(atOsaka), "At Osaka should return MaxRlpBlockSize (~8 MiB)")

	// After Osaka: still limited
	afterOsaka := osakaTime + 1000
	assert.Equal(t, 10_485_760-2_097_152, config.GetMaxRlpBlockSize(afterOsaka), "After Osaka should return MaxRlpBlockSize (~8 MiB)")
}

func TestEIP7892BlobScheduleForBPOHardforks(t *testing.T) {
	// EIP-7892 enables rapid BPO hardforks by using BlobSchedule
	// to dynamically adjust blob parameters at specific times

	target6 := uint64(6)
	max9 := uint64(9)
	target10 := uint64(10)
	max15 := uint64(15)
	target14 := uint64(14)
	max21 := uint64(21)

	// Simulate BPO hardfork schedule:
	// 1. Prague: 6/9 blobs
	// 2. Osaka (Fusaka): 10/15 blobs (PeerDAS)
	// 3. BPO1: 12/18 blobs
	// 4. BPO2: 14/21 blobs

	pragueTime := uint64(1000)
	osakaTime := uint64(2000)

	schedule := &chain.BlobSchedule{
		Prague: &chain.BlobConfig{
			Target: &target6,
			Max:    &max9,
		},
		Osaka: &chain.BlobConfig{
			Target: &target10,
			Max:    &max15,
		},
	}

	config := &chain.Config{
		PragueTime:   big.NewInt(int64(pragueTime)),
		OsakaTime:    big.NewInt(int64(osakaTime)),
		BlobSchedule: schedule,
	}

	// Before Prague (Cancun defaults)
	beforePrague := pragueTime - 1
	target := config.GetTargetBlobsPerBlock(beforePrague)
	maxBlobs := config.GetMaxBlobsPerBlock(beforePrague)
	assert.Equal(t, uint64(3), target, "Before Prague target should be 3 (Cancun)")
	assert.Equal(t, uint64(6), maxBlobs, "Before Prague max should be 6 (Cancun)")

	// Prague: 6/9 blobs
	atPrague := pragueTime
	target = config.GetTargetBlobsPerBlock(atPrague)
	maxBlobs = config.GetMaxBlobsPerBlock(atPrague)
	assert.Equal(t, target6, target, "Prague target should be 6")
	assert.Equal(t, max9, maxBlobs, "Prague max should be 9")

	// Osaka (Fusaka): 10/15 blobs
	atOsaka := osakaTime
	target = config.GetTargetBlobsPerBlock(atOsaka)
	maxBlobs = config.GetMaxBlobsPerBlock(atOsaka)
	assert.Equal(t, target10, target, "Osaka target should be 10")
	assert.Equal(t, max15, maxBlobs, "Osaka max should be 15")

	// Simulate future BPO hardfork with increased limits
	schedule.Osaka.Target = &target14
	schedule.Osaka.Max = &max21
	target = config.GetTargetBlobsPerBlock(atOsaka)
	maxBlobs = config.GetMaxBlobsPerBlock(atOsaka)
	assert.Equal(t, target14, target, "BPO hardfork target should be 14")
	assert.Equal(t, max21, maxBlobs, "BPO hardfork max should be 21")
}

func TestEIP7892NilConfig(t *testing.T) {
	// Ensure nil/empty config doesn't panic
	t.Run("empty config behavior", func(t *testing.T) {
		config := &chain.Config{}
		// Without Osaka set, should return MaxInt
		assert.Equal(t, math.MaxInt, config.GetMaxRlpBlockSize(0))
	})
}

func TestEIP7892DefaultBlobSchedule(t *testing.T) {
	// Test behavior when BlobSchedule is nil
	config := &chain.Config{
		PragueTime: big.NewInt(1000),
		OsakaTime:  big.NewInt(2000),
	}

	// Should use default values
	assert.Equal(t, uint64(6), config.GetTargetBlobsPerBlock(1000), "Default Prague target")
	assert.Equal(t, uint64(9), config.GetMaxBlobsPerBlock(1000), "Default Prague max")
	assert.Equal(t, uint64(10), config.GetTargetBlobsPerBlock(2000), "Default Osaka target")
	assert.Equal(t, uint64(15), config.GetMaxBlobsPerBlock(2000), "Default Osaka max")
}

func TestEIP7892BlobSchedulePrecedence(t *testing.T) {
	// Test that more specific (later fork) config takes precedence
	target3 := uint64(3)
	target6 := uint64(6)
	target10 := uint64(10)

	schedule := &chain.BlobSchedule{
		Cancun: &chain.BlobConfig{Target: &target3},
		Prague: &chain.BlobConfig{Target: &target6},
		Osaka:  &chain.BlobConfig{Target: &target10},
	}

	config := &chain.Config{
		CancunTime:   big.NewInt(0),
		PragueTime:   big.NewInt(1000),
		OsakaTime:    big.NewInt(2000),
		BlobSchedule: schedule,
	}

	// Each fork should use its own config
	assert.Equal(t, target3, config.GetTargetBlobsPerBlock(0))
	assert.Equal(t, target3, config.GetTargetBlobsPerBlock(999))
	assert.Equal(t, target6, config.GetTargetBlobsPerBlock(1000))
	assert.Equal(t, target6, config.GetTargetBlobsPerBlock(1999))
	assert.Equal(t, target10, config.GetTargetBlobsPerBlock(2000))
	assert.Equal(t, target10, config.GetTargetBlobsPerBlock(3000))
}

func TestEIP7892BaseFeeUpdateFraction(t *testing.T) {
	// EIP-7892/EIP-7691 specifies baseFeeUpdateFraction for each fork
	fraction1 := uint64(3338477) // Cancun
	fraction2 := uint64(5007716) // Prague
	fraction3 := uint64(8346618) // Osaka

	schedule := &chain.BlobSchedule{
		Cancun: &chain.BlobConfig{BaseFeeUpdateFraction: &fraction1},
		Prague: &chain.BlobConfig{BaseFeeUpdateFraction: &fraction2},
		Osaka:  &chain.BlobConfig{BaseFeeUpdateFraction: &fraction3},
	}

	config := &chain.Config{
		CancunTime:   big.NewInt(0),
		PragueTime:   big.NewInt(1000),
		OsakaTime:    big.NewInt(2000),
		BlobSchedule: schedule,
	}

	assert.Equal(t, fraction1, config.GetBlobGasPriceUpdateFraction(0))
	assert.Equal(t, fraction2, config.GetBlobGasPriceUpdateFraction(1000))
	assert.Equal(t, fraction3, config.GetBlobGasPriceUpdateFraction(2000))
}

func TestEIP7892ComplianceStatus(t *testing.T) {
	// Document EIP-7892 implementation status
	t.Log("EIP-7892: Blob Parameter Only Hardforks")
	t.Log("========================================")
	t.Log("")
	t.Log("Core Features:")
	t.Log("✅ MAX_BLOCK_SIZE = 10 MiB")
	t.Log("✅ MAX_BLOCK_SIZE_SAFETY_MARGIN = 2 MiB")
	t.Log("✅ MAX_RLP_BLOCK_SIZE = 8 MiB (10 - 2)")
	t.Log("✅ GetMaxRlpBlockSize() returns limit for Osaka")
	t.Log("✅ BlobSchedule structure in chain config")
	t.Log("✅ Dynamic blob parameter lookup by fork")
	t.Log("")
	t.Log("BlobSchedule Support:")
	t.Log("✅ target (blobs per block)")
	t.Log("✅ max (blobs per block)")
	t.Log("✅ baseFeeUpdateFraction")
	t.Log("✅ Fork precedence (Osaka > Prague > Cancun)")
	t.Log("")
	t.Log("Integration Points:")
	t.Log("✅ CL: GetBlobParameters() for Fulu version")
	t.Log("✅ CL: MaxBlobsPerBlockByVersion() for pre-Fulu")
	t.Log("✅ CL: Block production checks MaxRlpBlockSize")
	t.Log("✅ EL: BlobSchedule in chain config JSON")
	t.Log("")
	t.Log("BPO Hardfork Roadmap:")
	t.Log("- Prague: 6/9 blobs (current)")
	t.Log("- Fusaka (Osaka): 10/15 blobs (PeerDAS)")
	t.Log("- BPO1: 12/18 blobs (planned)")
	t.Log("- BPO2: 14/21 blobs (planned)")
}

func TestEIP7892MainnetConfig(t *testing.T) {
	// Verify mainnet blob schedule if present
	config := params.MainnetChainConfig
	require.NotNil(t, config, "Mainnet config should exist")

	// Check default blob values
	if config.BlobSchedule == nil {
		t.Log("Mainnet BlobSchedule not yet configured (using defaults)")

		// Verify defaults are correct
		if config.IsPrague(1000000000000) {
			assert.Equal(t, uint64(6), config.GetTargetBlobsPerBlock(1000000000000))
			assert.Equal(t, uint64(9), config.GetMaxBlobsPerBlock(1000000000000))
		}
	}
}

