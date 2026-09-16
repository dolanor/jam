package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

type jamd struct {
	config config
	cache  *cache
}

func NewJamd(config config) *jamd {
	j := &jamd{
		config: config,
	}

	if config.cacheEnabled {
		j.cache = newCache(config.cacheTTL)
	}

	return j
}

func (j *jamd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	backend, ok := j.config.host2backend[host]
	if !ok {
		slog.Error("proxying: no backend configured for host", "host", host)
		http.Error(w, "no backend configured for host", http.StatusBadGateway)
		return
	}

	key := newCacheKey(r)
	if j.cache != nil {
		// cache hit
		if entry, hit := j.cache.get(key); hit {
			if err := writeEntry(w, entry); err != nil {
				slog.Error("proxying: writing cached response", "error", err)
			}
			return
		}
	}

	backendURL, err := url.Parse(backend)
	if err != nil {
		slog.Error("proxying: parsing backend url", "error", err, "backend", backend)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// A server *http.Request can't be reused as a client request: RequestURI
	// must be empty and URL must be absolute (scheme+host), so build a fresh
	// outbound request targeting the backend instead of reusing r directly.
	outReq := r.Clone(r.Context())
	outReq.RequestURI = ""
	outReq.URL.Scheme = backendURL.Scheme
	outReq.URL.Host = backendURL.Host
	outReq.Host = backendURL.Host

	resp, err := http.DefaultClient.Do(outReq)
	if err != nil {
		slog.Error("proxying", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("proxying: reading response body", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	store := j.cache != nil && cacheable(r.Method, resp.StatusCode)
	if store {
		j.cache.set(key, resp.StatusCode, w.Header(), body)
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := w.Write(body); err != nil {
		slog.Error("proxying: writing response", "error", err)
	}
}
