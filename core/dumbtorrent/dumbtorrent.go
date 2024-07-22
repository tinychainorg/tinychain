package dumbtorrent

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
)

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

type workItemInfo struct {
	i    int
	item int64
}

func (i workItemInfo) String() string {
	return fmt.Sprintf("%d", i.item)
}

type workItemLog struct {
	item      *workItemInfo
	err       error
	startTime time.Time
	endTime   time.Time
}

type DumbBittorrentEngine struct {
	done    chan bool
	results map[int64][]byte
	err     error

	newWorkersChan chan *Peer
	peers          map[*Peer]bool
}

// A simple BitTorrent-like protocol in Go.
//
// dumbBitTorrent completes a set of work items using a worker pool, where each worker works on one item at a time. If a worker fails, the work item is reingested into the work queue and picked up by another worker. If there are no more workers to fill items, the function returns early with an error.
//
// The function returns a map of work item IDs to their results.
//
// The function also prints a summary of each worker's performance, including the number of jobs done, failed, and the average duration of each job.
func NewDumbBittorrentEngine() *DumbBittorrentEngine {
	return &DumbBittorrentEngine{
		done:           make(chan bool),
		results:        make(map[int64][]byte),
		newWorkersChan: make(chan *Peer),
		peers:          make(map[*Peer]bool),
	}
}

func (e *DumbBittorrentEngine) Wait() (map[int64][]byte, error) {
	<-e.done
	return e.results, e.err
}

func (e *DumbBittorrentEngine) AddWorker(peer *Peer) {
	if _, ok := e.peers[peer]; ok {
		fmt.Printf("skipping peer, already have them")
		return
	}

	e.peers[peer] = true
	e.newWorkersChan <- peer
}

func (e *DumbBittorrentEngine) Start(workItems []int64, initialWorkers []*Peer) {
	workerLogs := make(map[*Peer]*[]workItemLog)
	results := make(map[int64][]byte)
	workers := []*Peer{}

	var resultsMutex sync.Mutex
	var pendingWork sync.WaitGroup
	var onlineWorkers sync.WaitGroup

	dlog.Printf("starting download with %d peers", len(workers))
	dlog.Printf("downloading %d items", len(workItems))

	workItemsChan := make(chan workItemInfo, len(workItems))
	defer close(workItemsChan)
	for i, item := range workItems {
		workItemsChan <- workItemInfo{i, item}
		pendingWork.Add(1)
	}

	// Setup worker threads.
	// Start peer worker threads, which take work from the workItems channel.
	workerThread := func(peer *Peer) {
		onlineWorkers.Add(1)

		logs := []workItemLog{}
		workerLogs[peer] = &logs

		go func() {
			defer onlineWorkers.Done() // will get called even if peer.DoWork panics
			for {
				// Select work item.
				workItem, more := <-workItemsChan
				if !more {
					// work channel closed.
					return
				}

				startTime := time.Now()

				dlog.Printf("downloading work %d from peer %s", workItem.i, peer)

				// 2. Call peer.
				res, err := peer.DoWork(workItem.item)

				// Log the work item.
				itemLog := workItemLog{}
				itemLog.startTime = startTime
				itemLog.item = &workItem
				itemLog.endTime = time.Now()
				itemLog.err = err
				logs = append(logs, itemLog)

				// 2a. Handle error.
				if err != nil {
					dlog.Printf("downloading work %d from peer %s - peer failed", workItem.i, peer)

					// Re-insert into work items channel.
					workItemsChan <- workItem

					// Exit from worker pool.
					return
				}

				// 2b. Handle success.
				// Set result.
				resultsMutex.Lock()
				results[int64(workItem.item)] = res
				resultsMutex.Unlock()

				dlog.Printf("downloading work %d done", workItem.i)

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
		dlog.Printf("all work items done")
		break
	case <-workersDone:
		dlog.Printf("error: not enough workers to fill jobs")
		err = fmt.Errorf("not enough workers to fill jobs")
		break
	}

	// Print the status overview of each peer's logs.
	dlog.Printf("Peer summary table\n")
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

		dlog.Printf("Peer #%d: jobs=%d success=%d failed=%d avg_duration=%s rate_per_s=%.2f\n", i, len(*worklogs), jobs_done, jobs_failed, time.Duration(avg_duration), ratePerSecond)
	}

	close(e.newWorkersChan)
	close(newWorkerDoneChan)

	// Set results.
	e.results = results
	e.err = err
	e.done <- true
}

// How to add and remove workers dynamically? x
// How to add and remove work items dynamically?
// How to distribute work items based on custom scoring for workers?
