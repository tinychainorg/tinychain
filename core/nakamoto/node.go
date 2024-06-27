package nakamoto

import (
	"log"
	"os"
	"fmt"
)

var logger = log.New(os.Stdout, "corenode: ", log.Lshortfile)

type Node struct {
	Dag BlockDAG
	Miner *Miner
	Peer *PeerCore
}

func NewNode(dag BlockDAG, miner *Miner, peer *PeerCore) *Node {
	n := &Node{
		Dag: dag,
		Miner: miner,
		Peer: peer,
	}
	n.setup()
	return n
}

func (n *Node) setup() {
	// What does the node do?
	// - connect a blockdag
	// - connect a miner
	// - connect a peer
	
	// Listen for new blocks.
	n.Peer.OnNewBlock = func(b RawBlock) {
		logger.Printf("New block gossip from peer: block=%s\n", b.HashStr())

		if n.Dag.HasBlock(b.Hash()) {
			logger.Printf("Block already in DAG: block=%s\n", b.HashStr())
			return
		}

		isUnknownParent := n.Dag.HasBlock(b.ParentHash)
		if isUnknownParent {
			// We need to sync the chain.
			logger.Printf("Block parent unknown: block=%s\n", b.HashStr())
		}

		// Ingest the block.
		err := n.Dag.IngestBlock(b)
		if err != nil {
			logger.Printf("Failed to ingest block from peer: %s\n", err)
		}
	}

	n.Peer.OnGetBlocks = func(msg GetBlocksMessage) ([]RawBlock, error) {
		// Assert hashes length.
		MAX_GET_BLOCKS_LEN := 10
		if MAX_GET_BLOCKS_LEN < len(msg.BlockHashes) {
			return nil, fmt.Errorf("Too many hashes requested. Max is %d", MAX_GET_BLOCKS_LEN)
		}

		// TODO.
		return nil, nil
	}

	n.Miner.OnBlockSolution = func(b RawBlock) {
		logger.Printf("Mined new block: %s\n", b.HashStr())

		// Ingest the block.
		err := n.Dag.IngestBlock(b)
		if err != nil {
			logger.Printf("Failed to ingest block from peer: %s\n", err)
		}

		// Gossip the block.
		n.Peer.GossipBlock(b)
	}
}

func (n *Node) Start() {
	done := make(chan bool)
	
	go n.Peer.Start()
	go n.Miner.Start(1)

	<-done
}