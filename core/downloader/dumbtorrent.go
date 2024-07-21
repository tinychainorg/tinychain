package downloader

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

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

// how do we detect when we have run through all peers?
// is there a facility to add new peers to the pool? yes.
// how can we let programmers modify their worker pool logic on their own?
// can design our own facility ? minimal design, callbacks.
// -

//
// algorithm:
// - create slice of work items
// loop:
//  for each open work item
//    wait on available peer from pool
//    distribute work to peer
//      if peer failed:
//        mark peer as failed
//        return err
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

type Peer struct {
	DoWork func(chunkID int64) (result []byte, err error)
}

// Things to do:
//
// Work - a function called on a peer.
// Peers - a set of peers we have to do work.
// - can be dynamically changing - we add and remove peers
// - if a peer fails, they stop doing work for the pool
// Control loop:
// Get free worker + distribute work item
// Get free work item from workPool
// Send to peer work channel
// On reply -
// if error: reinsert work into workPool
// if success: insert result into results, reinsert worker into workerPool
// Add new peers to work pool if they don't exist

// continue while work is pending
// continue while workers are pending
//

//
// wait until all work items are processed?
// stopping conditions:
// - all work items completed
// - all peers are marked failed
//
//

func dumbBitTorrent(workItems []int64, peers []Peer) (map[int64][]byte, error) {
	workChan := make(chan int64, len(workItems))
	results := make(map[int64][]byte)

	var workersInfoMutex sync.Mutex
	var resultsMutex sync.Mutex
	var pendingWork sync.WaitGroup
	var onlineWorkers sync.WaitGroup

	dlog.Printf("starting download with %d peers", len(peers))
	dlog.Printf("downloading %d items", len(workItems))

	type workerInfo struct {
		working bool
		failed  bool
	}

	workersInfo := make(map[*Peer]*workerInfo)
	for _, peer := range peers {
		workersInfo[&peer] = &workerInfo{
			working: false,
			failed:  false,
		}
	}

	workItemsChan := make(chan int64, len(workItems))
	for _, item := range workItems {
		workItemsChan <- item
	}

	// Take work items and distribute them to peers.
	// Peer worker threads - listen to items, work on them, and return result or error.
	// Wait on all work items done, or all peers done.

	for {
		// 1. Get next work item.
		workItem := <-workItemsChan

		// 2. Distribute to peer.

		// Select an available peer.
		var available *Peer
		totalPeers := len(peers)
		failed := 0
		working := 0

		workersInfoMutex.Lock()
		for peer, worker := range workersInfo {
			if worker.failed {
				failed += 1
				continue
			}
			if worker.working {
				working += 1
				continue
			}
			available = peer
		}
		workersInfoMutex.Unlock()

		if totalPeers == working {
			dlog.Printf("all peers currently working, waiting")
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if totalPeers == failed {
			dlog.Printf("failed: tried all peers, none left")
		}

		// Try the available peer.
		go func(peer *Peer, workItem int64) {
			// 1. Set working.
			workersInfoMutex.Lock()
			workersInfo[peer].working = true
			workersInfoMutex.Unlock()

			// 2. Call peer.
			res, err := peer.DoWork(workItem)

			// 2a. Handle error.
			if err != nil {
				// Set failed, working=false.
				workersInfoMutex.Lock()
				workersInfo[peer].working = false
				workersInfo[peer].failed = true
				workersInfoMutex.Unlock()

				// Re-insert into work items channel.
				workItemsChan <- workItem
				return
			}

			// 2b. Handle success.
			// Set result.
			resultsMutex.Lock()
			results[workItem] = res
			resultsMutex.Unlock()

			// Set working=false.
			workersInfoMutex.Lock()
			workersInfo[peer].working = false
			workersInfoMutex.Unlock()
		}(available, workItem)
	}

	// Close work channel when all items are processed
	go func() {
		pendingWork.Wait()
		close(workChan)
	}()

	// Close workers channel when all workers are exited (success or failure).
	go func() {
		onlineWorkers.Wait()
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
