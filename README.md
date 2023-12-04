tinychain
=========

the tiny smart contract blockchain.

tinychain runs the Brainfuck VM, uses ECDSA for signing transactions, and a Git-based sequencer (yes you PR txs to get them into the mempool).

## usage.

```sh
# Install dependencies.
pipenv install
pipenv shell

# Run the node.
cd tinychain/
python node.py
```

## why?

It takes too long to digest the architecture of any modern blockchain like Ethereum, Optimism, etc.

geohot took PyTorch and distilled it into >10,000 LOC. let's do the same for a blockchain.

maybe we'll learn some things along the way.

## what is a blockchain?

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

## roadmap.

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

see `node.py` for design.


## featureset

 - **VM**. Brainfuck is used as the programming runtime. It includes its own gas metering - 1 gas for computation, 3 gas for writing to memory.

 - **Cryptography**. ECDSA is used for public-private cryptography.

 - **Networking**. Nodes run a HTTP API server, containing all methods. 

```
curl -X GET http://0.0.0.0:5100/api/machine_eval -H "Content-Type: application/json" -d '{"from_acc":"","to_acc":"","data":"+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++."}'
```