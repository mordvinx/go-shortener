package handler

import "github.com/mordvinx/go-shortener/internal/service"

type Handler struct {
	Base      string
	Shortener *service.Shortener
}
