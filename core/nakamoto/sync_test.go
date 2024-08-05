package nakamoto

import (
	"context"
	"fmt"
	"math"
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

func TestSyncSync(t *testing.T) {
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

	// Now we sync the node.
	//

	missingTip := node1.Dag.FullTip.Hash
	t.Logf("Missing tip: %x", missingTip)

	downloaded1 := node3.Sync()
	assert.Equal(node3.Dag.HeadersTip.HashStr(), node1.Dag.HeadersTip.HashStr())
	assertIntEqual(t, 15, downloaded1)
	downloaded2 := node3.Sync()
	assertIntEqual(t, 0, downloaded2)
	downloaded3 := node3.Sync()
	assertIntEqual(t, 0, downloaded3)
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

func TestGetPeerCommonAncestor(t *testing.T) {
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

	// Peer mock.
	peer1 := NewPeerCore(PeerConfig{ipAddress: "127.0.0.1", port: getRandomPort()})
	peer2 := NewPeerCore(PeerConfig{ipAddress: "127.0.0.1", port: getRandomPort()})

	go peer1.Start()
	go peer2.Start()

	// Wait for peers online.
	waitForPeersOnline([]*PeerCore{peer1, peer2})

	// Bootstrap.
	peer1.Bootstrap([]string{peer2.GetLocalAddr()})

	peer2.OnHasBlock = func(msg HasBlockMessage) (bool, error) {
		blockhash := msg.BlockHash
		for _, x := range remote_chainhashes {
			if x == blockhash {
				return true, nil
			}
		}
		return false, nil
	}

	//
	// Find the common ancestor.
	//

	remotePeer := peer1.GetPeers()[0]
	ancestor, n_iterations, err := GetPeerCommonAncestor(peer1, remotePeer, &local_chainhashes)
	if err != nil {
		t.Fatalf("Error finding common ancestor: %s", err)
	}
	t.Logf("Common ancestor: %x", ancestor)
	t.Logf("Found in %d iterations.", n_iterations)

	expectedIterations := math.Ceil(math.Log2(float64(len(local_chainhashes))))
	t.Logf("Expected iterations: %f", expectedIterations)

	// Now we assert the common ancestor.
	assert := assert.New(t)
	assert.Equal(local_chainhashes[89], ancestor)
	assertIntEqual(t, int(expectedIterations), n_iterations)
}



// Sync scenarios:
// 
// SCENARIO 1
// =========== 
// DESCRIPTION: local tip is behind remote tip, same branch. remote tip is heavier.
// NETWORK STATE:
// node1: a -> b -> c -> d -> e            (work=100)
// node2: a -> b -> c -> d -> e -> f -> g  (work=150)
// 
// SCENARIO 2
// =========== 
// DESCRIPTION: local tip is behind remote tip, fork branch. remote branch is heavier.
// NETWORK STATE:
// node1: a -> b -> ca -> da -> ea         (work=100)
// node2: a -> b -> cb -> db -> eb         (work=105)
// 
// SCENARIO 3
// =========== 
// DESCRIPTION: local tip is behind remote tip, fork branch. local branch is heavier.
// NETWORK STATE:
// node1: a -> b -> ca -> da -> ea         (work=105)
// node2: a -> b -> cb -> db -> eb         (work=100)
// 


func printBlockchainView(t *testing.T, label string, dag *BlockDAG) {
	// Print the entire hash chain according to node1.
	hashlist, err := dag.GetLongestChainHashList(dag.FullTip.Hash, dag.FullTip.Height+10)
	if err != nil {
		t.Fatalf("Error getting longest chain: %s", err)
	}
	t.Logf("")
	t.Logf("Longest chain (%s):", label)
	for i, hash := range hashlist {
		t.Logf("Block #%d: %x", i, hash)
	}
	t.Logf("")
}

func TestSyncRemoteForkBranchRemoteHeavier(t *testing.T) {
	assert := assert.New(t)
	peers := setupTestNetwork(t)

	node1 := peers[0]
	node2 := peers[1]

	// Then we check the tips.
	tip1 := node1.Dag.FullTip
	tip2 := node2.Dag.FullTip

	// Print the height of the tip.
	t.Logf("Tip 1 height: %d", tip1.Height)
	t.Logf("Tip 2 height: %d", tip2.Height)

	// Check that the tips are the same.
	assert.Equal(tip1.HashStr(), tip2.HashStr())

	// Node 1 mines 15 blocks, gossips with node 2
	node1.Miner.Start(15)

	// Wait for nodes [1,2] to sync.
	err := waitForNodesToSyncSameTip([]*Node{node1, node2})
	assert.Nil(err)

	// Disable nodes syncing.
	node1.Peer.OnNewBlock = nil
	node2.Peer.OnNewBlock = nil
	
	// Node 1 mines 5 blocks on alternative chain.
	// Node 2 mines 7 blocks on alternative chain.
	node1.Miner.Start(1)
	node2.Miner.Start(5)

	// Assert state.
	tip1 = node1.Dag.FullTip
	tip2 = node2.Dag.FullTip
	assertIntEqual(t, 16, tip1.Height)
	assertIntEqual(t, 20, tip2.Height)
	assert.NotEqual(tip1.HashStr(), tip2.HashStr())

	// Now print both hash chains.
	printBlockchainView(t, "Node 1", node1.Dag)
	printBlockchainView(t, "Node 2", node2.Dag)

	// Now sync node 2 to node 1.
	// Get the heavier tip.
	// nodes := []*Node{node1, node2}
	tips := []Block{tip1, tip2}
	var heavierTipIndex int = -1
	if tips[0].AccumulatedWork.Cmp(&tips[1].AccumulatedWork) == -1 {
		heavierTipIndex = 1
	} else if tips[1].AccumulatedWork.Cmp(&tips[0].AccumulatedWork) == -1 {
		heavierTipIndex = 0
	} else if tips[0].AccumulatedWork.Cmp(&tips[1].AccumulatedWork) == 0 {
		t.Errorf("Tips have the same work. Re-run test.")
	}
	t.Logf("Heavier tip index: %d", heavierTipIndex)
	assertIntEqual(t, 1, heavierTipIndex)

	// The common ancestor should be 



}