


import struct
import time
from hashlib import sha256
import queue
import threading
from concurrent.futures import *
from abc import ABC, abstractmethod


# Base block data structure.
# 

class Block:
    def __init__(self, prev_block_hash, txs = []):
        self.prev_block_hash = prev_block_hash
        self.txs = txs
        self.timestamp = 0
        self.difficulty_target = 0
        self.nonce = 0
    
    # The envelope is the data we are hashing.
    def envelope(self):
        return b"".join([
            self.prev_block_hash.to_bytes(32, byteorder='big'),
            self.nonce.to_bytes(32, byteorder='big'),
            struct.pack(
                "<QQQ",
                self.difficulty_target,
                int(self.timestamp),
                self.coinbase_acc
            )
        ])

    def hash(self):
        return sha256(self.envelope()).digest()


# Block encoding/decoding.
import yaml
def decode_block_yaml(txt):
    data = yaml.safe_load(txt)
    return Block(
        data['prev_block_hash'],
        data['nonce'],
        data['difficulty'],
        data['timestamp'],
        data['coinbase_acc'],
        data['txs']
    )

def encode_block_yaml(d):
    data = {
        'prev_block_hash': d.prev_block_hash,
        'nonce': d.nonce,
        'difficulty': d.difficulty,
        'timestamp': d.timestamp,
        'coinbase_acc': d.coinbase_acc,
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
    prev_block_hash = Column(String)
    txs = Column(String)
    timestamp = Column(Float, default=0.0)
    difficulty_target = Column(String, default=0)
    nonce = Column(Integer, default=0)

    # === DAG metadata. === #
    blockhash = Column(String, primary_key=True)
    height = Column(Integer, default=0)
    acc_work = Column(Integer, default=0)
    parent = relationship("DAGBlock", remote_side=[blockhash], backref='child', uselist=False)
    parent_blockhash = Column(String, ForeignKey('dag_blocks.blockhash'))
    # child = [inferred]

def get_database():
    DATABASE_URI = 'sqlite:///example.db'

    engine = create_engine(DATABASE_URI)
    Session = sessionmaker(bind=engine)
    return Session



# Difficulty information.
EPOCH_LENGTH_BLOCKS = 8
TARGET_BLOCK_RATE_SECOND = 1 * 3
EPOCH_TARGET_DURATION_SECONDS = EPOCH_LENGTH_BLOCKS * TARGET_BLOCK_RATE_SECOND
INITIAL_DIFFICULTY_TARGET = 2**256 - 1


# 
# Core consensus engine.
# 
class BitcoinConsensusEngine:
    def __init__(self, genesis_block, consensus_protocol):
        self.genesis_block = genesis_block
        self.blocks_by_id = {}
        self.blocks_by_height = {}
        self.consensus_protocol = consensus_protocol
        self.db_session = get_database()
    
    # Verifies a block.
    def verify_block(self):
        pass

    # Adds the block to the DAG.
    def add_block(self, raw_block):
        session = self.db_session()

        # 1. Create the core block details.
        dag_block = DAGBlock()
        dag_block.prev_block_hash = raw_block.prev_block_hash.to_bytes(32, byteorder='big').hex()
        dag_block.txs = "" #TODO
        dag_block.timestamp = raw_block.timestamp
        dag_block.difficulty_target = raw_block.difficulty_target
        dag_block.nonce = raw_block.nonce

        # 2. Get its parent and compute accumulators.
        # height = height + 1
        # acc_work = acc_work + difficulty
        # parent = parent.id
        parent = session.query(DAGBlock).filter(DAGBlock.blockhash == raw_block.prev_block_hash).first()
        if parent:
            dag_block.height = parent.height + 1
            dag_block.acc_work = parent.acc_work + raw_block.difficulty_target
            dag_block.parent_hash = parent.blockhash
        else:
            # This block has no parent, so it's a root block in the DAG.
            dag_block.height = 1
            dag_block.acc_work = raw_block.difficulty_target

        # 3. Save the new block.
        session.add(dag_block)
        session.commit()

        # 4. Update + save the parent.
        # parent.child = block.id
        # if parent:
        #     parent.child_backref.append(dag_block)
        #     session.add(parent)
        #     session.commit()

        session.close()

    
    def get_tips(self):
        # Query the DAG for top 6 blocks by acc_work.
        # return []
    

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

if __name__ == '__main__':
    genesis_block = Block(0, [])
    print("genesis block:")
    print(genesis_block.hash().hex())

    consensus_proto = MockConsensusProto()
    consensus = BitcoinConsensusEngine(genesis_block, consensus_proto)
    
    # Mine 16 blocks.
    print("mining 16 blocks...")
    chain = consensus.mine1(genesis_block, n_blocks=8)
    print("mined 16 blocks")
    print(chain)



# PYTHONPATH=. python3 tinychain/consensus/bitcoin.py