package main

import (
	"net/http"
	"sync"
	"time"
)

// cache is a fixed-TTL, in-memory store for proxied GET/HEAD responses.
// Entries are evicted lazily: a lookup past its TTL is treated as a miss and
// removed, there is no background sweeper.
type cache struct {
	ttl   time.Duration
	store sync.Map // cacheKey -> *cacheEntry
}

func newCache(ttl time.Duration) *cache {
	return &cache{ttl: ttl}
}

// cacheable reports whether a request/response pair is eligible for caching:
// only safe methods, and only successful responses.
func cacheable(method string, statusCode int) bool {
	switch method {
	case http.MethodGet, http.MethodHead:
	default:
		return false
	}

	return statusCode >= 200 && statusCode < 300
}

func (c *cache) get(key cacheKey) (cacheEntry, bool) {
	v, ok := c.store.Load(key)
	if !ok {
		return cacheEntry{}, false
	}

	entry := v.(*cacheEntry)
	if entry.expired(time.Now()) {
		// CompareAndDelete compares pointer identity, so it only removes
		// this exact entry, avoiding a race against a concurrent Store of a
		// fresher one for the same key.
		c.store.CompareAndDelete(key, v)
		return cacheEntry{}, false
	}

	return *entry, true
}

func (c *cache) set(key cacheKey, statusCode int, header http.Header, body []byte) {
	c.store.Store(key, &cacheEntry{
		statusCode: statusCode,
		header:     header.Clone(),
		body:       body,
		expiresAt:  time.Now().Add(c.ttl),
	})
}

// cacheKey identifies a cacheable request. Host is included because a single
// jamd instance can front multiple backends via host2backend.
type cacheKey struct {
	method string
	host   string
	path   string
	query  string
}

func newCacheKey(r *http.Request) cacheKey {
	return cacheKey{
		method: r.Method,
		host:   r.Host,
		path:   r.URL.Path,
		query:  r.URL.RawQuery,
	}
}

// cacheEntry is a stored response, ready to be replayed as-is.
type cacheEntry struct {
	statusCode int
	header     http.Header
	body       []byte
	expiresAt  time.Time
}

func (e cacheEntry) expired(now time.Time) bool {
	return now.After(e.expiresAt)
}

// writeEntry replays a cached entry onto w as if it had just come from the
// backend.
func writeEntry(w http.ResponseWriter, entry cacheEntry) error {
	dst := w.Header()
	for k, vv := range entry.header {
		dst[k] = vv
	}

	w.WriteHeader(entry.statusCode)
	_, err := w.Write(entry.body)
	return err
}
