package nakamoto

import (
	"fmt"
	"log"
	"sync"
	"time"
	"math/big"
)

type Node struct {
	Dag   BlockDAG
	Miner *Miner
	Peer  *PeerCore
	log *log.Logger
}

func NewNode(dag BlockDAG, miner *Miner, peer *PeerCore) *Node {
	n := &Node{
		Dag:   dag,
		Miner: miner,
		Peer:  peer,
		log: NewLogger("node", ""),
	}
	n.setup()
	return n
}

func (n *Node) setup() {
	// Listen for new blocks.
	n.Peer.OnNewBlock = func(b RawBlock) {
		n.log.Printf("New block gossip from peer: block=%s\n", b.HashStr())

		if n.Dag.HasBlock(b.Hash()) {
			n.log.Printf("Block already in DAG: block=%s\n", b.HashStr())
			return
		}

		isUnknownParent := n.Dag.HasBlock(b.ParentHash)
		if isUnknownParent {
			// We need to sync the chain.
			n.log.Printf("Block parent unknown: block=%s\n", b.HashStr())
		}

		// Ingest the block.
		err := n.Dag.IngestBlock(b)
		if err != nil {
			n.log.Printf("Failed to ingest block from peer: %s\n", err)
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
		n.log.Printf("Mined new block: %s\n", b.HashStr())

		// Ingest the block.
		err := n.Dag.IngestBlock(b)
		if err != nil {
			n.log.Printf("Failed to ingest block from miner: %s\n", err)
		}

		// Gossip the block.
		n.Peer.GossipBlock(b)
	}

	// Gossip the latest tip.
	n.Peer.OnGetTip = func(msg GetTipMessage) (BlockHeader, error) {
		tip := n.Dag.Tip
		// Convert to BlockHeader
		blockHeader := BlockHeader{
			ParentHash: 		  tip.ParentHash,
			ParentTotalWork: 	  tip.ParentTotalWork,
			Timestamp: 		  tip.Timestamp,
			NumTransactions: 	  tip.NumTransactions,
			TransactionsMerkleRoot: tip.TransactionsMerkleRoot,
			Nonce                  : tip.Nonce,
			Graffiti               : tip.Graffiti,
		}
		return blockHeader, nil
	}

	// Recompute the state after a new tip.
	n.Dag.OnNewTip = func(new_tip Block, prev_tip Block) {
		// Find the common ancestor of the two tips.
		// Revert the state to this ancestor.
		// Recompute the state from the ancestor to the new tip.
	}
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
		n.log.Printf("Failed to close database: %s\n", err)
	}
}
