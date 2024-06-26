package nakamoto

import (
	"database/sql"
	"fmt"
	"math/big"
	_ "github.com/mattn/go-sqlite3"
	"github.com/liamzebedee/tinychain-go/core"
)


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
	databaseVersion := 0
	if rows.Next() {
		rows.Scan(&databaseVersion)
	}

	// Log version.
	fmt.Printf("Database version: %d\n", databaseVersion)
	if databaseVersion == 0 {
		// Perform migrations.
		
		// Create tables.
		_, err = db.Exec("create table blocks (hash blob primary key, parent_hash blob, timestamp integer, num_transactions integer, transactions_merkle_root blob, nonce blob)")
		if err != nil {
			return nil, fmt.Errorf("error creating 'blocks' table: %s", err)
		}
		_, err = db.Exec("create table transactions (hash blob primary key, block_hash blob, sig blob, from_pubkey blob, data blob)")
		if err != nil {
			return nil, fmt.Errorf("error creating 'transactions' table: %s", err)
		}

		// Create indexes.
		_, err = db.Exec("create index blocks_parent_hash on blocks (parent_hash)")
		if err != nil {
			return nil, fmt.Errorf("error creating 'blocks_parent_hash' index: %s", err)
		}
	}

	return db, err
}

type Block struct {
	// Block header.
	ParentHash [32]byte
	Timestamp uint64
	NumTransactions uint64
	TransactionsMerkleRoot [32]byte
	Nonce [32]byte
	
	// Block body.
	Transactions []RawTransaction

	// Metadata.
	Height uint64
	Epoch uint64
	Work big.Int
	SizeBytes uint64
}

type Epoch struct {
	// Epoch number.
	Number uint64

	// Start block.
	StartBlockHash [32]byte
	// Start time.
	StartTime uint64

	// End block.
	EndBlockHash [32]byte
	// End time.
	EndTime uint64

	// Difficulty target.
	Difficulty big.Int
}

type BlockDAG struct {
	// The backing SQL database store.
	db *sql.DB
	stateMachine StateMachine

	// The genesis block hash.
	genesisBlockHash [32]byte
}

type StateMachine interface {
	VerifyTx(tx RawTransaction) error
}

func NewBlockDAGFromDB(db *sql.DB, stateMachine StateMachine, genesisBlockHash [32]byte) (BlockDAG, error) {
	dag := BlockDAG{
		db: db,
		stateMachine: stateMachine,
		genesisBlockHash: genesisBlockHash,
	}
	return dag, nil
}

func (dag *BlockDAG) GetGenesisBlockHash() ([32]byte) {
	return dag.genesisBlockHash
}


func (dag *BlockDAG) IngestBlock(raw RawBlock) error {
	// 1. Verify parent is known.
	if raw.ParentHash != dag.genesisBlockHash {
		parent_block := dag.GetBlockByHash(raw.ParentHash)
		if parent_block == nil {
			return fmt.Errorf("Unknown parent block.")
		}
	}

	// 2. Verify timestamp is within bounds.
	// TODO: subjectivity.

	// 3. Verify num transactions is the same as the length of the transactions list.
	if int(raw.NumTransactions) != len(raw.Transactions) {
		return fmt.Errorf("Num transactions does not match length of transactions list.")
	}

	// 4. Verify transactions are valid.
	for i, block_tx := range raw.Transactions {
		// TODO: Verify signature.
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
	epoch, err := dag.GetEpochForBlockHash(raw.ParentHash)
	if epoch == nil {
		return fmt.Errorf("Parent block epoch not found.")
	}
	if err != nil {
		return fmt.Errorf("Failed to get parent block epoch: %s.", err)
	}
	if !VerifyPOW(raw.Hash(), epoch.Difficulty) {
		return fmt.Errorf("POW solution is invalid.")
	}

	// 7. Verify block size is correctly computed.


	// Annotations:
	// 1. Add block height.
	// 2. Compute block epoch.
	// block := Block{
	// 	ParentHash: raw.ParentHash,
	// 	Timestamp: raw.Timestamp,
	// 	NumTransactions: raw.NumTransactions,
	// 	Transactions: raw.Transactions,
	// 	Height: parent_block.Height + 1,
	// 	Epoch: parent_block.Epoch,
	// 	SizeBytes: raw.SizeBytes(),
	// }
	
	// Ingest into database store.
	tx, err := dag.db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("insert into blocks (hash, parent_hash, timestamp, num_transactions, transactions_merkle_root, nonce) values (?, ?, ?, ?, ?, ?)", raw.Hash(), raw.ParentHash, raw.Timestamp, raw.NumTransactions, raw.TransactionsMerkleRoot, raw.Nonce)
	if err != nil {
		tx.Rollback()
		return err
	}
	for _, block_tx := range raw.Transactions {
		_, err = tx.Exec("insert into transactions (hash, block_hash, sig, from_pubkey, data) values (?, ?, ?, ?, ?)", block_tx.Hash(), raw.Hash(), block_tx.Sig, block_tx.FromPubkey, block_tx.Data)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()

	return nil
}

func (dag *BlockDAG) GetEpochForBlockHash(parentBlockHash [32]byte) (*Epoch, error) {
	// Special case: genesis block.
	if parentBlockHash == dag.genesisBlockHash {
		target := new(big.Int)
		target.SetString("0fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)

		return &Epoch{
			Number: 0,
			StartBlockHash: dag.genesisBlockHash,
			StartTime: 0,
			EndBlockHash: [32]byte{},
			EndTime: 0,
			Difficulty: *target,
		}, nil
	}

	return nil, nil
}

func (dag *BlockDAG) GetBlockByHash(hash [32]byte) (*Block) {
	block := Block{}

	// Query database.
	rows, err := dag.db.Query("select parent_hash, timestamp, num_transactions, transactions_merkle_root, nonce from blocks where hash = ? limit 1", hash)
	if err != nil {
		return nil
	}

	if rows.Next() {
		rows.Scan(&block.ParentHash, &block.Timestamp, &block.NumTransactions, &block.TransactionsMerkleRoot, &block.Nonce)
		return &block
	} else {
		return nil
	}
}

type BlockDAGInterface interface {
	// Ingest block.
	IngestBlock(b Block) error

	// Get block.
	GetBlockByHash(hash [32]byte) (Block)

	// Get epoch for block.
	GetEpochForBlockHash(parentBlockHash [32]byte) (uint64, error)
	
	// Get a list of blocks at height.
	GetBlocksByHeight(height uint64) ([]Block, error)

	// Get the tip of the chain, given a minimum number of confirmations.
	GetTips(minConfirmations uint64) ([]Block, error)
}