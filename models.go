package main

import "time"

type CacheItem struct {
	Key        string        `json:"key"`
	Value      string        `json:"value"`
	TTL        time.Duration `json:"ttl"`
	Expiration time.Time     `json:"expiration"`
}
