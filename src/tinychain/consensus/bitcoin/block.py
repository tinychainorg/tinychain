

def u256_to_str(x):
    return x.to_bytes(32, byteorder='big').hex()

def str_to_u256(x):
    return int(x, 16)


# Base block data structure.
# 

class Block:
    def __init__(self, parent_blockhash: int, txs = []):
        self.parent_blockhash = parent_blockhash
        self.txs = txs
        self.timestamp = 0
        self.difficulty_target = INITIAL_DIFFICULTY_TARGET
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


# Block encoding/decoding.
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

def encode_block_yaml(d):
    data = {
        'parent_blockhash': d.parent_blockhash,
        'nonce': d.nonce,
        'difficulty_target': d.difficulty_target,
        'timestamp': d.timestamp,
        'txs': d.txs
    }
    # Dump keys in order defined above.
    return yaml.dump(data, default_flow_style=False, sort_keys=False)