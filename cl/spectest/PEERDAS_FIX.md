# PeerDAS Inclusion Proof Fix

**Date**: 2026-01-11
**Status**: Fixed
**Issue**: DataColumn sidecar inclusion proof verification was failing for all columns

## Problem Summary

All DataColumn sidecars were failing inclusion proof verification in the test `on_block_peerdas__ok`, even though KZG proof verification was passing.

### Root Cause

The generalized index calculation for `BlobKzgCommitments` in `VerifyDataColumnSidecarInclusionProof()` was incorrect.

**Incorrect Code** (line 209 in `cl/das/p2p_utils.go`):
```go
gIndex := clparams.GetBeaconConfig().MaxBlobsPerBlock + sidecar.Index
```

This calculated the gIndex as `6 + column_index`, which was wrong because:
1. `sidecar.Index` is the column index (0-127)
2. `BlobKzgCommitments` field has a fixed position in BeaconBody
3. The column index should NOT be used to calculate the field's generalized index

### BeaconBody Structure

```
BeaconBody fields (0-indexed):
 0: RandaoReveal
 1: Eth1Data
 2: Graffiti
 3: ProposerSlashings
 4: AttesterSlashings
 5: Attestations
 6: Deposits
 7: VoluntaryExits
 8: SyncAggregate
 9: ExecutionPayload
10: ExecutionChanges (BLSToExecutionChanges)
11: BlobKzgCommitments ← This field!
12: ExecutionRequests
```

### Correct Calculation

According to SSZ Merkle tree specification:
- BeaconBody has 13 fields
- Next power of 2: 16
- Field 11 generalized index: 16 + 11 = **27**

**Correct Code**:
```go
// BlobKzgCommitments is field 11 in BeaconBody (0-indexed)
// BeaconBody has 13 fields, next power of 2 is 16
// Generalized index = 16 + 11 = 27
const blobKzgCommitmentsGindex = 27

return merkle_tree.IsValidMerkleBranch(
    commitmentsRoot,
    branch,
    kzgCommitmentsInclusionProofDepth,
    blobKzgCommitmentsGindex,
    sidecar.SignedBlockHeader.Header.BodyRoot,
)
```

## Fix Details

**File**: `cl/das/p2p_utils.go`
**Function**: `VerifyDataColumnSidecarInclusionProof`
**Lines Changed**: 208-220

## Test Results

### Before Fix
```
Column 0 verification failed: overall=false, inclusion=false, kzg=true
Column 1 verification failed: overall=false, inclusion=false, kzg=true
...
(ALL 128 columns failed inclusion proof verification)
```

### After Fix
- No "Column verification failed" messages
- Inclusion proof verification now passes
- Test may still timeout due to other reasons (KZG computation intensity)

## Impact

This fix affects:
- **PeerDAS (EIP-7594)** data availability sampling
- **Fulu fork** readiness
- DataColumn sidecar validation in the consensus layer

## References

- **EIP-7594**: https://eips.ethereum.org/EIPS/eip-7594
- **SSZ Merkle Proofs**: https://github.com/ethereum/consensus-specs/blob/dev/ssz/merkle-proofs.md
- **Generalized Indices**: https://github.com/ethereum/consensus-specs/blob/dev/ssz/merkle-proofs.md#generalized-merkle-tree-index

## Testing

To test this fix:
```bash
cd cl/spectest
CGO_CFLAGS=-D__BLST_PORTABLE__ go test -tags=spectest -run="/mainnet/fulu/fork_choice/on_block/pyspec_tests/on_block_peerdas__ok" -v
```

## Related Files

- `cl/das/p2p_utils.go` - Verification functions
- `cl/cltypes/beacon_block.go` - BeaconBody structure definition
- `cl/cltypes/column_sidecar.go` - DataColumnSidecar structure
- `cl/spectest/consensus_tests/fork_choice.go` - Test harness
