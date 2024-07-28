package nakamoto

import (
	"testing"
)

func TestMempool(t *testing.T) {
	// TODO implement.
}

// note: validators don’t have to trust builders not to withhold block bodies or publish invalid blocks because payment is unconditional. The validator’s fee still processes even if the proposed block is unavailable or declared invalid by other validators. In the latter case, the block is simply discarded, forcing the block builder to lose all transaction fees and MEV revenue.
// https://ethereum.org/en/developers/docs/mev/

// Flow: User -> Node -> Mempool <-> Miner
// Test: from user to node, that the user's submitted transaction makes it to the mempool
// test: that the builder's transaction verification works
// test: that the builder's bundle is the right size, and that the transactions are ordered by fee and that the builders commit/reveal is legit
// test: what happens if there's a reorg? the miner re-broadcasts back to the builders? WHERE'S THE CODE
// test: how do the fees work?
