package nakamoto

import (
	"testing"
	"math/big"
	"github.com/stretchr/testify/assert"

)

func newBlockdag() BlockDAG {
	db, err := OpenDB(":memory:")
	if err != nil {
		panic(err)
	}

	blockdag, err := NewFromDB(db, nil)
	if err != nil {
		panic(err)
	}

	return blockdag
}

func getTestingWallets() ([]Wallet) {
	wallet1, err := WalletFromPrivateKey("2053e3c0d239d12a554ef55895b89e5d044af7d09d8be9a8f6da22460f8260ca")
	if err != nil {
		t.Fatalf("Failed to create wallet: %s", err)
	}
	return []Wallet{wallet1}
}

func TestOpenDB(t *testing.T) {
	// test not null
	_, err := OpenDB(":memory:")
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
		blockdag.IngestBlock(curr_block)

		// Create a new block.
		timestamp := uint64(0)
		curr_block = RawBlock{
			ParentHash: curr_block.Hash(),
			Timestamp: timestamp,
			NumTransactions: 0,
			Transactions: []RawTransaction{},
		}

		// Exit if the chain is long enough.
		if len(chain) >= 6 {
			break
		}
	}
}


func TestAddBlockUnknownParent(t *testing.T) {
	assert := assert.New(t)
	blockdag := newBlockdag()

	b := RawBlock{
		ParentHash: [32]byte{0xCA, 0xFE, 0xBA, 0xBE},
		Timestamp: 0,
		NumTransactions: 0,
		TransactionsMerkleRoot: [32]byte{0xCA, 0xFE, 0xBA, 0xBE},
		Nonce: [32]byte{0xBB},
		Transactions: []RawTransaction{},		
	}

	err := blockdag.IngestBlock(b)
	assert.Equal(err.Error(), "Unknown parent block.")
}

func TestAddBlockTxCount(t *testing.T) {
	assert := assert.New(t)
	blockdag := newBlockdag()

	b := RawBlock{
		ParentHash: [32]byte{0xCA, 0xFE, 0xBA, 0xBE},
		Timestamp: 0,
		NumTransactions: 0,
		TransactionsMerkleRoot: [32]byte{0xCA, 0xFE, 0xBA, 0xBE},
		Nonce: [32]byte{0xBB},
		Transactions: []RawTransaction{
			RawTransaction{
				Sig: [64]byte{0xCA, 0xFE, 0xBA, 0xBE},
				Data: []byte{0xCA, 0xFE, 0xBA, 0xBE},
			},
		},
	}

	err := blockdag.IngestBlock(b)
	assert.Equal(err.Error(), "Num transactions does not match length of transactions list.")
}

func TestAddBlockTxsValid(t *testing.T) {
	assert := assert.New(t)
	blockdag := newBlockdag()

	// Create a transaction.
	tx := RawTransaction{
		Sig: [64]byte{},
		Data: []byte{0xCA, 0xFE, 0xBA, 0xBE},
	}

	b := RawBlock{
		ParentHash: [32]byte{0xCA, 0xFE, 0xBA, 0xBE},
		Timestamp: 0,
		NumTransactions: 1,
		TransactionsMerkleRoot: [32]byte{0xCA, 0xFE, 0xBA, 0xBE},
		Nonce: [32]byte{0xBB},
		Transactions: []RawTransaction{
			tx
		},
	}

	err := blockdag.IngestBlock(b)
	assert.Equal(err.Error(), "Transaction 0 is invalid.")
}

func TestAddBlockTxMerkleRootValid(t *testing.T) {
	assert := assert.New(t)
	blockdag := newBlockdag()

	b := RawBlock{
		ParentHash: [32]byte{0xCA, 0xFE, 0xBA, 0xBE},
		Timestamp: 0,
		NumTransactions: 0,
		TransactionsMerkleRoot: [32]byte{0xCA, 0xFE, 0xBA, 0xBE},
		Nonce: [32]byte{0xBB},
		Transactions: []RawTransaction{
			RawTransaction{
				Sig: [64]byte{0xCA, 0xFE, 0xBA, 0xBE},
				Data: []byte{0xCA, 0xFE, 0xBA, 0xBE},
			},
		},
	}

	err := blockdag.IngestBlock(b)
	assert.Equal(err.Error(), "Merkle root does not match computed merkle root.")
}

func TestAddBlockPOWSolutionValid(t *testing.T) {
	
}

func TestAddBlockSuccess(t *testing.T) {
	assert := assert.New(t)
	blockdag := newBlockdag()

	// Create a tx with a valid signature.
	tx := RawTransaction{
		Sig: [64]byte{},
		Data: []byte{0xCA, 0xFE, 0xBA, 0xBE},
	}
	wallets := getTestingWallets()
	sig, err := wallets[0].Sign(tx.Data)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %s", err)
	}
	copy(tx.Sig[:], sig)

	b = RawBlock{
		ParentHash: [32]byte{0xCA, 0xFE, 0xBA, 0xBE},
		Timestamp: Timestamp(),
		NumTransactions: 1,
		TransactionsMerkleRoot: [32]byte{},
		Nonce: [32]byte{0xBB},
		Transactions: []RawTransaction{
			tx
		},
	}
	b.TransactionsMerkleRoot = ComputeMerkleHash([][]byte{tx.Envelope()})

	// Mine the POW solution.
	target := new(big.Int)
	target.SetString("0fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
	solution, err := SolvePOW(b, *new(big.Int), *target, 1000000000000)
	if err != nil {
		t.Fatalf("Failed to solve POW: %s", err)
	}
	b.SetNonce(solution)


	err := blockdag.IngestBlock(b)
	assert.Equal(err.Error(), "Transaction 0 is invalid.")
}

func TestGetBlock(t *testing.T) {
	// Insert a block into DAG.
	// Query DAG to verify block inserted.
}


