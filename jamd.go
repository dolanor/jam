package jam

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/dolanor/jam/cache"
	"github.com/dolanor/jam/config"
)

type jamd struct {
	config config.Config
	cache  *cache.Cache
}

func NewJamd(config config.Config) *jamd {
	j := &jamd{
		config: config,
	}

	if config.CacheEnabled {
		j.cache = cache.New(config.CacheTTL)
	}

	return j
}

func (j *jamd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	backend, ok := j.config.Host2Backend[host]
	if !ok {
		slog.Error("proxying: no backend configured for host", "host", host)
		http.Error(w, "no backend configured for host", http.StatusBadGateway)
		return
	}

	key := cache.NewKey(r)
	if j.cache != nil {
		// cache hit
		if entry, hit := j.cache.Get(key); hit {
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

	j.saveToCache(key, r.Method, resp.StatusCode, w.Header(), body)

	w.WriteHeader(resp.StatusCode)
	if _, err := w.Write(body); err != nil {
		slog.Error("proxying: writing response", "error", err)
	}
}

func (j *jamd) saveToCache(key cache.Key, method string, statusCode int, header http.Header, body []byte) {
	store := j.cache != nil && cache.Cacheable(method, statusCode)
	if store {
		j.cache.Set(key, statusCode, header, body)
	}
}

// writeEntry replays a cached entry onto w as if it had just come from the
// backend.
func writeEntry(w http.ResponseWriter, entry cache.CacheEntry) error {
	dst := w.Header()
	for k, vv := range entry.Header {
		dst[k] = vv
	}

	w.WriteHeader(entry.StatusCode)
	_, err := w.Write(entry.Body)
	return err
}
