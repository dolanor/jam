package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
	be := httptest.NewServer(mux)
	cfg.host2backend["app.example.com"] = be.URL

	j := NewJamd(cfg)
	proxy := httptest.NewServer(j)

	c := proxy.Client()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", proxy.URL, "dolanor"), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Host", "app.example.com")

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
