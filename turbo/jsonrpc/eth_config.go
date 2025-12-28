package jsonrpc

import (
	"context"
	"math/big"

	"github.com/erigontech/erigon-lib/chain"
	"github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/common/hexutil"
	"github.com/erigontech/erigon/core/rawdb"
)

// EIP-7910: eth_config JSON-RPC method
// This method provides configuration information about the current and upcoming hard forks,
// allowing node operators and monitoring tools to verify client readiness.

// ForkConfig represents the configuration of a single hard fork
type ForkConfig struct {
	Name       string                    `json:"name"`                 // Fork name (e.g., "Cancun", "Prague", "Osaka")
	Block      *hexutil.Big              `json:"block,omitempty"`      // Activation block number (for block-based forks)
	Time       *hexutil.Big              `json:"time,omitempty"`       // Activation timestamp (for time-based forks)
	Active     bool                      `json:"active"`               // Whether the fork is currently active
	EIPs       []int                     `json:"eips,omitempty"`       // EIPs included in this fork
	Parameters map[string]interface{}    `json:"parameters,omitempty"` // Fork-specific parameters
}

// ConfigResponse is the return type of eth_config
type ConfigResponse struct {
	ChainID          hexutil.Uint64     `json:"chainId"`
	ChainName        string             `json:"chainName,omitempty"`
	Consensus        string             `json:"consensus"`
	DepositContract  *common.Address    `json:"depositContract,omitempty"`
	CurrentBlock     hexutil.Uint64     `json:"currentBlock"`
	CurrentTime      hexutil.Uint64     `json:"currentTime"`
	CurrentFork      *ForkConfig        `json:"currentFork"`
	NextFork         *ForkConfig        `json:"nextFork,omitempty"`
	Forks            []*ForkConfig      `json:"forks"`
	Precompiles      map[string]string  `json:"precompiles"`
	BlobParameters   *BlobParametersConfig `json:"blobParameters,omitempty"`
}

// BlobParametersConfig contains blob-related configuration
type BlobParametersConfig struct {
	TargetBlobsPerBlock      uint64 `json:"targetBlobsPerBlock"`
	MaxBlobsPerBlock         uint64 `json:"maxBlobsPerBlock"`
	BaseFeeUpdateFraction    uint64 `json:"baseFeeUpdateFraction"`
	MinBlobGasPrice          uint64 `json:"minBlobGasPrice"`
}

// Config implements eth_config. Returns the current and upcoming fork configuration.
// This is defined in EIP-7910.
func (api *APIImpl) Config(ctx context.Context) (*ConfigResponse, error) {
	tx, err := api.db.BeginRo(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	chainConfig, err := api.chainConfig(ctx, tx)
	if err != nil {
		return nil, err
	}

	// Get current block and time
	header := rawdb.ReadCurrentHeader(tx)
	if header == nil {
		return nil, nil
	}

	currentBlock := header.Number.Uint64()
	currentTime := header.Time

	// Build fork list and determine current/next fork
	forks := buildForkList(chainConfig, currentBlock, currentTime)
	currentFork, nextFork := determineCurrentAndNextFork(forks, currentBlock, currentTime)

	// Get deposit contract address
	var depositContract *common.Address
	if chainConfig.DepositContract != (common.Address{}) {
		depositContract = &chainConfig.DepositContract
	}

	// Build precompiles map
	precompiles := buildPrecompilesMap(chainConfig, currentTime)

	// Build blob parameters if applicable
	var blobParams *BlobParametersConfig
	if chainConfig.IsCancun(currentTime) {
		blobParams = &BlobParametersConfig{
			TargetBlobsPerBlock:   chainConfig.GetTargetBlobsPerBlock(currentTime),
			MaxBlobsPerBlock:      chainConfig.GetMaxBlobsPerBlock(currentTime),
			BaseFeeUpdateFraction: chainConfig.GetBlobGasPriceUpdateFraction(currentTime),
			MinBlobGasPrice:       chainConfig.GetMinBlobGasPrice(),
		}
	}

	// Determine consensus type
	consensus := getConsensusType(chainConfig)

	return &ConfigResponse{
		ChainID:         hexutil.Uint64(chainConfig.ChainID.Uint64()),
		ChainName:       chainConfig.ChainName,
		Consensus:       consensus,
		DepositContract: depositContract,
		CurrentBlock:    hexutil.Uint64(currentBlock),
		CurrentTime:     hexutil.Uint64(currentTime),
		CurrentFork:     currentFork,
		NextFork:        nextFork,
		Forks:           forks,
		Precompiles:     precompiles,
		BlobParameters:  blobParams,
	}, nil
}

// buildForkList constructs the list of all forks with their activation status
func buildForkList(cfg *chain.Config, currentBlock, currentTime uint64) []*ForkConfig {
	forks := make([]*ForkConfig, 0)

	// Block-based forks (pre-merge)
	addBlockFork := func(name string, block *big.Int, eips []int) {
		if block != nil {
			active := block.Uint64() <= currentBlock
			forks = append(forks, &ForkConfig{
				Name:   name,
				Block:  (*hexutil.Big)(block),
				Active: active,
				EIPs:   eips,
			})
		}
	}

	// Time-based forks (post-merge)
	addTimeFork := func(name string, time *big.Int, eips []int, params map[string]interface{}) {
		if time != nil {
			active := time.Uint64() <= currentTime
			forks = append(forks, &ForkConfig{
				Name:       name,
				Time:       (*hexutil.Big)(time),
				Active:     active,
				EIPs:       eips,
				Parameters: params,
			})
		}
	}

	// Pre-merge block-based forks
	addBlockFork("Homestead", cfg.HomesteadBlock, []int{2, 7, 8})
	addBlockFork("DAO Fork", cfg.DAOForkBlock, nil)
	addBlockFork("Tangerine Whistle", cfg.TangerineWhistleBlock, []int{150})
	addBlockFork("Spurious Dragon", cfg.SpuriousDragonBlock, []int{155, 160, 161, 170})
	addBlockFork("Byzantium", cfg.ByzantiumBlock, []int{100, 140, 196, 197, 198, 211, 214, 649, 658})
	addBlockFork("Constantinople", cfg.ConstantinopleBlock, []int{145, 1014, 1052, 1234, 1283})
	addBlockFork("Petersburg", cfg.PetersburgBlock, nil) // Removes EIP-1283
	addBlockFork("Istanbul", cfg.IstanbulBlock, []int{152, 1108, 1344, 1884, 2028, 2200})
	addBlockFork("Muir Glacier", cfg.MuirGlacierBlock, []int{2384})
	addBlockFork("Berlin", cfg.BerlinBlock, []int{2565, 2718, 2929, 2930})
	addBlockFork("London", cfg.LondonBlock, []int{1559, 3198, 3529, 3541, 3554})
	addBlockFork("Arrow Glacier", cfg.ArrowGlacierBlock, []int{4345})
	addBlockFork("Gray Glacier", cfg.GrayGlacierBlock, []int{5133})

	// The Merge (Paris)
	if cfg.TerminalTotalDifficulty != nil {
		forks = append(forks, &ForkConfig{
			Name:   "Paris (The Merge)",
			Active: cfg.TerminalTotalDifficultyPassed,
			EIPs:   []int{3675, 4399},
			Parameters: map[string]interface{}{
				"terminalTotalDifficulty": cfg.TerminalTotalDifficulty.String(),
			},
		})
	}

	// Post-merge time-based forks
	addTimeFork("Shanghai", cfg.ShanghaiTime, []int{3651, 3855, 3860, 4895}, nil)
	addTimeFork("Cancun", cfg.CancunTime, []int{1153, 4788, 4844, 5656, 6780, 7516}, nil)

	// Prague (Pectra)
	if cfg.PragueTime != nil {
		pragueParams := map[string]interface{}{
			"maxBlobsPerBlock":   cfg.BlobSchedule.MaxBlobsPerBlock(true, false),
			"targetBlobsPerBlock": cfg.BlobSchedule.TargetBlobsPerBlock(true, false),
		}
		addTimeFork("Prague", cfg.PragueTime, []int{
			2537, 2935, 6110, 7002, 7251, 7549, 7623, 7685, 7691, 7702, 7840,
		}, pragueParams)
	}

	// Osaka (Fusaka)
	if cfg.OsakaTime != nil {
		osakaParams := map[string]interface{}{
			"maxBlobsPerBlock":   cfg.BlobSchedule.MaxBlobsPerBlock(true, true),
			"targetBlobsPerBlock": cfg.BlobSchedule.TargetBlobsPerBlock(true, true),
		}
		addTimeFork("Osaka", cfg.OsakaTime, []int{
			7594, 7642, 7823, 7825, 7883, 7892, 7910,
		}, osakaParams)
	}

	return forks
}

// determineCurrentAndNextFork finds the current active fork and the next scheduled fork
func determineCurrentAndNextFork(forks []*ForkConfig, currentBlock, currentTime uint64) (*ForkConfig, *ForkConfig) {
	var currentFork, nextFork *ForkConfig

	for i := len(forks) - 1; i >= 0; i-- {
		fork := forks[i]
		if fork.Active {
			if currentFork == nil {
				currentFork = fork
			}
		} else if nextFork == nil {
			nextFork = fork
		}
	}

	return currentFork, nextFork
}

// buildPrecompilesMap returns a map of active precompile addresses
func buildPrecompilesMap(cfg *chain.Config, currentTime uint64) map[string]string {
	precompiles := map[string]string{
		"0x01": "ecRecover",
		"0x02": "sha256",
		"0x03": "ripemd160",
		"0x04": "identity",
	}

	// Byzantium (EIP-198)
	precompiles["0x05"] = "modexp"
	precompiles["0x06"] = "bn256Add"
	precompiles["0x07"] = "bn256ScalarMul"
	precompiles["0x08"] = "bn256Pairing"

	// Istanbul (EIP-152)
	precompiles["0x09"] = "blake2f"

	// Prague/Cancun (EIP-4844)
	if cfg.IsCancun(currentTime) {
		precompiles["0x0a"] = "kzgPointEvaluation"
	}

	// Prague (EIP-2537 - BLS12-381)
	if cfg.IsPrague(currentTime) {
		precompiles["0x0b"] = "bls12381G1Add"
		precompiles["0x0c"] = "bls12381G1Mul"
		precompiles["0x0d"] = "bls12381G1MultiExp"
		precompiles["0x0e"] = "bls12381G2Add"
		precompiles["0x0f"] = "bls12381G2Mul"
		precompiles["0x10"] = "bls12381G2MultiExp"
		precompiles["0x11"] = "bls12381Pairing"
		precompiles["0x12"] = "bls12381MapG1"
		precompiles["0x13"] = "bls12381MapG2"
	}

	return precompiles
}

// getConsensusType returns the consensus mechanism type
func getConsensusType(cfg *chain.Config) string {
	switch {
	case cfg.Ethash != nil:
		return "ethash"
	case cfg.Clique != nil:
		return "clique"
	case cfg.Aura != nil:
		return "aura"
	case cfg.Bor != nil:
		return "bor"
	case cfg.TerminalTotalDifficultyPassed:
		return "pos"
	default:
		return "unknown"
	}
}

