package main

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func (app *application) Routes() http.Handler {
	toUrl := parseToUrl(app.Origin)
	// Without HEAD Method ver:("net/http")
	// mux := http.NewServeMux()
	// mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	app.handleRequest(w, r, toUrl)
	// })
	// mux.HandleFunc("/clear-cache", app.Cache.clearCacheHandler)
	// mux.HandleFunc("/api/cache/add", app.handleCacheAdd)
	// mux.HandleFunc("/api/cache/delete/{key}", app.handleCacheDelete)
	// mux.HandleFunc("/api/cache/get/{key}", app.handleCacheGet)
	// return mux
	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		app.handleRequest(w, r, toUrl)
	})
	r.HandleFunc("/clear-cache", app.Cache.clearCacheHandler)
	r.HandleFunc("/api/cache/add", app.handleCacheAdd)
	r.HandleFunc("/api/cache/delete/{key}", app.handleCacheDelete)
	r.HandleFunc("/api/cache/get/{key}", app.handleCacheGet)

	return r
}

func parseToUrl(addr string) *url.URL {
	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}

	toUrl, err := url.Parse(addr)
	if err != nil {
		log.Fatal(err)
	}

	return toUrl
}
