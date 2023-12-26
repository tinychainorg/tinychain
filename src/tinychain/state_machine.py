from tinychain.vms.brainfuck import BrainfuckVM
from collections import defaultdict
from tinychain.gas_market import GasMarket

class InsufficientFundsException(Exception):
    pass

class StateMachine:
    def __init__(self, gas_market):
        # The VM this machine runs on.
        self.vm = BrainfuckVM()
        # The clock. aka last processed transaction ID.
        self.t = None
        # The accounts and their token balances.
        self.accounts = defaultdict(int)
        # The gas market.
        self.gas_market = GasMarket()

    # Evaluates a transaction and applies its effects.
    def apply(self, tx):
        # Exchange the token for gas.
        # In Ethereum, this is an auction.
        # Here we just used a fixed exchange rate.
        token_balance = self.accounts[tx.from_acc]
        gas_balance = self.gas_market.token_to_gas(token_balance)

        if gas_balance == 0:
            raise InsufficientFundsException()
        
        gas_limit = gas_balance

        # 1. Apply the tx.
        code = tx.data.decode('utf-8')
        (output, gas_used) = self.vm.apply(code=code, gas_limit=gas_limit)
        print("state-machine.apply tx={} gas_used={} output=\"{}\"".format(tx.id(), gas_used, output))

        # 2. Advance the clock.
        self.t = tx.id()

        # 3. Deduct gas costs from account.
        self.accounts[tx.from_acc] -= gas_used

    # Evaluates a transaction and doesn't persist its effects.
    def eval(self, tx):
        EVAL_GAS_LIMIT = 12_000_000
        (output, gas_used) = self.vm.eval(tx.data, gas_limit=EVAL_GAS_LIMIT)
        return (output, gas_used)
