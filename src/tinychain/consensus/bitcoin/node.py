
import multiprocessing
from threading import Thread

from tinychain.protocol_http import HttpProtocolNode
from tinychain.protocol import Protocol

from tinychain.consensus.bitcoin.consensus import BitcoinConsensusEngine
from tinychain.consensus.bitcoin.common import str_to_u256, get_database
import time

class SimpleConsensusNode:
    def __init__(self):
        pass

    # main loop:
    # 1. Mine blocks using latest state.
    # 2. Await new blocks, verify and download them.
    # 3. Update the state machine.
    def run(self, datakey, peer_listen_addr, bootstrap_peers: list = [], run_miner=False):
        # after every block solution, broadcast to network.
        # after every received block, download and verify it
        # after every consensus tip update, rejog the state machine.

        blockchain = None
        protocol = Protocol(blockchain, None)
        laddr, lport = peer_listen_addr.split(":")
        node = HttpProtocolNode(laddr, lport)
        node.register_method("get_blocks", protocol.get_blocks)
        node.register_method("broadcast_block", protocol.broadcast_block)
        node.register_method("get_tip", protocol.get_tip)
        protocol.on_broadcast_block = self.on_broadcast_block
        protocol.on_get_blocks = self.on_get_blocks
        protocol.on_get_tip = self.on_get_tip


        db = get_database(f"{datakey}", memory=False)
        consensus = BitcoinConsensusEngine(db, protocol)

        for peer_addr in bootstrap_peers:
            paddr, pport = peer_addr.split(":")
            protocol.connect_bootstrap_peer(paddr, pport)

        self.node = node
        self.protocol = protocol
        self.consensus = consensus

        # Sync tips from peers.
        Thread(target=self.sync_local_tip).start()
        # Now run threads.
        Thread(target=self.run_miner).start() if run_miner else None
        # Thread(target=self.run_main).start()
        Thread(target=self.run_node).start()

    def sync_local_tip(self):
        tips = self.consensus.get_tips()
        latest_tip = tips[0].blockhash
        remote_tips = self.protocol.peer_sync_tip(local_tip=latest_tip)
        for remote_tip in remote_tips:
            print(f"syncing unknown block: {remote_tip}")
            if not self.consensus.get_block_by_hash(remote_tip):
                self.consensus.download_unknown(remote_tip)

    def on_get_tip(self, local_tip):
        tips = self.consensus.get_tips()
        return tips[0].blockhash

    def run_main(self):
        while True:
            time.sleep(1)
            print(1111111111)

    def run_node(self):
        self.node.listen()

    def run_miner(self):
        last_block = None

        while True:
            tips = self.consensus.get_tips()
            latest_tip = tips[0]
            print(f"mining on tip: hash={latest_tip.blockhash}")
            chain = self.consensus.mine1(str_to_u256(latest_tip.blockhash), n_blocks=1)
            print("mined 1 block")
            last_block = chain[0]
            # Broadcast.      
            Thread(target=self.broadcast_block, args=(chain[0],)).start()

            # Mine max 1 block per second.
            diff = time.time() - last_block.timestamp
            time.sleep(max(0, 1 - diff))
    
    def broadcast_block(self, block):
        print(f"broadcasting block: {block.hash().hex()}")
        self.protocol.broadcast_block(block=encode_block_yaml(block))
    
    def on_broadcast_block(self, block):
        b = decode_block_yaml(block)
        self.consensus.on_recv_block(b)
    
    def on_get_blocks(self, blockhashes):
        print(f"on_get_blocks hashes={blockhashes}")
        lis = []
        for blockhash in blockhashes:
            b = self.consensus.get_block_by_hash(blockhash)
            if b:
                lis.append(encode_block_yaml(b.to_block()))
            else:
                print(f"block not found: {blockhash}")
        print(f"ret: {lis}")
        return lis


import socket
from contextlib import closing
   
def is_port_open(port):
    with closing(socket.socket(socket.AF_INET, socket.SOCK_STREAM)) as sock:
        return sock.connect_ex(("0.0.0.0", port)) == 0

def test_networking():
    if not is_port_open(5100):
        x = SimpleConsensusNode()
        x.run("node1", "0.0.0.0:5100", ["0.0.0.0:5101"], run_miner=True)
    else:
        x = SimpleConsensusNode()
        x.run("node2", "0.0.0.0:5101", ["0.0.0.0:5100"], run_miner=True)


if __name__ == "__main__":
    test_networking()