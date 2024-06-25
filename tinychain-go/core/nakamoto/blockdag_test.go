package nakamoto

import (
	"testing"
)

func TestOpenDB(t *testing.T) {
	// test not null
	err := OpenDB("test.db")
	if err != nil {
		t.Log(err)
	}
}

func TestImportBlocksIntoDAG(t *testing.T) {
	// Generate 10 blocks and insert them into DAG.
	blockdag := BlockDAG{}
	assert := assert.New(t)
	
	// Build a chain of 6 blocks.
	chain := make([]RawBlock, 0)
	curr_block := RawBlock{}

	// Fixed target for test.
	target := new(big.Int)
	target.SetString("0fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)

	for {
		solution, err := SolvePOW(curr_block, *new(big.Int), *target, 100000000000)
		if err != nil {
			assert.Nil(t, err)
		}

		// Seal the block.
		curr_block.SetNonce(solution)

		// Append the block to the chain.
		blockdag.CheckRawBlock(curr_block)

		// Create a new block.
		timestamp := uint64(0)
		curr_block = RawBlock{
			ParentHash: curr_block.Hash(),
			Timestamp: timestamp,
			NumTransactions: 0,
			Transactions: []Transaction{},
		}

		// Exit if the chain is long enough.
		if len(chain) >= 6 {
			break
		}
	}
}

func TestGetBlock(t *testing.T) {
	// Insert a block into DAG.
	// Query DAG to verify block inserted.
}


