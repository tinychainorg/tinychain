package nakamoto

import (
	"log"
	"os"

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
		
		// Ingest the block.
		err := n.Dag.IngestBlock(b)
		if err != nil {
			logger.Printf("Failed to ingest block from peer: %s\n", err)
		}
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