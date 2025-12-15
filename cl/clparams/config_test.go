/*
   Copyright 2022 Erigon-Lightclient contributors
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at
       http://www.apache.org/licenses/LICENSE-2.0
   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package clparams

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig(t *testing.T, n NetworkType) {
	network, beacon := GetConfigsByNetwork(n)

	require.Equal(t, *network, NetworkConfigs[n])
	require.Equal(t, *beacon, BeaconConfigs[n])
}

func TestGetConfigsByNetwork(t *testing.T) {
	testConfig(t, MainnetNetwork)
	testConfig(t, SepoliaNetwork)
	testConfig(t, GoerliNetwork)
	testConfig(t, GnosisNetwork)
	testConfig(t, ChiadoNetwork)
}

// TestStateVersionFulu tests Fulu version constants and methods
func TestStateVersionFulu(t *testing.T) {
	// Test FuluVersion constant
	assert.Equal(t, StateVersion(6), FuluVersion)

	// Test ClVersionToString
	assert.Equal(t, "fulu", ClVersionToString(FuluVersion))

	// Test StringToClVersion
	v, err := StringToClVersion("fulu")
	require.NoError(t, err)
	assert.Equal(t, FuluVersion, v)
}

// TestMainnetFuluForkConfig tests mainnet Fulu fork configuration
func TestMainnetFuluForkConfig(t *testing.T) {
	cfg := MainnetBeaconConfig

	// Fulu fork should be defined but not activated yet
	assert.NotZero(t, cfg.FuluForkVersion, "FuluForkVersion should be set")
	assert.Equal(t, ConfigForkVersion(0x06000000), cfg.FuluForkVersion)
	assert.Equal(t, uint64(math.MaxUint64), cfg.FuluForkEpoch, "FuluForkEpoch should be MaxUint64 (not activated)")

	// Electra should be activated
	assert.Equal(t, ConfigForkVersion(0x05000000), cfg.ElectraForkVersion)
	assert.Equal(t, uint64(364032), cfg.ElectraForkEpoch)
}

// TestGetForkVersionByVersion tests GetForkVersionByVersion for all versions including Fulu
func TestGetForkVersionByVersion(t *testing.T) {
	cfg := MainnetBeaconConfig

	tests := []struct {
		version     StateVersion
		expected    uint32
		description string
	}{
		{Phase0Version, uint32(cfg.GenesisForkVersion), "Phase0"},
		{AltairVersion, uint32(cfg.AltairForkVersion), "Altair"},
		{BellatrixVersion, uint32(cfg.BellatrixForkVersion), "Bellatrix"},
		{CapellaVersion, uint32(cfg.CapellaForkVersion), "Capella"},
		{DenebVersion, uint32(cfg.DenebForkVersion), "Deneb"},
		{ElectraVersion, uint32(cfg.ElectraForkVersion), "Electra"},
		{FuluVersion, uint32(cfg.FuluForkVersion), "Fulu"},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			result := cfg.GetForkVersionByVersion(tc.version)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestGetCurrentStateVersion tests GetCurrentStateVersion including Fulu transition
func TestGetCurrentStateVersion(t *testing.T) {
	cfg := MainnetBeaconConfig

	// Before Altair
	assert.Equal(t, Phase0Version, cfg.GetCurrentStateVersion(0))

	// At Altair epoch
	assert.Equal(t, AltairVersion, cfg.GetCurrentStateVersion(cfg.AltairForkEpoch))

	// At Bellatrix epoch
	assert.Equal(t, BellatrixVersion, cfg.GetCurrentStateVersion(cfg.BellatrixForkEpoch))

	// At Capella epoch
	assert.Equal(t, CapellaVersion, cfg.GetCurrentStateVersion(cfg.CapellaForkEpoch))

	// At Deneb epoch
	assert.Equal(t, DenebVersion, cfg.GetCurrentStateVersion(cfg.DenebForkEpoch))

	// At Electra epoch
	assert.Equal(t, ElectraVersion, cfg.GetCurrentStateVersion(cfg.ElectraForkEpoch))

	// For Fulu (when activated), test with Sepolia which has FuluForkEpoch set
	sepoliaCfg := BeaconConfigs[SepoliaNetwork]
	if sepoliaCfg.FuluForkEpoch < math.MaxUint64 {
		assert.Equal(t, FuluVersion, sepoliaCfg.GetCurrentStateVersion(sepoliaCfg.FuluForkEpoch))
	}
}
