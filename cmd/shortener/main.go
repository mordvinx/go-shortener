package main

import (
	"log"
	"net/http"

	"github.com/mordvinx/go-shortener/internal/config"
	"github.com/mordvinx/go-shortener/internal/handler"
	"github.com/mordvinx/go-shortener/internal/repository"
	"github.com/mordvinx/go-shortener/internal/service"
)

func main() {
	cfg := config.ParseFlags()

	h := handler.Handler{
		Base:      cfg.Shortener.BaseURL,
		Shortener: service.NewShortener(repository.NewMemory()),
	}

	if err := http.ListenAndServe(cfg.Server.Address, h.Router()); err != nil {
		log.Fatal(err)
	}
}
