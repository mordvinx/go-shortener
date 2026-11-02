package config

import "flag"

type ServerConfig struct {
	Address string
}

type ShortenerConfig struct {
	BaseURL string
}

type Config struct {
	Server    ServerConfig
	Shortener ShortenerConfig
}

func ParseFlags() *Config {
	parsedOptions := new(Config)

	flag.StringVar(&parsedOptions.Server.Address, "a", ":8080", "Укажите адрес запуска HTTP-сервера (например, localhost:8888)")
	flag.StringVar(&parsedOptions.Shortener.BaseURL, "b", "http://localhost:8080/", "Укажите базовый URL для сокращенных ссылок")
	flag.Parse()

	return parsedOptions
}
