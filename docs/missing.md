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

> When a Bitcoin block is produced there are essentially two times involved:
> 
> The timestamp in the block header, put there by the miner
> The actual time the block was produced.
> 
> As it happens, there are some incentives for miners to lie about the time. For instance, nefarious miners could add a timestamp which is in the future. For example, if a block took 10 minutes to produce, miners could claim it took them 15 minutes, by adding a timestamp 5 minutes into the future. If this pattern of adding 5 minutes is continued throughout a two week difficulty adjustment period, it would look like the average block time was 15 minutes, when in reality it was shorter than this. The difficulty could then adjust downwards in the next period, increasing mining revenue due to faster block times. Of course, the problem with this approach is that the Bitcoin clock continues to move further and further out of line with the real time.
> 
> To resolve or mitigate the above issue, Bitcoin has two mechanisms to protect against miners manipulating the timestamp:
> 
> **Median Past Time (MPT) Rule** – The timestamp must be further forwards than the median of the last eleven blocks. The median of eleven blocks implies six blocks could be re-organised and time would still not move backwards, which one can argue is reasonably consistent with the example, provided in Meni Rosenfeld’s 2012 paper, that six confirmations are necessary to decrease the probability of success below 0.1%, for an attacker with 10% of the network hashrate.
> 
> **Future Block Time Rule** – The timestamp cannot be more than 2 hours in the future based on the MAX_FUTURE_BLOCK_TIME constant, relative to the median time from the node’s peers. The maximum allowed gap between the time provided by the nodes and the local system clock is 90 minutes, another safeguard. It should be noted that unlike the MPT rule above, this is not a full consensus rule. Blocks with a timestamp too far in the future are not invalid, they can become valid as time moves forwards.
> 
> Rule number one ensures that the blockchain continues to move forwards in time and rule number two ensures that the chain does not move too far forwards. These time protection rules are not perfect, for example miners could still move the timestamps forward by producing timestamps in the future, within a two week period, however the impact of this would be limited.
>
> $ 2 hours / 2 weeks = 0.6% $
>
> As the above ratio illustrates, since two hours is only a small fraction of two weeks, the impact this manipulation has on network reliability and mining profitability may be limited. This is the equivalent of a reduction in the time between blocks from 10 minutes to 9 minutes and 54 seconds, in the two weeks after the difficulty adjustment. In addition to this, it is only a one-off change, as once the two-hour time shift has occurred, it cannot occur again, without first going backwards. At the same time, the miner may want to include a margin of safety before shifting forwards two hours, to reduce the risk of the block being rejected by the network.
> 
>

