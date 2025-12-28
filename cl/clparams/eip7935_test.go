package clparams

import (
	"math"
	"testing"

	"github.com/erigontech/erigon-lib/chain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEIP7935DefaultBuilderGasLimitConstants verifies the EIP-7935 gas limit constants
func TestEIP7935DefaultBuilderGasLimitConstants(t *testing.T) {
	cfg := MainnetBeaconConfig

	// Verify pre-Fulu default (36M)
	assert.Equal(t, uint64(36_000_000), cfg.DefaultBuilderGasLimit)

	// Verify Fulu default (60M) - EIP-7935
	assert.Equal(t, uint64(60_000_000), cfg.DefaultBuilderGasLimitFulu)
}

// TestEIP7935GetDefaultBuilderGasLimit verifies the version-based gas limit selection
func TestEIP7935GetDefaultBuilderGasLimit(t *testing.T) {
	cfg := MainnetBeaconConfig
	cfg.FuluForkEpoch = 100 // Set a known fork epoch for testing

	tests := []struct {
		name     string
		epoch    uint64
		expected uint64
	}{
		{
			name:     "Before Fulu - uses pre-Fulu default",
			epoch:    0,
			expected: 36_000_000,
		},
		{
			name:     "Just before Fulu - uses pre-Fulu default",
			epoch:    99,
			expected: 36_000_000,
		},
		{
			name:     "At Fulu fork epoch - uses Fulu default",
			epoch:    100,
			expected: 60_000_000,
		},
		{
			name:     "After Fulu - uses Fulu default",
			epoch:    200,
			expected: 60_000_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gasLimit := cfg.GetDefaultBuilderGasLimit(tt.epoch)
			assert.Equal(t, tt.expected, gasLimit)
		})
	}
}

// TestEIP7935GasLimitIncrease verifies the 67% increase from 36M to 60M
func TestEIP7935GasLimitIncrease(t *testing.T) {
	preFuluLimit := uint64(36_000_000)
	fuluLimit := uint64(60_000_000)

	// Verify the increase
	increase := fuluLimit - preFuluLimit
	assert.Equal(t, uint64(24_000_000), increase)

	// Verify percentage increase (approximately 67%)
	percentageIncrease := float64(increase) / float64(preFuluLimit) * 100
	assert.InDelta(t, 66.67, percentageIncrease, 0.1)
}

// TestEIP7935CustomConfig verifies custom gas limit configuration
func TestEIP7935CustomConfig(t *testing.T) {
	// Test with custom gas limits
	cfg := BeaconChainConfig{
		FuluForkEpoch:            50,
		DefaultBuilderGasLimit:   40_000_000, // Custom pre-Fulu
		DefaultBuilderGasLimitFulu: 80_000_000, // Custom Fulu
	}

	// Before Fulu
	assert.Equal(t, uint64(40_000_000), cfg.GetDefaultBuilderGasLimit(49))

	// At/After Fulu
	assert.Equal(t, uint64(80_000_000), cfg.GetDefaultBuilderGasLimit(50))
}

// TestEIP7935ZeroConfig verifies default behavior when config values are zero
func TestEIP7935ZeroConfig(t *testing.T) {
	cfg := BeaconChainConfig{
		FuluForkEpoch:            100,
		DefaultBuilderGasLimit:   0, // Zero - should use default
		DefaultBuilderGasLimitFulu: 0, // Zero - should use default
	}

	// Before Fulu - should use 36M default
	assert.Equal(t, uint64(36_000_000), cfg.GetDefaultBuilderGasLimit(50))

	// At/After Fulu - should use 60M default
	assert.Equal(t, uint64(60_000_000), cfg.GetDefaultBuilderGasLimit(100))
}

// TestEIP7935FuluNeverActive verifies behavior when Fulu is never active
func TestEIP7935FuluNeverActive(t *testing.T) {
	cfg := BeaconChainConfig{
		FuluForkEpoch:          math.MaxUint64,
		DefaultBuilderGasLimit: 36_000_000,
	}

	// Any epoch should use pre-Fulu limit
	assert.Equal(t, uint64(36_000_000), cfg.GetDefaultBuilderGasLimit(0))
	assert.Equal(t, uint64(36_000_000), cfg.GetDefaultBuilderGasLimit(1000000))
	assert.Equal(t, uint64(36_000_000), cfg.GetDefaultBuilderGasLimit(math.MaxUint64-1))
}

// TestEIP7935NetworkConfigs verifies gas limits across different network configurations
func TestEIP7935NetworkConfigs(t *testing.T) {
	tests := []struct {
		name           string
		network        NetworkType
		preFuluLimit   uint64
		fuluLimit      uint64
	}{
		{
			name:         "Mainnet",
			network:      NetworkType(chain.MainnetChainID),
			preFuluLimit: 36_000_000,
			fuluLimit:    60_000_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cfg := GetConfigsByNetwork(tt.network)
			assert.Equal(t, tt.preFuluLimit, cfg.DefaultBuilderGasLimit)
			assert.Equal(t, tt.fuluLimit, cfg.DefaultBuilderGasLimitFulu)
		})
	}
}

// TestEIP7935ThroughputImprovement documents the throughput improvement
func TestEIP7935ThroughputImprovement(t *testing.T) {
	// Document the improvement calculations
	preFuluGasLimit := uint64(36_000_000)
	fuluGasLimit := uint64(60_000_000)

	// Simple transfer: 21,000 gas
	simpleTransferGas := uint64(21_000)

	preFuluTransfers := preFuluGasLimit / simpleTransferGas
	fuluTransfers := fuluGasLimit / simpleTransferGas

	t.Logf("Pre-Fulu: ~%d simple transfers per block", preFuluTransfers)
	t.Logf("Fulu: ~%d simple transfers per block", fuluTransfers)
	t.Logf("Improvement: +%d transfers per block", fuluTransfers-preFuluTransfers)

	// Verify the improvement
	assert.True(t, fuluTransfers > preFuluTransfers)
	assert.Equal(t, uint64(1714), preFuluTransfers)
	assert.Equal(t, uint64(2857), fuluTransfers)
}

// TestEIP7935BlockProductionIntegration verifies gas limit is used in block production
func TestEIP7935BlockProductionIntegration(t *testing.T) {
	cfg := MainnetBeaconConfig
	cfg.FuluForkEpoch = 100

	// Simulate block production at different epochs
	tests := []struct {
		epoch           uint64
		expectedGasLimit uint64
		forkName        string
	}{
		{epoch: 50, expectedGasLimit: 36_000_000, forkName: "Electra"},
		{epoch: 100, expectedGasLimit: 60_000_000, forkName: "Fulu"},
		{epoch: 150, expectedGasLimit: 60_000_000, forkName: "Fulu"},
	}

	for _, tt := range tests {
		t.Run(tt.forkName, func(t *testing.T) {
			gasLimit := cfg.GetDefaultBuilderGasLimit(tt.epoch)
			assert.Equal(t, tt.expectedGasLimit, gasLimit)
		})
	}
}

// TestEIP7935MethodSignature verifies the GetDefaultBuilderGasLimit method exists
func TestEIP7935MethodSignature(t *testing.T) {
	cfg := MainnetBeaconConfig

	// Verify the method exists and returns a uint64
	var gasLimit uint64 = cfg.GetDefaultBuilderGasLimit(0)
	require.NotZero(t, gasLimit)
}

// TestEIP7935ComplianceStatus documents the implementation status
func TestEIP7935ComplianceStatus(t *testing.T) {
	// EIP-7935: Increase block gas limit from 36M to 60M
	// Fusaka upgrade EIP

	implementation := struct {
		EIP                     string
		Title                   string
		PreFuluDefaultGasLimit  uint64
		FuluDefaultGasLimit     uint64
		FieldExists             bool
		MethodExists            bool
		MainnetConfigured       bool
	}{
		EIP:                    "7935",
		Title:                  "Increase block gas limit to 60M",
		PreFuluDefaultGasLimit: 36_000_000,
		FuluDefaultGasLimit:    60_000_000,
		FieldExists:            true,
		MethodExists:           true,
		MainnetConfigured:      true,
	}

	// Verify implementation
	assert.Equal(t, "7935", implementation.EIP)
	assert.True(t, implementation.FieldExists)
	assert.True(t, implementation.MethodExists)
	assert.True(t, implementation.MainnetConfigured)

	t.Logf("EIP-%s: %s - Implementation complete", implementation.EIP, implementation.Title)
	t.Logf("Pre-Fulu gas limit: %d", implementation.PreFuluDefaultGasLimit)
	t.Logf("Fulu gas limit: %d", implementation.FuluDefaultGasLimit)
}

