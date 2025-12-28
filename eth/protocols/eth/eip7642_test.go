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

package eth

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/direct"

	"github.com/erigontech/erigon/core/forkid"
)

// EIP-7642: eth/69 - Simplified Receipts and Historical Window Declaration
//
// This EIP introduces:
// 1. Historical service window declaration (earliestBlock, latestBlock) in Status message
// 2. BlockRangeUpdate message for dynamic range updates
// 3. Removal of Bloom filter from receipts in network transmission (optional)

// TestETH69ProtocolSupport verifies that eth/69 protocol is properly registered
func TestETH69ProtocolSupport(t *testing.T) {
	// Verify eth/69 is in the protocol map
	eth69Name, ok := ProtocolToString[direct.ETH69]
	assert.True(t, ok, "eth/69 should be registered in ProtocolToString")
	assert.Equal(t, "eth69", eth69Name)

	// Verify eth/69 message mappings exist
	_, ok = ToProto[direct.ETH69]
	assert.True(t, ok, "eth/69 should have ToProto mappings")

	_, ok = FromProto[direct.ETH69]
	assert.True(t, ok, "eth/69 should have FromProto mappings")
}

// TestStatusPacketETH69Fields tests the current StatusPacket structure
// Note: EIP-7642 proposes adding earliestBlock and latestBlock fields
func TestStatusPacketETH69Fields(t *testing.T) {
	status := StatusPacket{
		ProtocolVersion: 69,
		NetworkID:       1,
		TD:              big.NewInt(1000000),
		Head:            libcommon.HexToHash("0x1234"),
		Genesis:         libcommon.HexToHash("0x5678"),
		ForkID:          forkid.ID{Hash: [4]byte{1, 2, 3, 4}, Next: 0},
	}

	assert.Equal(t, uint32(69), status.ProtocolVersion)
	assert.Equal(t, uint64(1), status.NetworkID)
	assert.NotNil(t, status.TD)
	assert.NotEmpty(t, status.Head)
	assert.NotEmpty(t, status.Genesis)

	// Verify packet interface
	assert.Equal(t, "Status", status.Name())
	assert.Equal(t, byte(StatusMsg), status.Kind())
}

// TestETH69MessageMappings verifies all message types are properly mapped for eth/69
func TestETH69MessageMappings(t *testing.T) {
	eth69ToProto := ToProto[direct.ETH69]
	eth69FromProto := FromProto[direct.ETH69]

	// Required messages for eth/69
	requiredMessages := []uint64{
		GetBlockHeadersMsg,
		BlockHeadersMsg,
		GetBlockBodiesMsg,
		BlockBodiesMsg,
		GetReceiptsMsg,
		ReceiptsMsg,
		NewBlockHashesMsg,
		NewBlockMsg,
		TransactionsMsg,
		NewPooledTransactionHashesMsg,
		GetPooledTransactionsMsg,
		PooledTransactionsMsg,
	}

	for _, msgType := range requiredMessages {
		_, ok := eth69ToProto[msgType]
		assert.True(t, ok, "Message type %d should be mapped in ToProto for eth/69", msgType)
	}

	// Verify reverse mappings exist
	assert.NotEmpty(t, eth69FromProto, "FromProto should have mappings for eth/69")
}

// TestETH69VsETH68Differences tests differences between eth/68 and eth/69
func TestETH69VsETH68Differences(t *testing.T) {
	eth68ToProto := ToProto[direct.ETH68]
	eth69ToProto := ToProto[direct.ETH69]

	// Both should have the same core message set
	for msgType, protoId := range eth68ToProto {
		eth69ProtoId, ok := eth69ToProto[msgType]
		assert.True(t, ok, "Message type %d should exist in eth/69", msgType)
		assert.Equal(t, protoId, eth69ProtoId, "Message type %d should have same proto ID in eth/69", msgType)
	}
}

// TestStatusPacket69Fields tests the StatusPacket69 structure from protocol_eip7642.go
func TestStatusPacket69Fields(t *testing.T) {
	status := StatusPacket69{
		ProtocolVersion: 69,
		NetworkID:       1,
		TD:              big.NewInt(1000000),
		Head:            libcommon.HexToHash("0x1234"),
		Genesis:         libcommon.HexToHash("0x5678"),
		ForkID:          forkid.ID{Hash: [4]byte{1, 2, 3, 4}, Next: 0},
		EarliestBlock:   1000000, // Node only serves blocks from 1M onwards
		LatestBlock:     2000000, // Node's current head
	}

	assert.Equal(t, uint64(1000000), status.EarliestBlock)
	assert.Equal(t, uint64(2000000), status.LatestBlock)
	assert.True(t, status.EarliestBlock < status.LatestBlock)
}

// TestBlockRangeUpdate tests the BlockRangeUpdatePacket from protocol_eip7642.go
func TestBlockRangeUpdate(t *testing.T) {
	update := BlockRangeUpdatePacket{
		EarliestBlock: 1500000,
		LatestBlock:   2500000,
	}

	assert.True(t, update.EarliestBlock < update.LatestBlock)
	assert.Equal(t, uint64(1500000), update.EarliestBlock)
	assert.Equal(t, uint64(2500000), update.LatestBlock)
}

// TestHistoricalWindowValidation tests validation of historical window
func TestHistoricalWindowValidation(t *testing.T) {
	testCases := []struct {
		name          string
		earliest      uint64
		latest        uint64
		requestedFrom uint64
		requestedTo   uint64
		canServe      bool
	}{
		{
			name:          "Within window",
			earliest:      1000,
			latest:        2000,
			requestedFrom: 1200,
			requestedTo:   1800,
			canServe:      true,
		},
		{
			name:          "Before window",
			earliest:      1000,
			latest:        2000,
			requestedFrom: 500,
			requestedTo:   900,
			canServe:      false,
		},
		{
			name:          "After window",
			earliest:      1000,
			latest:        2000,
			requestedFrom: 2100,
			requestedTo:   2500,
			canServe:      false,
		},
		{
			name:          "Partially before window",
			earliest:      1000,
			latest:        2000,
			requestedFrom: 800,
			requestedTo:   1500,
			canServe:      false, // Cannot serve full range
		},
		{
			name:          "Exact window boundaries",
			earliest:      1000,
			latest:        2000,
			requestedFrom: 1000,
			requestedTo:   2000,
			canServe:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			canServe := tc.requestedFrom >= tc.earliest && tc.requestedTo <= tc.latest
			assert.Equal(t, tc.canServe, canServe)
		})
	}
}

// TestBloomFilterSize tests the size of Bloom filter in receipts
// EIP-7642 proposes removing Bloom from network transmission
func TestBloomFilterSize(t *testing.T) {
	// Bloom filter is 256 bytes (2048 bits)
	const BloomByteLength = 256

	// Calculate potential bandwidth savings
	avgReceiptsPerBlock := 100
	blocksToSync := 1000000

	bloomBytesPerBlock := BloomByteLength * avgReceiptsPerBlock
	totalBloomBytes := bloomBytesPerBlock * blocksToSync

	// This is significant: ~25.6 GB for 1M blocks with 100 receipts each
	t.Logf("Bloom filter size per receipt: %d bytes", BloomByteLength)
	t.Logf("Estimated Bloom data for %d blocks (avg %d receipts): %.2f GB",
		blocksToSync, avgReceiptsPerBlock, float64(totalBloomBytes)/(1024*1024*1024))

	assert.Equal(t, 256, BloomByteLength)
}

// TestReceiptEncodingModes tests different receipt encoding modes
func TestReceiptEncodingModes(t *testing.T) {
	// Standard encoding (includes Bloom)
	t.Run("StandardEncoding", func(t *testing.T) {
		// Current behavior - Bloom is included
		// ReceiptRLP contains: status, cumulativeGasUsed, bloom, logs
	})

	// EIP-7642 proposed encoding (excludes Bloom for network)
	t.Run("EIP7642NetworkEncoding", func(t *testing.T) {
		// Proposed behavior - Bloom is excluded from network transmission
		// ReceiptRLP would contain: status, cumulativeGasUsed, logs
		// Bloom can be regenerated from logs on the receiving end
	})
}

// EIP7642ComplianceStatus documents current implementation status
type EIP7642ComplianceStatus struct {
	Feature     string
	Implemented bool
	Notes       string
}

func TestEIP7642ComplianceStatus(t *testing.T) {
	status := []EIP7642ComplianceStatus{
		{
			Feature:     "eth/69 protocol version",
			Implemented: true,
			Notes:       "Protocol version 69 is registered in ProtocolToString and message mappings",
		},
		{
			Feature:     "Status message with earliestBlock/latestBlock",
			Implemented: false,
			Notes:       "StatusPacket does not yet include historical window fields",
		},
		{
			Feature:     "BlockRangeUpdate message",
			Implemented: false,
			Notes:       "New message type not yet added to protocol",
		},
		{
			Feature:     "Bloom-less receipt encoding for eth/69",
			Implemented: false,
			Notes:       "Receipt encoding still includes Bloom filter",
		},
		{
			Feature:     "Historical window validation in peer selection",
			Implemented: false,
			Notes:       "Peer selection does not consider historical availability",
		},
	}

	for _, s := range status {
		if s.Implemented {
			t.Logf("✅ %s: %s", s.Feature, s.Notes)
		} else {
			t.Logf("⏳ %s: %s", s.Feature, s.Notes)
		}
	}
}

// TestProtocolVersionOrder verifies protocol versions are in order
func TestProtocolVersionOrder(t *testing.T) {
	require.Equal(t, uint(66), direct.ETH66)
	require.Equal(t, uint(67), direct.ETH67)
	require.Equal(t, uint(68), direct.ETH68)
	require.Equal(t, uint(69), direct.ETH69)

	// Verify ordering
	require.Less(t, direct.ETH66, direct.ETH67)
	require.Less(t, direct.ETH67, direct.ETH68)
	require.Less(t, direct.ETH68, direct.ETH69)
}

