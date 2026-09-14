package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffenv"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

var (
	ErrConfig                                 = errors.New("config")
	ErrConfigWrongScheme                      = fmt.Errorf("%w: %w", ErrConfig, errors.New("wrong scheme"))
	ErrConfigTLSCertKeyFileMissing            = fmt.Errorf("%w: %w", ErrConfig, errors.New("cert or key file not present for https"))
	ErrConfigHost2BackendMapping              = fmt.Errorf("%w: %w", ErrConfig, errors.New("error in host to backend mapping"))
	ErrConfigHost2BackendMappingDuplicateHost = fmt.Errorf("%w: %w", ErrConfigHost2BackendMapping, errors.New("duplicate host entry"))
)

type config struct {
	scheme string
	host   string
	port   int

	certFile string
	keyFile  string

	host2backend map[string]string
}

func loadConfig(args []string) (config, error) {
	cfg := config{
		host2backend: map[string]string{},
	}
	var host2backend []string

	fs := ff.NewFlagSet("jamd")
	fs.StringVar(&cfg.scheme, 'S', "scheme", "http", "scheme to listen on")
	fs.StringVar(&cfg.host, 'H', "host", "0.0.0.0", "host to listen on")
	fs.IntVar(&cfg.port, 'P', "port", 44444, "port to listen on")
	fs.StringVar(&cfg.certFile, 0, "cert-file", "", "cert file path for the TLS configuration")
	fs.StringVar(&cfg.keyFile, 0, "key-file", "", "key file path for the TLS configuration")
	fs.StringListVar(&host2backend, 0, "host2backend", "mapping between host to its backend")

	err := ff.Parse(fs, args,
		ff.WithEnvVarPrefix("JAMD"),
		ff.WithConfigFile(".env"),
		ff.WithConfigFileParser(ffenv.Parse),
	)
	if err != nil {
		fmt.Println(ffhelp.Flags(fs))
		return cfg, err
	}

	if strings.ToLower(cfg.scheme) == "https" {
		if cfg.certFile == "" || cfg.keyFile == "" {
			return cfg, ErrConfigTLSCertKeyFileMissing
		}
	}

	for _, v := range host2backend {
		host, backend, ok := strings.Cut(v, ":")
		if !ok {
			return cfg, ErrConfigHost2BackendMapping
		}

		_, ok = cfg.host2backend[host]
		if ok {
			return cfg, ErrConfigHost2BackendMappingDuplicateHost
		}

		cfg.host2backend[host] = backend
	}

	return cfg, nil
}
