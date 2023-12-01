from encoding import json_repr


# How this works:

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
# Encapsulating it, global memory looks like: 
# (storage (
#   (read key value)
#   (write key value)
# )
# Whereas something with access permissions looks more like:
# (access-check (storage (read write)))
# The (access-check) function performs the access check.
# In a normal blockchain, we have an additional "context" that the state machine gets which is used for access checks.
# The context is simple - the transaction (from, to, data).



class Transaction:
    def __init__(self, from_acc, to_acc, data):
        pass

class Block:
    def __init__(self, transactions, prev_block):
        pass

class VM:
    def __init__(self):
        pass

class Sequencer:
    def __init__(self):
        pass

class Network:
    def __init__(self):
        pass


if __name__ == "__main__":
    print(json_repr(Transaction(1, 2, 3)))