package nakamoto

import (
	"log"
	"os"

)

var logger = log.New(os.Stdout, "corenode: ", log.Lshortfile)

type Node struct {
	dag BlockDAG
	miner *Miner
	peer *PeerCore
}

func NewNode(dag BlockDAG, miner *Miner, peer *PeerCore) *Node {
	return &Node{
		dag: dag,
		miner: miner,
		peer: peer,
	}
}

func (n *Node) Start() {
	// What does the node do?
	// - connect a blockdag
	// - connect a miner
	// - connect a peer
	
	// Listen for new blocks.
	n.peer.OnNewBlock = func(b RawBlock) {
		logger.Printf("New block from peer: %s\n", b.HashStr())
		
		// Ingest the block.
		err := n.dag.IngestBlock(b)
		if err != nil {
			logger.Printf("Failed to ingest block from peer: %s\n", err)
		}
	}

	n.miner.OnBlockSolution = func(b RawBlock) {
		logger.Printf("Mined new block: %s\n", b.HashStr())

		// Ingest the block.
		err := n.dag.IngestBlock(b)
		if err != nil {
			logger.Printf("Failed to ingest block from peer: %s\n", err)
		}

		// Gossip the block.
		n.peer.GossipBlock(b)
	}

	go n.miner.Start()
	go n.peer.Start()

	
}