package main

import (
	"database/sql"
	"errors"
	"net/http"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// const (
// 	Location = "/var/cache/"
// )

type Cache struct {
	items map[string]*CacheItem
	mu    sync.Mutex
	DB    *sql.DB
}

func (c *Cache) newCache(DB *sql.DB) *Cache {
	go c.startExpirationHandler(5 * time.Minute)
	return &Cache{
		items: make(map[string]*CacheItem),
		DB:    DB,
	}
}

func (c *Cache) startExpirationHandler(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.ClearExpired()
		}
	}
}

func (c *Cache) Get(key string) (string, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// first get the cache from memory
	item, exists := c.items[key]
	if exists && !time.Now().After(item.Expiration) {
		return item.Value, true, nil
	}

	// if couldn't found, we try finding it in the sql
	stmt := `SELECT key, value, ttl, expiration 
	WHERE key = ?`
	row := c.DB.QueryRow(stmt, key)
	err := row.Scan(&item.Key, &item.Value, &item.TTL, &item.Expiration)
	if err != nil {
		return "", false, err
	}

	// check if it's expired
	if time.Now().After(item.Expiration) {
		return "", false, nil
	}

	// we put it into the memory,
	// so that we don't neet to visit database everytime
	c.items[key] = item
	return item.Value, true, nil
}

func (c *Cache) Set(key string, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists, _ := c.Get(key); exists {
		return errors.New("The cache has already existed!")
	}

	item := CacheItem{
		Key:        key,
		Value:      value,
		TTL:        ttl,
		Expiration: time.Now().Add(ttl),
	}

	// store it into the memory
	c.items[key] = &item

	// store it into the database
	stmt := `INSERT INTO cache (key, value, ttl, expiration)
	VALUES(?, ?, ?, ?)`
	_, err := c.DB.Exec(stmt, item.Key, item.Value, item.TTL, item.Expiration)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cache) Clear(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists, _ := c.Get(key); !exists {
		return errors.New("Cache doesn't exist!")
	}

	delete(c.items, key)
	// delete from database
	stmt := `DELETE FROM cache WHERE key = ?`
	_, err := c.DB.Exec(stmt, key)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cache) clearCacheHandler(w http.ResponseWriter, r *http.Request) {
	c.items = make(map[string]*CacheItem)
	// delete all rows in the database
	stmt := `DELETE * FROM cache`
	_, err := c.DB.Exec(stmt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to clear cache table in the database"))
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Cache cleared successfully"))
}

func (c *Cache) ClearExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, item := range c.items {
		if time.Now().After(item.Expiration) {
			c.Clear(key)
		}
	}
}
