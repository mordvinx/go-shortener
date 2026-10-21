package config

import "flag"

type Config struct {
	Address string
	BaseURL string
}

var ParsedOptions = new(Config)

func SetupFlags() {
	flag.StringVar(&ParsedOptions.Address, "a", ":8080", "Укажите адрес запуска HTTP-сервера (например, localhost:8888)")
	flag.StringVar(&ParsedOptions.BaseURL, "b", "http://localhost:8080/", "Укажите базовый URL для сокращенных ссылок")
}

func ParseFlags() {
	flag.Parse()
}
