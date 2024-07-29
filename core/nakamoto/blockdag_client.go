package nakamoto

import (
	"database/sql"
	"fmt"
)

// The methods of the BlockDAG engine:
//
// Light sync:
// - IngestHeader
// - IngestBlockBodies
//
// Full sync:
// - IngestBlock
//

// The methods of the BlockDAG client:
//
// Difficulty:
// - GetEpochForBlockhash
//
// Blocks:
// - GetBlockByHash
// - GetBlockTransactions
// - GetRawBlockDataByHash
//
// Tip:
// - GetLatestFullTip
// - GetLatestHeadersTip
// - GetPath
// - GetLongestChainHashList
//
// Sync:
// - HasBlock
//

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
	rows, err := dag.db.Query(
		`select hash, parent_hash, difficulty, parent_total_work, timestamp, num_transactions, transactions_merkle_root, nonce, graffiti, height, epoch, size_bytes, acc_work from blocks where hash = ? limit 1`,
		hash[:],
	)
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
		return nil, ErrBlockNotFound
	}
}

func (dag *BlockDAG) GetBlockTransactions(hash [32]byte) (*[]Transaction, error) {
	// Query database, get transactions count for blockhash.
	rows, err := dag.db.Query(
		`SELECT COUNT(*) FROM transactions_blocks WHERE block_hash = ?;`,
		hash[:],
	)
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
		SELECT txs.hash, txs.sig, txs.from_pubkey, txs.to_pubkey, txs.amount, txs.fee, txs.nonce, txblocks.txindex, txs.version
		FROM transactions txs
		JOIN transactions_blocks txblocks ON txs.hash = txblocks.transaction_hash
		WHERE txblocks.block_hash = ?
		ORDER BY txblocks.txindex ASC;
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
		txindex := uint64(0)
		version := 0 // TODO

		err := rows.Scan(&hash, &sig, &fromPubkey, &toPubkey, &amount, &fee, &nonce, &txindex, &version)
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
		tx.TxIndex = txindex
		tx.Version = byte(version)

		txs[txindex] = tx
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

// func (dag *BlockDAG) IsSynced(hash [32]byte) bool {
// 	// Synchronisation occurs in two phases: headers-only and then full block sync.
// 	// We can determine if we have fully synced a block like so:
// 	// - if the block has 0 transactions, then we only need the headers, and the block is fully synced.
// 	// - if the block has > 0 transactions, then we check if the number of transcations we have downloaded in the transactions_blocks table is equal to the number of transactions in the block. If it is, we have fully synced the block.
// 	return true // TODO.
// }

func (dag *BlockDAG) HasBlock(hash [32]byte) bool {
	rows, err := dag.db.Query(`
		select count(*) from blocks where hash = ?`,
		hash[:],
	)
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
	rows, err := dag.db.Query(`
		select hash from blocks order by acc_work desc limit 1
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

// Gets the latest block in the longest chain.
func (dag *BlockDAG) GetLatestFullTip() (Block, error) {
	// Query the highest accumulated work block in the database.
	rows, err := dag.db.Query(`
		SELECT hash 
		FROM (
			-- Case 1: Blocks with transactions.
			-- Only blocks with their transactions downloaded are considered for the "full tip".
			SELECT b.hash, b.acc_work
			FROM blocks b
			JOIN (
				SELECT block_hash, COUNT(*) AS num_transactions
				FROM transactions_blocks
				GROUP BY block_hash
			) tb ON b.hash = tb.block_hash
			WHERE b.num_transactions = tb.num_transactions

			UNION

			-- Case 2: Blocks without transactions.
			-- If a block has no transactions, then it is fully downloaded and is considered for the "full tip".
			SELECT b.hash, b.acc_work
			FROM blocks b
			WHERE b.num_transactions = 0
			AND NOT EXISTS (
				SELECT 1 
				FROM transactions_blocks tb 
				WHERE tb.block_hash = b.hash
			)
		) AS combined
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
// This returns the list in chronological order. list[0] is genesis.
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
	defer rows.Close()

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

	if direction == 1 {
		// Get the longest chain hash list.
		// First begin from the current tip, accumulate all hashes back to genesis.
		currentTip := dag.FullTip
		longestchain, err := dag.GetLongestChainHashList(currentTip.Hash, currentTip.Height+1) // TODO off-by-one error here.
		if err != nil {
			return list, err
		}

		// Find the start hash.
		startIndex := 0
		for i, hash := range longestchain {
			if hash == startHash {
				startIndex = i
				break
			}
		}

		// Slice the list from startIndex.
		list = longestchain[startIndex:]

		// Cap the list at depthFromTip.
		if uint64(len(list)) > depthFromTip {
			list = list[:depthFromTip]
		}

		return list, nil
	}

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
		ORDER BY depth ASC;`
	rows, err := dag.db.Query(
		queryDirectionBackwards,
		startHash[:],
		depthFromTip,
	)
	if err != nil {
		return list, err
	}
	defer rows.Close()

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





func (dag *BlockDAG) GetTransactionByHash(hash [32]byte) (*Transaction, error) {
	tx := Transaction{}

	// Query database.
	rows, err := dag.db.Query(
		`SELECT hash, sig, from_pubkey, to_pubkey, amount, fee, nonce, version FROM transactions WHERE hash = ?;`,
		hash[:],
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		hash := []byte{}
		sig := []byte{}
		fromPubkey := []byte{}
		toPubkey := []byte{}
		amount := uint64(0)
		fee := uint64(0)
		nonce := uint64(0)
		version := 0 // TODO

		err := rows.Scan(&hash, &sig, &fromPubkey, &toPubkey, &amount, &fee, &nonce, &version)
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
		tx.Version = byte(version)

		return &tx, nil
	} else {
		return nil, err
	}
}

func (dag *BlockDAG) GetEpochById(id string) (*Epoch, error) {
	epoch := Epoch{}

	// Query database.
	rows, err := dag.db.Query(
		`select id, start_block_hash, start_time, start_height, difficulty from epochs where id = ? limit 1`,
		id,
	)
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
		return nil, err
	}

	return &epoch, nil
}

func (dag *BlockDAG) GetTransactionBlocks(txHash [32]byte) ([][32]byte, error) {
	blocks := make([][32]byte, 0)

	// Query database.
	rows, err := dag.db.Query(
		`SELECT block_hash FROM transactions_blocks WHERE transaction_hash = ?;`,
		txHash[:],
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		blockHash := []byte{}
		err := rows.Scan(&blockHash)
		if err != nil {
			return nil, err
		}

		hash := [32]byte{}
		copy(hash[:], blockHash)

		blocks = append(blocks, hash)
	}

	return blocks, nil
}

func (dag *BlockDAG) GetDB() *sql.DB {
	return dag.db
}