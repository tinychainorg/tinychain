package nakamoto

import (
	"fmt"
	"math/big"
)

// The Nakamoto consensus configuration, pertaining to difficulty readjustment, genesis block, and block size.
type ConsensusConfig struct {
	// The length of a difficulty epoch in blocks.
	EpochLengthBlocks uint64 `json:"epoch_length_blocks"`

	// The target length of one epoch in milliseconds.
	TargetEpochLengthMillis uint64 `json:"target_epoch_length_millis"`

	// The difficulty of the genesis block.
	GenesisDifficulty big.Int `json:"genesis_difficulty"`

	// The parent block hash for the genesis block. This is a special case, as the genesis block has a parent we don't know the preimage for.
	GenesisParentBlockHash [32]byte `json:"genesis_block_hash"`

	// Maximum block size.
	MaxBlockSizeBytes uint64 `json:"max_block_size_bytes"`
}

// Builds the raw genesis block from the consensus configuration.
//
// NOTE: This function essentially creates the genesis block from a short configuration.
// If the values are changed, the genesis hash will change, and a bunch of tests will fail / need to be updated with the new hash.
// These tests have been marked with the comment string find:GENESIS-BLOCK-ASSERTS so you can find them easily.
func GetRawGenesisBlockFromConfig(consensus ConsensusConfig) RawBlock {
	txs := []RawTransaction{
		// Genesis coinbase transaction.
		// Run `go test ./... -count=1 -v -run TestCreateGenesisCoinbaseTx` to generate a genesis coinbase tx.
		// This will output a coinbase transaction with a valid signature.
		RawTransaction{
			Version:    1,
			Sig:        [64]byte{0x86, 0xaf, 0x5f, 0x4b, 0x76, 0xea, 0x1c, 0xd2, 0xfb, 0xd4, 0x0f, 0xec, 0x93, 0x90, 0x70, 0x58, 0x47, 0xa1, 0x36, 0xb2, 0xc7, 0x0d, 0x10, 0x7b, 0xdd, 0x3e, 0x92, 0x27, 0xfd, 0xcb, 0x5e, 0xbb, 0x1c, 0x50, 0x0e, 0xfa, 0x02, 0x6a, 0x30, 0x44, 0x71, 0x15, 0xcc, 0x97, 0xf4, 0x15, 0x7f, 0x56, 0xd3, 0x3d, 0xb3, 0x30, 0xd6, 0x66, 0x06, 0xbb, 0xc1, 0x02, 0xae, 0x41, 0x39, 0xdb, 0x67, 0x93},
			FromPubkey: [65]byte{0x04, 0x61, 0xbf, 0x49, 0x39, 0x38, 0x55, 0xec, 0x77, 0x08, 0x1b, 0x61, 0xe1, 0xb1, 0x5d, 0x6a, 0xd9, 0x2b, 0x14, 0x26, 0x81, 0xe4, 0x0c, 0xeb, 0x07, 0x33, 0x4b, 0x63, 0x32, 0x73, 0x40, 0x2e, 0x24, 0xb2, 0x71, 0xc9, 0x14, 0x90, 0xc6, 0x39, 0x77, 0x5d, 0x0f, 0x00, 0x75, 0x9a, 0xc6, 0x1a, 0xf3, 0x5a, 0x4b, 0x24, 0xc6, 0x74, 0xf2, 0x81, 0x0c, 0xc1, 0x29, 0xfa, 0x04, 0x43, 0x6a, 0xa6, 0x84},
			ToPubkey:   [65]byte{0x04, 0x61, 0xbf, 0x49, 0x39, 0x38, 0x55, 0xec, 0x77, 0x08, 0x1b, 0x61, 0xe1, 0xb1, 0x5d, 0x6a, 0xd9, 0x2b, 0x14, 0x26, 0x81, 0xe4, 0x0c, 0xeb, 0x07, 0x33, 0x4b, 0x63, 0x32, 0x73, 0x40, 0x2e, 0x24, 0xb2, 0x71, 0xc9, 0x14, 0x90, 0xc6, 0x39, 0x77, 0x5d, 0x0f, 0x00, 0x75, 0x9a, 0xc6, 0x1a, 0xf3, 0x5a, 0x4b, 0x24, 0xc6, 0x74, 0xf2, 0x81, 0x0c, 0xc1, 0x29, 0xfa, 0x04, 0x43, 0x6a, 0xa6, 0x84},
			Amount:     5000000000,
			Fee:        0,
			Nonce:      0,
		},
	}
	block := RawBlock{
		// Special case: The genesis block has a parent we don't know the preimage for.
		ParentHash:             consensus.GenesisParentBlockHash,
		ParentTotalWork:        [32]byte{},
		Difficulty:             BigIntToBytes32(consensus.GenesisDifficulty),
		Timestamp:              0,
		NumTransactions:        1,
		TransactionsMerkleRoot: GetMerkleRootForTxs(txs),
		Nonce:                  [32]byte{},
		Graffiti:               [32]byte{0xca, 0xfe, 0xba, 0xbe, 0xde, 0xca, 0xfb, 0xad, 0xde, 0xad, 0xbe, 0xef}, // 0x cafebabe decafbad deadbeef
		Transactions:           txs,
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

	return block
}
