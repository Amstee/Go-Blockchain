package config

type BlockchainConf struct {
	Version string
	CheckSumLen int
}

var BlockchainConfig = new(BlockchainConf)