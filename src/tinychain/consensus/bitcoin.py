


import struct
import time
from hashlib import sha256
import queue
import threading
from concurrent.futures import *
from abc import ABC, abstractmethod

def u256_to_str(x):
    return x.to_bytes(32, byteorder='big').hex()


# Difficulty information.
EPOCH_LENGTH_BLOCKS = 8
TARGET_BLOCK_RATE_SECOND = 1 * 3
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
    difficulty_target *= difficulty_scale_f
    difficulty_target = int(difficulty_target)

    print(f"epoch duration={epoch_duration} difficulty={difficulty_target}")
    print(f"difficulty retarget factor={difficulty_scale_f} difficulty={difficulty_target}")

    return difficulty_target


# Base block data structure.
# 

class Block:
    def __init__(self, parent_block_hash: int, txs = []):
        self.parent_block_hash = parent_block_hash
        self.txs = txs
        self.timestamp = 0
        self.difficulty_target = INITIAL_DIFFICULTY_TARGET
        self.nonce = 0
    
    # The envelope is the data we are hashing.
    def envelope(self):
        return b"".join([
            self.parent_block_hash.to_bytes(32, byteorder='big'),
            self.nonce.to_bytes(32, byteorder='big'),
            self.difficulty_target.to_bytes(32, byteorder='big'),
            struct.pack(
                "<Q",
                int(self.timestamp)
                # self.difficulty_target
            )
        ])

    def hash(self):
        return sha256(self.envelope()).digest()
    
    def hash_int(self):
        return int.from_bytes(self.hash(), byteorder='big')


# Block encoding/decoding.
import yaml
def decode_block_yaml(txt):
    data = yaml.safe_load(txt)
    return Block(
        data['parent_block_hash'],
        data['txs'],
        data['difficulty_target']
    )

def encode_block_yaml(d):
    data = {
        'parent_block_hash': d.parent_block_hash,
        'nonce': d.nonce,
        'difficulty_target': d.difficulty_target,
        'timestamp': d.timestamp,
        'txs': d.txs
    }
    # Dump keys in order defined above.
    return yaml.dump(data, default_flow_style=False, sort_keys=False)


# Database.
# Block DAG data structure.
# Includes metadata.

import sqlite3
from sqlalchemy import Column, Integer, String, ForeignKey, Float
from sqlalchemy.orm import relationship
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

Base = declarative_base()

class DAGBlock(Base):
    __tablename__ = 'dag_blocks'

    # === Base block details. === #
    txs = Column(String)
    timestamp = Column(Float, default=0.0)
    difficulty_target = Column(String, default="0")
    nonce = Column(Integer, default=0)

    # === DAG metadata. === #
    blockhash = Column(String, primary_key=True)
    height = Column(Integer, default=0)
    acc_work = Column(String, default="0")
    parent = relationship("DAGBlock", remote_side=[blockhash], backref='child', uselist=False)
    parent_blockhash = Column(String, ForeignKey('dag_blocks.blockhash'), nullable=True) 
    # child = inferred

def get_database(name, memory=False):
    DATABASE_URI = f'sqlite:///bitcoin_{name}.db'
    if memory:
        DATABASE_URI = 'sqlite:///:memory:'
    engine = create_engine(DATABASE_URI)
    Base.metadata.create_all(engine)
    Session = sessionmaker(bind=engine)
    return Session

class MockDbSessionMaker():
    def __init__():
        pass
    def __call__(self):
        return None





# 
# Core consensus engine.
# 
class BitcoinConsensusEngine:
    def __init__(self, genesis_block, consensus_protocol, db):
        self.genesis_block = genesis_block
        self.blocks_by_id = {}
        self.blocks_by_height = {}
        self.consensus_protocol = consensus_protocol
        self.db = db
    
    # Verifies a block.
    def verify_block(self):
        # verify timestamp.
        # compute next epoch if need be.
        # verify POW solution / difficulty.
        # verify tx signatures.
        pass

    def on_block_gossip(self, raw_block):
        # request ancestors via gossip.
        # then verify each block, and ingest.
        pass

    def get_block_by_hash(self, block_hash):
        assert isinstance(block_hash, str)
        session = self.db()
        return session.query(DAGBlock).filter(DAGBlock.blockhash == block_hash).first()

    # Adds the block to the DAG.
    def add_block(self, raw_block):
        session = self.db()

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
        dag_block.parent_blockhash = raw_block.parent_block_hash.to_bytes(32, byteorder='big').hex()
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

        session.close()
    
    def get_tips(self):
        # Query the DAG for top 6 blocks by acc_work.
        session = self.db()
        tips = session.query(DAGBlock).order_by(DAGBlock.acc_work.desc()).limit(6).all()
        session.close()

        return tips
    
    def get_difficulty(self, parent_block_hash):
        # assert isinstance(parent_block_hash, int)

        # Case: genesis block.
        if parent_block_hash == 0:
            return INITIAL_DIFFICULTY_TARGET
        
        # Get the height in the chain.
        parent_block = self.get_block_by_hash(u256_to_str(parent_block_hash))
        if not parent_block:
            raise Exception(f"parent block not found: {parent_block_hash}")
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

    def mine1(self, parent_block_hash: int, start_nonce=0, n_blocks=16):
        chain = []
        block = Block(parent_block_hash, [])

        # 1. Solve proof-of-work puzzle.
        while len(chain) < n_blocks:
            block.timestamp = time.time()
            block.nonce = start_nonce

            # 1. Get difficulty for block.
            difficulty_target = self.get_difficulty(block.parent_block_hash)
            print(f"difficulty_target: {difficulty_target}")
            block.difficulty_target = difficulty_target

            while True:
                # update timestamp every 500ms (throttled).
                # https://bitcoin.stackexchange.com/questions/3165/what-hash-rate-can-a-raspberry-pi-achieve-can-the-gpu-be-used 
                # 0.2 MH/s - 200 KH/s - 200,000 H/s
                if block.nonce % 200_000:
                    block.timestamp = time.time()
                
                # increment nonce.
                block.nonce += 1

                # hash.
                h = block.hash()

                if int.from_bytes(h, byteorder='big') < difficulty_target:
                    break
            
            # self.on_solution(block)
            print(f"POW solution block={len(chain)} target={difficulty_target} nonce={block.nonce} hash={h.hex()}")
            chain.append(block)
            self.add_block(block)

            block = Block(int.from_bytes(h, byteorder='big'), [])
            block.timestamp = time.time()
            block.nonce = start_nonce
        
        return chain


# Networking.
# 

class ConsensusProtocol(ABC):
    @abstractmethod
    def download_block(self, block_id):
        pass

class MockConsensusProto:
    def download_block(self, block_id):
        print(f"downloading block {block_id}")
        return Block(block_id, [])


# Run.
# 


def test_mining():
    genesis_block = Block(0, [])
    db = get_database('mining', memory=False)
    # db = get_database('mining', memory=True)
    consensus_proto = MockConsensusProto()
    consensus = BitcoinConsensusEngine(genesis_block, consensus_proto, db)
    
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
    consensus = BitcoinConsensusEngine(genesis_block, consensus_proto, db)
    
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
    consensus = BitcoinConsensusEngine(genesis_block, consensus_proto, db)
    


    # Mine 2 paths.
    print("mining 2 paths...")
    # chain1 = consensus.mine1(genesis_block, n_blocks=8)

    # Get tips.
    print("getting tips...")
    tips = consensus.get_tips()
    for tip in tips:
        print(f"tip: {tip.blockhash} {tip.acc_work}")
    

if __name__ == '__main__':
    test_mining()
    # test_2_tips()
    


# PYTHONPATH=. python3 tinychain/consensus/bitcoin.py