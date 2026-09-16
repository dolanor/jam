package cache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"
)

func TestCacheableMethod(t *testing.T) {
	tests := []struct {
		method     string
		statusCode int
		want       bool
	}{
		{http.MethodGet, http.StatusOK, true},
		{http.MethodHead, http.StatusOK, true},
		{http.MethodGet, http.StatusCreated, true},
		{http.MethodGet, http.StatusNotFound, false},
		{http.MethodGet, http.StatusInternalServerError, false},
		{http.MethodPost, http.StatusOK, false},
		{http.MethodPut, http.StatusOK, false},
		{http.MethodDelete, http.StatusOK, false},
	}

	for _, tt := range tests {
		got := Cacheable(tt.method, tt.statusCode)
		if got != tt.want {
			t.Errorf("cacheable(%q, %d) = %v, want %v", tt.method, tt.statusCode, got, tt.want)
		}
	}
}

func TestCacheSetGet(t *testing.T) {
	c := New(time.Minute)

	key := Key{method: http.MethodGet, host: "example.com", path: "/foo", query: ""}
	header := http.Header{"Content-Type": []string{"text/plain"}}
	body := []byte("hello")

	if _, ok := c.Get(key); ok {
		t.Fatal("expected miss before set")
	}

	c.Set(key, http.StatusOK, header, body)

	entry, ok := c.Get(key)
	if !ok {
		t.Fatal("expected hit after set")
	}
	if entry.StatusCode != http.StatusOK {
		t.Errorf("statusCode = %d, want %d", entry.StatusCode, http.StatusOK)
	}
	if string(entry.Body) != string(body) {
		t.Errorf("body = %q, want %q", entry.Body, body)
	}
	if got := entry.Header.Get("Content-Type"); got != "text/plain" {
		t.Errorf("header Content-Type = %q, want %q", got, "text/plain")
	}
}

func TestCacheSetClonesHeader(t *testing.T) {
	c := New(time.Minute)
	key := Key{method: http.MethodGet, host: "example.com", path: "/foo"}

	header := http.Header{"X-Test": []string{"original"}}
	c.Set(key, http.StatusOK, header, []byte("body"))

	// Mutating the caller's header after set must not affect the cached
	// entry: cache.set must clone, not alias, the header.
	header.Set("X-Test", "mutated")

	entry, ok := c.Get(key)
	if !ok {
		t.Fatal("expected hit")
	}
	if got := entry.Header.Get("X-Test"); got != "original" {
		t.Errorf("cached header X-Test = %q, want %q (should not observe caller mutation)", got, "original")
	}
}

func TestCacheExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New(10 * time.Millisecond)
		key := Key{method: http.MethodGet, host: "example.com", path: "/foo"}

		c.Set(key, http.StatusOK, http.Header{}, []byte("body"))

		if _, ok := c.Get(key); !ok {
			t.Fatal("expected hit immediately after set")
		}

		time.Sleep(20 * time.Millisecond)

		if _, ok := c.Get(key); ok {
			t.Fatal("expected miss after ttl expiry")
		}
	})
}

func TestCacheKeyDistinguishesRequests(t *testing.T) {
	c := New(time.Minute)

	base := Key{method: http.MethodGet, host: "example.com", path: "/foo", query: ""}
	c.Set(base, http.StatusOK, http.Header{}, []byte("base"))

	variants := []Key{
		{method: http.MethodHead, host: "example.com", path: "/foo", query: ""},
		{method: http.MethodGet, host: "other.com", path: "/foo", query: ""},
		{method: http.MethodGet, host: "example.com", path: "/bar", query: ""},
		{method: http.MethodGet, host: "example.com", path: "/foo", query: "q=1"},
	}

	for _, v := range variants {
		if _, ok := c.Get(v); ok {
			t.Errorf("expected miss for distinct key %+v", v)
		}
	}
}

func TestNewCacheKeyFromRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://example.com/foo?bar=baz", nil)
	r.Host = "example.com"

	got := NewKey(r)
	want := Key{method: http.MethodGet, host: "example.com", path: "/foo", query: "bar=baz"}

	if got != want {
		t.Errorf("newCacheKey() = %+v, want %+v", got, want)
	}
}
