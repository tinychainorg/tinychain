


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
import struct
import time
from hashlib import sha256
from tinychain import tinychain



import queue
import threading




class Block:
    def __init__(self, prev_block_hash, txs):
        self.prev_block_hash = prev_block_hash
        self.nonce = 0
        self.difficulty = 0
        self.timestamp = 0
        self.coinbase_acc = 0

        self.txs = txs
    
    # The envelope is the data we are signing.
    # It excludes the signature.
    def envelope(self):
        return b"".join([
            self.prev_block_hash.to_bytes(32, byteorder='big'),
            self.nonce.to_bytes(32, byteorder='big'),
            struct.pack(
                "<QQQ",
                self.difficulty,
                self.timestamp,
                self.coinbase_acc
            )
        ])

    def hash(self):
        return sha256(self.envelope()).digest()



class MinerThread(threading.Thread):
    def __init__(self,  *args, **kwargs):
        super(MinerThread, self).__init__(*args, **kwargs)
        self._stop_event = threading.Event()

    def stop(self):
        self._stop_event.set()

    def stopped(self):
        return self._stop_event.is_set()


class BitcoinConsensusEngine:
    def __init__(self, genesis_block):
        self.genesis_block = genesis_block
    
    def on_new_block(self):
        # cancel current mining.
        # begin mining new thing.


    def on_solution(self, block):
        pass
    
    # Hashcash works by finding a number such that the hash of the number has a certain number of leading zeros.
    # Encoded at a low level, this means that the hash of the number is less than a certain value.
    # This value is the difficulty target.
    #
    # The difficulty target is retargeted such that the average time to solve the puzzle is 10 minutes.
    # The difficulty is adjusted every 2016 blocks, which is approximately every 2 weeks.
    # The difficulty is adjusted by a factor of 4 in either direction.    
    def mine(self, block):
        chain = []

        # 1. Set difficulty.
        difficulty_target = 2**256 - 1

        # 2. Solve proof-of-work puzzle.
        while True:
            # Mine.
            while True:
                block.nonce += 1
                h = block.hash()
                if int.from_bytes(h) < difficulty_target:
                    break
            
            block.timestamp = time.time()
            self.on_solution(block)
            print(f"POW solution block={len(chain)} target={difficulty_target} nonce={block.nonce} hash={h.hex()}")
            chain.append(block)

            # A difficulty epoch is `EPOCH_LENGTH_BLOCKS`.
            # The target block rate is `TARGET_BLOCK_RATE_SECOND`.
            # The target epoch duration is `EPOCH_TARGET_DURATION_SECONDS`.
            # e.g. 8 blocks per epoch, 1 block per 3 seconds, 24 seconds per epoch.
            EPOCH_LENGTH_BLOCKS = 8
            TARGET_BLOCK_RATE_SECOND = 1 * 3
            EPOCH_TARGET_DURATION_SECONDS = EPOCH_LENGTH_BLOCKS * TARGET_BLOCK_RATE_SECOND

            # Retarget difficulty every epoch.
            if len(chain) % EPOCH_LENGTH_BLOCKS == 0:
                # Get all blocks of last epoch
                epoch_span = chain[-EPOCH_LENGTH_BLOCKS:]
                epoch_start = epoch_span[0]
                epoch_end = epoch_span[-1]
                # Calculate epoch duration.
                epoch_duration = epoch_end.timestamp - epoch_start.timestamp
                print(f"epoch duration={epoch_duration} difficulty={difficulty_target}")
                
                
                # Rescale difficulty target.
                difficulty_scale_f = epoch_duration / EPOCH_TARGET_DURATION_SECONDS
                
                # to make blocks faster, lower the difficulty target
                # to make blocks slower, increase the difficulty target
                difficulty_target *= difficulty_scale_f

                print(f"epoch duration={epoch_duration} difficulty={difficulty_target}")
                print(f"difficulty retarget factor={difficulty_scale_f} difficulty={difficulty_target}")

            next_block = Block(int.from_bytes(h), [])
            block = next_block


# Basic function of consensus engine:
# - maintain current block
# - mine on that block for a nonce
# - when nonce is found, broadcast block to network
# - when new block is received, check if it is valid. then swap to mining on that block.
# - all the consensus engine does is tell us (1) the current tip of the chain

if __name__ == '__main__':
    genesis_block = Block(0, [])
    print("genesis block:")
    print(genesis_block.hash().hex())

    # mine the next block
    consensus_engine = BitcoinConsensusEngine(genesis_block)
    consensus_engine.mine(genesis_block)

    