package nakamoto

// This is the core implementation of the block DAG data structure.
// It mostly does these things:
// - ingests new blocks, validates transactions
// - manages reading/writing to the backing SQLite database.

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/mattn/go-sqlite3"

	"github.com/liamzebedee/tinychain-go/core"
	"github.com/stretchr/testify/assert"
)

type MockStateMachine struct{}

func newMockStateMachine() *MockStateMachine {
	return &MockStateMachine{}
}
func (m *MockStateMachine) VerifyTx(tx RawTransaction) error {
	return nil
}

func newBlockdag() (BlockDAG, ConsensusConfig, *sql.DB, RawBlock) {
	db, err := OpenDB(":memory:?journal_mode=WAL&synchronous=NORMAL&locking_mode=IMMEDIATE")
	// db, err := OpenDB("test.sqlite3")
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1) // :memory: only
	// Set WAL mode and synchronous to NORMAL
	_, err = db.Exec("PRAGMA journal_mode = WAL;")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("PRAGMA synchronous = NORMAL;")
	if err != nil {
		panic(err)
	}
	// Set transaction locking mode to IMMEDIATE
	_, err = db.Exec("PRAGMA locking_mode = IMMEDIATE;")
	if err != nil {
		panic(err)
	}
	// Set busy timeout to 5000 ms
	_, err = db.Exec("PRAGMA busy_timeout = 5000;")
	if err != nil {
		panic(err)
	}

	stateMachine := newMockStateMachine()

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

	genesisBlock := GetRawGenesisBlockFromConfig(conf)

	blockdag, err := NewBlockDAGFromDB(db, stateMachine, conf)
	if err != nil {
		panic(err)
	}

	return blockdag, conf, db, genesisBlock
}

func newValidTx(t *testing.T) (RawTransaction, error) {
	wallets := getTestingWallets(t)

	tx := RawTransaction{
		Version:    1,
		Sig:        [64]byte{},
		FromPubkey: wallets[0].PubkeyBytes(),
		ToPubkey:   [65]byte{},
		Amount:     0,
		Fee:        0,
		Nonce:      0,
	}

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

func getTestingWallets(t *testing.T) []core.Wallet {
	wallet1, err := core.WalletFromPrivateKey("2053e3c0d239d12a554ef55895b89e5d044af7d09d8be9a8f6da22460f8260ca")
	if err != nil {
		t.Fatalf("Failed to create wallet: %s", err)
	}
	wallet2, err := core.CreateRandomWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %s", err)
	}
	return []core.Wallet{*wallet1, *wallet2}
}

// Copies an in-memory SQLite database to a file.
// Thank you: https://rbn.im/backing-up-a-SQLite-database-with-Go/backing-up-a-SQLite-database-with-Go.html
func backupDBToFile(destDb, srcDb *sql.DB) error {
	destConn, err := destDb.Conn(context.Background())
	if err != nil {
		return err
	}

	srcConn, err := srcDb.Conn(context.Background())
	if err != nil {
		return err
	}

	return destConn.Raw(func(destConn interface{}) error {
		return srcConn.Raw(func(srcConn interface{}) error {
			destSQLiteConn, ok := destConn.(*sqlite3.SQLiteConn)
			if !ok {
				return fmt.Errorf("can't convert destination connection to SQLiteConn")
			}

			srcSQLiteConn, ok := srcConn.(*sqlite3.SQLiteConn)
			if !ok {
				return fmt.Errorf("can't convert source connection to SQLiteConn")
			}

			b, err := destSQLiteConn.Backup("main", srcSQLiteConn, "main")
			if err != nil {
				return fmt.Errorf("error initializing SQLite backup: %w", err)
			}

			done, err := b.Step(-1)
			if !done {
				return fmt.Errorf("step of -1, but not done")
			}
			if err != nil {
				return fmt.Errorf("error in stepping backup: %w", err)
			}

			err = b.Finish()
			if err != nil {
				return fmt.Errorf("error finishing backup: %w", err)
			}

			return err
		})
	})
}

// Usage: saveDbForInspection(blockdag.db, "testing.db")
func saveDbForInspection(db *sql.DB) error {
	// Backup DB to file.
	backupDb, err := OpenDB("testing.db")
	if err != nil {
		return fmt.Errorf("Failed to open backup database: %s", err)
	}
	err = backupDBToFile(backupDb, db)
	if err != nil {
		return fmt.Errorf("Failed to backup database: %s", err)
	}
	return nil
}

func TestOpenDB(t *testing.T) {
	// test not null
	_, err := OpenDB(":memory:")
	if err != nil {
		t.Log(err)
	}
}

func TestDagLatestTipIsSet(t *testing.T) {
	assert := assert.New(t)
	dag, _, _, genesisBlock := newBlockdag()

	// The genesis block should be the latest tip.
	// FIXME
	assert.Equal(genesisBlock.Hash(), dag.FullTip.Hash)
}

func TestDagAddBlockUnknownParent(t *testing.T) {
	assert := assert.New(t)
	blockdag, _, _, _ := newBlockdag()

	b := RawBlock{
		ParentHash:             [32]byte{0xCA, 0xFE, 0xBA, 0xBE},
		Timestamp:              0,
		NumTransactions:        0,
		TransactionsMerkleRoot: [32]byte{0xCA, 0xFE, 0xBA, 0xBE},
		Nonce:                  [32]byte{0xBB},
		Transactions:           []RawTransaction{},
	}

	err := blockdag.IngestBlock(b)
	assert.Equal("Unknown parent block.", err.Error())
}

func TestDagAddBlockTxCount(t *testing.T) {
	assert := assert.New(t)
	blockdag, _, _, genesisBlock := newBlockdag()

	tx, err := newValidTx(t)
	if err != nil {
		panic(err)
	}

	b := RawBlock{
		ParentHash:             genesisBlock.Hash(),
		Timestamp:              0,
		NumTransactions:        0,
		TransactionsMerkleRoot: [32]byte{0xCA, 0xFE, 0xBA, 0xBE},
		Nonce:                  [32]byte{0xBB},
		Transactions: []RawTransaction{
			tx,
		},
	}

	err = blockdag.IngestBlock(b)
	assert.Equal("Num transactions does not match length of transactions list.", err.Error())
}

func TestDagAddBlockTxsValid(t *testing.T) {
	assert := assert.New(t)
	blockdag, _, _, genesisBlock := newBlockdag()

	// Create a transaction.
	tx, err := newValidTx(t)
	if err != nil {
		panic(err)
	}
	// Set invalid signature.
	tx.Sig = [64]byte{0xCA, 0xFE, 0xBA, 0xBE}

	b := RawBlock{
		ParentHash:             genesisBlock.Hash(),
		Timestamp:              0,
		NumTransactions:        1,
		TransactionsMerkleRoot: [32]byte{0xCA, 0xFE, 0xBA, 0xBE},
		Nonce:                  [32]byte{0xBB},
		Transactions: []RawTransaction{
			tx,
		},
	}

	err = blockdag.IngestBlock(b)
	assert.Equal("Transaction 0 is invalid: signature invalid.", err.Error())
}

func TestDagAddBlockTxMerkleRootValid(t *testing.T) {
	assert := assert.New(t)
	blockdag, _, _, genesisBlock := newBlockdag()

	tx, err := newValidTx(t)
	if err != nil {
		panic(err)
	}

	b := RawBlock{
		ParentHash:             genesisBlock.Hash(),
		Timestamp:              0,
		NumTransactions:        1,
		TransactionsMerkleRoot: [32]byte{0xCA, 0xFE, 0xBA, 0xBE},
		Nonce:                  [32]byte{0xBB},
		Transactions: []RawTransaction{
			tx,
		},
	}

	err = blockdag.IngestBlock(b)
	assert.Equal("Merkle root does not match computed merkle root.", err.Error())
}

func TestDagAddBlockSuccess(t *testing.T) {
	assert := assert.New(t)
	blockdag, _, _, genesisBlock := newBlockdag()

	// Create a tx with a valid signature.
	wallets := getTestingWallets(t)
	tx := RawTransaction{
		Version:    1,
		Sig:        [64]byte{},
		FromPubkey: wallets[0].PubkeyBytes(),
		ToPubkey:   [65]byte{},
		Amount:     0,
		Fee:        0,
		Nonce:      0,
	}
	tx.FromPubkey = wallets[0].PubkeyBytes()

	// sig, err := wallets[0].Sign(tx.Envelope())
	// if err != nil {
	// 	t.Fatalf("Failed to sign transaction: %s", err)
	// }
	// t.Logf("Signature: %s\n", hex.EncodeToString(sig))

	sigHex := "084401618c78b2e778cba17eb04892331f1f69f860c24f039e1da6b959830ab567efc3fb2403af2c63c65a17648348211ce2fb5251f038d5151dff67152d1f6a"
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		t.Fatalf("Failed to decode signature: %s", err)
	}
	copy(tx.Sig[:], sigBytes)

	b := RawBlock{
		ParentHash:             genesisBlock.Hash(),
		ParentTotalWork:        BigIntToBytes32(*CalculateWork(Bytes32ToBigInt(genesisBlock.Hash()))),
		Timestamp:              1719379532750,
		NumTransactions:        1,
		TransactionsMerkleRoot: [32]byte{},
		Nonce:                  [32]byte{},
		Graffiti:               [32]byte{},
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
	assert.Equal(nil, err)
}

// This test creates a block from a signature created at runtime, and as such is non-deterministic.
// Creating a new signature will result in different solutions for the POW puzzle, since the blockhash is dependent on
// the merklized transaction list, whose hash will change based on the content of tx[0].Sig.
func TestDagAddBlockWithDynamicSignature(t *testing.T) {
	assert := assert.New(t)
	blockdag, _, _, genesisBlock := newBlockdag()

	// Create a tx with a valid signature.
	tx, err := newValidTx(t)
	if err != nil {
		panic(err)
	}

	// Log the signature.
	t.Logf("Signature: %s\n", hex.EncodeToString(tx.Sig[:]))

	b := RawBlock{
		ParentHash:             genesisBlock.Hash(),
		ParentTotalWork:        BigIntToBytes32(*CalculateWork(Bytes32ToBigInt(genesisBlock.Hash()))),
		Timestamp:              1719379532750,
		NumTransactions:        1,
		TransactionsMerkleRoot: [32]byte{},
		Nonce:                  [32]byte{},
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

func TestDagGetBlockByHashGenesis(t *testing.T) {
	assert := assert.New(t)
	dag, conf, _, genesisBlock := newBlockdag()

	// Test we can get the genesis block.
	block, err := dag.GetBlockByHash(genesisBlock.Hash())
	assert.Equal(nil, err)

	// Check the genesis block.
	t.Logf("Genesis block: %v\n", block.Hash)
	t.Logf("Genesis block size: %d\n", block.SizeBytes)

	// RawBlock.
	genesisNonce := Bytes32ToBigInt(genesisBlock.Nonce)
	assert.Equal(conf.GenesisParentBlockHash, block.ParentHash)
	assert.Equal(uint64(0), block.Timestamp)
	assert.Equal(uint64(0), block.NumTransactions)
	assert.Equal([32]byte{}, block.TransactionsMerkleRoot)
	assert.Equal(big.NewInt(21).String(), genesisNonce.String())
	// Block.
	assert.Equal(uint64(0), block.Height)
	assert.Equal(GetIdForEpoch(genesisBlock.Hash(), 0), block.Epoch)
	assert.Equal(uint64(208), block.SizeBytes)
	assert.Equal(HexStringToBytes32("0877dbb50dc6df9056f4caf55f698d5451a38015f8e536e9c82ca3f5265c38c7"), block.Hash)
	t.Logf("Block: acc_work=%s\n", block.AccumulatedWork.String())
	assert.Equal(big.NewInt(30).String(), block.AccumulatedWork.String())
}

func TestDagBlockDAGInitialised(t *testing.T) {
	assert := assert.New(t)
	_, conf, db, genesisBlock := newBlockdag()

	// Query the blocks column.
	genesisBlockHash := genesisBlock.Hash()
	rows, err := db.Query("select hash, parent_hash, parent_total_work, timestamp, num_transactions, transactions_merkle_root, nonce, graffiti, height, epoch, size_bytes, acc_work from blocks where hash = ? limit 1", genesisBlockHash[:])
	if err != nil {
		t.Fatalf("Failed to query blocks table: %s", err)
	}
	block := Block{}

	assert.True(rows.Next(), "Failed to find genesis block in database.")
	{
		hash := []byte{}
		parentHash := []byte{}
		transactionsMerkleRoot := []byte{}
		nonce := []byte{}
		graffiti := []byte{}
		accWorkBuf := []byte{}
		parentTotalWorkBuf := []byte{}

		err = rows.Scan(
			&hash,
			&parentHash,
			&parentTotalWorkBuf,
			&block.Timestamp,
			&block.NumTransactions,
			&transactionsMerkleRoot,
			&nonce,
			&graffiti,
			&block.Height,
			&block.Epoch,
			&block.SizeBytes,
			&accWorkBuf,
		)

		if err != nil {
			t.Fatalf("Failed to scan row: %s", err)
		}

		// Debug log the hash.
		t.Logf("DB Hash: %s\n", hex.EncodeToString(hash))

		copy(block.Hash[:], hash)
		copy(block.ParentHash[:], parentHash)
		copy(block.TransactionsMerkleRoot[:], transactionsMerkleRoot)
		copy(block.Nonce[:], nonce)
		copy(block.Graffiti[:], graffiti)

		accWork := [32]byte{}
		copy(accWork[:], accWorkBuf)
		block.AccumulatedWork = Bytes32ToBigInt(accWork)

		parentTotalWork := [32]byte{}
		copy(parentTotalWork[:], parentTotalWorkBuf)
		block.ParentTotalWork = Bytes32ToBigInt(parentTotalWork)
	}

	rows.Close()

	// Compute the acc work.
	rawGenesisBlock := GetRawGenesisBlockFromConfig(conf)
	accWork := CalculateWork(Bytes32ToBigInt(rawGenesisBlock.Hash()))
	t.Logf("Genesis block: %v\n", Bytes32ToHexString(rawGenesisBlock.Hash()))
	t.Logf("Genesis block acc work: %s\n", accWork.String())

	t.Logf("Block: %v\n", block.Hash)

	// Check the genesis block.
	genesisNonce := Bytes32ToBigInt(genesisBlock.Nonce)
	assert.Equal(genesisBlock.Hash(), block.Hash)
	assert.Equal(conf.GenesisParentBlockHash, block.ParentHash)
	assert.Equal(big.NewInt(0).String(), block.ParentTotalWork.String())
	assert.Equal(uint64(0), block.Timestamp)
	assert.Equal(uint64(0), block.NumTransactions)
	assert.Equal([32]byte{}, block.TransactionsMerkleRoot)
	assert.Equal(big.NewInt(21).String(), genesisNonce.String())
	assert.Equal(uint64(0), block.Height)
	assert.Equal(GetIdForEpoch(genesisBlock.Hash(), 0), block.Epoch)

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
	assert.Equal(GetIdForEpoch(genesisBlock.Hash(), 0), epoch.Id)
	assert.Equal(HexStringToBytes32("0877dbb50dc6df9056f4caf55f698d5451a38015f8e536e9c82ca3f5265c38c7"), epoch.StartBlockHash)
	assert.Equal(uint64(0), epoch.StartTime)
	assert.Equal(uint64(0), epoch.StartHeight)
	assert.Equal(conf.GenesisDifficulty, epoch.Difficulty)
}

func TestDagGetEpochForBlockHashGenesis(t *testing.T) {
	assert := assert.New(t)
	blockdag, _, _, genesisBlock := newBlockdag()

	// Test we can get the genesis epoch.
	epoch, err := blockdag.GetEpochForBlockHash(genesisBlock.Hash())
	assert.Equal(nil, err)
	assert.Equal(GetIdForEpoch(genesisBlock.Hash(), 0), epoch.Id)
}

func TestDagGetEpochForBlockHashNewBlock(t *testing.T) {
	assert := assert.New(t)
	blockdag, _, _, genesisBlock := newBlockdag()

	// Create a tx with a valid signature.
	tx, err := newValidTx(t)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %s", err)
	}

	raw := RawBlock{
		ParentHash:             genesisBlock.Hash(),
		ParentTotalWork:        BigIntToBytes32(*CalculateWork(Bytes32ToBigInt(genesisBlock.Hash()))),
		Timestamp:              1719379532750,
		NumTransactions:        1,
		TransactionsMerkleRoot: [32]byte{},
		Nonce:                  [32]byte{},
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

func TestDagGetLatestTip(t *testing.T) {
	assert := assert.New(t)
	blockdag, _, _, genesisBlock := newBlockdag()

	// The genesis will be the first tip.
	current_tip, err := blockdag.GetLatestFullTip()
	assert.Equal(nil, err)
	assert.Equal(genesisBlock.Hash(), current_tip.Hash)

	// Mine a few blocks.
	tx, err := newValidTx(t)
	if err != nil {
		t.Fatalf("Failed to create valid tx: %s", err)
	}

	// Construct block template for mining.
	raw := RawBlock{
		ParentHash:             current_tip.Hash,
		ParentTotalWork:        BigIntToBytes32(*CalculateWork(Bytes32ToBigInt(genesisBlock.Hash()))),
		Timestamp:              Timestamp(),
		NumTransactions:        1,
		TransactionsMerkleRoot: [32]byte{},
		Nonce:                  [32]byte{},
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
	current_tip, err = blockdag.GetLatestFullTip()
	assert.Equal(nil, err)
	assert.Equal(raw.Hash(), current_tip.Hash)

	// Check the in-memory latest tip is updated.
	assert.Equal(raw.Hash(), blockdag.FullTip.Hash)
}

func TestMinerProcedural(t *testing.T) {
	dag, _, _, genesisBlock := newBlockdag()

	// Mine 10 blocks.
	current_tip := genesisBlock.Hash()
	current_height := uint64(0)

	// Get genesis block.
	genesis, err := dag.GetBlockByHash(genesisBlock.Hash())
	if err != nil {
		t.Fatalf("Failed to get genesis block: %s", err)
	}
	// genesis total work.
	t.Logf("Genesis total work: %s\n", genesis.AccumulatedWork.String())
	current_height = genesis.Height + 1

	// Genesis block has 1 accumulated work.
	acc_work := genesis.AccumulatedWork

	for i := 0; i < 10; i++ {
		tx, err := newValidTx(t)
		if err != nil {
			t.Fatalf("Failed to create valid tx: %s", err)
		}

		// Construct block template for mining.
		raw := RawBlock{
			ParentHash:             current_tip,
			ParentTotalWork:        BigIntToBytes32(acc_work),
			Timestamp:              Timestamp(),
			NumTransactions:        1,
			TransactionsMerkleRoot: [32]byte{},
			Nonce:                  [32]byte{},
			Transactions: []RawTransaction{
				tx,
			},
		}
		raw.TransactionsMerkleRoot = core.ComputeMerkleHash([][]byte{tx.Envelope()})

		t.Logf("Mining block height=%d parentTotalWork=%s\n", current_height, acc_work.String())

		// Mine the POW solution.

		// First get the right epoch.
		var difficulty big.Int
		epoch, err := dag.GetEpochForBlockHash(raw.ParentHash)
		if err != nil {
			t.Fatalf("Failed to get epoch for block hash: %s", err)
		}
		if current_height%dag.consensus.EpochLengthBlocks == 0 {
			difficulty = RecomputeDifficulty(epoch.StartTime, raw.Timestamp, epoch.Difficulty, dag.consensus.TargetEpochLengthMillis, dag.consensus.EpochLengthBlocks, current_height)
		} else {
			difficulty = epoch.Difficulty
		}

		// Solve the POW puzzle.
		solution, err := SolvePOW(raw, *big.NewInt(0), difficulty, 1000000000000)
		if err != nil {
			t.Fatalf("Failed to solve POW: %s", err)
		}
		raw.SetNonce(solution)

		// Check the acc work.
		work := CalculateWork(Bytes32ToBigInt(raw.Hash()))
		acc_work = *acc_work.Add(&acc_work, work)
		current_height += 1

		t.Logf("Solution: height=%d hash=%s nonce=%s acc_work=%s\n", current_height, Bytes32ToString(raw.Hash()), solution.String(), acc_work.String())

		// Ingest the block.
		err = dag.IngestBlock(raw)
		if err != nil {
			t.Fatalf("Failed to ingest block: %s", err)
		}

		// Print log.
		block, err := dag.GetBlockByHash(raw.Hash())
		current_tip = block.Hash
	}
}

func newBlockdagLongEpoch() (BlockDAG, ConsensusConfig, *sql.DB) {
	db, err := OpenDB(":memory:")
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1) // :memory: only
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
		EpochLengthBlocks:       20000,
		TargetEpochLengthMillis: 1000,
		GenesisDifficulty:       *genesis_difficulty,
		GenesisParentBlockHash:  genesisBlockHash,
		MaxBlockSizeBytes:       2 * 1024 * 1024, // 2MB
	}

	blockdag, err := NewBlockDAGFromDB(db, stateMachine, conf)
	if err != nil {
		panic(err)
	}

	return blockdag, conf, db
}

// 2s to mine 10,000 blocks.
func TestDagGetLongestChainHashList(t *testing.T) {
	assert := assert.New(t)
	dag, _, _ := newBlockdagLongEpoch()

	// Insert 10,000 blocks.
	var N_BLOCKS int64 = 300
	minerWallet, err := core.CreateRandomWallet()
	if err != nil {
		t.Fatalf("Failed to create miner wallet: %s", err)
	}

	expectedHashList := [][32]byte{}
	miner := NewMiner(dag, minerWallet)
	miner.OnBlockSolution = func(block RawBlock) {
		err := dag.IngestBlock(block)
		if err != nil {
			t.Fatalf("Failed to ingest block: %s", err)
		}
		expectedHashList = append(expectedHashList, block.Hash())
	}
	miner.Start(N_BLOCKS)

	// Get the tip.
	tip, err := dag.GetLatestFullTip()
	if err != nil {
		t.Fatalf("Failed to get tip: %s", err)
	}

	// Assert tip is at height 10,000.
	assert.Equal(uint64(N_BLOCKS), tip.Height)

	// Now get the longest chain hash list.
	var depthFromTip uint64 = 100 // get the most recent 1000 of the 10,000
	hashlist, err := dag.GetLongestChainHashList(tip.Hash, depthFromTip)
	if err != nil {
		t.Fatalf("Failed to get longest chain hash list: %s", err)
	}

	t.Logf("Longest chain - expected - hash list: len=%d\n", len(expectedHashList))
	t.Logf("Longest chain - actual - hash list: len=%d\n", len(hashlist))

	// Print both hashlists, line by line, with each hash
	// printed in hex format.
	for i, hash := range expectedHashList[uint64(len(expectedHashList))-depthFromTip:] {
		t.Logf("block #%d: expected=%s actual=%s\n", i, Bytes32ToString(hash), Bytes32ToString(hashlist[i]))
	}

	// assert the two hashlists are the same one-by-one
	for i, hash := range expectedHashList[uint64(len(expectedHashList))-depthFromTip:] {
		assert.Equal(Bytes32ToString(hash), Bytes32ToString(hashlist[i]))
	}
}

// Tests get path on a simple chain with no branches. ie. a -> b -> c -> d -> e
func TestDagGetPathSingleChain(t *testing.T) {
	// Testing the GetPath function is done differently to the other tests.
	// It is a bit of a hack for the time being.
	// Basically to setup the test state:
	// - we mine a few blocks
	// - we test we can get the path from any point forwards and reverse in that path
	// - then we mine a couple blocks on height-1 (ie. an alternative branch)
	// - and we test the getpath function here too.
	assert := assert.New(t)
	dag, _, _, genesisBlock := newBlockdag()
	wallet := getTestingWallets(t)[0]
	miner := NewMiner(dag, &wallet)
	miner.OnBlockSolution = func(block RawBlock) {
		err := dag.IngestBlock(block)
		if err != nil {
			t.Fatalf("Failed to ingest block: %s", err)
		}
	}

	// Mine 3 blocks.
	blocksMined := miner.Start(3)
	if len(blocksMined) != 3 {
		t.Fatalf("Failed to mine 3 blocks.")
	}

	// Log the mined blocks.
	fullChain := []RawBlock{}
	t.Logf("Blockchain:\n")
	fullChain = append(fullChain, genesisBlock)
	fullChain = append(fullChain, blocksMined...)
	for i, block := range fullChain {
		t.Logf("block #%d: %s\n", i+1, block.HashStr())
	}

	// Get path.
	tip, err := dag.GetLatestFullTip()
	if err != nil {
		t.Fatalf("Failed to get tip: %s", err)
	}

	// (1) Get the path from the tip to the genesis (backwards traversal).
	path, err := dag.GetPath(tip.Hash, 4, -1)
	if err != nil {
		t.Fatalf("Failed to get path: %s", err)
	}

	// 1a. Check the path is of correct length.
	assert.Equal(4, len(path))

	// 1b. Check the path is correct and is in traversal order.
	for i, blockHash := range path {
		t.Logf("Path block #%d: %x\n", i+1, blockHash)
		// We asked for the path from tip to genesis, so the path should be in reverse order.
		assert.Equal(fullChain[len(fullChain)-1-i].HashStr(), Bytes32ToHexString(blockHash))
	}

	// (2) Get the path from the genesis to the tip (forwards traversal).
	path, err = dag.GetPath(genesisBlock.Hash(), 4, 1)
	if err != nil {
		t.Fatalf("Failed to get path: %s", err)
	}

	// 2a. Check the path is of correct length.
	assert.Equal(4, len(path))

	// 2b. Check the path is correct and is in traversal order.
	for i, blockHash := range path {
		t.Logf("Path block #%d: %x\n", i+1, blockHash)
		// We asked for the path from tip to genesis, so the path should be in reverse order.
		assert.Equal(fullChain[i].HashStr(), Bytes32ToHexString(blockHash))
	}
}

// Tests get path on a complex chain with one branch.
func TestDagGetPathComplexChain(t *testing.T) {
	// Testing the GetPath function is done differently to the other tests.
	// It is a bit of a hack for the time being.
	// Basically to setup the test state:
	// - we mine a few blocks
	// - we test we can get the path from any point forwards and reverse in that path
	// - then we mine a couple blocks on height-1 (ie. an alternative branch)
	// - and we test the getpath function here too.
	assert := assert.New(t)
	dag, _, _, genesisBlock := newBlockdag()
	wallet := getTestingWallets(t)[0]
	miner := NewMiner(dag, &wallet)
	miner.OnBlockSolution = func(block RawBlock) {
		err := dag.IngestBlock(block)
		if err != nil {
			t.Fatalf("Failed to ingest block: %s", err)
		}
	}

	// Mine 3 blocks.
	blocksMined := miner.Start(3)
	if len(blocksMined) != 3 {
		t.Fatalf("Failed to mine 3 blocks.")
	}

	// Log the mined blocks.
	fullChain := []RawBlock{}
	t.Logf("Blockchain:\n")
	fullChain = append(fullChain, genesisBlock)
	fullChain = append(fullChain, blocksMined...)
	for i, block := range fullChain {
		t.Logf("block #%d: %s\n", i+1, block.HashStr())
	}

	// Get path.
	tip, err := dag.GetLatestFullTip()
	if err != nil {
		t.Fatalf("Failed to get tip: %s", err)
	}

	// (1) Get the path from the tip to the genesis (backwards traversal).
	path, err := dag.GetPath(tip.Hash, 4, -1)
	if err != nil {
		t.Fatalf("Failed to get path: %s", err)
	}

	// 1a. Check the path is of correct length.
	assert.Equal(4, len(path))

	// (2) Insert a new branch.
	altBranchBaseBlock, err := dag.GetBlockByHash(blocksMined[0].Hash())
	if err != nil {
		t.Fatalf("Failed to get block: %s", err)
	}
	var alternativeTipForMining Block = *altBranchBaseBlock
	miner.GetTipForMining = func() Block {
		return alternativeTipForMining
	}
	miner.OnBlockSolution = func(block RawBlock) {
		err := dag.IngestBlock(block)
		if err != nil {
			t.Fatalf("Failed to ingest block: %s", err)
		}
		blk, err := dag.GetBlockByHash(block.Hash())
		if err != nil {
			t.Fatalf("Failed to get block: %s", err)
		}
		alternativeTipForMining = *blk
	}
	altBranchBlocks := miner.Start(15)

	// (3) Log the two chains.
	t.Logf("First branch:\n")
	for i, block := range fullChain {
		t.Logf("block #%d: %s\n", i+1, block.HashStr())
	}
	t.Log()

	altBranch := []RawBlock{}
	t.Logf("Alternative branch:\n")
	altBranch = append(altBranch, genesisBlock)
	altBranch = append(altBranch, blocksMined[0])
	altBranch = append(altBranch, altBranchBlocks...)
	for i, block := range altBranch {
		t.Logf("block #%d: %s\n", i+1, block.HashStr())
	}

	// Log the accumulated work on the tips of both branches.
	firstBranchTip, _ := dag.GetBlockByHash(fullChain[len(fullChain)-1].Hash())
	secondBranchTip, _ := dag.GetBlockByHash(altBranch[len(altBranch)-1].Hash())
	t.Logf("First branch tip acc work: %s\n", firstBranchTip.AccumulatedWork.String())
	t.Logf("Second branch tip acc work: %s\n", secondBranchTip.AccumulatedWork.String())

	// Check the current tip.
	tip, err = dag.GetLatestFullTip()
	if err != nil {
		t.Fatalf("Failed to get tip: %s", err)
	}
	t.Logf("Current tip: %s\n", tip.HashStr())

	// Assert that the second branch has more work.
	assert.True(secondBranchTip.AccumulatedWork.Cmp(&firstBranchTip.AccumulatedWork) > 0)

	//
	// TESTS BEGIN HERE.
	//

	// We will be testing these things:
	// - traverse forward from genesis to first branch tip
	// - traverse backwards from first branch tip to genesis
	// - traverse forward from genesis to second branch tip
	// - traverse backwards from second branch tip to genesis
	//

	// (4) Get the path from the tip to the genesis (backwards traversal).
	path, err = dag.GetPath(genesisBlock.Hash(), 17, 1)
	if err != nil {
		t.Fatalf("Failed to get path: %s", err)
	}
	// 1a. Check the path is of correct length.
	assert.Equal(17, len(path))

	// 1b. Check the path is correct and is in traversal order.
	for i, blockHash := range path {
		t.Logf("Path block #%d: %x\n", i+1, blockHash)
		assert.Equal(altBranch[i].HashStr(), Bytes32ToHexString(blockHash))
	}

	/*
				It was at this point I learnt, there is no way to implement forwards iteration for the GetPath function.

				Why? Because you cannot know in the middle of a chain whether you are on the heaviest chain. Because the accumulated work may be low for the (n+1)th block, but then peak in the (n+2)th block.

				And here's the test log to prove it:

				The second branch tip may be the heaviest chain, but while getting the path from genesis, we inadvertently select the block in the first branch because it beats the accumulated work of the block from the alternative branch.

				    blockdag_test.go:983: First branch:
		    blockdag_test.go:985: block #1: 0877dbb50dc6df9056f4caf55f698d5451a38015f8e536e9c82ca3f5265c38c7
		    blockdag_test.go:985: block #2: 0abb5a235c89d9ab6c3578917b116958174601d0f0e873ccf0356430a5e68731
		    blockdag_test.go:985: block #3: 01790765a25dad409399744cc320c05080f21b2b503987871ebc4627285e86e9
		    blockdag_test.go:985: block #4: 06e1fe2a52f70b999334f7dfc7a4d95f8b6f979b5f10c1be8c07a96a635b461c
		    blockdag_test.go:987:
		    blockdag_test.go:990: Alternative branch:
		    blockdag_test.go:995: block #1: 0877dbb50dc6df9056f4caf55f698d5451a38015f8e536e9c82ca3f5265c38c7
		    blockdag_test.go:995: block #2: 0abb5a235c89d9ab6c3578917b116958174601d0f0e873ccf0356430a5e68731
		    blockdag_test.go:995: block #3: 0e8d5a512bb727f808e5e33e461046edad272884d4c03709813ccafbf790d30c
		    blockdag_test.go:995: block #4: 050161bc043a3dece33d5d16b37d7bcbde0872887dcf57b62275f018e64c445e
		    blockdag_test.go:995: block #5: 0ca13067fc288169b89115ae19c0ffa6d64dca8bd4adf72fb1c65fce21d434dc
		    blockdag_test.go:995: block #6: 1fceb631662fc8d251804e0ae3cef5a6626f8828fca9d7c685f918fed4d082a4
		    blockdag_test.go:995: block #7: 3850a3aee99233174618ceb3927e444742525ce1a1985b3fbcf875b8ce041315
		    blockdag_test.go:995: block #8: 27e832f88d76774e26a7b1c8b188c1008930bbfc15b0593232f9a703db97cda5
		    blockdag_test.go:995: block #9: 1aad930ee0224ea91af3d78421b3afa95b67df99cc77af8bc9ec6df17419efd3
		    blockdag_test.go:995: block #10: 49d4ddc7c3e2fc851ed723e69a974bab9ed0005bc2e961924fa433237127e076
		    blockdag_test.go:995: block #11: 000a7ca20fcae413dd9ce197ca3a719cae485566023eae3edae0f4a3615ba4e9
		    blockdag_test.go:995: block #12: 086e57e38868b4ac11c496c4bd76fa95f7c649cce7de01a0710e06ef8b612ed4
		    blockdag_test.go:995: block #13: 0a7bea10ae6133411fbee14ffa9a306f31ae96caaf2a7358000addc265033c93
		    blockdag_test.go:995: block #14: 0046b9e8d2c4f7a93bec4b28dd179512be54d25907956033a1067d99d3557cfb
		    blockdag_test.go:995: block #15: 084716aa877141a09bedd5d25bb3c299773a7eb021b1dbbe182aba641faf18f4
		    blockdag_test.go:995: block #16: 0001365b9aaa6dc24c392d18f5b223d5d7f4d9b3b370b26b9531feab4e157356
		    blockdag_test.go:995: block #17: 00c305ed264e014eeedadc5a74a22271402398f589a6e21ef52e206c1073c6dc
		    blockdag_test.go:1001: First branch tip acc work: 263
		    blockdag_test.go:1002: Second branch tip acc work: 61823
		    blockdag_test.go:1009: Current tip: 00c305ed264e014eeedadc5a74a22271402398f589a6e21ef52e206c1073c6dc
		    blockdag_test.go:1031:
		        	Error Trace:	/Users/liamz/tinychain-go/core/nakamoto/blockdag_test.go:1031
		        	Error:      	Not equal:
		        	            	expected: 17
		        	            	actual  : 4
		        	Test:       	TestDagGetPathComplexChain
		    blockdag_test.go:1035: Path block #1: 0877dbb50dc6df9056f4caf55f698d5451a38015f8e536e9c82ca3f5265c38c7
		    blockdag_test.go:1035: Path block #2: 0abb5a235c89d9ab6c3578917b116958174601d0f0e873ccf0356430a5e68731
		    blockdag_test.go:1035: Path block #3: 01790765a25dad409399744cc320c05080f21b2b503987871ebc4627285e86e9
		    blockdag_test.go:1036:
		        	Error Trace:	/Users/liamz/tinychain-go/core/nakamoto/blockdag_test.go:1036
		        	Error:      	Not equal:
		        	            	expected: "0e8d5a512bb727f808e5e33e461046edad272884d4c03709813ccafbf790d30c"
		        	            	actual  : "01790765a25dad409399744cc320c05080f21b2b503987871ebc4627285e86e9"

		        	            	Diff:
		        	            	--- Expected
		        	            	+++ Actual
		        	            	@@ -1 +1 @@
		        	            	-0e8d5a512bb727f808e5e33e461046edad272884d4c03709813ccafbf790d30c
		        	            	+01790765a25dad409399744cc320c05080f21b2b503987871ebc4627285e86e9
		        	Test:       	TestDagGetPathComplexChain
		    blockdag_test.go:1035: Path block #4: 06e1fe2a52f70b999334f7dfc7a4d95f8b6f979b5f10c1be8c07a96a635b461c
		    blockdag_test.go:1036:
		        	Error Trace:	/Users/liamz/tinychain-go/core/nakamoto/blockdag_test.go:1036
		        	Error:      	Not equal:
		        	            	expected: "050161bc043a3dece33d5d16b37d7bcbde0872887dcf57b62275f018e64c445e"
		        	            	actual  : "06e1fe2a52f70b999334f7dfc7a4d95f8b6f979b5f10c1be8c07a96a635b461c"
	*/

	/*
		Old code, useful for pure SQL testing:



		// Print the full chain hash list of node 3.
		// longestChainHashList, err := node3.Dag.GetLongestChainHashList(tip3.Hash, tip3.Height)
		// // for i, hash := range longestChainHashList {
		// // 	t.Logf("#%d: %x", i+1, hash)
		// // }
		// // t.Logf("Tip 3 hash: %s", tip3.HashStr())
		// // assert.Equal(17, len(longestChainHashList))
		// // assert.Equal(tip3.Hash, longestChainHashList[len(longestChainHashList)-1])

		// path1, err := node3.Dag.GetPath(tip1.Hash, uint64(3), 1)

		// t.Logf("")
		// t.Logf("inserting fake block history on alternative branch")

		// // Insert a few non descript path entries on the altnerative branch.
		// tx, err := node3.Dag.db.Begin()
		// if err != nil {
		// 	t.Fatalf("Failed to begin transaction: %s", err)
		// }
		// blockhash := tip1.Hash
		// tmpblocks := make([][32]byte, 0)
		// // "mine" block 1
		// tmpaccwork := tip1.AccumulatedWork
		// getAccWork := func(i int64) []byte {
		// 	buf := BigIntToBytes32(*tmpaccwork.Add(&tmpaccwork, big.NewInt(i*1000000)))
		// 	return buf[:]
		// }
		// blockhash[0] += 1
		// tmpblocks = append(tmpblocks, tip1.Hash)
		// _, err = tx.Exec(
		// 	"INSERT INTO blocks (parent_hash, hash, height, acc_work) VALUES (?, ?, ?, ?)",
		// 	tmpblocks[len(tmpblocks)-1][:],
		// 	blockhash[:],
		// 	tip1.Height+1,
		// 	getAccWork(1),
		// )
		// if err != nil {
		// 	t.Fatalf("Failed to insert block: %s", err)
		// }
		// tmpblocks = append(tmpblocks, blockhash)
		// // "mine" block 2
		// blockhash[0] += 1
		// _, err = tx.Exec(
		// 	"INSERT INTO blocks (parent_hash, hash, height, acc_work) VALUES (?, ?, ?, ?)",
		// 	(tmpblocks[len(tmpblocks)-1])[:],
		// 	blockhash[:],
		// 	tip1.Height+2,
		// 	getAccWork(2),
		// )
		// if err != nil {
		// 	t.Fatalf("Failed to insert block: %s", err)
		// }
		// tmpblocks = append(tmpblocks, blockhash)
		// err = tx.Commit()
		// if err != nil {
		// 	t.Fatalf("Failed to commit transaction: %s", err)
		// }
		// t.Logf("inserted blocks:")
		// t.Logf("- %x [fork point]", tmpblocks[0])
		// t.Logf("- %x", tmpblocks[1])
		// t.Logf("- %x", tmpblocks[2])
		// t.Logf("")

		// longestChainHashList2, err := node3.Dag.GetLongestChainHashList(node3.Dag.FullTip.Hash, node3.Dag.FullTip.Height)
		// for i, hash := range longestChainHashList {
		// 	t.Logf("#%d: %x", i+1, hash)
		// }
		// t.Log("")
		// for i, hash := range longestChainHashList2 {
		// 	t.Logf("#%d: %x", i+1, hash)
		// }
		// path2, err := node3.Dag.GetPath(tip1.Hash, uint64(2), 1)

		// // Path1
		// t.Logf("heights(15 - 17) path: %x", path1)

		// // Path2
		// t.Logf("heights(15 - 17) path: %x", path2)
		// assert.Nil(err)

		// path3, err := node3.Dag.GetPath(tmpblocks[2], uint64(2), -1)
		// assert.Nil(err)
		// t.Logf("heights(17_fork - 15) path: %x", path3)

	*/
}

func TestDagIngestHeader(t *testing.T) {
	// Ingest header.
	// Updates both full and header tip.
}

func TestDagIngestBodyMissingHeader(t *testing.T) {}

func TestDagIngestBody(t *testing.T) {
	// Ingest body.
	// Updates both full and header tip.
}
