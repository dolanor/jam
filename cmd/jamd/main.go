package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/dolanor/jam"
	"github.com/dolanor/jam/config"
)

const (
	ExitCodeErrRun    = 1
	ExitCodeErrConfig = 3
)

func main() {
	err := run(os.Args[1:])
	if err != nil {
		slog.Error("run", "error", err)

		switch {
		case errors.Is(err, config.Err):
			os.Exit(ExitCodeErrConfig)
		default:
			os.Exit(ExitCodeErrRun)
		}
	}
}

func run(args []string) error {
	cfg, err := config.Load(args)
	if err != nil {
		slog.Error("load configuration", "error", err)
		os.Exit(ExitCodeErrConfig)
	}

	hostPort := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	slog.Info("listen and serve", "scheme", cfg.Scheme, "host", cfg.Host, "port", cfg.Port, "host2backend", cfg.Host2Backend)

	j := jam.NewJamd(cfg)

	switch strings.ToLower(cfg.Scheme) {
	case "http":
		err = http.ListenAndServe(hostPort, j)
	case "https":
		err = http.ListenAndServeTLS(hostPort, cfg.CertFile, cfg.KeyFile, j)
	default:
		return fmt.Errorf("%w: %q", config.ErrWrongScheme, cfg.Scheme)
	}
	if err != nil {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}
