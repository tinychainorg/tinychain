Missing pieces
==============

## ECDSA public key recovery

In Ethereum/Bitcoin, we can recover the public key from a signature.

The way this works - ECDSA public keys have two points (R1, R2), and on each point there is an (X,Y). For any given signature, you can recover two public keys - since ECDSA is based on squaring and there are two solutions to the square root (+ve and -ve). By encoding a value when you create the signature which specifies the parity (positive/negative) of one of these points, you can reliably recover the correct public key from the signature.

https://learnmeabitcoin.com/technical/keys/public-key/#:~:text=A%20compressed%20public%20key%20is,coordinate%20is%20even%20or%20odd.

https://en.bitcoin.it/wiki/Message_signing#ECDSA_signing,_with_P2PKH_compressed_addresses

I had some trouble implementing this.

## Timestamp protections.

https://blog.bitmex.com/bitcoins-block-timestamp-protection-rules/

