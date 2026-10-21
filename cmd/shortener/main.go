package main

import (
	"log"
	"net/http"

	"github.com/mordvinx/go-shortener/internal/config"
	"github.com/mordvinx/go-shortener/internal/handler"
	"github.com/mordvinx/go-shortener/internal/service"
)

func main() {
	config.SetupFlags()
	config.ParseFlags()

	h := handler.Handler{
		Base:      config.ParsedOptions.BaseURL,
		Shortener: &service.Shortener{},
	}

	if err := http.ListenAndServe(config.ParsedOptions.Address, h.Router()); err != nil {
		log.Fatal(err)
	}
}
