package main

import "net/http"

type jamd struct {
	config config
}

func NewJamd(config config) *jamd {
	return &jamd{
		config: config,
	}
}

func (j *jamd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}
