package config

import "os"

func ParseEnv(cfg *Config) {
	if address, ok := os.LookupEnv("SERVER_ADDRESS"); ok && address != "" {
		cfg.Server.Address = address
	}

	if base, ok := os.LookupEnv("BASE_URL"); ok && base != "" {
		cfg.Shortener.BaseURL = base
	}
}
