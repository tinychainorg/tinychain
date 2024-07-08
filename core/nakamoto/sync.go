package nakamoto

// Network messages:
// - get_current_tip (include block header)
// - has_block? (blockhash)
// - get_block_headers from_blockhash skip=n limit=n
// - get_block_bodies from_blockhash skip=n limit=n

// I'm being too smart about this
// Simply contact all our peers, asking for latest tip
// 

// Things I am worrying about and not sure how to do:
// - where else do we recompute state?
// - where else do we restart the miner?
func (n *Node) Sync() {
	n.log.Printf("Performing sync...\n")

	// How do you sync?
	// You just do this forever:
	/*
	for {
		current_tip = blah

		get next +10 block headers from peers
		validate the chain
	}
	*/

	// Download all block headers from the common ancestor to tip.
	// Download all block bodies from the common ancestor to tip.


	// TODO parallelise this algo:
	// For one peer.
		// Get common ancestor. 640B space cost, 17 messages time cost, for 840,000 blocks.
		// Download all block headers from common ancestor to tip.
		// Validate block headers.
		// Download all block bodies from common ancestor to tip.
		// Validate and ingest blocks.
	
	// A parallel version of this algorithm:
	// - split the blocks we need up into batches of 10.
	// - perform one step:
	//   - batch_size / num_peers
	//   - download
	//   - measure peer download speed
	//    - fuck we have to check the peer even has the block for this step in their inventory

	// 6. Sync:
	//   a. Compute the common ancestor (interactive binary search).
	//   b. In parallel, download all the block headers from the common ancestor to the tip.
	//   c. Validate these block headers.
	//   d. In parallel, download all the block bodies (transactions) from the common ancestor to the tip.
	//   e. Validate and ingest these blocks.

	// 7. Sync complete, now rework:
	//   a. Recompute the state.
	//   b. Recompute the mempool. Mempool size = K txs.
	//      - Remove all transactions that have been sequenced in the chain. O(K) lookups.
	//      - Reinsert any transcations that were included in blocks that were orphaned, to a maximum depth of 1 day of blocks (144 blocks). O(144)
	//      - Revalidate the tx set. O(K).
	//   c. Begin mining on the new tip.
	// 
}

// Contacts all our peers in parallel, gets the block header of their tip, and returns the best tip based on total work.
func (n *Node) sync_getBestTipFromPeers() ([32]byte) {
	syncLog := NewLogger("node", "sync")

	// 1. Contact all our peers.
	// 2. Get their current tips in parallel.
	syncLog.Printf("Getting tips from %d peers...\n", len(n.Peer.peers))

	var wg sync.WaitGroup
	
	tips := make([]BlockHeader, 0)
	tipsChan := make(chan BlockHeader, len(n.Peer.peers))
    timeout := time.After(5 * time.Second)

	for _, peer := range n.Peer.peers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tip, err := n.Peer.GetTip(peer)
			if err != nil {
				syncLog.Printf("Failed to get tip from peer: %s\n", err)
				return
			}
			syncLog.Printf("Got tip from peer: hash=%s\n", tip.HashStr())
			tipsChan <- tip
		}()
	}

	go func() {
        wg.Wait()
        close(tipsChan)
    }()

	for {
		select {
		case tip, ok := <-tipsChan:
			if !ok {
				break
			}
			tips = append(tips, tip)
		case <-timeout:
			syncLog.Printf("Timed out getting tips from peers\n")
		}
	}
	
	syncLog.Printf("Received %d tips\n", len(tips))
	if len(tips) == 0 {
		syncLog.Printf("No tips received. Exiting sync.\n")
		return
	}
	
	// 3. Sort the tips by max(work).
	// 4. Reduce the tips to (tip, work, num_peers).
	// 5. Choose the tip with the highest work and the most peers mining on it.
	numPeersOnTip := make(map[[32]byte]int)
	tipWork := make(map[[32]byte]*big.Int)

	highestWork := big.NewInt(0)
	bestTipHash := [32]byte{}

	for _, tip := range tips {
		hash := tip.Hash()
		// TODO embed difficulty into block header so we can verify POW.
		work := CalculateWork(Bytes32ToBigInt(hash))

		// -1 if x < y
		// highestWork < work
		if highestWork.Cmp(work) == -1 {
			highestWork = work
			bestTipHash = hash
		}

		numPeersOnTip[hash] += 1
		tipWork[hash] = work
	}

	syncLog.Printf("Best tip: %s\n", bestTipHash)
	return bestTipHash
}

// Computes the common ancestor of our local canonical chain and a remote peer's canonical chain through an interactive binary search.
// O(log N * query_size).
// query_size = 32 B, N = 850,000
// log(850,000) * 32 = 20 * 32 = 640 B
func (n *Node) sync_computeCommonAncestorWithPeer(remotePeer Peer, local_chainhashes &[][32]byte) [32]byte {
	// 850,000 Bitcoin blocks since 2009.
	// 850000*32 = 27.2 MB
	// Not too bad, we can fit it all in memory.

	// 6a. Compute the common ancestor (interactive binary search).
	// This is a classical binary search algorithm.
	floor := 0
	ceil := len(local_chainhashes)
	n_iterations := 0

	for (floor + 1) < ceil {
		guess_idx := (floor + ceil) / 2
		guess_value := local_chainhashes[guess_idx]

		t.Logf("Iteration %d: floor=%d, ceil=%d, guess_idx=%d, guess_value=%x", n_iterations, floor, ceil, guess_idx, guess_value)
		n_iterations += 1

		// Send our tip's blockhash
		// Peer responds with "SEEN" or "NOT SEEN"
		// If "SEEN", we move to the right half.
		// If "NOT SEEN", we move to the left half.
		if n.Peer.HasBlock(peer, guess_value) {
			// Move to the right half.
			floor = guess_idx
		} else {
			// Move to the left half.
			ceil = guess_idx
		}
	}

	ancestor := local_chainhashes[floor]
	t.Logf("Common ancestor: %x", ancestor)
	t.Logf("Found in %d iterations.", n_iterations)
	return ancestor
}