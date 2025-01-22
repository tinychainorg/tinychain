package nakamoto

import (
	"encoding/binary"
	"encoding/json"
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
	peer := NewPeerCore(PeerConfig{ipAddress: "127.0.0.1", port: getRandomPort()})

	// Create the node.
	node := NewNode(&dag, miner, peer)

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
	tip1 := node1.Dag.FullTip
	tip2 := node2.Dag.FullTip

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

	for i := 0; i < 10; i++ {
		// Node 1 mines blocks.
		node1.Miner.Start(node1Speed)

		// Node 2 mines blocks.
		node2.Miner.Start(node2Speed)
	}

	// Wait for the nodes to sync.
	time.Sleep(25 * time.Second)

	// Then we check the tips.
	tip1 := node1.Dag.FullTip
	tip2 := node2.Dag.FullTip

	// Check that the tips are the same.
	assert.Equal(tip1.Hash, tip2.Hash)
	// Check that we are on node1's branch which has more work.
	node1Tip := node1.Dag.FullTip
	// Print the height of the tip.
	t.Logf("Tip height: %d", node1Tip.Height)
}

// Here we test synchronisation. Will a node that mines misses gossipped blocks catch up with the network?
func TestNodeSyncMissingBlocks(t *testing.T) {
	t.SkipNow() // TODO when sync is done.

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
	tip1 := node1.Dag.FullTip
	tip2 := node2.Dag.FullTip

	// Check that the tips are the same.
	assert.Equal(tip1, tip2)
}

func uint64To32ByteArray(num uint64) [32]byte {
	var arr [32]byte
	binary.BigEndian.PutUint64(arr[24:], num) // Store the uint64 in the last 8 bytes of the array
	return arr
}
