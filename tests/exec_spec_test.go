//go:build integration

package tests

import (
	"path/filepath"
	"testing"

	"github.com/erigontech/erigon-lib/config3"
	"github.com/erigontech/erigon-lib/log/v3"
)

func TestExecutionSpec(t *testing.T) {
	if config3.EnableHistoryV3InTest {
		t.Skip("fix me in e3 please")
	}

	defer log.Root().SetHandler(log.Root().GetHandler())
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlError, log.StderrHandler))

	bt := new(testMatcher)

	dir := filepath.Join(".", "execution-spec-tests")
	// Skip .meta directories which contain metadata, not test files
	bt.skipLoad(`^\.meta/`)
	bt.skipLoad(`/\.meta/`)
	// Skip Engine API tests - they require a different test framework (use Hive instead)
	bt.skipLoad(`^blockchain_tests_engine/`)
	// Skip EIP-2935 block hashes tests - known issue with large block number tests
	bt.skipLoad(`eip2935_historical_block_hashes_from_state/block_hashes/`)
	checkStateRoot := true

	bt.walk(t, dir, func(t *testing.T, name string, test *BlockTest) {
		// import pre accounts & construct test genesis block & state root
		if err := bt.checkFailure(t, test.Run(t, checkStateRoot)); err != nil {
			t.Error(err)
		}
	})
}
