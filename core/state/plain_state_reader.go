package state

import (
	"bytes"
	"encoding/binary"

	"github.com/erigontech/erigon-lib/kv/dbutils"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/kv"

	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/core/types/accounts"
)

// EIP7702FixVersion is used to track code changes for debugging
const EIP7702FixVersion = "v13-all-readers-recovery"

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
	// v12: Restore CodeHash recovery for EIP-7702 delegation accounts
	// Only recover if:
	// 1. Account's CodeHash is empty (from Erigon 3 snapshot)
	// 2. PlainContractCode has an entry
	// 3. The entry is NOT emptyCodeHash (delegation wasn't revoked)
	// 4. The code at that hash is a valid EIP-7702 delegation
	if a.IsEmptyCodeHash() {
		if codeHash, err2 := r.db.GetOne(kv.PlainContractCode, dbutils.PlainGenerateStoragePrefix(address[:], a.Incarnation)); err2 == nil && len(codeHash) > 0 && !bytes.Equal(codeHash, emptyCodeHash) {
			// Verify the code is a valid EIP-7702 delegation before using this CodeHash
			if code, err3 := r.db.GetOne(kv.Code, codeHash); err3 == nil && types.IsDelegation(code) {
				a.CodeHash = libcommon.BytesToHash(codeHash)
			}
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
