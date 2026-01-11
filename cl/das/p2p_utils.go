package das

import (
	"crypto/sha256"
	"fmt"
	"runtime"

	goethkzg "github.com/crate-crypto/go-eth-kzg"

	"github.com/erigontech/erigon/cl/clparams"
	"github.com/erigontech/erigon/cl/cltypes"
	"github.com/erigontech/erigon/cl/merkle_tree"
	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/log/v3"
)

// CellsPerExtBlob is the number of cells in an extended blob (128).
const CellsPerExtBlob = goethkzg.CellsPerExtBlob

// RecoverBlobs recovers blobs from data column sidecars.
// This implements the blob recovery mechanism from EIP-7594.
func RecoverBlobs(sidecars []*cltypes.DataColumnSidecar) ([]cltypes.Blob, error) {
	if len(sidecars) == 0 {
		return nil, nil
	}

	// Group cells by blob index
	blobCount := 0
	for _, sidecar := range sidecars {
		if sidecar.KzgCommitments.Len() > blobCount {
			blobCount = sidecar.KzgCommitments.Len()
		}
	}

	if blobCount == 0 {
		return nil, nil
	}

	ctx, err := goethkzg.NewContext4096Secure()
	if err != nil {
		return nil, fmt.Errorf("failed to create KZG context: %w", err)
	}

	blobs := make([]cltypes.Blob, blobCount)
	numGoRoutines := runtime.NumCPU()

	// Recover each blob
	for blobIdx := 0; blobIdx < blobCount; blobIdx++ {
		var cellIDs []uint64
		var cells []*goethkzg.Cell

		// Collect cells for this blob from all sidecars
		for _, sidecar := range sidecars {
			if sidecar.Column.Len() > blobIdx {
				cellIDs = append(cellIDs, sidecar.Index)
				cellPtr := sidecar.Column.Get(blobIdx)
				cell := goethkzg.Cell(*cellPtr)
				cells = append(cells, &cell)
			}
		}

		if len(cells) < CellsPerExtBlob/2 {
			return nil, fmt.Errorf("not enough cells for blob %d recovery: need at least %d, got %d",
				blobIdx, CellsPerExtBlob/2, len(cells))
		}

		// Recover all cells
		recoveredCells, _, err := ctx.RecoverCellsAndComputeKZGProofs(cellIDs, cells, numGoRoutines)
		if err != nil {
			return nil, fmt.Errorf("failed to recover cells for blob %d: %w", blobIdx, err)
		}

		// Reconstruct blob from cells
		var blobData goethkzg.Blob
		for i, cell := range recoveredCells {
			if cell != nil {
				copy(blobData[i*len(cell):], cell[:])
			}
		}
		blobs[blobIdx] = cltypes.Blob(blobData)
	}

	return blobs, nil
}

// VerifyDataColumnSidecar verifies that a data column sidecar is valid.
// This includes verifying the inclusion proof and KZG proofs.
func VerifyDataColumnSidecar(sidecar *cltypes.DataColumnSidecar) bool {
	if sidecar == nil {
		return false
	}
	return VerifyDataColumnSidecarInclusionProof(sidecar) && VerifyDataColumnsSidecarKZGProofs(sidecar)
}

// VerifyDataColumnsSidecarKZGProofs verifies the KZG proofs for a data column sidecar.
// This implements verify_cell_kzg_proof_batch from EIP-7594.
func VerifyDataColumnsSidecarKZGProofs(sidecar *cltypes.DataColumnSidecar) bool {
	// Basic structure validation
	if sidecar == nil {
		return false
	}
	if sidecar.KzgCommitments.Len() == 0 {
		return false
	}

	ctx, err := goethkzg.NewContext4096Secure()
	if err != nil {
		log.Error("[PeerDAS] Failed to create KZG context", "err", err)
		return false
	}

	// Build verification inputs
	commitments := make([]goethkzg.KZGCommitment, sidecar.KzgCommitments.Len())
	for i := 0; i < sidecar.KzgCommitments.Len(); i++ {
		commitment := sidecar.KzgCommitments.Get(i)
		commitments[i] = goethkzg.KZGCommitment(*commitment)
	}

	cellIndices := make([]uint64, sidecar.Column.Len())
	cells := make([]*goethkzg.Cell, sidecar.Column.Len())
	proofs := make([]goethkzg.KZGProof, sidecar.KzgProofs.Len())

	for i := 0; i < sidecar.Column.Len(); i++ {
		cellIndices[i] = sidecar.Index
		cellPtr := sidecar.Column.Get(i)
		cell := goethkzg.Cell(*cellPtr)
		cells[i] = &cell
	}

	for i := 0; i < sidecar.KzgProofs.Len(); i++ {
		proof := sidecar.KzgProofs.Get(i)
		proofs[i] = goethkzg.KZGProof(*proof)
	}

	err = ctx.VerifyCellKZGProofBatch(commitments, cellIndices, cells, proofs)
	if err != nil {
		log.Debug("[PeerDAS] KZG proof verification failed", "err", err, "columnIndex", sidecar.Index)
		return false
	}

	return true
}

// ComputeCells computes cells from a blob.
// This implements compute_cells from EIP-7594.
func ComputeCells(blob *cltypes.Blob) ([]cltypes.Cell, error) {
	ctx, err := goethkzg.NewContext4096Secure()
	if err != nil {
		return nil, fmt.Errorf("failed to create KZG context: %w", err)
	}

	goethBlob := (*goethkzg.Blob)(blob)
	numGoRoutines := runtime.NumCPU()

	cells, err := ctx.ComputeCells(goethBlob, numGoRoutines)
	if err != nil {
		return nil, fmt.Errorf("failed to compute cells: %w", err)
	}

	result := make([]cltypes.Cell, CellsPerExtBlob)
	for i := 0; i < CellsPerExtBlob; i++ {
		if cells[i] != nil {
			result[i] = cltypes.Cell(*cells[i])
		}
	}

	return result, nil
}

// VerifyDataColumnSidecarKZGProofs is an alias for VerifyDataColumnsSidecarKZGProofs
// for compatibility with code expecting the singular form
func VerifyDataColumnSidecarKZGProofs(sidecar *cltypes.DataColumnSidecar) bool {
	return VerifyDataColumnsSidecarKZGProofs(sidecar)
}

// ComputeSubnetForDataColumnSidecar computes the subnet ID for a given data column sidecar index.
// This function is re-entrant and thread-safe.
func ComputeSubnetForDataColumnSidecar(columnIndex cltypes.ColumnIndex) uint64 {
	return columnIndex % clparams.GetBeaconConfig().DataColumnSidecarSubnetCount
}

// VerifyDataColumnSidecarInclusionProof verifies if the inclusion proof in the sidecar is correct.
// This function is re-entrant and thread-safe.
func VerifyDataColumnSidecarInclusionProof(sidecar *cltypes.DataColumnSidecar) bool {
	// Convert branch to hashes for merkle proof verification
	branch := make([][32]byte, sidecar.KzgCommitmentsInclusionProof.Length())
	for i := range branch {
		branch[i] = sidecar.KzgCommitmentsInclusionProof.Get(i)
	}

	// Calculate the leaf - first we need the commitments root
	if sidecar.KzgCommitments.Len() == 0 {
		return false
	}

	// Compute kzg_commitment_inclusion_proof_depth
	kzgCommitmentsInclusionProofDepth := merkle_tree.FloorLog2(clparams.GetBeaconConfig().MaxBlobsPerBlock) +
		1 + // FloorLog2(MAX_BLOB_COMMITMENTS_PER_BLOCK)
		1 // Add 1 for the body index

	// Calculate the expected root by applying merkle proof
	// First build the commitments root
	commitmentsRoot, err := sidecar.KzgCommitments.HashSSZ()
	if err != nil {
		return false
	}

	// Verify the merkle proof from commitments root to block body root
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
}

// ComputeDataColumnHash computes the hash for a data column sidecar
func ComputeDataColumnHash(sidecar *cltypes.DataColumnSidecar) libcommon.Hash {
	// Combine relevant fields to create a unique hash
	data := make([]byte, 0, 128)
	data = append(data, sidecar.SignedBlockHeader.Header.BodyRoot[:]...)
	data = append(data, byte(sidecar.Index))
	commitmentsHash, _ := sidecar.KzgCommitments.HashSSZ()
	data = append(data, commitmentsHash[:]...)
	
	return sha256.Sum256(data)
}
