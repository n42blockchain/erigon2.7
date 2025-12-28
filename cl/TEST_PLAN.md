# CL Test Plan and Status

## Overview

This document describes the current status of CL (Consensus Layer) tests and the plan for comprehensive testing after the v3.3.2 migration.

## Test Categories

### 1. Unit Tests (Package-Level)

#### ✅ Passing (Compilation and Runtime)
| Package | Test Count | Status |
|---------|------------|--------|
| `cl/utils` | 5 tests | ✅ PASS |
| `cl/utils/bls` | 7 tests | ✅ PASS |
| `cl/utils/eth2shuffle` | 2 tests | ✅ PASS |
| `cl/utils/eth_clock` | 3 tests | ✅ PASS |
| `cl/cltypes` | 15 tests | ✅ PASS |
| `cl/cltypes/solid` | 20+ tests | ✅ PASS |
| `cl/merkle_tree` | 5 tests | ✅ PASS |
| `cl/ssz` | 3 tests | ✅ PASS |
| `cl/fork` | 2 tests | ✅ PASS |
| `cl/pool` | 3 tests | ✅ PASS |
| `cl/clparams` | 2 tests | ✅ PASS |
| `cl/clparams/initial_state` | 1 test | ✅ PASS |
| `cl/persistence/base_encoding` | 6 tests | ✅ PASS |
| `cl/persistence/beacon_indicies` | 5 tests | ✅ PASS |
| `cl/persistence/blob_storage` | 3 tests | ✅ PASS |
| `cl/persistence/db_config` | 1 test | ✅ PASS |
| `cl/persistence/format/snapshot_format` | 2 tests | ✅ PASS |
| `cl/persistence/state` | 5 tests | ✅ PASS |
| `cl/persistence/state/historical_states_reader` | 3 tests | ✅ PASS |
| `cl/phase1/core/state` | 10 tests | ✅ PASS |
| `cl/phase1/core/state/raw` | 3 tests | ✅ PASS |
| `cl/phase1/core/state/shuffling` | 2 tests | ✅ PASS |
| `cl/phase1/forkchoice` | 2 tests | ✅ PASS |
| `cl/phase1/forkchoice/fork_graph` | 3 tests | ✅ PASS |
| `cl/phase1/forkchoice/fork_graph/diff_storage` | 2 tests | ✅ PASS |
| `cl/phase1/forkchoice/optimistic` | 1 test | ✅ PASS |
| `cl/phase1/network/services` | 10 tests | ✅ PASS |
| `cl/transition/impl/eth2` | 3 tests | ✅ PASS |
| `cl/transition/impl/eth2/statechange` | 5 tests | ✅ PASS |
| `cl/aggregation` | 2 tests | ✅ PASS |
| `cl/validator/attestation_producer` | 2 tests | ✅ PASS |
| `cl/validator/sync_contribution_pool` | 3 tests | ✅ PASS |
| `cl/beacon/beaconevents` | 2 tests | ✅ PASS |
| `cl/beacon/beacontest` | 1 test | ✅ PASS |
| `cl/beacon/builder` | 2 tests | ✅ PASS |
| `cl/sentinel/handlers` | 10 tests | ✅ PASS |

#### ⚠️ Compilation Fixed, Runtime Failures (PendingDeposits Table Missing)
| Package | Issue | Action Required |
|---------|-------|-----------------|
| `cl/sentinel` | Missing PendingDeposits table | Add table to kv/tables.go |
| `cl/beacon/handler` | Missing PendingDeposits table | Add table to kv/tables.go |
| `cl/antiquary` | Missing PendingDeposits table | Add table to kv/tables.go |

### 2. Spec Tests (Ethereum Consensus Spec)

Located in `cl/spectest/`:

#### Test Categories
- `bls` - BLS signature tests
- `epoch_processing` - Epoch transition tests
- `finality` - Finality tests
- `fork_choice` - Fork choice tests
- `forks` - Fork transition tests
- `light_client` - Light client tests
- `networking` - Networking tests
- `operations` - Block operation tests
- `rewards` - Reward calculation tests
- `sanity` - Sanity checks
- `shuffling` - Validator shuffling tests
- `ssz_static` - SSZ encoding tests
- `transition` - State transition tests

#### Running Spec Tests
```bash
# Download test data (run once)
cd cl/spectest && make tests

# Run all spec tests
make mainnet

# Run specific fork tests
make electra
make fulu
make deneb
make capella
```

### 3. Integration Tests

#### Lighthouse Interoperability
- Location: External (requires running Lighthouse node)
- Purpose: Cross-client validation
- Status: Not automated in CI

#### Erigon CL Self-Tests
- Location: `cl/phase1/` subdirectories
- Purpose: End-to-end CL functionality
- Status: Partially implemented

## Known Issues

### 1. Missing Database Tables
The following tables need to be added to `erigon-lib/kv/tables.go`:
- `PendingDeposits`
- `PendingDepositsDump`
- `PendingConsolidations`
- `PendingConsolidationsDump`
- `PendingPartialWithdrawals`
- `PendingPartialWithdrawalsDump`
- `ParentRootToBlockRoots`

These are already declared as string constants in `tables.go` but need to be:
1. Added to the appropriate table creation list
2. Registered for the `memdb` test database

### 2. API Compatibility
Some v3.3.2 APIs differ from the current codebase:
- `EthereumClock` interface (mock vs real)
- `PeerDasStateReader` interface
- `Sentinel` constructor signature
- `ExecutionEngine` interface methods

## Testing Recommendations

### Phase 1: Fix DB Schema (Priority: High)
1. Add missing tables to `kv/tables.go`
2. Ensure tables are created in `memdb.NewTestDB`
3. Re-run failing tests

### Phase 2: Enable Spec Tests (Priority: High)
1. Download consensus spec test data
2. Run spec tests for each fork version
3. Track test pass rates

### Phase 3: Add Missing Unit Tests (Priority: Medium)
Key areas needing more tests:
- EIP-7549 (Committee index in attestations)
- EIP-7594 (PeerDAS)
- Electra/Fulu state transitions
- Consolidation request handling

### Phase 4: Integration Testing (Priority: Medium)
1. Set up Lighthouse interoperability tests
2. Add beacon API endpoint tests
3. Test sync scenarios

## Running All CL Tests

```bash
# Run all unit tests (some will fail due to DB issues)
go test ./cl/...

# Run tests for specific package
go test -v ./cl/utils/...
go test -v ./cl/cltypes/...
go test -v ./cl/phase1/...

# Run with verbose output and race detection
go test -v -race ./cl/...

# Run specific test by name
go test -v -run TestName ./cl/package/...
```

## Test Coverage Tracking

To generate coverage report:
```bash
go test -coverprofile=coverage.out ./cl/...
go tool cover -html=coverage.out -o coverage.html
```

## Contributing Tests

When adding new tests:
1. Follow existing test patterns in the package
2. Use `memdb.NewTestDB(t)` for database tests
3. Use `clparams.NetworkType(chain.MainnetChainID)` for network IDs
4. Avoid using deprecated constructors (use struct literals)
5. Import `hex` package instead of `libcommon.Bytes2Hex`

