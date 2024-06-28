# Nakamoto consensus.

Bitcoin consensus is really quite simple.

```
Block
    Prev block hash
    Transactions
    Nonce

Hashcash - a work algorithm, where you can measure CPU expenditure.

Longest chain consensus
Longest = chain with the most accumulated work

What is work? 
Accumulated hashpower (measured by hashcash solutions) that follows the difficulty readjustment schedule

Difficulty readjustment - in order to regulate the block production of the network,
the hashcash difficulty target is adjusted every epoch to target an epoch length of 2 weeks

Epochs are defined every N blocks
Each block includes a timestamp, which is regulated by the network to be close to real clock time


Our algorithm for running this is really simple:
- maintain a block DAG structure
- each block has a parent and a nullable child
- each block has its height (depth in DAG path)
- each block has its accumulated difficulty stored with it
- each block has its index

The "tip" of the chain refers to the block with the most accumulated difficulty - this is the longest chain rule

There are only 3 routines:
- mine - produce blocks, gossip them
- verify_block - verify the POW solution, the transactions and their signatures
- ingest_block - add the block to the DAG
```


## Bitcoin architecture.

```
Miner thread
-> works on mining next block
-> mine(block) takes a block, which includes existing details like difficulty target
-> can be restarted
-> start_mining:
    takes the current mempool (txs)
    and the current tip (block)
    and mines until we get a signal to stop
    then restarts

Before we mine, we figure out what the current difficulty is.

When we receive a new block:
- download any parents until we reach a common block
- then verify the earliest block to the latest
- we also recompute the difficulty for the epochs ingested
- when this is all done, we restart the miner on the freshest tip.

When we mine a new block:
- broadcast the block
- mine on this new tip
```

Finally networking:

```
ConsensusEngine calls protocol to gossip blocks
Node uses ConsensusEngine to Sequence transactions
Node uses transaction Sequence to run replicated State Machine
Protocol uses Wire protocol to communicate over JSON-RPC HTTPS
Node uses Protocol to receive user requests to send transcations
Node receives user transactions and gossips them over network


This doesn't even do fee markets yet. But it should soon.

```


And then on top of this- the launch:

```
People will download the "brainnode" software

```