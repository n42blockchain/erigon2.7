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

| `cl/beacon/handler` | 25+ tests | ✅ PASS |
| `cl/phase1/core/checkpoint_sync` | 3 tests | ✅ PASS |

#### ✅ EIP-Specific Tests (New)
| Package | EIP | Test Count | Status |
|---------|-----|------------|--------|
| `cl/cltypes/solid` | EIP-7549 (Electra) | 8+ tests | ✅ PASS |
| `cl/das/utils` | EIP-7594 (DAS/Fulu) | 14 tests | ✅ PASS |
| `cl/clparams` | Version handling | 8 tests | ✅ PASS |
| `cl/phase1/execution_client/block_collector` | Block collector | 4 tests | ✅ PASS |

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

### 1. Database Tables (FIXED)
The following tables have been added to `ChaindataTables` in `erigon-lib/kv/tables.go`:
- ✅ `PendingDeposits`
- ✅ `PendingDepositsDump`
- ✅ `PendingConsolidations`
- ✅ `PendingConsolidationsDump`
- ✅ `PendingPartialWithdrawals`
- ✅ `PendingPartialWithdrawalsDump`

### 2. Pre-existing Test Issues ✅ FIXED
~~The following test has a pre-existing issue unrelated to v3.3.2 migration:~~
- `cl/phase1/execution_client/block_collector/block_collector_test.go`: 
  - ✅ FIXED: Changed `return next(k, nil, nil)` to `return nil` in ETL callback
  - The ETL collector was calling `next()` with nil values when no DB bucket was provided

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

### Phase 3: Add Missing Unit Tests (Priority: Medium) ✅ DONE
EIP-specific tests have been added:
- ✅ EIP-7549 (Electra Attestation Committee Bits) - COMPREHENSIVE
  - `TestElectraAttestation` - Basic Electra attestation with CommitteeBits
  - `TestGetCommitteeIndexFromBits_NoCommitteeBitsSet` - Error case
  - `TestGetCommitteeIndexFromBits_MultipleCommitteeBits` - Multiple bits handling
  - `TestSingleAttestation` - SSZ encode/decode, HashSSZ, Clone
  - `TestSingleAttestationToAttestation` - Conversion with member index
  - `TestElectraAttestationEncodingSizeSSZ` - Size calculation
  - `TestAttestationJSONUnmarshalElectra/Deneb` - JSON parsing
  - `TestElectraSingleAttestationCommitteeIndexMustBeZeroInData` - EIP-7549 validation
  - `TestElectraSingleAttestationValidCommitteeIndex` - Network service validation
  - `TestElectraSingleAttestationWithData` - ToAttestation conversion
- ✅ EIP-7594 (PeerDAS/Fulu)
  - `TestGetCustodyGroups*`
  - `TestComputeColumnsForCustodyGroup*`
  - `TestGetCustodyColumns*`
- ✅ EIP-6110 (Deposit Requests)
  - `TestDepositRequestEncodeDecode`
  - `TestDepositRequestHashSSZ`
  - `TestDepositRequestClone`
  - `TestDepositRequestStatic`
  - `TestPendingDepositEncodeDecode`
  - `TestPendingDepositHashSSZ`
  - `TestDepositRequestDataLength`
  - `TestDepositRequestInterfaceCompliance`
- ✅ EIP-7002 (Withdrawal Requests)
  - `TestWithdrawalRequestEncodeDecode`
  - `TestWithdrawalRequestHashSSZ`
  - `TestWithdrawalRequestClone`
  - `TestWithdrawalRequestStatic`
  - `TestPendingPartialWithdrawalEncodeDecode`
  - `TestPendingPartialWithdrawalHashSSZ`
  - `TestWithdrawalRequestDataLength`
  - `TestWithdrawalRequestInterfaceCompliance`
  - `TestWithdrawalRequestFullWithdrawal`
  - `TestWithdrawalRequestPartialWithdrawal`
- ✅ EIP-7251 (Consolidation Requests)
  - `TestConsolidationRequestEncodeDecode`
  - `TestConsolidationRequestHashSSZ`
  - `TestConsolidationRequestClone`
  - `TestConsolidationRequestStatic`
  - `TestPendingConsolidationEncodeDecode`
  - `TestPendingConsolidationHashSSZ`
  - `TestConsolidationRequestDataLength`
  - `TestConsolidationRequestInterfaceCompliance`
  - `TestConsolidationRequestSwitchToCompounding`
  - `TestConsolidationRequestDifferentPubkeys`
- ✅ Version handling tests for Electra/Fulu
  - `TestStateVersion*`
  - `TestElectraVersionCheck`
  - `TestFuluVersionCheck`
  - `TestVersionFeatureDetection`

Still needs more coverage:
- Full state transition tests for Electra/Fulu
- Additional network service tests for Electra gossip protocol

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

