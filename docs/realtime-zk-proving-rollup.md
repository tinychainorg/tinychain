Realtime ZK proofing and rollup
===============================

I want tinychain to be zk provable. As such, we have full nakamoto consensus, involving hashcash proof-of-work, difficulty epochs, etc.

But all of the hashing / ecdsa is replaced with zk-friendly primitives:

- hashcash: SHA256 replaced with Poseidon
- signatures: ECDSA replaced with groth16 ZK signatures

What does this look like?

## ZK signatures.

Let's explain ZK signatures first of all. A ZK signature is what is used in privacy-based schemes like Tornado Cash, Zcash and Aztec's protocol. A public key K is defined as the hash of a secret value, S. A signature is a ZK proof which reveals knowledge of the secret value, H(S) == K. To spend a coin, we produce this proof, which is akin to a digital signature, as well as specifying the new public key it is locked to, which is a simple hash H(S), otherwise known as K. 

In the groth16 proving scheme, using BN252 as the field, the proof size is 3 field elements. log2(252) ~= 8 bytes, so a groth16 proof is 3*8=24 bytes. 

Using the Poseidon ZK-friendly hashing function, according to the benchmarks from their paper https://eprint.iacr.org/2019/458.pdf using the libsnark library: it takes 43.1ms to generate a proof, and 1.2ms to verify.

I've run some benchmarks using the tinychain Go implementation, which uses go's crypto/ecdsa package.

1 ECDSA sign - 213.833µs\
1 ECDSA verify - 125.459µs\
Signature size - 64 B

Comparing this against groth16 BN256 Poseidon signatures:

1 ZK sign - 43.1ms\
1 ZK verify - 1.2ms\
Signature size - 24 B

So roughly speaking, with a block size of 1MB:

Tx base size w/out signature = 91 B\
ECDSA tx size = 91 + 64 = 155 B\
ZK tx size = 91 + 24 = 115 B

Total ECDSA txs / 1 MB block = 1000*1000 / 155 = 6451\
Total ZK txs / 1 MB block = 1000*1000 / 115 = 8695

ECDSA sig verification time / block = 6451*125/1000 = 806ms\
ZK sig verification time / block = 8695*1.2 = 10,434ms = 1.04s

## Hashcash solution

The hashcash solution is another groth16 proof, again using the Poseidon hashing function, so an additional proof is only 24 bytes to attach to the block.

## Proving a block.

Now comes to proving a block. We can aggregate groth16 proofs using a variety of systems, but I'm just going to use SnarkPack as an example https://eprint.iacr.org/2021/529.pdf

"SnarkPack can aggregate 8192 proofs in 8.7s and verify them in 163ms"

The Snarkpack proofs involve 120 public inputs, so they are actually larger than our signature proofs which will involve just two (a public input is necessary to "return data" from the proof; commit to the public key we are verifying, thus exposing it). 

Nonetheless, we can use Snarkpack as a napkin estimate.

"[Snarkpack can] aggregate n Groth16 zkSNARKs with a O(log n) proof size and verifier time"

This might mean that miners can realtime generate proofs of blocks as they mine them.

1 proof = 24 B\
1 sig = 1 proof\
Max block size 1MB = 8695 ZK sigs / block

For reasoning's sake, instead of 8695 txs, let's consider Snarkpack's 8192 proofs, which is pretty close to our maximum number of ZK sigs in a block.

Time to prove 1 block = 8.7s\
Time to verify 1 block's aggregate proof = 163ms\
Size of block aggregated proof ~= 20kB (from paper)

## Recursively verifying blocks

Now we can prove 1 block within the bounds of "realtime" (8.7s) and the proof is very small (20kB). Could we verify the previous block, thus creating a single proof which verifies the entire chain? I think so.

I guess the next part of this is actually committing to the blockchain's state inside the block proof. Then you never need to recompute the state.

Reconstructing the state is actually very cheap - I ran some numbers last night and it took 9s to process ~1 years worth of transactions at full blocks of 1MB containing 6451 txs per block.

The big bottleneck is actually downloading the txs in order to compute the state. But if you didn't have to, if there was a ZK proof of the up-to-date chain state, then it would be O(N) cost where N is the number of UXTO's.

## State size.

The really good part about this is that means the chain can actually be...tiny.  Since rebuilding state after a reorg is so cheap, you don't need to store full state snapshots, you can just store state diffs and reapply them.  

So rough napkin estimates of state -   

number of UXTO's in bitcoin as of today = 64,000 https://blockchain.com/explorer/charts/utxo-count 

total size of a state leaf = 32B (account/pubkey) + 6B (balance uint64) = 38B  

state size = 64000*38B = 64000*38/1000/1000 = 2.432 MB

That's...pretty small

Although passing 2.432 MB as an input to a groth16 proof might be hard. We technically only need to pass in the 3 state leaves which are being modified - the from balance, the to balance, and the miner's balance (for fees).
