// 
// This is the core implementation of the block DAG data structure.
// It mostly does these things:
// - ingests new blocks, validates transactions
// - manages reading/writing to the backing SQLite database.
// 
package nakamoto

import (
	"encoding/hex"
	"math/big"
	"testing"
	"database/sql"
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


func newBlockdag() (BlockDAG, ConsensusConfig, *sql.DB) {
	// See: https://stackoverflow.com/questions/77134000/intermittent-table-missing-error-in-sqlite-memory-database
	db, err := OpenDB("file:memdb1?mode=memory")
	// db, err := OpenDB("test.sqlite3")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("PRAGMA journal_mode = WAL;")
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
		MaxBlockSizeBytes: 2*1024*1024, // 2MB
	}

	blockdag, err := NewBlockDAGFromDB(db, stateMachine, conf)
	if err != nil {
		panic(err)
	}

	return blockdag, conf, db
}

func newValidTx(t *testing.T) (RawTransaction, error) {
	wallets := getTestingWallets(t)
	
	tx := RawTransaction{
		FromPubkey: [65]byte{},
		Sig: [64]byte{},
		Data: []byte{0xCA, 0xFE, 0xBA, 0xBE},
	}
	tx.FromPubkey = wallets[0].PubkeyBytes()

	envelope := tx.Envelope()
	sig, err := wallets[0].Sign(envelope)
	if err != nil {
		return RawTransaction{}, err
	}

	copy(tx.Sig[:], sig)

	// Sanity check verify.
	if !core.VerifySignature(wallets[0].PubkeyStr(), sig, envelope) {
		t.Fatalf("Failed to verify signature.")
	}

	return tx, nil
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
	blockdag, _, _ := newBlockdag()

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
	blockdag, consensus, _ := newBlockdag()

	b := RawBlock{
		ParentHash: consensus.GenesisBlockHash,
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
	blockdag, consensus, _ := newBlockdag()

	// Create a transaction.
	tx := RawTransaction{
		Sig: [64]byte{},
		Data: []byte{0xCA, 0xFE, 0xBA, 0xBE},
	}

	b := RawBlock{
		ParentHash: consensus.GenesisBlockHash,
		Timestamp: 0,
		NumTransactions: 1,
		TransactionsMerkleRoot: [32]byte{0xCA, 0xFE, 0xBA, 0xBE},
		Nonce: [32]byte{0xBB},
		Transactions: []RawTransaction{
			tx,
		},
	}

	err := blockdag.IngestBlock(b)
	assert.Equal("Transaction 0 is invalid: signature invalid.", err.Error())
}

func TestAddBlockTxMerkleRootValid(t *testing.T) {
	assert := assert.New(t)
	blockdag, consensus, _ := newBlockdag()

	tx, err := newValidTx(t)
	if err != nil {
		panic(err)
	}

	b := RawBlock{
		ParentHash: consensus.GenesisBlockHash,
		Timestamp: 0,
		NumTransactions: 1,
		TransactionsMerkleRoot: [32]byte{0xCA, 0xFE, 0xBA, 0xBE},
		Nonce: [32]byte{0xBB},
		Transactions: []RawTransaction{
			tx,
		},
	}

	err = blockdag.IngestBlock(b)
	assert.Equal("Merkle root does not match computed merkle root.", err.Error())
}

func TestAddBlockSuccess(t *testing.T) {
	assert := assert.New(t)
	blockdag, _, _ := newBlockdag()

	// Create a tx with a valid signature.
	wallets := getTestingWallets(t)
	tx := RawTransaction{
		FromPubkey: [65]byte{},
		Sig: [64]byte{},
		Data: []byte{0xCA, 0xFE, 0xBA, 0xBE},
	}
	tx.FromPubkey = wallets[0].PubkeyBytes()
	// sig, err := wallets[0].Sign(tx.Envelope())
	// if err != nil {
	// 	t.Fatalf("Failed to sign transaction: %s", err)
	// }
	// t.Logf("Signature: %s\n", hex.EncodeToString(sig))
	sigHex := "2b803490a6f14f9937f9ec49f6cc2de7bbb2e77ee54ea1b89d740352846e215601c1417f85c92553ed0972b259ac4009b8e3330c4cec9aded29e5fb2db9d89f2"
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		t.Fatalf("Failed to decode signature: %s", err)
	}
	copy(tx.Sig[:], sigBytes)

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
	// epoch, err := blockdag.GetEpochForBlockHash(b.ParentHash)
	// if err != nil {
	// 	t.Fatalf("Failed to get epoch for block hash: %s", err)
	// }
	// solution, err := SolvePOW(b, *big.NewInt(0), epoch.Difficulty, 1000000000000)
	// if err != nil {
	// 	t.Fatalf("Failed to solve POW: %s", err)
	// }
	// t.Logf("Solution: %s\n", solution.String())

	nonceBigInt := big.NewInt(44)
	b.SetNonce(*nonceBigInt)

	err = blockdag.IngestBlock(b)
	assert.Equal(nil, err)
}

// This test creates a block from a signature created at runtime, and as such is non-deterministic.
// Creating a new signature will result in different solutions for the POW puzzle, since the blockhash is dependent on
// the merklized transaction list, whose hash will change based on the content of tx[0].Sig.
func TestAddBlockWithDynamicSignature(t *testing.T) {
	assert := assert.New(t)
	blockdag, _, _ := newBlockdag()

	// Create a tx with a valid signature.
	tx := RawTransaction{
		FromPubkey: [65]byte{},
		Sig: [64]byte{},
		Data: []byte{0xCA, 0xFE, 0xBA, 0xBE},
	}
	wallets := getTestingWallets(t)
	tx.FromPubkey = wallets[0].PubkeyBytes()
	sig, err := wallets[0].Sign(tx.Envelope())
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


func TestBlockDAGInitialised(t *testing.T) {
	assert := assert.New(t)
	_, conf, db := newBlockdag()

	// Query the blocks column.
	rows, err := db.Query("SELECT hash, parent_hash, timestamp, num_transactions, transactions_merkle_root, nonce, height, epoch, size_bytes FROM blocks where hash = ?", conf.GenesisBlockHash[:])
	if err != nil {
		t.Fatalf("Failed to query blocks table: %s", err)
	}
	genesisBlock := Block{}
	for rows.Next() {
		hash := []byte{}
		parentHash := []byte{}
		transactionsMerkleRoot := []byte{}
		nonce := []byte{}

		err := rows.Scan(&hash, &parentHash, &genesisBlock.Timestamp, &genesisBlock.NumTransactions, &transactionsMerkleRoot, &nonce, &genesisBlock.Height, &genesisBlock.Epoch, &genesisBlock.SizeBytes)

		if err != nil {
			t.Fatalf("Failed to scan row: %s", err)
		}

		copy(genesisBlock.Hash[:], hash)
		copy(genesisBlock.ParentHash[:], parentHash)
		copy(genesisBlock.TransactionsMerkleRoot[:], transactionsMerkleRoot)
		copy(genesisBlock.Nonce[:], nonce)
	}
	rows.Close()

	// Check the genesis block.
	t.Logf("Genesis block: %v\n", genesisBlock.Hash)
	assert.Equal(conf.GenesisBlockHash, genesisBlock.Hash)
	assert.Equal([32]byte{}, genesisBlock.ParentHash)
	assert.Equal(uint64(0), genesisBlock.Timestamp)
	assert.Equal(uint64(0), genesisBlock.NumTransactions)
	assert.Equal([32]byte{}, genesisBlock.TransactionsMerkleRoot)
	assert.Equal([32]byte{}, genesisBlock.Nonce)
	assert.Equal(uint64(0), genesisBlock.Height)
	assert.Equal(GetIdForEpoch(conf.GenesisBlockHash, 0), genesisBlock.Epoch)

	// Query the epochs column.
	rows, err = db.Query("SELECT id, start_block_hash, start_time, start_height, difficulty FROM epochs")
	if err != nil {
		t.Fatalf("Failed to query epochs table: %s", err)
	}
	epoch := Epoch{}
	for rows.Next() {
		startBlockHash := []byte{}
		difficulty := []byte{}
		err := rows.Scan(&epoch.Id, &startBlockHash, &epoch.StartTime, &epoch.StartHeight, &difficulty)
		if err != nil {
			t.Fatalf("Failed to scan row: %s", err)
		}

		copy(epoch.StartBlockHash[:], startBlockHash)
		diffBytes32 := [32]byte{}
		copy(diffBytes32[:], difficulty)
		epoch.Difficulty = Bytes32ToBigInt(diffBytes32)
	}

	// Check the genesis epoch.
	t.Logf("Genesis epoch: %v\n", epoch.Id)

	assert.Equal(GetIdForEpoch(conf.GenesisBlockHash, 0), epoch.Id)
	assert.Equal(conf.GenesisBlockHash, epoch.StartBlockHash)
	assert.Equal(uint64(0), epoch.StartTime)
	assert.Equal(uint64(0), epoch.StartHeight)
	assert.Equal(conf.GenesisDifficulty, epoch.Difficulty)
}

func TestGetEpochForBlockHashGenesis(t *testing.T) {
	assert := assert.New(t)
	blockdag, _, _ := newBlockdag()

	// Test we can get the genesis epoch.
	epoch, err := blockdag.GetEpochForBlockHash(blockdag.consensus.GenesisBlockHash)
	assert.Equal(nil, err)
	assert.Equal(GetIdForEpoch(blockdag.consensus.GenesisBlockHash, 0), epoch.Id)
}

func TestGetBlockByHashGenesis(t *testing.T) {
	assert := assert.New(t)
	dag, conf, _ := newBlockdag()

	// Test we can get the genesis block.
	block, err := dag.GetBlockByHash(conf.GenesisBlockHash)
	assert.Equal(nil, err)

	// Check the genesis block.
	t.Logf("Genesis block: %v\n", block.Hash)
	assert.Equal(conf.GenesisBlockHash, block.Hash)
	assert.Equal([32]byte{}, block.ParentHash)
	assert.Equal(uint64(0), block.Timestamp)
	assert.Equal(uint64(0), block.NumTransactions)
	assert.Equal([32]byte{}, block.TransactionsMerkleRoot)
	assert.Equal([32]byte{}, block.Nonce)
	assert.Equal(uint64(0), block.Height)
	assert.Equal(GetIdForEpoch(conf.GenesisBlockHash, 0), block.Epoch)
}

func TestGetEpochForBlockHashNewBlock(t *testing.T) {
	assert := assert.New(t)
	blockdag, _, _ := newBlockdag()

	// Create a tx with a valid signature.
	tx := RawTransaction{
		FromPubkey: [65]byte{},
		Sig: [64]byte{},
		Data: []byte{0xCA, 0xFE, 0xBA, 0xBE},
	}
	wallets := getTestingWallets(t)
	tx.FromPubkey = wallets[0].PubkeyBytes()
	sig, err := wallets[0].Sign(tx.Envelope())
	if err != nil {
		t.Fatalf("Failed to sign transaction: %s", err)
	}
	// Log the signature.
	t.Logf("Signature: %s\n", hex.EncodeToString(sig))
	copy(tx.Sig[:], sig)

	raw := RawBlock{
		ParentHash: blockdag.consensus.GenesisBlockHash,
		Timestamp: 1719379532750,
		NumTransactions: 1,
		TransactionsMerkleRoot: [32]byte{},
		Nonce: [32]byte{},
		Transactions: []RawTransaction{
			tx,
		},
	}
	raw.TransactionsMerkleRoot = core.ComputeMerkleHash([][]byte{tx.Envelope()})

	// Mine the POW solution.
	epoch, err := blockdag.GetEpochForBlockHash(raw.ParentHash)
	if err != nil {
		t.Fatalf("Failed to get epoch for block hash: %s", err)
	}
	solution, err := SolvePOW(raw, *big.NewInt(0), epoch.Difficulty, 1000000000000)
	if err != nil {
		t.Fatalf("Failed to solve POW: %s", err)
	}
	t.Logf("Solution: %s\n", solution.String())
	raw.SetNonce(solution)

	err = blockdag.IngestBlock(raw)
	if err != nil {
		t.Fatalf("Failed to ingest block: %s", err)
	}
	assert.Equal(nil, err)

	// Verify block ingested into data store.
	block, err := blockdag.GetBlockByHash(raw.Hash())
	assert.Equal(nil, err)
	assert.Equal(raw.Hash(), block.Hash)
	assert.Equal(raw.ParentHash, block.ParentHash)
	assert.Equal(raw.Timestamp, block.Timestamp)
	assert.Equal(raw.NumTransactions, block.NumTransactions)
	assert.Equal(raw.TransactionsMerkleRoot, block.TransactionsMerkleRoot)
	assert.Equal(raw.Nonce, block.Nonce)
	assert.Equal(uint64(1), block.Height)
	assert.Equal(GetIdForEpoch(raw.ParentHash, 0), block.Epoch)
}

func TestGetCurrentTips(t *testing.T) {
	assert := assert.New(t)
	blockdag, _, _ := newBlockdag()

	// The genesis will be the first tip.
	current_tip, err := blockdag.GetCurrentTip()
	assert.Equal(nil, err)
	assert.Equal(blockdag.consensus.GenesisBlockHash, current_tip.Hash)

	// Mine a few blocks.
	tx, err := newValidTx(t)
	if err != nil {
		t.Fatalf("Failed to create valid tx: %s", err)
	}

	// Construct block template for mining.
	raw := RawBlock{
		ParentHash: current_tip.Hash,
		Timestamp: Timestamp(),
		NumTransactions: 1,
		TransactionsMerkleRoot: [32]byte{},
		Nonce: [32]byte{},
		Transactions: []RawTransaction{
			tx,
		},
	}
	raw.TransactionsMerkleRoot = core.ComputeMerkleHash([][]byte{tx.Envelope()})

	// Mine the POW solution.
	epoch, err := blockdag.GetEpochForBlockHash(raw.ParentHash)
	if err != nil {
		t.Fatalf("Failed to get epoch for block hash: %s", err)
	}
	solution, err := SolvePOW(raw, *big.NewInt(0), epoch.Difficulty, 1000000000000)
	if err != nil {
		t.Fatalf("Failed to solve POW: %s", err)
	}

	raw.SetNonce(solution)

	// Ingest the block.
	err = blockdag.IngestBlock(raw)
	if err != nil {
		t.Fatalf("Failed to ingest block: %s", err)
	}

	// Check if the block is the latest tip.
	current_tip, err = blockdag.GetCurrentTip()
	assert.Equal(nil, err)
	assert.Equal(raw.Hash(), current_tip.Hash)
}



func TestMinerProcedural(t *testing.T) {
	dag, conf, _ := newBlockdag()

	// Mine 10 blocks.
	current_tip := conf.GenesisBlockHash
	current_height := uint64(0)

	// Get genesis block.
	genesis, err := dag.GetBlockByHash(conf.GenesisBlockHash)
	if err != nil {
		t.Fatalf("Failed to get genesis block: %s", err)
	}
	current_height = genesis.Height + 1

	for i := 0; i < 10; i++ {
		tx, err := newValidTx(t)
		if err != nil {
			t.Fatalf("Failed to create valid tx: %s", err)
		}

		// Construct block template for mining.
		raw := RawBlock{
			ParentHash: current_tip,
			Timestamp: Timestamp(),
			NumTransactions: 1,
			TransactionsMerkleRoot: [32]byte{},
			Nonce: [32]byte{},
			Transactions: []RawTransaction{
				tx,
			},
		}
		raw.TransactionsMerkleRoot = core.ComputeMerkleHash([][]byte{tx.Envelope()})

		// Mine the POW solution.

		// First get the right epoch.
		var difficulty big.Int
		epoch, err := dag.GetEpochForBlockHash(raw.ParentHash)
		if err != nil {
			t.Fatalf("Failed to get epoch for block hash: %s", err)
		}
		if current_height % dag.consensus.EpochLengthBlocks == 0 {
			difficulty = RecomputeDifficulty(epoch.StartTime, raw.Timestamp, epoch.Difficulty, dag.consensus.TargetEpochLengthMillis, dag.consensus.EpochLengthBlocks, current_height)
		} else {
			difficulty = epoch.Difficulty
		}

		solution, err := SolvePOW(raw, *big.NewInt(0), difficulty, 1000000000000)
		if err != nil {
			t.Fatalf("Failed to solve POW: %s", err)
		}
		t.Logf("Solution: height=%d hash=%s nonce=%s\n", current_height, Bytes32ToString(raw.Hash()), solution.String())
		raw.SetNonce(solution)

		// Ingest the block.
		err = dag.IngestBlock(raw)
		if err != nil {
			t.Fatalf("Failed to ingest block: %s", err)
		}
		// Get the full block.
		block, err := dag.GetBlockByHash(raw.Hash())

		t.Logf("Solution: height=%d hash=%s nonce=%s acc_work=%s\n", current_height, Bytes32ToString(block.Hash), solution.String(), block.AccumulatedWork.String())

		current_tip = block.Hash
		current_height += 1
	}
}

func TestWTFSignature(t *testing.T) {
	for {
		_, err := newValidTx(t)
		if err != nil {
			panic(err)
		}
	}
}