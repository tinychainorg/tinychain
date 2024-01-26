tinychain
=========

**an ultralight blockchain core, written in Python.**

tinychain is the smallest implementation of a blockchain (BFT replicated state machine) you will ever find. It reimplements the full bitcoin consensus (Nakamoto consensus) with a custom VM based on Brainfuck.

1366 lines of code so far. inspired by [geohot's tinygrad](https://github.com/geohot/tinygrad).

 * cryptography
 * transactions
 * consensus - Nakamoto POW
 * VM’s - Brainfuck
 * state machine
 * gas markets
 * protocol, RPC, and P2P networking

Let the devs do what they do best - building cool stuff.

| **Area**     | **Description**                                                                                                                                                                                                                                                                                            | **Status**  |
|--------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------|
| VM           | [Brainfuck](https://en.wikipedia.org/wiki/Brainfuck) smart contracts                                                                                                                                                                                                                                       | ✅⚠️ 60% Done |
| Consensus    | Bitcoin / Nakamoto / POW with ZK-friendly hash function                                                                                                                                                                                                                                                    | ⚠️ WIP       |
| Tokenomics   | Ethereum-like - native token + fixed exchange rate to gas                                                                                                                                                                                                                                                  | ✅ Done      |
| Cryptography | ECDSA wallets, SECP256k1 curve (same as BTC), SHA-2 256 bit hash function                                                                                                                                                                                                                                  | ✅ Done      |
| Networking   | P2P and RPC servers both use HTTP, gossip network architecture                                                                                                                                                                                                                                             | ⚠️ WIP       |
| ZK proofs    | ZK for compression of tinychain. Use either [groth16](https://github.com/erhant/zkbrainfuck) or [halo2](https://github.com/cryptape/ckb-bf-zkvm) SNARK proofs for brainfuck. TBA we will rework consensus/crypto to use SNARK-friendly tech (MiMC/Poseidon hash function, SNARK-friendly signature scheme) |             |

## Install.

Requirements:

 * Python 3.
 * `pipenv`, `pip` or something like it.

Instructions:

```sh
# Install dependencies.
pipenv install
pipenv shell
```

## Usage.

Demo is a work-in-progress:

```py
# Run two nodes which will mine and sync.
PYTHONPATH=./src python3 src/tinychain/consensus/bitcoin.py
PYTHONPATH=./src python3 src/tinychain/consensus/bitcoin.py
```

## Why?

It takes too long to digest the architecture of any modern blockchain like Ethereum, Optimism, etc.

geohot took PyTorch and distilled it into >10,000 LOC. let's do the same for a blockchain.

maybe we'll learn some things along the way.

## What is a blockchain?

It's really quite an interesting combination of many things.

 * a blockchain is a P2P system based on a programmable database
 * users can run programs on this database
 * they run these programs by cryptographically signing transactions
 * users pay nodes in tokens for running the network
 * how is the cost of running transactions measured?
 * the programs run inside a VM, which has a metering facility for resource usage in terms of computation and storage
 * the unit of account for metering is called gas
 * gas is bought in an algorithmic market for the blockchain's native token. This is usually implemented as a "gas price auction"
 * the order in which these transactions are run is determined according to a consensus algorithm.
 * the consensus algorithm elects a node which is given the power to decide the sequence of transactions for that epoch
 * bitcoin uses proof-of-work, meaning that the more CPU's you have, the more likely you are to become the leader
 * given the sequence of transactions, we can run the state machine
 * the state machine bundles the VM, with a shared context of accounts and their gas credits
 * and this is all bundled together in the node, which provides facilities for querying the state of database

The goal of this project is to elucidate the primitives throughout this invention, in a very simple way, so you can experiment and play with different VM's and code.

## Roadmap.

 - [x] VM
 - [ ] smart contracts
 - [x] wallet
 - [x] transactions
 - [ ] CLI
 - [x] state machine
 - [x] sequencer
 - [x] accounts / gas
 - [ ] node
 - [ ] consensus
 - [ ] networking

See `node.py` for design.

## Feature set.

 - **VM** and **state machine model**. Brainfuck is used as the programming runtime. It includes its own gas metering - 1 gas for computation, 3 gas for writing to memory. There is no in-built object model for now - there is only the Brainfuck VM, and its memory. Any program can write to memory and overwrite other Brainfuck.

 - **Gas market / tokenomics**. Like Ethereum, this chain has a token and an internal unit of account called gas. There is no progressive gas auctions (PGA's) yet - for now it is a fixed exchange rate (see `gas_market.py`).

 - **Consensus**. Bitcoin/Nakamoto consensus is currently being implemented, meaning the network runs via proof-of-work. In future, I hope to implement Tendermint proof-of-stake (see `consensus/tendermint.py` for more) with the token being staked actually hosted on an Ethereum network (L1/L2).

 - **Cryptography**. ECDSA is used for public-private cryptography, with the SECP256k1 curve (used by Bitcoin) and the SHA-2 hash function with size set to 256 bits.

 - **Networking**. The P2P protocol and node API server both run over HTTP. This was easy.

```
curl -X GET http://0.0.0.0:5100/api/machine_eval -H "Content-Type: application/json" -d '{"from_acc":"","to_acc":"","data":"+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++."}'
```
