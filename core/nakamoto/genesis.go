package nakamoto

import (
	"fmt"
	"math/big"
)

// The Nakamoto consensus configuration, pertaining to difficulty readjustment, genesis block, and block size.
type ConsensusConfig struct {
	// The length of an epoch.
	EpochLengthBlocks uint64 `json:"epoch_length_blocks"`

	// The target block production rate in terms of 1 epoch.
	TargetEpochLengthMillis uint64 `json:"target_epoch_length_millis"`

	// Genesis difficulty target.
	GenesisDifficulty big.Int `json:"genesis_difficulty"`

	// The genesis parent block hash.
	GenesisParentBlockHash [32]byte `json:"genesis_block_hash"`

	// Maximum block size.
	MaxBlockSizeBytes uint64 `json:"max_block_size_bytes"`
}

// Builds the raw genesis block from the consensus configuration.
func GetRawGenesisBlockFromConfig(consensus ConsensusConfig) RawBlock {
	block := RawBlock{
		// Special case: The genesis block has a parent we don't know the preimage for.
		ParentHash:             consensus.GenesisParentBlockHash,
		ParentTotalWork:        [32]byte{},
		Difficulty:             BigIntToBytes32(consensus.GenesisDifficulty),
		Timestamp:              0,
		NumTransactions:        0,
		TransactionsMerkleRoot: [32]byte{},
		Nonce:                  [32]byte{},
		Graffiti:               [32]byte{0xca, 0xfe, 0xba, 0xbe, 0xde, 0xca, 0xfb, 0xad, 0xde, 0xad, 0xbe, 0xef}, // 0x cafebabe decafbad deadbeef
		Transactions:           []RawTransaction{},
	}

	// Mine the block.
	solution, err := SolvePOW(block, *new(big.Int), consensus.GenesisDifficulty, 100)
	if err != nil {
		panic(err)
	}
	block.SetNonce(solution)

	// Sanity-check: verify the block.
	if !VerifyPOW(block.Hash(), consensus.GenesisDifficulty) {
		panic("Genesis block POW solution is invalid.")
	}

	// Calculate work.
	work := CalculateWork(Bytes32ToBigInt(block.Hash()))

	fmt.Printf("Genesis block hash=%x work=%s\n", block.Hash(), work.String())

	// to block header
	// header := BlockHeader{
	// 	ParentHash:             block.ParentHash,
	// 	ParentTotalWork:        block.ParentTotalWork,
	// 	Difficulty:             block.Difficulty,
	// 	Timestamp:              block.Timestamp,
	// 	NumTransactions:        block.NumTransactions,
	// 	TransactionsMerkleRoot: block.TransactionsMerkleRoot,
	// 	Nonce:                  block.Nonce,
	// 	Graffiti:               block.Graffiti,
	// }
	// fmt.Printf("Genesis header block hash=%x\n", header.BlockHash())
	// fmt.Printf("ParentHash: %x\n", header.ParentHash)
	// fmt.Printf("ParentTotalWork: %x\n", header.ParentTotalWork)
	// fmt.Printf("Difficulty: %x\n", header.Difficulty)
	// fmt.Printf("Timestamp: %x\n", header.Timestamp)
	// fmt.Printf("NumTransactions: %d\n", header.NumTransactions)
	// fmt.Printf("TransactionsMerkleRoot: %x\n", header.TransactionsMerkleRoot)
	// fmt.Printf("Nonce: %x\n", header.Nonce)
	// fmt.Printf("Graffiti: %x\n", header.Graffiti)

	return block
}
