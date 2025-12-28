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

package clparams

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Version ordering tests (important for EIP version checking)
func TestStateVersionOrdering(t *testing.T) {
	versions := []StateVersion{
		Phase0Version,
		AltairVersion,
		BellatrixVersion,
		CapellaVersion,
		DenebVersion,
		ElectraVersion,
		FuluVersion,
	}

	// Test that versions are in ascending order
	for i := 1; i < len(versions); i++ {
		assert.Greater(t, versions[i], versions[i-1],
			"version %s should be greater than %s", versions[i], versions[i-1])
	}
}

func TestStateVersionString(t *testing.T) {
	tests := []struct {
		version  StateVersion
		expected string
	}{
		{Phase0Version, "phase0"},
		{AltairVersion, "altair"},
		{BellatrixVersion, "bellatrix"},
		{CapellaVersion, "capella"},
		{DenebVersion, "deneb"},
		{ElectraVersion, "electra"},
		{FuluVersion, "fulu"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.version.String())
		})
	}
}

func TestStateVersionComparison(t *testing.T) {
	// Test Before
	assert.True(t, DenebVersion.Before(ElectraVersion))
	assert.True(t, ElectraVersion.Before(FuluVersion))
	assert.False(t, FuluVersion.Before(DenebVersion))

	// Test After
	assert.True(t, FuluVersion.After(ElectraVersion))
	assert.True(t, ElectraVersion.After(DenebVersion))
	assert.False(t, DenebVersion.After(ElectraVersion))

	// Test Equal
	assert.True(t, ElectraVersion.Equal(ElectraVersion))
	assert.False(t, ElectraVersion.Equal(FuluVersion))

	// Test BeforeOrEqual
	assert.True(t, DenebVersion.BeforeOrEqual(DenebVersion))
	assert.True(t, DenebVersion.BeforeOrEqual(ElectraVersion))
	assert.False(t, FuluVersion.BeforeOrEqual(ElectraVersion))

	// Test AfterOrEqual
	assert.True(t, FuluVersion.AfterOrEqual(FuluVersion))
	assert.True(t, FuluVersion.AfterOrEqual(ElectraVersion))
	assert.False(t, DenebVersion.AfterOrEqual(ElectraVersion))
}

func TestStringToClVersion(t *testing.T) {
	tests := []struct {
		str      string
		expected StateVersion
		wantErr  bool
	}{
		{"phase0", Phase0Version, false},
		{"altair", AltairVersion, false},
		{"bellatrix", BellatrixVersion, false},
		{"capella", CapellaVersion, false},
		{"deneb", DenebVersion, false},
		{"electra", ElectraVersion, false},
		{"fulu", FuluVersion, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			version, err := StringToClVersion(tt.str)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, version)
			}
		})
	}
}

// Electra version specific checks (EIP-7549, EIP-7251, EIP-6110)
func TestElectraVersionCheck(t *testing.T) {
	// EIP-7549: Attestation Committee Bits starts from Electra
	assert.True(t, ElectraVersion.AfterOrEqual(ElectraVersion))
	assert.True(t, FuluVersion.AfterOrEqual(ElectraVersion))
	assert.False(t, DenebVersion.AfterOrEqual(ElectraVersion))

	// Check that Electra is version 5 (index)
	assert.Equal(t, StateVersion(5), ElectraVersion)
}

// Fulu version specific checks (EIP-7594 DAS)
func TestFuluVersionCheck(t *testing.T) {
	// PeerDAS starts from Fulu
	assert.True(t, FuluVersion.AfterOrEqual(FuluVersion))
	assert.False(t, ElectraVersion.AfterOrEqual(FuluVersion))
	assert.False(t, DenebVersion.AfterOrEqual(FuluVersion))

	// Check that Fulu is version 6 (index)
	assert.Equal(t, StateVersion(6), FuluVersion)
}

// Test version progression for upgrade path
func TestVersionUpgradePath(t *testing.T) {
	// Verify the correct upgrade path
	upgradePath := []StateVersion{
		Phase0Version,
		AltairVersion,
		BellatrixVersion,
		CapellaVersion,
		DenebVersion,
		ElectraVersion,
		FuluVersion,
	}

	for i := range upgradePath {
		// Each version should be exactly 1 greater than the previous
		assert.Equal(t, StateVersion(i), upgradePath[i])
	}
}

// Test version-specific feature detection
func TestVersionFeatureDetection(t *testing.T) {
	// Pre-Merge check (Bellatrix and after have execution payloads)
	assert.True(t, BellatrixVersion.AfterOrEqual(BellatrixVersion))
	assert.False(t, AltairVersion.AfterOrEqual(BellatrixVersion))

	// Withdrawals check (Capella and after)
	assert.True(t, CapellaVersion.AfterOrEqual(CapellaVersion))
	assert.False(t, BellatrixVersion.AfterOrEqual(CapellaVersion))

	// Blobs check (Deneb and after)
	assert.True(t, DenebVersion.AfterOrEqual(DenebVersion))
	assert.False(t, CapellaVersion.AfterOrEqual(DenebVersion))

	// MaxEB/EIP-7251 check (Electra and after)
	assert.True(t, ElectraVersion.AfterOrEqual(ElectraVersion))
	assert.False(t, DenebVersion.AfterOrEqual(ElectraVersion))

	// DAS/PeerDAS check (Fulu and after)
	assert.True(t, FuluVersion.AfterOrEqual(FuluVersion))
	assert.False(t, ElectraVersion.AfterOrEqual(FuluVersion))
}

