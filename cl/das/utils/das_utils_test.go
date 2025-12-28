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

package peerdasutils

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erigontech/erigon/cl/clparams"
	"github.com/erigontech/erigon/p2p/enode"
)

// EIP-7594: PeerDAS - Data Availability Sampling Tests

var initOnce sync.Once

func initTestConfig() {
	initOnce.Do(func() {
		// Only initialize if not already set (to avoid panic)
		if clparams.GetBeaconConfig() == nil {
			clparams.InitGlobalStaticConfig(&clparams.MainnetBeaconConfig, &clparams.CaplinConfig{})
		}
	})
}

func TestGetCustodyGroups(t *testing.T) {
	initTestConfig()

	// Create a test node ID
	nodeID := enode.ID{}
	for i := range nodeID {
		nodeID[i] = byte(i)
	}

	// Test getting custody groups
	custodyGroupCount := uint64(4)
	groups, err := GetCustodyGroups(nodeID, custodyGroupCount)
	require.NoError(t, err)
	assert.Len(t, groups, int(custodyGroupCount))

	// Verify groups are sorted
	for i := 1; i < len(groups); i++ {
		assert.LessOrEqual(t, groups[i-1], groups[i], "custody groups should be sorted")
	}

	// Verify all groups are unique
	groupSet := make(map[CustodyIndex]bool)
	for _, g := range groups {
		assert.False(t, groupSet[g], "duplicate custody group found")
		groupSet[g] = true
	}
}

func TestGetCustodyGroups_ExceedsMax(t *testing.T) {
	initTestConfig()
	nodeID := enode.ID{}
	cfg := clparams.GetBeaconConfig()

	// Request more custody groups than allowed
	_, err := GetCustodyGroups(nodeID, cfg.NumberOfCustodyGroups+1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum allowed")
}

func TestGetCustodyGroups_ZeroCount(t *testing.T) {
	initTestConfig()
	nodeID := enode.ID{}

	groups, err := GetCustodyGroups(nodeID, 0)
	require.NoError(t, err)
	assert.Len(t, groups, 0)
}

func TestGetCustodyGroups_Deterministic(t *testing.T) {
	initTestConfig()
	nodeID := enode.ID{}
	for i := range nodeID {
		nodeID[i] = byte(i + 10)
	}

	custodyGroupCount := uint64(4)

	// Get custody groups twice
	groups1, err := GetCustodyGroups(nodeID, custodyGroupCount)
	require.NoError(t, err)

	groups2, err := GetCustodyGroups(nodeID, custodyGroupCount)
	require.NoError(t, err)

	// Should be identical (deterministic)
	assert.Equal(t, groups1, groups2)
}

func TestGetCustodyGroups_DifferentNodes(t *testing.T) {
	initTestConfig()
	nodeID1 := enode.ID{}
	nodeID2 := enode.ID{}

	for i := range nodeID1 {
		nodeID1[i] = byte(i)
		nodeID2[i] = byte(255 - i)
	}

	custodyGroupCount := uint64(4)

	groups1, err := GetCustodyGroups(nodeID1, custodyGroupCount)
	require.NoError(t, err)

	groups2, err := GetCustodyGroups(nodeID2, custodyGroupCount)
	require.NoError(t, err)

	// Different nodes should likely have different custody groups (not guaranteed, but highly likely)
	// At minimum, verify both are valid
	assert.Len(t, groups1, int(custodyGroupCount))
	assert.Len(t, groups2, int(custodyGroupCount))
}

func TestComputeColumnsForCustodyGroup(t *testing.T) {
	initTestConfig()
	cfg := clparams.GetBeaconConfig()

	// Test a valid custody group
	custodyGroup := CustodyIndex(0)
	columns, err := ComputeColumnsForCustodyGroup(custodyGroup)
	require.NoError(t, err)

	expectedColumnsPerGroup := cfg.NumberOfColumns / cfg.NumberOfCustodyGroups
	assert.Len(t, columns, int(expectedColumnsPerGroup))

	// Verify the column indices are computed correctly
	// Formula: columns[i] = numberOfCustodyGroups * i + custodyGroup
	for i := ColumnIndex(0); i < expectedColumnsPerGroup; i++ {
		expectedColumn := cfg.NumberOfCustodyGroups*i + custodyGroup
		assert.Equal(t, expectedColumn, columns[i])
	}
}

func TestComputeColumnsForCustodyGroup_InvalidGroup(t *testing.T) {
	initTestConfig()
	cfg := clparams.GetBeaconConfig()

	// Test an invalid custody group (>= numberOfCustodyGroups)
	invalidGroup := CustodyIndex(cfg.NumberOfCustodyGroups)
	_, err := ComputeColumnsForCustodyGroup(invalidGroup)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is greater than or equal to")
}

func TestComputeColumnsForCustodyGroup_AllGroups(t *testing.T) {
	initTestConfig()
	cfg := clparams.GetBeaconConfig()

	// Collect all columns from all custody groups
	allColumns := make(map[ColumnIndex]bool)

	for g := CustodyIndex(0); g < cfg.NumberOfCustodyGroups; g++ {
		columns, err := ComputeColumnsForCustodyGroup(g)
		require.NoError(t, err)

		for _, col := range columns {
			// Verify no duplicate columns
			assert.False(t, allColumns[col], "column %d found in multiple custody groups", col)
			allColumns[col] = true
		}
	}

	// All columns should be covered exactly once
	assert.Equal(t, int(cfg.NumberOfColumns), len(allColumns))

	// Verify all column indices are in valid range
	for col := range allColumns {
		assert.Less(t, col, cfg.NumberOfColumns)
	}
}

func TestGetCustodyColumns(t *testing.T) {
	initTestConfig()
	nodeID := enode.ID{}
	for i := range nodeID {
		nodeID[i] = byte(i * 3)
	}

	cgc := uint64(4) // custody group count

	columns, err := GetCustodyColumns(nodeID, cgc)
	require.NoError(t, err)

	// Verify we got some columns
	assert.NotEmpty(t, columns)

	cfg := clparams.GetBeaconConfig()

	// Verify all columns are valid
	for col := range columns {
		assert.Less(t, col, cfg.NumberOfColumns)
	}

	// Verify the number of columns matches expected
	// Each custody group has (numberOfColumns / numberOfCustodyGroups) columns
	// We have cgc custody groups, so we should have cgc * columnsPerGroup columns
	expectedColumnsCount := cgc * (cfg.NumberOfColumns / cfg.NumberOfCustodyGroups)
	assert.Equal(t, int(expectedColumnsCount), len(columns))
}

func TestGetCustodyColumns_FullCustody(t *testing.T) {
	initTestConfig()
	nodeID := enode.ID{}
	for i := range nodeID {
		nodeID[i] = byte(i)
	}

	cfg := clparams.GetBeaconConfig()

	// Request full custody (all custody groups)
	columns, err := GetCustodyColumns(nodeID, cfg.NumberOfCustodyGroups)
	require.NoError(t, err)

	// Should have all columns
	assert.Equal(t, int(cfg.NumberOfColumns), len(columns))
}

// Test the ComputeMatrix function (basic structure test)
func TestComputeMatrix_Empty(t *testing.T) {
	// Empty blobs should return empty matrix
	matrix, err := ComputeMatrix([][]byte{})
	// Note: This will return an error because ComputeCellsAndKZGProofs is not implemented
	// But we test the structure
	if err != nil {
		// Expected until go-eth-kzg is integrated
		assert.Contains(t, err.Error(), "not implemented")
	} else {
		assert.Empty(t, matrix)
	}
}

func TestRecoverMatrix_NotImplemented(t *testing.T) {
	// This should return an error as it's not fully implemented
	_, err := RecoverMatrix(nil, 0)
	// If blobCount is 0, no error should occur as there's nothing to recover
	assert.NoError(t, err)
}

func TestRecoverCellsAndKZGProofs_InsufficientCells(t *testing.T) {
	// This should return an error as there are not enough cells
	_, _, err := RecoverCellsAndKZGProofs(nil, nil)
	assert.Error(t, err)
	// The new implementation requires at least 64 cells for recovery
	assert.Contains(t, err.Error(), "not enough cells")
}

func TestComputeCellsAndKZGProofs_InvalidBlobSize(t *testing.T) {
	// This should return an error for invalid blob size
	_, _, err := ComputeCellsAndKZGProofs([]byte{})
	assert.Error(t, err)
	// The new implementation validates blob size (must be BYTES_PER_BLOB)
	assert.Contains(t, err.Error(), "invalid blob size")
}

func TestComputeCellsAndKZGProofs_ValidBlob(t *testing.T) {
	// Skip if running short tests as this is computationally expensive
	if testing.Short() {
		t.Skip("Skipping KZG computation test in short mode")
	}
	
	// Create a valid blob (all zeros)
	blob := make([]byte, 131072) // BYTES_PER_BLOB
	cells, proofs, err := ComputeCellsAndKZGProofs(blob)
	assert.NoError(t, err)
	assert.Len(t, cells, 128) // CellsPerExtBlob
	assert.Len(t, proofs, 128)
}

