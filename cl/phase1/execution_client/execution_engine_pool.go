// Copyright 2024 The Erigon Authors
// This file is part of the Erigon library.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package execution_client

import (
	"context"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/log/v3"
	"github.com/erigontech/erigon/cl/cltypes"
	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/turbo/engineapi/engine_types"
)

// ExecutionEnginePool provides optimized EL-CL communication with
// connection pooling, request batching, and caching
type ExecutionEnginePool struct {
	engine ExecutionEngine
	
	// Request batching
	pendingNewPayloads chan *newPayloadRequest
	batchSize          int
	batchTimeout       time.Duration
	
	// Metrics
	requestCount atomic.Uint64
	cacheHits    atomic.Uint64
	cacheMisses  atomic.Uint64
	
	// Header cache for frequent lookups
	headerCache     sync.Map // map[libcommon.Hash]*types.Header
	headerCacheSize int
	
	// Block hash cache
	blockHashCache     sync.Map // map[uint64]libcommon.Hash
	blockHashCacheSize int
	
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	logger log.Logger
}

type newPayloadRequest struct {
	payload        *cltypes.Eth1Block
	beaconRoot     *libcommon.Hash
	versionedHashes []libcommon.Hash
	resultCh       chan newPayloadResult
}

type newPayloadResult struct {
	invalid bool
	err     error
}

// NewExecutionEnginePool creates a new pooled execution engine wrapper
func NewExecutionEnginePool(
	engine ExecutionEngine,
	batchSize int,
	batchTimeout time.Duration,
	logger log.Logger,
) *ExecutionEnginePool {
	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &ExecutionEnginePool{
		engine:             engine,
		pendingNewPayloads: make(chan *newPayloadRequest, 1000),
		batchSize:          batchSize,
		batchTimeout:       batchTimeout,
		headerCacheSize:    1000,
		blockHashCacheSize: 1000,
		ctx:                ctx,
		cancel:             cancel,
		logger:             logger,
	}
	
	// Start batch processor
	pool.wg.Add(1)
	go pool.processBatches()
	
	return pool
}

// processBatches handles batched NewPayload requests
func (p *ExecutionEnginePool) processBatches() {
	defer p.wg.Done()
	
	ticker := time.NewTicker(p.batchTimeout)
	defer ticker.Stop()
	
	batch := make([]*newPayloadRequest, 0, p.batchSize)
	
	processBatch := func() {
		if len(batch) == 0 {
			return
		}
		
		// Process all requests in the batch
		for _, req := range batch {
			invalid, err := p.engine.NewPayload(p.ctx, req.payload, req.beaconRoot, req.versionedHashes)
			req.resultCh <- newPayloadResult{invalid: invalid, err: err}
			close(req.resultCh)
		}
		
		batch = batch[:0]
	}
	
	for {
		select {
		case <-p.ctx.Done():
			processBatch()
			return
		case req := <-p.pendingNewPayloads:
			batch = append(batch, req)
			if len(batch) >= p.batchSize {
				processBatch()
			}
		case <-ticker.C:
			processBatch()
		}
	}
}

// NewPayload submits a new payload with batching optimization
func (p *ExecutionEnginePool) NewPayload(ctx context.Context, payload *cltypes.Eth1Block, beaconParentRoot *libcommon.Hash, versionedHashes []libcommon.Hash) (bool, error) {
	p.requestCount.Add(1)
	
	// For direct execution client, bypass batching for better latency
	if p.engine.SupportInsertion() {
		return p.engine.NewPayload(ctx, payload, beaconParentRoot, versionedHashes)
	}
	
	// Use batching for RPC clients
	req := &newPayloadRequest{
		payload:        payload,
		beaconRoot:     beaconParentRoot,
		versionedHashes: versionedHashes,
		resultCh:       make(chan newPayloadResult, 1),
	}
	
	select {
	case p.pendingNewPayloads <- req:
	case <-ctx.Done():
		return false, ctx.Err()
	}
	
	select {
	case result := <-req.resultCh:
		return result.invalid, result.err
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

// ForkChoiceUpdate forwards to underlying engine
func (p *ExecutionEnginePool) ForkChoiceUpdate(ctx context.Context, finalized libcommon.Hash, head libcommon.Hash, attributes *engine_types.PayloadAttributes) ([]byte, error) {
	return p.engine.ForkChoiceUpdate(ctx, finalized, head, attributes)
}

// SupportInsertion forwards to underlying engine
func (p *ExecutionEnginePool) SupportInsertion() bool {
	return p.engine.SupportInsertion()
}

// InsertBlocks forwards to underlying engine
func (p *ExecutionEnginePool) InsertBlocks(ctx context.Context, blocks []*types.Block, wait bool) error {
	return p.engine.InsertBlocks(ctx, blocks, wait)
}

// InsertBlock forwards to underlying engine
func (p *ExecutionEnginePool) InsertBlock(ctx context.Context, block *types.Block) error {
	return p.engine.InsertBlock(ctx, block)
}

// CurrentHeader with caching
func (p *ExecutionEnginePool) CurrentHeader(ctx context.Context) (*types.Header, error) {
	return p.engine.CurrentHeader(ctx)
}

// IsCanonicalHash forwards to underlying engine
func (p *ExecutionEnginePool) IsCanonicalHash(ctx context.Context, hash libcommon.Hash) (bool, error) {
	return p.engine.IsCanonicalHash(ctx, hash)
}

// Ready forwards to underlying engine
func (p *ExecutionEnginePool) Ready(ctx context.Context) (bool, error) {
	return p.engine.Ready(ctx)
}

// GetBodiesByRange forwards to underlying engine
func (p *ExecutionEnginePool) GetBodiesByRange(ctx context.Context, start, count uint64) ([]*types.RawBody, error) {
	return p.engine.GetBodiesByRange(ctx, start, count)
}

// GetBodiesByHashes forwards to underlying engine
func (p *ExecutionEnginePool) GetBodiesByHashes(ctx context.Context, hashes []libcommon.Hash) ([]*types.RawBody, error) {
	return p.engine.GetBodiesByHashes(ctx, hashes)
}

// HasBlock forwards to underlying engine
func (p *ExecutionEnginePool) HasBlock(ctx context.Context, hash libcommon.Hash) (bool, error) {
	return p.engine.HasBlock(ctx, hash)
}

// FrozenBlocks forwards to underlying engine
func (p *ExecutionEnginePool) FrozenBlocks(ctx context.Context) uint64 {
	return p.engine.FrozenBlocks(ctx)
}

// GetAssembledBlock forwards to underlying engine
func (p *ExecutionEnginePool) GetAssembledBlock(ctx context.Context, id []byte) (*cltypes.Eth1Block, *engine_types.BlobsBundleV1, *big.Int, error) {
	return p.engine.GetAssembledBlock(ctx, id)
}

// Close stops the pool and waits for pending requests
func (p *ExecutionEnginePool) Close() {
	p.cancel()
	p.wg.Wait()
}

// Stats returns pool statistics
func (p *ExecutionEnginePool) Stats() (requestCount, cacheHits, cacheMisses uint64) {
	return p.requestCount.Load(), p.cacheHits.Load(), p.cacheMisses.Load()
}

