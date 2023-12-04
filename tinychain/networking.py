from flask import Flask, jsonify, request
import socket
import functools
import traceback
from blockchain import Blockchain


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

# The networking protocol implemented here is atop HTTP.
# Why HTTP?
# - we need a reliable transport layer - TCP provides this
# - we need "messages" that aren't limited to the maximum size of a datagram (MTU) - HTTP provides this via paths ie. /api/...
# - we need message types - HTTP provides this via paths too
# - it's much more useful to have a protocol with a pre-existing toolset for interacting with it - eg. browsers, curl, etc.
# - it's fun! we can even host a web server for people to publicly check out our node
# - it's easy to implement security on top via SSL/TLS
class HttpProtocolNode:
    def __init__(self, addr, port, protocol):
        self.app = Flask(__name__)
        self.addr = addr
        self.port = port
        self.protocol = protocol
        self.registered_methods = []

        self.register_read_method(self.protocol.net_getPeers)
        self.register_read_method(self.protocol.user_getBalance)
        self.register_read_method(self.protocol.machine_eval)

    def listen(self):
        self.app.add_url_rule(
            '/api/', 
            view_func=self.index_methods,
            methods=['GET']
        )

        print("Listening on http://{}:{}/api/".format(self.addr, self.port))
        self.app.run(host=self.addr, port=self.port)
    
    def index_methods(self):
        return "Registered methods:<ul><li>{}</ul>".format("<li>".join(self.registered_methods))

    def register_read_method(self, method):
        self.registered_methods.append(method.__name__)
        partial_wrap = functools.partial(self.wrap_method, method, 'GET')
        partial_wrap.__name__ = method.__name__

        self.app.add_url_rule(
            '/api/{}'.format(method.__name__), 
            view_func=partial_wrap,
            methods=['GET']
        )

    def register_write_method(self, method):
        self.registered_methods.append(method.__name__)
        partial_wrap = functools.partial(self.wrap_method, method, 'POST')
        partial_wrap.__name__ = method.__name__

        self.app.add_url_rule(
            '/api/{}'.format(method.__name__),
            view_func=partial_wrap(method),
            methods=['POST']
        )

    def wrap_method(self, method, *args, **kwargs):
        # print(request.json)
        try:
            res = method(**request.json)
            return jsonify(result=res, status=200)
        except Exception as err:
            print(err)
            traceback.print_tb(err.__traceback__)
            return jsonify(error=err, status=500)


if __name__ == "__main__":
    blockchain = Blockchain()
    # blockchain.run()
    protocol = Protocol(blockchain)
    node1 = HttpProtocolNode("0.0.0.0", 5100, protocol)
    node1.listen()

    # node2 = HttpProtocolNode("0.0.0.0", 5101)
    # node2.listen()

