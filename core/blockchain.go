// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package core implements the Ethereum consensus protocol.
package core

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"golang.org/x/crypto/sha3"

	math2 "github.com/erigontech/erigon-lib/common/math"
	"github.com/erigontech/erigon-lib/log/v3"
	"github.com/erigontech/erigon-lib/rlp"

	"github.com/erigontech/erigon-lib/chain"
	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/common/cmp"
	"github.com/erigontech/erigon-lib/common/dbg"
	"github.com/erigontech/erigon-lib/common/math"
	"github.com/erigontech/erigon-lib/metrics"
	"github.com/erigontech/erigon/common/u256"
	"github.com/erigontech/erigon/consensus"

	"github.com/erigontech/erigon/core/state"
	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/core/vm"
	"github.com/erigontech/erigon/core/vm/evmtypes"
	"github.com/erigontech/erigon/eth/ethutils"
	bortypes "github.com/erigontech/erigon/polygon/bor/types"
)

var (
	blockExecutionTimer = metrics.GetOrCreateSummary("chain_execution_seconds")
)

type SyncMode string

const (
	TriesInMemory = 128

	// See gas_limit in https://github.com/gnosischain/specs/blob/master/execution/withdrawals.md
	SysCallGasLimit = uint64(30_000_000)
)

type RejectedTx struct {
	Index int    `json:"index"    gencodec:"required"`
	Err   string `json:"error"    gencodec:"required"`
}

type RejectedTxs []*RejectedTx

type EphemeralExecResult struct {
	StateRoot        libcommon.Hash         `json:"stateRoot"`
	TxRoot           libcommon.Hash         `json:"txRoot"`
	ReceiptRoot      libcommon.Hash         `json:"receiptsRoot"`
	LogsHash         libcommon.Hash         `json:"logsHash"`
	Bloom            types.Bloom            `json:"logsBloom"        gencodec:"required"`
	Receipts         types.Receipts         `json:"receipts"`
	Rejected         RejectedTxs            `json:"rejected,omitempty"`
	Difficulty       *math2.HexOrDecimal256 `json:"currentDifficulty" gencodec:"required"`
	GasUsed          math.HexOrDecimal64    `json:"gasUsed"`
	StateSyncReceipt *types.Receipt         `json:"-"`
}

// ExecuteBlockEphemerally runs a block from provided stateReader and
// writes the result to the provided stateWriter
func ExecuteBlockEphemerally(
	chainConfig *chain.Config, vmConfig *vm.Config,
	blockHashFunc func(n uint64) libcommon.Hash,
	engine consensus.Engine, block *types.Block,
	stateReader state.StateReader, stateWriter state.WriterWithChangeSets,
	chainReader consensus.ChainReader, getTracer func(txIndex int, txHash libcommon.Hash) (vm.EVMLogger, error),
	logger log.Logger,
) (*EphemeralExecResult, error) {

	defer blockExecutionTimer.ObserveDuration(time.Now())
	block.Uncles()
	ibs := state.New(stateReader)
	header := block.Header()

	// Debug: Print header blob gas info
	debugBlock := dbg.DebugBlockExecution()
	if debugBlock > 0 && debugBlock == header.Number.Uint64() {
		excessBlobGas := uint64(0)
		blobGasUsed := uint64(0)
		if header.ExcessBlobGas != nil {
			excessBlobGas = *header.ExcessBlobGas
		}
		if header.BlobGasUsed != nil {
			blobGasUsed = *header.BlobGasUsed
		}
		fmt.Printf("\n========== DEBUG BLOCK %d ==========\n", header.Number.Uint64())
		fmt.Printf("[DEBUG BLOCK] Block=%d Time=%d ExcessBlobGas=%d BlobGasUsed=%d\n",
			header.Number.Uint64(), header.Time, excessBlobGas, blobGasUsed)
		fmt.Printf("  BaseFee=%s, Hash=%s\n", header.BaseFee.String(), block.Hash().Hex())
		fmt.Printf("  GasLimit=%d, TxCount=%d\n", block.GasLimit(), len(block.Transactions()))
	}

	usedGas := new(uint64)
	usedBlobGas := new(uint64)
	gp := new(GasPool)
	gp.AddGas(block.GasLimit()).AddBlobGas(chainConfig.GetMaxBlobGasPerBlock(block.Time()))

	if err := InitializeBlockExecution(engine, chainReader, block.Header(), chainConfig, ibs, logger); err != nil {
		return nil, err
	}

	var rejectedTxs []*RejectedTx
	includedTxs := make(types.Transactions, 0, block.Transactions().Len())
	receipts := make(types.Receipts, 0, block.Transactions().Len())
	noop := state.NewNoopWriter()
	// Track gas for debugging - store per-tx gas info
	type txGasInfo struct {
		txType  uint8
		gasUsed uint64
	}
	txGasInfos := make([]txGasInfo, 0, block.Transactions().Len())

	// Debug: Track nonce changes for EIP-7702 authority addresses
	authorityToTrack := libcommon.HexToAddress("0x4DE23f3f0Fb3318287378AdbdE030cf61714b2f3")
	if chainConfig.IsPrague(header.Time) {
		fmt.Printf("[NONCE TRACK] Block %d: Initial nonce of %s = %d\n",
			block.NumberU64(), authorityToTrack.Hex(), ibs.GetNonce(authorityToTrack))
	}

	var gasBeforeTx uint64
	for i, tx := range block.Transactions() {
		gasBeforeTx = *usedGas

		// Debug: Check if this tx is from the authority we're tracking
		sender, _ := tx.Sender(*types.LatestSignerForChainID(chainConfig.ChainID))
		nonceBefore := ibs.GetNonce(authorityToTrack)
		if chainConfig.IsPrague(header.Time) && sender == authorityToTrack {
			fmt.Printf("[NONCE TRACK] TX %d: from=%s (AUTHORITY!) nonce=%d, auth_nonce_before=%d\n",
				i, sender.Hex(), tx.GetNonce(), nonceBefore)
		}

		ibs.SetTxContext(tx.Hash(), block.Hash(), i)
		writeTrace := false
		if vmConfig.Debug && vmConfig.Tracer == nil {
			tracer, err := getTracer(i, tx.Hash())
			if err != nil {
				return nil, fmt.Errorf("could not obtain tracer: %w", err)
			}
			vmConfig.Tracer = tracer
			writeTrace = true
		}
		receipt, _, err := ApplyTransaction(chainConfig, blockHashFunc, engine, nil, gp, ibs, noop, header, tx, usedGas, usedBlobGas, *vmConfig)
		if writeTrace {
			if ftracer, ok := vmConfig.Tracer.(vm.FlushableTracer); ok {
				ftracer.Flush(tx)
			}

			vmConfig.Tracer = nil
		}

		// Debug: Check nonce after tx
		nonceAfter := ibs.GetNonce(authorityToTrack)
		if chainConfig.IsPrague(header.Time) && nonceAfter != nonceBefore {
			fmt.Printf("[NONCE TRACK] TX %d: authority nonce changed %d -> %d (tx from %s)\n",
				i, nonceBefore, nonceAfter, sender.Hex())
		}

		if err != nil {
			if !vmConfig.StatelessExec {
				return nil, fmt.Errorf("could not apply tx %d from block %d [%v]: %w", i, block.NumberU64(), tx.Hash().Hex(), err)
			}
			rejectedTxs = append(rejectedTxs, &RejectedTx{i, err.Error()})
		} else {
			includedTxs = append(includedTxs, tx)
			if !vmConfig.NoReceipts {
				receipts = append(receipts, receipt)
			}
			// Store gas info for later debugging
			txGasInfos = append(txGasInfos, txGasInfo{
				txType:  tx.Type(),
				gasUsed: *usedGas - gasBeforeTx,
			})
		}
	}

	receiptSha := types.DeriveSha(receipts)
	if !vmConfig.StatelessExec && chainConfig.IsByzantium(header.Number.Uint64()) && !vmConfig.NoReceipts && receiptSha != block.ReceiptHash() {
		// Always print debug info when receipt mismatch occurs
		fmt.Printf("\n========== RECEIPT MISMATCH BLOCK %d ==========\n", block.NumberU64())
		fmt.Printf("Expected: %s, Got: %s\n", block.ReceiptHash().Hex(), receiptSha.Hex())
		fmt.Printf("GasUsed: execution=%d, header=%d, diff=%d\n", *usedGas, header.GasUsed, int64(*usedGas)-int64(header.GasUsed))
		fmt.Printf("TxCount=%d, ReceiptCount=%d\n", len(block.Transactions()), len(receipts))
		fmt.Printf("IsPrague=%v, IsOsaka=%v, Time=%d\n", chainConfig.IsPrague(header.Time), chainConfig.IsOsaka(header.Time), header.Time)

		// Print all transactions with their gas usage (grouped by type)
		fmt.Printf("\n--- TX Gas Usage by Type ---\n")
		typeGas := make(map[uint8]uint64)
		typeCount := make(map[uint8]int)
		for _, info := range txGasInfos {
			typeGas[info.txType] += info.gasUsed
			typeCount[info.txType]++
		}
		for t := uint8(0); t <= 4; t++ {
			if typeCount[t] > 0 {
				fmt.Printf("Type %d: count=%d, totalGas=%d\n", t, typeCount[t], typeGas[t])
			}
		}

		// Print Type 4 (EIP-7702) transactions with full details
		fmt.Printf("\n--- Type 4 (EIP-7702 SetCode) TXs ---\n")
		for i, tx := range includedTxs {
			if tx.Type() == 4 {
				fmt.Printf("[TX %d] Hash=%s GasUsed=%d DataLen=%d GasLimit=%d\n",
					i, tx.Hash().Hex(), txGasInfos[i].gasUsed, len(tx.GetData()), tx.GetGas())
			}
		}

		// Print first 10 and last 10 transactions
		fmt.Printf("\n--- First 10 TXs ---\n")
		for i := 0; i < min(10, len(txGasInfos)); i++ {
			info := txGasInfos[i]
			fmt.Printf("[TX %d] Type=%d GasUsed=%d\n", i, info.txType, info.gasUsed)
		}
		if len(txGasInfos) > 20 {
			fmt.Printf("... (skipping %d middle TXs) ...\n", len(txGasInfos)-20)
		}
		fmt.Printf("\n--- Last 10 TXs ---\n")
		for i := max(10, len(txGasInfos)-10); i < len(txGasInfos); i++ {
			info := txGasInfos[i]
			fmt.Printf("[TX %d] Type=%d GasUsed=%d\n", i, info.txType, info.gasUsed)
		}

		fmt.Printf("\n========== END DEBUG ==========\n\n")
		if dbg.LogHashMismatchReason() {
			logReceipts(receipts, includedTxs, chainConfig, header, logger)
		}
		return nil, fmt.Errorf("mismatched receipt headers for block %d (%s != %s)", block.NumberU64(), receiptSha.Hex(), block.ReceiptHash().Hex())
	}

	if !vmConfig.StatelessExec && *usedGas != header.GasUsed {
		fmt.Printf("\n========== GAS MISMATCH BLOCK %d ==========\n", block.NumberU64())
		fmt.Printf("GasUsed: execution=%d, header=%d, diff=%d\n", *usedGas, header.GasUsed, int64(*usedGas)-int64(header.GasUsed))
		fmt.Printf("========== END DEBUG ==========\n\n")
		return nil, fmt.Errorf("gas used by execution: %d, in header: %d", *usedGas, header.GasUsed)
	}

	if header.BlobGasUsed != nil && *usedBlobGas != *header.BlobGasUsed {
		return nil, fmt.Errorf("blob gas used by execution: %d, in header: %d", *usedBlobGas, *header.BlobGasUsed)
	}

	var bloom types.Bloom
	if !vmConfig.NoReceipts {
		bloom = types.CreateBloom(receipts)
		if !vmConfig.StatelessExec && bloom != header.Bloom {
			return nil, fmt.Errorf("bloom computed by execution: %x, in header: %x", bloom, header.Bloom)
		}
	}

	if !vmConfig.ReadOnly {
		txs := block.Transactions()
		if _, _, _, _, err := FinalizeBlockExecution(engine, stateReader, block.Header(), txs, block.Uncles(), stateWriter, chainConfig, ibs, receipts, block.Withdrawals(), chainReader, false, logger); err != nil {
			return nil, err
		}
	}
	blockLogs := ibs.Logs()
	execRs := &EphemeralExecResult{
		TxRoot:      types.DeriveSha(includedTxs),
		ReceiptRoot: receiptSha,
		Bloom:       bloom,
		LogsHash:    rlpHash(blockLogs),
		Receipts:    receipts,
		Difficulty:  (*math2.HexOrDecimal256)(header.Difficulty),
		GasUsed:     math.HexOrDecimal64(*usedGas),
		Rejected:    rejectedTxs,
	}

	if chainConfig.Bor != nil {
		var logs []*types.Log
		for _, receipt := range receipts {
			logs = append(logs, receipt.Logs...)
		}

		stateSyncReceipt := &types.Receipt{}
		if chainConfig.Consensus == chain.BorConsensus && len(blockLogs) > 0 {
			slices.SortStableFunc(blockLogs, func(i, j *types.Log) int { return cmp.Compare(i.Index, j.Index) })

			if len(blockLogs) > len(logs) {
				stateSyncReceipt.Logs = blockLogs[len(logs):] // get state-sync logs from `state.Logs()`

				// fill the state sync with the correct information
				bortypes.DeriveFieldsForBorReceipt(stateSyncReceipt, block.Hash(), block.NumberU64(), receipts)
				stateSyncReceipt.Status = types.ReceiptStatusSuccessful
			}
		}

		execRs.StateSyncReceipt = stateSyncReceipt
	}

	return execRs, nil
}

func logReceipts(receipts types.Receipts, txns types.Transactions, cc *chain.Config, header *types.Header, logger log.Logger) {
	if len(receipts) == 0 {
		// no-op, can happen if vmConfig.NoReceipts=true or vmConfig.StatelessExec=true
		return
	}

	// note we do not return errors from this func since this is a debug-only
	// informative feature that is best-effort and should not interfere with execution
	if len(receipts) != len(txns) {
		logger.Error("receipts and txns sizes differ", "receiptsLen", receipts.Len(), "txnsLen", txns.Len())
		return
	}

	marshalled := make([]map[string]interface{}, 0, len(receipts))
	for i, receipt := range receipts {
		txn := txns[i]
		marshalled = append(marshalled, ethutils.MarshalReceipt(receipt, txn, cc, header, txn.Hash(), true))
	}

	result, err := json.Marshal(marshalled)
	if err != nil {
		logger.Error("marshalling error when logging receipts", "err", err)
		return
	}

	logger.Info("marshalled receipts", "result", string(result))
}

func rlpHash(x interface{}) (h libcommon.Hash) {
	hw := sha3.NewLegacyKeccak256()
	rlp.Encode(hw, x) //nolint:errcheck
	hw.Sum(h[:0])
	return h
}

func SysCallContract(contract libcommon.Address, data []byte, chainConfig *chain.Config, ibs *state.IntraBlockState, header *types.Header, engine consensus.EngineReader, constCall bool) (result []byte, err error) {
	msg := types.NewMessage(
		state.SystemAddress,
		&contract,
		0, u256.Num0,
		SysCallGasLimit,
		u256.Num0,
		nil, nil,
		data, nil, false,
		true, // isFree - system calls don't consume block gas
		nil,  // maxFeePerBlobGas
	)
	vmConfig := vm.Config{NoReceipts: true, RestoreState: constCall}
	// Create a new context to be used in the EVM environment
	isBor := chainConfig.Bor != nil
	var txContext evmtypes.TxContext
	var author *libcommon.Address
	if isBor {
		author = &header.Coinbase
		txContext = evmtypes.TxContext{}
	} else {
		author = &state.SystemAddress
		txContext = NewEVMTxContext(msg)
	}
	blockContext := NewEVMBlockContext(header, GetHashFn(header, nil), engine, author, chainConfig)
	evm := vm.NewEVM(blockContext, txContext, ibs, chainConfig, vmConfig)

	ret, _, err := evm.Call(
		vm.AccountRef(msg.From()),
		*msg.To(),
		msg.Data(),
		msg.Gas(),
		msg.Value(),
		false,
	)
	if isBor && err != nil {
		return nil, nil
	}
	return ret, err
}

// SysCreate is a special (system) contract creation methods for genesis constructors.
func SysCreate(contract libcommon.Address, data []byte, chainConfig chain.Config, ibs *state.IntraBlockState, header *types.Header) (result []byte, err error) {
	msg := types.NewMessage(
		contract,
		nil, // to
		0, u256.Num0,
		SysCallGasLimit,
		u256.Num0,
		nil, nil,
		data, nil, false,
		true, // isFree
		nil,  // maxFeePerBlobGas
	)
	vmConfig := vm.Config{NoReceipts: true}
	// Create a new context to be used in the EVM environment
	author := &contract
	txContext := NewEVMTxContext(msg)
	blockContext := NewEVMBlockContext(header, GetHashFn(header, nil), nil, author, &chainConfig)
	evm := vm.NewEVM(blockContext, txContext, ibs, &chainConfig, vmConfig)

	ret, _, err := evm.SysCreate(
		vm.AccountRef(msg.From()),
		msg.Data(),
		msg.Gas(),
		msg.Value(),
		contract,
	)
	return ret, err
}

func FinalizeBlockExecution(
	engine consensus.Engine, stateReader state.StateReader,
	header *types.Header, txs types.Transactions, uncles []*types.Header,
	stateWriter state.WriterWithChangeSets, cc *chain.Config,
	ibs *state.IntraBlockState, receipts types.Receipts,
	withdrawals []*types.Withdrawal, chainReader consensus.ChainReader,
	isMining bool,
	logger log.Logger,
) (newBlock *types.Block, newTxs types.Transactions, newReceipt types.Receipts, retRequests types.FlatRequests, err error) {
	syscall := func(contract libcommon.Address, data []byte) ([]byte, error) {
		return SysCallContract(contract, data, cc, ibs, header, engine, false /* constCall */)
	}

	if isMining {
		newBlock, newTxs, newReceipt, retRequests, err = engine.FinalizeAndAssemble(cc, header, ibs, txs, uncles, receipts, withdrawals, chainReader, syscall, nil, logger)
	} else {
		newTxs, newReceipt, retRequests, err = engine.Finalize(cc, header, ibs, txs, uncles, receipts, withdrawals, chainReader, syscall, logger)
	}
	if err != nil {
		return nil, nil, nil, nil, err
	}

	if err := ibs.CommitBlock(cc.Rules(header.Number.Uint64(), header.Time), stateWriter); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("committing block %d failed: %w", header.Number.Uint64(), err)
	}

	if err := stateWriter.WriteChangeSets(); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("writing changesets for block %d failed: %w", header.Number.Uint64(), err)
	}
	return newBlock, newTxs, newReceipt, retRequests, nil
}

func InitializeBlockExecution(engine consensus.Engine, chain consensus.ChainHeaderReader, header *types.Header,
	cc *chain.Config, ibs *state.IntraBlockState, logger log.Logger,
) error {
	engine.Initialize(cc, chain, header, ibs, func(contract libcommon.Address, data []byte, ibState *state.IntraBlockState, header *types.Header, constCall bool) ([]byte, error) {
		return SysCallContract(contract, data, cc, ibState, header, engine, constCall)
	}, logger)
	noop := state.NewNoopWriter()
	ibs.FinalizeTx(cc.Rules(header.Number.Uint64(), header.Time), noop)
	return nil
}
