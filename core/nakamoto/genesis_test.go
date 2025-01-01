package nakamoto

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/liamzebedee/tinychain-go/core"
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

func formatByteArrayDynamic(b []byte) string {
	out := fmt.Sprintf("[%d]byte{", len(b))
	for i, v := range b {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("0x%02x", v)
	}
	out += "}"
	return out
}

func TestWalletCreateSignTransferTx(t *testing.T) {
	wallet, err := core.CreateRandomWallet()
	if err != nil {
		panic(err)
	}
	tx := MakeCoinbaseTx(wallet, uint64(GetBlockReward(0)))

	// JSON dump.
	// str, err := json.Marshal(tx)
	// if err != nil {
	// 	panic(err)
	// }

	// Print as a Go-formatted RawTransaction{} for usage in genesis.go.
	fmt.Printf("Coinbase tx:\n")
	fmt.Printf("RawTransaction {\n")
	fmt.Printf("From: %v,\n", formatByteArrayDynamic(tx.FromPubkey[:]))
	fmt.Printf("To: %v,\n", formatByteArrayDynamic(tx.ToPubkey[:]))
	fmt.Printf("Sig: %v,\n", formatByteArrayDynamic(tx.Sig[:]))
	fmt.Printf("Amount: %d,\n", tx.Amount)
	fmt.Printf("Fee: %d,\n", tx.Fee)
	fmt.Printf("Nonce: %d,\n", tx.Nonce)
	fmt.Printf("}\n")
}
