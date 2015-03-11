package server

import (
	"net/http"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/server/heroku"
)

var DefaultOptions = Options{
	Heroku: heroku.DefaultOptions,
}

type Options struct {
	Heroku heroku.Options
}

func New(e *empire.Empire, options Options) http.Handler {
	r := mux.NewRouter()

	// Mount the heroku api
	h := heroku.New(e, options.Heroku)
	r.Headers("Accept", heroku.AcceptHeader).Handler(h)

	n := negroni.Classic()
	n.UseHandler(r)

	return n
}
