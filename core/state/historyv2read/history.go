package historyv2read

import (
	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/kv"
	"github.com/erigontech/erigon-lib/kv/temporal/historyv2"
	"github.com/erigontech/erigon/core/types/accounts"
)

const DefaultIncarnation = uint64(1)

// emptyCodeHash is the known hash of an empty code (Keccak256 of empty bytes)
var emptyCodeHash = libcommon.HexToHash("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes()

func RestoreCodeHash(tx kv.Getter, key, v []byte, force *libcommon.Hash) ([]byte, error) {
	var acc accounts.Account
	if err := acc.DecodeForStorage(v); err != nil {
		return nil, err
	}
	if force != nil {
		acc.CodeHash = *force
		v = make([]byte, acc.EncodingLengthForStorage())
		acc.EncodeForStorage(v)
		return v, nil
	}
	// v11: NO CodeHash recovery - testing clean state
	return v, nil
}

func GetAsOf(tx kv.Tx, indexC kv.Cursor, changesC kv.CursorDupSort, storage bool, key []byte, timestamp uint64) (v []byte, fromHistory bool, err error) {
	v, ok, err := historyv2.FindByHistory(indexC, changesC, storage, key, timestamp)
	if err != nil {
		return nil, true, err
	}
	if ok {
		return v, true, nil
	}
	v, err = tx.GetOne(kv.PlainState, key)
	return v, false, err
}
