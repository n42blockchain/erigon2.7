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

package cltypes

import (
	"encoding/json"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/types/clonable"
	"github.com/erigontech/erigon/cl/merkle_tree"
	ssz2 "github.com/erigontech/erigon/cl/ssz"
)

// PendingDeposit represents a pending deposit in Electra
type PendingDeposit struct {
	Pubkey                libcommon.Bytes48 `json:"pubkey"`
	WithdrawalCredentials libcommon.Hash    `json:"withdrawal_credentials"`
	Amount                uint64            `json:"amount,string"`
	Signature             libcommon.Bytes96 `json:"signature"`
	Slot                  uint64            `json:"slot,string"`
}

func (p *PendingDeposit) EncodeSSZ(buf []byte) ([]byte, error) {
	return ssz2.MarshalSSZ(buf, p.Pubkey[:], p.WithdrawalCredentials[:], &p.Amount, p.Signature[:], &p.Slot)
}

func (p *PendingDeposit) DecodeSSZ(buf []byte, _ int) error {
	return ssz2.UnmarshalSSZ(buf, 0, p.Pubkey[:], p.WithdrawalCredentials[:], &p.Amount, p.Signature[:], &p.Slot)
}

func (p *PendingDeposit) EncodingSizeSSZ() int {
	return 48 + 32 + 8 + 96 + 8 // 192 bytes
}

func (p *PendingDeposit) HashSSZ() ([32]byte, error) {
	return merkle_tree.HashTreeRoot(p.Pubkey[:], p.WithdrawalCredentials[:], &p.Amount, p.Signature[:], &p.Slot)
}

func (p *PendingDeposit) Clone() clonable.Clonable {
	return &PendingDeposit{
		Pubkey:                p.Pubkey,
		WithdrawalCredentials: p.WithdrawalCredentials,
		Amount:                p.Amount,
		Signature:             p.Signature,
		Slot:                  p.Slot,
	}
}

// PendingPartialWithdrawal represents a pending partial withdrawal in Electra
type PendingPartialWithdrawal struct {
	Index             uint64 `json:"index,string"`
	Amount            uint64 `json:"amount,string"`
	WithdrawableEpoch uint64 `json:"withdrawable_epoch,string"`
}

func (p *PendingPartialWithdrawal) EncodeSSZ(buf []byte) ([]byte, error) {
	return ssz2.MarshalSSZ(buf, &p.Index, &p.Amount, &p.WithdrawableEpoch)
}

func (p *PendingPartialWithdrawal) DecodeSSZ(buf []byte, _ int) error {
	return ssz2.UnmarshalSSZ(buf, 0, &p.Index, &p.Amount, &p.WithdrawableEpoch)
}

func (p *PendingPartialWithdrawal) EncodingSizeSSZ() int {
	return 24 // 8 + 8 + 8 bytes
}

func (p *PendingPartialWithdrawal) HashSSZ() ([32]byte, error) {
	return merkle_tree.HashTreeRoot(&p.Index, &p.Amount, &p.WithdrawableEpoch)
}

func (p *PendingPartialWithdrawal) Clone() clonable.Clonable {
	return &PendingPartialWithdrawal{
		Index:             p.Index,
		Amount:            p.Amount,
		WithdrawableEpoch: p.WithdrawableEpoch,
	}
}

// PendingConsolidation represents a pending consolidation request in Electra
type PendingConsolidation struct {
	SourceIndex uint64 `json:"source_index,string"`
	TargetIndex uint64 `json:"target_index,string"`
}

func (p *PendingConsolidation) EncodeSSZ(buf []byte) ([]byte, error) {
	return ssz2.MarshalSSZ(buf, &p.SourceIndex, &p.TargetIndex)
}

func (p *PendingConsolidation) DecodeSSZ(buf []byte, _ int) error {
	return ssz2.UnmarshalSSZ(buf, 0, &p.SourceIndex, &p.TargetIndex)
}

func (p *PendingConsolidation) EncodingSizeSSZ() int {
	return 16 // 8 + 8 bytes
}

func (p *PendingConsolidation) HashSSZ() ([32]byte, error) {
	return merkle_tree.HashTreeRoot(&p.SourceIndex, &p.TargetIndex)
}

func (p *PendingConsolidation) Clone() clonable.Clonable {
	return &PendingConsolidation{
		SourceIndex: p.SourceIndex,
		TargetIndex: p.TargetIndex,
	}
}

// DepositRequest represents a deposit request from execution layer
type DepositRequest struct {
	Pubkey                libcommon.Bytes48 `json:"pubkey"`
	WithdrawalCredentials libcommon.Hash    `json:"withdrawal_credentials"`
	Amount                uint64            `json:"amount,string"`
	Signature             libcommon.Bytes96 `json:"signature"`
	Index                 uint64            `json:"index,string"`
}

func (d *DepositRequest) EncodeSSZ(buf []byte) ([]byte, error) {
	return ssz2.MarshalSSZ(buf, d.Pubkey[:], d.WithdrawalCredentials[:], &d.Amount, d.Signature[:], &d.Index)
}

func (d *DepositRequest) DecodeSSZ(buf []byte, _ int) error {
	return ssz2.UnmarshalSSZ(buf, 0, d.Pubkey[:], d.WithdrawalCredentials[:], &d.Amount, d.Signature[:], &d.Index)
}

func (d *DepositRequest) EncodingSizeSSZ() int {
	return 48 + 32 + 8 + 96 + 8 // 192 bytes
}

func (d *DepositRequest) HashSSZ() ([32]byte, error) {
	return merkle_tree.HashTreeRoot(d.Pubkey[:], d.WithdrawalCredentials[:], &d.Amount, d.Signature[:], &d.Index)
}

func (d *DepositRequest) Clone() clonable.Clonable {
	return &DepositRequest{
		Pubkey:                d.Pubkey,
		WithdrawalCredentials: d.WithdrawalCredentials,
		Amount:                d.Amount,
		Signature:             d.Signature,
		Index:                 d.Index,
	}
}

func (d *DepositRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Pubkey                string `json:"pubkey"`
		WithdrawalCredentials string `json:"withdrawal_credentials"`
		Amount                string `json:"amount"`
		Signature             string `json:"signature"`
		Index                 string `json:"index"`
	}{
		Pubkey:                libcommon.Bytes48(d.Pubkey).String(),
		WithdrawalCredentials: d.WithdrawalCredentials.String(),
		Amount:                json.Number(string(rune(d.Amount))).String(),
		Signature:             libcommon.Bytes96(d.Signature).String(),
		Index:                 json.Number(string(rune(d.Index))).String(),
	})
}

// WithdrawalRequest represents a withdrawal request from execution layer
type WithdrawalRequest struct {
	SourceAddress   libcommon.Address `json:"source_address"`
	ValidatorPubkey libcommon.Bytes48 `json:"validator_pubkey"`
	Amount          uint64            `json:"amount,string"`
}

func (w *WithdrawalRequest) EncodeSSZ(buf []byte) ([]byte, error) {
	return ssz2.MarshalSSZ(buf, w.SourceAddress[:], w.ValidatorPubkey[:], &w.Amount)
}

func (w *WithdrawalRequest) DecodeSSZ(buf []byte, _ int) error {
	return ssz2.UnmarshalSSZ(buf, 0, w.SourceAddress[:], w.ValidatorPubkey[:], &w.Amount)
}

func (w *WithdrawalRequest) EncodingSizeSSZ() int {
	return 20 + 48 + 8 // 76 bytes
}

func (w *WithdrawalRequest) HashSSZ() ([32]byte, error) {
	return merkle_tree.HashTreeRoot(w.SourceAddress[:], w.ValidatorPubkey[:], &w.Amount)
}

func (w *WithdrawalRequest) Clone() clonable.Clonable {
	return &WithdrawalRequest{
		SourceAddress:   w.SourceAddress,
		ValidatorPubkey: w.ValidatorPubkey,
		Amount:          w.Amount,
	}
}

// ConsolidationRequest represents a consolidation request from execution layer
type ConsolidationRequest struct {
	SourceAddress libcommon.Address `json:"source_address"`
	SourcePubkey  libcommon.Bytes48 `json:"source_pubkey"`
	TargetPubkey  libcommon.Bytes48 `json:"target_pubkey"`
}

func (c *ConsolidationRequest) EncodeSSZ(buf []byte) ([]byte, error) {
	return ssz2.MarshalSSZ(buf, c.SourceAddress[:], c.SourcePubkey[:], c.TargetPubkey[:])
}

func (c *ConsolidationRequest) DecodeSSZ(buf []byte, _ int) error {
	return ssz2.UnmarshalSSZ(buf, 0, c.SourceAddress[:], c.SourcePubkey[:], c.TargetPubkey[:])
}

func (c *ConsolidationRequest) EncodingSizeSSZ() int {
	return 20 + 48 + 48 // 116 bytes
}

func (c *ConsolidationRequest) HashSSZ() ([32]byte, error) {
	return merkle_tree.HashTreeRoot(c.SourceAddress[:], c.SourcePubkey[:], c.TargetPubkey[:])
}

func (c *ConsolidationRequest) Clone() clonable.Clonable {
	return &ConsolidationRequest{
		SourceAddress: c.SourceAddress,
		SourcePubkey:  c.SourcePubkey,
		TargetPubkey:  c.TargetPubkey,
	}
}

