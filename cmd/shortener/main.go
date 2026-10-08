package main

import (
	"log"
	"net/http"

	"github.com/mordvinx/go-shortener/internal/handler"
	"github.com/mordvinx/go-shortener/internal/service"
)

func main() {
	h := handler.Handler{
		Shortener: &service.Shortener{},
	}

	if err := http.ListenAndServe(":8080", h.Router()); err != nil {
		log.Fatal(err)
	}
}
