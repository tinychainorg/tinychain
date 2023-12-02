


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


from sequencer import FileSystemSequencer
from state_machine import StateMachine
from gas_market import GasMarket

class Node:
    def __init__(self):
        pass

    def run(self):
        # The node is the orchestrator:
        # - the state machine combines a VM to run transactions, with the context of state and gas usage.
        # - the sequencer provides the order of transactions.
        # - the node runs the state machine on each transaction, in order.
        # - the node also provides RPC API's so users can read the state.
        gas_market = GasMarket()
        state_machine = StateMachine(gas_market)
        
        state_machine.accounts["945b45ec3cc2e838e9ef50b70c9065e5a1ad4d992050320ae6a5b70c6b744f3a91abb481b653067cd4aeacc2b7ad37cac04ddd422c22611499b7da18138c3ec6"] = 100

        sequencer = FileSystemSequencer("../testnet-1/txs")
        for tx in sequencer.txs:
            print("processing tx: {}".format(tx.id()))
            state_machine.apply(tx)

        print(state_machine.vm.dump_memory())


if __name__ == "__main__":
    node = Node()
    node.run()