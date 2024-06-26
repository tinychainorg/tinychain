# tinychain

A full blockchain in Go;

## To Do.

Work breakdown:

- [x] building a simple block
- [x] hashing a simple block
- [x] building a proof of work solution
- [x] building a chain of blocks
- [x] computing the epoch/difficulty window for a chain of blocks
- [x] creating a merkle tree accumulation of transactions
- [ ] computing the cumulative work in a chain of blocks
- [ ] constructing a blockdag and then choosing a tip
- [ ] build a simple mempool module
- [ ] adding a simple state machine
    - [ ] ValidateBlock
        - first transaction is the coinbase
        - maintain a uxto set - unspent transaction outputs
        - validate txs - validate signature, transfer the coins
- [ ] adding a method to recompute the state machine and using cached state 
- [ ] implementing networking
    - [ ] simple wrapper for sockets - address, port, and hof's to wrap the latency delays dropped packets etc
    - [ ] peers connect
    - [ ] peers can send messages, peers can register message handlers
- [ ] implement simple peer
    - [ ] can send and receive blocks via network
    - [ ] gossip block, gossip tx, get blocks, sync tip
- [ ] implement peer discovery
- [ ] implement wallet and cli tool
- [ ] implement miner process
- [ ] implement admin api
- [ ] finally implement the CLI tool
    - [ ] start miner
    - [ ] stop miner
    - [ ] check balance
    - [ ] check network online nodes
    - [ ] send coins
    - [ ] receive coins
