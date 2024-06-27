Differences from Bitcoin
========================

 * The difficulty target is represented as `[32]bytes`; it is uncompressed. There is no `nBits` or custom mantissa.
 * Transaction signatures are in their uncompressed ECDSA form. They are `[65]bytes`, which includes the ECDSA signature type of `0x4`. There is no ECDSA signature recovery.
 * Transactions do not have a VM environment. Although this chain runs a UXTO model, transactions are very simple transfers of coins with a signature allowed.

