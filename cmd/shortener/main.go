package main

import (
	"log"
	"net/http"

	"go.uber.org/zap"

	"github.com/mordvinx/go-shortener/internal/config"
	"github.com/mordvinx/go-shortener/internal/handler"
	"github.com/mordvinx/go-shortener/internal/middleware"
	"github.com/mordvinx/go-shortener/internal/repository"
	"github.com/mordvinx/go-shortener/internal/service"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = logger.Sync() }()

	cfg := config.ParseFlags()
	config.ParseEnv(cfg)

	store, err := repository.NewFile(cfg.Storage.FilePath)
	if err != nil {
		log.Fatal(err)
	}

	h := handler.Handler{
		Base:      cfg.Shortener.BaseURL,
		Shortener: service.NewShortener(store, store.NextCounter()),
	}

	if err := http.ListenAndServe(cfg.Server.Address, middleware.Logging(logger)(middleware.Gzip(h.Router()))); err != nil {
		log.Fatal(err)
	}
}
