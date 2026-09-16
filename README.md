# Jamd

Jamd is a simple reverse proxy with some in-memory caching features.

## Use

```console
go run github.com/dolanor/jamd@latest --host2backend github.com:http://1.2.3.4:8080
```

### Configuration

#### Flags

```console
NAME
  jamd

FLAGS
  -S, --scheme STRING         scheme to listen on (default: http)
  -H, --host STRING           host to listen on (default: 0.0.0.0)
  -P, --port INT              port to listen on (default: 44444)
      --cert-file STRING      cert file path for the TLS configuration
      --key-file STRING       key file path for the TLS configuration
      --host2backend STRING   mapping between host to its backend (format: <hostname>:<backend_scheme>://<backend_hostname>[:<backend_port>]. eg. github.com:http://1.2.3.4:8080)
      --cache-enabled         cache proxied GET/HEAD responses in memory (default: false)
      --cache-ttl DURATION    how long a cached response stays fresh (default: 1m0s)
```

#### Environment

```env
JAMD_SCHEME=https
JAMD_HOST=0.0.0.0
JAMD_PORT=44445
JAMD_CERT_FILE=./mycert.cert
JAMD_KEY_FILE=./mykey.key
JAMD_HOST2BACKEND=github.com:http://1.2.3.4:8080
JAMD_CACHE_ENABLED=true
JAMD_CACHE_TTL=1m
```
