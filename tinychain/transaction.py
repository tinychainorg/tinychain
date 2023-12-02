from hashlib import sha256
import yaml
from wallet import Wallet
from crypto import verify_sig

def decode_tx_yaml(txt):
    data = yaml.safe_load(txt)
    return Tx(data['from'], data['to'], data['data'], data['sig'])

def encode_tx_yaml(tx):
    data = {
        'from': tx.from_acc,
        'to': tx.to_acc,
        'data': tx.data.hex(),
        'sig': tx.sig.hex()
    }
    # Dump keys in order defined above.
    return yaml.dump(data, default_flow_style=False, sort_keys=False)

class Tx:
    def __init__(self, from_acc=None, to_acc=None, data=None, sig=None):
        self.from_acc = from_acc
        self.to_acc = to_acc
        self.data = data
        self.sig = sig

    def parse(self, str):
        pass

    def __repr__(self) -> str:
        sig_str = "" if self.sig is None else self.sig.hex()
        return "Tx(from={}, to={}, data={}, sig={})".format(self.from_acc, self.to_acc, self.data, sig_str)

    # The envelope is the data we are signing.
    # It excludes the signature.
    def envelope(self):
        return b"".join([
            bytes.fromhex(self.from_acc),
            bytes.fromhex(self.to_acc),
            self.data
        ])


if __name__ == "__main__":
    wallet_liam = Wallet.create_random()
    wallet_sylve = Wallet.create_random()

    tx = Tx()
    tx.from_acc = wallet_liam.pubkey_str()
    tx.to_acc = wallet_sylve.pubkey_str()
    tx.data = b"hello sylve"
    tx.sig = wallet_liam.sign(tx.envelope())

    print("tx: {}".format(tx)) 
    
    # from_pubkey_known = wallet_liam.pubkey_str()
    # print("from_pubkey: {}".format(from_pubkey_known))

    # from_pubkey_recovered = recover_pubkey(tx.sig, tx.envelope()).to_string().hex()
    # print("from_pubkey_recovered: {}".format(from_pubkey_recovered))

    # is_valid = verify_sig(from_pubkey_recovered, tx.sig, tx.envelope())
    # print("Verify signature: {}".format(is_valid))

    is_valid = verify_sig(tx.from_acc, tx.sig, tx.envelope())
    print("Verify signature: {}".format(is_valid))

    print(encode_tx_yaml(tx))