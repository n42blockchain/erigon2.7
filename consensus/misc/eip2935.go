package misc

import (
	"github.com/erigontech/erigon-lib/log/v3"
	"github.com/holiman/uint256"

	"github.com/erigontech/erigon-lib/chain"
	libcommon "github.com/erigontech/erigon-lib/common"

	"github.com/erigontech/erigon/consensus"
	"github.com/erigontech/erigon/core/state"
	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/params"
)

// DeployPragueSystemContracts deploys all Prague system contracts at fork transition
// This includes EIP-2935 (historical block hashes), EIP-7002 (withdrawals), and EIP-7251 (consolidations)
func DeployPragueSystemContracts(state *state.IntraBlockState, logger log.Logger) {
	// Deploy EIP-2935 history storage contract
	deploySystemContract(state, params.HistoryStorageAddress, params.HistoryStorageCode, "EIP-2935 History Storage", logger)

	// Note: EIP-7002 and EIP-7251 contracts are deployed via separate mechanisms
	// They may use deployment transactions or be injected differently
	// For now, we focus on EIP-2935 which is the main blocker
}

// deploySystemContract safely deploys a system contract at fork transition
// It preserves any existing balance at the address (for nonzero_balance test cases)
func deploySystemContract(state *state.IntraBlockState, address libcommon.Address, code []byte, name string, logger log.Logger) {
	// Check if contract already has code
	existingCodeSize := state.GetCodeSize(address)
	if existingCodeSize > 0 {
		logger.Debug("System contract already has code, skipping deployment", "contract", name, "address", address.Hex(), "codeSize", existingCodeSize)
		return
	}

	// Ensure account exists (this preserves existing balance if any)
	if !state.Exist(address) {
		state.CreateAccount(address, true)
	}

	// Set the contract code
	state.SetCode(address, code)

	// Set nonce to 1 (standard for deployed contracts)
	state.SetNonce(address, 1)

	logger.Info("Deployed system contract at fork transition", "contract", name, "address", address.Hex(), "codeSize", len(code))
}

// EnsureEip2935ContractDeployed ensures the EIP-2935 history storage contract is deployed
// DEPRECATED: Use DeployPragueSystemContracts instead for fork transition
// This function is kept for reference but should not be called directly
func EnsureEip2935ContractDeployed(state *state.IntraBlockState) {
	// Check if contract already has code
	if state.GetCodeSize(params.HistoryStorageAddress) > 0 {
		return // Already deployed
	}

	// Deploy the system contract
	log.Info("[EIP-2935] Deploying history storage system contract", "address", params.HistoryStorageAddress.Hex())
	state.CreateAccount(params.HistoryStorageAddress, true)
	state.SetCode(params.HistoryStorageAddress, params.HistoryStorageCode)
	state.SetNonce(params.HistoryStorageAddress, 1)
}

func StoreBlockHashesEip2935(header *types.Header, state *state.IntraBlockState, config *chain.Config, headerReader consensus.ChainHeaderReader) {
	if state.GetCodeSize(params.HistoryStorageAddress) == 0 {
		log.Debug("[EIP-2935] No code deployed to HistoryStorageAddress before call to store EIP-2935 history")
		return
	}
	headerNum := header.Number.Uint64()
	if headerNum == 0 { // Activation of fork at Genesis
		return
	}
	storeHash(headerNum-1, header.ParentHash, state)
}

func storeHash(num uint64, hash libcommon.Hash, state *state.IntraBlockState) {
	slotNum := num % params.BlockHashHistoryServeWindow
	storageSlot := libcommon.BytesToHash(uint256.NewInt(slotNum).Bytes())
	parentHashInt := uint256.NewInt(0).SetBytes32(hash.Bytes())
	state.SetState(params.HistoryStorageAddress, &storageSlot, *parentHashInt)
}
