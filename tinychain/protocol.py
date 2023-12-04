from blockchain import Blockchain

CLIENT_VERSION = '0.0.1'
PROTOCOL_VERSION = '0.0.1'

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
    def __init__(self, blockchain: Blockchain):
        self.peers = []
        self.blockchain = blockchain
    
    def run(self):
        # Routine: Run the peer bootstrap subroutine.
        # Routine: Run the peer gossip subroutine.
        # Routine: Run the consensus subroutine.
        # Routine: Run the transaction gossip subroutine.
        pass

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
        peer = HttpProtocolPeer(addr, port)
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
