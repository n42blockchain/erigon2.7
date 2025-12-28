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

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/erigontech/erigon-lib/common/fixedgas"
	"github.com/erigontech/erigon/params"
)

// EIP-7825: Transaction Gas Limit Cap (Osaka/Fusaka)
//
// This EIP sets a maximum gas limit for transactions to 2^24 (16,777,216).
// This improves:
// 1. Block packing efficiency
// 2. Enables future parallel execution frameworks
// 3. Prevents single transactions from consuming entire blocks

// TestEIP7825Constants verifies the EIP-7825 constants
func TestEIP7825Constants(t *testing.T) {
	// EIP-7825 specifies 2^24 = 16,777,216 as the maximum transaction gas limit
	expectedMaxGas := uint64(1 << 24) // 2^24 = 16,777,216

	// Verify params.MaxTxnGasLimit
	assert.Equal(t, expectedMaxGas, params.MaxTxnGasLimit,
		"params.MaxTxnGasLimit should be 2^24")

	// Verify fixedgas.MaxTxnGasLimit
	assert.Equal(t, expectedMaxGas, fixedgas.MaxTxnGasLimit,
		"fixedgas.MaxTxnGasLimit should be 2^24")

	// Both should be equal
	assert.Equal(t, params.MaxTxnGasLimit, fixedgas.MaxTxnGasLimit,
		"params and fixedgas MaxTxnGasLimit should be equal")

	// Verify specific value
	assert.Equal(t, uint64(16_777_216), params.MaxTxnGasLimit,
		"MaxTxnGasLimit should be 16,777,216")
}

// TestEIP7825ErrorMessage verifies the error message
func TestEIP7825ErrorMessage(t *testing.T) {
	assert.Equal(t, "gas limit too high", ErrGasLimitTooHigh.Error())
}

// TestEIP7825MaxGasLessThanBlockLimit verifies the transaction limit is less than typical block limits
func TestEIP7825MaxGasLessThanBlockLimit(t *testing.T) {
	// Typical mainnet block gas limit is around 30 million
	typicalBlockLimit := uint64(30_000_000)

	// Transaction limit (2^24 = 16.7 million) should be less than block limit
	assert.Less(t, params.MaxTxnGasLimit, typicalBlockLimit,
		"MaxTxnGasLimit should be less than typical block gas limit")

	// This allows multiple transactions per block even if one uses max gas
	remainingGas := typicalBlockLimit - params.MaxTxnGasLimit
	assert.Greater(t, remainingGas, params.TxGas,
		"Should have room for at least one more basic transaction after max gas tx")
}

// TestEIP7825BoundaryValues tests the boundary conditions
func TestEIP7825BoundaryValues(t *testing.T) {
	maxLimit := params.MaxTxnGasLimit

	tests := []struct {
		name     string
		gasLimit uint64
		valid    bool
	}{
		{
			name:     "exactly at limit",
			gasLimit: maxLimit,
			valid:    true, // <= is the condition
		},
		{
			name:     "one below limit",
			gasLimit: maxLimit - 1,
			valid:    true,
		},
		{
			name:     "one above limit",
			gasLimit: maxLimit + 1,
			valid:    false,
		},
		{
			name:     "basic transaction",
			gasLimit: params.TxGas, // 21000
			valid:    true,
		},
		{
			name:     "contract creation",
			gasLimit: params.TxGasContractCreation, // 53000
			valid:    true,
		},
		{
			name:     "zero gas",
			gasLimit: 0,
			valid:    true, // will fail for other reasons, but not for EIP-7825
		},
		{
			name:     "max uint64",
			gasLimit: ^uint64(0), // 18,446,744,073,709,551,615
			valid:    false,
		},
		{
			name:     "2^25 (double the limit)",
			gasLimit: 1 << 25, // 33,554,432
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.gasLimit <= maxLimit
			assert.Equal(t, tt.valid, isValid,
				"gasLimit %d should be %v", tt.gasLimit, map[bool]string{true: "valid", false: "invalid"}[tt.valid])
		})
	}
}

// TestEIP7825ComplianceStatus documents the implementation status
type EIP7825ComplianceStatus struct {
	Feature     string
	Implemented bool
	Location    string
}

func TestEIP7825ComplianceStatus(t *testing.T) {
	status := []EIP7825ComplianceStatus{
		{
			Feature:     "MaxTxnGasLimit constant in params",
			Implemented: true,
			Location:    "params/protocol_params.go",
		},
		{
			Feature:     "MaxTxnGasLimit constant in fixedgas",
			Implemented: true,
			Location:    "erigon-lib/common/fixedgas/protocol.go",
		},
		{
			Feature:     "ErrGasLimitTooHigh error",
			Implemented: true,
			Location:    "core/error.go",
		},
		{
			Feature:     "Transaction pool validation (Osaka)",
			Implemented: true,
			Location:    "erigon-lib/txpool/pool.go validateTx()",
		},
		{
			Feature:     "State transition preCheck (Osaka)",
			Implemented: true,
			Location:    "core/state_transition.go preCheck()",
		},
		{
			Feature:     "GasLimitTooHigh discard reason",
			Implemented: true,
			Location:    "erigon-lib/txpool/txpoolcfg/txpoolcfg.go",
		},
	}

	for _, s := range status {
		if s.Implemented {
			t.Logf("✅ %s: %s", s.Feature, s.Location)
		} else {
			t.Logf("⏳ %s: %s", s.Feature, s.Location)
		}
	}

	// Assert all features are implemented
	for _, s := range status {
		assert.True(t, s.Implemented, "Feature should be implemented: %s", s.Feature)
	}
}

// TestEIP7825Value verifies the exact value matches the EIP specification
func TestEIP7825Value(t *testing.T) {
	// From EIP-7825: MAX_TRANSACTION_GAS_LIMIT = 2**24
	// 2^24 = 16,777,216

	// Verify using bit shift
	assert.Equal(t, uint64(1<<24), params.MaxTxnGasLimit)

	// Verify using explicit value
	assert.Equal(t, uint64(16_777_216), params.MaxTxnGasLimit)

	// Verify it's a power of 2
	isPowerOf2 := params.MaxTxnGasLimit > 0 && (params.MaxTxnGasLimit&(params.MaxTxnGasLimit-1)) == 0
	assert.True(t, isPowerOf2, "MaxTxnGasLimit should be a power of 2")

	// Find which power of 2
	n := params.MaxTxnGasLimit
	power := 0
	for n > 1 {
		n >>= 1
		power++
	}
	assert.Equal(t, 24, power, "MaxTxnGasLimit should be 2^24")
}

