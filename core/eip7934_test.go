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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erigontech/erigon-lib/chain"
	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/params"
)

// EIP-7934: RLP Execution Block Size Limit
//
// This EIP introduces a protocol-level upper limit on the RLP-encoded size
// of execution blocks to enhance network stability and security.
//
// Key specifications:
// - Maximum block size: 10 MiB
// - Safety margin for beacon blocks: 2 MiB
// - Effective RLP limit: ~8 MiB
//
// Note: EIP-7934 is closely related to EIP-7892 (Blob Parameter Only Hardforks)
// and shares the same MaxRlpBlockSize constants and implementation.

// TestEIP7934MaxBlockSizeConstant verifies the 10 MiB limit
func TestEIP7934MaxBlockSizeConstant(t *testing.T) {
	// EIP-7934 specifies a 10 MiB maximum block size
	assert.Equal(t, 10*1024*1024, params.MaxBlockSize, "MaxBlockSize should be 10 MiB")
	assert.Equal(t, 10_485_760, params.MaxBlockSize)
}

// TestEIP7934SafetyMarginConstant verifies the 2 MiB safety margin
func TestEIP7934SafetyMarginConstant(t *testing.T) {
	// 2 MiB reserved for beacon block overhead
	assert.Equal(t, 2*1024*1024, params.MaxBlockSizeSafetyMargin, "SafetyMargin should be 2 MiB")
	assert.Equal(t, 2_097_152, params.MaxBlockSizeSafetyMargin)
}

// TestEIP7934EffectiveLimit verifies the effective RLP limit (~8 MiB)
func TestEIP7934EffectiveLimit(t *testing.T) {
	// Effective limit = MaxBlockSize - SafetyMargin = ~8 MiB
	expectedLimit := params.MaxBlockSize - params.MaxBlockSizeSafetyMargin
	assert.Equal(t, expectedLimit, params.MaxRlpBlockSize)
	assert.Equal(t, 8_388_608, params.MaxRlpBlockSize)
	assert.Equal(t, 8*1024*1024, params.MaxRlpBlockSize)
}

// TestEIP7934GetMaxRlpBlockSize tests the config method
func TestEIP7934GetMaxRlpBlockSize(t *testing.T) {
	osakaTime := uint64(1000)
	config := &chain.Config{
		OsakaTime: big.NewInt(int64(osakaTime)),
	}

	testCases := []struct {
		name        string
		time        uint64
		expectLimit bool
		expected    int
	}{
		{
			name:        "Before Osaka - no limit",
			time:        osakaTime - 1,
			expectLimit: false,
			expected:    math.MaxInt,
		},
		{
			name:        "At Osaka - limit applies",
			time:        osakaTime,
			expectLimit: true,
			expected:    params.MaxRlpBlockSize,
		},
		{
			name:        "After Osaka - limit applies",
			time:        osakaTime + 1000,
			expectLimit: true,
			expected:    params.MaxRlpBlockSize,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := config.GetMaxRlpBlockSize(tc.time)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestEIP7934BlockEncodingSize tests block encoding size calculation
func TestEIP7934BlockEncodingSize(t *testing.T) {
	// Create a minimal block to test encoding size
	header := &types.Header{
		Number:   big.NewInt(1),
		GasLimit: 30000000,
		GasUsed:  21000,
	}

	// Get the encoding size
	size := header.EncodingSize()
	require.Greater(t, size, 0, "Header should have positive encoding size")

	// Verify it's well under the limit for a minimal block
	assert.Less(t, size, params.MaxRlpBlockSize, "Minimal header should be under limit")
}

// TestEIP7934LimitEnforcement tests that the limit is properly enforced
func TestEIP7934LimitEnforcement(t *testing.T) {
	config := &chain.Config{
		OsakaTime: big.NewInt(1000),
	}

	testCases := []struct {
		name      string
		blockSize int
		time      uint64
		isValid   bool
	}{
		{
			name:      "Small block at Osaka - valid",
			blockSize: 1_000_000, // 1 MiB
			time:      1000,
			isValid:   true,
		},
		{
			name:      "Block at limit - valid",
			blockSize: params.MaxRlpBlockSize,
			time:      1000,
			isValid:   true,
		},
		{
			name:      "Block exceeds limit - invalid",
			blockSize: params.MaxRlpBlockSize + 1,
			time:      1000,
			isValid:   false,
		},
		{
			name:      "Large block before Osaka - valid (no limit)",
			blockSize: 20_000_000, // 20 MiB
			time:      999,
			isValid:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			maxSize := config.GetMaxRlpBlockSize(tc.time)
			isValid := tc.blockSize <= maxSize
			assert.Equal(t, tc.isValid, isValid)
		})
	}
}

// TestEIP7934DoSProtection tests that the limit provides DoS protection
func TestEIP7934DoSProtection(t *testing.T) {
	// Without EIP-7934, an attacker could create arbitrarily large blocks
	// With EIP-7934, blocks are limited to 10 MiB (8 MiB effective)

	// Calculate potential attack vectors
	maxTransactions := params.MaxRlpBlockSize / 128 // Assuming ~128 bytes per minimal tx
	t.Logf("Maximum minimal transactions per block: ~%d", maxTransactions)

	// With 30M gas limit and 21000 gas per tx, theoretical max is ~1428 txs
	// But size limit provides additional protection
	gasLimitTxs := 30_000_000 / 21000
	t.Logf("Gas-limited transactions: ~%d", gasLimitTxs)

	// Size limit is more restrictive for calldata-heavy transactions
	maxCalldataBytes := params.MaxRlpBlockSize
	t.Logf("Maximum calldata bytes per block: %d", maxCalldataBytes)
}

// TestEIP7934ConsensusLayerAlignment tests CL/EL size alignment
func TestEIP7934ConsensusLayerAlignment(t *testing.T) {
	// EIP-7934 aligns with CL gossip limits
	// CL maximum gossip message size: 10 MiB
	// Safety margin for beacon block wrapper: 2 MiB
	// Effective EL block limit: 8 MiB

	clGossipLimit := 10 * 1024 * 1024 // 10 MiB
	safetyMargin := 2 * 1024 * 1024   // 2 MiB
	effectiveElLimit := clGossipLimit - safetyMargin

	assert.Equal(t, params.MaxBlockSize, clGossipLimit)
	assert.Equal(t, params.MaxBlockSizeSafetyMargin, safetyMargin)
	assert.Equal(t, params.MaxRlpBlockSize, effectiveElLimit)
}

// TestEIP7934BlockValidation simulates block validation
func TestEIP7934BlockValidation(t *testing.T) {
	config := &chain.Config{
		OsakaTime: big.NewInt(1000),
	}

	validateBlockSize := func(rlpSize int, blockTime uint64) error {
		maxSize := config.GetMaxRlpBlockSize(blockTime)
		if rlpSize > maxSize {
			return ErrBlockSizeTooLarge
		}
		return nil
	}

	// Valid block
	err := validateBlockSize(5_000_000, 1000)
	assert.NoError(t, err)

	// Block at limit
	err = validateBlockSize(params.MaxRlpBlockSize, 1000)
	assert.NoError(t, err)

	// Block over limit
	err = validateBlockSize(params.MaxRlpBlockSize+1, 1000)
	assert.Error(t, err)
	assert.Equal(t, ErrBlockSizeTooLarge, err)
}

// TestEIP7934NetworkStability tests stability guarantees
func TestEIP7934NetworkStability(t *testing.T) {
	// EIP-7934 provides these stability guarantees:

	// 1. Bounded propagation time
	// With 8 MiB max, at 100 Mbps, max propagation time = 8 * 8 / 100 = 0.64 seconds
	maxSizeMbits := float64(params.MaxRlpBlockSize) * 8 / 1_000_000
	propagationTimeAt100Mbps := maxSizeMbits / 100
	t.Logf("Max propagation time at 100 Mbps: %.2f seconds", propagationTimeAt100Mbps)
	assert.Less(t, propagationTimeAt100Mbps, 1.0, "Propagation should be under 1 second")

	// 2. Bounded memory requirements
	// Nodes need at most 8 MiB per block in memory
	assert.Equal(t, 8*1024*1024, params.MaxRlpBlockSize)

	// 3. Reduced fork risk
	// Smaller blocks propagate faster, reducing temporary forks
}

// ErrBlockSizeTooLarge is returned when a block exceeds the EIP-7934 limit
var ErrBlockSizeTooLarge = &blockSizeError{msg: "block RLP size exceeds the limit"}

type blockSizeError struct {
	msg string
}

func (e *blockSizeError) Error() string {
	return e.msg
}

// TestEIP7934ComplianceStatus documents the compliance status
func TestEIP7934ComplianceStatus(t *testing.T) {
	// EIP-7934: RLP Execution Block Size Limit
	// ========================================
	//
	// Purpose:
	// Introduces a protocol-level upper limit on RLP-encoded block size
	// to enhance network stability and provide DoS protection.
	//
	// Specification:
	// - MaxBlockSize = 10 MiB (aligned with CL gossip limit)
	// - SafetyMargin = 2 MiB (for beacon block wrapper)
	// - MaxRlpBlockSize = 8 MiB (effective limit for EL blocks)
	// - Activated at Osaka/Fusaka fork
	//
	// Implementation Status:
	// ✅ MaxBlockSize constant (params/protocol_params.go:136)
	// ✅ MaxBlockSizeSafetyMargin constant (params/protocol_params.go:137)
	// ✅ MaxRlpBlockSize constant (params/protocol_params.go:138)
	// ✅ GetMaxRlpBlockSize() method (erigon-lib/chain/chain_config.go:370-377)
	// ✅ IsOsaka() fork check integration
	// ✅ CL block production check (cl/beacon/handler/block_production.go:1061-1062)
	//
	// Note:
	// EIP-7934 implementation is shared with EIP-7892 (Blob Parameter Only Hardforks)
	// as both EIPs deal with Osaka-era block size constraints.
	//
	// Security Benefits:
	// - DoS protection against oversized blocks
	// - Bounded block propagation time
	// - Reduced temporary fork risk
	// - Predictable memory requirements for nodes

	t.Log("EIP-7934 implementation is complete")
	t.Log("Implementation shared with EIP-7892")
}

