package nakamoto

// A unique instance of the blockchain.
type BlockchainNetwork struct {
	GenesisBlock    RawBlock
	ConsensusConfig ConsensusConfig
	Name            string
}

func NewBlockchainNetwork(name string, conf ConsensusConfig) BlockchainNetwork {
	return BlockchainNetwork{
		Name:            name,
		GenesisBlock:    GetRawGenesisBlockFromConfig(conf),
		ConsensusConfig: conf,
	}
}

func GetNetworks() map[string]BlockchainNetwork {
	network_testnet1 := NewBlockchainNetwork("testnet1", ConsensusConfig{
		EpochLengthBlocks:       10,
		TargetEpochLengthMillis: 1000 * 60, // 1min, 1 block every 10s
		GenesisDifficulty:       HexStringToBigInt("0fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
		// https://serhack.me/articles/story-behind-alternative-genesis-block-bitcoin/ ;)
		GenesisParentBlockHash: HexStringToBytes32("000006b15d1327d67e971d1de9116bd60a3a01556c91b6ebaa416ebc0cfaa647"), // +1 for testnet
		MaxBlockSizeBytes:      2 * 1024 * 1024,                                                                        // 2MB
	})

	networks := map[string]BlockchainNetwork{
		"testnet1":   network_testnet1,
		"terrydavis": network_testnet1,
	}

	return networks
}
