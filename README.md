tinychain
=========

the tiny smart contract blockchain.

```sh
$ find . -type f -name "*.py" -exec cat {} + | grep -v '^ *#' | grep -v '^\s*$' | wc -l
430
```

only *341* lines of code (50% done). 

dependencies are `ecdsa`, `pyyaml`.

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

see `node.py` for design.

## design.

```py
# Transaction contains data.
# `data` is bytecode.
# bytecode is evaluated by a VM.
# The VM sits within the context of a state machine.
# The state machine is one function - transition.
# Transition looks like: (state, transaction) -> apply(transaction) -> (new_state, exit_code, gas_used)
# if the exit_code is 0, then the transaction is valid.
# The VM maintains an internal unit of resource metering called "gas".
# Gas is expended by the VM as it executes the bytecode, and is priced according to various metrics of resource usage (computation, storage).
# A transaction that exceeds the tx gas limit fails.
# A transaction that exceeds the block gas limit fails.
# A transaction that exceeds its individual gas allowance fails.

# Transactions are constructed by accounts within the system.
# An account is a public-private keypair.
# Each account has a balance.

# The sequencer is a function which gives us the order of transactions.
# Ordinarily, the sequencer is built from a consensus protocol, like hashcash proof-of-work or proof-of-stake.
# For this prototype, we'll make something even simpler.
# We'll use the Github PR consensus algorithm (aka social consensus).
# The GitSequencer syncs a Git repository, and processes transactions inside the txs/ directory.

# There is a currency, called "sneed" which is used to pay for gas.
# Sneed and gas are fungible at 1:1 exchange rate.
# Each account's balance refers to their SNEED balance.
# (Bonus: implement a gas price auction mechanism.)
# To mint sneed to an account, you call the "SneedToken" contract.

# Smart contracts are implemented very simply on top of the state machine.
# Ethereum's model is clunky. We all know this. It is impossible to compose contracts in Ethereum, due to how
# state is implemented.
# Smart contract state on Ethereum just means an area of machine memory where only the contract can read/write to.
# Can we encapsulate it better?
# (introducing lisp)
```