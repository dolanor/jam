package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
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
	//req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", proxy.URL, "dolanor"), nil)
	req, err := http.NewRequest(http.MethodGet, proxy.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = backendURL.Host

	slog.Info("TEST", "cfg", cfg.host2backend, "proxyURL", proxy.URL, "beURL", backend.URL)

	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	t.Log("resp status:", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)

	if got != want {
		t.Fatalf("\n\tgot : %v\n\twant: %v", got, want)
	}
}
