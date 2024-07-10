Missing pieces
==============

## ECDSA public key recovery

In Ethereum/Bitcoin, we can recover the public key from a signature.

The way this works - ECDSA public keys have two points (R1, R2), and on each point there is an (X,Y). For any given signature, you can recover two public keys - since ECDSA is based on squaring and there are two solutions to the square root (+ve and -ve). By encoding a value when you create the signature which specifies the parity (positive/negative) of one of these points, you can reliably recover the correct public key from the signature.

I had some trouble implementing this.
