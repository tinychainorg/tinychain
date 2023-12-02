import ecdsa
from ecdsa import VerifyingKey, BadSignatureError, SECP256k1
from ecdsa.util import sigdecode_der
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
    
    # A wallet's address is the double hash of its public key.
    # address = sha256(sha256(pubkey))
    def address(self):
        return sha256(sha256(self.pubkey_str().encode()).digest()).digest().hex()

    @staticmethod
    def create_random():
        prvkey = ecdsa.SigningKey.generate(curve=ecdsa.SECP256k1, hashfunc=sha256)
        return Wallet(prvkey)
    
    @staticmethod
    def from_private_key(private_key_hex):
        prvkey = ecdsa.SigningKey.from_string(bytes.fromhex(private_key_hex), curve=ecdsa.SECP256k1, hashfunc=sha256)
        return Wallet(prvkey)

    def sign(self, msg):
        full_signature = sign_msg(self.prvkey, msg)

        # print(f"Generated Signature: {full_signature.hex()}")

        return full_signature

# Sign a message using a private key.
def sign_msg(prvkey, msg):
    # assert type of `msg` is bytes
    if type(msg) != bytes:
        raise Exception("msg must be bytes")

    signature = prvkey.sign(msg, hashfunc=sha256)

    return signature


# Verify a signature.
def verify_sig(pubkey_hex, sig, msg):
    # assert type of `data` is bytes
    if type(msg) != bytes:
        raise Exception("data must be bytes")

    vk = ecdsa.VerifyingKey.from_string(
        bytes.fromhex(pubkey_hex),
        curve=ecdsa.SECP256k1, 
        hashfunc=sha256
    )

    return vk.verify(sig, msg, hashfunc=sha256)


if __name__ == "__main__":
    w = Wallet.create_random()
    print("Wallet details:")
    print("  Address: {}".format(w.address()))
    print("  Private key: {}".format(w.prvkey_str()))
    print("  Public key: {}".format(w.pubkey_str()))

    print()
    msg = b"hello world"
    # sig = w.sign(msg)
    sig = sign_msg(w.prvkey, msg)
    print("Signature: {}".format(sig.hex()))

    is_valid = verify_sig(w.pubkey_str(), sig, msg)
    print("Verify signature: {}".format(is_valid))

    # print()
    # print("Recovering public key from signature...")
    # recovered_pubkey = recover_pubkey(sig, msg)
    # print("Recovered Public Key:", recovered_pubkey.to_string().hex())
        
