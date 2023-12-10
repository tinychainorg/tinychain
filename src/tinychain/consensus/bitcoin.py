


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



import queue
import threading
from concurrent.futures import *


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
                int(self.timestamp),
                self.coinbase_acc
            )
        ])

    def hash(self):
        return sha256(self.envelope()).digest()



from abc import ABC, abstractmethod

class ConsensusProtocol(ABC):
    @abstractmethod
    def download_block(self, block_id):
        pass



# A difficulty epoch is `EPOCH_LENGTH_BLOCKS`.
# The target block rate is `TARGET_BLOCK_RATE_SECOND`.
# The target epoch duration is `EPOCH_TARGET_DURATION_SECONDS`.
# e.g. 8 blocks per epoch, 1 block per 3 seconds, 24 seconds per epoch.
EPOCH_LENGTH_BLOCKS = 8
TARGET_BLOCK_RATE_SECOND = 1 * 3
EPOCH_TARGET_DURATION_SECONDS = EPOCH_LENGTH_BLOCKS * TARGET_BLOCK_RATE_SECOND
INITIAL_DIFFICULTY_TARGET = 2**256 - 1

# How does the consensus engine work? 
# The height of a block refers to its depth in the DAG, which begins at the genesis block.
# The tip of the DAG is the block with the most accumulated work.
# Work is defined as the sum of difficulty ie. hashpower.
# We run two routines:
# 1. Miner - mine for a solution to the hashcash puzzle.
# 2. Syncer - process new blocks, determine the longest chain, notify the miner and trigger the on_new_tip event.
class BitcoinConsensusEngine:
    def __init__(self, genesis_block, consensus_protocol):
        self.genesis_block = genesis_block
        self.blocks_by_id = {}
        self.blocks_by_height = {}
        self.consensus_protocol = consensus_protocol
    
    def get_difficulty_targets_for_chain(self, blocks):
        past_difficulty_targets = [INITIAL_DIFFICULTY_TARGET]
        
        difficulty_target = INITIAL_DIFFICULTY_TARGET
        
        for i, block in enumerate(blocks):
            print(i, block.hash().hex())

            # Retarget difficulty every epoch.
            if (i+1) % EPOCH_LENGTH_BLOCKS == 0:
                # Get all blocks of last epoch
                epoch_span = blocks[-EPOCH_LENGTH_BLOCKS:]
                print(epoch_span)

                epoch_start = epoch_span[0]
                epoch_end = epoch_span[-1]
                # Calculate epoch duration.
                epoch_duration = epoch_end.timestamp - epoch_start.timestamp
                print(f"epoch duration={epoch_duration} difficulty={difficulty_target}")
                
                # Rescale difficulty target.
                difficulty_scale_f = epoch_duration / EPOCH_TARGET_DURATION_SECONDS
                
                # to make blocks faster, lower the difficulty target
                # to make blocks slower, increase the difficulty target
                past_difficulty_targets.append(difficulty_target)
                difficulty_target *= difficulty_scale_f

                print(f"epoch duration={epoch_duration} difficulty={difficulty_target}")
                print(f"difficulty retarget factor={difficulty_scale_f} difficulty={difficulty_target}")
        
        past_difficulty_targets.append(difficulty_target)
        return past_difficulty_targets
    
    def compute_acc_difficulty(self, chain):
        difficulty_targets = self.get_difficulty_targets_for_chain(chain)
        print(f"difficulty targets={difficulty_targets}")

        # Compute the accumulated difficulty.
        accumulated_work = 0
        last_timestamp = 0

        for i, block in enumerate(chain):
            height = i
            difficulty_target = difficulty_targets[i // EPOCH_LENGTH_BLOCKS]

            # Verify POW solution.
            h = block.hash()
            solved = False
            if int.from_bytes(h, byteorder='big') < difficulty_target:
                solved = True
            
            if not solved:
                raise Exception(f"Invalid POW solution for block height={height} hash={h.hex()} difficulty={difficulty_target}")

            # Verify timestamp.
            valid_timestamp = False
            if last_timestamp < block.timestamp:
                valid_timestamp = True
            
            if not valid_timestamp:
                raise Exception(f"Invalid timestamp for block height={height} timestamp={block.timestamp} last_timestamp={last_timestamp}")
            
            # Accumulate work.
            accumulated_work += int.from_bytes(h, byteorder='big')
        
        print(f"accumulated work={accumulated_work}")
        return accumulated_work

    def get_tip(self, new_block):
        dag_path = []
        block = new_block

        # 1. Download all blocks in the path that we don't know of.
        while True:
            parent_block_id = block.prev_block_hash
            if parent_block_id not in self.blocks_by_id:
                # Download parent block.
                block = self.consensus_protocol.download_block(parent_id)
                # Verify work.
                # TODO.
                dag_path.append(block)
        
        # 2. Compute the accumulated work in this path.
        # As part of this, the following constraints hold:
        # 1) the hashcash POW solution is valid for each block.
        # 2) the block's timestamp is greater than the parent's timestamp.
        # 3) the difficulty adjustment algorithm is followed.
        
        # Get the full path to this latest tip.
        past_blocks = []
        curr_block = dag_path[-1]
        while curr_block.id != self.genesis_block.id:
            past_blocks.append(curr_block)
            curr_block = self.blocks_by_id[curr_block.prev_block_hash]
        past_blocks.append(self.genesis_block)
        full_path = past_blocks + dag_path
        difficulty_targets = self.get_difficulty_targets_for_chain(full_path)

        # 3. Save the new tip + blocks.
        # TODO.

        # 4. If tip != new_tip, restart mining on the new tip.
        # TODO.


    # Hashcash works by finding a number such that the hash of the number has a certain number of leading zeros.
    # Encoded at a low level, this means that the hash of the number is less than a certain value.
    # This value is the difficulty target.
    #
    # The difficulty target is retargeted such that the average time to solve the puzzle is 10 minutes.
    # The difficulty is adjusted every 2016 blocks, which is approximately every 2 weeks.
    # The difficulty is adjusted by a factor of 4 in either direction.    
    def mine1(self, block, n_blocks=16):
        chain = []

        # 1. Set difficulty.
        difficulty_target = INITIAL_DIFFICULTY_TARGET

        # 2. Solve proof-of-work puzzle.
        while len(chain) < n_blocks:
            # Mine.
            while True:
                block.nonce += 1
                h = block.hash()
                if int.from_bytes(h, byteorder='big') < difficulty_target:
                    break
            
            block.timestamp = time.time()
            # self.on_solution(block)
            print(f"POW solution block={len(chain)} target={difficulty_target} nonce={block.nonce} hash={h.hex()}")
            chain.append(block)

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

            next_block = Block(int.from_bytes(h, byteorder='big'), [])
            block = next_block
        
        return chain
    
    # def mine2(self, block):
    #     chain = []

    #     # 1. Set difficulty.
    #     difficulty_target = 2**256 - 1
        
    #     with concurrent.futures.ThreadPoolExecutor() as executor:
    #         miner_get_next_block = executor.submit(solve_proof_of_work, difficulty_target, block)
    #         network_get_next_block = executor.submit()

    #         # Wait for either of the futures to complete
    #         # get the next block - await [self.miner.next_block, self.network.next_block]
    #         done, not_done = concurrent.futures.wait([future_1, future_2], return_when=concurrent.futures.FIRST_COMPLETED)

    #         # Get the result from the completed future
    #         result = done.pop().result()  # Get the result of the completed future

    #         # Cancel the other future which hasn't completed
    #         for future in not_done:
    #             future.cancel()


def solve_proof_of_work(difficulty_target, block):
    while True:
        block.nonce += 1
        h = block.hash()
        if int.from_bytes(h, byteorder='big') < difficulty_target:
            break
        time.sleep(1/1000) # 1ms
    print(f"POW solution block={len(chain)} target={difficulty_target} nonce={block.nonce} hash={h.hex()}")
    return block.nonce


# PYTHONPATH=. python3 tinychain/consensus/bitcoin.py

class MockConsensusProto:
    def download_block(self, block_id):
        print(f"downloading block {block_id}")
        return Block(block_id, [])

if __name__ == '__main__':
    genesis_block = Block(0, [])
    print("genesis block:")
    print(genesis_block.hash().hex())

    consensus_proto = MockConsensusProto()
    consensus = BitcoinConsensusEngine(genesis_block, consensus_proto)
    
    # Mine 16 blocks.
    print("mining 16 blocks...")
    chain = consensus.mine1(genesis_block, n_blocks=16)
    print("mined 16 blocks")
    print(chain)
    print("chain tip:")
    print(chain[-1].hash().hex())
    
    # compute diff targets
    targets = consensus.get_difficulty_targets_for_chain(chain)

    # compute acc
    acc = consensus.compute_acc_difficulty(chain)
    


    