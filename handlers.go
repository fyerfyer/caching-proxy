package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/gorilla/mux"
)

func (app *application) handleRequest(w http.ResponseWriter, r *http.Request, toUrl *url.URL) {
	cacheKey := r.URL.String()

	// try getting info from cache
	if cache, exist, _ := app.Cache.Get(cacheKey); exist {
		w.Header().Set("X-Cache", "HIT")
		w.Write([]byte(cache))
		return
	}

	// otherwise, forward the request
	proxy := httputil.NewSingleHostReverseProxy(toUrl)

	proxy.Director = func(req *http.Request) {
		req.Host = toUrl.Host
		req.URL.Scheme = toUrl.Scheme
		req.URL.Host = toUrl.Host
	}

	proxy.ModifyResponse = func(r *http.Response) error {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}

		app.Cache.Set(cacheKey, string(body), 5*time.Minute)
		w.Header().Set("X-Cache", "MISS")
		w.Write(body)
		return nil
	}

	proxy.ServeHTTP(w, r)
}

func (app *application) handleCacheAdd(w http.ResponseWriter, r *http.Request) {
	var item CacheItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if item.Key == "" || item.Value == "" || item.TTL == 0 {
		http.Error(w, "Key and value and tll are required", http.StatusBadRequest)
		return
	}

	item.Expiration = time.Now().Add(item.TTL)

	app.Cache.Set(item.Key, item.Value, item.TTL)
	app.Cache.items[item.Key] = &item

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Cache created successfully"}`))
}

func (app *application) handleCacheGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	if key == "" {
		http.Error(w, "Invalid key", http.StatusBadRequest)
		return
	}

	value, _, err := app.Cache.Get(key)
	if err != nil {
		http.Error(w, "Failed to fetch the cache", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, "Failed to encode post", http.StatusInternalServerError)
		return
	}
}

func (app *application) handleCacheDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	if key == "" {
		http.Error(w, "Invalid key", http.StatusBadRequest)
		return
	}

	err := app.Cache.Clear(key)
	if err != nil {
		http.Error(w, "Failed to delete the cache", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Cache delete successfully"}`))
}
