Testing
=======

Building a blockchain is a complex endeavor. Testing the components we build can be even more complex. Thankfully, tinychain is proving relatively straightforward to test. Why is that? I think it be due to the design. When software components (functions, structs, etc) are decoupled, and are mainly composed of pure functions and one-way flows of data, it is easier to write tests for isolated functionality.

## Helpers.

There are multiple test helpers defined throughout the test files. This document will quickly become out-of-date, but for the time being, I thought I would write them down so we can look at them from a birds-eye view:

 - `newBlockdag`. Sets up an in-memory SQL database, and a block DAG for testing using a consensus configuration suitable for testing.
 - `newBlockdagForMiner`. Similar to `newBlockdag`, modified to suit mining tests.
 - `getTestingWallets`. Generates and gets a set of wallets for testing.
 - `newValidTx`. Creates a new transaction and signs it using `wallets[0]` of the testing wallets.
 - `saveDbForInspection`. Saves the in-memory SQL database to a file, for inspection.
 - `healthCheck`. Checks a TCP server is running or not.
 - `getRandomPort`. Gets a random TCP port that is free to bind to and listen on.
 - `waitForPeersOnline`. Waits until all peers have begun listening and serving RPC's.
 - `newNodeFromConfig`. Creates a new `Node`, consisting of a blockDAG, miner and peer, with its the RPC server listening on a random port.
 - `fullyConnectPeers`. Given a set of peers, builds a fully connected network, where every peer is connected to every other peer.
 - `waitForNodesToSyncSameTip`. Waits for a set of nodes to synchronise to the same block tip, by polling their tip regularly, or else timing out.
 - `setupTestNetwork`. Sets up a fully-connected network of peers.
 - `logPeerTips`. Log a set of peer tips.

## Callbacks / events.

### Decoupling.

Callbacks are used extensively in tinychain's design and are used to decouple components. An example of this is the `Node`:

```go
func (n *Node) setup() {
	// Listen for new blocks.
	n.Peer.OnNewBlock = func(b RawBlock) {
		n.log.Printf("New block gossip from peer: block=%s\n", b.HashStr())

		if n.Dag.HasBlock(b.Hash()) {
			n.log.Printf("Block already in DAG: block=%s\n", b.HashStr())
			return
		}

		isUnknownParent := n.Dag.HasBlock(b.ParentHash)
		if isUnknownParent {
			// We need to sync the chain.
			n.log.Printf("Block parent unknown: block=%s\n", b.HashStr())
		}

		// Ingest the block.
		err := n.Dag.IngestBlock(b)
		if err != nil {
			n.log.Printf("Failed to ingest block from peer: %s\n", err)
		}
	}
```

When the peer RPC server receives a `new_block` message, it will trigger the `Peer.OnNewBlock` callback if it is set.

This is an example of designing things with a one-way flow of state. Rather than passing the Node into the peer RPC server, we simply emit events from the server which the peer can listen on. Notably, if there is no callback defined, the peer will simply ignore it. This makes it simple to test the peer and the node separately from each other.

Callbacks are basically an event emitter-subscriber pattern, only for the base case of `n=1` subscribers. There may be a time where we need `n>1` subscribers, but until then, this is what we need.

### Disabling callbacks to test functionality.

Callbacks are sometimes disabled in tests. For example, in this test of blockchain synchronisation:

```go
	// Now let node3 mine some blocks, disable sync to other nodes.
	time.Sleep(400 * time.Millisecond)
	node1.Peer.OnNewBlock = nil
	node2.Peer.OnNewBlock = nil
	node3.Peer.OnNewBlock = nil
	node3.Miner.Start(2)
```

This test creates a system state where node3 has mined 2 blocks on the current tip, and node1 and node2 are unaware of this tip. It does this by disabling their event handlers for new blocks. 

## Simulating networks locally.

It might seem gargantuan to test a distributed network, but it is just another piece of software.

There are a couple things to keep in mind while testing:

 - **The network is executing in parallel**. There is a distinction here between parallelism and concurrency. Concurrency means there are multiple processes running, but parallelism means they are running simutaneously. A network is by design, a parallel system as each node is its own CPU which can run at the same time as other CPU's. As such, we need to design our simulation with parallelism as a key property.
   
   While simulating a network locally, we can really only "imitate" parallelism using the number of CPU cores we have. As such, the networks here are designed to be networks of 4 nodes, corresponding to 4 CPU's which is relatively standard on PC's nowadays.

   Tinychain is implemented in Go, which means we use goroutines for parallelism/concurrency. We have to manually implement checks to wait for nodes to resolve to the state we are testing for; since `n(goroutines) > n(cpu's)`, we never have "true parallelism" as there are always multiple goroutines on a single CPU. However, for testing purposes, we can achieve something approximately similar to the theoretical desired design. 

 - **Bind to localhost**. When we create a test network, we create a set of nodes which each listen on a random open system port. It's important that the node listens on the `localhost` or `127.0.0.1` hostname - this is the lowest permission host. If we listen on `0.0.0.0` on macOS, then it can trigger permissions popups (for fuck's sake, I wish Steve was still here).


There are multiple helpers designed to construct test networks:

 - `getRandomPort`
 - `fullyConnectPeers`
 - `healthcheck` and `waitForPeersOnline`
 - `waitForNodesToSyncSameTip`

