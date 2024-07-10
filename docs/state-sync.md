State sync
==========

https://erigon.substack.com/p/erigon-stage-sync-and-control-flows

When a node receives a block that it doesn't know, it will attempt to resync.
Resyncing is an interesting process:
- We don't know how out of sync the node is already. We can produce some simple estimates though by asking other nodes for their latest tip, and then computing the accumulated work done on this tip added together with the ParentTotalWork.
- We can then compare this with our tip, and if it has less work, we can start syncing from the other node's tip.

The resyncing process inherently needs to traverse a block DAG it has no reference point for. One tool nodes can use to sync very quickly is downloading only block headers.

Significantly though, this does not interact with the blockdag. Since the block DAG calculates the longest chain, which requires calculating difficulty, which requires knowing the parent blocks, it's infeasible to ingest blocks into the DAG unless we know the full ancestry of the block (ie the parent).

Once a peer is aware that it is not mining on the longest chain, it stops its miner since there's no point.

We parameterise Nakamoto by a finality threshold, which is defined as 6 blocks.
This was defined in Satoshi's original whitepaper, which describes probabilistically speaking - the blockchain is final after 6 blocks.
The miner is only stopped if the peer has no chance at mining a longer chain than its peers - which means if the peer is 6 blocks behind the longest chain.

When the block headers have been downloaded, the peer then begins to download the full blocks.
During this period, the peer is not mining, but it is still receiving and gossipping blocks and transactions.
The blocks that are received are ingested into a temporary block store. 
Once the peer has received blocks up until a common ancestor, it can reverse their order into a forward-facing order and ingest them into the block DAG.
During this time, the current state of this chain will point towards the latest ancestor the peer has, until it has completed syncing.

This syncing process is not unique to the first sync. The node will follow the same syncing process at all times if it falls out-of-sync.

One thing to note - the block DAG currently ingests blocks one at a time. There is a case to be made that after each block is ingested, a new tip is computed, and this can trigger a lot of recomputation of the state. 
To mitigate this, we can batch the block ingestion process, and only recompute the state after all blocks have been ingested.

The state machine is reasonably fast to recompute state. Through benchmarking, it is revealed that the state machine can compute the state for a day's worth of transactions in a single second. This is measured without signature validation, as that occurs once only outside of the state machine in a signature cache.
This being said, at a block time of 10mins, the node will lag 10mins at 10*60 = 600 days worth of data. 

When the node starts up, it runs through the following routines:
1. Get the current tip from our block DAG.
    a. Load the state from disk. If the state.blockhash != tip.hash, then we recompute the state.
2. Bootstrap to peers in the network.
3. Full sync to determine latest tip.
4. Recompute the state.
5. Begin the miner, begin ingesting blocks and transactions.

The sync process is uniform:
1. Ask peers for their latest tips. Verify the POW of each tip, and then pick the tip with the most work.
1. Download block headers using this tip. 3 years of block headers is 32.8MB, so it is feasible to download this in a short amount of time.
2. For the earliest ancestor, download the full block and ingest each one.

The sync process:
- Ask our peers for exponentially increasing steps of block headers. This allows us to quickly get a view into the entire longest chain state. It is a noninteractive bissection in essence.
- Determine/pinpoint the earliest ancestor block, then begin downloading the full blocks from all our peers.
- As we are downloading, we continually keep track of the latest block being gossipped to us. 

Let's model the cost of syncing.
3 yrs worth of block headers = 
  block header size = 32 + 32 + 32 + 8 + 8 + 32 + 32 + 32 = 208 bytes\
  blocks per day = (1/10) * 60 * 24 = 144 blocks\
  3 years of headers = 144 * 365 * 3 * 208 = 32,797,440 bytes = 32.8MB

Still, 30mb is a lot to send to one peer while we are gossipping a lot of other stuff. It can impact our throughput and thus the latency of the network.

Peers can download this in an hour at most.
