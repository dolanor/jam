package cache

import (
	"net/http"
	"sync"
	"time"
)

// Cache is a fixed-TTL, in-memory store for proxied GET/HEAD responses.
// Entries are evicted lazily: a lookup past its TTL is treated as a miss and
// removed, there is no background sweeper.
type Cache struct {
	ttl   time.Duration
	store sync.Map // cacheKey -> *cacheEntry
}

func New(ttl time.Duration) *Cache {
	return &Cache{ttl: ttl}
}

// Cacheable reports whether a request/response pair is eligible for caching:
// only safe methods, and only successful responses.
func Cacheable(method string, statusCode int) bool {
	switch method {
	case http.MethodGet, http.MethodHead:
	default:
		return false
	}

	return statusCode >= 200 && statusCode < 300
}

func (c *Cache) Get(key Key) (CacheEntry, bool) {
	v, ok := c.store.Load(key)
	if !ok {
		return CacheEntry{}, false
	}

	entry := v.(*CacheEntry)
	if entry.expired(time.Now()) {
		// CompareAndDelete compares pointer identity, so it only removes
		// this exact entry, avoiding a race against a concurrent Store of a
		// fresher one for the same key.
		c.store.CompareAndDelete(key, v)
		return CacheEntry{}, false
	}

	return *entry, true
}

func (c *Cache) Set(key Key, statusCode int, header http.Header, body []byte) {
	c.store.Store(key, &CacheEntry{
		StatusCode: statusCode,
		Header:     header.Clone(),
		Body:       body,
		ExpiresAt:  time.Now().Add(c.ttl),
	})
}

// Key identifies a cacheable request. Host is included because a single
// jamd instance can front multiple backends via host2backend.
type Key struct {
	method string
	host   string
	path   string
	query  string
}

func NewKey(r *http.Request) Key {
	return Key{
		method: r.Method,
		host:   r.Host,
		path:   r.URL.Path,
		query:  r.URL.RawQuery,
	}
}

// CacheEntry is a stored response, ready to be replayed as-is.
type CacheEntry struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	ExpiresAt  time.Time
}

func (e CacheEntry) expired(now time.Time) bool {
	return now.After(e.ExpiresAt)
}
