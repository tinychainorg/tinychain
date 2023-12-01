tinychain
=========

a simple and powerful smart contract blockchain. 

## why?

It takes too long to digest the architecture of any modern blockchain like Ethereum, Optimism, etc.

geohot took PyTorch and distilled it into >10,000 LOC. let's do the same for a blockchain.

maybe we'll learn some things along the way.

## roadmap.

 - [ ] VM
 - [ ] smart contracts
 - [ ] transactions
 - [ ] sequencer
 - [ ] accounts / gas
 - [ ] storage and state
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