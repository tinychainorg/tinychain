import ecdsa
from ecdsa import VerifyingKey, BadSignatureError, SECP256k1
from ecdsa.util import sigdecode_der
from hashlib import sha256

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