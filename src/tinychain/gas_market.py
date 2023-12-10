

class GasMarket:
    def __init__(self):
        pass
    
    # Exchange rate of tokens (e.g. ETH) to gas units.
    def token_to_gas(self, token_amount):
        return token_amount * 1_000_000