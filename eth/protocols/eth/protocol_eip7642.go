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
	"fmt"
	"io"
	"math/big"

	libcommon "github.com/erigontech/erigon-lib/common"
	rlp2 "github.com/erigontech/erigon-lib/rlp"

	"github.com/erigontech/erigon/core/forkid"
)

// EIP-7642: eth/69 - Simplified Receipts and Historical Window Declaration
//
// This file contains the core types and functions for EIP-7642 support:
// 1. StatusPacket69 - Extended status message with historical window fields
// 2. BlockRangeUpdatePacket - Dynamic block range update message
// 3. HistoricalWindow - Utility type for block range validation

const (
	// BlockRangeUpdateMsg is the message ID for BlockRangeUpdate (EIP-7642)
	// This is assigned after existing eth/69 messages
	BlockRangeUpdateMsg = 0x11 // 17 in decimal
)

// StatusPacket69 is the network packet for the eth/69 status message with EIP-7642 fields.
// It extends the base StatusPacket with historical window declaration.
type StatusPacket69 struct {
	ProtocolVersion uint32
	NetworkID       uint64
	TD              *big.Int
	Head            libcommon.Hash
	Genesis         libcommon.Hash
	ForkID          forkid.ID

	// EIP-7642 additions:
	EarliestBlock uint64 // Earliest block this node can serve
	LatestBlock   uint64 // Latest block this node can serve (usually same as head number)
}

// Name returns the human-readable name of the packet.
func (*StatusPacket69) Name() string { return "Status69" }

// Kind returns the message type identifier.
func (*StatusPacket69) Kind() byte { return StatusMsg }

// EncodeRLP encodes the status packet to RLP format.
func (p *StatusPacket69) EncodeRLP(w io.Writer) error {
	return rlp2.Encode(w, []interface{}{
		p.ProtocolVersion,
		p.NetworkID,
		p.TD,
		p.Head,
		p.Genesis,
		p.ForkID,
		p.EarliestBlock,
		p.LatestBlock,
	})
}

// DecodeRLP decodes the status packet from RLP format.
func (p *StatusPacket69) DecodeRLP(s *rlp2.Stream) error {
	var packet struct {
		ProtocolVersion uint32
		NetworkID       uint64
		TD              *big.Int
		Head            libcommon.Hash
		Genesis         libcommon.Hash
		ForkID          forkid.ID
		EarliestBlock   uint64
		LatestBlock     uint64
	}
	if err := s.Decode(&packet); err != nil {
		return err
	}
	p.ProtocolVersion = packet.ProtocolVersion
	p.NetworkID = packet.NetworkID
	p.TD = packet.TD
	p.Head = packet.Head
	p.Genesis = packet.Genesis
	p.ForkID = packet.ForkID
	p.EarliestBlock = packet.EarliestBlock
	p.LatestBlock = packet.LatestBlock
	return nil
}

// Validate validates the status packet fields.
func (p *StatusPacket69) Validate() error {
	if p.EarliestBlock > p.LatestBlock {
		return fmt.Errorf("invalid historical window: earliest %d > latest %d",
			p.EarliestBlock, p.LatestBlock)
	}
	return nil
}

// ToStatusPacket converts StatusPacket69 to the base StatusPacket.
// This is useful for backward compatibility with existing code.
func (p *StatusPacket69) ToStatusPacket() *StatusPacket {
	return &StatusPacket{
		ProtocolVersion: p.ProtocolVersion,
		NetworkID:       p.NetworkID,
		TD:              p.TD,
		Head:            p.Head,
		Genesis:         p.Genesis,
		ForkID:          p.ForkID,
	}
}

// FromStatusPacket creates a StatusPacket69 from a base StatusPacket.
// The historical window fields are set to default values (0, 0).
func FromStatusPacket(base *StatusPacket, earliestBlock, latestBlock uint64) *StatusPacket69 {
	return &StatusPacket69{
		ProtocolVersion: base.ProtocolVersion,
		NetworkID:       base.NetworkID,
		TD:              base.TD,
		Head:            base.Head,
		Genesis:         base.Genesis,
		ForkID:          base.ForkID,
		EarliestBlock:   earliestBlock,
		LatestBlock:     latestBlock,
	}
}

// BlockRangeUpdatePacket is the network packet for announcing block range updates (EIP-7642).
// Nodes send this message when their available historical data range changes.
type BlockRangeUpdatePacket struct {
	EarliestBlock uint64 // Updated earliest block the node can serve
	LatestBlock   uint64 // Updated latest block the node can serve
}

// Name returns the human-readable name of the packet.
func (*BlockRangeUpdatePacket) Name() string { return "BlockRangeUpdate" }

// Kind returns the message type identifier.
func (*BlockRangeUpdatePacket) Kind() byte { return BlockRangeUpdateMsg }

// EncodeRLP encodes the packet to RLP format.
func (p *BlockRangeUpdatePacket) EncodeRLP(w io.Writer) error {
	return rlp2.Encode(w, []interface{}{
		p.EarliestBlock,
		p.LatestBlock,
	})
}

// DecodeRLP decodes the packet from RLP format.
func (p *BlockRangeUpdatePacket) DecodeRLP(s *rlp2.Stream) error {
	var packet struct {
		EarliestBlock uint64
		LatestBlock   uint64
	}
	if err := s.Decode(&packet); err != nil {
		return err
	}
	p.EarliestBlock = packet.EarliestBlock
	p.LatestBlock = packet.LatestBlock
	return nil
}

// Validate validates the packet fields.
func (p *BlockRangeUpdatePacket) Validate() error {
	if p.EarliestBlock > p.LatestBlock {
		return fmt.Errorf("invalid block range: earliest %d > latest %d",
			p.EarliestBlock, p.LatestBlock)
	}
	return nil
}

// HistoricalWindow represents a node's available historical data range.
// This is used for peer selection and request routing.
type HistoricalWindow struct {
	EarliestBlock uint64
	LatestBlock   uint64
}

// NewHistoricalWindow creates a new HistoricalWindow.
func NewHistoricalWindow(earliest, latest uint64) *HistoricalWindow {
	return &HistoricalWindow{
		EarliestBlock: earliest,
		LatestBlock:   latest,
	}
}

// CanServe returns true if the window can serve the requested block range.
func (w *HistoricalWindow) CanServe(from, to uint64) bool {
	return from >= w.EarliestBlock && to <= w.LatestBlock
}

// CanServeBlock returns true if the window can serve a single block.
func (w *HistoricalWindow) CanServeBlock(blockNum uint64) bool {
	return blockNum >= w.EarliestBlock && blockNum <= w.LatestBlock
}

// Overlaps returns true if the window overlaps with the requested range.
func (w *HistoricalWindow) Overlaps(from, to uint64) bool {
	return from <= w.LatestBlock && to >= w.EarliestBlock
}

// Size returns the number of blocks in the window.
func (w *HistoricalWindow) Size() uint64 {
	if w.LatestBlock < w.EarliestBlock {
		return 0
	}
	return w.LatestBlock - w.EarliestBlock + 1
}

// Update updates the window with new values.
func (w *HistoricalWindow) Update(earliest, latest uint64) error {
	if earliest > latest {
		return fmt.Errorf("invalid window update: earliest %d > latest %d", earliest, latest)
	}
	w.EarliestBlock = earliest
	w.LatestBlock = latest
	return nil
}

// String returns a string representation of the window.
func (w *HistoricalWindow) String() string {
	return fmt.Sprintf("[%d, %d]", w.EarliestBlock, w.LatestBlock)
}

// PeerHistoricalInfo stores historical window information for a peer.
// This can be used by the peer manager for intelligent request routing.
type PeerHistoricalInfo struct {
	Window      *HistoricalWindow
	LastUpdated uint64 // Block number when last updated
}

// NewPeerHistoricalInfo creates a new PeerHistoricalInfo.
func NewPeerHistoricalInfo(earliest, latest, lastUpdated uint64) *PeerHistoricalInfo {
	return &PeerHistoricalInfo{
		Window:      NewHistoricalWindow(earliest, latest),
		LastUpdated: lastUpdated,
	}
}

// UpdateFromStatus updates the info from a StatusPacket69.
func (p *PeerHistoricalInfo) UpdateFromStatus(status *StatusPacket69) {
	p.Window.EarliestBlock = status.EarliestBlock
	p.Window.LatestBlock = status.LatestBlock
}

// UpdateFromBlockRange updates the info from a BlockRangeUpdatePacket.
func (p *PeerHistoricalInfo) UpdateFromBlockRange(update *BlockRangeUpdatePacket, currentBlock uint64) {
	p.Window.EarliestBlock = update.EarliestBlock
	p.Window.LatestBlock = update.LatestBlock
	p.LastUpdated = currentBlock
}

