package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffenv"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

var (
	Err                                 = errors.New("config")
	ErrWrongScheme                      = fmt.Errorf("%w: %w", Err, errors.New("wrong scheme"))
	ErrTLSCertKeyFileMissing            = fmt.Errorf("%w: %w", Err, errors.New("cert or key file not present for https"))
	ErrHost2BackendMapping              = fmt.Errorf("%w: %w", Err, errors.New("error in host to backend mapping"))
	ErrHost2BackendMappingDuplicateHost = fmt.Errorf("%w: %w", ErrHost2BackendMapping, errors.New("duplicate host entry"))
)

type Config struct {
	Scheme string
	Host   string
	Port   int

	CertFile string
	KeyFile  string

	Host2Backend map[string]string

	CacheEnabled bool
	CacheTTL     time.Duration
}

func Load(args []string) (Config, error) {
	cfg := Config{
		Host2Backend: map[string]string{},
	}
	var host2backend []string

	fs := ff.NewFlagSet("jamd")
	fs.StringVar(&cfg.Scheme, 'S', "scheme", "http", "scheme to listen on")
	fs.StringVar(&cfg.Host, 'H', "host", "0.0.0.0", "host to listen on")
	fs.IntVar(&cfg.Port, 'P', "port", 44444, "port to listen on")
	fs.StringVar(&cfg.CertFile, 0, "cert-file", "", "cert file path for the TLS configuration")
	fs.StringVar(&cfg.KeyFile, 0, "key-file", "", "key file path for the TLS configuration")
	fs.StringListVar(&host2backend, 0, "host2backend", "mapping between host to its backend (format: <hostname>:<backend_scheme>://<backend_hostname>[:<backend_port>]. eg. github.com:http://1.2.3.4:8080)")
	fs.BoolVarDefault(&cfg.CacheEnabled, 0, "cache-enabled", false, "cache proxied GET/HEAD responses in memory")
	fs.DurationVar(&cfg.CacheTTL, 0, "cache-ttl", time.Minute, "how long a cached response stays fresh")

	err := ff.Parse(fs, args,
		ff.WithEnvVarPrefix("JAMD"),
		ff.WithConfigFile(".env"),
		ff.WithConfigFileParser(ffenv.Parse),
	)
	if err != nil {
		fmt.Println(ffhelp.Flags(fs))
		return cfg, err
	}

	if strings.ToLower(cfg.Scheme) == "https" {
		if cfg.CertFile == "" || cfg.KeyFile == "" {
			return cfg, ErrTLSCertKeyFileMissing
		}
	}

	for _, v := range host2backend {
		host, backend, ok := strings.Cut(v, ":")
		if !ok {
			return cfg, ErrHost2BackendMapping
		}

		_, ok = cfg.Host2Backend[host]
		if ok {
			return cfg, ErrHost2BackendMappingDuplicateHost
		}

		cfg.Host2Backend[host] = backend
	}

	return cfg, nil
}
