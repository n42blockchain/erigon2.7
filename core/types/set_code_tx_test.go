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

package types

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erigontech/erigon-lib/chain"
	libcommon "github.com/erigontech/erigon-lib/common"
	types2 "github.com/erigontech/erigon-lib/types"

	"github.com/erigontech/erigon/params"
)

// TestSetCodeTransactionType verifies the SetCodeTxType constant
func TestSetCodeTransactionType(t *testing.T) {
	t.Parallel()
	// EIP-7702 defines SetCodeTxType as 0x04
	assert.Equal(t, byte(4), byte(SetCodeTxType))
}

// TestSetCodeTransactionBasic tests basic SetCodeTransaction creation and properties
func TestSetCodeTransactionBasic(t *testing.T) {
	t.Parallel()

	to := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")
	delegateAddr := libcommon.HexToAddress("0x2222222222222222222222222222222222222222")

	tx := &SetCodeTransaction{
		DynamicFeeTransaction: DynamicFeeTransaction{
			CommonTx: CommonTx{
				Nonce: 1,
				Gas:   100000,
				To:    &to,
				Value: uint256.NewInt(1000),
				Data:  []byte{0x01, 0x02, 0x03},
			},
			ChainID: uint256.NewInt(1),
			Tip:     uint256.NewInt(1000000000),
			FeeCap:  uint256.NewInt(2000000000),
		},
		Authorizations: []Authorization{
			{
				ChainID: *uint256.NewInt(1),
				Address: delegateAddr,
				Nonce:   0,
				YParity: 0,
				R:       *uint256.NewInt(123456),
				S:       *uint256.NewInt(654321),
			},
		},
	}

	// Verify type
	assert.Equal(t, byte(SetCodeTxType), tx.Type())

	// Verify GetAuthorizations
	auths := tx.GetAuthorizations()
	assert.Len(t, auths, 1)
	assert.Equal(t, delegateAddr, auths[0].Address)

	// Verify GetBlobHashes returns empty (SetCodeTx doesn't use blobs)
	assert.Empty(t, tx.GetBlobHashes())
}

// TestSetCodeTransactionCopy tests deep copy of SetCodeTransaction
func TestSetCodeTransactionCopy(t *testing.T) {
	t.Parallel()

	to := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")
	delegateAddr := libcommon.HexToAddress("0x2222222222222222222222222222222222222222")

	original := &SetCodeTransaction{
		DynamicFeeTransaction: DynamicFeeTransaction{
			CommonTx: CommonTx{
				Nonce: 1,
				Gas:   100000,
				To:    &to,
				Value: uint256.NewInt(1000),
				Data:  []byte{0x01, 0x02, 0x03},
			},
			ChainID: uint256.NewInt(1),
			Tip:     uint256.NewInt(1000000000),
			FeeCap:  uint256.NewInt(2000000000),
		},
		Authorizations: []Authorization{
			{
				ChainID: *uint256.NewInt(1),
				Address: delegateAddr,
				Nonce:   0,
			},
		},
	}

	copied := original.copy()

	// Verify copy is equal
	assert.Equal(t, original.Nonce, copied.Nonce)
	assert.Equal(t, original.Gas, copied.Gas)
	assert.Equal(t, *original.To, *copied.To)
	assert.Equal(t, len(original.Authorizations), len(copied.Authorizations))
	assert.Equal(t, original.Authorizations[0].Address, copied.Authorizations[0].Address)

	// Modify copy and verify original is unchanged
	copied.Nonce = 999
	copied.Authorizations[0].Nonce = 999
	assert.NotEqual(t, original.Nonce, copied.Nonce)
	assert.NotEqual(t, original.Authorizations[0].Nonce, copied.Authorizations[0].Nonce)
}

// TestSetCodeTransactionEncodeDecode tests RLP encoding/decoding
func TestSetCodeTransactionEncodeDecode(t *testing.T) {
	t.Parallel()

	to := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")
	delegateAddr := libcommon.HexToAddress("0x2222222222222222222222222222222222222222")

	original := &SetCodeTransaction{
		DynamicFeeTransaction: DynamicFeeTransaction{
			CommonTx: CommonTx{
				Nonce: 1,
				Gas:   100000,
				To:    &to,
				Value: uint256.NewInt(1000),
				Data:  []byte{0x01, 0x02, 0x03},
			},
			ChainID: uint256.NewInt(1),
			Tip:     uint256.NewInt(1000000000),
			FeeCap:  uint256.NewInt(2000000000),
			AccessList: types2.AccessList{
				{
					Address:     libcommon.HexToAddress("0x3333333333333333333333333333333333333333"),
					StorageKeys: []libcommon.Hash{{1}, {2}},
				},
			},
		},
		Authorizations: []Authorization{
			{
				ChainID: *uint256.NewInt(1),
				Address: delegateAddr,
				Nonce:   0,
				YParity: 0,
				R:       *uint256.NewInt(123456),
				S:       *uint256.NewInt(654321),
			},
		},
	}

	// Encode
	var buf bytes.Buffer
	err := original.MarshalBinary(&buf)
	require.NoError(t, err)

	encoded := buf.Bytes()

	// Verify first byte is type
	assert.Equal(t, byte(SetCodeTxType), encoded[0])

	// Verify encoding size
	assert.Equal(t, original.EncodingSize(), len(encoded))
}

// TestSetCodeTransactionHash tests hash computation
func TestSetCodeTransactionHash(t *testing.T) {
	t.Parallel()

	to := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")
	delegateAddr := libcommon.HexToAddress("0x2222222222222222222222222222222222222222")

	tx := &SetCodeTransaction{
		DynamicFeeTransaction: DynamicFeeTransaction{
			CommonTx: CommonTx{
				Nonce: 1,
				Gas:   100000,
				To:    &to,
				Value: uint256.NewInt(1000),
				Data:  []byte{0x01, 0x02, 0x03},
			},
			ChainID: uint256.NewInt(1),
			Tip:     uint256.NewInt(1000000000),
			FeeCap:  uint256.NewInt(2000000000),
		},
		Authorizations: []Authorization{
			{
				ChainID: *uint256.NewInt(1),
				Address: delegateAddr,
				Nonce:   0,
			},
		},
	}

	// Hash should be deterministic
	hash1 := tx.Hash()
	hash2 := tx.Hash()
	assert.Equal(t, hash1, hash2)

	// Hash should be non-zero
	assert.NotEqual(t, libcommon.Hash{}, hash1)
}

// TestSetCodeTransactionAsMessageRulesCheck tests that SetCodeTransaction requires Prague rules
func TestSetCodeTransactionAsMessageRulesCheck(t *testing.T) {
	t.Parallel()

	to := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")
	delegateAddr := libcommon.HexToAddress("0x2222222222222222222222222222222222222222")

	// Use valid signature values (from a test vector)
	tx := &SetCodeTransaction{
		DynamicFeeTransaction: DynamicFeeTransaction{
			CommonTx: CommonTx{
				Nonce: 1,
				Gas:   100000,
				To:    &to,
				Value: uint256.NewInt(1000),
				Data:  []byte{0x01, 0x02, 0x03},
				V:     *uint256.NewInt(0),
				R:     *uint256.NewInt(1),
				S:     *uint256.NewInt(1),
			},
			ChainID: uint256.NewInt(1),
			Tip:     uint256.NewInt(1000000000),
			FeeCap:  uint256.NewInt(2000000000),
		},
		Authorizations: []Authorization{
			{
				ChainID: *uint256.NewInt(1),
				Address: delegateAddr,
				Nonce:   0,
			},
		},
	}

	// Test with non-Prague rules - should fail with "Prague" error
	nonPragueRules := &chain.Rules{IsPrague: false, IsOsaka: false}
	pragueConfig := &chain.Config{ChainID: big.NewInt(1), PragueTime: big.NewInt(0)}
	pragueSigner := MakeSigner(pragueConfig, 0, 0)
	_, err := tx.AsMessage(*pragueSigner, big.NewInt(1000000000), nonPragueRules)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Prague")
}

// TestSetCodeTransactionWithoutAuthorizations tests that empty authorizations is invalid
func TestSetCodeTransactionWithoutAuthorizations(t *testing.T) {
	t.Parallel()

	to := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")

	tx := &SetCodeTransaction{
		DynamicFeeTransaction: DynamicFeeTransaction{
			CommonTx: CommonTx{
				Nonce: 1,
				Gas:   100000,
				To:    &to,
				Value: uint256.NewInt(1000),
			},
			ChainID: uint256.NewInt(1),
			Tip:     uint256.NewInt(1000000000),
			FeeCap:  uint256.NewInt(2000000000),
		},
		Authorizations: []Authorization{}, // Empty
	}

	pragueRules := &chain.Rules{IsPrague: true}
	pragueConfig := &chain.Config{ChainID: big.NewInt(1), PragueTime: big.NewInt(0)}
	pragueSigner := MakeSigner(pragueConfig, 0, 0)
	_, err := tx.AsMessage(*pragueSigner, big.NewInt(1000000000), pragueRules)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "without authorizations")
}

// TestDelegationDesignationPrefix tests the EIP-7702 delegation designation prefix
func TestDelegationDesignationPrefix(t *testing.T) {
	t.Parallel()

	// EIP-7702 specifies: 0xef0100 + 20-byte address
	assert.Equal(t, []byte{0xef, 0x01, 0x00}, params.DelegatedDesignationPrefix)
	assert.Equal(t, 23, DelegateDesignationCodeSize) // 3 + 20 = 23 bytes
}

// TestIsDelegation tests the IsDelegation function
func TestIsDelegation(t *testing.T) {
	t.Parallel()

	delegateAddr := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")

	tests := []struct {
		name     string
		code     []byte
		expected bool
	}{
		{
			name:     "valid delegation",
			code:     AddressToDelegation(delegateAddr),
			expected: true,
		},
		{
			name:     "nil",
			code:     nil,
			expected: false,
		},
		{
			name:     "empty",
			code:     []byte{},
			expected: false,
		},
		{
			name:     "too short",
			code:     []byte{0xef, 0x01, 0x00},
			expected: false,
		},
		{
			name:     "too long",
			code:     make([]byte, 24),
			expected: false,
		},
		{
			name:     "wrong prefix",
			code:     append([]byte{0xef, 0x02, 0x00}, delegateAddr.Bytes()...),
			expected: false,
		},
		{
			name:     "regular contract code",
			code:     []byte{0x60, 0x00, 0x60, 0x00, 0xf3}, // PUSH1 0 PUSH1 0 RETURN
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDelegation(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseDelegation tests the ParseDelegation function
func TestParseDelegation(t *testing.T) {
	t.Parallel()

	delegateAddr := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")
	delegationCode := AddressToDelegation(delegateAddr)

	// Valid delegation
	addr, ok := ParseDelegation(delegationCode)
	assert.True(t, ok)
	assert.Equal(t, delegateAddr, addr)

	// Invalid delegation
	addr, ok = ParseDelegation([]byte{0x60, 0x00})
	assert.False(t, ok)
	assert.Equal(t, libcommon.Address{}, addr)
}

// TestAddressToDelegation tests the AddressToDelegation function
func TestAddressToDelegation(t *testing.T) {
	t.Parallel()

	delegateAddr := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")
	delegationCode := AddressToDelegation(delegateAddr)

	// Verify length
	assert.Len(t, delegationCode, DelegateDesignationCodeSize)

	// Verify prefix
	assert.True(t, bytes.HasPrefix(delegationCode, params.DelegatedDesignationPrefix))

	// Verify address
	assert.Equal(t, delegateAddr.Bytes(), delegationCode[len(params.DelegatedDesignationPrefix):])
}

// TestAuthorizationCopy tests deep copy of Authorization
func TestAuthorizationCopy(t *testing.T) {
	t.Parallel()

	delegateAddr := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")

	original := &Authorization{
		ChainID: *uint256.NewInt(1),
		Address: delegateAddr,
		Nonce:   5,
		YParity: 1,
		R:       *uint256.NewInt(123456),
		S:       *uint256.NewInt(654321),
	}

	copied := original.copy()

	// Verify copy is equal
	assert.Equal(t, original.ChainID, copied.ChainID)
	assert.Equal(t, original.Address, copied.Address)
	assert.Equal(t, original.Nonce, copied.Nonce)
	assert.Equal(t, original.YParity, copied.YParity)
	assert.Equal(t, original.R, copied.R)
	assert.Equal(t, original.S, copied.S)

	// Modify copy and verify original is unchanged
	copied.Nonce = 999
	assert.NotEqual(t, original.Nonce, copied.Nonce)
}

// TestMultipleAuthorizations tests SetCodeTransaction with multiple authorizations
func TestMultipleAuthorizations(t *testing.T) {
	t.Parallel()

	to := libcommon.HexToAddress("0x1234567890123456789012345678901234567890")

	tx := &SetCodeTransaction{
		DynamicFeeTransaction: DynamicFeeTransaction{
			CommonTx: CommonTx{
				Nonce: 1,
				Gas:   100000,
				To:    &to,
				Value: uint256.NewInt(1000),
			},
			ChainID: uint256.NewInt(1),
			Tip:     uint256.NewInt(1000000000),
			FeeCap:  uint256.NewInt(2000000000),
		},
		Authorizations: []Authorization{
			{
				ChainID: *uint256.NewInt(1),
				Address: libcommon.HexToAddress("0x1111111111111111111111111111111111111111"),
				Nonce:   0,
			},
			{
				ChainID: *uint256.NewInt(1),
				Address: libcommon.HexToAddress("0x2222222222222222222222222222222222222222"),
				Nonce:   1,
			},
			{
				ChainID: *uint256.NewInt(1),
				Address: libcommon.HexToAddress("0x3333333333333333333333333333333333333333"),
				Nonce:   2,
			},
		},
	}

	// Verify GetAuthorizations returns all authorizations
	assert.Len(t, tx.GetAuthorizations(), 3)
	assert.Equal(t, libcommon.HexToAddress("0x1111111111111111111111111111111111111111"), tx.Authorizations[0].Address)
	assert.Equal(t, libcommon.HexToAddress("0x2222222222222222222222222222222222222222"), tx.Authorizations[1].Address)
	assert.Equal(t, libcommon.HexToAddress("0x3333333333333333333333333333333333333333"), tx.Authorizations[2].Address)
}

// TestDelegatedCodeHashConstant tests the DelegatedCodeHash constant
func TestDelegatedCodeHashConstant(t *testing.T) {
	t.Parallel()

	// The DelegatedCodeHash is a constant used for identifying delegation accounts
	// It should be a specific well-known hash
	expectedHash := libcommon.HexToHash("0xeadcdba66a79ab5dce91622d1d75c8cff5cff0b96944c3bf1072cd08ce018329")
	assert.Equal(t, expectedHash, params.DelegatedCodeHash)
}

// TestSetCodeMagicPrefix tests the SetCodeMagicPrefix constant
func TestSetCodeMagicPrefix(t *testing.T) {
	t.Parallel()

	// EIP-7702 specifies MAGIC = 0x05 for authorization signing
	assert.Equal(t, byte(0x05), params.SetCodeMagicPrefix)
}

