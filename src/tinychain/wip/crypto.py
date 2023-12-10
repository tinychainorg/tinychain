# WIP
# Not working.
# - need to fix recovery ID

import ecdsa
from ecdsa import VerifyingKey, BadSignatureError, SECP256k1
from ecdsa.util import sigdecode_der
from hashlib import sha256



# Sign a message using a private key.
def sign_msg(prvkey, msg):
    # assert type of `msg` is bytes
    if type(msg) != bytes:
        raise Exception("msg must be bytes")

    message_hash = sha256(msg).digest()
    signature = prvkey.sign_deterministic(message_hash, hashfunc=sha256)

    # Create a full signature including the recovery ID (`v`).
    # Get the recovery ID (v value) based on the parity of the Y-coordinate of the public key
    r, s = signature[:32], signature[32:]

    # A recovery ID can have the values 0..3 depending on the following conditions:
    # Is R.y even and R.x less than the curve order n: recovery_id := 0
    # Is R.y odd and R.x less than the curve order n: recovery_id := 1
    # Is R.y even and R.x more than the curve order n: recovery_id := 2
    # Is R.y odd and R.x more than the curve order n: recovery_id := 3
    # [1] https://ethereum.stackexchange.com/a/118342 

    # R
    R = prvkey.get_verifying_key().pubkey
    recovery_id = recovery_id_for_R(R) + 27
    print(f"encoded recovery_id: {recovery_id}")

    # recovery_id = 27 + (prvkey.get_verifying_key().pubkey.point.y() & 1)
    full_signature = r + s + bytes([recovery_id])

    return full_signature

# 0 - y is even, x is finite
# 1 - y is odd, x is finite
# 2 - y is even, x is too large
# 3 - y is odd, x is too large
def recovery_id_for_R(R):
    print(f"R.point.y(): {R.point.y()}")
    print(f"R.point.x(): {R.point.x()}")
    if R.point.y() % 2 == 0 and R.point.x() < SECP256k1.order:
        return 0
    elif R.point.y() % 2 == 1 and R.point.x() < SECP256k1.order:
        return 1
    elif R.point.y() % 2 == 0 and R.point.x() > SECP256k1.order:
        return 2
    elif R.point.y() % 2 == 1 and R.point.x() > SECP256k1.order:
        return 3

# Recover the public key from a signature.
# In Ethereum, this is called `ecrecover`.
def recover_pubkey(sig, msg):
    assert len(sig) == 65
    
    # Extract r, s, and v values from the signature
    r = int.from_bytes(sig[0:32], byteorder='big')
    s = int.from_bytes(sig[32:64], byteorder='big')
    v = sig[64]
    
    # Generate the message hash
    message_hash = sha256(msg).digest()

    recovered_keys = VerifyingKey.from_public_key_recovery(
        sig[0:64],
        message_hash,
        curve=ecdsa.SECP256k1, 
        hashfunc=sha256
    )

    for vk in recovered_keys:
        print("Recovered public key: {}".format(vk.to_string().hex()))

    # # Choose the correct public key based on the recovered v value
    # recovered_key = recovered_keys[0] if v == 27 else recovered_keys[1]

    print(f"v: {v}")

    # Check each recovered key against the v value to choose the correct key
    for vk in recovered_keys:
        R = vk.pubkey
        recovery_id = recovery_id_for_R(R) + 27
        print(f"recovery_id: {recovery_id}")
        # if recovery_id == v:
        #     print("Recovered public key: {}".format(vk.to_string().hex()))
        #     return vk


    # print("Recovered public key: {}".format(recovered_key.to_string().hex()))
    return None

# Verify a signature.
def verify_sig(pubkey_hex, sig, msg):
    assert len(sig) == 65

    # assert type of `data` is bytes
    if type(msg) != bytes:
        raise Exception("data must be bytes")

    vk = ecdsa.VerifyingKey.from_string(
        bytes.fromhex(pubkey_hex), 
        curve=ecdsa.SECP256k1, 
        hashfunc=sha256
    )

    message_hash = sha256(msg).digest()

    return vk.verify(sig[0:64], message_hash)