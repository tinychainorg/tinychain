package nakamoto

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type DownloadWorkItem = SyncGetBlockDataMessage
type DownloadWorkResult = SyncGetBlockDataReply
type DownloadPeer interface {
	String() string
	Work(item DownloadWorkItem) (result DownloadWorkResult, err error)
}

type workItemInfo struct {
	i    int
	item DownloadWorkItem
}

func (i workItemInfo) String() string {
	// Get block heights as a range start-end.
	heights := i.item.Heights
	ranges := heights.Ranges()
	rangesStr := []string{}
	for _, r := range ranges {
		rangesStr = append(rangesStr, fmt.Sprintf("%d-%d", r[0], r[1]))
	}
	idxStr := fmt.Sprintf("%s", rangesStr)

	return fmt.Sprintf("base_block=%x rel_heights=%s headers=%t bodies=%t", i.item.FromBlock[0:8], idxStr, i.item.Headers, i.item.Bodies)
}

type workItemLog struct {
	item      *workItemInfo
	err       error
	startTime time.Time
	endTime   time.Time
}

type DownloadEngine struct {
	done    chan bool
	results map[int]DownloadWorkResult
	err     error

	newWorkersChan chan DownloadPeer
	peers          map[DownloadPeer]bool
	dlog           *log.Logger
}

// A simple BitTorrent-like protocol in Go.
//
// dumbBitTorrent completes a set of work items using a worker pool, where each worker works on one item at a time. If a worker fails, the work item is reingested into the work queue and picked up by another worker. If there are no more workers to fill items, the function returns early with an error.
//
// The function returns a map of work item IDs to their results.
//
// The function also prints a summary of each worker's performance, including the number of jobs done, failed, and the average duration of each job.
func NewDownloadEngine() *DownloadEngine {
	return &DownloadEngine{
		done:           make(chan bool),
		results:        make(map[int]DownloadWorkResult),
		newWorkersChan: make(chan DownloadPeer),
		peers:          make(map[DownloadPeer]bool),
		dlog:           NewLogger("sync", "downloader"),
	}
}

func (e *DownloadEngine) Wait() (map[int]DownloadWorkResult, error) {
	<-e.done
	return e.results, e.err
}

func (e *DownloadEngine) AddWorker(peer DownloadPeer) {
	if _, ok := e.peers[peer]; ok {
		fmt.Printf("skipping peer, already have them")
		return
	}

	e.peers[peer] = true
	e.newWorkersChan <- peer
}

func (e *DownloadEngine) Start(workItems []DownloadWorkItem, initialWorkers []DownloadPeer) {
	workerLogs := make(map[DownloadPeer]*[]workItemLog)
	results := make(map[int]DownloadWorkResult)
	workers := []DownloadPeer{}

	var resultsMutex sync.Mutex
	var pendingWork sync.WaitGroup
	var onlineWorkers sync.WaitGroup

	e.dlog.Printf("starting download with %d peers", len(workers))
	e.dlog.Printf("downloading %d items", len(workItems))

	workItemsChan := make(chan workItemInfo, len(workItems))
	defer close(workItemsChan)
	for i, item := range workItems {
		workItemsChan <- workItemInfo{i, item}
		pendingWork.Add(1)
	}

	// Setup worker threads.
	// Start peer worker threads, which take work from the workItems channel.
	workerThread := func(peer DownloadPeer) {
		onlineWorkers.Add(1)

		logs := []workItemLog{}
		workerLogs[peer] = &logs

		go func() {
			defer onlineWorkers.Done() // will get called even if peer.DoWork panics
			for {
				// Select work item.
				workItemInfo, more := <-workItemsChan
				if !more {
					// work channel closed.
					return
				}

				startTime := time.Now()

				e.dlog.Printf("downloading work %s from peer %s", workItemInfo.String(), peer.String())

				// 2. Call peer.
				res, err := peer.Work(workItemInfo.item)

				// Log the work item.
				itemLog := workItemLog{}
				itemLog.startTime = startTime
				itemLog.item = &workItemInfo
				itemLog.endTime = time.Now()
				itemLog.err = err
				logs = append(logs, itemLog)

				// 2a. Handle error.
				if err != nil {
					e.dlog.Printf("downloading work %d from peer %s - peer failed", workItemInfo.i, peer)

					// Re-insert into work items channel.
					workItemsChan <- workItemInfo

					// Exit from worker pool.
					return
				}

				// 2b. Handle success.
				// Set result.
				resultsMutex.Lock()
				results[workItemInfo.i] = res
				resultsMutex.Unlock()

				e.dlog.Printf("downloading work %d done", workItemInfo.i)

				// Mark work done.
				pendingWork.Done()
			}
		}()
	}

	for _, peer := range initialWorkers {
		workers = append(workers, peer)
		workerThread(peer)
	}

	newWorkerDoneChan := make(chan bool)
	go func() {
		for {
			select {
			case peer := <-e.newWorkersChan:
				workers = append(workers, peer) // TODO dangerous but safe.
				workerThread(peer)
			case <-newWorkerDoneChan:
				return
			}
		}
	}()

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
	var err error
	select {
	case <-workDone:
		e.dlog.Printf("all work items done")
		break
	case <-workersDone:
		e.dlog.Printf("error: not enough workers to fill jobs")
		err = fmt.Errorf("not enough workers to fill jobs")
		break
	}

	// Print the status overview of each peer's logs.
	e.dlog.Printf("Peer summary table\n")
	for i, peer := range workers {
		// Get the log.
		worklogs := workerLogs[peer]

		jobs_done := 0
		jobs_failed := 0
		durations := []time.Duration{}
		ratePerSecond := 0.0

		for _, wlog := range *worklogs {
			if wlog.err != nil {
				jobs_failed += 1
			} else {
				jobs_done += 1
			}

			durations = append(durations, wlog.endTime.Sub(wlog.startTime))
		}

		var total_duration float64
		var avg_duration float64
		for _, x := range durations {
			total_duration += float64(x.Milliseconds())
		}
		if len(durations) > 0 {
			avg_duration = total_duration / float64(len(durations))
		}

		ratePerSecond = float64(jobs_done) / (total_duration / 1000)

		e.dlog.Printf("Peer #%d: jobs=%d success=%d failed=%d avg_duration=%.2f ms rate_per_s=%.2f\n", i, len(*worklogs), jobs_done, jobs_failed, avg_duration, ratePerSecond)
	}

	close(e.newWorkersChan)
	close(newWorkerDoneChan)

	// Set results.
	e.results = results
	e.err = err
	e.done <- true
}
