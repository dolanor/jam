package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
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
	hhost := r.Header.Get("Host")
	slog.Info("cfg", "map", j.config.host2backend, "host", host, "hhost", hhost)

	for k, v := range r.Header {
		for _, vv := range v {
			fmt.Println(k, ":", vv)
		}
	}

	//w.Write([]byte("Hello World from:" + host))
	c := http.DefaultClient
	resp, err := c.Do(r)
	if err != nil {
		slog.Error("proxying", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	n, err := io.Copy(w, resp.Body)
	if err != nil {
		slog.Error("proxying: copying", "error", err, "bytes", n)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
