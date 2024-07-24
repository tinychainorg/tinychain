package nakamoto

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/liamzebedee/tinychain-go/core"
)

// downloadPeerImpl is a wrapper around a Peer that implements the DownloadPeer interface.
type downloadPeerImpl struct {
	ourpeer *PeerCore
	peer    *Peer
}

func (d downloadPeerImpl) String() string {
	return d.peer.String()
}

// DownloadPeerImpl performs one type of work: SyncGetBlockData.
func (d downloadPeerImpl) Work(item DownloadWorkItem) (DownloadWorkResult, error) {
	return d.ourpeer.SyncGetBlockData(*d.peer, item.FromBlock, item.Heights, item.Headers, item.Bodies)
}

// Downloads block headers/bodies in parallel from a set of peers for a set of heights, relative to a base blockhash and height.
//
// The total number of headers we are downloading is represented by the count of items inside the heightMap.
// The header size is estimated as 200 B. So for 2048 headers, we are downloading 409 KB.
// The total download is then divided into a fixed-size workload of 50 KB each, which we call chunks.
// These download chunk work items are then distributed to the peers in a round-robin fashion (ie. modulo).
//
// This function supports downloading as few as 1 header, which will download from a single peer, or 2048 headers, which
// will download from as many as 9 peers in parallel.
func (n *Node) SyncDownloadData(fromNode [32]byte, heightMap core.Bitset, peers []Peer, getHeaders bool, getBodies bool) ([]BlockHeader, []BlockBody, error) {
	// Size of a block header is 200 B.
	HEADER_SIZE := 200

	// Size of a chunk we request from a peer is 50 KB.
	CHUNK_SIZE := 50 * 1000

	// Total number of headers we're requesting.
	NUM_HEADERS := heightMap.Count()

	// Total number of chunks (download work items).
	NUM_CHUNKS := (NUM_HEADERS * HEADER_SIZE / CHUNK_SIZE) + 1

	// So for example:
	// header_size = 200 B
	// chunk_size = 50 KB
	// ...
	// num_headers = 100
	// num_chunks = (100 * 200 / 50,000) + 1 = 1 chunks
	// ...
	// num_headers = 2048
	// num_chunks = (2048 * 200 / 50,000) + 1 = 9 chunks

	// Then we distribute these work items to our peers.
	workItems := make([]DownloadWorkItem, NUM_CHUNKS)

	// Distribute the work:
	// ...
	// num_headers = 2048
	// num_chunks = (2048 * 200 / 50,000) + 1 = 9 chunks
	// work:
	// 1: heights 0..79
	// 2: heights 80..159
	// N: heights
	// i * NUM_HEADERS / NUM_CHUNKS = i * 2048 / 9 = i * 227
	// i*227 = 0, 227, 454, 681, 908, 1135, 1362, 1589, 1816
	for i := 0; i < NUM_CHUNKS; i++ {
		startHeight := i * (NUM_HEADERS / NUM_CHUNKS)
		endHeight := (i + 1) * (NUM_HEADERS / NUM_CHUNKS)
		heights := core.NewBitset(heightMap.Size())
		for j := startHeight; j < endHeight; j++ {
			heights.Insert(j)
		}

		workItems[i] = DownloadWorkItem{
			Type:      "sync_get_data",
			FromBlock: fromNode,
			Heights:   *heights,
			Headers:   true,
			Bodies:    false,
		}

		// print work item
		// n.syncLog.Printf("Work item %d: %v\n", i, workItems[i])
	}

	// Distribute the work items to our peers.
	dlPeers := []DownloadPeer{}
	for _, peer := range peers {
		dlPeers = append(dlPeers, downloadPeerImpl{n.Peer, &peer})
	}

	torrent := NewDownloadEngine()
	go torrent.Start(workItems, dlPeers)
	results, err := torrent.Wait()
	if err != nil {
		return nil, nil, fmt.Errorf("Error downloading: %s", err)
	}

	// Print headers received.
	n.syncLog.Printf("Received %d headers\n", len(results[0].Headers))

	headers := []BlockHeader{}
	for _, result := range results {
		headers = append(headers, result.Headers...)
	}
	bodies := []BlockBody{}
	for _, result := range results {
		bodies = append(bodies, result.Bodies...)
	}

	return headers, bodies, nil
}

// sync_get_tip_at_depth
type SyncGetTipAtDepthMessage struct {
	Type      string   `json:"type"`
	FromBlock [32]byte `json:"fromBlock"`
	Depth     uint64   `json:"depth"`
	Direction int      `json:"direction"`
}

type SyncGetTipAtDepthReply struct {
	Type string   `json:"type"`
	Tip  [32]byte `json:"tip"`
}

// sync_get_data
type SyncGetBlockDataMessage struct {
	Type      string      `json:"type"`
	FromBlock [32]byte    `json:"fromBlock"`
	Heights   core.Bitset `json:"heights"`
	Headers   bool        `json:"headers"`
	Bodies    bool        `json:"bodies"`
}

type SyncGetBlockDataReply struct {
	Type    string             `json:"type"`
	Headers []BlockHeader      `json:"headers"`
	Bodies  [][]RawTransaction `json:"bodies"`
}

// Verify the header chain we have received.
// ie. A -> B -> C ... -> Z
// We should have all the headers from A to Z.
func orderValidateHeaders(root [32]byte, headers []BlockHeader) []BlockHeader {
	// Verify the header chain we have received.
	// ie. A -> B -> C ... -> Z
	// We should have all the headers from A to Z.
	base := root
	chain := make([]BlockHeader, 0)

	// Build cache of next pointers.
	nextRefs := make(map[[32]byte]int)
	for i, header := range headers {
		nextRefs[header.ParentHash] = i
	}

	// While we have a child, append to the chain.
	for {
		if next, ok := nextRefs[base]; ok {
			node := headers[next]
			chain = append(chain, node)
			base = node.BlockHash()
		} else {
			break
		}
	}

	return chain
}

func (n *Node) getPeerTips(baseBlock [32]byte, depth uint64, dir int) (map[[32]byte][]Peer, error) {
	// NOTE: we only request their tip hash in order to bucket them.
	peersTips := make(map[[32]byte][]Peer)

	for _, peer := range n.Peer.peers {
		tip, err := n.Peer.SyncGetTipAtDepth(peer, baseBlock, depth, dir)
		if err != nil {
			// Skip. Peer will not be used for downloading.
			continue
		}
		// check if slice exists
		if _, ok := peersTips[tip]; !ok {
			peersTips[tip] = make([]Peer, 0)
		}
		peersTips[tip] = append(peersTips[tip], peer)
	}

	return peersTips, nil
}

// Syncs the node with the network.
//
// The blockchain sync algorithm is the most complex part of the system. The Nakamoto blockchain is defined simply as a linked list of blocks, where the canonical chain is the one with the most amount of work done on it. A blockchain network is composed of peers who disseminate blocks and transactions, and take turns in being the leader to mine a new block.
// Due to the properties of the P2P network, namely asynchronicity, network partitions, and latency, it is possible for nodes to have different views of the blockchain. Thus in practice, in order to converge on the canonical chain, blockchain nodes must keep track of the block tree (a type of DAG), where there are multiple differing branches.
//
// Synchronisation is the process of downloading the block tree from our peers, until our local tip matches the remote tip of the heaviest chain. At its core, the sync algorithm is a greedy iterative search, where we continue downloading block headers from all peers until we reach their tip (a complete view of the network's state).
//
// The sync algorithm traverses the block DAG in windows of 2048 blocks. At each iteration, it asks each of its peers for their tip at height N+2048, buckets them by tip hash, and downloads block headers in parallel from peers who share a mutual tip. After validating block headers, it chooses the heaviest tip and downloads block bodies in parallel, validates and ingests them. The algorithm resolves when our local tip matches the heaviest remote tip of our peer's tips.
//
// Parallel downloads are done BitTorrent-style, where we divide the total download work into fixed-size work items of 50KB each, and distribute them to all our peers in a round-robin fashion. So for 2048 block headers at 200 B each, this is 409 KB of download work, divided into 9 chunks of 50 KB each. If our peer set includes 3 peers, then 9/3 = 3 chunks are downloaded from each peer. The parallel download algorithm scales automatically with the number of peers we have and the amount of work to download, so if peers drop out, the algorithm will still continue to download from the remaining peers. The download also represents its download request compactly using a bitstring - a request for 2048 block headers is represented as a bitstring of 2048 bits, where a bit at index i represents a want for a header at height start_height + i. This data format is compact, allowing peers to specify download requests for N blocks in N bits, as opposed to N uint32 integers O(4N), while also remaining flexible - peers can indicate as few as 1 header to download.
//
// The sync algorithm is designed so it can be called at any time.
func (n *Node) Sync() int {
	n.syncLog.Printf("Performing sync...\n")

	// The sync algorithm is a greedy iterative search.
	// We continue downloading block headers from a peer until we reach their tip.

	// TODO handle peers joining.
	WINDOW_SIZE := 2048

	// Greedily searches the block DAG from a tip hash, downloading headers in parallel from peers from all subbranches up to a depth.
	// The depth is referred to as the "window size", and is a constant value of 2048 blocks.
	search := func(currentTipHash [32]byte) int {
		// 1. Get the tips from all our peers and bucket them.
		peersTips, err := n.getPeerTips(currentTipHash, uint64(WINDOW_SIZE), 1)
		if err != nil {
			n.syncLog.Printf("Failed to get peer tips: %s\n", err)
			return 0
		}

		// 2. For each tip, download a window of headers and ingest them.
		downloaded := 0
		for _, peers := range peersTips {
			// 2a. Identify heights.
			heights := core.NewBitset(WINDOW_SIZE)
			for i := 0; i < WINDOW_SIZE; i++ {
				heights.Insert(i)
			}

			// 2b. Download headers.
			headers, _, err := n.SyncDownloadData(currentTipHash, *heights, peers, true, false)
			if err != nil {
				n.syncLog.Printf("Failed to download headers: %s\n", err)
				continue
			}

			// 2c. Validate headers.
			// Sanity-check: verify we have all the headers for the heights in order. TODO.
			headers2 := orderValidateHeaders(currentTipHash, headers)

			// 2d. Ingest headers.
			for _, header := range headers2 {
				err := n.Dag.IngestHeader(header)
				if err != nil {
					// Skip. We will not be able to download the bodies.
					continue
				}

				downloaded += 1
			}

			n.syncLog.Printf("Downloaded %d headers\n", downloaded)
		}

		// 3. Return the number of headers downloaded.
		return downloaded
	}

	currentTip, err := n.Dag.GetLatestHeadersTip()
	if err != nil {
		n.syncLog.Printf("Failed to get latest tip: %s\n", err)
		return 0
	}

	totalSynced := 0

	for {
		// Search for headers from current tip.
		downloaded := search(currentTip.Hash)
		totalSynced += downloaded

		// Exit when there are no more headers to download.
		if downloaded == 0 {
			n.syncLog.Printf("synchronisation done\n")
			break
		}
	}

	n.syncLog.Printf("Total headers downloaded: %d\n", totalSynced)

	return totalSynced
}

// Contacts all our peers in parallel, gets the block header of their tip, and returns the best tip based on total work.
func (n *Node) sync_getBestTipFromPeers() [32]byte {
	syncLog := NewLogger("node", "sync")

	// 1. Contact all our peers.
	// 2. Get their current tips in parallel.
	syncLog.Printf("Getting tips from %d peers...\n", len(n.Peer.peers))

	var wg sync.WaitGroup

	tips := make([]BlockHeader, 0)
	tipsChan := make(chan BlockHeader, len(n.Peer.peers))
	// timeout := time.After(5 * time.Second)

	for _, peer := range n.Peer.peers {
		wg.Add(1)
		go func(peer Peer) {
			defer wg.Done()
			tip, err := n.Peer.GetTip(peer)
			if err != nil {
				syncLog.Printf("Failed to get tip from peer: %s\n", err)
				return
			}
			syncLog.Printf("Got tip from peer: hash=%s\n", tip.BlockHashStr())
			tipsChan <- tip
		}(peer)
	}

	go func() {
		wg.Wait()
		close(tipsChan)
	}()

	// TODO WIP
	// for {
	// 	select {
	// 	case tip, ok := <-tipsChan:
	// 		if !ok {
	// 			break
	// 		}
	// 		tips = append(tips, tip)
	// 	case <-timeout:
	// 		syncLog.Printf("Timed out getting tips from peers\n")
	// 	}
	// }

	syncLog.Printf("Received %d tips\n", len(tips))
	if len(tips) == 0 {
		syncLog.Printf("No tips received. Exiting sync.\n")
		return [32]byte{} // TODO, should return error
	}

	// 3. Sort the tips by max(work).
	// 4. Reduce the tips to (tip, work, num_peers).
	// 5. Choose the tip with the highest work and the most peers mining on it.
	numPeersOnTip := make(map[[32]byte]int)
	tipWork := make(map[[32]byte]*big.Int)

	highestWork := big.NewInt(0)
	bestTipHash := [32]byte{}

	for _, tip := range tips {
		hash := tip.BlockHash()
		// TODO embed difficulty into block header so we can verify POW.
		work := CalculateWork(Bytes32ToBigInt(hash))

		// -1 if x < y
		// highestWork < work
		if highestWork.Cmp(work) == -1 {
			highestWork = work
			bestTipHash = hash
		}

		numPeersOnTip[hash] += 1
		tipWork[hash] = work
	}

	syncLog.Printf("Best tip: %s\n", bestTipHash)
	return bestTipHash
}

// Computes the common ancestor of our local canonical chain and a remote peer's canonical chain through an interactive binary search.
// O(log N * query_size).
func (n *Node) sync_computeCommonAncestorWithPeer(peer Peer, local_chainhashes *[][32]byte) [32]byte {
	syncLog := NewLogger("node", "sync")

	// 6a. Compute the common ancestor (interactive binary search).
	// This is a classical binary search algorithm.
	floor := 0
	ceil := len(*local_chainhashes)
	n_iterations := 0

	for (floor + 1) < ceil {
		guess_idx := (floor + ceil) / 2
		guess_value := (*local_chainhashes)[guess_idx]

		syncLog.Printf("Iteration %d: floor=%d, ceil=%d, guess_idx=%d, guess_value=%x", n_iterations, floor, ceil, guess_idx, guess_value)
		n_iterations += 1

		// Send our tip's blockhash
		// Peer responds with "SEEN" or "NOT SEEN"
		// If "SEEN", we move to the right half.
		// If "NOT SEEN", we move to the left half.
		has, err := n.Peer.HasBlock(peer, guess_value)
		if err != nil {
			syncLog.Printf("Failed to get block from peer: %s\n", err)
			continue
		}
		if has {
			// Move to the right half.
			floor = guess_idx
		} else {
			// Move to the left half.
			ceil = guess_idx
		}
	}

	ancestor := (*local_chainhashes)[floor]
	syncLog.Printf("Common ancestor: %x", ancestor)
	syncLog.Printf("Found in %d iterations.", n_iterations)
	return ancestor
}
