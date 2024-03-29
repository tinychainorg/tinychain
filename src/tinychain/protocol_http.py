# This implements the wire protocol - aka the encoding of the abstract protocol into a real form
# which we can communicate over an actual network (read: wires).
# 
# This file contains:
#  - a HTTP node: an implementation of our local node, aka a server for protocol methods
#  - a HTTP peer: an implementation of a remote node, aka a client for protocol methods
# 
# This is a simple RPC layer over HTTP. 
# Nodes are HTTP servers which serve requests under /api/.
# 
# Why HTTP as our wire protocol? Why not JSON-RPC, gRPC, libp2p, a binary protocol over TCP/UDP, etc.?
# - we need a reliable transport layer - TCP provides this
# - we need "messages" that aren't limited to the maximum size of a datagram (MTU) - HTTP provides this via paths ie. /api/...
# - we need message types - HTTP provides this via paths too
# - it's much more useful to have a protocol with a pre-existing toolset for interacting with it - eg. browsers, curl, etc.
# - it's fun! we can even host a web server for people to publicly check out our node
# - it's easy to implement security on top via SSL/TLS
# 
# 
import socket
import functools
import traceback
import requests
from flask import Flask, jsonify, request
from tinychain.blockchain import Blockchain
# from tinychain.protocol import Protocol
from threading import Thread


class HttpProtocolNode:
    def __init__(self, addr, port):
        self.app = Flask(__name__)
        self.addr = addr
        self.port = port
        self.registered_methods = []

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

    def handle(self, method_name, method):
        self.registered_methods.append(method_name)
        partial_wrap = functools.partial(self.wrap_method, method_name, method, 'POST')
        partial_wrap.__name__ = method_name

        self.app.add_url_rule(
            f"/api/{method_name}",
            view_func=partial_wrap,
            methods=['POST']
        )

    def wrap_method(self, method_name, method, *args, **kwargs):
        print(f"[{self.addr}:{self.port}] handling request for method {method_name} ({method.__name__}) with args={args} kwargs={kwargs}")
        # print(request.json)
        try:
            res = method(**request.json)
            return jsonify(result=res, status=200)
        except Exception as err:
            print(err)
            traceback.print_tb(err.__traceback__)
            return jsonify(error=err, status=500)

# Meta-programming helper.
# https://stackoverflow.com/a/29333454/453438
from types import FunctionType
def methods(cls):
    return [x for x, y in cls.__dict__.items() if type(y) == FunctionType]

class HttpProtocolPeer:
    def __init__(self, addr, port, protocol):
        self.addr = addr
        self.port = port
        # self.methods = methods(protocol)
        # self.methods = methods(Protocol)

    def __getattr__(self, name):
        return functools.partial(self.call_method, name)
        # if name in self.methods:
        #     return functools.partial(self.call_method, name)
        # else:
        #     raise AttributeError("{} has no attribute {}".format(self.__class__.__name__, name))
    
    def call_method(self, name, *args, **kwargs):
        print(f"[{self.addr}:{self.port}] calling method {name} with args={args} kwargs={kwargs}")

        # HTTP request to node.
        # Handle Connection refused
        res = requests.post("http://{}:{}/api/{}".format(self.addr, self.port, name), json=kwargs)
        
        print(res.json())
        
        if res.status_code != 200:
            raise Exception("Error calling method {}: {}".format(name, res.json()))
        elif res.json()['status'] != 200:
            raise Exception("Error calling method {}: {}".format(name, res.json()))
        
        return res.json()['result']


def test_start_node(host="0.0.0.0", port=5000):
    # protocol = Protocol(Blockchain())
    
    # Create a hTTP node.
    # node = HttpProtocolNode(host, port, protocol)
    # Create the protocol.
    # Add consensus gossip to the protocol.
    # Register method on node.
    # Connect the consensus module to the protocol.
    # Start the miner.
    pass


    

def test_miners(self):
    # two nodes
    
    # node_b = HttpProtocolNode("0.0.0.0", 5200, protocol)

    # python start two threads
    Thread(target=node_a.listen()).start()
    # Thread(target=node_b.listen()).start()






if __name__ == "__main__":
    # blockchain = Blockchain()
    # protocol = Protocol(blockchain)
    # node1 = HttpProtocolNode("0.0.0.0", 5100, protocol)
    # node1.listen()

    # q = HttpProtocolPeer("0.0.0.0", 5100)
    # balance = q.user_getBalance("aaaa")
    # print("balance={}".format(balance))

    # node2 = HttpProtocolNode("0.0.0.0", 5101)
    # node2.listen()
    
    pass