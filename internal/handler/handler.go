package handler

type Shortener interface {
	Shorten(url string) (string, error)
	Resolve(id string) (string, error)
}

type Handler struct {
	Base      string
	Shortener Shortener
}
