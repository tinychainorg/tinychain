package nakamoto

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/liamzebedee/tinychain-go/core"
	"github.com/stretchr/testify/assert"
)

func newNodeFromConfig(t *testing.T) *Node {
	// DAG.
	dag, _, _, _ := newBlockdag()

	// Miner.
	minerWallet, err := core.CreateRandomWallet()
	if err != nil {
		t.Fatalf("Failed to create miner wallet: %s", err)
	}

	miner := NewMiner(dag, minerWallet)

	// Peer.
	peer := NewPeerCore(PeerConfig{address: "127.0.0.1", port: getRandomPort()})

	// Create the node.
	node := NewNode(dag, miner, peer)

	return node
}

func TestNewNode(t *testing.T) {
	node1 := newNodeFromConfig(t)
	// Start the node.
	go node1.Start()

	ch := make(chan bool)

	// Setup timeout.
	go func() {
		time.Sleep(1500 * time.Millisecond)
		ch <- true
	}()

	<-ch
}

func TestTwoNodesGossipBlocks(t *testing.T) {
	assert := assert.New(t)

	node1 := newNodeFromConfig(t)
	node2 := newNodeFromConfig(t)

	// Start the node.
	go node1.Peer.Start()
	go node2.Peer.Start()

	// Wait for peers to come online.
	waitForPeersOnline([]*PeerCore{node1.Peer, node2.Peer})

	// Bootstrap.
	node1.Peer.Bootstrap([]string{
		node2.Peer.GetLocalAddr(),
	})
	node2.Peer.Bootstrap([]string{
		node1.Peer.GetLocalAddr(),
	})

	// Node 1 solves a block, and gossips it to node 2.
	newBlockChan := make(chan NewBlockMessage)
	node2.Peer.server.RegisterMesageHandler("new_block", func(message []byte) (interface{}, error) {
		var msg NewBlockMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			return nil, err
		}

		newBlockChan <- msg
		return nil, nil
	})

	// Start node 1 miner.
	go node1.Miner.Start(1)

	// Wait for new block.
	select {
	case msg := <-newBlockChan:
		assert.Equal("new_block", msg.Type)
		// Print block hash.
		t.Logf("New block hash: %s", msg.RawBlock.HashStr())
	case <-time.After(8 * time.Second):
		t.Error("Timed out.")
	}
}

func TestTwoNodeEqualMining(t *testing.T) {
	assert := assert.New(t)
	node1 := newNodeFromConfig(t)
	node2 := newNodeFromConfig(t)

	// Start the node.
	go node1.Peer.Start()
	go node2.Peer.Start()

	// Wait for peers to come online.
	waitForPeersOnline([]*PeerCore{node1.Peer, node2.Peer})

	// Bootstrap.
	node1.Peer.Bootstrap([]string{
		node2.Peer.GetLocalAddr(),
	})
	node2.Peer.Bootstrap([]string{
		node1.Peer.GetLocalAddr(),
	})

	// Both nodes mine 20 blocks.
	// For the purposes of making this test reproducible, due to how
	// Go's coroutine scheduler isn't exactly fair, we will ping-pong between
	// the two nodes to ensure that they both mine a block each.
	for i := 0; i < 30/2; i++ {
		// Node 1 mines a block.
		node1.Miner.Start(1)
		// Node 2 mines a block.
		node2.Miner.Start(1)
	}

	// Then we check the tips.
	tip1 := node1.Dag.Tip
	tip2 := node2.Dag.Tip

	// Check that the tips are the same.
	assert.Equal(tip1, tip2)
}

func TestTwoNodeUnequalMining(t *testing.T) {
	assert := assert.New(t)
	node1 := newNodeFromConfig(t)
	node2 := newNodeFromConfig(t)

	// Start the node.
	go node1.Peer.Start()
	go node2.Peer.Start()

	// Wait for peers to come online.
	waitForPeersOnline([]*PeerCore{node1.Peer, node2.Peer})

	// Bootstrap.
	node1.Peer.Bootstrap([]string{
		node2.Peer.GetLocalAddr(),
	})
	node2.Peer.Bootstrap([]string{
		node1.Peer.GetLocalAddr(),
	})

	// In this test, one node mines more blocks than another.
	// And we test the sync.
	node1Speed := int64(3)
	node2Speed := int64(1)

	for i := 0; i < 30; i++ {
		// Node 1 mines blocks.
		node1.Miner.Start(node1Speed)

		// Node 2 mines blocks.
		node2.Miner.Start(node2Speed)
	}

	// Wait for the nodes to sync.
	time.Sleep(25 * time.Second)

	// Then we check the tips.
	tip1 := node1.Dag.Tip
	tip2 := node2.Dag.Tip

	// Check that the tips are the same.
	assert.Equal(tip1.Hash, tip2.Hash)
	// Check that we are on node1's branch which has more work.
	node1Tip := node1.Dag.Tip
	// Print the height of the tip.
	t.Logf("Tip height: %d", node1Tip.Height)
}

func TestStateMachineUpdatesForTip(t *testing.T) {
	// When the node tip is updated, then we recompute the state given the new transaction sequence.
}

func TestNodeSyncMissingBlocksUnknownParent(t *testing.T) {

}

// Here we test synchronisation. Will a node that mines misses gossipped blocks catch up with the network?
func TestNodeSyncMissingBlocks(t *testing.T) {
	assert := assert.New(t)
	node1 := newNodeFromConfig(t)
	node2 := newNodeFromConfig(t)

	// Start node1 only.
	go node1.Peer.Start()
	// Wait for node1 to come online.
	waitForPeersOnline([]*PeerCore{node1.Peer})

	// Bootstrap.
	node1.Peer.Bootstrap([]string{
		node2.Peer.GetLocalAddr(),
	})
	node2.Peer.Bootstrap([]string{
		node1.Peer.GetLocalAddr(),
	})

	node1.Miner.Start(10)

	// Start node2.
	go node2.Peer.Start()
	// Wait for node2 to come online.
	waitForPeersOnline([]*PeerCore{node2.Peer})

	// Wait for node 2 to sync completely.

	// Then we check the tips.
	tip1 := node1.Dag.Tip
	tip2 := node2.Dag.Tip

	// Check that the tips are the same.
	assert.Equal(tip1, tip2)
}

func TestNodeSyncNoReorg(t *testing.T) {

}

// func TestNodeSyncReorg(t *testing.T) {
// 	// Two peers for test:
// 	// - peer A. Has mined a chain of 10 blocks.
// 	// - peer B. Has mined an alternative chain of 5 blocks.
// 	//
// 	// Test routine:
// 	// Peer A starts up, mines 10 blocks.
// 	// Peer B starts up, mines 5 blocks, then connects to peer B.
// 	// Peer B syncs with peer A.
// 	// Peer B downloads the headers of peer A, and then downloads the bodies, ingests the blocks.
// 	// Assert: peer B tip == peer A tip (longest chain)
// 	// Assert: peer B state == peer A state (longest chain)

// 	// assert := assert.New(t)

// 	node1 := newNodeFromConfig(t)
// 	node2 := newNodeFromConfig(t)

// 	// Peer 1 starts up.
// 	node1.Peer.Start()
// 	node1.Miner.Start(10)

// 	node2.Peer.Start()
// 	node2.Miner.Start(5)

// 	// Connect peer 2 to peer 1.
// 	node2.Peer.Bootstrap([]string{
// 		node1.Peer.GetLocalAddr(),
// 	})
// }

func TestNodeBuildState(t *testing.T) {
	// Peer A. Mines chain of 10 blocks.
	// Assert that the state is correctly computed.

	node1 := newNodeFromConfig(t)

	// Peer 1 starts up.
	node1.Peer.Start()
	node1.Miner.Start(10)

	//
}

func TestNodeBuildStateReorg(t *testing.T) {
	// Peer A. Mines chain of 10 blocks.
	// Assert that the state is correctly computed.
}

// One part of the block sync algorithm is determining the common ancestor of two chains:
//
//	Chain 1: the chain we have on our local node.
//	Chain 2: the chain of a remote peer who has a more recent tip.
//
// We determine the common ancestor in order to download the most minimal set of block headers required to sync to the latest tip.
// There are a few approaches to this:
// - naive approach: download all headers from the tip to the remote peer's genesis block, and then compare the headers to find the common ancestor. This is O(N) where N is the length of the longest chain.
// - naive approach 2: send the peer the block we have at (height - 6), which is according to Nakamoto's calculations, "probabilistically final" and unlikely to be reorg-ed. Ask them if they have this block, and if so, sync the remaining 6 blocks. This fails when there is ongoing volatile reorgs, as well as doesn't work for a full sync.
// - slightly less naive approach: send the peer "checkpoints" at a regular interval. So for the full list of block hashes, we send H/I where I is the interval size, and use this to sync. This is O(H/I).
// - slightly slightly less naive approach: send the peer a list of "checkpoints" at exponentially decreasing intervals. This is smart since the finality of a block increases exponentially with the number of confirmations. This is O(H/log(H)).
// - the most efficient approach. Interactively binary search with the node. At each step of the binary search, we split their view of the chain hash list in half, and ask them if they have the block at the midpoint.
//
// Let me explain the binary search.
// <------------------------>   our view
// <------------------------> their view
// n=1
// <------------|-----------> their view
// <------------------|-----> their view
// <---------------|--------> their view
// At each iteration we ask: do you have a block at height/2 with this hash?
// - if the answer is yes, we move to the right half.
// - if the answer is no, we move to the left half.
// We continue until the length of our search space = 1.
//
// Now for some modelling.
// Finding the common ancestor is O(log N). Each message is (blockhash [32]byte, height uint64). Message size is 40 bytes.
// Total networking cost is O(40 * log N), bitcoin's chain height is 850585, O(40 * log 850585) = O(40 * 20) = O(800) bytes.
// Less than 1KB of data to find common ancestor.
func TestInteractiveBinarySearchFindCommonAncestor(t *testing.T) {
	local_chainhashes := [][32]byte{}
	remote_chainhashes := [][32]byte{}

	// Populate blockhashes for test.
	for i := 0; i < 100; i++ {
		local_chainhashes = append(local_chainhashes, uint64To32ByteArray(uint64(i)))
		remote_chainhashes = append(remote_chainhashes, uint64To32ByteArray(uint64(i)))
	}
	// Set remote to branch at block height 90.
	for i := 90; i < 100; i++ {
		remote_chainhashes[i] = uint64To32ByteArray(uint64(i + 1000))
	}

	// Print both for debugging.
	t.Logf("Local chainhashes:\n")
	for _, x := range local_chainhashes {
		t.Logf("%x", x)
	}
	t.Logf("\n")
	t.Logf("Remote chainhashes:\n")
	for _, x := range remote_chainhashes {
		t.Logf("%x", x)
	}
	t.Logf("\n")

	// Peer method.
	hasBlockhash := func(blockhash [32]byte) bool {
		for _, x := range remote_chainhashes {
			if x == blockhash {
				return true
			}
		}
		return false
	}

	//
	// Find the common ancestor.
	//

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
		if hasBlockhash(guess_value) {
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

	expectedIterations := math.Ceil(math.Log2(float64(len(local_chainhashes))))
	t.Logf("Expected iterations: %f", expectedIterations)
}

func uint64To32ByteArray(num uint64) [32]byte {
	var arr [32]byte
	binary.BigEndian.PutUint64(arr[24:], num) // Store the uint64 in the last 8 bytes of the array
	return arr
}