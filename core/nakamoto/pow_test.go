package nakamoto

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestBuildBlock(t *testing.T) {
	genesis_block := RawBlock{}
	// test not null
	t.Log(genesis_block)
}

func TestGenesisBlockHash(t *testing.T) {
	genesis_block := RawBlock{}
	// get envelope
	envelope := genesis_block.Envelope()
	// now hash it
	h := sha256.New()
	h.Write(envelope)
	fmt.Printf("%x\n", h.Sum(nil))

	expected, err := hex.DecodeString("b5fdab78d8947eacc864bfeecb4d2100780e5afe1cd8efafb124887913ac49fa")

	if err != nil {
		t.Fatalf("Failed to decode expected hash")
	}

	if !bytes.Equal(h.Sum(nil), expected) {
		t.Fatalf("Genesis block hash is incorrect")
	}
}

func TestProofOfWorkSolver(t *testing.T) {
	// create a genesis block
	genesis_block := RawBlock{}
	nonce := new(big.Int)
	target := new(big.Int)
	target.SetString("0000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)

	solution, err := SolvePOW(genesis_block, *nonce, *target, 1000000)
	if err != nil {
		t.Fatalf("Failed to solve proof of work")
	}
	fmt.Printf("Solution: %x\n", solution.String())
}

func TestBuildChainOfBlocks(t *testing.T) {
	assert := assert.New(t)

	// Build a chain of 6 blocks.
	chain := make([]RawBlock, 0)
	curr_block := RawBlock{}

	// Fixed target for test.
	target := new(big.Int)
	target.SetString("0fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)

	for {
		fmt.Printf("Mining block %x\n", curr_block.Hash())
		solution, err := SolvePOW(curr_block, *new(big.Int), *target, 100000000000)
		if err != nil {
			assert.Nil(t, err)
		}
		fmt.Printf("Solution: %x\n", solution.String())

		// Seal the block.
		curr_block.SetNonce(solution)

		// Append the block to the chain.
		chain = append(chain, curr_block)

		// Create a new block.
		timestamp := uint64(0)
		curr_block = RawBlock{
			ParentHash:      curr_block.Hash(),
			Timestamp:       timestamp,
			NumTransactions: 0,
			Transactions:    []RawTransaction{},
		}

		// Exit if the chain is long enough.
		if len(chain) >= 6 {
			break
		}
	}
}

// func TestSomething(t *testing.T) {
// 	// Setup the configuration for a consensus epoch.
// 	genesis_difficulty := new(big.Int)
// 	genesis_difficulty.SetString("0fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)

// 	conf := ConsensusConfig{
// 		EpochLengthBlocks: 5,
// 		TargetEpochLengthMillis: 2000,
// 		GenesisDifficulty: *genesis_difficulty,
// 		GenesisBlockHash: [32]byte{},
// 		MaxBlockSizeBytes: 1000000,
// 	}
// 	difficulty := conf.GenesisDifficulty

// 	// Now mine 2 epochs worth of blocks.
// 	chain := make([]RawBlock, 0)
// 	curr_block := RawBlock{}
// 	for {
// 		fmt.Printf("Mining block %x\n", curr_block.Hash())
// 		solution, err := SolvePOW(curr_block, *new(big.Int), difficulty, 100000000000)
// 		if err != nil {
// 			t.Fatalf("Failed to solve proof of work")
// 		}
// 		fmt.Printf("Solution: %x\n", solution.String())

// 		// Seal the block.
// 		curr_block.SetNonce(solution)
// 		curr_block.Timestamp = Timestamp()

// 		// Append the block to the chain.
// 		chain = append(chain, curr_block)

// 		// Create a new block.
// 		curr_block = RawBlock{
// 			ParentHash: curr_block.Hash(),
// 			Timestamp: 0,
// 			NumTransactions: 0,
// 			Transactions: []RawTransaction{},
// 		}

// 		// Recompute the difficulty.
// 		if len(chain) % int(conf.EpochLengthBlocks) == 0 {
// 			// Compute the time taken to mine the last epoch.
// 			epoch_start := chain[len(chain) - int(conf.EpochLengthBlocks)].Timestamp
// 			epoch_end := chain[len(chain) - 1].Timestamp
// 			epoch_duration := epoch_end - epoch_start
// 			if epoch_duration == 0 {
// 				epoch_duration = 1
// 			}
// 			epoch_index := len(chain) / int(conf.EpochLengthBlocks)
// 			fmt.Printf("epoch i=%d start_time=%d end_time=%d duration=%d \n", epoch_index, epoch_start, epoch_end, epoch_duration)

// 			// Compute the target epoch length.
// 			target_epoch_length := conf.TargetEpochLengthMillis * conf.EpochLengthBlocks

// 			// Compute the new difficulty.
// 			// difficulty = difficulty * (epoch_duration / target_epoch_length)
// 			new_difficulty := new(big.Int)
// 			new_difficulty.Mul(&conf.GenesisDifficulty, big.NewInt(int64(epoch_duration)))
// 			new_difficulty.Div(new_difficulty, big.NewInt(int64(target_epoch_length)))

// 			fmt.Printf("New difficulty: %x\n", new_difficulty.String())

// 			// Update the difficulty.
// 			difficulty = *new_difficulty
// 		}

// 		fmt.Printf("Chain length: %d\n", len(chain))
// 		if len(chain) >= 4 * int(conf.EpochLengthBlocks) {
// 			break
// 		}
// 	}
// }

func TestCalculateWork(t *testing.T) {
	diff_target := new(big.Int)
	diff_target.SetString("0fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
	// max_diff_target := big.NewInt(0).Exp(big.NewInt(2), big.NewInt(256), nil)

	acc_work := big.NewInt(0)
	block_template := RawBlock{}

	// https://bitcoin.stackexchange.com/questions/105213/how-is-cumulative-pow-calculated-to-decide-between-competing-chains
	// work = 2^256 / (target + 1)
	// difficulty = max_target / target

	// Solve 30 blocks, adjust difficulty every 10.
	for i := 0; i < 30; i++ {
		solution, err := SolvePOW(block_template, *new(big.Int), *diff_target, 100000000000)
		if err != nil {
			t.Fatalf("Failed to solve proof of work")
		}

		// Seal the block.
		block_template.SetNonce(solution)

		// Setup next block.
		block_template = RawBlock{
			ParentHash: block_template.Hash(),
			Timestamp:  0,
		}

		// Calculate the work.
		work := big.NewInt(2).Exp(big.NewInt(2), big.NewInt(256), nil)
		work.Div(work, big.NewInt(0).Add(diff_target, big.NewInt(1)))
		fmt.Printf("Work: %x\n", work.String())

		acc_work.Add(acc_work, work)
		fmt.Printf("Acc Work: %x\n", acc_work.String())
	}
}
