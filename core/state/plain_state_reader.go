package state

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/erigontech/erigon-lib/kv/dbutils"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/kv"

	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/core/types/accounts"
)

// EIP7702FixVersion is used to track code changes for debugging
const EIP7702FixVersion = "v7"

var _ StateReader = (*PlainStateReader)(nil)

// PlainStateReader reads data from so called "plain state".
// Data in the plain state is stored using un-hashed account/storage items
// as opposed to the "normal" state that uses hashes of merkle paths to store items.
type PlainStateReader struct {
	db kv.Getter
}

func NewPlainStateReader(db kv.Getter) *PlainStateReader {
	return &PlainStateReader{
		db: db,
	}
}

func (r *PlainStateReader) ReadAccountData(address libcommon.Address) (*accounts.Account, error) {
	enc, err := r.db.GetOne(kv.PlainState, address.Bytes())
	if err != nil {
		return nil, err
	}
	if len(enc) == 0 {
		return nil, nil
	}
	var a accounts.Account
	if err = a.DecodeForStorage(enc); err != nil {
		return nil, err
	}
	// EIP-7702: Only attempt to recover CodeHash from PlainContractCode if:
	// 1. The decoded account has empty CodeHash AND
	// 2. The original encoding indicates CodeHash should exist (fieldSet bit 3 is set)
	//
	// If the original encoding doesn't have CodeHash field (bit 3 not set), the account
	// genuinely has no code, and any data in PlainContractCode is stale/invalid.
	// This prevents recovering stale delegation CodeHashes for accounts that no longer have delegations.
	if a.IsEmptyCodeHash() && accounts.HasCodeHashInStorage(enc) {
		prefix := dbutils.PlainGenerateStoragePrefix(address[:], a.Incarnation)
		if codeHash, err1 := r.db.GetOne(kv.PlainContractCode, prefix); err1 == nil {
			if len(codeHash) > 0 {
				// Verify the code is a valid EIP-7702 delegation before using this CodeHash
				if code, err2 := r.db.GetOne(kv.Code, codeHash); err2 == nil && types.IsDelegation(code) {
					// Debug: log CodeHash recovery
					delegationTarget, _ := types.ParseDelegation(code)
					fmt.Printf("[EIP7702-%s] Recovered delegation CodeHash for %x -> %x (target: %x)\n",
						EIP7702FixVersion, address, codeHash[:8], delegationTarget)
					a.CodeHash = libcommon.BytesToHash(codeHash)
				}
			}
		} else {
			return nil, err1
		}
	}
	return &a, nil
}

func (r *PlainStateReader) ReadAccountStorage(address libcommon.Address, incarnation uint64, key *libcommon.Hash) ([]byte, error) {
	compositeKey := dbutils.PlainGenerateCompositeStorageKey(address.Bytes(), incarnation, key.Bytes())
	enc, err := r.db.GetOne(kv.PlainState, compositeKey)
	if err != nil {
		return nil, err
	}
	if len(enc) == 0 {
		return nil, nil
	}
	return enc, nil
}

func (r *PlainStateReader) ReadAccountCode(address libcommon.Address, incarnation uint64, codeHash libcommon.Hash) ([]byte, error) {
	if bytes.Equal(codeHash.Bytes(), emptyCodeHash) {
		return nil, nil
	}
	code, err := r.db.GetOne(kv.Code, codeHash.Bytes())
	if len(code) == 0 {
		return nil, nil
	}
	return code, err
}

func (r *PlainStateReader) ReadAccountCodeSize(address libcommon.Address, incarnation uint64, codeHash libcommon.Hash) (int, error) {
	code, err := r.ReadAccountCode(address, incarnation, codeHash)
	return len(code), err
}

func (r *PlainStateReader) ReadAccountIncarnation(address libcommon.Address) (uint64, error) {
	b, err := r.db.GetOne(kv.IncarnationMap, address.Bytes())
	if err != nil {
		return 0, err
	}
	if len(b) == 0 {
		return 0, nil
	}
	return binary.BigEndian.Uint64(b), nil
}
