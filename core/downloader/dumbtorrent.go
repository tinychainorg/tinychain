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

func newLogger(prefix string, prefix2 string) *log.Logger {
	prefixFull := color.HiGreenString(fmt.Sprintf("[%s] ", prefix))
	if prefix2 != "" {
		prefixFull += color.HiYellowString(fmt.Sprintf("(%s) ", prefix2))
	}
	return log.New(os.Stdout, prefixFull, log.Ldate|log.Ltime|log.Lmsgprefix)
}

var dlog = newLogger("downloader", "")

type Peer struct {
	id     string
	DoWork func(chunkID int64) (result []byte, err error)
}

func (p Peer) String() string {
	return p.id
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
	results := make(map[int64][]byte)

	var resultsMutex sync.Mutex
	var pendingWork sync.WaitGroup
	var onlineWorkers sync.WaitGroup

	dlog.Printf("starting download with %d peers", len(peers))
	dlog.Printf("downloading %d items", len(workItems))

	workItemsChan := make(chan int64, len(workItems))
	defer close(workItemsChan)
	for _, item := range workItems {
		workItemsChan <- item
		pendingWork.Add(1)
	}

	// Setup worker threads.
	// Start peer worker threads, which take work from the workItems channel.
	for _, peer := range peers {
		onlineWorkers.Add(1)
		go func() {
			defer onlineWorkers.Done() // will get called even if peer.DoWork panics
			for {
				// Select work item.
				workItem, more := <-workItemsChan
				if !more {
					// work channel closed.
					return
				}

				dlog.Printf("downloading work %d from peer %s", workItem, peer)

				// 2. Call peer.
				res, err := peer.DoWork(workItem)

				// 2a. Handle error.
				if err != nil {
					dlog.Printf("downloading work %d from peer %s - peer failed", workItem, peer)

					// Re-insert into work items channel.
					workItemsChan <- workItem

					// Exit from worker pool.
					return
				}

				// 2b. Handle success.
				// Set result.
				resultsMutex.Lock()
				results[workItem] = res
				resultsMutex.Unlock()

				dlog.Printf("downloading work %d done", workItem)

				// Mark work done.
				pendingWork.Done()
			}
		}()
	}

	workDone := make(chan bool)
	workersDone := make(chan bool)

	// Close workDone channel when all items are processed
	go func() {
		pendingWork.Wait()
		close(workDone)
	}()

	// Close workersDone channel when all workers are exited (success or failure).
	go func() {
		onlineWorkers.Wait()
		close(workersDone)
	}()

	// Wait for all work to be done or an error to occur
	select {
	case <-workDone:
		dlog.Printf("all work items done")
		return results, nil
	case <-workersDone:
		dlog.Printf("error: not enough workers to fill jobs")
		return nil, nil
	}
}
