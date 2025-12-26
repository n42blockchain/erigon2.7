package state

import (
	"bytes"
	"encoding/binary"

	"github.com/erigontech/erigon-lib/kv/dbutils"

	"github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/kv"
	"github.com/erigontech/erigon-lib/kv/kvcache"

	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/core/types/accounts"
)

// CachedReader2 is a wrapper for an instance of type StateReader
// This wrapper only makes calls to the underlying reader if the item is not in the cache
type CachedReader2 struct {
	cache kvcache.CacheView
	db    kv.Tx
}

// NewCachedReader2 wraps a given state reader into the cached reader
func NewCachedReader2(cache kvcache.CacheView, tx kv.Tx) *CachedReader2 {
	return &CachedReader2{cache: cache, db: tx}
}

// ReadAccountData is called when an account needs to be fetched from the state
func (r *CachedReader2) ReadAccountData(address common.Address) (*accounts.Account, error) {
	enc, err := r.cache.Get(address[:])
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
	// 2. The original encoding indicates CodeHash should exist
	if a.IsEmptyCodeHash() && accounts.HasCodeHashInStorage(enc) {
		prefix := dbutils.PlainGenerateStoragePrefix(address[:], a.Incarnation)
		if codeHash, err1 := r.db.GetOne(kv.PlainContractCode, prefix); err1 == nil {
			if len(codeHash) > 0 {
				// Verify the code is a valid EIP-7702 delegation before using this CodeHash
				if code, err2 := r.db.GetOne(kv.Code, codeHash); err2 == nil && types.IsDelegation(code) {
					a.CodeHash.SetBytes(codeHash)
				}
			}
		} else {
			return nil, err1
		}
	}
	return &a, nil
}

func (r *CachedReader2) ReadAccountStorage(address common.Address, incarnation uint64, key *common.Hash) ([]byte, error) {
	compositeKey := dbutils.PlainGenerateCompositeStorageKey(address.Bytes(), incarnation, key.Bytes())
	enc, err := r.cache.Get(compositeKey)
	if err != nil {
		return nil, err
	}
	if len(enc) == 0 {
		return nil, nil
	}
	return enc, nil
}

func (r *CachedReader2) ReadAccountCode(address common.Address, incarnation uint64, codeHash common.Hash) ([]byte, error) {
	if bytes.Equal(codeHash.Bytes(), emptyCodeHash) {
		return nil, nil
	}
	code, err := r.cache.GetCode(codeHash.Bytes())
	if len(code) == 0 {
		return nil, nil
	}
	return code, err
}

func (r *CachedReader2) ReadAccountCodeSize(address common.Address, incarnation uint64, codeHash common.Hash) (int, error) {
	code, err := r.ReadAccountCode(address, incarnation, codeHash)
	return len(code), err
}

func (r *CachedReader2) ReadAccountIncarnation(address common.Address) (uint64, error) {
	b, err := r.db.GetOne(kv.IncarnationMap, address.Bytes())
	if err != nil {
		return 0, err
	}
	if len(b) == 0 {
		return 0, nil
	}
	return binary.BigEndian.Uint64(b), nil
}
