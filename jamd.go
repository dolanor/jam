package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

type jamd struct {
	config config
}

func NewJamd(config config) *jamd {
	return &jamd{
		config: config,
	}
}

func (j *jamd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slog.Info("req uri", "uri", r.RequestURI)

	host := r.Host
	backend, ok := j.config.host2backend[host]
	if !ok {
		slog.Error("proxying: no backend configured for host", "host", host)
		http.Error(w, "no backend configured for host", http.StatusBadGateway)
		return
	}

	backendURL, err := url.Parse(backend)
	if err != nil {
		slog.Error("proxying: parsing backend url", "error", err, "backend", backend)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("cfg", "map", j.config.host2backend, "host", host, "backend", backend)

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
	w.WriteHeader(resp.StatusCode)

	n, err := io.Copy(w, resp.Body)
	if err != nil {
		slog.Error("proxying: copying", "error", err, "bytes", n)
	}
}
