


import struct
import time
from hashlib import sha256
import queue
import threading
from concurrent.futures import *
from abc import ABC, abstractmethod

from tinychain.consensus.bitcoin.block import *


# Difficulty information.
EPOCH_LENGTH_BLOCKS = 8
# TARGET_BLOCK_RATE_SECOND = 1 * 3
TARGET_BLOCK_RATE_SECOND = 1
EPOCH_TARGET_DURATION_SECONDS = EPOCH_LENGTH_BLOCKS * TARGET_BLOCK_RATE_SECOND
INITIAL_DIFFICULTY_TARGET = 2**256 - 1

# epoch_span is a list of blocks in the epoch.
# difficulty_target_0 is the difficulty target of the first block in the epoch.
def get_epoch_difficulty(epoch_span, difficulty_target_0):
    for block in epoch_span:
        print(f"block {block.height}: {block.blockhash} {block.difficulty_target} {block.timestamp}")

    difficulty_target = difficulty_target_0
    epoch_start = epoch_span[0]
    epoch_end = epoch_span[-1]

    # Calculate epoch duration.
    epoch_duration = epoch_end.timestamp - epoch_start.timestamp
    print(f"epoch duration={epoch_duration} difficulty={difficulty_target}")
    
    # Rescale difficulty target.
    difficulty_scale_f = epoch_duration / EPOCH_TARGET_DURATION_SECONDS
    print(f"difficulty retarget factor={difficulty_scale_f} difficulty={difficulty_target}")
    
    # to make blocks faster, lower the difficulty target
    # to make blocks slower, increase the difficulty target
    difficulty_target *= difficulty_scale_f # scale difficulty
    difficulty_target = int(difficulty_target) # round to int
    difficulty_target = min(difficulty_target, INITIAL_DIFFICULTY_TARGET) # clamp to max

    return difficulty_target


# 
# Core consensus engine.
# 
class BitcoinConsensusEngine:
    def __init__(self, db, protocol):
        self.db = db
        self.protocol = protocol
        self.session = self.db()
        
        genesis_block = GENESIS_BLOCK
        self.last_tip = genesis_block.hash().hex()
        print(f"genesis block: {genesis_block.hash().hex()}")
        # assert genesis_block.hash().hex() == "da7a58879c46a9066190deee9d4a1d1118233a1dab9d5c4188b378015860a648"
        if self.get_block_by_hash(genesis_block.hash().hex()):
            print("genesis block already exists")
        else:
            self.add_block(genesis_block)
        
        # the blockhash of the last tip.
        self.last_tip = self.get_tips()[0].blockhash
    
    # Verifies a block.
    def verify_block(self, block):
        # 1. Get the block's parent.
        # 2. Verify the timestamp is monotonically increasing (ie. A < B)
        # 3. Verify the POW solution.
        # 4. Verify the difficulty is computed correctly.
        # 5. Verify the transaction signatures are valid.
        return True

    def download_unknown(self, blockhash):
        raw_blocks = self.protocol.peer_get_blocks([blockhash])
        if len(raw_blocks) == 0:
            raise Exception(f"download_unknown - block not found: {blockhash}")
        
        return self.on_recv_block(raw_blocks[0])

    def on_recv_block(self, raw_block):
        # blocks to ingest.
        blocks = [raw_block]

        # download all the unknown blocks in this path.
        # request ancestors via gossip.
        while True:
            parent_blockhash = raw_block.parent_blockhash.to_bytes(32, byteorder='big').hex()
            parent_block = self.get_block_by_hash(parent_blockhash)
            if parent_block:
                break
            else:
                raw_blocks = self.protocol.peer_get_blocks([parent_blockhash])
                if len(raw_blocks) == 0:
                    raise Exception(f"block not found: {parent_blockhash}")
                for block in raw_blocks:
                    # block = decode_block_yaml(block_data)
                    blocks.append(block)
                    raw_block = block # TODO brittle af
                    # TODO verify the right block is received.
        
        # now verify them in order of age (oldest first).
        # sort by age- 
        blocks = sorted(blocks, key=lambda x: x.timestamp)

        # verify each block.
        for block in blocks:
            has_block = self.get_block_by_hash(block.hash().hex()) is not None
            if has_block:
                continue
            
            is_valid = self.verify_block(block)
            if is_valid:
                self.add_block(block)
            else:
                raise Exception(f"block is invalid: {block}")

    def get_block_by_hash(self, block_hash):
        assert isinstance(block_hash, str)
        # print(f"get_block_by_hash: {block_hash}")
        return self.session.query(DAGBlock).filter(DAGBlock.blockhash == block_hash).first()

    # Adds the block to the DAG.
    def add_block(self, raw_block):
        session = self.session

        # 1. Create the core block details.
        dag_block = DAGBlock()
        dag_block.blockhash = raw_block.hash().hex()
        dag_block.txs = "" #TODO
        dag_block.timestamp = raw_block.timestamp
        dag_block.difficulty_target = str(raw_block.difficulty_target)
        dag_block.nonce = raw_block.nonce

        # 2. Get its parent and compute accumulators.
        # height = height + 1
        # acc_work = acc_work + difficulty
        # parent = parent.id
        dag_block.parent_blockhash = raw_block.parent_blockhash.to_bytes(32, byteorder='big').hex()
        parent = session.query(DAGBlock).filter(DAGBlock.blockhash == dag_block.parent_blockhash).first()
        
        if parent:
            dag_block.height = parent.height + 1
            # handle a max of 2^256 blocks
            # the max value of difficulty is 2^256
            # 2^256 * 2^256 = 2^512
            work = (2**256 - raw_block.difficulty_target)
            acc_work = int(parent.acc_work, 16) + work
            dag_block.acc_work = acc_work.to_bytes(64, byteorder='big').hex()
        else:
            # This block has no parent, so it's a root block in the DAG.
            dag_block.height = 0
            acc_work = (2**256 - raw_block.difficulty_target)
            dag_block.acc_work = acc_work.to_bytes(64, byteorder='big').hex()

        # 3. Save the new block.
        session.add(dag_block)
        session.commit()

        # Update the new tip
        self.update_last_tip()
    
    def update_last_tip(self):
        current_tip = self.last_tip
        self.last_tip = self.get_tips()[0].blockhash
        if current_tip != self.last_tip:
            print(f"reorg / tip updated: {current_tip} -> {self.last_tip}")
            return True
        
    def get_tips(self):
        # Query the DAG for top 6 blocks by acc_work.
        session = self.session
        tips = session.query(DAGBlock).order_by(DAGBlock.acc_work.desc()).limit(6).all()
        return tips
    
    # Gets the difficulty target for a block.
    def get_difficulty(self, parent_blockhash):
        # Case: genesis block.
        if parent_blockhash == 0:
            return INITIAL_DIFFICULTY_TARGET
        
        # Get the height in the chain.
        parent_block = self.get_block_by_hash(u256_to_str(parent_blockhash))
        if not parent_block:
            raise Exception(f"parent block not found: {parent_blockhash}")
        block_height = parent_block.height + 1

        # Detect whether this is a new epoch.
        is_epoch_boundary = block_height % EPOCH_LENGTH_BLOCKS == 0

        # Case: within epoch.
        if not is_epoch_boundary:
            return int(parent_block.difficulty_target)
        
        # Case: block on epoch boundary.
        # Retarget difficulty every epoch.

        # Get all blocks of last epoch, by traversing the DAG upwards (ancestors) to the start of the epoch.
        epoch_start_height = block_height - EPOCH_LENGTH_BLOCKS
        curr_block = parent_block
        chain = [curr_block]
        while epoch_start_height <= curr_block.height:
            if curr_block.height == 0:
                break
            curr_block = self.get_block_by_hash(curr_block.parent_blockhash)
            chain.append(curr_block)
        
        difficulty_target_1 = get_epoch_difficulty(list(reversed(chain)), int(curr_block.difficulty_target))
        return difficulty_target_1

    def mine1(self, parent_blockhash: int, start_nonce=0, n_blocks=16):
        chain = []
        block = Block(parent_blockhash, [])

        # 1. Solve proof-of-work puzzle.
        while len(chain) < n_blocks:
            parent_block = self.get_block_by_hash(u256_to_str(block.parent_blockhash))
            block.timestamp = time.time()
            block.nonce = start_nonce
            block.height = parent_block.height + 1 or 0
            
            # 1. Get difficulty for block.
            difficulty_target = self.get_difficulty(block.parent_blockhash)
            print(f"difficulty_target: {difficulty_target}")
            block.difficulty_target = difficulty_target

            while True:
                # update timestamp every 500ms (throttled).
                # https://bitcoin.stackexchange.com/questions/3165/what-hash-rate-can-a-raspberry-pi-achieve-can-the-gpu-be-used 
                # 0.2 MH/s - 200 KH/s - 200,000 H/s
                if block.nonce % (200_000 / 2):
                    block.timestamp = time.time()
                
                # increment nonce.
                block.nonce += 1

                # hash.
                h = block.hash()

                # check POW solution
                if int.from_bytes(h, byteorder='big') < difficulty_target:
                    break
            
            print(f"POW solution block={block.height} target={difficulty_target} nonce={block.nonce} hash={h.hex()}")
            chain.append(block)
            self.add_block(block)

            block = Block(int.from_bytes(h, byteorder='big'), [])
            block.timestamp = time.time()
            block.nonce = start_nonce
        
        return chain



# Run.
# 


def test_mining():
    genesis_block = Block(0, [])
    db = get_database('mining', memory=False)
    # db = get_database('mining', memory=True)
    consensus = BitcoinConsensusEngine(db)
    
    # Mine 16 blocks.
    print("mining 16 blocks...")
    chain = consensus.mine1(0, n_blocks=4)
    print("mined 16 blocks")
    print(chain)

    # Now mine 2 subchains from common ancestor.
    ancestor_block = chain[-1]
    print(f"ancestor block hash: {ancestor_block.hash().hex()}")
    chain1 = consensus.mine1(ancestor_block.hash_int(), start_nonce=1, n_blocks=4)
    chain2 = consensus.mine1(ancestor_block.hash_int(), start_nonce=2, n_blocks=4)





def test_1():
    genesis_block = Block(0, [])
    print("genesis block:")
    print(genesis_block.hash().hex())

    db = get_database('testnet')


    consensus_proto = MockConsensusProto()
    consensus = BitcoinConsensusEngine(db)
    
    # Mine 16 blocks.
    print("mining 16 blocks...")
    # chain = consensus.mine1(genesis_block, n_blocks=8)
    chain = consensus.mine1(genesis_block, n_blocks=32)
    print("mined 16 blocks")
    print(chain)
    
    # Add blocks to DAG.
    print("adding blocks to DAG...")
    for block in chain:
        consensus.add_block(block)

def test_2_tips():
    genesis_block = Block(0, [])
    db = get_database('testnet1')
    consensus_proto = MockConsensusProto()
    consensus = BitcoinConsensusEngine(db)
    


    # Mine 2 paths.
    print("mining 2 paths...")
    # chain1 = consensus.mine1(genesis_block, n_blocks=8)

    # Get tips.
    print("getting tips...")
    tips = consensus.get_tips()
    for tip in tips:
        print(f"tip: {tip.blockhash} {tip.acc_work}")
    

if __name__ == '__main__':
    test_networking()
    # test_mining()
    # test_2_tips()
    
