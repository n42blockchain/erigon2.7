// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erigontech/erigon-lib/chain"
	"github.com/erigontech/erigon-lib/common"
)

func TestCheckCompatible(t *testing.T) {
	type test struct {
		stored, new *chain.Config
		head        uint64
		wantErr     *chain.ConfigCompatError
	}
	tests := []test{
		{stored: AllProtocolChanges, new: AllProtocolChanges, head: 0, wantErr: nil},
		{stored: AllProtocolChanges, new: AllProtocolChanges, head: 100, wantErr: nil},
		{
			stored:  &chain.Config{TangerineWhistleBlock: big.NewInt(10)},
			new:     &chain.Config{TangerineWhistleBlock: big.NewInt(20)},
			head:    9,
			wantErr: nil,
		},
		{
			stored: AllProtocolChanges,
			new:    &chain.Config{HomesteadBlock: nil},
			head:   3,
			wantErr: &chain.ConfigCompatError{
				What:         "Homestead fork block",
				StoredConfig: big.NewInt(0),
				NewConfig:    nil,
				RewindTo:     0,
			},
		},
		{
			stored: AllProtocolChanges,
			new:    &chain.Config{HomesteadBlock: big.NewInt(1)},
			head:   3,
			wantErr: &chain.ConfigCompatError{
				What:         "Homestead fork block",
				StoredConfig: big.NewInt(0),
				NewConfig:    big.NewInt(1),
				RewindTo:     0,
			},
		},
		{
			stored: &chain.Config{HomesteadBlock: big.NewInt(30), TangerineWhistleBlock: big.NewInt(10)},
			new:    &chain.Config{HomesteadBlock: big.NewInt(25), TangerineWhistleBlock: big.NewInt(20)},
			head:   25,
			wantErr: &chain.ConfigCompatError{
				What:         "Tangerine Whistle fork block",
				StoredConfig: big.NewInt(10),
				NewConfig:    big.NewInt(20),
				RewindTo:     9,
			},
		},
		{
			stored:  &chain.Config{ConstantinopleBlock: big.NewInt(30)},
			new:     &chain.Config{ConstantinopleBlock: big.NewInt(30), PetersburgBlock: big.NewInt(30)},
			head:    40,
			wantErr: nil,
		},
		{
			stored: &chain.Config{ConstantinopleBlock: big.NewInt(30)},
			new:    &chain.Config{ConstantinopleBlock: big.NewInt(30), PetersburgBlock: big.NewInt(31)},
			head:   40,
			wantErr: &chain.ConfigCompatError{
				What:         "Petersburg fork block",
				StoredConfig: nil,
				NewConfig:    big.NewInt(31),
				RewindTo:     30,
			},
		},
	}

	for _, test := range tests {
		err := test.stored.CheckCompatible(test.new, test.head)
		if !reflect.DeepEqual(err, test.wantErr) {
			t.Errorf("error mismatch:\nstored: %v\nnew: %v\nhead: %v\nerr: %v\nwant: %v", test.stored, test.new, test.head, err, test.wantErr)
		}
	}
}

func TestGetBurntContract(t *testing.T) {
	// Ethereum
	assert.Nil(t, MainnetChainConfig.GetBurntContract(0))
	assert.Nil(t, MainnetChainConfig.GetBurntContract(10_000_000))

	// Gnosis Chain
	addr := GnosisChainConfig.GetBurntContract(19_040_000)
	require.NotNil(t, addr)
	assert.Equal(t, common.HexToAddress("0x6BBe78ee9e474842Dbd4AB4987b3CeFE88426A92"), *addr)
	addr = GnosisChainConfig.GetBurntContract(19_040_001)
	require.NotNil(t, addr)
	assert.Equal(t, common.HexToAddress("0x6BBe78ee9e474842Dbd4AB4987b3CeFE88426A92"), *addr)

	// Mumbai
	addr = MumbaiChainConfig.GetBurntContract(22640000)
	require.NotNil(t, addr)
	assert.Equal(t, common.HexToAddress("0x70bcA57F4579f58670aB2d18Ef16e02C17553C38"), *addr)
	addr = MumbaiChainConfig.GetBurntContract(22640000 + 1)
	require.NotNil(t, addr)
	assert.Equal(t, common.HexToAddress("0x70bcA57F4579f58670aB2d18Ef16e02C17553C38"), *addr)
	addr = MumbaiChainConfig.GetBurntContract(41874000 - 1)
	require.NotNil(t, addr)
	assert.Equal(t, common.HexToAddress("0x70bcA57F4579f58670aB2d18Ef16e02C17553C38"), *addr)
	addr = MumbaiChainConfig.GetBurntContract(41874000)
	require.NotNil(t, addr)
	assert.Equal(t, common.HexToAddress("0x617b94CCCC2511808A3C9478ebb96f455CF167aA"), *addr)
	addr = MumbaiChainConfig.GetBurntContract(41874000 + 1)
	require.NotNil(t, addr)
	assert.Equal(t, common.HexToAddress("0x617b94CCCC2511808A3C9478ebb96f455CF167aA"), *addr)

	// Amoy
	addr = AmoyChainConfig.GetBurntContract(0)
	require.NotNil(t, addr)
	assert.Equal(t, common.HexToAddress("0x000000000000000000000000000000000000dead"), *addr)
}

func TestMainnetBlobSchedule(t *testing.T) {
	// Original EIP-4844 values
	assert.Equal(t, uint64(6), MainnetChainConfig.GetMaxBlobsPerBlock(0))
	assert.Equal(t, uint64(786432), MainnetChainConfig.GetMaxBlobGasPerBlock(0))
	assert.Equal(t, uint64(393216), MainnetChainConfig.GetTargetBlobGasPerBlock(0))
	assert.Equal(t, uint64(3338477), MainnetChainConfig.GetBlobGasPriceUpdateFraction(0))

	b := MainnetChainConfig.BlobSchedule
	isPrague := false
	isOsaka := false
	assert.Equal(t, uint64(3), b.TargetBlobsPerBlock(isPrague, isOsaka))
	assert.Equal(t, uint64(6), b.MaxBlobsPerBlock(isPrague, isOsaka))
	assert.Equal(t, uint64(3338477), b.BaseFeeUpdateFraction(isPrague, isOsaka))

	// EIP-7691: Blob throughput increase
	isPrague = true
	assert.Equal(t, uint64(6), b.TargetBlobsPerBlock(isPrague, isOsaka))
	assert.Equal(t, uint64(9), b.MaxBlobsPerBlock(isPrague, isOsaka))
	assert.Equal(t, uint64(5007716), b.BaseFeeUpdateFraction(isPrague, isOsaka))

	// Fusaka/Osaka: PeerDAS blob throughput increase
	isOsaka = true
	assert.Equal(t, uint64(10), b.TargetBlobsPerBlock(isPrague, isOsaka))
	assert.Equal(t, uint64(15), b.MaxBlobsPerBlock(isPrague, isOsaka))
	assert.Equal(t, uint64(8346618), b.BaseFeeUpdateFraction(isPrague, isOsaka))
}

func TestGnosisBlobSchedule(t *testing.T) {
	b := GnosisChainConfig.BlobSchedule

	// Cancun values
	isPrague := false
	isOsaka := false
	assert.Equal(t, uint64(1), b.TargetBlobsPerBlock(isPrague, isOsaka))
	assert.Equal(t, uint64(2), b.MaxBlobsPerBlock(isPrague, isOsaka))
	assert.Equal(t, uint64(1112826), b.BaseFeeUpdateFraction(isPrague, isOsaka))

	// should remain the same in Pectra for Gnosis
	isPrague = true
	assert.Equal(t, uint64(1), b.TargetBlobsPerBlock(isPrague, isOsaka))
	assert.Equal(t, uint64(2), b.MaxBlobsPerBlock(isPrague, isOsaka))
	assert.Equal(t, uint64(1112826), b.BaseFeeUpdateFraction(isPrague, isOsaka))
}

// EIP-7691: Blob throughput increase
// Tests the blob schedule constants and gas calculations

func TestEIP7691BlobGasCalculations(t *testing.T) {
	// EIP-7691 specifies:
	// Cancun: target=3, max=6, BASE_FEE_UPDATE_FRACTION=3338477
	// Prague: target=6, max=9, BASE_FEE_UPDATE_FRACTION=5007716
	// Osaka:  target=10, max=15, BASE_FEE_UPDATE_FRACTION=8346618

	// Blob gas per blob = 131072 (BlobGasPerBlob)
	const blobGasPerBlob = uint64(131072)

	// Test Cancun (pre-Prague)
	maxBlobsCancun := MainnetChainConfig.GetMaxBlobsPerBlock(0)
	assert.Equal(t, uint64(6), maxBlobsCancun)
	assert.Equal(t, uint64(6)*blobGasPerBlob, MainnetChainConfig.GetMaxBlobGasPerBlock(0))

	// Test target blob gas
	targetBlobsCancun := MainnetChainConfig.GetTargetBlobsPerBlock(0)
	assert.Equal(t, uint64(3), targetBlobsCancun)
	assert.Equal(t, uint64(3)*blobGasPerBlob, MainnetChainConfig.GetTargetBlobGasPerBlock(0))
}

func TestEIP7691PragueValues(t *testing.T) {
	// Create a config with Prague enabled
	pragueTime := big.NewInt(1000)
	config := &chain.Config{
		PragueTime: pragueTime,
	}

	// Before Prague
	assert.Equal(t, uint64(6), config.GetMaxBlobsPerBlock(999))
	assert.Equal(t, uint64(3), config.GetTargetBlobsPerBlock(999))
	assert.Equal(t, uint64(3338477), config.GetBlobGasPriceUpdateFraction(999))

	// After Prague - EIP-7691 values
	assert.Equal(t, uint64(9), config.GetMaxBlobsPerBlock(1000))
	assert.Equal(t, uint64(6), config.GetTargetBlobsPerBlock(1000))
	assert.Equal(t, uint64(5007716), config.GetBlobGasPriceUpdateFraction(1000))
}

func TestEIP7691OsakaValues(t *testing.T) {
	// Create a config with Osaka enabled
	pragueTime := big.NewInt(1000)
	osakaTime := big.NewInt(2000)
	config := &chain.Config{
		PragueTime: pragueTime,
		OsakaTime:  osakaTime,
	}

	// Before Osaka (in Prague)
	assert.Equal(t, uint64(9), config.GetMaxBlobsPerBlock(1500))
	assert.Equal(t, uint64(6), config.GetTargetBlobsPerBlock(1500))

	// After Osaka - increased blob throughput
	assert.Equal(t, uint64(15), config.GetMaxBlobsPerBlock(2000))
	assert.Equal(t, uint64(10), config.GetTargetBlobsPerBlock(2000))
	assert.Equal(t, uint64(8346618), config.GetBlobGasPriceUpdateFraction(2000))
}

func TestEIP7691CustomBlobSchedule(t *testing.T) {
	// Test custom blob schedule override
	customTarget := uint64(4)
	customMax := uint64(8)
	customFraction := uint64(4000000)

	config := &chain.Config{
		BlobSchedule: &chain.BlobSchedule{
			Cancun: &chain.BlobConfig{
				Target:                &customTarget,
				Max:                   &customMax,
				BaseFeeUpdateFraction: &customFraction,
			},
		},
	}

	assert.Equal(t, customMax, config.GetMaxBlobsPerBlock(0))
	assert.Equal(t, customTarget, config.GetTargetBlobsPerBlock(0))
	assert.Equal(t, customFraction, config.GetBlobGasPriceUpdateFraction(0))
}

func TestEIP7691NilBlobSchedule(t *testing.T) {
	// Test with nil blob schedule - should use defaults
	config := &chain.Config{
		BlobSchedule: nil,
	}

	// Should return EIP-4844 defaults
	assert.Equal(t, uint64(6), config.GetMaxBlobsPerBlock(0))
	assert.Equal(t, uint64(3), config.GetTargetBlobsPerBlock(0))
	assert.Equal(t, uint64(3338477), config.GetBlobGasPriceUpdateFraction(0))
}

func TestEIP7691MaxBlobGasConsistency(t *testing.T) {
	const blobGasPerBlob = uint64(131072)

	// Verify that max blob gas = max blobs * blob gas per blob
	// Cancun
	assert.Equal(t, uint64(6)*blobGasPerBlob, MainnetChainConfig.GetMaxBlobGasPerBlock(0))

	// Prague (EIP-7691)
	pragueTime := big.NewInt(1000)
	config := &chain.Config{
		PragueTime: pragueTime,
	}
	assert.Equal(t, uint64(9)*blobGasPerBlob, config.GetMaxBlobGasPerBlock(1000))

	// Osaka
	osakaTime := big.NewInt(2000)
	config.OsakaTime = osakaTime
	assert.Equal(t, uint64(15)*blobGasPerBlob, config.GetMaxBlobGasPerBlock(2000))
}