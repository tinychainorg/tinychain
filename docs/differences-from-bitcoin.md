Differences from Bitcoin
========================

Design simplifications:

 * Work, as is calculated for the purposes of determining the longest chain, is based on the hashcash `guess` rather than adding together the block difficulty. This seems to be a more precise estimate of the miner's hashpower. Intuitionally, you can deduce that if a miner has solved a hashcash puzzle with a large number of zeroes, bearing any insane technological advances like quantum computing it may have over other nodes, it's probably likely it has expended a lot of work on it.
 * The accumulated work is stored in consensus as part of the block hashing envelope. This is for one reason only - during state syncing, a node will ask all its peers for their latest tips. One very cheap way to determine the longest chain is by checking the `ParentTotalWork` field, which is updated as part of consensus, and summing it with the work demonstrated in that block's POW solution. I'm not sure if this has been tried before but I believe it can work.

Differences:

 * Transactions do not have a VM environment.
 * Bitcoin features protection against quantum computing attacks, since coins are locked to a preimage of a public key (RIPEMD(SHA256(pubkey))) using P2PKH rather than locked to a public key itself. 

Missing efficiencies:

 * The difficulty target is represented as `[32]bytes`; it is uncompressed. There is no `nBits` or custom mantissa.
 * Transaction signatures are in their uncompressed ECDSA form. They are `[65]bytes`, which includes the ECDSA signature type of `0x4`. There is no ECDSA signature recovery.
 * Pubkeys are used in their raw form instead of their hash as in Bitcoin. 
