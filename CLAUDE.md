# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a fork of Erigon 2.6 (module path: `github.com/erigontech/erigon`), upgraded with a new Consensus Layer (Caplin) and dependency updates, while **preserving the original Erigon 2.x data format** (DB schema v6.1, MDBX-backed). It is an Ethereum execution layer client with an embeddable consensus layer, operating as an Archive Node by default.

Go version: 1.24.6. CGO is required (MDBX, secp256k1, silkworm).

## Build Commands

```bash
# Build main erigon binary
make erigon

# Build all binaries (erigon + all sub-commands)
make all

# Build specific sub-command (e.g., rpcdaemon, caplin, integration, downloader)
make rpcdaemon
make caplin

# Debug build (with MDBX debug, no optimizations, compatible with delve)
make dbg

# Build MDBX CLI tools (mdbx_stat, mdbx_copy, etc.)
make db-tools

# Install dev tools (gencodec, mockgen, abigen, codecgen)
make devtools
```

Binaries output to `./build/bin/`.

## Testing

```bash
# Run all unit tests (includes erigon-lib tests first)
make test

# Run integration tests (240m timeout)
make test-integration

# Run a single package's tests
CGO_CFLAGS="-DMDBX_FORCE_ASSERTIONS=0 -O -D__BLST_PORTABLE__" go test -trimpath -tags nosqlite,noboltdb -count=1 ./turbo/jsonrpc/...

# Run a single test function
CGO_CFLAGS="-DMDBX_FORCE_ASSERTIONS=0 -O -D__BLST_PORTABLE__" go test -trimpath -tags nosqlite,noboltdb -count=1 -run TestFunctionName ./path/to/package/...

# Run erigon-lib tests separately
make test-erigon-lib

# Run hive tests (requires docker, act, GITHUB_TOKEN)
make test-hive
```

Build tags: `nosqlite,noboltdb` are always required. Add `integration` for integration tests, `e3` for E3-specific tests.

## Linting

```bash
make lint           # Full lint (golangci-lint + license check + mod tidy)
make lintci         # Just golangci-lint
make lint-deps      # Install lint dependencies
```

## Code Generation

```bash
make gen            # All codegen (mocks, solc, abigen, gencodec, codecgen, graphql, grpc)
make mocks          # Regenerate test mocks only
make grpc           # Regenerate gRPC from proto files (in erigon-lib)
```

## Architecture

### Two-Module Structure

- **Root module** (`github.com/erigontech/erigon`): Execution layer, consensus engines, RPC, p2p, sync logic
- **`erigon-lib/`** (`github.com/erigontech/erigon-lib`): Core library with database abstractions, gRPC interfaces, downloader, ETL framework, state management. Linked via `replace` directive in go.mod

### Entry Points (`cmd/`)

- `cmd/erigon/` - Main node binary (via `turbo/app`, `turbo/cli`, `turbo/node`)
- `cmd/rpcdaemon/` - Standalone JSON-RPC daemon (can connect to erigon via gRPC)
- `cmd/caplin/` - Standalone Caplin consensus layer client
- `cmd/sentry/` - P2P sentry service
- `cmd/downloader/` - Snapshot downloader (BitTorrent-based)
- `cmd/txpool/` - Transaction pool service
- `cmd/integration/` - Integration testing and state manipulation tool
- `cmd/sentinel/` - Beacon chain P2P sentinel

### Staged Sync (`eth/stagedsync/`)

Core sync mechanism. Blocks are processed through ordered stages, each with `Forward`, `Unwind`, and `Prune` functions:

Snapshots -> Headers -> BorHeimdall -> BlockHashes -> Bodies -> Senders -> Execution -> HashState -> IntermediateHashes -> AccountHistoryIndex -> StorageHistoryIndex -> LogIndex -> CallTraces -> TxLookup -> Finish

Stage orchestration: `eth/stagedsync/default_stages.go`
Stage IDs: `eth/stagedsync/stages/stages.go`
Stage loop: `turbo/stages/stageloop.go`

### Consensus Layer - Caplin (`cl/`)

Embedded beacon chain client (replaces external CL like Prysm/Lighthouse):
- `cl/phase1/` - Core beacon chain state transition
- `cl/clstages/` - CL staged sync stages (BeaconBlocks, BeaconState, BeaconIndexes)
- `cl/beacon/` - Beacon API and chain management
- `cl/sentinel/` - Beacon chain P2P networking
- `cl/antiquary/` - Historical beacon data management
- `cl/cltypes/` - Consensus layer types (BeaconBlock, BeaconState, etc.)
- `cl/fork/` - Fork choice logic
- `cl/transition/` - State transition functions
- `cl/pool/` - Attestation/sync committee pools

### Database Layer (`erigon-lib/kv/`)

Storage is MDBX-based (via `erigontech/mdbx-go`). Key abstractions:
- `kv/kv_interface.go` - Core `RoDB`, `RwDB`, `Tx`, `RwTx` interfaces
- `kv/mdbx/` - MDBX implementation
- `kv/memdb/` - In-memory DB (for testing)
- `kv/remotedb/` + `kv/remotedbserver/` - Remote DB access via gRPC
- `kv/temporal/` - Temporal (versioned) database layer
- `kv/tables.go` - All database table definitions and schema version
- `kv/rawdbv3/` - Raw database access helpers

Data format: Plain State storage (unhashed keys) for execution, Hashed State generated separately for Merkle root computation.

### RPC Layer (`turbo/jsonrpc/`)

JSON-RPC method implementations: `eth_*`, `debug_*`, `trace_*`, `erigon_*`, `bor_*`, etc. The RPC daemon can run:
1. Embedded within the erigon node
2. Standalone via `cmd/rpcdaemon/`, connecting to erigon's gRPC API

Engine API (`turbo/engineapi/`): Handles communication between execution and consensus layers via Engine API (newPayload, forkchoiceUpdated).

### gRPC Interfaces (`erigon-lib/gointerfaces/`)

Inter-process communication protocols: `remote/` (KV, ETH backend), `sentry/` (P2P), `sentinel/` (beacon P2P), `txpool/`, `downloader/`, `execution/`. Proto definitions come from `erigontech/interfaces` repo.

### Multi-Chain Support

- Ethereum mainnet + testnets (default)
- Polygon/Bor (`polygon/`, `consensus/aura/`, `eth/stagedsync/stage_bor_heimdall.go`)
- Gnosis Chain (`consensus/aura/`)

### Other Key Packages

- `core/vm/` - EVM implementation
- `core/state/` - State management (IntraBlockState, plain/hashed state)
- `core/types/` - Block, transaction, receipt, log types
- `consensus/` - Consensus engines (ethash, clique, aura, merge)
- `p2p/` - DevP2P networking stack
- `turbo/snapshotsync/` - Snapshot-based sync
- `erigon-lib/downloader/` - BitTorrent snapshot downloader
- `erigon-lib/state/` - State aggregation (for history)
- `erigon-lib/etl/` - Extract-Transform-Load framework for efficient DB writes

## Running

```bash
# Run erigon node (archive node by default)
./build/bin/erigon --datadir=<path>

# Run with embedded Caplin CL
./build/bin/erigon --datadir=<path> --internalcl

# Run standalone RPC daemon
./build/bin/rpcdaemon --private.api.addr=localhost:9090 --http.api=eth,debug,trace
```
