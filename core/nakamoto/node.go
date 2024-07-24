package nakamoto

import (
	"fmt"
	"log"
	"time"
)

type Node struct {
	Dag           *BlockDAG
	Miner         *Miner
	Peer          *PeerCore
	StateMachine1 *StateMachine
	log           *log.Logger
	syncLog       *log.Logger
	stateLog      *log.Logger
}

func NewNode(dag *BlockDAG, miner *Miner, peer *PeerCore) *Node {
	stateMachine, err := NewStateMachine(nil)
	if err != nil {
		panic(err)
	}

	n := &Node{
		Dag:           dag,
		Miner:         miner,
		Peer:          peer,
		StateMachine1: stateMachine,
		log:           NewLogger("node", ""),
		syncLog:       NewLogger("node", "sync"),
		stateLog:      NewLogger("node", "state"),
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
		return n.Dag.FullTip.ToBlockHeader(), nil
	}

	n.Peer.OnSyncGetTipAtDepth = func(msg SyncGetTipAtDepthMessage) (SyncGetTipAtDepthReply, error) {
		direction := msg.Direction
		if direction != 1 && direction != -1 {
			return SyncGetTipAtDepthReply{}, fmt.Errorf("Invalid direction: %d", direction)
		}

		// Get the tip at the depth.
		path, err := n.Dag.GetPath(msg.FromBlock, uint64(msg.Depth), msg.Direction)
		if err != nil {
			return SyncGetTipAtDepthReply{}, err
		}

		if len(path) == 0 {
			return SyncGetTipAtDepthReply{}, fmt.Errorf("No tip found at depth %d", msg.Depth)
		}

		return SyncGetTipAtDepthReply{
			Tip: path[len(path)-1],
		}, nil

	}

	// Upload blocks to other peers.
	n.Peer.OnSyncGetData = func(msg SyncGetBlockDataMessage) (SyncGetBlockDataReply, error) {
		reply := SyncGetBlockDataReply{
			Headers: []BlockHeader{},
			Bodies:  [][]RawTransaction{},
		}

		// 1. Get the full path forward from baseNode -> baseNode.height + WINDOW_SIZE
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
		// 1. Rebuild state.
		// 2. Regenerate current mempool.

		n.stateLog.Printf("rebuild-state\n")
		start := time.Now()

		err := n.rebuildState()
		if err != nil {
			n.stateLog.Printf("Failed to rebuild state: %s\n", err)
			return
		}

		duration := time.Since(start)
		n.stateLog.Printf("rebuild-state completed duration=%s n_blocks=%d\n", duration.String(), n.Dag.FullTip.Height)
	}

	// When we get a tx, add it to the mempool.
	// When mempool changes, restart miner.
	// When DAG tip changes, restart miner.
	// When we first boot node, perform a full sync before doing anything.
	// When we get new block that doesn't have known parent, do a sync.

	// 7. Sync complete, now rework:
	//   a. Recompute the state.
	//   b. Recompute the mempool. Mempool size = K txs.
	//      - Remove all transactions that have been sequenced in the chain. O(K) lookups.
	//      - Reinsert any transcations that were included in blocks that were orphaned, to a maximum depth of 1 day of blocks (144 blocks). O(144)
	//      - Revalidate the tx set. O(K).
	//   c. Begin mining on the new tip.

	// When we get new transaction, add it to mempool.
	n.Peer.OnNewTransaction = func(tx RawTransaction) {
		// Add transaction to mempool.
		// TODO.
	}
}

func (n *Node) rebuildState() error {
	longestChainHashList, err := n.Dag.GetLongestChainHashList(n.Dag.FullTip.Hash, n.Dag.FullTip.Height)
	if err != nil {
		n.stateLog.Printf("Failed to get longest chain hash list: %s\n", err)
		return err
	}

	state2, err := RebuildState(n.Dag, *n.StateMachine1, longestChainHashList)
	if err != nil {
		n.stateLog.Printf("Failed to rebuild state: %s\n", err)
		return err
	}

	n.StateMachine1 = state2

	return nil
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
