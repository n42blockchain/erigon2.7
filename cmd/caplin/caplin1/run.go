package caplin1

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"google.golang.org/grpc/credentials"

	proto_downloader "github.com/erigontech/erigon-lib/gointerfaces/downloader"
	"github.com/erigontech/erigon/cl/aggregation"
	"github.com/erigontech/erigon/cl/antiquary"
	"github.com/erigontech/erigon/cl/beacon"
	"github.com/erigontech/erigon/cl/beacon/beaconevents"
	"github.com/erigontech/erigon/cl/beacon/handler"
	"github.com/erigontech/erigon/cl/beacon/synced_data"
	"github.com/erigontech/erigon/cl/clparams/initial_state"
	"github.com/erigontech/erigon/cl/cltypes"
	"github.com/erigontech/erigon/cl/cltypes/solid"
	"github.com/erigontech/erigon/cl/rpc"
	"github.com/erigontech/erigon/cl/sentinel"
	"github.com/erigontech/erigon/cl/sentinel/service"
	"github.com/erigontech/erigon/cl/utils/eth_clock"
	"github.com/erigontech/erigon/cl/validator/attestation_producer"
	"github.com/erigontech/erigon/cl/validator/committee_subscription"
	"github.com/erigontech/erigon/cl/validator/sync_contribution_pool"
	"github.com/erigontech/erigon/cl/validator/validator_params"
	"github.com/erigontech/erigon/eth/ethconfig"
	"github.com/erigontech/erigon/params"
	"github.com/erigontech/erigon/turbo/snapshotsync"
	"github.com/erigontech/erigon/turbo/snapshotsync/freezeblocks"
	"github.com/erigontech/erigon/cl/das"
	"github.com/erigontech/erigon/p2p/enode"
	"golang.org/x/sync/semaphore"

	"github.com/spf13/afero"

	"github.com/erigontech/erigon/cl/persistence/beacon_indicies"
	"github.com/erigontech/erigon/cl/persistence/blob_storage"
	"github.com/erigontech/erigon/cl/persistence/db_config"
	"github.com/erigontech/erigon/cl/persistence/format/snapshot_format"
	state_accessors "github.com/erigontech/erigon/cl/persistence/state"
	"github.com/erigontech/erigon/cl/persistence/state/historical_states_reader"
	"github.com/erigontech/erigon/cl/phase1/core/state"
	"github.com/erigontech/erigon/cl/phase1/execution_client"
	peerdasstate "github.com/erigontech/erigon/cl/das/state"
	"github.com/erigontech/erigon/cl/phase1/forkchoice"
	"github.com/erigontech/erigon/cl/phase1/forkchoice/fork_graph"
	"github.com/erigontech/erigon/cl/phase1/forkchoice/public_keys_registry"
	"github.com/erigontech/erigon/cl/phase1/network"
	"github.com/erigontech/erigon/cl/phase1/network/services"
	"github.com/erigontech/erigon/cl/phase1/stages"
	"github.com/erigontech/erigon/cl/pool"

	"github.com/Giulio2002/bls"
	"github.com/erigontech/erigon-lib/log/v3"

	"github.com/erigontech/erigon-lib/common/datadir"
	"github.com/erigontech/erigon-lib/kv"
	"github.com/erigontech/erigon-lib/kv/mdbx"
	"github.com/erigontech/erigon/cl/clparams"
)

func OpenCaplinDatabase(ctx context.Context,
	databaseConfig db_config.DatabaseConfiguration,
	beaconConfig *clparams.BeaconChainConfig,
	ethClock eth_clock.EthereumClock,
	dbPath string,
	blobDir string,
	engine execution_client.ExecutionEngine,
	wipeout bool,
	blobPruneDistance uint64,
) (kv.RwDB, blob_storage.BlobStorage, error) {
	dataDirIndexer := path.Join(dbPath, "beacon_indicies")
	blobDbPath := path.Join(blobDir, "chaindata")

	if wipeout {
		os.RemoveAll(dataDirIndexer)
		os.RemoveAll(blobDbPath)
	}

	os.MkdirAll(dbPath, 0700)
	os.MkdirAll(dataDirIndexer, 0700)

	db := mdbx.MustOpen(dataDirIndexer)
	blobDB := mdbx.MustOpen(blobDbPath)

	tx, err := db.BeginRw(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	if err := db_config.WriteConfigurationIfNotExist(ctx, tx, databaseConfig); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	{ // start ticking forkChoice
		go func() {
			<-ctx.Done()
			db.Close()     // close sql database here
			blobDB.Close() // close blob database here
		}()
	}
	return db, blob_storage.NewBlobStore(blobDB, afero.NewBasePathFs(afero.NewOsFs(), blobDir), blobPruneDistance, beaconConfig, ethClock), nil
}

func RunCaplinPhase1(ctx context.Context, engine execution_client.ExecutionEngine, config *ethconfig.Config, networkConfig *clparams.NetworkConfig,
	beaconConfig *clparams.BeaconChainConfig, ethClock eth_clock.EthereumClock, state *state.CachingBeaconState, dirs datadir.Dirs, eth1Getter snapshot_format.ExecutionBlockReaderByNumber,
	snDownloader proto_downloader.DownloaderClient, backfilling, blobBackfilling bool, states bool, indexDB kv.RwDB, blobStorage blob_storage.BlobStorage, creds credentials.TransportCredentials) error {
	ctx, cn := context.WithCancel(ctx)
	defer cn()

	logger := log.New("app", "caplin")

	csn := freezeblocks.NewCaplinSnapshots(ethconfig.BlocksFreezing{}, beaconConfig, dirs, logger)
	rcsn := freezeblocks.NewBeaconSnapshotReader(csn, eth1Getter, beaconConfig)

	pool := pool.NewOperationsPool(beaconConfig)
	attestationProducer := attestation_producer.New(ctx, beaconConfig)

	caplinFcuPath := path.Join(dirs.Tmp, "caplin-forkchoice")
	os.RemoveAll(caplinFcuPath)
	err := os.MkdirAll(caplinFcuPath, 0o755)
	if err != nil {
		return err
	}
	fcuFs := afero.NewBasePathFs(afero.NewOsFs(), caplinFcuPath)
	syncedDataManager := synced_data.NewSyncedDataManager(beaconConfig, true)

	syncContributionPool := sync_contribution_pool.NewSyncContributionPool(beaconConfig)
	eventEmitter := beaconevents.NewEventEmitter()
	aggregationPool := aggregation.NewAggregationPool(ctx, beaconConfig, networkConfig, ethClock)
	localValidators := validator_params.NewValidatorParams()

	// Create data column storage for PeerDAS
	columnFs := afero.NewBasePathFs(afero.NewOsFs(), path.Join(dirs.Tmp, "data-columns"))
	dataColumnStorage := blob_storage.NewDataColumnStore(columnFs, 1000, beaconConfig, ethClock, eventEmitter)

	// Create PeerDAS state
	peerDasState := peerdasstate.NewPeerDasState(beaconConfig, networkConfig)

	// Create Caplin state snapshots
	stateSn := snapshotsync.NewCaplinStateSnapshots()

	// Create semaphore for snapshot building
	snBuildSema := semaphore.NewWeighted(1)

	// Create batch signature verifier
	batchSignatureVerifier := services.NewBatchSignatureVerifier(ctx, nil)
	go batchSignatureVerifier.Start()

	forkChoice, err := forkchoice.NewForkChoiceStore(ethClock, state, engine, pool, fork_graph.NewForkGraphDisk(state, syncedDataManager, fcuFs, config.BeaconRouter, eventEmitter), eventEmitter, syncedDataManager, blobStorage, public_keys_registry.NewInMemoryPublicKeysRegistry(), localValidators, false)
	if err != nil {
		logger.Error("Could not create forkchoice", "err", err)
		return err
	}
	bls.SetEnabledCaching(true)
	state.ForEachValidator(func(v solid.Validator, idx, total int) bool {
		pk := v.PublicKey()
		if err := bls.LoadPublicKeyIntoCache(pk[:], false); err != nil {
			panic(err)
		}
		return true
	})

	forkDigest, err := ethClock.CurrentForkDigest()
	if err != nil {
		return err
	}
	activeIndicies := state.GetActiveValidatorsIndices(state.Slot() / beaconConfig.SlotsPerEpoch)

	sentinelClient, _, err := service.StartSentinelService(&sentinel.SentinelConfig{
		IpAddr:         config.LightClientDiscoveryAddr,
		Port:           int(config.LightClientDiscoveryPort),
		TCPPort:        uint(config.LightClientDiscoveryTCPPort),
		NetworkConfig:  networkConfig,
		BeaconConfig:   beaconConfig,
		TmpDir:         dirs.Tmp,
		EnableBlocks:   true,
		ActiveIndicies: uint64(len(activeIndicies)),
	}, rcsn, blobStorage, indexDB, &service.ServerConfig{
		Network: "tcp",
		Addr:    fmt.Sprintf("%s:%d", config.SentinelAddr, config.SentinelPort),
		Creds:   creds,
		InitialStatus: &cltypes.Status{
			ForkDigest:     forkDigest,
			FinalizedRoot:  state.FinalizedCheckpoint().Root,
			FinalizedEpoch: state.FinalizedCheckpoint().Epoch,
			HeadSlot:       state.FinalizedCheckpoint().Epoch * beaconConfig.SlotsPerEpoch,
			HeadRoot:       state.FinalizedCheckpoint().Root,
		},
	}, ethClock, forkChoice, dataColumnStorage, peerDasState, logger)
	if err != nil {
		return err
	}
	beaconRpc := rpc.NewBeaconRpcP2P(ctx, sentinelClient, beaconConfig, ethClock, state)
	committeeSub := committee_subscription.NewCommitteeSubscribeManagement(ctx, indexDB, beaconConfig, networkConfig, ethClock, sentinelClient, aggregationPool, syncedDataManager)
	// Define gossip services
	blockService := services.NewBlockService(ctx, indexDB, forkChoice, syncedDataManager, ethClock, beaconConfig, eventEmitter)
	blobService := services.NewBlobSidecarService(ctx, beaconConfig, forkChoice, syncedDataManager, ethClock, eventEmitter, false)
	dataColumnSidecarService := services.NewDataColumnSidecarService(beaconConfig, ethClock, forkChoice, syncedDataManager, dataColumnStorage, eventEmitter)
	syncCommitteeMessagesService := services.NewSyncCommitteeMessagesService(beaconConfig, ethClock, syncedDataManager, syncContributionPool, batchSignatureVerifier, false)
	attestationService := services.NewAttestationService(ctx, forkChoice, committeeSub, ethClock, syncedDataManager, beaconConfig, networkConfig, eventEmitter, batchSignatureVerifier)
	syncContributionService := services.NewSyncContributionService(syncedDataManager, beaconConfig, syncContributionPool, ethClock, eventEmitter, batchSignatureVerifier, false)
	aggregateAndProofService := services.NewAggregateAndProofService(ctx, syncedDataManager, forkChoice, beaconConfig, pool, false, batchSignatureVerifier)
	voluntaryExitService := services.NewVoluntaryExitService(pool, eventEmitter, syncedDataManager, beaconConfig, ethClock, batchSignatureVerifier)
	blsToExecutionChangeService := services.NewBLSToExecutionChangeService(pool, eventEmitter, syncedDataManager, beaconConfig, batchSignatureVerifier)
	proposerSlashingService := services.NewProposerSlashingService(pool, syncedDataManager, beaconConfig, ethClock, eventEmitter)
	// Create the gossip manager
	gossipManager := network.NewGossipReceiver(sentinelClient, forkChoice, beaconConfig, networkConfig, ethClock, eventEmitter, committeeSub,
		blockService, blobService, dataColumnSidecarService, syncCommitteeMessagesService, syncContributionService, aggregateAndProofService,
		attestationService, voluntaryExitService, blsToExecutionChangeService, proposerSlashingService)
	{ // start ticking forkChoice
		go func() {
			tickInterval := time.NewTicker(2 * time.Millisecond)
			for {
				select {
				case <-tickInterval.C:
					forkChoice.OnTick(uint64(time.Now().Unix()))
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	{ // start the gossip manager
		go gossipManager.Start(ctx)
		logger.Info("Started Ethereum 2.0 Gossip Service")
	}

	{ // start logging peers
		go func() {
			logIntervalPeers := time.NewTicker(1 * time.Minute)
			for {
				select {
				case <-logIntervalPeers.C:
					if peerCount, err := beaconRpc.Peers(); err == nil {
						logger.Info("P2P", "peers", peerCount)
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	tx, err := indexDB.BeginRw(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	dbConfig, err := db_config.ReadConfiguration(ctx, tx)
	if err != nil {
		return err
	}

	if err := beacon_indicies.WriteHighestFinalized(tx, 0); err != nil {
		return err
	}

	vTables := state_accessors.NewStaticValidatorTable()
	// Read the current table
	if states {
		if err := state_accessors.ReadValidatorsTable(tx, vTables); err != nil {
			return err
		}
	}
	// get the initial state
	genesisState, err := initial_state.GetGenesisState(clparams.NetworkType(beaconConfig.DepositNetworkID))
	if err != nil {
		return err
	}
	antiq := antiquary.NewAntiquary(ctx, blobStorage, genesisState, vTables, beaconConfig, dirs, snDownloader, indexDB, stateSn, csn, rcsn, syncedDataManager, logger, states, backfilling, blobBackfilling, false, snBuildSema)
	// Create the antiquary
	go func() {
		if err := antiq.Loop(); err != nil {
			logger.Error("Antiquary failed", "err", err)
		}
	}()

	if err := tx.Commit(); err != nil {
		return err
	}

	statesReader := historical_states_reader.NewHistoricalStatesReader(beaconConfig, rcsn, vTables, genesisState, stateSn, syncedDataManager)
	validatorParameters := validator_params.NewValidatorParams()

	// Create PeerDas
	caplinConfig := clparams.CaplinConfig{}
	peerDas := das.NewPeerDas(ctx, beaconRpc, beaconConfig, &caplinConfig, dataColumnStorage, blobStorage, sentinelClient, enode.ID{}, ethClock, peerDasState)
	if config.BeaconRouter.Active {
		apiHandler := handler.NewApiHandler(
			logger,
			networkConfig,
			ethClock,
			beaconConfig,
			indexDB,
			forkChoice,
			pool,
			rcsn,
			syncedDataManager,
			statesReader,
			sentinelClient,
			params.GitTag,
			&config.BeaconRouter,
			eventEmitter,
			blobStorage,
			dataColumnStorage,
			csn,
			validatorParameters,
			attestationProducer,
			engine,
			syncContributionPool,
			committeeSub,
			aggregationPool,
			syncCommitteeMessagesService,
			syncContributionService,
			aggregateAndProofService,
			attestationService,
			voluntaryExitService,
			blsToExecutionChangeService,
			proposerSlashingService,
			nil,     // builderClient
			stateSn, // stateSnapshots
			true,    // enableMemoizedHeadState
			peerDas, // peerDas
		)
		go beacon.ListenAndServe(&beacon.LayeredBeaconHandler{
			ArchiveApi: apiHandler,
		}, config.BeaconRouter)
		log.Info("Beacon API started", "addr", config.BeaconRouter.Address)
	}

	stageCfg := stages.ClStagesCfg(beaconRpc, antiq, ethClock, beaconConfig, state, engine, gossipManager, forkChoice, indexDB, csn, rcsn, dirs, dbConfig.PruneDepth, caplinConfig, syncedDataManager, eventEmitter, blobStorage, attestationProducer, peerDas)
	sync := stages.ConsensusClStages(ctx, stageCfg)

	logger.Info("[Caplin] starting clstages loop")
	err = sync.StartWithStage(ctx, "DownloadHistoricalBlocks", logger, stageCfg)
	logger.Info("[Caplin] exiting clstages loop")
	if err != nil {
		return err
	}
	return err
}
