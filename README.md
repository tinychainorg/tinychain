## tinychain

[![CI](https://github.com/tinychainorg/tinychain/actions/workflows/go.yml/badge.svg)](https://github.com/tinychainorg/tinychain/actions/workflows/go.yml)

[Website](https://www.tinycha.in) | [API documentation](https://pkg.go.dev/github.com/tinychainorg/tinychain) | [Concepts](./docs/concepts.md)

**tinychain is the tiniest blockchain implementation in the world.**

We're building a cleanroom implementation of a Nakamoto-style blockchain. Think of it like [tinygrad](https://github.com/tinygrad/tinygrad) but for bitcoin.

The core goals of this project are to:

 1. Build a blockchain in under 10K lines of code, from scratch.
 2. Launch a blockchain network (and prove this isn't just a library or light client; it's the real deal!).
 3. Distill a blockchain network down to its core primitives at all layers of the stack. 
 4. Create an educational resource for studying blockchains, the actual engineering of how one works - not just the conceptual idea of proof-of-work, but the engineering aspects - like synchronisation, DAG's, and other algorithms. 
 5. Build a foundation to innovate and experiment more quickly with new tech, like DAG-based sequencing algorithms, Bitcoin MEV and proposer-builder separation, instant sync using zero-knowledge proofs, and more.

Taking inspiration from projects like SQLite, TempleOS, Cosmos/Tendermint, Linux, KHTML, tinygrad, nanogpt. These are all open-source technologies, embodying extremely minimal implementations of software ([tinygrad](https://github.com/tinygrad/tinygrad), [nanogpt](https://github.com/karpathy/nanoGPT), [tinyRAM](https://hackernoon.com/tiny-ram-review-architecture-design-and-assembly-instructions)), projects launched as hobbies to compete with professional endeavors ([Linux](https://en.wikipedia.org/wiki/History_of_Linux#The_creation_of_Linux)), code built out of open-source love for building something really high quality ([KHTML](https://en.wikipedia.org/wiki/WebKit), which later spawned WebKit), engines that distilled a complex thing into its core ideas ([Tendermint/Cosmos](https://tendermint.com/core/)), highly portable and reliable tools ([SQLite](https://www.sqlite.org))...and then just free agents building entirely new branches of the tech tree, like [TempleOS](https://en.wikipedia.org/wiki/TempleOS).

## Overview.

**Specification:**

 * Nakamoto consensus.
   * Hashcash.
   * Dynamic difficulty retargeting (epochs).
   * Proof-of-work consensus - longest/heaviest chain rule.
   * Merklized transaction tree for light client availability.
 * State sync - greedy iterative search for blocks, light client sync, parallelised block header download from multiple peers.
 * Bitcoin-style tokenomics - coinbase, transaction fees.
 * State machine - with an account-based model.
 * ECDSA (curve P256) wallets for signing transactions. Signature malleability fixes.
 * Core data structures: RawBlock, RawTx, Block, Tx, Epoch, BlockDAG, current tip, Miner, NetPeer, Node
 * Networking: HTTP peer interface, messages/methods include [bootstrap/gossip peers, gossip blocks, gossip txs].
 * Miner: mine new blocks on the tip, measure hashrate.
 * CLI: start a node, connect to the network, mine blocks.

**Dependencies:**

 * [go-sqlite3](https://github.com/mattn/go-sqlite3?tab=readme-ov-file)

## Status.

tinychain is in-development. Currently we can run a node, mine blocks, ingest them into a DAG, create and sign transactions, run a state machine, build the UXTO set from processing transactions, connect to peers and gossip.

In progress: state synchronisation, user wallet API's.

## Install.

Requirements: Go 1.2.3+.

```sh
make && cd build/ && ./tinychain node --port 8000 --db testnet.db -mine
```

## Running a tiny Nakamoto network.

```sh
# Terminal 1:
./e2e/node1.sh
# Terminal 2:
./e2e/node2.sh
```

