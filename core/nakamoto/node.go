package nakamoto

import (
	"fmt"
	"log"
	"time"
)

type Node struct {
	Dag     *BlockDAG
	Miner   *Miner
	Peer    *PeerCore
	StateMachine1 *StateMachine
	log     *log.Logger
	syncLog *log.Logger
}

type SyncState struct {
	// How does the sync design work with the block DAG?
	// The block DAG thus far is only designed for queuing blocks.
}

func NewNode(dag *BlockDAG, miner *Miner, peer *PeerCore) *Node {
	stateMachine, err := NewStateMachine(nil)
	if err != nil {
		panic(err)
	}

	n := &Node{
		Dag:     dag,
		Miner:   miner,
		Peer:    peer,
		StateMachine1: stateMachine,
		log:     NewLogger("node", ""),
		syncLog: NewLogger("node", "sync"),
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
		tip := n.Dag.FullTip
		// Convert to BlockHeader
		blockHeader := BlockHeader{
			ParentHash:             tip.ParentHash,
			ParentTotalWork:        tip.ParentTotalWork, // TODO: Fix this.
			Timestamp:              tip.Timestamp,
			NumTransactions:        tip.NumTransactions,
			TransactionsMerkleRoot: tip.TransactionsMerkleRoot,
			Nonce:                  tip.Nonce,
			Graffiti:               tip.Graffiti,
		}
		return blockHeader, nil
	}

	// Upload blocks to other peers.
	n.Peer.OnSyncGetData = func(msg SyncGetDataMessage) (SyncGetDataReply, error) {
		reply := SyncGetDataReply{
			Headers: make([]BlockHeader, 0),
			Bodies:  make([][]RawTransaction, 0),
		}

		// 1. Get the path forward from baseNode -> baseNode.height + WINDOW_SIZE
		nodes1, err := n.Dag.GetPath(msg.FromBlock, uint64(msg.Heights.Size()), 1)
		if err != nil {
			return reply, err
		}

		// 2. Filter the nodes included in the height set. 
		nodes2 := make([][32]byte, 0)
		for i, node := range nodes1 {
			if msg.Heights.Contains(i) {
				nodes2 = append(nodes2, node)
			}
		}

		// 3. Fetch their headers or bodies and return them.
		if msg.Headers {
			// Get the headers.
			for _, node := range nodes2 {
				block, err := n.Dag.GetBlockByHash(node)
				if err != nil {
					return reply, err
				}

				reply.Headers = append(reply.Headers, block.ToBlockHeader())
			}
		} else if msg.Bodies {
			// Get the bodies.
			for _, node := range nodes2 {
				// Get the transactions.
				transactions, err := n.Dag.GetBlockTransactions(node)
				if err != nil {
					return reply, err
				}

				rawTransactions := make([]RawTransaction, 0)
				for _, tx := range *transactions {
					rawTransactions = append(rawTransactions, tx.ToRawTransaction())
				}

				reply.Bodies = append(reply.Bodies, rawTransactions)
			}
		}

		return reply, nil
	}

	// Recompute the state after a new tip.
	n.Dag.OnNewFullTip = func(new_tip Block, prev_tip Block) {
		n.log.Printf("rebuild-state\n")
		start := time.Now()

		// Rebuild state.
		n.rebuildState()
		
		duration := time.Since(start)
		n.log.Printf("rebuild-state completed duration=%s blocks=%d\n", n.Dag.FullTip.Height, duration.String())
	}
}

func (n *Node) rebuildState() {
	longestChainHashList, err := n.Dag.GetLongestChainHashList(n.Dag.FullTip.Hash, uint64(10000000000000000000))
	if err != nil {
		n.log.Printf("Failed to get longest chain hash list: %s\n", err)
		return
	}

	state2, err := RebuildState(n.Dag, *n.StateMachine1, longestChainHashList)
	if err != nil {
		n.log.Printf("Failed to rebuild state: %s\n", err)
		return
	}

	n.StateMachine1 = state2
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
