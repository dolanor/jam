package jam

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dolanor/jam/internal/cache"
	"github.com/dolanor/jam/internal/config"
)

func TestJamdServeHTTP(t *testing.T) {
	cfg := config.Config{
		Host2Backend: map[string]string{},
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
	cfg.Host2Backend[backendURL.Host] = backend.URL

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

func TestJamdServeHTTPCacheHit(t *testing.T) {
	cases := map[string]struct {
		config   config.Config
		wantHits int32
	}{
		"with cache": {
			config: config.Config{
				Host2Backend: map[string]string{},
				CacheEnabled: true,
				CacheTTL:     time.Minute,
			},
			wantHits: 1,
		},
		"without cache": {
			config: config.Config{
				Host2Backend: map[string]string{},
			},
			wantHits: 2,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := c.config

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
			cfg.Host2Backend[backendURL.Host] = backend.URL

			j := NewJamd(cfg)
			proxy := httptest.NewServer(j)
			defer proxy.Close()

			cl := proxy.Client()

			for i := range 2 {
				req, err := http.NewRequest(http.MethodGet, proxy.URL, nil)
				if err != nil {
					t.Fatal(err)
				}
				req.Host = backendURL.Host

				resp, err := cl.Do(req)
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

			if got := hits.Load(); got != c.wantHits {
				t.Fatalf("backend hits = %d, want 1 (second request should have been served from cache)", got)
			}
		})
	}
}

func TestWriteEntry(t *testing.T) {
	entry := cache.CacheEntry{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Cache": []string{"HIT"}},
		Body:       []byte("cached body"),
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
