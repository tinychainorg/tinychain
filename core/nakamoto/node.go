package nakamoto

import (
	"fmt"
)

var nodeLog = NewLogger("node", "")

type Node struct {
	Dag   BlockDAG
	Miner *Miner
	Peer  *PeerCore
}

func NewNode(dag BlockDAG, miner *Miner, peer *PeerCore) *Node {
	n := &Node{
		Dag:   dag,
		Miner: miner,
		Peer:  peer,
	}
	n.setup()
	return n
}

func (n *Node) setup() {
	// Listen for new blocks.
	n.Peer.OnNewBlock = func(b RawBlock) {
		nodeLog.Printf("New block gossip from peer: block=%s\n", b.HashStr())

		if n.Dag.HasBlock(b.Hash()) {
			nodeLog.Printf("Block already in DAG: block=%s\n", b.HashStr())
			return
		}

		isUnknownParent := n.Dag.HasBlock(b.ParentHash)
		if isUnknownParent {
			// We need to sync the chain.
			nodeLog.Printf("Block parent unknown: block=%s\n", b.HashStr())
		}

		// Ingest the block.
		err := n.Dag.IngestBlock(b)
		if err != nil {
			nodeLog.Printf("Failed to ingest block from peer: %s\n", err)
		}
	}

	// Upload blocks to other peers.
	n.Peer.OnGetBlocks = func(msg GetBlocksMessage) ([][]byte, error) {
		// Assert hashes length.
		MAX_GET_BLOCKS_LEN := 10
		if MAX_GET_BLOCKS_LEN < len(msg.BlockHashes) {
			return nil, fmt.Errorf("Too many hashes requested. Max is %d", MAX_GET_BLOCKS_LEN)
		}

		reply := make([][]byte, 0)
		for _, hash := range msg.BlockHashes {
			blockhash := HexStringToBytes32(hash)

			// Get the raw block.
			rawBlockData, err := n.Dag.GetRawBlockDataByHash(blockhash)
			if err != nil {
				// If there is an error getting the block hash, skip it.
				continue
			}

			reply = append(reply, rawBlockData)
		}

		// return reply, nil
		return nil, nil
	}

	// Gossip blocks when we mine a new solution.
	n.Miner.OnBlockSolution = func(b RawBlock) {
		nodeLog.Printf("Mined new block: %s\n", b.HashStr())

		// Ingest the block.
		err := n.Dag.IngestBlock(b)
		if err != nil {
			nodeLog.Printf("Failed to ingest block from miner: %s\n", err)
		}

		// Gossip the block.
		n.Peer.GossipBlock(b)
	}

	// Gossip the latest tip.
	n.Peer.OnGetTip = func(msg GetTipMessage) ([32]byte, error) {
		tip := n.Dag.Tip.Hash
		return tip, nil
	}

	// Recompute the state after a new tip.
	n.Dag.OnNewTip = func(new_tip Block, prev_tip Block) {
		// Find the common ancestor of the two tips.
		// Revert the state to this ancestor.
		// Recompute the state from the ancestor to the new tip.
	}
}

func (n *Node) Sync() {
	// 1. Contact all our peers.
	// 2. Get their current tips (including block header and "optimistic" height).
	// 3. Sort the tips by max(work).
	// 4. Reduce the tips to (tip, work, num_peers).
	// 5. Choose the tip with the highest work and the most peers mining on it.
	// 6. Sync:
	//   a. Compute the common ancestor.
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

	


	// 
	// Simple modelling of the costs:
	// 
	// Assumptions:
	// Number of peers : P = 5
	// Download bandwidth : 1MB/s
	// Block rate = 1 block / 10 mins
	// Max block size = 1 MB
	// Block header size = 208 B
	// Transaction size = 155 B
	// Block body max size = 999792 B
	// Maximum transactions per block = 6450
	// Assuming full blocks, 1 block = 1MB
	// Our last sync = 1 week ago = 7*24*60/(1/10) = 1008 blocks
	// 
	// Getting tips from all peers. O(P * 208) bytes.
	// Downloading block headers. O(1008 * 208) bytes.
	//   Total download = 1008 * 208 = 209,664 = 209 KB
	//   Download on each peer. O(1008 * 208 / P) bytes per peer.
	//                          O(1008 * 208 / 5) = 41932 = 41 KB
	//   Time to sync headers = 1008 * 208 / 1 MB/s = 1000*208/1000/1000 = 0.2s
	// Downloading block bodies. O(1008 * 999792)
	//   Total download = 1008 * 999792 = 1,007,790,336 = 1.007 GB
	//   Download on each peer. O(1008 * 999792 / P) bytes per peer.
	//                          O(1008 * 999792 / 5) = 201,558,067 = 201 MB
	//   Time to sync bodies = 1008 * 999792 / 1 MB/s = 1000*999792/1000/1000 = 999s = 16.65 mins
	// 

	// 
	// What about when we simply miss a block in the 1...6 finality period?
	// We will know because the peer sends us the new block, and we don't know the parent.
	// 
	// 

	// tips := make([]BlockHeader, 0)
	// for _, peer := range n.Peer.Peers {
	// 	tip, err := peer.GetTip()
	// 	if err != nil {
	// 		nodeLog.Printf("Failed to get tip from peer: %s\n", err)
	// 		continue
	// 	}

	// 	// Get the block header for each of these tips.
	// 	blockHeaders, err := peer.GetBlockHeaders(tip)
	// 	if err != nil {
	// 		nodeLog.Printf("Failed to get block header from peer: %s\n", err)
	// 		continue
	// 	}
	// 	if len(blockHeaders) == 0 {
	// 		nodeLog.Printf("No block headers from peer: %s\n", peer.GetLocalAddr())
	// 		continue
	// 	}
	// 	blockHeader := blockHeaders[0]

	// 	// Verify block header's work.
	// 	// TODO.
	// 	work := CalculateWork(Bytes32ToBigInt(blockHeader.Hash()))

	// 	tips = append(tips, blockHeader)
	// }

	// Verify the POW on the tip to check the tip is valid.
	// Select the tip with the highest work according to ParentTotalWork.
	// TODO.

	// Get a set of block "checkpoints", evenly spaced out, from the peer.
	// O(N/(10*6*24)) = O(N/1440) blocks = 1 checkpoint for each day.
	// We can then use these to determine:
	// (1) the multiple branches of the blocktree (if any)
	// (2) which peers store which blocks
	// (3) a mapping of ancestor -> []peer

	// Expressed by intuition, we ask all our peers for their view of the longest chain on each day stretching back to genesis.
	// From there, we can determine the "common base" they are building of, and where it diverges.
	// Where there is a common base ancestor, we can request that block history from all those peers who share that common ancestor IN PARALLEL.
	// It is akin to BitTorrent, where we can download chunks in parallel from all peers in a swarm.

	// Download all of the block headers from tip backwards.
	// Download the blocks from the tip to the common ancestor from all our peers.
	// Store them in a temporary storage.
	// Ingest the blocks in reverse order.
	// Begin mining.
}

func (n *Node) Start() {
	done := make(chan bool)

	go n.Peer.Start()
	// go n.Miner.Start(-1)

	<-done
}

func (n *Node) Shutdown() {
	// Close the database.
	err := n.Dag.db.Close()
	if err != nil {
		nodeLog.Printf("Failed to close database: %s\n", err)
	}
}
