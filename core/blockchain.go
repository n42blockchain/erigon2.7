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
	"bytes"
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
	if dbg.DebugBlockExecution() == header.Number.Uint64() {
		excessBlobGas := uint64(0)
		blobGasUsed := uint64(0)
		if header.ExcessBlobGas != nil {
			excessBlobGas = *header.ExcessBlobGas
		}
		if header.BlobGasUsed != nil {
			blobGasUsed = *header.BlobGasUsed
		}
		fmt.Printf("[DEBUG BLOCK] Block=%d Time=%d ExcessBlobGas=%d BlobGasUsed=%d\n",
			header.Number.Uint64(), header.Time, excessBlobGas, blobGasUsed)
		fmt.Printf("  BaseFee=%s, Hash=%s\n", header.BaseFee.String(), block.Hash().Hex())
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
	for i, tx := range block.Transactions() {
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
		}
	}

	// Execute FinalizeBlockExecution first to get system call gas (for Prague+)
	// In Prague and later, system calls (EIP-7002, EIP-7251) consume gas that should be counted towards GasUsed
	var syscallGasUsed uint64
	if !vmConfig.ReadOnly {
		txs := block.Transactions()
		var err error
		syscallGasUsed, _, _, _, _, err = FinalizeBlockExecutionWithSyscallGas(engine, stateReader, block.Header(), txs, block.Uncles(), stateWriter, chainConfig, ibs, receipts, block.Withdrawals(), chainReader, false, logger)
		if err != nil {
			return nil, err
		}
		// Add system call gas to usedGas for Prague and later
		if syscallGasUsed > 0 {
			*usedGas += syscallGasUsed
			if dbg.DebugBlockExecution() == header.Number.Uint64() {
				fmt.Printf("[DEBUG SYSCALL GAS] Block=%d SyscallGasUsed=%d TotalUsedGas=%d\n",
					header.Number.Uint64(), syscallGasUsed, *usedGas)
			}
		}
	}

	receiptSha := types.DeriveSha(receipts)
	if !vmConfig.StatelessExec && chainConfig.IsByzantium(header.Number.Uint64()) && !vmConfig.NoReceipts && receiptSha != block.ReceiptHash() {
		if dbg.LogHashMismatchReason() {
			logReceipts(receipts, includedTxs, chainConfig, header, logger)
		}

		// Debug: Print detailed receipt information for mismatch analysis
		rules := chainConfig.Rules(header.Number.Uint64(), header.Time)
		fmt.Printf("\n========== RECEIPT HASH MISMATCH DEBUG ==========\n")
		fmt.Printf("Block: %d, Time: %d, Hash: %s\n", block.NumberU64(), header.Time, block.Hash().Hex())
		fmt.Printf("Rules: IsPrague=%v, IsOsaka=%v, IsCancun=%v\n", rules.IsPrague, rules.IsOsaka, rules.IsCancun)
		fmt.Printf("Computed ReceiptHash: %s\n", receiptSha.Hex())
		fmt.Printf("Expected ReceiptHash: %s\n", block.ReceiptHash().Hex())
		fmt.Printf("Total Receipts: %d, Total Txs: %d\n", len(receipts), len(includedTxs))
		fmt.Printf("**GAS CHECK**: usedGas=%d (txGas=%d + syscallGas=%d), header.GasUsed=%d, diff=%d\n",
			*usedGas, *usedGas-syscallGasUsed, syscallGasUsed, header.GasUsed, int64(*usedGas)-int64(header.GasUsed))
		fmt.Printf("\n--- Per-Transaction Receipt Details ---\n")
		for i, receipt := range receipts {
			tx := includedTxs[i]
			fmt.Printf("Tx[%d] Hash: %s\n", i, tx.Hash().Hex())
			fmt.Printf("  Type: %d, Status: %d, GasUsed: %d, CumulativeGas: %d\n",
				receipt.Type, receipt.Status, receipt.GasUsed, receipt.CumulativeGasUsed)
			fmt.Printf("  LogsCount: %d, Bloom(first8): %x\n", len(receipt.Logs), receipt.Bloom[:8])
			for j, lg := range receipt.Logs {
				fmt.Printf("    Log[%d]: Addr=%s, Topics=%d, DataLen=%d\n",
					j, lg.Address.Hex(), len(lg.Topics), len(lg.Data))
			}
			// Calculate individual receipt hash for debugging
			var buf bytes.Buffer
			receipts[i:i+1].EncodeIndex(0, &buf)
			fmt.Printf("  ReceiptRLP(first32): %x\n", buf.Bytes()[:min(32, buf.Len())])
		}
		fmt.Printf("========== END DEBUG ==========\n\n")

		return nil, fmt.Errorf("mismatched receipt headers for block %d (%s != %s)", block.NumberU64(), receiptSha.Hex(), block.ReceiptHash().Hex())
	}

	if !vmConfig.StatelessExec && *usedGas != header.GasUsed {
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
	result, _, err = SysCallContractWithGas(contract, data, chainConfig, ibs, header, engine, constCall)
	return
}

// SysCallContractWithGas is like SysCallContract but also returns gas used.
func SysCallContractWithGas(contract libcommon.Address, data []byte, chainConfig *chain.Config, ibs *state.IntraBlockState, header *types.Header, engine consensus.EngineReader, constCall bool) (result []byte, gasUsed uint64, err error) {
	msg := types.NewMessage(
		state.SystemAddress,
		&contract,
		0, u256.Num0,
		SysCallGasLimit,
		u256.Num0,
		nil, nil,
		data, nil, false,
		true, // isFree
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

	ret, leftOverGas, err := evm.Call(
		vm.AccountRef(msg.From()),
		*msg.To(),
		msg.Data(),
		msg.Gas(),
		msg.Value(),
		false,
	)
	if isBor && err != nil {
		return nil, 0, nil
	}
	gasUsed = msg.Gas() - leftOverGas
	return ret, gasUsed, err
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
	_, newBlock, newTxs, newReceipt, retRequests, err = FinalizeBlockExecutionWithSyscallGas(
		engine, stateReader, header, txs, uncles, stateWriter, cc, ibs, receipts, withdrawals, chainReader, isMining, logger,
	)
	return
}

// FinalizeBlockExecutionWithSyscallGas is like FinalizeBlockExecution but also returns gas used by system calls.
// In Prague and later, system calls (EIP-7002 withdrawal requests, EIP-7251 consolidation requests) consume gas
// that should be counted towards the block's GasUsed.
func FinalizeBlockExecutionWithSyscallGas(
	engine consensus.Engine, stateReader state.StateReader,
	header *types.Header, txs types.Transactions, uncles []*types.Header,
	stateWriter state.WriterWithChangeSets, cc *chain.Config,
	ibs *state.IntraBlockState, receipts types.Receipts,
	withdrawals []*types.Withdrawal, chainReader consensus.ChainReader,
	isMining bool,
	logger log.Logger,
) (syscallGasUsed uint64, newBlock *types.Block, newTxs types.Transactions, newReceipt types.Receipts, retRequests types.FlatRequests, err error) {
	var totalSyscallGas uint64
	syscall := func(contract libcommon.Address, data []byte) ([]byte, error) {
		result, gasUsed, err := SysCallContractWithGas(contract, data, cc, ibs, header, engine, false /* constCall */)
		if err == nil && (cc.IsPrague(header.Time) || cc.IsOsaka(header.Time)) {
			totalSyscallGas += gasUsed
		}
		return result, err
	}
	if isMining {
		newBlock, newTxs, newReceipt, retRequests, err = engine.FinalizeAndAssemble(cc, header, ibs, txs, uncles, receipts, withdrawals, chainReader, syscall, nil, logger)
	} else {
		newTxs, newReceipt, retRequests, err = engine.Finalize(cc, header, ibs, txs, uncles, receipts, withdrawals, chainReader, syscall, logger)
	}
	if err != nil {
		return 0, nil, nil, nil, nil, err
	}

	if err := ibs.CommitBlock(cc.Rules(header.Number.Uint64(), header.Time), stateWriter); err != nil {
		return 0, nil, nil, nil, nil, fmt.Errorf("committing block %d failed: %w", header.Number.Uint64(), err)
	}

	if err := stateWriter.WriteChangeSets(); err != nil {
		return 0, nil, nil, nil, nil, fmt.Errorf("writing changesets for block %d failed: %w", header.Number.Uint64(), err)
	}
	return totalSyscallGas, newBlock, newTxs, newReceipt, retRequests, nil
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
