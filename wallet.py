import ecdsa
from hashlib import sha256


# A wallet is a public-private keypair.
# The public key is the address.
# The private key is used to sign transactions.
# SECP256k1 is the Bitcoin elliptic curve used in this wallet.
class Wallet:
    def __init__(self, prvkey):
        self.prvkey = prvkey
    
    def __repr__(self) -> str:
        return self.pubkey_str()

    def pubkey(self):
        return self.prvkey.get_verifying_key()
    
    def pubkey_str(self):
        return self.pubkey().to_string().hex()
    
    def prvkey_str(self):
        return self.prvkey.to_string().hex()
    
    @staticmethod
    def create_random():
        prvkey = ecdsa.SigningKey.generate(curve=ecdsa.SECP256k1, hashfunc=sha256)
        return Wallet(prvkey)
    
    @staticmethod
    def from_private_key(private_key_hex):
        prvkey = ecdsa.SigningKey.from_string(bytes.fromhex(private_key_hex), curve=ecdsa.SECP256k1, hashfunc=sha256)
        return Wallet(prvkey)

    def sign(self, data):
        # assert type of `data` is bytes
        if type(data) != bytes:
            raise Exception("data must be bytes")
        return self.prvkey.sign(data)

    @staticmethod
    def verify(pubkey_hex, sig, msg):
        # assert type of `data` is bytes
        if type(msg) != bytes:
            raise Exception("data must be bytes")
        vk = ecdsa.VerifyingKey.from_string(bytes.fromhex(pubkey_hex), curve=ecdsa.SECP256k1, hashfunc=sha256)
        return vk.verify(sig, msg)


if __name__ == "__main__":
    w = Wallet.create_random()
    print("Wallet details:")
    print("  Private key: {}".format(w.prvkey_str()))
    print("  Public key: {}".format(w.pubkey_str()))

    msg = b"hello world"
    sig = w.sign(msg)
    print("Signature: {}".format(sig.hex()))

    is_valid = Wallet.verify(w.pubkey_str(), sig, msg)
    print("Verify signature: {}".format(is_valid))
