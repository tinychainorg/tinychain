# tinychain

A full blockchain in Go;

 * Nakamoto consensus.
   * Hashcash.
   * Dynamic difficulty retargeting (epochs).
   * Proof-of-work consensus - longest/heaviest chain rule.
   * Merklized transaction tree for light client availability.
 * State sync - interactive binary search for common ancestor, parallelised block header download from multiple peers.
 * Ethereum-like coin state machine - basic ERC20 transfers.
 * ECDSA (curve P256) wallets for signing transactions. Signature malleability fixes.
 * Core data structures: RawBlock, RawTx, Block, Tx, Epoch, BlockDAG, current tip, Miner, NetPeer, Node
 * Networking: HTTP peer interface, messages/methods include [bootstrap/gossip peers, gossip blocks, gossip txs].
 * Miner: mine new blocks on the tip, measure hashrate.
 * CLI: start a node, connect to the network, mine blocks.

Simplifications of the design:

 * The difficulty target is represented as `[32]bytes`; it is uncompressed. There is no `nBits` or custom mantissa.
 * Transaction signatures are in their uncompressed ECDSA form. They are `[65]bytes`, which includes the ECDSA signature type of `0x4`.
 * Transactions specify `from` and `to` in terms of raw ECDSA public keys. There is no ECDSA signature recovery to guess the pubkey from a signature.
 * The state machine is an account-based model, not a UXTO model. It implements just a transfer for coins.

Dependencies:

 * [go-sqlite3](https://github.com/mattn/go-sqlite3?tab=readme-ov-file)

Couple philosophies of this project:

 - distilling a blockchain network down to its core primitives at all layers of the stack.

![database view](./assets/db-view.png)

## Install.

Make sure you have Go 1.2.3+ installed.

```sh
make && cd build/ && ./tinychain node --port 8121 --db testnet.db
```

## To Do.

Work breakdown:

- [x] building a simple block
- [x] hashing a simple block
- [x] building a proof of work solution
- [x] building a chain of blocks
- [x] computing the epoch/difficulty window for a chain of blocks
- [x] creating a merkle tree accumulation of transactions
- [x] computing the cumulative work in a chain of blocks
- [x] constructing a blockdag and then choosing a tip
- [x] fix tests, institute CI as development practice
- [x] simple coin state machine
    - [x] coinbase
    - [x] state diffs
    - [x] adding a method to recompute the state machine and using cached state 
    - [ ] state snapshots/checkpoints
- [x] adding a simple state machine
    - [x] ValidateBlock
        - first transaction is the coinbase
        - maintain a uxto set - unspent transaction outputs
        - validate txs - validate signature, transfer the coins
- [x] implement simple peer
    - [x] can send and receive blocks via network
    - [x] gossip block, gossip tx, get blocks, sync tip
    - [x] implement peer discovery and bootstrapping
- [x] implement state machine, state snapshots
  - [ ] design state model
  - [ ] model state growth costs and time to reconstruct state
- [ ] implement state sync
    - [ ] get tips from all peers in parallel
    - [ ] interactive binary search to find common ancestor for peer
    - [ ] download block headers
    - [ ] download block bodies
    - [ ] validate and ingest blocks
    - [ ] implement the temporary block header cache, tx cache for when we receive stuff from network
- [ ] implement node startup routine
    - [ ] 1. Get the current tip from our block DAG. Load the state from disk. If the state.blockhash != tip.hash, then we recompute the state.
    - [ ] 2. Bootstrap to peers in the network.
    - [ ] 3. Full sync to determine latest tip.
    - [ ] 4. Recompute the state.
    - [ ] 5. Begin the miner, begin ingesting blocks and transactions.
- [ ] implement node state API's
- [ ] implement tokenomics module
    - [ ] pure function given the current block, return the coinbase reward.
- [ ] implement mempool
    - [ ] peer api to submit txs
    - [ ] gossip transactions to peers
    - [ ] mempool data structure
    - [ ] mempool.addtransaction - validate tx not already mined, calculate feerate, kick other transactions
    - [ ] peer api to query mempool
    - [ ] peer api to query feerate
    - [ ] connect mempool to miner. restart miner on new mempool transactions. restart miner on new tip (I think this already happens?)
    - [ ] after new tip, mempool should clear out transactions that have already been mined.
    - [ ] handle reorg so we can reinsert values into mempool.
    - [ ] peer api to query tx confirmations (for exchange and wallets etc.)
- [ ] improve txs
    - [ ] replay protection for txs, tx nonce
    - [x] add version to RawBlock, RawTransaction for future prosperity
- [x] implementing networking
    - [ ] simple wrapper for sockets - address, port, and hof's to wrap the latency delays dropped packets etc
    - [x] peers connect
    - [x] peers can send messages, peers can register message handlers
    - [x] gossip peers routine
    - [ ] peer discovery from bittorrent trackers
- [ ] implement wallet and cli tool
- [x] implement miner process
- [ ] implement admin api
- [x] finally implement the CLI tool
    - [x] start miner
    - [x] stop miner
    - [ ] check balance
    - [ ] check network online nodes
    - [ ] send coins
    - [ ] receive coins
- [ ] refactoring
    - [x] add nonce to tx
    - [ ] rename difficulty -> difficulty_target
    - [x] refactor miner code to be pretty
    - [ ] improve robustness of sql queries- need to verify we use right number of columns and ?'s
    - [ ] improve block/rawblock. missing fields, unset fields etc. tests for this.
    - [ ] double check big endian canonical encoding
    - [ ] add zlib compression for rawblock, block (just like google's sstable)
- [ ] observability tooling
    - [ ] index the pubkeys of miners of the network (coinbase txs)
    - [ ] index the hashrate of individual nodes
    - [ ] dashboard covering all nodes in network, uptime, % blocks contributed
    - [ ] reorgs
- [ ] implement a simple tinychain exchange using django. accepts DAI/USD/WBTC and swaps to TINY coin. 
- [ ] generate zk proofs of chain history
