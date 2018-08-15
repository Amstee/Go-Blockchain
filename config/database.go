package config

type DatabaseConfig struct {
	Port int
	Uri string
	DatabaseType string
	DatabaseFile string
}

var DbConf = new(DatabaseConfig)
