package state

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/erigontech/erigon-lib/crypto"
	"github.com/erigontech/erigon-lib/kv/dbutils"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/kv"

	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/core/types/accounts"
)

// EIP7702FixVersion is used to track code changes for debugging
const EIP7702FixVersion = "v17-format-diag"

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

// Diagnostic counters for EIP-7702 CodeHash recovery
var (
	diagEmptyCodeHashCount       int64
	diagPlainContractCodeFound   int64
	diagPlainContractCodeMissing int64
	diagRecoverySuccess          int64
	diagCodeDomainFound          int64
	diagRawDataSamples           int64 // Limit raw data output
)

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
	// v17: Diagnostic version to check data format
	// Output raw data for first few accounts with empty CodeHash
	if a.IsEmptyCodeHash() {
		diagEmptyCodeHashCount++

		// Output raw data for first 3 accounts to diagnose format
		if diagRawDataSamples < 3 {
			diagRawDataSamples++
			// Check if it looks like V2 or V3 format
			// V2: first byte is fieldSet (bit flags)
			// V3: first byte is nonceBytes length
			fieldSet := enc[0]
			fmt.Printf("[EIP7702-RAW] addr=%x len=%d raw[0]=%d(0x%x) fieldSet_bits=%08b nonce=%d balance=%s inc=%d codeHash=%x\n",
				address[:4], len(enc), fieldSet, fieldSet, fieldSet, a.Nonce, a.Balance.String(), a.Incarnation, a.CodeHash[:4])
		}

		recovered := false

		// Method 1: Try PlainContractCode table
		incarnation := a.Incarnation
		if incarnation == 0 {
			incarnation = 1
		}
		if codeHash, err2 := r.db.GetOne(kv.PlainContractCode, dbutils.PlainGenerateStoragePrefix(address[:], incarnation)); err2 == nil && len(codeHash) > 0 && !bytes.Equal(codeHash, emptyCodeHash) {
			diagPlainContractCodeFound++
			if code, err3 := r.db.GetOne(kv.Code, codeHash); err3 == nil && types.IsDelegation(code) {
				a.CodeHash = libcommon.BytesToHash(codeHash)
				diagRecoverySuccess++
				recovered = true
			}
		}

		// Method 2: Try CodeDomain via TemporalTx (for Erigon 3 snapshots)
		if !recovered {
			if ttx, ok := r.db.(kv.TemporalTx); ok {
				// Get latest code from CodeDomain
				if code, ok2, err2 := ttx.DomainGet(kv.CodeDomain, address.Bytes(), nil); err2 == nil && ok2 && len(code) > 0 {
					if types.IsDelegation(code) {
						a.CodeHash = crypto.Keccak256Hash(code)
						diagCodeDomainFound++
						diagRecoverySuccess++
						recovered = true
					}
				}
			}
		}

		if !recovered {
			diagPlainContractCodeMissing++
		}
	}
	return &a, nil
}

// GetDiagnostics returns diagnostic counters for CodeHash recovery
func GetDiagnostics() (emptyCodeHash, found, missing, success, codeDomain int64) {
	return diagEmptyCodeHashCount, diagPlainContractCodeFound, diagPlainContractCodeMissing, diagRecoverySuccess, diagCodeDomainFound
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
