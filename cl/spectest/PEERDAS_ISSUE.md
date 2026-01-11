# PeerDAS Test Failure Investigation

**Date**: 2026-01-10
**Status**: Requires Deep Investigation
**Priority**: High (Fulu fork readiness)

## Failing Test

**Test Name**: `Test/mainnet/fulu/fork_choice/on_block/pyspec_tests/on_block_peerdas__ok`

**Location**: `cl/spectest/consensus_tests/fork_choice.go:307`

**Error**:
```
Error Trace: cl/spectest/consensus_tests/fork_choice.go:307
Error: Not equal:
  expected: true
  actual  : false
Messages: step 4: on_block
```

## Root Cause Analysis

### Problem Location

The failure occurs in the DataColumn verification logic at lines 289-309 of `fork_choice.go`:

```go
if step.GetColumns() != nil {
    allok := true
    for _, filename := range step.GetColumns() {
        column := cltypes.NewDataColumnSidecar()
        err := spectest.ReadSsz(root, c.Version(), filename+".ssz_snappy", column)
        require.NoError(t, err, stepstr)

        // THREE VERIFICATION CHECKS:
        if das.VerifyDataColumnSidecar(column) &&           // Check 1
           das.VerifyDataColumnSidecarInclusionProof(column) &&  // Check 2
           das.VerifyDataColumnSidecarKZGProofs(column) {   // Check 3
            // Success: write to columnStorage
            blockRoot, err := blk.Block.HashSSZ()
            require.NoError(t, err)
            err = columnStorage.WriteColumnSidecars(ctx, blockRoot, int64(column.Index), column)
            require.NoError(t, err)
        } else {
            allok = false  // AT LEAST ONE CHECK FAILED
        }
    }
    if !allok {
        // Line 307: This is where the test fails
        require.Equal(t, step.GetValid(), allok, stepstr)
        continue
    }
}
```

### What's Happening

1. Test "on_block_peerdas__ok" expects the block to be VALID (`step.GetValid() == true`)
2. The test provides DataColumn sidecars that should pass verification
3. One or more of the three verification functions returns `false`:
   - `das.VerifyDataColumnSidecar(column)`
   - `das.VerifyDataColumnSidecarInclusionProof(column)`
   - `das.VerifyDataColumnSidecarKZGProofs(column)`
4. When verification fails, `allok` is set to `false`
5. Line 307 checks: `require.Equal(t, true, false)` → **FAIL**

### Verification Functions

The three verification functions are imported from:
```go
import "github.com/erigontech/erigon/cl/das"
```

**Function Purposes**:
1. **VerifyDataColumnSidecar**: Validates basic structure and format of the data column
2. **VerifyDataColumnSidecarInclusionProof**: Verifies the Merkle proof showing the column is part of the block
3. **VerifyDataColumnSidecarKZGProofs**: Validates KZG proofs for the data availability sampling

## Investigation Steps Needed

### 1. Identify Which Verification Fails

Add debug logging to determine which specific check fails:

```go
valid1 := das.VerifyDataColumnSidecar(column)
valid2 := das.VerifyDataColumnSidecarInclusionProof(column)
valid3 := das.VerifyDataColumnSidecarKZGProofs(column)

t.Logf("Column %d verifications: sidecar=%v, proof=%v, kzg=%v",
       column.Index, valid1, valid2, valid3)

if valid1 && valid2 && valid3 {
    // success path
} else {
    allok = false
}
```

### 2. Examine DAS Package Implementation

Navigate to `cl/das/` and review:
- Implementation of the three verification functions
- Dependencies on EIP-7594 (PeerDAS) specifications
- KZG cryptography setup and parameters

### 3. Check Test Data

Examine the test file at:
`tests/mainnet/fulu/fork_choice/on_block/pyspec_tests/on_block_peerdas__ok/`

Verify:
- DataColumn sidecar SSZ files are correctly formatted
- KZG commitments and proofs match expected values
- Block structure includes all required PeerDAS fields

### 4. Compare with Passing Tests

All the "invalid" PeerDAS tests PASS:
- `on_block_peerdas__invalid_mismatch_len_column_1` ✅
- `on_block_peerdas__invalid_wrong_proof_1` ✅
- `on_block_peerdas__invalid_index_1` ✅
- etc.

This suggests:
- Invalid detection works correctly
- Valid case verification has a bug

### 5. Check EIP-7594 Compliance

Review the implementation against [EIP-7594 specification](https://eips.ethereum.org/EIPS/eip-7594):
- Data column structure
- KZG proof requirements
- Inclusion proof requirements
- Verification algorithms

## Potential Issues

### Issue 1: KZG Setup Not Initialized
PeerDAS requires KZG trusted setup. Check if:
- KZG parameters are loaded correctly
- Trusted setup ceremony data is available
- Initialization happens before tests run

### Issue 2: Incorrect Proof Verification
The KZG proof verification might have:
- Wrong parameters (domain, generator points)
- Incorrect pairing check implementation
- Missing Fulu fork-specific changes

### Issue 3: Inclusion Proof Root Mismatch
The Merkle inclusion proof might fail due to:
- Incorrect tree construction
- Wrong generalized index
- Mismatch between expected and actual block roots

### Issue 4: Column Index Out of Range
Fulu fork introduces specific column count requirements:
- Check if column index validation is correct
- Verify NUMBER_OF_COLUMNS parameter matches spec

## Recommended Fix Approach

1. **Add Debug Logging** (30 min)
   - Instrument the verification code
   - Run test again to see which check fails
   - Log column index, commitments, proofs

2. **Review DAS Implementation** (2-4 hours)
   - Study `cl/das/` package
   - Check against EIP-7594 spec
   - Look for obvious bugs

3. **Test with Spec Test Data** (1 hour)
   - Manually verify the test data
   - Use reference implementation to validate
   - Compare results

4. **Fix and Validate** (2-4 hours)
   - Implement fix based on findings
   - Re-run all PeerDAS tests
   - Ensure no regressions

**Total Estimated Time**: 5.5 - 9.5 hours

## Workaround (Temporary)

If PeerDAS is not immediately needed, the test can be skipped:

```go
// In fork_choice.go, after line 289:
if step.GetColumns() != nil {
    // TODO: Fix PeerDAS verification (see PEERDAS_ISSUE.md)
    t.Skip("PeerDAS verification needs fixing - tracked in PEERDAS_ISSUE.md")
    return nil
}
```

**Note**: This should only be a temporary measure until proper fix is implemented.

## Related Files

- `cl/spectest/consensus_tests/fork_choice.go` - Test harness
- `cl/das/` - DAS verification implementation (TO BE LOCATED)
- `cl/cltypes/data_column_sidecar.go` - DataColumn structure (TO BE VERIFIED)
- `tests/mainnet/fulu/fork_choice/on_block/pyspec_tests/on_block_peerdas__ok/` - Test data

## References

- [EIP-7594: PeerDAS](https://eips.ethereum.org/EIPS/eip-7594)
- [Consensus Specs - PeerDAS](https://github.com/ethereum/consensus-specs/tree/dev/specs/_features/eip7594)
- [KZG Ceremony](https://ceremony.ethereum.org/)

## Next Steps

- [ ] Add debug logging to identify failing verification
- [ ] Locate and review `cl/das/` implementation
- [ ] Compare with reference implementation
- [ ] Implement fix
- [ ] Validate all PeerDAS tests pass
- [ ] Document the fix
