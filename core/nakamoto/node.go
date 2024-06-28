package nakamoto

import (
	"fmt"
)

var nodeLog = NewLogger("node")

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
}

func (n *Node) Sync() {
	// Contact all our peers.
	// Get their current tips.
	// Get the blocks for each of these tips.
	// Verify the POW on the tip to check the tip is valid.
	// Select the tip with the highest work according to ParentTotalWork.
	// Download the blocks from the tip to the common ancestor from all our peers.
	// Store them in a temporary storage.
	// Ingest the blocks in reverse order.
	// Begin mining.
}

func (n *Node) Start() {
	done := make(chan bool)

	go n.Peer.Start()
	go n.Miner.Start(-1)

	<-done
}

func (n *Node) Shutdown() {
	// Close the database.
	err := n.Dag.db.Close()
	if err != nil {
		nodeLog.Printf("Failed to close database: %s\n", err)
	}
}
