// Copyright 2024 The Erigon Authors
// This file is part of the Erigon library.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eth

import (
	"context"
	"errors"

	"github.com/erigontech/erigon-lib/common/datadir"
	protodownloader "github.com/erigontech/erigon-lib/gointerfaces/downloader"
	"github.com/erigontech/erigon-lib/kv"
	"github.com/erigontech/erigon-lib/log/v3"
	"google.golang.org/grpc/credentials"

	"github.com/erigontech/erigon/cl/beacon/beacon_router_configuration"
	"github.com/erigontech/erigon/cl/clparams"
	"github.com/erigontech/erigon/cl/clparams/initial_state"
	"github.com/erigontech/erigon/cl/persistence/db_config"
	"github.com/erigontech/erigon/cl/phase1/core"
	"github.com/erigontech/erigon/cl/phase1/core/state"
	"github.com/erigontech/erigon/cl/phase1/execution_client"
	"github.com/erigontech/erigon/cl/utils/eth_clock"
	"github.com/erigontech/erigon/cmd/caplin/caplin1"
	"github.com/erigontech/erigon/eth/ethconfig"
	"github.com/erigontech/erigon/turbo/snapshotsync/freezeblocks"
)

// CaplinService represents the embedded Caplin consensus layer service
type CaplinService struct {
	ctx             context.Context
	cancel          context.CancelFunc
	logger          log.Logger
	config          *ethconfig.Config
	beaconConfig    *clparams.BeaconChainConfig
	networkConfig   *clparams.NetworkConfig
	executionEngine execution_client.ExecutionEngine
	dirs            datadir.Dirs
	snDownloader    protodownloader.DownloaderClient
	blockReader     freezeblocks.BeaconSnapshotReader
	creds           credentials.TransportCredentials

	indexDB kv.RwDB
	running bool
}

// NewCaplinService creates a new embedded Caplin CL service
func NewCaplinService(
	ctx context.Context,
	logger log.Logger,
	config *ethconfig.Config,
	executionEngine execution_client.ExecutionEngine,
	dirs datadir.Dirs,
	snDownloader protodownloader.DownloaderClient,
	creds credentials.TransportCredentials,
) (*CaplinService, error) {
	networkType := clparams.NetworkType(config.NetworkID)
	networkConfig, beaconConfig := clparams.GetConfigsByNetwork(networkType)

	ctx, cancel := context.WithCancel(ctx)

	return &CaplinService{
		ctx:             ctx,
		cancel:          cancel,
		logger:          logger.New("service", "caplin"),
		config:          config,
		beaconConfig:    beaconConfig,
		networkConfig:   networkConfig,
		executionEngine: executionEngine,
		dirs:            dirs,
		snDownloader:    snDownloader,
		creds:           creds,
	}, nil
}

// Start starts the Caplin CL service
func (s *CaplinService) Start() error {
	if s.running {
		return nil
	}

	s.logger.Info("Starting embedded Caplin consensus layer")

	// Get the genesis state for this network
	genesisState, err := initial_state.GetGenesisState(clparams.NetworkType(s.config.NetworkID))
	if err != nil {
		s.logger.Error("Failed to get genesis state", "err", err)
		return err
	}

	// Try to get checkpoint state if available
	var beaconState *state.CachingBeaconState
	checkpointUri := clparams.GetCheckpointSyncEndpoint(clparams.NetworkType(s.config.NetworkID))
	if checkpointUri != "" {
		beaconState, err = core.RetrieveBeaconState(s.ctx, s.beaconConfig, checkpointUri)
		if err != nil {
			s.logger.Warn("Failed to retrieve checkpoint state, starting from genesis", "err", err)
			beaconState = genesisState
		}
	} else {
		beaconState = genesisState
	}

	ethClock := eth_clock.NewEthereumClock(beaconState.GenesisTime(), beaconState.GenesisValidatorsRoot(), s.beaconConfig)

	// Open Caplin database
	indexDB, blobStorage, err := caplin1.OpenCaplinDatabase(
		s.ctx,
		db_config.DefaultDatabaseConfiguration,
		s.beaconConfig,
		ethClock,
		s.dirs.CaplinIndexing,
		s.dirs.CaplinBlobs,
		s.executionEngine,
		false,   // wipeout
		100_000, // blobPruneDistance
	)
	if err != nil {
		s.logger.Error("Failed to open Caplin database", "err", err)
		return err
	}
	s.indexDB = indexDB

	// Setup beacon router configuration
	rcfg := beacon_router_configuration.RouterConfiguration{
		Protocol:         "tcp",
		Address:          s.config.BeaconRouter.Address,
		ReadTimeTimeout:  s.config.BeaconRouter.ReadTimeTimeout,
		WriteTimeout:     s.config.BeaconRouter.WriteTimeout,
		IdleTimeout:      s.config.BeaconRouter.IdleTimeout,
		AllowedOrigins:   s.config.BeaconRouter.AllowedOrigins,
		AllowedMethods:   s.config.BeaconRouter.AllowedMethods,
		AllowCredentials: s.config.BeaconRouter.AllowCredentials,
		Active:           s.config.BeaconRouter.Active,
		Validator:        s.config.BeaconRouter.Validator,
	}

	// Run Caplin in a goroutine
	go func() {
		if err := caplin1.RunCaplinPhase1(
			s.ctx,
			s.executionEngine,
			&ethconfig.Config{
				LightClientDiscoveryAddr:    s.config.LightClientDiscoveryAddr,
				LightClientDiscoveryPort:    s.config.LightClientDiscoveryPort,
				LightClientDiscoveryTCPPort: s.config.LightClientDiscoveryTCPPort,
				BeaconRouter:                rcfg,
				SentinelAddr:                s.config.SentinelAddr,
				SentinelPort:                s.config.SentinelPort,
			},
			s.networkConfig,
			s.beaconConfig,
			ethClock,
			beaconState,
			s.dirs,
			nil, // eth1Getter - will use execution client
			s.snDownloader,
			s.config.CaplinConfig.Backfilling,
			s.config.CaplinConfig.BlobBackfilling,
			s.config.CaplinConfig.Archive,
			indexDB,
			blobStorage,
			s.creds,
		); err != nil {
			// Don't log context cancellation as error - it's normal shutdown
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				s.logger.Debug("Caplin service stopped", "reason", err)
			} else {
				s.logger.Error("Caplin service error", "err", err)
			}
		}
	}()

	s.running = true
	s.logger.Info("Caplin consensus layer started successfully")
	return nil
}

// Stop stops the Caplin CL service
func (s *CaplinService) Stop() {
	if !s.running {
		return
	}

	s.logger.Info("Stopping Caplin consensus layer")
	s.cancel()

	if s.indexDB != nil {
		s.indexDB.Close()
	}

	s.running = false
	s.logger.Info("Caplin consensus layer stopped")
}

// Running returns true if the service is running
func (s *CaplinService) Running() bool {
	return s.running
}
