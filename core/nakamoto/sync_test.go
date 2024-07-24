package nakamoto

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/liamzebedee/tinychain-go/core"
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

// Print all tips
func logPeerTips(t *testing.T, label string, tips map[[32]byte][]Peer) {
	t.Logf("%s peer tips:", label)
	for hash, peers := range tips {
		peersStr := ""
		for _, peer := range peers {
			peersStr += peer.String() + " "
		}
		t.Logf("%x: %s", hash, peersStr)
	}
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

	logPeerTips(t, "Node 1", node1_peerTips)
	logPeerTips(t, "Node 2", node2_peerTips)
	logPeerTips(t, "Node 3", node3_peerTips)

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

func TestSyncScheduleDownloadWork1(t *testing.T) {
	// After getting the tips, then we need to divide them into work units.

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

	// Now disable block mining on node 3.
	node3.Peer.OnNewBlock = nil

	// Node 1 mines 15 blocks, gossips with node 2
	node1.Miner.Start(15)

	// Wait for nodes [1,2] to sync.
	err := waitForNodesToSyncSameTip([]*Node{node1, node2})
	assert.Nil(err)

	// Print the entire hash chain according to node1.
	hashlist, err := node1.Dag.GetLongestChainHashList(node1.Dag.FullTip.Hash, node1.Dag.FullTip.Height+10)
	if err != nil {
		t.Fatalf("Error getting longest chain: %s", err)
	}
	t.Logf("")
	t.Logf("Longest chain according to node 1:")
	for i, hash := range hashlist {
		t.Logf("Block #%d: %x", i+1, hash)
	}
	t.Logf("")

	// Wait some time for other goroutines to process.
	time.Sleep(400 * time.Millisecond)

	// Basically speaking:
	// - for one set of tips
	// -- for each tip
	// --- divide the work into chunks: work / chunk_size
	// --- setup peer worker threads
	// --- distribute work to peer workers
	// wait for download to complete.

	// Get the latest tips.
	tip1 = node1.Dag.FullTip
	tip2 = node2.Dag.FullTip
	tip3 = node3.Dag.FullTip

	// Now we commence syncing for node 3 from nodes [1,2].
	node3_peerTips, _ := node3.getPeerTips(tip3.Hash, uint64(20), 1)
	logPeerTips(t, "Node 3", node3_peerTips)
	// 000292441f3c7a39f44cfaf092bfb6f4399256b50bf23aa27da22da067b937a1

	missingTip := tip1.Hash
	// print missing tip
	t.Logf("Missing tip: %x", missingTip)
	assert.Equal(2, len(node3_peerTips[missingTip]))

	// Setup work items.
	heights1 := core.NewBitset(2048)
	heights1.Insert(0)
	heights1.Insert(1)
	heights2 := core.NewBitset(2048)
	heights2.Insert(2)
	heights2.Insert(3)
	heights3 := core.NewBitset(2048)
	heights3.Insert(4)
	heights3.Insert(5)

	workItems := []DownloadWorkItem{
		{
			Type:      "sync_get_data",
			FromBlock: tip3.Hash,
			Heights:   *heights1,
			Headers:   true,
			Bodies:    false,
		},
		{
			Type:      "sync_get_data",
			FromBlock: tip3.Hash,
			Heights:   *heights2,
			Headers:   true,
			Bodies:    false,
		},
		{
			Type:      "sync_get_data",
			FromBlock: tip3.Hash,
			Heights:   *heights3,
			Headers:   true,
			Bodies:    false,
		},
	}
	// Print each work item.
	for i, item := range workItems {
		t.Logf("Work item #%d: heights=%d", i, item.Heights.Count())
	}

	node3_peers := node3.Peer.GetPeers()
	dlPeers := []DownloadPeer{
		downloadPeerImpl{node3.Peer, &node3_peers[0]},
		downloadPeerImpl{node3.Peer, &node3_peers[1]},
	}

	torrent := NewDownloadEngine()
	go torrent.Start(workItems, dlPeers)
	results, err := torrent.Wait()
	if err != nil {
		t.Errorf("Error downloading: %s", err)
	}
	for i, result := range results {
		for j, header := range result.Headers {
			t.Logf("Header #%d-%d: %x (parent %x, nonce %x)", (i + 1), (j + 1), header.BlockHash(), header.ParentHash, header.Nonce)
		}
	}

	// Verify the headers we get back are the right ones.

	// Now we need to order the headers.
	all_headers := []BlockHeader{}
	for _, result := range results {
		all_headers = append(all_headers, result.Headers...)
	}
	headers2 := orderValidateHeaders(tip3.Hash, all_headers)

	// Now print header chain.
	t.Logf("Ordering headers...")
	for i, header := range headers2 {
		t.Logf("Header #%d: %x", i+1, header.BlockHash())
	}

	// Now ingest headers.
	for _, header := range headers2 {
		err := node3.Dag.IngestHeader(header)
		if err != nil {
			t.Errorf("Error ingesting header: %s", err)
		}
	}

	// Now get the new headers tip.
	tip3 = node3.Dag.HeadersTip
	t.Logf("New tip height: %x", tip3.Height)
	t.Logf("New tip: %x", tip3.Hash)

	// Now repeat the same for block bodies.
	for i, _ := range workItems {
		workItems[i].Headers = false
		workItems[i].Bodies = true
	}
	torrent = NewDownloadEngine()
	go torrent.Start(workItems, dlPeers)
	results, err = torrent.Wait()
	if err != nil {
		t.Errorf("Error downloading: %s", err)
	}
	for i, result := range results {
		for j, body := range result.Bodies {
			merkleRoot := GetMerkleRootForTxs(body)
			t.Logf("Body #%d-%d: %d (merkle_root=%x)", (i + 1), (j + 1), len(body), merkleRoot)
		}
	}

	// We don't need to order bodies.
	// Now ingest bodies.
	for _, result := range results {
		for _, body := range result.Bodies {
			err := node3.Dag.IngestBlockBody(body)
			if err != nil {
				t.Logf("Error ingesting body: %s", err)
			}
		}
	}

	// Now print the new full tip.
	tip3 = node3.Dag.FullTip
	t.Logf("New full tip height: %d", tip3.Height)
	t.Logf("New full tip: %x", tip3.Hash)

}

func TestSyncSyncDownloadDataHeaders(t *testing.T) {
	// After getting the tips, then we need to divide them into work units.

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

	// Now disable block mining on node 3.
	node3.Peer.OnNewBlock = nil

	// Node 1 mines 15 blocks, gossips with node 2
	node1.Miner.Start(15)

	// Wait for nodes [1,2] to sync.
	err := waitForNodesToSyncSameTip([]*Node{node1, node2})
	assert.Nil(err)

	// Print the entire hash chain according to node1.
	hashlist, err := node1.Dag.GetLongestChainHashList(node1.Dag.FullTip.Hash, node1.Dag.FullTip.Height+10)
	if err != nil {
		t.Fatalf("Error getting longest chain: %s", err)
	}
	t.Logf("")
	t.Logf("Longest chain according to node 1:")
	for i, hash := range hashlist {
		t.Logf("Block #%d: %x", i+1, hash)
	}
	t.Logf("")

	// Wait some time for other goroutines to process.
	time.Sleep(400 * time.Millisecond)

	// Basically speaking:
	// - for one set of tips
	// -- for each tip
	// --- divide the work into chunks: work / chunk_size
	// --- setup peer worker threads
	// --- distribute work to peer workers
	// wait for download to complete.

	// Get the latest tips.
	tip1 = node1.Dag.FullTip
	tip2 = node2.Dag.FullTip
	tip3 = node3.Dag.FullTip

	// Now we commence syncing for node 3 from nodes [1,2].
	node3_peerTips, _ := node3.getPeerTips(tip3.Hash, uint64(20), 1)
	logPeerTips(t, "Node 3", node3_peerTips)
	// 000292441f3c7a39f44cfaf092bfb6f4399256b50bf23aa27da22da067b937a1

	missingTip := tip1.Hash
	// print missing tip
	t.Logf("Missing tip: %x", missingTip)
	assert.Equal(2, len(node3_peerTips[missingTip]))

	heights := core.NewBitset(2048)
	for i := 0; i < 20; i++ {
		heights.Insert(i)
	}

	headers, _, err := node3.SyncDownloadData(node3.Dag.HeadersTip.Hash, *heights, node3.Peer.GetPeers(), true, false)
	if err != nil {
		t.Errorf("Error downloading headers: %s", err)
	}

	headers2 := orderValidateHeaders(node3.Dag.HeadersTip.Hash, headers)
	t.Logf("Ordered headers length: %d", len(headers2))
	for _, header := range headers2 {
		err := node3.Dag.IngestHeader(header)
		if err != nil {
			t.Errorf("Error ingesting header: %s", err)
		}
	}

	// Now get the new headers tip.
	tip3 = node3.Dag.HeadersTip
	t.Logf("New tip height: %d", tip3.Height)
	t.Logf("New tip: %x", tip3.Hash)

	assert.Equal(node3.Dag.HeadersTip.HashStr(), node1.Dag.HeadersTip.HashStr())
}
