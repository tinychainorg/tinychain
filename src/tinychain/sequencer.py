import os
from transaction import decode_tx_yaml
from crypto import verify_sig

class FileSystemSequencer():
    def __init__(self, data_path):
        self.txs = []

        # Read all transactions from testnet-1/txs.
        for tx in os.listdir(data_path):
            tx = decode_tx_yaml(open(data_path + "/" + tx).read())

            # Verify signature.
            if not verify_sig(tx.from_acc, tx.sig, tx.envelope()):
                raise Exception("Invalid signature on tx: {}".format(tx))

            self.txs.append(tx)

    def tick(self):
        # Collect all incoming transactions.
        pass


class GitSequencer():
    def __init__(self):
        self.txs = []

    def tick(self):
        # Collect all incoming transactions.
        pass
