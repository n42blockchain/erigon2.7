package core

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"

	"github.com/erigontech/erigon/cl/cltypes"
	"github.com/erigontech/erigon/cl/phase1/core/state"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/log/v3"
	"github.com/erigontech/erigon/cl/clparams"
)

func extractSlotFromSerializedBeaconState(beaconState []byte) (uint64, error) {
	if len(beaconState) < 48 {
		return 0, fmt.Errorf("checkpoint sync read failed, too short")
	}
	return binary.LittleEndian.Uint64(beaconState[40:48]), nil
}

// extractForkVersionFromSerializedBeaconState extracts the current fork version from serialized beacon state
// Fork structure starts at byte 48: PreviousVersion(4) + CurrentVersion(4) + Epoch(8)
// CurrentVersion is at bytes 52-55
func extractForkVersionFromSerializedBeaconState(beaconState []byte) (uint32, error) {
	if len(beaconState) < 56 {
		return 0, fmt.Errorf("checkpoint sync read failed, too short for fork version")
	}
	return binary.LittleEndian.Uint32(beaconState[52:56]), nil
}

// getVersionFromForkVersion determines the state version from the fork version
func getVersionFromForkVersion(beaconConfig *clparams.BeaconChainConfig, forkVersion uint32) clparams.StateVersion {
	switch forkVersion {
	case uint32(beaconConfig.ElectraForkVersion):
		return clparams.ElectraVersion
	case uint32(beaconConfig.DenebForkVersion):
		return clparams.DenebVersion
	case uint32(beaconConfig.CapellaForkVersion):
		return clparams.CapellaVersion
	case uint32(beaconConfig.BellatrixForkVersion):
		return clparams.BellatrixVersion
	case uint32(beaconConfig.AltairForkVersion):
		return clparams.AltairVersion
	default:
		return clparams.Phase0Version
	}
}

func RetrieveBeaconState(ctx context.Context, beaconConfig *clparams.BeaconChainConfig, uri string) (*state.CachingBeaconState, error) {
	log.Info("[Checkpoint Sync] Requesting beacon state", "uri", uri)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/octet-stream")
	if err != nil {
		return nil, fmt.Errorf("checkpoint sync request failed %s", err)
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = r.Body.Close()
	}()
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("checkpoint sync failed, bad status code %d", r.StatusCode)
	}
	marshaled, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("checkpoint sync read failed %s", err)
	}

	slot, err := extractSlotFromSerializedBeaconState(marshaled)
	if err != nil {
		return nil, fmt.Errorf("checkpoint sync read failed %s", err)
	}

	// Try to detect version from fork version in the beacon state itself
	// This is more reliable than using fork epochs from config
	forkVersion, err := extractForkVersionFromSerializedBeaconState(marshaled)
	var version clparams.StateVersion
	if err == nil {
		version = getVersionFromForkVersion(beaconConfig, forkVersion)
	}

	// Fallback to epoch-based version detection if fork version doesn't match any known version
	if version == clparams.Phase0Version && forkVersion != uint32(beaconConfig.GenesisForkVersion) {
		epoch := slot / beaconConfig.SlotsPerEpoch
		version = beaconConfig.GetCurrentStateVersion(epoch)
	}

	beaconState := state.New(beaconConfig)
	err = beaconState.DecodeSSZ(marshaled, int(version))
	if err != nil {
		// If decoding fails, try with progressively newer versions as fallback
		for tryVersion := version + 1; tryVersion <= clparams.ElectraVersion; tryVersion++ {
			beaconState = state.New(beaconConfig)
			if err = beaconState.DecodeSSZ(marshaled, int(tryVersion)); err == nil {
				log.Info("[Checkpoint Sync] Beacon state retrieved", "slot", slot)
				return beaconState, nil
			}
		}
		return nil, fmt.Errorf("checkpoint sync decode failed (tried all versions up to electra): %s", err)
	}
	log.Info("[Checkpoint Sync] Beacon state retrieved", "slot", slot)
	return beaconState, nil
}

func RetrieveBlock(ctx context.Context, beaconConfig *clparams.BeaconChainConfig, uri string, expectedBlockRoot *libcommon.Hash) (*cltypes.SignedBeaconBlock, error) {
	log.Debug("[Checkpoint Sync] Requesting beacon block", "uri", uri)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/octet-stream")
	if err != nil {
		return nil, fmt.Errorf("checkpoint sync request failed %s", err)
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = r.Body.Close()
	}()
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("checkpoint sync failed, bad status code %d", r.StatusCode)
	}
	marshaled, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("checkpoint sync read failed %s", err)
	}
	if len(marshaled) < 108 {
		return nil, fmt.Errorf("checkpoint sync read failed, too short")
	}
	currentSlot := binary.LittleEndian.Uint64(marshaled[100:108])
	v := beaconConfig.GetCurrentStateVersion(currentSlot / beaconConfig.SlotsPerEpoch)

	block := cltypes.NewSignedBeaconBlock(beaconConfig)
	err = block.DecodeSSZ(marshaled, int(v))
	if err != nil {
		return nil, fmt.Errorf("checkpoint sync decode failed %s", err)
	}
	if expectedBlockRoot != nil {
		has, err := block.Block.HashSSZ()
		if err != nil {
			return nil, fmt.Errorf("checkpoint sync decode failed %s", err)
		}
		if has != *expectedBlockRoot {
			return nil, fmt.Errorf("checkpoint sync decode failed, unexpected block root %s", has)
		}
	}
	return block, nil
}
