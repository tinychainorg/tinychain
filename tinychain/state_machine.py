from tinychain.vms.brainfuck import BrainfuckVM

class StateMachine:
    def __init__(self):
        # The VM this machine runs on.
        self.vm = BrainfuckVM()
        # The clock. aka last processed transaction ID.
        self.t = None

    # Evaluates a transaction and applies its effects.
    def apply(self, tx):
        self.vm.apply(tx.data)
        self.t = tx.id()

    # Evaluates a transaction and doesn't persist its effects.
    def eval(self, tx):
        self.vm.eval(tx.data)
