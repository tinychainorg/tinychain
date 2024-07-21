package downloader

import (
	"errors"
	"testing"
	"time"
)

func makeMockPeer(fails bool, latency int) Peer {
	return Peer{
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
	workItems := []int64{1, 2, 3, 4, 5}
	peers := []Peer{
		makeMockPeer(false, 100),

		makeMockPeer(true, 100),

		makeMockPeer(false, 100),
		makeMockPeer(false, 100),
	}

	results, err := dumbBitTorrent(workItems, peers)
	if err != nil {
		// Handle error
	}

	// Things to test:
	// download from

	// Use results
	for chunkID, data := range results {
		// Process chunk data
		t.Logf("Chunk %d: %s", chunkID, string(data))
	}
}
