package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

func TestJamdServeHTTP(t *testing.T) {
	cfg := config{
		host2backend: map[string]string{},
	}

	want := "Hello backend"
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(want))
	})
	backend := httptest.NewServer(mux)
	defer backend.Close()

	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	cfg.host2backend[backendURL.Host] = backend.URL

	j := NewJamd(cfg)
	proxy := httptest.NewServer(j)
	defer proxy.Close()

	c := proxy.Client()
	req, err := http.NewRequest(http.MethodGet, proxy.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = backendURL.Host

	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)

	if got != want {
		t.Fatalf("\n\tgot : %v\n\twant: %v", got, want)
	}
}

func TestJamdServeHTTPCacheHitSkipsBackend(t *testing.T) {
	cfg := config{
		host2backend: map[string]string{},
		cacheEnabled: true,
		cacheTTL:     time.Minute,
	}

	want := "Hello backend"
	var hits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte(want))
	})
	backend := httptest.NewServer(mux)
	defer backend.Close()

	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	cfg.host2backend[backendURL.Host] = backend.URL

	j := NewJamd(cfg)
	proxy := httptest.NewServer(j)
	defer proxy.Close()

	c := proxy.Client()

	for i := range 2 {
		req, err := http.NewRequest(http.MethodGet, proxy.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = backendURL.Host

		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}

		if got := string(body); got != want {
			t.Fatalf("request %d:\n\tgot : %v\n\twant: %v", i, got, want)
		}
	}

	if got := hits.Load(); got != 1 {
		t.Fatalf("backend hits = %d, want 1 (second request should have been served from cache)", got)
	}
}

func TestJamdServeHTTPCacheDisabledHitsBackendEveryTime(t *testing.T) {
	cfg := config{
		host2backend: map[string]string{},
	}

	var hits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte("Hello backend"))
	})
	backend := httptest.NewServer(mux)
	defer backend.Close()

	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	cfg.host2backend[backendURL.Host] = backend.URL

	j := NewJamd(cfg)
	proxy := httptest.NewServer(j)
	defer proxy.Close()

	c := proxy.Client()

	for range 2 {
		req, err := http.NewRequest(http.MethodGet, proxy.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = backendURL.Host

		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}

	if got := hits.Load(); got != 2 {
		t.Fatalf("backend hits = %d, want 2 (caching disabled, backend should be hit every time)", got)
	}
}
