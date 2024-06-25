package nakamoto

import (
	"database/sql"
	"fmt"
	"math/big"
	_ "github.com/mattn/go-sqlite3"
	"github.com/liamzebedee/tinychain-go/core"
)


func OpenDB(dbPath string) (error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	
	// Check schemas.
	// rows, err := db.Query("select text from mytable where name regexp '^golang'")
	// if err != nil {
	// 	return err
	// }

	// for rows.Next() {
	// 	var text string
	// 	rows.Scan(&text)
	// 	fmt.Println(text)
	// }

	return nil
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
	db *sql.DB
	stateMachine StateMachine
}

type StateMachine interface {
	VerifyTx(tx RawTransaction) error
}

func NewFromDB(db *sql.DB, stateMachine StateMachine) (BlockDAG, error) {
	return BlockDAG{
		db: db,
		stateMachine: stateMachine,
	}, nil
}

func (dag *BlockDAG) IngestBlock(b Block) error {
	return nil
}

func (dag *BlockDAG) CheckRawBlock(raw RawBlock) error {
	block := Block{
		ParentHash: raw.ParentHash,
		Timestamp: raw.Timestamp,
		NumTransactions: raw.NumTransactions,
		TransactionsMerkleRoot: raw.TransactionsMerkleRoot,
		Nonce: raw.Nonce,

		Transactions: raw.Transactions,

		Height: 0,
		Epoch: 0,
		SizeBytes: 0,
	}

	// 1. Verify parent is known.
	parent_block := dag.GetBlockByHash(block.ParentHash)
	if parent_block == nil {
		return fmt.Errorf("Parent block not found.")
	}

	// 2. Verify timestamp is within bounds.
	// TODO: subjectivity.

	// 3. Verify num transactions is the same as the length of the transactions list.
	if int(block.NumTransactions) != len(block.Transactions) {
		return fmt.Errorf("Num transactions does not match length of transactions list.")
	}
	
	// 4. Verify transactions are valid.
	for i, tx := range block.Transactions {
		err := dag.stateMachine.VerifyTx(tx)
		if err != nil {
			return fmt.Errorf("Transaction %d is invalid.", i)
		}
	}

	// 5. Verify transaction merkle root is valid.
	txlist := make([][]byte, len(block.Transactions))
	for i, tx := range block.Transactions {
		txlist[i] = tx.Envelope()
	}
	expectedMerkleRoot := core.ComputeMerkleHash(txlist)
	if expectedMerkleRoot != block.TransactionsMerkleRoot {
		return fmt.Errorf("Merkle root does not match computed merkle root.")
	}

	// 6. Verify POW solution is valid.
	epoch, err := dag.GetEpochForBlockHash(block.ParentHash)
	if epoch == nil {
		return fmt.Errorf("Parent block epoch not found.")
	}
	if err != nil {
		return err
	}
	if !VerifyPOW(raw.Hash(), epoch.Difficulty) {
		return fmt.Errorf("POW solution is invalid.")
	}

	// 7. Verify block size is correctly computed.


	// Annotations:
	// 1. Add block height.
	// 2. Compute block epoch.
	return nil
}

func (dag *BlockDAG) GetEpochForBlockHash(hash [32]byte) (*Epoch, error) {
	return nil, nil
}

func (dag *BlockDAG) GetBlockByHash(hash [32]byte) (*Block) {
	return &Block{}
}

type BlockDAGInterface interface {
	// Ingest block.
	IngestBlock(b Block) error

	// Get block.
	GetBlockByHash(hash [32]byte) (Block)

	// Get epoch for block.
	GetEpochForBlockHash(hash [32]byte) (uint64, error)
	
	// Get a list of blocks at height.
	GetBlocksByHeight(height uint64) ([]Block, error)

	// Get the tip of the chain, given a minimum number of confirmations.
	GetTips(minConfirmations uint64) ([]Block, error)
}