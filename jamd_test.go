package main

import (
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
)

func TestJamdServeHTTP(t *testing.T) {
	cfg := config{}
	j := NewJamd(cfg)
	s := httptest.NewServer(j)

	c := s.Client()
	resp, err := c.Get(fmt.Sprintf("%s/%s", s.URL, "/"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(body))
}
