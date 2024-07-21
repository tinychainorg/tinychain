package downloader

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/fatih/color"
)

// can you implement a simple "dumb bittorrent" in go in a single function or so, with the following features:

// - there is a slice of work items. each work item specifies a chunk id, int64
// - there is a slice of peers, represented by a function call(chunkid) which is a sychronous call which returns a result, err
// - the algorithm should distribute work items to all peers in parallel, and wait until all the results are ready
// - if there is an error, the algorithm should retry the work item on a different peer and mark the current peer as failing
// - the requests sent to peers should be in parallel

// how the algorithm completes:
// - all of the requests have returned results, or failures.
// - for failures, we have retried on other peers.

//
// algorithm:
// - create slice of work items
// loop:
//  for each open work item
//    wait on available peer from pool
//    distribute work to peer
//      if peer failed:
//        mark peer as failed
//        re-add work item back into work pool
//      else if peer success
//        re-add peer into pool
//

type PeerWorkInfo struct {
}

func newLogger(prefix string, prefix2 string) *log.Logger {
	prefixFull := color.HiGreenString(fmt.Sprintf("[%s] ", prefix))
	if prefix2 != "" {
		prefixFull += color.HiYellowString(fmt.Sprintf("(%s) ", prefix2))
	}
	return log.New(os.Stdout, prefixFull, log.Ldate|log.Ltime|log.Lmsgprefix)
}

var dlog = newLogger("downloader", "")

type Peer func(chunkID int64) (result []byte, err error)

func dumbBitTorrent(workItems []int64, peers []Peer) (map[int64][]byte, error) {
	results := make(map[int64][]byte)
	var resultsMutex sync.Mutex
	var wg sync.WaitGroup

	workChan := make(chan int64, len(workItems))

	// Fill work channel
	for _, item := range workItems {
		workChan <- item
	}

	dlog.Printf("starting download with %d peers", len(peers))
	dlog.Printf("downloading %d items", len(workItems))

	// Start worker goroutines
	for i := range peers {
		wg.Add(1)
		go func(peerIndex int) {
			defer wg.Done()
			for chunkID := range workChan {
				dlog.Printf("downloading chunk %d from peer %d", chunkID, peerIndex)

				result, err := peers[peerIndex](chunkID)
				if err != nil {
					// Retry with another peer
					workChan <- chunkID
					continue
				}
				resultsMutex.Lock()
				results[chunkID] = result
				resultsMutex.Unlock()
			}
		}(i)
	}

	// Close work channel when all items are processed
	go func() {
		wg.Wait()
		close(workChan)
	}()

	// Wait for all work to be done or an error to occur
	select {
	case <-workChan:
		// All work is done
		return results, nil
	case err := <-errChan:
		return nil, err
	}
}
