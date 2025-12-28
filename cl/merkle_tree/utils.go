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

package merkle_tree

import (
	"github.com/erigontech/erigon/cl/utils"
)

func NextPowerOfTwo(n uint64) uint64 {
	if n == 0 {
		return 1
	}
	// http://graphics.stanford.edu/~seander/bithacks.html#RoundUpPowerOf2
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++

	return n
}

// GetDepth returns the depth of a merkle tree with a given number of nodes.
// The depth is defined as the number of levels in the tree, with the root
// node at level 0 and each child node at a level one greater than its parent.
// If the number of nodes is less than or equal to 1, the depth is 0.
func GetDepth(v uint64) uint8 {
	// If there are 0 or 1 nodes, the depth is 0.
	if v <= 1 {
		return 0
	}

	// Initialize the depth to 0.
	depth := uint8(0)

	// Divide the number of nodes by 2 until it is less than or equal to 1.
	// The number of iterations is the depth of the tree.
	for v > 1 {
		v >>= 1
		depth++
	}

	return depth
}

// FloorLog2 returns floor(log2(n)) for n > 0
func FloorLog2(n uint64) uint64 {
	if n <= 1 {
		return 0
	}
	result := uint64(0)
	for n > 1 {
		n >>= 1
		result++
	}
	return result
}

// IsValidMerkleBranch verifies a merkle proof against a root
func IsValidMerkleBranch(leaf [32]byte, branch []byte32, depth uint64, index uint64, root [32]byte) bool {
	value := leaf
	for i := uint64(0); i < depth; i++ {
		if i < uint64(len(branch)) {
			if (index>>i)&1 == 1 {
				value = hashTwo(branch[i], value)
			} else {
				value = hashTwo(value, branch[i])
			}
		}
	}
	return value == root
}

type byte32 = [32]byte

func hashTwo(a, b [32]byte) [32]byte {
	return utils.Sha256(a[:], b[:])
}



























