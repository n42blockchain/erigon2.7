// Copyright 2024 The Erigon Authors
// This file is part of Erigon.
//
// Erigon is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Erigon is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Erigon. If not, see <http://www.gnu.org/licenses/>.

package caches

import (
	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon/cl/phase1/core/state/lru"
)

const MaxActiveValidatorsCacheSize = 8

type activeValidatorsCacheKey struct {
	epoch uint64
	root  libcommon.Hash
}

type activeValidatorsCacheVal struct {
	activeValidators   []uint64
	totalActiveBalance uint64
}

type ActiveValidatorsCache struct {
	cache *lru.Cache[activeValidatorsCacheKey, activeValidatorsCacheVal]
}

var ActiveValidatorsCacheGlobal = NewActiveValidatorsCache(MaxActiveValidatorsCacheSize)

func NewActiveValidatorsCache(size int) *ActiveValidatorsCache {
	cache, err := lru.New[activeValidatorsCacheKey, activeValidatorsCacheVal]("active_validators", size)
	if err != nil {
		panic(err)
	}
	return &ActiveValidatorsCache{
		cache: cache,
	}
}

func (c *ActiveValidatorsCache) Get(epoch uint64, root libcommon.Hash) ([]uint64, uint64, bool) {
	val, ok := c.cache.Get(activeValidatorsCacheKey{epoch: epoch, root: root})
	if !ok {
		return nil, 0, false
	}
	return val.activeValidators, val.totalActiveBalance, true
}

func (c *ActiveValidatorsCache) Put(epoch uint64, root libcommon.Hash, activeValidators []uint64, totalActiveBalance uint64) {
	c.cache.Add(activeValidatorsCacheKey{epoch: epoch, root: root}, activeValidatorsCacheVal{
		activeValidators:   activeValidators,
		totalActiveBalance: totalActiveBalance,
	})
}
