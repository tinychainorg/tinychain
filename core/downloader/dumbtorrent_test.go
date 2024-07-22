package downloader

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

var peerIdGenerator = 0

func makeMockPeer(fails bool, latency int) *Peer {
	peerid := peerIdGenerator
	peerIdGenerator += 1

	return &Peer{
		id: fmt.Sprintf("%d", peerid),
		DoWork: func(chunkID int64) ([]byte, error) {
			if fails {
				return nil, errors.New("can't get chunk")
			}
			time.Sleep(time.Duration(latency) * time.Millisecond)
			return []byte("data"), nil
		},
	}
}

func TestDumbTorrent(t *testing.T) {
	// workItems := []int64{1, 2, 3, 4, 5}
	workItems := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	peers := []*Peer{
		makeMockPeer(false, 350),

		// makeMockPeer(true, 100),
		// makeMockPeer(true, 100),
		// makeMockPeer(true, 300),

		// makeMockPeer(false, 150),
		// makeMockPeer(false, 250),
	}

	engine := NewDumbBittorrentEngine()
	go engine.Start(workItems, peers)

	// wait 200s.
	// add new peer.
	time.Sleep(800 * time.Millisecond)
	t.Logf("doing stuff")
	go engine.AddWorker(makeMockPeer(false, 300))
	engine.AddWorker(makeMockPeer(false, 300))

	t.Logf("doing stuff 2")

	results, err := engine.Wait()
	if err != nil {
		// Handle error
		panic(err)
	}

	// Print results in order.
	for _, chunkID := range workItems {
		res := results[chunkID]
		t.Logf("Chunk %d: %s", chunkID, string(res))
	}
}
