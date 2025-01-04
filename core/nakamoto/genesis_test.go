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
	block := GetRawGenesisBlockFromConfig(conf)
	genesisNonce := Bytes32ToBigInt(block.Nonce)

	// Print the hash.
	fmt.Printf("Genesis block hash: %x\n", block.Hash())

	// Check the genesis block.
	// find:GENESIS-BLOCK-ASSERTS
	assert.Equal(HexStringToBytes32("04ce8ce628e56bab073ff2298f1f9d0e96d31fb7a81f388d8fe6e4aa4dc1aaa8"), block.Hash())
	assert.Equal(conf.GenesisParentBlockHash, block.ParentHash)
	assert.Equal(BigIntToBytes32(*big.NewInt(0)), block.ParentTotalWork)
	assert.Equal(uint64(0), block.Timestamp)
	assert.Equal(uint64(1), block.NumTransactions)
	assert.Equal([32]uint8{0x59, 0xe0, 0xaa, 0xf, 0x1f, 0xe6, 0x6f, 0x3b, 0xe, 0xb0, 0xc, 0xa3, 0x31, 0x33, 0x1a, 0x69, 0x1, 0xc4, 0xc4, 0xa1, 0x21, 0x99, 0xba, 0xa0, 0x16, 0x77, 0xfd, 0xe2, 0xd4, 0xb7, 0xc6, 0x88}, block.TransactionsMerkleRoot)
	assert.Equal(big.NewInt(19).String(), genesisNonce.String())
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
	tx := MakeCoinbaseTx(wallet, GetBlockReward(0))

	// JSON dump.
	// str, err := json.Marshal(tx)
	// if err != nil {
	// 	panic(err)
	// }

	// Print as a Go-formatted RawTransaction{} for usage in genesis.go.
	fmt.Printf("Coinbase tx:\n")
	fmt.Printf("RawTransaction {\n")
	fmt.Printf("Version: %d,\n", tx.Version)
	fmt.Printf("Sig: %s,\n", formatByteArrayDynamic(tx.Sig[:]))
	fmt.Printf("FromPubkey: %s,\n", formatByteArrayDynamic(tx.FromPubkey[:]))
	fmt.Printf("ToPubkey: %s,\n", formatByteArrayDynamic(tx.ToPubkey[:]))
	fmt.Printf("Amount: %d,\n", tx.Amount)
	fmt.Printf("Fee: %d,\n", tx.Fee)
	fmt.Printf("Nonce: %d,\n", tx.Nonce)
	fmt.Printf("}\n")
}
