package nakamoto

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func fullyConnectPeers(peers []*PeerCore) {
	for _, peer := range peers {
		for _, peer2 := range peers {
			if peer != peer2 {
				peer.AddPeer(peer2.GetLocalAddr())
			}
		}
	}
}

func waitForNodesToSyncSameTip(nodes []*Node) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("Timed out waiting for tips to sync.")
		default:
			basetip := nodes[0].Dag.FullTip
			tipsAllMatch := true
			for _, node := range nodes {
				if node.Dag.FullTip.HashStr() != basetip.HashStr() {
					tipsAllMatch = false
					break
				}
			}

			if tipsAllMatch {
				return nil
			}
		}
	}
}

func setupTestNetwork(t *testing.T) []*Node {
	node1 := newNodeFromConfig(t)
	node2 := newNodeFromConfig(t)
	node3 := newNodeFromConfig(t)

	// Start the node.
	go node1.Peer.Start()
	go node2.Peer.Start()
	go node3.Peer.Start()

	// Wait for peers to come online.
	waitForPeersOnline([]*PeerCore{node1.Peer, node2.Peer, node3.Peer})

	// Bootstrap.
	fullyConnectPeers([]*PeerCore{node1.Peer, node2.Peer, node3.Peer})

	// Wait for the nodes to sync.
	// time.Sleep(25 * time.Second)

	return []*Node{node1, node2, node3}
}

func TestSyncGetPeerTips(t *testing.T) {
	assert := assert.New(t)
	peers := setupTestNetwork(t)

	node1 := peers[0]
	node2 := peers[1]
	node3 := peers[2]

	// Then we check the tips.
	tip1 := node1.Dag.FullTip
	tip2 := node2.Dag.FullTip
	tip3 := node3.Dag.FullTip

	// Print the height of the tip.
	t.Logf("Tip 1 height: %d", tip1.Height)
	t.Logf("Tip 2 height: %d", tip2.Height)
	t.Logf("Tip 3 height: %d", tip3.Height)

	// Check that the tips are the same.
	assert.Equal(tip1.HashStr(), tip2.HashStr())
	assert.Equal(tip1.HashStr(), tip3.HashStr())

	// Setup 3 peers.
	// 2 are on the same chain.
	// 1 is on a different chain.
	// Now mine some blocks.
	node1.Miner.Start(15)

	// Setup to wait until tips are the same or timeout after 5s.
	err := waitForNodesToSyncSameTip(peers)
	assert.Nil(err)

	// Now let node3 mine some blocks, disable sync to other nodes.
	time.Sleep(400 * time.Millisecond)
	node1.Peer.OnNewBlock = nil
	node2.Peer.OnNewBlock = nil
	node3.Peer.OnNewBlock = nil
	node3.Miner.Start(2)

	// Assert node3 tip != node1.Tip
	tip1 = node1.Dag.FullTip
	tip2 = node2.Dag.FullTip
	tip3 = node3.Dag.FullTip
	t.Logf("Tip 1 height: %d", tip1.Height)
	t.Logf("Tip 2 height: %d", tip2.Height)
	t.Logf("Tip 3 height: %d", tip3.Height)
	assert.Equal(tip1.HashStr(), tip2.HashStr())
	assert.NotEqual(tip3.HashStr(), tip1.HashStr())

	// Now we test the function.
	node1_peerTips, _ := node1.getPeerTips(tip1.Hash, uint64(6), 1)
	node2_peerTips, _ := node2.getPeerTips(tip1.Hash, uint64(6), 1)
	node3_peerTips, _ := node3.getPeerTips(tip1.Hash, uint64(1), 1)

	// Print all tips
	logTips := func(label string, tips map[[32]byte][]Peer) {
		t.Logf("%s peer tips:", label)
		for hash, peers := range tips {
			peersStr := ""
			for _, peer := range peers {
				peersStr += peer.String() + " "
			}
			t.Logf("%x: %s", hash, peersStr)
		}
	}
	logTips("Node 1", node1_peerTips)
	logTips("Node 2", node2_peerTips)
	logTips("Node 3", node3_peerTips)

	// Now we assert the following:
	// (1) Node 3 should have 2 nodes with the same tip.
	// (2) Nodes 1,2 should have 2 nodes with 2 different tips.
	//
	// In terms of what this looks like conceptually:
	// node1_peerTips:
	//  tip1: [node2]
	//  tip3: [node3]
	// node2_peerTips:
	//  tip1: [node1]
	//  tip3: [node3]
	// node3_peerTips:
	//  tip1: [node1 node2]
	//
	// NOTE: We cannot test the specific peers found yet because we don't have a suitable way to globally identify them. We could use the external address though it may appear differently for each peer. TODO.

	// (1) Node 3 should have 2 nodes with the same tip.
	assert.Equal(1, len(node3_peerTips))            // only one entry.
	assert.Contains(node3_peerTips, tip1.Hash)      // tip1 is the only entry.
	assert.Equal(2, len(node3_peerTips[tip1.Hash])) // 2 peers with the same tip.

	// (2a) Node 1 should have 2 nodes with 2 different tips.
	assert.Equal(2, len(node1_peerTips)) // two entries.
	// tip1 and tip3 are the entries.
	assert.Contains(node1_peerTips, tip1.Hash)
	assert.Contains(node1_peerTips, tip3.Hash)
	// tip1 has 1 peer, tip3 has 1 peer.
	assert.Equal(1, len(node1_peerTips[tip1.Hash]))
	assert.Equal(1, len(node1_peerTips[tip3.Hash]))

	// (2b) Node 2 should have 2 nodes with 2 different tips.
	assert.Equal(2, len(node2_peerTips)) // two entries.
	// tip1 and tip3 are the entries.
	assert.Contains(node2_peerTips, tip1.Hash)
	assert.Contains(node2_peerTips, tip3.Hash)
	// tip1 has 1 peer, tip3 has 1 peer.
	assert.Equal(1, len(node2_peerTips[tip1.Hash]))
	assert.Equal(1, len(node2_peerTips[tip3.Hash]))
}

func TestSyncScheduleDownloadWork(t *testing.T) {
	// After getting the tips, then we need to divide them into work units.

	// Basically speaking:
	// - for one set of tips
	// -- for each tip
	// --- divide the work into chunks: work / chunk_size
	// --- setup peer worker threads
	// --- distribute work to peer workers
	// wait for download to complete.

	// to modify the dumbtorrent code to do this:
	// - workItem should basically be the call arguments to the SyncGetData function
	// - peer is the same NetPeer, though DoWork calls ourpeer.PeerSyncGetData(peer, args)
	// - the result type is something like SyncGetDataReply.

}
