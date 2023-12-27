from tinychain.blockchain import Blockchain
from tinychain.protocol_http import HttpProtocolPeer

CLIENT_VERSION = '0.0.1'
PROTOCOL_VERSION = '0.0.1'

import yaml
def decode_block_yaml(txt):
    data = yaml.safe_load(txt)
    b = Block(
        data['parent_blockhash'],
        data['txs']
    )
    b.nonce = data['nonce']
    b.difficulty_target = data['difficulty_target']
    b.timestamp = data['timestamp']
    return b

import struct
from hashlib import sha256
class Block:
    def __init__(self, parent_blockhash: int, txs = []):
        self.parent_blockhash = parent_blockhash
        self.txs = txs
        self.timestamp = 0
        self.difficulty_target = 0
        self.nonce = 0

    # The envelope is the data we are hashing.
    def envelope(self):
        return b"".join([
            self.parent_blockhash.to_bytes(32, byteorder='big'),
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


# This is the core of our protocol.
# Message types:
# - net_getPeers
# - net_getBlocks
# - net_getTransactions
# - consensus_propose()
# - consensus_prevote()
# - consensus_precommit()
# - consensus_commit()
# - machine_eval
# - user_sendTransaction -> result
# - user_getTransaction
# - user_getBalance
class Protocol:
    def __init__(self, blockchain: Blockchain, consensus_engine):
        self.peers = []
        self.blockchain = blockchain
        self.consensus_engine = consensus_engine
    
    def run(self):
        # Routine: Run the peer bootstrap subroutine.
        # Routine: Run the peer gossip subroutine.
        # Routine: Run the consensus subroutine.
        # Routine: Run the transaction gossip subroutine.
        pass

    def broadcast_block(self, block=""):
        for peer in self.peers:
            peer.recv_block(block=block)

    def routine_bootstrap_peers(self):
        # Connect to the configured bootstrap peers.
        pass
    def routine_gossip_peers(self):
        # Regularly update our peer list with new peers to prevent network churn.
        pass
    def routine_run_consensus(self):
        # Run the consensus algorithm.
        pass
    def routine_gossip_transactions(self):
        # Listen to new transactions and gossip them to other peers.
        pass


    def connect_bootstrap_peer(self, addr, port):
        peer = HttpProtocolPeer(addr, port, self)
        self.peers.append(peer)

        # dial and discover other peers.
        # TODO.
        # new_peers = peer.net_getPeers()
        return peer
    


    # Public RPC methods.
    # 
    def net_version(self):
        return {
            # The software client.
            'client': f'tinychain-client-{CLIENT_VERSION}',
            # The network name.
            'network': "sydney",
            # The wire protocol.
            'protocol': f'tinychain-protocol-{PROTOCOL_VERSION}'
        }

    def net_getPeers(self):
        return { 'peers': self.peers }

    def user_getBalance(self, addr=None):
        return self.blockchain.state_machine.accounts[addr]
    
    def machine_eval(self, from_acc="", to_acc="", data=""):
        print("machine_eval from={} to={} data={}".format(from_acc, to_acc, data))
        # TODO: just for now
        class DummyTx:
            def __init__(self, from_acc, to_acc, data):
                self.from_acc = from_acc
                self.to_acc = to_acc
                self.data = data
        
        (output, gas_used) = self.blockchain.state_machine.eval(DummyTx(from_acc, to_acc, data))
        return { 'output': output, 'gas_used': gas_used }
    
    def broadcast_block(self, block=None):
        self.on_broadcast_block(block)

    def get_blocks(self, blockhashes=[]):
        return self.on_get_blocks(blockhashes)

    def get_tip(self, local_tip):
        return self.on_get_tip(local_tip)

    def peer_get_blocks(self, blockhashes=[]):
        blocks = []
        for peer in self.peers:
            res = peer.get_blocks(blockhashes=blockhashes)
            for d in res:
                block = decode_block_yaml(d)
                blocks.append(block)
            if len(blocks) == len(blockhashes):
                break
        
        return blocks

    def peer_sync_tip(self, local_tip):
        tips = []
        for peer in self.peers:
            res = peer.get_tip(local_tip=local_tip)
            tips.append(res)
        return tips
