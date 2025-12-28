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

package block_collector_test

import (
	"context"
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/log/v3"
	"github.com/erigontech/erigon/cl/antiquary/tests"
	"github.com/erigontech/erigon/cl/clparams"
	"github.com/erigontech/erigon/cl/phase1/execution_client"
	"github.com/erigontech/erigon/cl/phase1/execution_client/block_collector"
	"github.com/erigontech/erigon/core/types"
)

func TestBlockCollectorAccumulateAndFlush(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	engine := execution_client.NewMockExecutionEngine(ctrl)
	blocks, _, _ := tests.GetBellatrixRandom()

	blocksLeft := make(map[uint64]struct{})
	for _, block := range blocks {
		blocksLeft[block.Block.Body.ExecutionPayload.BlockNumber] = struct{}{}
	}

	// Set up mock expectations for InsertBlocks
	engine.EXPECT().InsertBlocks(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes().DoAndReturn(func(ctx context.Context, blocks []*types.Block, wait bool) error {
		for _, block := range blocks {
			delete(blocksLeft, block.NumberU64())
		}
		return nil
	})

	// Set up mock expectations for CurrentHeader (called during Flush when batch is processed)
	engine.EXPECT().CurrentHeader(gomock.Any()).Return(&types.Header{
		Number: big.NewInt(0),
	}, nil).AnyTimes()

	// Set up mock expectations for ForkChoiceUpdate (may be called during Flush)
	engine.EXPECT().ForkChoiceUpdate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte{}, nil).AnyTimes()

	// Create temp directory for the collector
	tmpDir := t.TempDir()
	bc := block_collector.NewBlockCollector(log.Root(), engine, &clparams.MainnetBeaconConfig, math.MaxUint64, tmpDir)

	for _, block := range blocks {
		err := bc.AddBlock(block.Block)
		if err != nil {
			t.Fatal(err)
		}
	}
	require.NoError(t, bc.Flush(context.Background()))
	require.Empty(t, blocksLeft)
}

func TestBlockCollectorEmptyFlush(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	engine := execution_client.NewMockExecutionEngine(ctrl)
	tmpDir := t.TempDir()

	bc := block_collector.NewBlockCollector(log.Root(), engine, &clparams.MainnetBeaconConfig, math.MaxUint64, tmpDir)

	// Flush on empty collector should succeed without calling InsertBlocks
	require.NoError(t, bc.Flush(context.Background()))
}

func TestBlockCollectorAddBlock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	engine := execution_client.NewMockExecutionEngine(ctrl)
	blocks, _, _ := tests.GetBellatrixRandom()

	tmpDir := t.TempDir()
	bc := block_collector.NewBlockCollector(log.Root(), engine, &clparams.MainnetBeaconConfig, math.MaxUint64, tmpDir)

	// Add first block
	if len(blocks) > 0 {
		err := bc.AddBlock(blocks[0].Block)
		require.NoError(t, err)
	}
}

func TestBlockCollectorMultipleBatches(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	engine := execution_client.NewMockExecutionEngine(ctrl)
	blocks, _, _ := tests.GetBellatrixRandom()

	insertedBlocks := make(map[libcommon.Hash]struct{})

	engine.EXPECT().InsertBlocks(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes().DoAndReturn(func(ctx context.Context, blocks []*types.Block, wait bool) error {
		for _, block := range blocks {
			insertedBlocks[block.Hash()] = struct{}{}
		}
		return nil
	})

	engine.EXPECT().CurrentHeader(gomock.Any()).Return(&types.Header{
		Number: big.NewInt(0),
	}, nil).AnyTimes()

	engine.EXPECT().ForkChoiceUpdate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte{}, nil).AnyTimes()

	tmpDir := t.TempDir()
	bc := block_collector.NewBlockCollector(log.Root(), engine, &clparams.MainnetBeaconConfig, math.MaxUint64, tmpDir)

	// Add all blocks
	for _, block := range blocks {
		err := bc.AddBlock(block.Block)
		require.NoError(t, err)
	}

	require.NoError(t, bc.Flush(context.Background()))

	// Verify all blocks with execution payloads were processed
	// Note: Some blocks may have block number 0 which are skipped
	processedCount := 0
	for _, block := range blocks {
		if block.Block.Body.ExecutionPayload.BlockNumber > 0 {
			processedCount++
		}
	}
	require.Equal(t, processedCount, len(insertedBlocks))
}



























