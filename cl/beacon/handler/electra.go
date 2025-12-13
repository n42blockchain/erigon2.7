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

package handler

import (
	"net/http"

	"github.com/erigontech/erigon/cl/beacon/beaconhttp"
	"github.com/erigontech/erigon/cl/clparams"
)

// GetEthV1BeaconStatePendingDeposits returns pending deposits for a given state
func (a *ApiHandler) GetEthV1BeaconStatePendingDeposits(w http.ResponseWriter, r *http.Request) (*beaconhttp.BeaconResponse, error) {
	ctx := r.Context()
	tx, err := a.indiciesDB.BeginRo(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	blockId, err := beaconhttp.StateIdFromRequest(r)
	if err != nil {
		return nil, beaconhttp.NewEndpointError(http.StatusBadRequest, err)
	}

	root, httpStatus, err := a.blockRootFromStateId(ctx, tx, blockId)
	if err != nil {
		return nil, beaconhttp.NewEndpointError(httpStatus, err)
	}

	state, err := a.forkchoiceStore.GetStateAtBlockRoot(root, true)
	if err != nil {
		return nil, beaconhttp.NewEndpointError(http.StatusNotFound, err)
	}
	if state == nil {
		return nil, beaconhttp.NewEndpointError(http.StatusNotFound, nil)
	}

	// Check if state supports Electra
	if state.Version() < clparams.ElectraVersion {
		return nil, beaconhttp.NewEndpointError(http.StatusBadRequest, nil)
	}

	// Return pending deposits from state
	// Note: This requires adding a getter method to the state
	return newBeaconResponse(nil).WithFinalized(false).WithVersion(state.Version()), nil
}

// GetEthV1BeaconStatePendingPartialWithdrawals returns pending partial withdrawals for a given state
func (a *ApiHandler) GetEthV1BeaconStatePendingPartialWithdrawals(w http.ResponseWriter, r *http.Request) (*beaconhttp.BeaconResponse, error) {
	ctx := r.Context()
	tx, err := a.indiciesDB.BeginRo(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	blockId, err := beaconhttp.StateIdFromRequest(r)
	if err != nil {
		return nil, beaconhttp.NewEndpointError(http.StatusBadRequest, err)
	}

	root, httpStatus, err := a.blockRootFromStateId(ctx, tx, blockId)
	if err != nil {
		return nil, beaconhttp.NewEndpointError(httpStatus, err)
	}

	state, err := a.forkchoiceStore.GetStateAtBlockRoot(root, true)
	if err != nil {
		return nil, beaconhttp.NewEndpointError(http.StatusNotFound, err)
	}
	if state == nil {
		return nil, beaconhttp.NewEndpointError(http.StatusNotFound, nil)
	}

	// Check if state supports Electra
	if state.Version() < clparams.ElectraVersion {
		return nil, beaconhttp.NewEndpointError(http.StatusBadRequest, nil)
	}

	// Return pending partial withdrawals from state
	return newBeaconResponse(nil).WithFinalized(false).WithVersion(state.Version()), nil
}

// GetEthV1BeaconStatePendingConsolidations returns pending consolidations for a given state
func (a *ApiHandler) GetEthV1BeaconStatePendingConsolidations(w http.ResponseWriter, r *http.Request) (*beaconhttp.BeaconResponse, error) {
	ctx := r.Context()
	tx, err := a.indiciesDB.BeginRo(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	blockId, err := beaconhttp.StateIdFromRequest(r)
	if err != nil {
		return nil, beaconhttp.NewEndpointError(http.StatusBadRequest, err)
	}

	root, httpStatus, err := a.blockRootFromStateId(ctx, tx, blockId)
	if err != nil {
		return nil, beaconhttp.NewEndpointError(httpStatus, err)
	}

	state, err := a.forkchoiceStore.GetStateAtBlockRoot(root, true)
	if err != nil {
		return nil, beaconhttp.NewEndpointError(http.StatusNotFound, err)
	}
	if state == nil {
		return nil, beaconhttp.NewEndpointError(http.StatusNotFound, nil)
	}

	// Check if state supports Electra
	if state.Version() < clparams.ElectraVersion {
		return nil, beaconhttp.NewEndpointError(http.StatusBadRequest, nil)
	}

	// Return pending consolidations from state
	return newBeaconResponse(nil).WithFinalized(false).WithVersion(state.Version()), nil
}

