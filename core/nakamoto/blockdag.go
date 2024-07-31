package nakamoto

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"

	"github.com/liamzebedee/tinychain-go/core"
	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrBlockNotFound = fmt.Errorf("Block not found.")
)

// The block DAG is the core data structure of the Nakamoto consensus protocol.
// It is a directed acyclic graph of blocks, where each block has a parent block.
// As it is infeasible to store the entirety of the blockchain in-memory,
// the block DAG is backed by a SQL database.
type BlockDAG struct {
	// The backing SQL database store, which stores:
	// - blocks
	// - epochs
	// - transactions
	db *sql.DB

	// The state machine.
	stateMachine StateMachineInterface

	// Consensus settings.
	consensus ConsensusConfig

	// Tips mutex.
	tipsMutex *sync.Mutex

	// The "light client" tip. This is the tip of the heaviest chain of block headers.
	HeadersTip Block

	// The "full node" tip. This is the tip of the heaviest chain of full blocks.
	FullTip Block

	// OnNewTip handler.
	OnNewHeadersTip func(tip Block, prevTip Block)
	OnNewFullTip    func(tip Block, prevTip Block)

	log *log.Logger
}

func NewBlockDAGFromDB(db *sql.DB, stateMachine StateMachineInterface, consensus ConsensusConfig) (BlockDAG, error) {
	dag := BlockDAG{
		db:           db,
		stateMachine: stateMachine,
		consensus:    consensus,
		log:          NewLogger("blockdag", ""),
		tipsMutex:    &sync.Mutex{},
	}

	err := dag.initialiseBlockDAG()
	if err != nil {
		panic(err)
	}

	err = dag.UpdateTip()
	if err != nil {
		panic(err)
	}

	return dag, nil
}

// Initalises the block DAG with the genesis block.
func (dag *BlockDAG) initialiseBlockDAG() error {
	genesisBlock := GetRawGenesisBlockFromConfig(dag.consensus)
	genesisBlockHash := genesisBlock.Hash()
	genesisHeight := uint64(0)

	// Check if we have already initialised the database.
	tx, err := dag.db.Begin()
	if err != nil {
		return err
	}

	rows, err := tx.Query("select count(*) from blocks where hash = ?", genesisBlockHash[:])
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

	// Begin initialisation.
	dag.log.Printf("Initialising block DAG...\n")

	// Insert the genesis epoch.
	epoch0 := Epoch{
		Number:         0,
		StartBlockHash: genesisBlockHash,
		StartTime:      genesisBlock.Timestamp,
		StartHeight:    genesisHeight,
		Difficulty:     dag.consensus.GenesisDifficulty,
	}
	_, err = tx.Exec(
		"insert into epochs (id, start_block_hash, start_time, start_height, difficulty) values (?, ?, ?, ?, ?)",
		epoch0.GetId(),
		epoch0.StartBlockHash[:],
		epoch0.StartTime,
		epoch0.StartHeight,
		epoch0.Difficulty.Bytes(),
	)
	if err != nil {
		return err
	}

	work := CalculateWork(Bytes32ToBigInt(genesisBlock.Hash()))
	dag.log.Printf("Inserted genesis epoch difficulty=%s\n", dag.consensus.GenesisDifficulty.String())
	accWorkBuf := BigIntToBytes32(*work)

	// Insert the genesis block.
	_, err = tx.Exec(
		"insert into blocks (hash, parent_hash, parent_total_work, difficulty, timestamp, num_transactions, transactions_merkle_root, nonce, graffiti, height, epoch, size_bytes, acc_work) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		genesisBlockHash[:],
		genesisBlock.ParentHash[:],
		genesisBlock.ParentTotalWork[:],
		genesisBlock.Difficulty[:],
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

	if err = tx.Commit(); err != nil {
		panic(err)
	}

	dag.log.Printf("Inserted genesis block hash=%s work=%s\n", hex.EncodeToString(genesisBlockHash[:]), work.String())

	return nil
}

func (dag *BlockDAG) updateHeadersTip() error {
	prev_tip := dag.HeadersTip
	curr_tip, err := dag.GetLatestHeadersTip()
	if err != nil {
		return err
	}

	if prev_tip.Hash != curr_tip.Hash {
		dag.log.Printf("New headers tip: height=%d hash=%s\n", curr_tip.Height, curr_tip.HashStr())
		dag.HeadersTip = curr_tip
		if dag.OnNewHeadersTip == nil {
			return nil
		}
		dag.OnNewHeadersTip(curr_tip, prev_tip)
	}

	return nil
}

func (dag *BlockDAG) updateFullTip() error {
	prev_tip := dag.FullTip
	curr_tip, err := dag.GetLatestFullTip()
	if err != nil {
		return err
	}

	if prev_tip.Hash != curr_tip.Hash {
		dag.log.Printf("New full tip: height=%d hash=%s\n", curr_tip.Height, curr_tip.HashStr())
		dag.FullTip = curr_tip
		if dag.OnNewFullTip == nil {
			return nil
		}
		dag.OnNewFullTip(curr_tip, prev_tip)
		if dag.OnNewFullTip != nil {
			dag.OnNewFullTip(curr_tip, prev_tip)
		}
	}

	return nil
}

func (dag *BlockDAG) UpdateTip() error {
	dag.tipsMutex.Lock()
	defer dag.tipsMutex.Unlock()

	err := dag.updateHeadersTip()
	if err != nil {
		return err
	}

	err = dag.updateFullTip()
	if err != nil {
		return err
	}

	return nil
}

// Ingests a block header, and recomputes the headers tip. Used by light clients / SPV sync.
func (dag *BlockDAG) IngestHeader(raw BlockHeader) error {
	// 1. Verify parent is known.
	parentBlock, err := dag.GetBlockByHash(raw.ParentHash)
	if err != nil {
		return err
	}
	if parentBlock == nil {
		return fmt.Errorf("Unknown parent block.")
	}

	// 6. Verify POW solution is valid.
	height := uint64(parentBlock.Height + 1)
	var epoch *Epoch

	// 6a. Compute the current difficulty epoch.
	//

	// Are we on an epoch boundary?
	if height%dag.consensus.EpochLengthBlocks == 0 {
		// Recompute difficulty and create new epoch.
		dag.log.Printf("Recomputing difficulty for epoch %d\n", height/dag.consensus.EpochLengthBlocks)

		// Get current epoch.
		epoch, err = dag.GetEpochForBlockHash(raw.ParentHash)
		if err != nil {
			return err
		}
		newDifficulty := RecomputeDifficulty(epoch.StartTime, raw.Timestamp, epoch.Difficulty, dag.consensus.TargetEpochLengthMillis, dag.consensus.EpochLengthBlocks, height)

		epoch = &Epoch{
			Number:         height / dag.consensus.EpochLengthBlocks,
			StartBlockHash: raw.BlockHash(),
			StartTime:      raw.Timestamp,
			StartHeight:    height,
			Difficulty:     newDifficulty,
		}
		diffBytes := BigIntToBytes32(newDifficulty)

		_, err := dag.db.Exec(
			"insert into epochs (id, start_block_hash, start_time, start_height, difficulty) values (?, ?, ?, ?, ?)",
			epoch.GetId(),
			epoch.StartBlockHash[:],
			epoch.StartTime,
			epoch.StartHeight,
			diffBytes[:],
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
	blockHash := raw.BlockHash()
	if !VerifyPOW(blockHash, epoch.Difficulty) {
		return fmt.Errorf("POW solution is invalid.")
	}

	// 6c. Verify parent total work is correct.
	parentTotalWork := Bytes32ToBigInt(raw.ParentTotalWork)
	if parentBlock.AccumulatedWork.Cmp(&parentTotalWork) != 0 {
		dag.log.Printf("Comparing parent total work. expected=%s actual=%s\n", parentBlock.AccumulatedWork.String(), parentTotalWork.String())
		return fmt.Errorf("Parent total work is incorrect.")
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
	_, err = tx.Exec(
		"insert into blocks (hash, parent_hash, parent_total_work, timestamp, num_transactions, transactions_merkle_root, nonce, graffiti, height, epoch, size_bytes, acc_work) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		blockHash[:],
		raw.ParentHash[:],
		raw.ParentTotalWork[:],
		raw.Timestamp,
		raw.NumTransactions,
		raw.TransactionsMerkleRoot[:],
		raw.Nonce[:],
		raw.Graffiti[:],
		height,
		epoch.GetId(),
		0, // Block size is 0 until we get transactions.
		acc_work_buf[:],
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	// Update the headers tip.
	err = dag.UpdateTip()
	if err != nil {
		return err
	}

	return nil
}

// Ingests a block's body, which is linked to a previously ingested block header.
func (dag *BlockDAG) IngestBlockBody(body []RawTransaction) error {
	// Get the merkle root for this body.
	txMerkleRoot := GetMerkleRootForTxs(body)

	// Lookup the block that has this merkle root.
	rows, err := dag.db.Query(`select hash from blocks where transactions_merkle_root = ?`, txMerkleRoot[:])
	if err != nil {
		return err
	}
	blockhashBuf := make([]byte, 32)
	if rows.Next() {
		rows.Scan(&blockhashBuf)
	} else {
		return fmt.Errorf("Block header not found for txs merkle root.")
	}
	rows.Close()

	// Verify we have not already ingested the txs.
	rows, err = dag.db.Query(`select count(*) from transactions_blocks where block_hash = ?`, blockhashBuf)
	if err != nil {
		return err
	}
	count := 0
	if rows.Next() {
		rows.Scan(&count)
	}
	rows.Close()
	// dag.log.Printf("Blocks with merkle root: %d\n", count)
	// TODO later.

	if count > 0 {
		return fmt.Errorf("Block already has transactions ingested.")
	}

	// Lookup block header.
	blockhash := [32]byte{}
	copy(blockhash[:], blockhashBuf)
	block, err := dag.GetBlockByHash(blockhash)
	if err != nil {
		return err
	}
	if block == nil {
		return fmt.Errorf("Block header missing during body ingestion.")
	}
	raw := block.ToRawBlock()
	raw.Transactions = body

	// 2. Verify timestamp is within bounds.
	// TODO: subjectivity.

	// 3. Verify num transactions is the same as the length of the transactions list.
	if int(raw.NumTransactions) != len(raw.Transactions) {
		return fmt.Errorf("Num transactions does not match length of transactions list.")
	}

	// 4. Verify transactions are valid.
	// TODO: We can parallelise this.
	// This is one of the most expensive operations of the blockchain node.
	for i, block_tx := range raw.Transactions {
		dag.log.Printf("Verifying transaction %d\n", i)
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
	expectedMerkleRoot := GetMerkleRootForTxs(raw.Transactions)
	if expectedMerkleRoot != raw.TransactionsMerkleRoot {
		return fmt.Errorf("Merkle root does not match computed merkle root.")
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

	// Update block size.

	// Insert transactions, transactions_blocks.
	for i, block_tx := range raw.Transactions {
		txhash := block_tx.Hash()

		_, err = tx.Exec(
			`insert into transactions_blocks (block_hash, transaction_hash, txindex) values (?, ?, ?)`,
			blockhash[:],
			txhash[:],
			i,
		)
		if err != nil {
			tx.Rollback()
			return err
		}

		// Check if we already have the transaction.
		rows, err := tx.Query("select count(*) from transactions where hash = ?", txhash[:])
		if err != nil {
			tx.Rollback()
			return err
		}
		count := 0
		if rows.Next() {
			rows.Scan(&count)
		}
		rows.Close()

		if count > 0 {
			continue
		}

		// Insert the transaction.
		_, err = tx.Exec(
			"insert into transactions (hash, sig, from_pubkey, to_pubkey, amount, fee, nonce, version) values (?, ?, ?, ?, ?, ?, ?, ?)",
			txhash[:],
			block_tx.Sig[:],
			block_tx.FromPubkey[:],
			block_tx.ToPubkey[:],
			block_tx.Amount,
			block_tx.Fee,
			block_tx.Nonce,
			block_tx.Version,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()

	// Update the tip.
	err = dag.UpdateTip()
	if err != nil {
		return err
	}

	return nil
}

// Ingests a full block, and recomputes the full tip.
func (dag *BlockDAG) IngestBlock(raw RawBlock) error {
	// 1. Verify parent is known.
	parentBlock, err := dag.GetBlockByHash(raw.ParentHash)

	// Check if block not found error.
	if errors.Is(err, ErrBlockNotFound) {
		return fmt.Errorf("Unknown parent block.")
	}
	if err != nil {
		return err
	}

	// 2. Verify timestamp is within bounds.
	// TODO: subjectivity.

	// 3. Verify num transactions is the same as the length of the transactions list.
	if int(raw.NumTransactions) != len(raw.Transactions) {
		return fmt.Errorf("Num transactions does not match length of transactions list.")
	}

	// 4. Verify transactions are valid.
	// TODO: We can parallelise this.
	// This is one of the most expensive operations of the blockchain node.
	for i, block_tx := range raw.Transactions {
		dag.log.Printf("Verifying transaction %d\n", i)
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
	expectedMerkleRoot := GetMerkleRootForTxs(raw.Transactions)
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
		dag.log.Printf("Recomputing difficulty for epoch %d\n", height/dag.consensus.EpochLengthBlocks)

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
		diffBytes := BigIntToBytes32(newDifficulty)
		
		_, err := dag.db.Exec(
			"insert into epochs (id, start_block_hash, start_time, start_height, difficulty) values (?, ?, ?, ?, ?)",
			epoch.GetId(),
			epoch.StartBlockHash[:],
			epoch.StartTime,
			epoch.StartHeight,
			diffBytes[:],
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
		dag.log.Printf("Comparing parent total work. expected=%s actual=%s\n", parentBlock.AccumulatedWork.String(), parentTotalWork.String())
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

	// Insert transactions, transactions_blocks.
	for i, block_tx := range raw.Transactions {
		txhash := block_tx.Hash()

		_, err = tx.Exec(
			`insert into transactions_blocks (block_hash, transaction_hash, txindex) values (?, ?, ?)`,
			blockhash[:],
			txhash[:],
			i,
		)
		if err != nil {
			tx.Rollback()
			return err
		}

		// Check if we already have the transaction.
		rows, err := tx.Query("select count(*) from transactions where hash = ?", txhash[:])
		if err != nil {
			tx.Rollback()
			return err
		}
		count := 0
		if rows.Next() {
			rows.Scan(&count)
		}
		rows.Close()

		if count > 0 {
			continue
		}

		// Insert the transaction.
		_, err = tx.Exec(
			"insert into transactions (hash, sig, from_pubkey, to_pubkey, amount, fee, nonce, version) values (?, ?, ?, ?, ?, ?, ?, ?)",
			txhash[:],
			block_tx.Sig[:],
			block_tx.FromPubkey[:],
			block_tx.ToPubkey[:],
			block_tx.Amount,
			block_tx.Fee,
			block_tx.Nonce,
			block_tx.Version,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()

	// Update the tip.
	err = dag.UpdateTip()
	if err != nil {
		return err
	}

	return nil
}
