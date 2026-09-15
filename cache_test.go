package main

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
		got := cacheable(tt.method, tt.statusCode)
		if got != tt.want {
			t.Errorf("cacheable(%q, %d) = %v, want %v", tt.method, tt.statusCode, got, tt.want)
		}
	}
}

func TestCacheSetGet(t *testing.T) {
	c := newCache(time.Minute)

	key := cacheKey{method: http.MethodGet, host: "example.com", path: "/foo", query: ""}
	header := http.Header{"Content-Type": []string{"text/plain"}}
	body := []byte("hello")

	if _, ok := c.get(key); ok {
		t.Fatal("expected miss before set")
	}

	c.set(key, http.StatusOK, header, body)

	entry, ok := c.get(key)
	if !ok {
		t.Fatal("expected hit after set")
	}
	if entry.statusCode != http.StatusOK {
		t.Errorf("statusCode = %d, want %d", entry.statusCode, http.StatusOK)
	}
	if string(entry.body) != string(body) {
		t.Errorf("body = %q, want %q", entry.body, body)
	}
	if got := entry.header.Get("Content-Type"); got != "text/plain" {
		t.Errorf("header Content-Type = %q, want %q", got, "text/plain")
	}
}

func TestCacheSetClonesHeader(t *testing.T) {
	c := newCache(time.Minute)
	key := cacheKey{method: http.MethodGet, host: "example.com", path: "/foo"}

	header := http.Header{"X-Test": []string{"original"}}
	c.set(key, http.StatusOK, header, []byte("body"))

	// Mutating the caller's header after set must not affect the cached
	// entry: cache.set must clone, not alias, the header.
	header.Set("X-Test", "mutated")

	entry, ok := c.get(key)
	if !ok {
		t.Fatal("expected hit")
	}
	if got := entry.header.Get("X-Test"); got != "original" {
		t.Errorf("cached header X-Test = %q, want %q (should not observe caller mutation)", got, "original")
	}
}

func TestCacheExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := newCache(10 * time.Millisecond)
		key := cacheKey{method: http.MethodGet, host: "example.com", path: "/foo"}

		c.set(key, http.StatusOK, http.Header{}, []byte("body"))

		if _, ok := c.get(key); !ok {
			t.Fatal("expected hit immediately after set")
		}

		time.Sleep(20 * time.Millisecond)

		if _, ok := c.get(key); ok {
			t.Fatal("expected miss after ttl expiry")
		}
	})
}

func TestCacheKeyDistinguishesRequests(t *testing.T) {
	c := newCache(time.Minute)

	base := cacheKey{method: http.MethodGet, host: "example.com", path: "/foo", query: ""}
	c.set(base, http.StatusOK, http.Header{}, []byte("base"))

	variants := []cacheKey{
		{method: http.MethodHead, host: "example.com", path: "/foo", query: ""},
		{method: http.MethodGet, host: "other.com", path: "/foo", query: ""},
		{method: http.MethodGet, host: "example.com", path: "/bar", query: ""},
		{method: http.MethodGet, host: "example.com", path: "/foo", query: "q=1"},
	}

	for _, v := range variants {
		if _, ok := c.get(v); ok {
			t.Errorf("expected miss for distinct key %+v", v)
		}
	}
}

func TestNewCacheKeyFromRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://example.com/foo?bar=baz", nil)
	r.Host = "example.com"

	got := newCacheKey(r)
	want := cacheKey{method: http.MethodGet, host: "example.com", path: "/foo", query: "bar=baz"}

	if got != want {
		t.Errorf("newCacheKey() = %+v, want %+v", got, want)
	}
}

func TestWriteEntry(t *testing.T) {
	entry := cacheEntry{
		statusCode: http.StatusOK,
		header:     http.Header{"X-Cache": []string{"HIT"}},
		body:       []byte("cached body"),
	}

	rec := httptest.NewRecorder()
	if err := writeEntry(rec, entry); err != nil {
		t.Fatalf("writeEntry() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("X-Cache"); got != "HIT" {
		t.Errorf("header X-Cache = %q, want %q", got, "HIT")
	}
	if got := rec.Body.String(); got != "cached body" {
		t.Errorf("body = %q, want %q", got, "cached body")
	}
}
