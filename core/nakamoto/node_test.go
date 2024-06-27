package nakamoto

import (
	"testing"
	"github.com/liamzebedee/tinychain-go/core"
	"encoding/json"
	"time"
	"github.com/stretchr/testify/assert"
)

func newNodeFromConfig(t *testing.T, port string) (*Node) {
	// DAG.
	dag, _, _ := newBlockdag()

	// Miner.
	minerWallet, err := core.CreateRandomWallet()
	if err != nil {
		t.Fatalf("Failed to create miner wallet: %s", err)
	}

	miner := NewMiner(dag, minerWallet)

	// Peer.
	peer := NewPeerCore(PeerConfig{port: port})

	// Create the node.
	node := NewNode(dag, miner, peer)

	return node
}

func TestNewNode(t *testing.T) {
	node1 := newNodeFromConfig(t, "8080")
	// Start the node.
	go node1.Start()
}

func TestTwoNodeNetwork(t *testing.T) {
	node1 := newNodeFromConfig(t, "8080")
	node2 := newNodeFromConfig(t, "8081")

	// done := make(chan bool)

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
		assert.Equal(t, "new_block", msg.Type)
		// Print block hash.
		t.Logf("New block hash: %s", msg.RawBlock.HashStr())
	case <-time.After(8 * time.Second):
		t.Error("Timed out.")
	}
}