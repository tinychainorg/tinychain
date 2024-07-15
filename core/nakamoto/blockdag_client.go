package nakamoto

import (
	"fmt"
)

// GetEpochForBlockhash
// GetBlockByHash
// GetBlockTransactions
// GetRawBlockDataByHash
// HasBlock
// GetLatestTip
// GetLongestChainHashList

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
	rows, err := dag.db.Query("select hash, parent_hash, difficulty, parent_total_work, timestamp, num_transactions, transactions_merkle_root, nonce, graffiti, height, epoch, size_bytes, acc_work from blocks where hash = ? limit 1", hash[:])
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		hash := []byte{}
		parentHash := []byte{}
		difficultyBuf := []byte{}
		transactionsMerkleRoot := []byte{}
		nonce := []byte{}
		graffiti := []byte{}
		accWorkBuf := []byte{}
		parentTotalWorkBuf := []byte{}

		err := rows.Scan(
			&hash,
			&parentHash,
			&difficultyBuf,
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
			return nil, err
		}

		copy(block.Hash[:], hash)
		copy(block.ParentHash[:], parentHash)
		copy(block.Difficulty[:], difficultyBuf)
		copy(block.TransactionsMerkleRoot[:], transactionsMerkleRoot)
		copy(block.Nonce[:], nonce)
		copy(block.Graffiti[:], graffiti)

		accWork := [32]byte{}
		copy(accWork[:], accWorkBuf)
		block.AccumulatedWork = Bytes32ToBigInt(accWork)

		parentTotalWork := [32]byte{}
		copy(parentTotalWork[:], parentTotalWorkBuf)
		block.ParentTotalWork = Bytes32ToBigInt(parentTotalWork)

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
	rows, err = dag.db.Query(`
		SELECT t.hash, t.sig, t.from_pubkey, t.to_pubkey, t.amount, t.fee, t.nonce, tb.txindex, t.version
		FROM transactions t
		JOIN transactions_blocks tb ON t.hash = tb.transaction_hash
		WHERE tb.block_hash = ?
		ORDER BY tb.txindex ASC;
	`, hash[:])
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		tx := Transaction{}
		hash := []byte{}
		sig := []byte{}
		fromPubkey := []byte{}
		toPubkey := []byte{}
		amount := uint64(0)
		fee := uint64(0)
		nonce := uint64(0)
		var index uint64 = 0
		version := 0

		err := rows.Scan(&hash, &sig, &fromPubkey, &toPubkey, &amount, &fee, &nonce, &index, &version)
		if err != nil {
			return nil, err
		}

		copy(tx.Hash[:], hash)
		copy(tx.Sig[:], sig)
		copy(tx.FromPubkey[:], fromPubkey)
		copy(tx.ToPubkey[:], toPubkey)
		tx.Amount = amount
		tx.Fee = fee
		tx.Nonce = nonce
		tx.TxIndex = index
		tx.Version = byte(version)

		txs[index] = tx
	}

	return &txs, nil
}

func (dag *BlockDAG) GetRawBlockDataByHash(hash [32]byte) ([]byte, error) {
	// TODO.
	// get block from disk
	// get txs from disk
	// load into raw block
	return []byte{}, nil
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

// Gets the latest block in the longest chain.
func (dag *BlockDAG) GetLatestHeadersTip() (Block, error) {
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

	hashBuf := []byte{}
	err = rows.Scan(&hashBuf)
	if err != nil {
		return Block{}, err
	}
	rows.Close()

	hash := [32]byte{}
	copy(hash[:], hashBuf)

	// Get the block.
	block, err := dag.GetBlockByHash(hash)
	if err != nil {
		return Block{}, err
	}

	return *block, nil
}

// Gets the latest block in the longest chain.
func (dag *BlockDAG) GetLatestFullTip() (Block, error) {
	// Query the highest accumulated work block in the database.
	rows, err := dag.db.Query(`
		SELECT hash 
		FROM blocks 
		WHERE hash IN (
			SELECT block_hash 
			FROM transactions_blocks
		) 
		ORDER BY acc_work DESC 
		LIMIT 1;
	`)
	if err != nil {
		return Block{}, err
	}
	if !rows.Next() {
		return Block{}, fmt.Errorf("No blocks found.")
	}

	hashBuf := []byte{}
	err = rows.Scan(&hashBuf)
	if err != nil {
		return Block{}, err
	}
	rows.Close()

	hash := [32]byte{}
	copy(hash[:], hashBuf)

	// Get the block.
	block, err := dag.GetBlockByHash(hash)
	if err != nil {
		return Block{}, err
	}

	return *block, nil
}

// Gets the list of hashes for the longest chain, traversing backwards from startHash and accumulating depthFromTip items.
func (dag *BlockDAG) GetLongestChainHashList(startHash [32]byte, depthFromTip uint64) ([][32]byte, error) {
	list := make([][32]byte, 0, depthFromTip)

	// Hey, I bet you didn't know SQL could do this, right?
	// Neither did I. It's called a recursive common table expression.
	// It's a way to traverse a tree structure in SQL.
	// Pretty cool, huh?
	rows, err := dag.db.Query(`
		WITH RECURSIVE block_path AS (
			SELECT hash, parent_hash, 1 AS depth
			FROM blocks
			WHERE hash = ?

			UNION ALL

			SELECT b.hash, b.parent_hash, bp.depth + 1
			FROM blocks b
			INNER JOIN block_path bp ON b.hash = bp.parent_hash
			WHERE bp.depth < ?
		)
		SELECT hash, parent_hash
		FROM block_path
		ORDER BY depth DESC;`,
		startHash[:],
		depthFromTip,
	)
	if err != nil {
		return list, err
	}

	for rows.Next() {
		hashBuf := []byte{}
		parentHashBuf := []byte{}

		hash := [32]byte{}
		parentHash := [32]byte{}

		err := rows.Scan(&hashBuf, &parentHashBuf)
		if err != nil {
			return list, err
		}

		copy(hash[:], hashBuf)
		copy(parentHash[:], parentHashBuf)

		list = append(list, hash)
	}

	return list, nil
}

// Iterates forwards (direction = 1) or backwards (direction = -1) from startHash, accumulating `depthFromTip` items in the canonical longest chain linked list.
// The returned list is in traversal order.
func (dag *BlockDAG) GetPath(startHash [32]byte, depthFromTip uint64, direction int) ([][32]byte, error) {
	list := make([][32]byte, 0, depthFromTip)

	// When iterating backwards, we don't have to worry about accumulated work. Since we're going backwards, we can just follow the parent hash.
	queryDirectionBackwards := `
		WITH RECURSIVE block_path AS (
			SELECT hash, parent_hash, 1 AS depth
			FROM blocks
			WHERE hash = ?

			UNION ALL

			SELECT b.hash, b.parent_hash, bp.depth + 1
			FROM blocks b
			INNER JOIN block_path bp ON b.hash = bp.parent_hash
			WHERE bp.depth < ?
		)
		SELECT hash, parent_hash
		FROM block_path
		ORDER BY depth DESC;`
	
	// When iterating forward, find the block with highest accumulated work.
	queryDirectionForwards := `
		WITH RECURSIVE block_path AS (
			SELECT hash, parent_hash, acc_work, 1 AS depth
			FROM blocks
			WHERE hash = ?

			UNION ALL

			SELECT b.hash, b.parent_hash, b.acc_work, bp.depth + 1
			FROM blocks b
			INNER JOIN block_path bp 
			ON bp.hash = b.parent_hash
			WHERE bp.depth < ?
			ORDER BY b.acc_work DESC
			LIMIT 1
		)
		SELECT hash, parent_hash, acc_work
		FROM block_path
		ORDER BY depth ASC;`
	
	query := ""
	if direction == 1 {
		query = queryDirectionForwards
	} else {
		query = queryDirectionBackwards
	}

	rows, err := dag.db.Query(
		query,
		startHash[:],
		depthFromTip,
	)
	if err != nil {
		return list, err
	}

	for rows.Next() {
		hashBuf := []byte{}
		parentHashBuf := []byte{}

		hash := [32]byte{}
		parentHash := [32]byte{}

		err := rows.Scan(&hashBuf, &parentHashBuf)
		if err != nil {
			return list, err
		}

		copy(hash[:], hashBuf)
		copy(parentHash[:], parentHashBuf)

		list = append(list, hash)
	}

	return list, nil
}
