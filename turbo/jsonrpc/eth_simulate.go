package jsonrpc

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/holiman/uint256"

	"github.com/erigontech/erigon-lib/chain"
	"github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/common/hexutil"
	"github.com/erigontech/erigon-lib/common/hexutility"
	"github.com/erigontech/erigon-lib/common/math"
	"github.com/erigontech/erigon-lib/log/v3"
	types2 "github.com/erigontech/erigon-lib/types"

	"github.com/erigontech/erigon/core"
	"github.com/erigontech/erigon/core/state"
	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/core/vm"
	"github.com/erigontech/erigon/core/vm/evmtypes"
	"github.com/erigontech/erigon/rpc"
	"github.com/erigontech/erigon/turbo/adapter/ethapi"
	"github.com/erigontech/erigon/turbo/rpchelper"
)

// Maximum number of block state calls that can be simulated
const maxSimulateBlocks = 16

// SimulateBlockOverrides contains fields to override block parameters in simulation
type SimulateBlockOverrides struct {
	Number        *hexutil.Uint64 `json:"number,omitempty"`
	Time          *hexutil.Uint64 `json:"time,omitempty"`
	GasLimit      *hexutil.Uint64 `json:"gasLimit,omitempty"`
	FeeRecipient  *common.Address `json:"feeRecipient,omitempty"`
	PrevRandao    *common.Hash    `json:"prevRandao,omitempty"`
	BaseFeePerGas *hexutil.Big    `json:"baseFeePerGas,omitempty"`
	BlobBaseFee   *hexutil.Big    `json:"blobBaseFee,omitempty"`
}

// SimulateCall represents a single call within a block simulation
type SimulateCall struct {
	From                 *common.Address   `json:"from,omitempty"`
	To                   *common.Address   `json:"to,omitempty"`
	Gas                  *hexutil.Uint64   `json:"gas,omitempty"`
	GasPrice             *hexutil.Big      `json:"gasPrice,omitempty"`
	MaxFeePerGas         *hexutil.Big      `json:"maxFeePerGas,omitempty"`
	MaxPriorityFeePerGas *hexutil.Big      `json:"maxPriorityFeePerGas,omitempty"`
	Value                *hexutil.Big      `json:"value,omitempty"`
	Nonce                *hexutil.Uint64   `json:"nonce,omitempty"`
	Input                *hexutility.Bytes `json:"input,omitempty"`
	Data                 *hexutility.Bytes `json:"data,omitempty"`
	AccessList           *types2.AccessList `json:"accessList,omitempty"`
	ChainID              *hexutil.Big      `json:"chainId,omitempty"`
}

// SimulateBlockStateCall represents a block with state overrides and calls
type SimulateBlockStateCall struct {
	BlockOverrides *SimulateBlockOverrides `json:"blockOverrides,omitempty"`
	StateOverrides *ethapi.StateOverrides  `json:"stateOverrides,omitempty"`
	Calls          []SimulateCall          `json:"calls"`
}

// SimulateCallResult represents the result of a single call
type SimulateCallResult struct {
	ReturnData hexutility.Bytes `json:"returnData"`
	Logs       []*types.Log     `json:"logs"`
	GasUsed    hexutil.Uint64   `json:"gasUsed"`
	Status     hexutil.Uint64   `json:"status"`
	Error      *SimulateError   `json:"error,omitempty"`
}

// SimulateError represents an error in simulation
type SimulateError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// SimulateBlockResult represents the result of simulating a single block
type SimulateBlockResult struct {
	Number        hexutil.Uint64       `json:"number"`
	Hash          common.Hash          `json:"hash"`
	Timestamp     hexutil.Uint64       `json:"timestamp"`
	GasLimit      hexutil.Uint64       `json:"gasLimit"`
	GasUsed       hexutil.Uint64       `json:"gasUsed"`
	FeeRecipient  common.Address       `json:"feeRecipient"`
	BaseFeePerGas *hexutil.Big         `json:"baseFeePerGas,omitempty"`
	PrevRandao    common.Hash          `json:"prevRandao"`
	Calls         []SimulateCallResult `json:"calls"`
}

// SimulateV1 implements eth_simulateV1
// See: https://github.com/ethereum/execution-apis/pull/484
func (api *APIImpl) SimulateV1(
	ctx context.Context,
	blockStateCalls []SimulateBlockStateCall,
	blockNrOrHash *rpc.BlockNumberOrHash,
	returnFullTransactionObjects *bool,
	traceTransfers *bool,
	validation *bool,
) ([]SimulateBlockResult, error) {
	if len(blockStateCalls) == 0 {
		return nil, errors.New("empty blockStateCalls")
	}
	if len(blockStateCalls) > maxSimulateBlocks {
		return nil, fmt.Errorf("too many blocks to simulate, max is %d", maxSimulateBlocks)
	}

	tx, err := api.db.BeginRo(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	chainConfig, err := api.chainConfig(ctx, tx)
	if err != nil {
		return nil, err
	}

	// Determine the base block
	var blockNum uint64
	var hash common.Hash
	if blockNrOrHash != nil {
		blockNum, hash, _, err = rpchelper.GetBlockNumber(*blockNrOrHash, tx, api.filters)
	} else {
		blockNum, hash, _, err = rpchelper.GetBlockNumber(latestNumOrHash, tx, api.filters)
	}
	if err != nil {
		return nil, err
	}

	block, err := api.blockWithSenders(ctx, tx, hash, blockNum)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block %d not found", blockNum)
	}

	header := block.Header()

	stateReader, err := rpchelper.CreateStateReader(
		ctx, tx,
		rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(blockNum)),
		0, api.filters, api.stateCache, api.historyV3(tx), chainConfig.ChainName,
	)
	if err != nil {
		return nil, err
	}

	st := state.New(stateReader)

	// Setup timeout
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Initialize block context from the base block
	getHash := func(i uint64) common.Hash {
		h, err := api._blockReader.CanonicalHash(ctx, tx, i)
		if err != nil {
			log.Debug("Can't get block hash by number", "number", i)
		}
		return h
	}

	blockCtx := core.NewEVMBlockContext(header, getHash, api.engine(), nil, chainConfig)
	txCtx := evmtypes.TxContext{}
	evm := vm.NewEVM(blockCtx, txCtx, st, chainConfig, vm.Config{Debug: false, NoBaseFee: true})

	// Cancel handler
	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()

	results := make([]SimulateBlockResult, 0, len(blockStateCalls))
	currentBlockNum := blockNum + 1
	currentTime := header.Time + chainConfig.SecondsPerSlot()

	for _, blockStateCall := range blockStateCalls {
		// Apply block overrides
		if blockStateCall.BlockOverrides != nil {
			applySimulateBlockOverrides(&blockCtx, blockStateCall.BlockOverrides, currentBlockNum, currentTime)
		} else {
			blockCtx.BlockNumber = currentBlockNum
			blockCtx.Time = currentTime
		}

		// Apply state overrides
		if blockStateCall.StateOverrides != nil {
			if err := blockStateCall.StateOverrides.Override(evm.IntraBlockState().(*state.IntraBlockState)); err != nil {
				return nil, err
			}
		}

		gp := new(core.GasPool).AddGas(math.MaxUint64).AddBlobGas(math.MaxUint64)
		rules := chainConfig.Rules(blockCtx.BlockNumber, blockCtx.Time)

		blockResult := SimulateBlockResult{
			Number:       hexutil.Uint64(blockCtx.BlockNumber),
			Timestamp:    hexutil.Uint64(blockCtx.Time),
			GasLimit:     hexutil.Uint64(blockCtx.GasLimit),
			FeeRecipient: blockCtx.Coinbase,
			PrevRandao:   common.Hash{}, // Would need PrevRandao from context
			Calls:        make([]SimulateCallResult, 0, len(blockStateCall.Calls)),
		}

		if blockCtx.BaseFee != nil {
			baseFee := new(big.Int).Set(blockCtx.BaseFee.ToBig())
			blockResult.BaseFeePerGas = (*hexutil.Big)(baseFee)
		}

		var totalGasUsed uint64

		for _, call := range blockStateCall.Calls {
			callResult, gasUsed, err := executeSimulateCall(ctx, evm, &blockCtx, gp, call, api.GasCap, rules)
			if err != nil {
				return nil, err
			}

			if evm.Cancelled() {
				return nil, fmt.Errorf("execution aborted (timeout)")
			}

			totalGasUsed += gasUsed
			blockResult.Calls = append(blockResult.Calls, callResult)

			_ = st.FinalizeTx(rules, state.NewNoopWriter())
		}

		blockResult.GasUsed = hexutil.Uint64(totalGasUsed)
		// Generate a pseudo block hash based on block number
		blockResult.Hash = common.BytesToHash(common.BigToHash(big.NewInt(int64(blockCtx.BlockNumber))).Bytes())

		results = append(results, blockResult)

		// Prepare for next block
		currentBlockNum = blockCtx.BlockNumber + 1
		currentTime = blockCtx.Time + chainConfig.SecondsPerSlot()
	}

	return results, nil
}

// applySimulateBlockOverrides applies block overrides to the block context
func applySimulateBlockOverrides(blockCtx *evmtypes.BlockContext, overrides *SimulateBlockOverrides, defaultNum uint64, defaultTime uint64) {
	if overrides.Number != nil {
		blockCtx.BlockNumber = uint64(*overrides.Number)
	} else {
		blockCtx.BlockNumber = defaultNum
	}

	if overrides.Time != nil {
		blockCtx.Time = uint64(*overrides.Time)
	} else {
		blockCtx.Time = defaultTime
	}

	if overrides.GasLimit != nil {
		blockCtx.GasLimit = uint64(*overrides.GasLimit)
	}

	if overrides.FeeRecipient != nil {
		blockCtx.Coinbase = *overrides.FeeRecipient
	}

	if overrides.BaseFeePerGas != nil {
		blockCtx.BaseFee, _ = uint256.FromBig(overrides.BaseFeePerGas.ToInt())
	}

	if overrides.PrevRandao != nil {
		blockCtx.PrevRanDao = overrides.PrevRandao
	}
}

// executeSimulateCall executes a single call in the simulation
func executeSimulateCall(
	ctx context.Context,
	evm *vm.EVM,
	blockCtx *evmtypes.BlockContext,
	gp *core.GasPool,
	call SimulateCall,
	gasCap uint64,
	rules *chain.Rules,
) (SimulateCallResult, uint64, error) {
	result := SimulateCallResult{
		Logs: make([]*types.Log, 0),
	}

	// Convert SimulateCall to message
	msg, err := simulateCallToMessage(call, blockCtx.BaseFee, gasCap)
	if err != nil {
		result.Error = &SimulateError{
			Code:    -32000,
			Message: err.Error(),
		}
		return result, 0, nil
	}

	txCtx := core.NewEVMTxContext(msg)
	evm.Reset(txCtx, evm.IntraBlockState())

	execResult, err := core.ApplyMessage(evm, msg, gp, true, false)
	if err != nil {
		result.Error = &SimulateError{
			Code:    -32000,
			Message: err.Error(),
		}
		return result, 0, nil
	}

	result.GasUsed = hexutil.Uint64(execResult.UsedGas)

	if execResult.Err != nil {
		result.Status = 0
		result.Error = &SimulateError{
			Code:    3, // Execution reverted
			Message: execResult.Err.Error(),
		}
		if len(execResult.Revert()) > 0 {
			result.Error.Data = hexutility.Encode(execResult.Revert())
		}
	} else {
		result.Status = 1
	}

	result.ReturnData = execResult.Return()
	// Note: logs would need to be collected from the state

	return result, execResult.UsedGas, nil
}

// simulateCallToMessage converts a SimulateCall to a types.Message
func simulateCallToMessage(call SimulateCall, baseFee *uint256.Int, gasCap uint64) (types.Message, error) {
	var from common.Address
	if call.From != nil {
		from = *call.From
	}

	var to *common.Address
	if call.To != nil {
		to = call.To
	}

	var gas uint64
	if call.Gas != nil {
		gas = uint64(*call.Gas)
	} else {
		gas = gasCap
	}

	var value *uint256.Int
	if call.Value != nil {
		value, _ = uint256.FromBig(call.Value.ToInt())
	} else {
		value = uint256.NewInt(0)
	}

	var data []byte
	if call.Input != nil {
		data = *call.Input
	} else if call.Data != nil {
		data = *call.Data
	}

	var gasPrice *uint256.Int
	var gasFeeCap *uint256.Int
	var gasTipCap *uint256.Int

	if call.GasPrice != nil {
		gasPrice, _ = uint256.FromBig(call.GasPrice.ToInt())
		gasFeeCap = gasPrice
		gasTipCap = gasPrice
	} else {
		if call.MaxFeePerGas != nil {
			gasFeeCap, _ = uint256.FromBig(call.MaxFeePerGas.ToInt())
		} else if baseFee != nil {
			gasFeeCap = baseFee.Clone()
		} else {
			gasFeeCap = uint256.NewInt(0)
		}

		if call.MaxPriorityFeePerGas != nil {
			gasTipCap, _ = uint256.FromBig(call.MaxPriorityFeePerGas.ToInt())
		} else {
			gasTipCap = uint256.NewInt(0)
		}
	}

	var accessList types2.AccessList
	if call.AccessList != nil {
		accessList = *call.AccessList
	}

	msg := types.NewMessage(
		from,
		to,
		0, // nonce - will be set by state if needed
		value,
		gas,
		gasPrice,
		gasFeeCap,
		gasTipCap,
		data,
		accessList,
		false, // checkNonce
		false, // isFree
		nil,   // maxFeePerBlobGas
	)

	return msg, nil
}
