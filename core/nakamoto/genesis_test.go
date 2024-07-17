package nakamoto

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)


func TestGetRawGenesisBlockFromConfig(t *testing.T) {
	assert := assert.New(t)

	genesis_difficulty := new(big.Int)
	genesis_difficulty.SetString("0fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)

	conf := ConsensusConfig{
		EpochLengthBlocks:       5,
		TargetEpochLengthMillis: 2000,
		GenesisDifficulty:       *genesis_difficulty,
		// https://serhack.me/articles/story-behind-alternative-genesis-block-bitcoin/ ;)
		GenesisParentBlockHash: HexStringToBytes32("000006b15d1327d67e971d1de9116bd60a3a01556c91b6ebaa416ebc0cfaa646"),
		MaxBlockSizeBytes:      2 * 1024 * 1024, // 2MB
	}

	// Get the genesis block.
	genesisBlock := GetRawGenesisBlockFromConfig(conf)
	genesisNonce := Bytes32ToBigInt(genesisBlock.Nonce)

	// Print the hash.
	fmt.Printf("Genesis block hash: %x\n", genesisBlock.Hash())

	// Check the genesis block.
	assert.Equal(HexStringToBytes32("0877dbb50dc6df9056f4caf55f698d5451a38015f8e536e9c82ca3f5265c38c7"), genesisBlock.Hash())
	assert.Equal(conf.GenesisParentBlockHash, genesisBlock.ParentHash)
	assert.Equal(BigIntToBytes32(*big.NewInt(0)), genesisBlock.ParentTotalWork)
	assert.Equal(uint64(0), genesisBlock.Timestamp)
	assert.Equal(uint64(0), genesisBlock.NumTransactions)
	assert.Equal([32]byte{}, genesisBlock.TransactionsMerkleRoot)
	assert.Equal(big.NewInt(21).String(), genesisNonce.String())
}
