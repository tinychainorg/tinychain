package nakamoto

import (
	"context"
	"fmt"
	"math/big"
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
	assert.NotEqual(tip3.HashStr(), tip1.HashStr())

	// Now we test the function.
	node1_peerTips, err := node1.getPeerTips(tip1.Hash, uint64(6), 1)
	node2_peerTips, err := node2.getPeerTips(tip1.Hash, uint64(6), 1)
	node3_peerTips, err := node3.getPeerTips(tip1.Hash, uint64(1), 1)

	// Print all tips
	t.Logf("Node 1 peer tips: %v", node1_peerTips)
	t.Logf("Node 2 peer tips: %v", node2_peerTips)
	t.Logf("Node 3 peer tips: %v", node3_peerTips)

	// Test GetPath on node3.

	// Print the full chain hash list of node 3.
	longestChainHashList, err := node3.Dag.GetLongestChainHashList(tip3.Hash, tip3.Height)
	// for i, hash := range longestChainHashList {
	// 	t.Logf("#%d: %x", i+1, hash)
	// }
	// t.Logf("Tip 3 hash: %s", tip3.HashStr())
	// assert.Equal(17, len(longestChainHashList))
	// assert.Equal(tip3.Hash, longestChainHashList[len(longestChainHashList)-1])

	path1, err := node3.Dag.GetPath(tip1.Hash, uint64(3), 1)

	t.Logf("")
	t.Logf("inserting fake block history on alternative branch")

	// Insert a few non descript path entries on the altnerative branch.
	tx, err := node3.Dag.db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %s", err)
	}
	blockhash := tip1.Hash
	tmpblocks := make([][32]byte, 0)
	// "mine" block 1
	tmpaccwork := tip1.AccumulatedWork
	getAccWork := func(i int64) []byte {
		buf := BigIntToBytes32(*tmpaccwork.Add(&tmpaccwork, big.NewInt(i*1000000)))
		return buf[:]
	}
	blockhash[0] += 1
	tmpblocks = append(tmpblocks, tip1.Hash)
	_, err = tx.Exec(
		"INSERT INTO blocks (parent_hash, hash, height, acc_work) VALUES (?, ?, ?, ?)",
		tmpblocks[len(tmpblocks)-1][:],
		blockhash[:],
		tip1.Height+1,
		getAccWork(1),
	)
	if err != nil {
		t.Fatalf("Failed to insert block: %s", err)
	}
	tmpblocks = append(tmpblocks, blockhash)
	// "mine" block 2
	blockhash[0] += 1
	_, err = tx.Exec(
		"INSERT INTO blocks (parent_hash, hash, height, acc_work) VALUES (?, ?, ?, ?)",
		(tmpblocks[len(tmpblocks)-1])[:],
		blockhash[:],
		tip1.Height+2,
		getAccWork(2),
	)
	if err != nil {
		t.Fatalf("Failed to insert block: %s", err)
	}
	tmpblocks = append(tmpblocks, blockhash)
	err = tx.Commit()
	if err != nil {
		t.Fatalf("Failed to commit transaction: %s", err)
	}
	t.Logf("inserted blocks:")
	t.Logf("- %x [fork point]", tmpblocks[0])
	t.Logf("- %x", tmpblocks[1])
	t.Logf("- %x", tmpblocks[2])
	t.Logf("")

	longestChainHashList2, err := node3.Dag.GetLongestChainHashList(node3.Dag.FullTip.Hash, node3.Dag.FullTip.Height)
	for i, hash := range longestChainHashList {
		t.Logf("#%d: %x", i+1, hash)
	}
	t.Log("")
	for i, hash := range longestChainHashList2 {
		t.Logf("#%d: %x", i+1, hash)
	}
	path2, err := node3.Dag.GetPath(tip1.Hash, uint64(2), 1)

	// Path1
	t.Logf("heights(15 - 17) path: %x", path1)

	// Path2
	t.Logf("heights(15 - 17) path: %x", path2)
	assert.Nil(err)

	path3, err := node3.Dag.GetPath(tmpblocks[2], uint64(2), -1)
	assert.Nil(err)
	t.Logf("heights(17_fork - 15) path: %x", path3)

	path4, err := node3.Dag.GetPath(node3.Dag.FullTip.Hash, uint64(2), -1)
	assert.Nil(err)
	t.Logf("heights(17 - 15) path: %x", path4)

	saveDbForInspection(node3.Dag.db)
	// assert.Equal(3, len(path))
}
