Idea: UXTO
==========

The state of a Bitcoin blockchain is the UXTO set - the set of unspent transaction outputs. Each UXTO is a "one-use" piece of state - a transaction consumes inputs (UXTO's) and sends their value to different outputs (which is where UXTO's are created).

There are two implications to this model:

 1. Nodes must store an entire UXTO set.
 2. In an account-based model, nodes must check for transaction uniqueness via account nonces.

Here is an idea for simplifying this:

 1. Each block commits to a sparse merkle tree containing the state.
 2. The state consists of UXTO's which are identified by key and value.
 3. It is simple for light clients to verify the validity of a UXTO based on the current state of this tree, since sparse merkle trees allow for inclusion/exclusion proofs.
 4. But this incurs a large cost - in order to spend a UXTO, users must produce a leaf proof - which is O(N log N) leaves in space. This times 8192 txs (the full size of a 2MB block?) means quite a lot more data storage. Each hash is 32 bytes, so at least 16*32 is a lot of bytes added.
 5. So - what we can do. Nodes can prove these leaves into a ZK proof - thus reducing the cost down to O(log N) where N is the number of computational steps - and close to constant size if proof aggregation is used.

https://eprint.iacr.org/2019/611

https://bitcoinops.org/en/topics/utreexo/