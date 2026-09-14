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
	ErrConfig                      = errors.New("config")
	ErrConfigWrongScheme           = fmt.Errorf("%w: %w", ErrConfig, errors.New("wrong scheme"))
	ErrConfigTLSCertKeyFileMissing = fmt.Errorf("%w: %w", ErrConfig, errors.New("cert or key file not present for https"))
)

type config struct {
	scheme string
	host   string
	port   int

	certFile string
	keyFile  string
}

func loadConfig(args []string) (config, error) {
	cfg := config{}

	fs := ff.NewFlagSet("jamd")
	fs.StringVar(&cfg.scheme, 'S', "scheme", "http", "scheme to listen on")
	fs.StringVar(&cfg.host, 'H', "host", "0.0.0.0", "host to listen on")
	fs.IntVar(&cfg.port, 'P', "port", 44444, "port to listen on")
	fs.StringVar(&cfg.certFile, 0, "cert-file", "", "cert file path for the TLS configuration")
	fs.StringVar(&cfg.keyFile, 0, "key-file", "", "key file path for the TLS configuration")

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

	return cfg, nil
}
