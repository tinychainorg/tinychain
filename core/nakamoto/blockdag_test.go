package nakamoto

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/liamzebedee/tinychain-go/core"
	"github.com/stretchr/testify/assert"
)

type MockStateMachine struct {}
func newMockStateMachine() *MockStateMachine {
	return &MockStateMachine{}
}
func (m *MockStateMachine) VerifyTx(tx RawTransaction) error {
	return nil
}


func newBlockdag() BlockDAG {
	// db, err := OpenDB(":memory:")
	db, err := OpenDB("test.sqlite3")
	if err != nil {
		panic(err)
	}

	stateMachine := newMockStateMachine()

	genesis_difficulty := new(big.Int)
	genesis_difficulty.SetString("0fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)

	// https://serhack.me/articles/story-behind-alternative-genesis-block-bitcoin/ ;) 
	genesisBlockHash_, err := hex.DecodeString("000006b15d1327d67e971d1de9116bd60a3a01556c91b6ebaa416ebc0cfaa646")
	if err != nil {
		panic(err)
	}
	genesisBlockHash := [32]byte{}
	copy(genesisBlockHash[:], genesisBlockHash_)

	conf := ConsensusConfig{
		EpochLengthBlocks: 5,
		TargetEpochLengthMillis: 2000,
		GenesisDifficulty: *genesis_difficulty,
		GenesisBlockHash: genesisBlockHash,
	}

	blockdag, err := NewBlockDAGFromDB(db, stateMachine, conf)
	if err != nil {
		panic(err)
	}

	return blockdag
}

func getTestingWallets(t *testing.T) ([]core.Wallet) {
	wallet1, err := core.WalletFromPrivateKey("2053e3c0d239d12a554ef55895b89e5d044af7d09d8be9a8f6da22460f8260ca")
	if err != nil {
		t.Fatalf("Failed to create wallet: %s", err)
	}
	return []core.Wallet{*wallet1}
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
	assert.Equal("Unknown parent block.", err.Error())
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
	assert.Equal("Num transactions does not match length of transactions list.", err.Error())
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
			tx,
		},
	}

	err := blockdag.IngestBlock(b)
	assert.Equal("Transaction 0 is invalid.", err.Error())
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
	assert.Equal("Merkle root does not match computed merkle root.", err.Error())
}

func TestAddBlockPOWSolutionValid(t *testing.T) {
	
}

func TestAddBlockSuccess(t *testing.T) {
	assert := assert.New(t)
	blockdag := newBlockdag()

	// Create a tx with a valid signature.
	tx := RawTransaction{
		FromPubkey: [64]byte{},
		Sig: [64]byte{},
		Data: []byte{0xCA, 0xFE, 0xBA, 0xBE},
	}
	sigHex := "1b7885066a4633c0ffe3e0cf4f6e8d77e2ec94f5eef46851e061aca8d3274f26f49a9dd28c0c1bc6c943b8663a57f9885bcee12fe1245f4bca3830e7927cddb7"
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		t.Fatalf("Failed to decode signature: %s", err)
	}
	copy(tx.Sig[:], sigBytes)

	nonceBigInt := big.NewInt(6)

	b := RawBlock{
		ParentHash: blockdag.consensus.GenesisBlockHash,
		Timestamp: 1719379532750,
		NumTransactions: 1,
		TransactionsMerkleRoot: [32]byte{},
		Nonce: [32]byte{},
		Transactions: []RawTransaction{
			tx,
		},
	}
	b.TransactionsMerkleRoot = core.ComputeMerkleHash([][]byte{tx.Envelope()})
	b.SetNonce(*nonceBigInt)

	err = blockdag.IngestBlock(b)
	assert.Equal(nil, err)
}

// This test creates a block from a signature created at runtime, and as such is non-deterministic.
// Creating a new signature will result in different solutions for the POW puzzle, since the blockhash is dependent on
// the merklized transaction list, whose hash will change based on the content of tx[0].Sig.
func TestAddBlockWithDynamicSignature(t *testing.T) {
	assert := assert.New(t)
	blockdag := newBlockdag()

	// Create a tx with a valid signature.
	tx := RawTransaction{
		FromPubkey: [64]byte{},
		Sig: [64]byte{},
		Data: []byte{0xCA, 0xFE, 0xBA, 0xBE},
	}
	wallets := getTestingWallets(t)
	tx.FromPubkey = wallets[0].PubkeyBytes()
	sig, err := wallets[0].Sign(tx.Data)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %s", err)
	}
	// Log the signature.
	t.Logf("Signature: %s\n", hex.EncodeToString(sig))
	copy(tx.Sig[:], sig)

	b := RawBlock{
		ParentHash: blockdag.consensus.GenesisBlockHash,
		Timestamp: 1719379532750,
		NumTransactions: 1,
		TransactionsMerkleRoot: [32]byte{},
		Nonce: [32]byte{},
		Transactions: []RawTransaction{
			tx,
		},
	}
	b.TransactionsMerkleRoot = core.ComputeMerkleHash([][]byte{tx.Envelope()})

	// Mine the POW solution.
	epoch, err := blockdag.GetEpochForBlockHash(b.ParentHash)
	if err != nil {
		t.Fatalf("Failed to get epoch for block hash: %s", err)
	}
	solution, err := SolvePOW(b, *big.NewInt(0), epoch.Difficulty, 1000000000000)
	if err != nil {
		t.Fatalf("Failed to solve POW: %s", err)
	}
	t.Logf("Solution: %s\n", solution.String())
	b.SetNonce(solution)

	err = blockdag.IngestBlock(b)
	if err != nil {
		t.Fatalf("Failed to ingest block: %s", err)
	}
	assert.Equal(nil, err)
}

func TestGetBlock(t *testing.T) {
	// Insert a block into DAG.
	// Query DAG to verify block inserted.
}


