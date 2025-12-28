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

package eth

import (
	"bytes"
	"math/big"
	"testing"

	libcommon "github.com/erigontech/erigon-lib/common"
	rlp2 "github.com/erigontech/erigon-lib/rlp"

	"github.com/erigontech/erigon/core/forkid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for StatusPacket69

func TestStatusPacket69_EncodeDecodeRLP(t *testing.T) {
	original := &StatusPacket69{
		ProtocolVersion: 69,
		NetworkID:       1,
		TD:              big.NewInt(1000000),
		Head:            libcommon.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
		Genesis:         libcommon.HexToHash("0xfedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321"),
		ForkID:          forkid.ID{Hash: [4]byte{1, 2, 3, 4}, Next: 1000},
		EarliestBlock:   1000000,
		LatestBlock:     2000000,
	}

	// Encode
	var buf bytes.Buffer
	err := original.EncodeRLP(&buf)
	require.NoError(t, err)

	// Decode
	decoded := &StatusPacket69{}
	stream := rlp2.NewStream(bytes.NewReader(buf.Bytes()), 0)
	err = decoded.DecodeRLP(stream)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, original.ProtocolVersion, decoded.ProtocolVersion)
	assert.Equal(t, original.NetworkID, decoded.NetworkID)
	assert.Equal(t, original.TD.Cmp(decoded.TD), 0)
	assert.Equal(t, original.Head, decoded.Head)
	assert.Equal(t, original.Genesis, decoded.Genesis)
	assert.Equal(t, original.ForkID, decoded.ForkID)
	assert.Equal(t, original.EarliestBlock, decoded.EarliestBlock)
	assert.Equal(t, original.LatestBlock, decoded.LatestBlock)
}

func TestStatusPacket69_Validate(t *testing.T) {
	tests := []struct {
		name      string
		earliest  uint64
		latest    uint64
		expectErr bool
	}{
		{
			name:      "valid window",
			earliest:  1000,
			latest:    2000,
			expectErr: false,
		},
		{
			name:      "same earliest and latest",
			earliest:  1000,
			latest:    1000,
			expectErr: false,
		},
		{
			name:      "earliest greater than latest",
			earliest:  2000,
			latest:    1000,
			expectErr: true,
		},
		{
			name:      "zero values",
			earliest:  0,
			latest:    0,
			expectErr: false,
		},
		{
			name:      "full node (genesis to head)",
			earliest:  0,
			latest:    20000000,
			expectErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status := &StatusPacket69{
				ProtocolVersion: 69,
				NetworkID:       1,
				TD:              big.NewInt(1000000),
				Head:            libcommon.Hash{},
				Genesis:         libcommon.Hash{},
				ForkID:          forkid.ID{},
				EarliestBlock:   tc.earliest,
				LatestBlock:     tc.latest,
			}
			err := status.Validate()
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStatusPacket69_Name(t *testing.T) {
	status := &StatusPacket69{}
	assert.Equal(t, "Status69", status.Name())
}

func TestStatusPacket69_Kind(t *testing.T) {
	status := &StatusPacket69{}
	assert.Equal(t, byte(StatusMsg), status.Kind())
}

func TestStatusPacket69_ToStatusPacket(t *testing.T) {
	status69 := &StatusPacket69{
		ProtocolVersion: 69,
		NetworkID:       1,
		TD:              big.NewInt(1000000),
		Head:            libcommon.HexToHash("0x1234"),
		Genesis:         libcommon.HexToHash("0x5678"),
		ForkID:          forkid.ID{Hash: [4]byte{1, 2, 3, 4}, Next: 0},
		EarliestBlock:   1000000,
		LatestBlock:     2000000,
	}

	base := status69.ToStatusPacket()
	assert.Equal(t, status69.ProtocolVersion, base.ProtocolVersion)
	assert.Equal(t, status69.NetworkID, base.NetworkID)
	assert.Equal(t, status69.TD, base.TD)
	assert.Equal(t, status69.Head, base.Head)
	assert.Equal(t, status69.Genesis, base.Genesis)
	assert.Equal(t, status69.ForkID, base.ForkID)
}

func TestFromStatusPacket(t *testing.T) {
	base := &StatusPacket{
		ProtocolVersion: 69,
		NetworkID:       1,
		TD:              big.NewInt(1000000),
		Head:            libcommon.HexToHash("0x1234"),
		Genesis:         libcommon.HexToHash("0x5678"),
		ForkID:          forkid.ID{Hash: [4]byte{1, 2, 3, 4}, Next: 0},
	}

	status69 := FromStatusPacket(base, 500000, 1500000)
	assert.Equal(t, base.ProtocolVersion, status69.ProtocolVersion)
	assert.Equal(t, base.NetworkID, status69.NetworkID)
	assert.Equal(t, base.TD, status69.TD)
	assert.Equal(t, base.Head, status69.Head)
	assert.Equal(t, base.Genesis, status69.Genesis)
	assert.Equal(t, base.ForkID, status69.ForkID)
	assert.Equal(t, uint64(500000), status69.EarliestBlock)
	assert.Equal(t, uint64(1500000), status69.LatestBlock)
}

// Tests for BlockRangeUpdatePacket

func TestBlockRangeUpdatePacket_EncodeDecodeRLP(t *testing.T) {
	original := &BlockRangeUpdatePacket{
		EarliestBlock: 1500000,
		LatestBlock:   2500000,
	}

	// Encode
	var buf bytes.Buffer
	err := original.EncodeRLP(&buf)
	require.NoError(t, err)

	// Decode
	decoded := &BlockRangeUpdatePacket{}
	stream := rlp2.NewStream(bytes.NewReader(buf.Bytes()), 0)
	err = decoded.DecodeRLP(stream)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, original.EarliestBlock, decoded.EarliestBlock)
	assert.Equal(t, original.LatestBlock, decoded.LatestBlock)
}

func TestBlockRangeUpdatePacket_Validate(t *testing.T) {
	tests := []struct {
		name      string
		earliest  uint64
		latest    uint64
		expectErr bool
	}{
		{
			name:      "valid update",
			earliest:  1000,
			latest:    2000,
			expectErr: false,
		},
		{
			name:      "invalid update - earliest > latest",
			earliest:  2000,
			latest:    1000,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			update := &BlockRangeUpdatePacket{
				EarliestBlock: tc.earliest,
				LatestBlock:   tc.latest,
			}
			err := update.Validate()
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBlockRangeUpdatePacket_Name(t *testing.T) {
	update := &BlockRangeUpdatePacket{}
	assert.Equal(t, "BlockRangeUpdate", update.Name())
}

func TestBlockRangeUpdatePacket_Kind(t *testing.T) {
	update := &BlockRangeUpdatePacket{}
	assert.Equal(t, byte(BlockRangeUpdateMsg), update.Kind())
}

// Tests for HistoricalWindow

func TestHistoricalWindow_CanServe(t *testing.T) {
	window := NewHistoricalWindow(1000, 2000)

	tests := []struct {
		name     string
		from     uint64
		to       uint64
		expected bool
	}{
		{"within window", 1200, 1800, true},
		{"before window", 500, 900, false},
		{"after window", 2100, 2500, false},
		{"exact boundaries", 1000, 2000, true},
		{"starts before", 800, 1500, false},
		{"ends after", 1500, 2500, false},
		{"single block within", 1500, 1500, true},
		{"single block at start", 1000, 1000, true},
		{"single block at end", 2000, 2000, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, window.CanServe(tc.from, tc.to))
		})
	}
}

func TestHistoricalWindow_CanServeBlock(t *testing.T) {
	window := NewHistoricalWindow(1000, 2000)

	tests := []struct {
		name     string
		blockNum uint64
		expected bool
	}{
		{"within window", 1500, true},
		{"at start", 1000, true},
		{"at end", 2000, true},
		{"before window", 999, false},
		{"after window", 2001, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, window.CanServeBlock(tc.blockNum))
		})
	}
}

func TestHistoricalWindow_Overlaps(t *testing.T) {
	window := NewHistoricalWindow(1000, 2000)

	tests := []struct {
		name     string
		from     uint64
		to       uint64
		expected bool
	}{
		{"fully within", 1200, 1800, true},
		{"before window", 500, 900, false},
		{"after window", 2100, 2500, false},
		{"overlaps start", 800, 1500, true},
		{"overlaps end", 1500, 2500, true},
		{"contains window", 500, 2500, true},
		{"touches start", 800, 1000, true},
		{"touches end", 2000, 2500, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, window.Overlaps(tc.from, tc.to))
		})
	}
}

func TestHistoricalWindow_Size(t *testing.T) {
	tests := []struct {
		name     string
		earliest uint64
		latest   uint64
		expected uint64
	}{
		{"normal window", 1000, 2000, 1001},
		{"single block", 1000, 1000, 1},
		{"zero to 10", 0, 10, 11},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			window := NewHistoricalWindow(tc.earliest, tc.latest)
			assert.Equal(t, tc.expected, window.Size())
		})
	}
}

func TestHistoricalWindow_Update(t *testing.T) {
	window := NewHistoricalWindow(1000, 2000)

	// Valid update
	err := window.Update(500, 2500)
	assert.NoError(t, err)
	assert.Equal(t, uint64(500), window.EarliestBlock)
	assert.Equal(t, uint64(2500), window.LatestBlock)

	// Invalid update
	err = window.Update(3000, 1000)
	assert.Error(t, err)
	// Values should remain unchanged
	assert.Equal(t, uint64(500), window.EarliestBlock)
	assert.Equal(t, uint64(2500), window.LatestBlock)
}

func TestHistoricalWindow_String(t *testing.T) {
	window := NewHistoricalWindow(1000, 2000)
	assert.Equal(t, "[1000, 2000]", window.String())
}

// Tests for PeerHistoricalInfo

func TestPeerHistoricalInfo_UpdateFromStatus(t *testing.T) {
	info := NewPeerHistoricalInfo(0, 0, 0)

	status := &StatusPacket69{
		EarliestBlock: 1000000,
		LatestBlock:   2000000,
	}

	info.UpdateFromStatus(status)
	assert.Equal(t, uint64(1000000), info.Window.EarliestBlock)
	assert.Equal(t, uint64(2000000), info.Window.LatestBlock)
}

func TestPeerHistoricalInfo_UpdateFromBlockRange(t *testing.T) {
	info := NewPeerHistoricalInfo(0, 0, 0)

	update := &BlockRangeUpdatePacket{
		EarliestBlock: 1500000,
		LatestBlock:   2500000,
	}

	info.UpdateFromBlockRange(update, 2500000)
	assert.Equal(t, uint64(1500000), info.Window.EarliestBlock)
	assert.Equal(t, uint64(2500000), info.Window.LatestBlock)
	assert.Equal(t, uint64(2500000), info.LastUpdated)
}

// Integration tests

func TestEIP7642Integration_StatusHandshake(t *testing.T) {
	// Simulate a status handshake between two nodes
	node1Status := &StatusPacket69{
		ProtocolVersion: 69,
		NetworkID:       1,
		TD:              big.NewInt(1000000),
		Head:            libcommon.HexToHash("0x1234"),
		Genesis:         libcommon.HexToHash("0x5678"),
		ForkID:          forkid.ID{Hash: [4]byte{1, 2, 3, 4}, Next: 0},
		EarliestBlock:   0,        // Full node
		LatestBlock:     20000000, // Head at 20M
	}

	node2Status := &StatusPacket69{
		ProtocolVersion: 69,
		NetworkID:       1,
		TD:              big.NewInt(900000),
		Head:            libcommon.HexToHash("0xabcd"),
		Genesis:         libcommon.HexToHash("0x5678"),
		ForkID:          forkid.ID{Hash: [4]byte{1, 2, 3, 4}, Next: 0},
		EarliestBlock:   19000000, // Light node, last 1M blocks
		LatestBlock:     20000000,
	}

	// Node 1 can serve all of node 2's historical requests
	assert.True(t, NewHistoricalWindow(node1Status.EarliestBlock, node1Status.LatestBlock).CanServe(19000000, 20000000))

	// Node 2 cannot serve node 1's full history requests
	assert.False(t, NewHistoricalWindow(node2Status.EarliestBlock, node2Status.LatestBlock).CanServe(0, 1000000))
}

func TestEIP7642Integration_BlockRangeUpdate(t *testing.T) {
	// Simulate a node updating its available range after pruning
	info := NewPeerHistoricalInfo(0, 20000000, 20000000)

	// Node prunes old data
	update := &BlockRangeUpdatePacket{
		EarliestBlock: 10000000, // Pruned first 10M blocks
		LatestBlock:   20100000, // New head
	}

	err := update.Validate()
	require.NoError(t, err)

	info.UpdateFromBlockRange(update, 20100000)

	// Verify the peer's window is updated
	assert.Equal(t, uint64(10000000), info.Window.EarliestBlock)
	assert.Equal(t, uint64(20100000), info.Window.LatestBlock)

	// Old blocks are no longer available
	assert.False(t, info.Window.CanServeBlock(5000000))
	// New blocks are available
	assert.True(t, info.Window.CanServeBlock(15000000))
}

func TestBlockRangeUpdateMsg_Constant(t *testing.T) {
	// Verify the message type constant
	assert.Equal(t, byte(0x11), byte(BlockRangeUpdateMsg))
}

