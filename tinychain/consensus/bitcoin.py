


# Bitcoin consensus is based on proof-of-work / Hashcash algorithms.
# 
# === Blocks. ===
# The basic unit of time is the block.
# A block is a collection of transactions, as well as consensus-related metadata.
# Blocks are linked together into a chain, such that each block is built upon the work of the last.
# Nodes collect transactions into a block, and then compete to solve a cryptographic puzzle to decide the next "block" in the chain.
# 
# === The proof-of-work puzzle. ===
# The puzzle is based on finding a hash which has a certain number of leading zeros (a primitive known as hashcash).
# The number of leading zeros is referred to as the `difficulty`, and is retargeted such that the average time to solve the puzzle is 10 minutes.
# When a node solves the puzzle, it broadcasts the block to the network.
# 
# === Consensus. ===
# Consensus is based on the longest chain rule. The longest chain is the chain with the most cumulative difficulty. 
# The likelihood of a reorganization decreases exponentially as the number of blocks increases.
# 


class Block:
    def __init__(self, prev_block_hash, txs):
        self.prev_block_hash = prev_block_hash
        self.txs = txs
        self.nonce = 0
        self.difficulty = 0
        self.timestamp = 0
        self.coinbase_acc = ""

class BitcoinConsensusEngine:
    def __init__(self, node):
        self.node = node
    
    def on_new_block(self):
        pass
    
    def solve_pow(self, challenge_fn, difficulty):
        nonce = 0
        while True:
            nonce += 1
            if challenge_fn(nonce) < difficulty:
                return nonce
    
    def mine(self, block):
        def challenge_fn(nonce):
            # 1. Add nonce to block.
            # 2. Hash block.
            # 3. Return hash.
            return 0
        