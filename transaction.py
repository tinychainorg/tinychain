from hashlib import sha256
import yaml

def parse_text_tx(txt):
    data = yaml.safe_load(txt)
    return Tx(data['from'], data['to'], data['data'], data['sig'])

class Tx:
    def __init__(self, from_acc, to_acc, data, sig):
        self.from_acc = from_acc
        self.to_acc = to_acc
        self.data = data
        self.sig = sig

    def parse(self, str):
        pass


if __name__ == "__main__":
    print(parse_text_tx(open("testnet-1/txs/1.yaml").read()))