// Package config handles app configurations
package config

import (
	"cmp"
	"os"
)

const (
	cfgDBPath         = "DB_PATH"
	defaultDBPath     = "taskland.db"
	cfgServerAddr     = "SERVER_ADDR"
	defaultServerAddr = ":8080"
)

type Config struct {
	DBPath     string
	ServerAddr string
}

func Load() Config {
	var config Config
	config.DBPath = cmp.Or(os.Getenv(cfgDBPath), defaultDBPath)
	config.ServerAddr = cmp.Or(os.Getenv(cfgServerAddr), defaultServerAddr)
	return config
}
