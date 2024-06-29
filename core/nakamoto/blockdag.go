package nakamoto

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/liamzebedee/tinychain-go/core"
	_ "github.com/mattn/go-sqlite3"
)

var logger = NewLogger("blockdag")

func OpenDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Check to perform migrations.
	_, err = db.Exec("create table if not exists tinychain_version (version int)")
	if err != nil {
		return nil, fmt.Errorf("error checking database version: %s", err)
	}
	// Check the database version.
	rows, err := db.Query("select version from tinychain_version limit 1")
	if err != nil {
		return nil, fmt.Errorf("error checking database version: %s", err)
	}
	defer rows.Close()
	databaseVersion := 0
	if rows.Next() {
		rows.Scan(&databaseVersion)
	}

	// Log version.
	fmt.Printf("Database version: %d\n", databaseVersion)

	// Migration: v0.
	if databaseVersion == 0 {
		// Perform migrations.
		fmt.Printf("Running migration: %d\n", databaseVersion)

		// Create tables.
		_, err = db.Exec("create table epochs (id TEXT PRIMARY KEY, start_block_hash blob, start_time integer, start_height integer, difficulty blob)")
		if err != nil {
			return nil, fmt.Errorf("error creating 'epochs' table: %s", err)
		}

		_, err = db.Exec("create table blocks (hash blob primary key, parent_hash blob, difficulty blob, timestamp integer, num_transactions integer, transactions_merkle_root blob, nonce blob, graffiti blob, height integer, epoch TEXT, size_bytes integer, parent_total_work blob, acc_work blob, foreign key (epoch) REFERENCES epochs (id))")
		if err != nil {
			return nil, fmt.Errorf("error creating 'blocks' table: %s", err)
		}

		_, err = db.Exec("create table raw_blocks (hash blob primary key, data blob)")
		if err != nil {
			return nil, fmt.Errorf("error creating 'raw_blocks' table: %s", err)
		}

		_, err = db.Exec("create table transactions (hash blob, block_hash blob, sig blob, from_pubkey blob, data blob, txindex int)")
		if err != nil {
			return nil, fmt.Errorf("error creating 'transactions' table: %s", err)
		}

		// Create indexes.
		_, err = db.Exec("create index blocks_parent_hash on blocks (parent_hash)")
		if err != nil {
			return nil, fmt.Errorf("error creating 'blocks_parent_hash' index: %s", err)
		}

		// Update version.
		dbVersion := 1
		_, err = db.Exec("insert into tinychain_version (version) values (?)", dbVersion)
		if err != nil {
			return nil, fmt.Errorf("error updating database version: %s", err)
		}

		fmt.Printf("Database upgraded to: %d\n", dbVersion)
	}

	return db, err
}

func NewBlockDAGFromDB(db *sql.DB, stateMachine StateMachine, consensus ConsensusConfig) (BlockDAG, error) {
	dag := BlockDAG{
		db:           db,
		stateMachine: stateMachine,
		consensus:    consensus,
	}

	err := dag.initialiseBlockDAG()
	if err != nil {
		panic(err)
	}

	dag.Tip, err = dag.GetCurrentTip()
	if err != nil {
		panic(err)
	}

	return dag, nil
}

func (dag *BlockDAG) initialiseBlockDAG() error {
	genesisBlockHash := dag.consensus.GenesisBlockHash
	genesisBlock := RawBlock{
		ParentHash:             [32]byte{},
		ParentTotalWork:        BigIntToBytes32(*big.NewInt(0)),
		Difficulty:             BigIntToBytes32(dag.consensus.GenesisDifficulty),
		Timestamp:              0,
		NumTransactions:        0,
		TransactionsMerkleRoot: [32]byte{},
		Nonce:                  [32]byte{},
		Graffiti:               [32]byte{0xca, 0xfe, 0xba, 0xbe, 0xde, 0xca, 0xfb, 0xad}, // 0x cafebabe decafbad
		Transactions:           []RawTransaction{},
	}
	genesisHeight := uint64(0)

	// Check if we have already initialised the database.
	rows, err := dag.db.Query("select count(*) from blocks where hash = ?", genesisBlockHash[:])
	if err != nil {
		return err
	}
	count := 0
	if rows.Next() {
		rows.Scan(&count)
	}
	if count > 0 {
		return nil
	}
	rows.Close()

	// Begin initialisation.
	logger.Printf("Initialising block DAG...\n")

	// Insert the genesis epoch.
	epoch0 := Epoch{
		Number:         0,
		StartBlockHash: genesisBlockHash,
		StartTime:      genesisBlock.Timestamp,
		StartHeight:    genesisHeight,
		Difficulty:     dag.consensus.GenesisDifficulty,
	}
	_, err = dag.db.Exec(
		"insert into epochs (id, start_block_hash, start_time, start_height, difficulty) values (?, ?, ?, ?, ?)",
		epoch0.GetId(),
		epoch0.StartBlockHash[:],
		epoch0.StartTime,
		epoch0.StartHeight,
		// BigIntToBytes32(epoch0.Difficulty),
		epoch0.Difficulty.Bytes(),
	)
	if err != nil {
		return err
	}

	logger.Printf("Inserted genesis epoch difficulty=%s\n", dag.consensus.GenesisDifficulty.String())
	accWorkBuf := BigIntToBytes32(*big.NewInt(1)) // parent_total_work + 1

	// Insert the genesis block.
	_, err = dag.db.Exec(
		"insert into blocks (hash, parent_hash, parent_total_work, difficulty, timestamp, num_transactions, transactions_merkle_root, nonce, graffiti, height, epoch, size_bytes, acc_work) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		genesisBlockHash[:],
		genesisBlock.ParentHash[:],
		genesisBlock.ParentTotalWork[:],
		dag.consensus.GenesisDifficulty.Bytes(),
		genesisBlock.Timestamp,
		genesisBlock.NumTransactions,
		genesisBlock.TransactionsMerkleRoot[:],
		genesisBlock.Nonce[:],
		genesisBlock.Graffiti[:],
		genesisHeight,
		epoch0.GetId(),
		genesisBlock.SizeBytes(),
		PadBytes(accWorkBuf[:], 32),
	)
	if err != nil {
		return err
	}

	logger.Printf("Inserted genesis block hash=%s\n", hex.EncodeToString(genesisBlockHash[:]))

	return nil
}

func (dag *BlockDAG) IngestBlock(raw RawBlock) error {
	// 1. Verify parent is known.
	parentBlock, err := dag.GetBlockByHash(raw.ParentHash)
	if err != nil {
		return err
	}
	if parentBlock == nil {
		return fmt.Errorf("Unknown parent block.")
	}

	// 2. Verify timestamp is within bounds.
	// TODO: subjectivity.

	// 3. Verify num transactions is the same as the length of the transactions list.
	if int(raw.NumTransactions) != len(raw.Transactions) {
		return fmt.Errorf("Num transactions does not match length of transactions list.")
	}

	// 4. Verify transactions are valid.
	for i, block_tx := range raw.Transactions {
		logger.Printf("Verifying transaction %d\n", i)
		isValid := core.VerifySignature(
			hex.EncodeToString(block_tx.FromPubkey[:]),
			block_tx.Sig[:],
			block_tx.Envelope(),
		)
		if !isValid {
			return fmt.Errorf("Transaction %d is invalid: signature invalid.", i)
		}

		// This depends on where exactly we are verifying the sig.
		err := dag.stateMachine.VerifyTx(block_tx)

		if err != nil {
			return fmt.Errorf("Transaction %d is invalid.", i)
		}
	}

	// 5. Verify transaction merkle root is valid.
	txlist := make([][]byte, len(raw.Transactions))
	for i, block_tx := range raw.Transactions {
		txlist[i] = block_tx.Envelope()
	}
	expectedMerkleRoot := core.ComputeMerkleHash(txlist)
	if expectedMerkleRoot != raw.TransactionsMerkleRoot {
		return fmt.Errorf("Merkle root does not match computed merkle root.")
	}

	// 6. Verify POW solution is valid.
	height := uint64(parentBlock.Height + 1)
	var epoch *Epoch

	// 6a. Compute the current difficulty epoch.
	//

	// Are we on an epoch boundary?
	if height%dag.consensus.EpochLengthBlocks == 0 {
		// Recompute difficulty and create new epoch.
		logger.Printf("Recomputing difficulty for epoch %d\n", height/dag.consensus.EpochLengthBlocks)

		// Get current epoch.
		epoch, err = dag.GetEpochForBlockHash(raw.ParentHash)
		if err != nil {
			return err
		}
		newDifficulty := RecomputeDifficulty(epoch.StartTime, raw.Timestamp, epoch.Difficulty, dag.consensus.TargetEpochLengthMillis, dag.consensus.EpochLengthBlocks, height)

		epoch = &Epoch{
			Number:         height / dag.consensus.EpochLengthBlocks,
			StartBlockHash: raw.Hash(),
			StartTime:      raw.Timestamp,
			StartHeight:    height,
			Difficulty:     newDifficulty,
		}
		_, err := dag.db.Exec(
			"insert into epochs (id, start_block_hash, start_time, start_height, difficulty) values (?, ?, ?, ?, ?)",
			epoch.GetId(),
			epoch.StartBlockHash[:],
			epoch.StartTime,
			epoch.StartHeight,
			newDifficulty.Bytes(),
		)
		if err != nil {
			return err
		}
	} else {
		// Lookup current epoch.
		epoch, err = dag.GetEpochForBlockHash(raw.ParentHash)
		if epoch == nil {
			return fmt.Errorf("Parent block epoch not found.")
		}
		if err != nil {
			return err
		}
	}

	// 6b. Verify POW solution.
	blockHash := raw.Hash()
	if !VerifyPOW(blockHash, epoch.Difficulty) {
		return fmt.Errorf("POW solution is invalid.")
	}

	// 6c. Verify parent total work is correct.
	parentTotalWork := Bytes32ToBigInt(raw.ParentTotalWork)
	if parentBlock.AccumulatedWork.Cmp(&parentTotalWork) != 0 {
		return fmt.Errorf("Parent total work is incorrect.")
	}

	// 7. Verify block size is within bounds.
	if dag.consensus.MaxBlockSizeBytes < raw.SizeBytes() {
		return fmt.Errorf("Block size exceeds maximum block size.")
	}

	// 8. Ingest block into database store.
	tx, err := dag.db.Begin()
	if err != nil {
		return err
	}

	acc_work := new(big.Int)
	work := CalculateWork(Bytes32ToBigInt(blockHash))
	acc_work.Add(&parentBlock.AccumulatedWork, work)
	acc_work_buf := BigIntToBytes32(*acc_work)

	// Insert block.
	blockhash := raw.Hash()
	_, err = tx.Exec(
		"insert into blocks (hash, parent_hash, parent_total_work, timestamp, num_transactions, transactions_merkle_root, nonce, graffiti, height, epoch, size_bytes, acc_work) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		blockhash[:],
		raw.ParentHash[:],
		raw.ParentTotalWork[:],
		raw.Timestamp,
		raw.NumTransactions,
		raw.TransactionsMerkleRoot[:],
		raw.Nonce[:],
		raw.Graffiti[:],
		height,
		epoch.GetId(),
		raw.SizeBytes(),
		acc_work_buf[:],
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Insert raw block.
	_, err = tx.Exec(
		"insert into raw_blocks (hash, data) values (?, ?)",
		blockhash[:],
		raw.Bytes(),
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Insert transactions.
	for i, block_tx := range raw.Transactions {
		txhash := block_tx.Hash()
		_, err = tx.Exec(
			"insert into transactions (hash, block_hash, sig, from_pubkey, data, txindex) values (?, ?, ?, ?, ?, ?)",
			txhash[:],
			blockhash[:],
			block_tx.Sig[:],
			block_tx.FromPubkey[:],
			block_tx.Data[:],
			i,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()

	// Update the tip.
	// TODO this is bad for performance.
	// TODO also this is not atomic.
	curr_tip, err := dag.GetCurrentTip()
	if err != nil {
		return err
	}
	if curr_tip.Hash != dag.Tip.Hash {
		logger.Printf("New tip: height=%d hash=%s\n", curr_tip.Height, curr_tip.HashStr())
		dag.Tip = curr_tip
	}

	return nil
}

// Gets the epoch for a given block hash.
func (dag *BlockDAG) GetEpochForBlockHash(blockhash [32]byte) (*Epoch, error) {
	// Lookup the parent block.
	parentBlockEpochId := ""
	rows, err := dag.db.Query("select epoch from blocks where hash = ? limit 1", blockhash[:])
	if err != nil {
		return nil, err
	}
	if rows.Next() {
		rows.Scan(&parentBlockEpochId)
	} else {
		return nil, fmt.Errorf("Parent block not found.")
	}
	rows.Close()

	// Get the epoch.
	epoch := Epoch{}
	rows, err = dag.db.Query("select id, start_block_hash, start_time, start_height, difficulty from epochs where id = ? limit 1", parentBlockEpochId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		startBlockHash := []byte{}
		difficulty := []byte{}
		err := rows.Scan(&epoch.Id, &startBlockHash, &epoch.StartTime, &epoch.StartHeight, &difficulty)
		if err != nil {
			return nil, err
		}

		copy(epoch.StartBlockHash[:], startBlockHash)
		diffBytes32 := [32]byte{}
		copy(diffBytes32[:], difficulty)
		epoch.Difficulty = Bytes32ToBigInt(diffBytes32)
	} else {
		return nil, fmt.Errorf("Epoch not found.")
	}

	return &epoch, nil
}

func (dag *BlockDAG) GetBlockByHash(hash [32]byte) (*Block, error) {
	block := Block{}

	// Query database.
	rows, err := dag.db.Query("select hash, parent_hash, timestamp, num_transactions, transactions_merkle_root, nonce, graffiti, height, epoch, size_bytes, acc_work from blocks where hash = ? limit 1", hash[:])
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		hash := []byte{}
		parentHash := []byte{}
		transactionsMerkleRoot := []byte{}
		nonce := []byte{}
		graffiti := []byte{}
		accWorkBuf := []byte{}
		accWork := [32]byte{}

		err := rows.Scan(
			&hash,
			&parentHash,
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
			return nil, err
		}

		copy(block.Hash[:], hash)
		copy(block.ParentHash[:], parentHash)
		copy(block.TransactionsMerkleRoot[:], transactionsMerkleRoot)
		copy(block.Nonce[:], nonce)
		copy(block.Graffiti[:], graffiti)
		copy(accWork[:], accWorkBuf)
		block.AccumulatedWork = Bytes32ToBigInt(accWork)

		return &block, nil
	} else {
		return nil, err
	}
}

func (dag *BlockDAG) GetBlockTransactions(hash [32]byte) (*[]Transaction, error) {
	// Query database, get transactions count for blockhash.
	rows, err := dag.db.Query("select count(*) from transactions where block_hash = ?", hash[:])
	if err != nil {
		return nil, err
	}

	count := 0
	if rows.Next() {
		rows.Scan(&count)
	}
	rows.Close()

	// Construct the buffer.
	txs := make([]Transaction, count)

	// Load the transactions in.
	rows, err = dag.db.Query("select hash, sig, from_pubkey, data, txindex from transactions where block_hash = ?", hash[:])
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		tx := Transaction{}
		hash := []byte{}
		sig := []byte{}
		fromPubkey := []byte{}
		data := []byte{}
		index := 0

		err := rows.Scan(&hash, &sig, &fromPubkey, &data, &index)
		if err != nil {
			return nil, err
		}

		copy(tx.Hash[:], hash)
		copy(tx.Sig[:], sig)
		copy(tx.FromPubkey[:], fromPubkey)
		copy(tx.Data[:], data)

		txs[index] = tx
	}

	return &txs, nil
}

func (dag *BlockDAG) GetRawBlockDataByHash(hash [32]byte) ([]byte, error) {
	// Load from database.
	rows, err := dag.db.Query("select data from raw_blocks where hash = ?", hash[:])
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		data := []byte{}
		err := rows.Scan(&data)
		if err != nil {
			return nil, err
		}

		return data, nil
	} else {
		return nil, fmt.Errorf("Block not found.")
	}
}

func (dag *BlockDAG) HasBlock(hash [32]byte) bool {
	rows, err := dag.db.Query("select count(*) from blocks where hash = ?", hash[:])
	if err != nil {
		return false
	}
	count := 0
	if rows.Next() {
		rows.Scan(&count)
	}
	rows.Close()

	return count > 0
}

func (dag *BlockDAG) GetCurrentTip() (Block, error) {
	// The tip of the chain is defined as the chain with the longest proof-of-work.
	// Simply put, given a DAG of blocks, where each block has an accumulated work, we want to find the path with the highest accumulated work.

	// Query the highest accumulated work block in the database.
	rows, err := dag.db.Query("select hash from blocks order by acc_work desc limit 1")
	if err != nil {
		return Block{}, err
	}
	if !rows.Next() {
		return Block{}, fmt.Errorf("No blocks found.")
	}

	hash := []byte{}
	err = rows.Scan(&hash)
	if err != nil {
		return Block{}, err
	}
	rows.Close()

	fhash := [32]byte{}
	copy(fhash[:], hash)

	// Get the block.
	block, err := dag.GetBlockByHash(fhash)
	if err != nil {
		return Block{}, err
	}

	return *block, nil
}
