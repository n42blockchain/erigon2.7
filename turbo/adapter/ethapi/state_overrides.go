package ethapi

import (
	"fmt"
	"math/big"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/common/hexutil"
	"github.com/holiman/uint256"

	"github.com/erigontech/erigon/core/state"
	"github.com/erigontech/erigon/core/vm/evmtypes"
)

// BlockOverrides contains fields for overriding block context parameters in eth_call
type BlockOverrides struct {
	Number        *hexutil.Uint64   `json:"number,omitempty"`
	Time          *hexutil.Uint64   `json:"time,omitempty"`
	GasLimit      *hexutil.Uint64   `json:"gasLimit,omitempty"`
	FeeRecipient  *libcommon.Address `json:"feeRecipient,omitempty"`
	PrevRandao    *libcommon.Hash   `json:"prevRandao,omitempty"`
	BaseFeePerGas *hexutil.Big      `json:"baseFeePerGas,omitempty"`
	BlobBaseFee   *hexutil.Big      `json:"blobBaseFee,omitempty"`
}

// Apply applies the block overrides to the given block context
func (o *BlockOverrides) Apply(blockCtx *evmtypes.BlockContext) {
	if o == nil {
		return
	}
	if o.Number != nil {
		blockCtx.BlockNumber = uint64(*o.Number)
	}
	if o.Time != nil {
		blockCtx.Time = uint64(*o.Time)
	}
	if o.GasLimit != nil {
		blockCtx.GasLimit = uint64(*o.GasLimit)
	}
	if o.FeeRecipient != nil {
		blockCtx.Coinbase = *o.FeeRecipient
	}
	if o.BaseFeePerGas != nil {
		blockCtx.BaseFee, _ = uint256.FromBig(o.BaseFeePerGas.ToInt())
	}
	if o.PrevRandao != nil {
		blockCtx.PrevRanDao = o.PrevRandao
	}
	// Note: BlobBaseFee would need to be applied if the block context has a field for it
}

type StateOverrides map[libcommon.Address]Account

func (overrides *StateOverrides) Override(state *state.IntraBlockState) error {

	for addr, account := range *overrides {
		// Override account nonce.
		if account.Nonce != nil {
			state.SetNonce(addr, uint64(*account.Nonce))
		}
		// Override account(contract) code.
		if account.Code != nil {
			state.SetCode(addr, *account.Code)
		}
		// Override account balance.
		if account.Balance != nil {
			balance, overflow := uint256.FromBig((*big.Int)(*account.Balance))
			if overflow {
				return fmt.Errorf("account.Balance higher than 2^256-1")
			}
			state.SetBalance(addr, balance)
		}
		if account.State != nil && account.StateDiff != nil {
			return fmt.Errorf("account %s has both 'state' and 'stateDiff'", addr.Hex())
		}
		// Replace entire state if caller requires.
		if account.State != nil {
			intState := map[libcommon.Hash]uint256.Int{}
			for key, value := range *account.State {
				intValue := new(uint256.Int).SetBytes32(value.Bytes())
				intState[key] = *intValue
			}
			state.SetStorage(addr, intState)
		}
		// Apply state diff into specified accounts.
		if account.StateDiff != nil {
			for key, value := range *account.StateDiff {
				key := key
				intValue := new(uint256.Int).SetBytes32(value.Bytes())
				state.SetState(addr, &key, *intValue)
			}
		}
	}

	return nil
}
