package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
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
		case errors.Is(err, ErrConfig):
			os.Exit(ExitCodeErrConfig)
		default:
			os.Exit(ExitCodeErrRun)
		}
	}
}

func run(args []string) error {
	cfg, err := loadConfig(args)
	if err != nil {
		slog.Error("load configuration", "error", err)
		os.Exit(ExitCodeErrConfig)
	}

	hostPort := fmt.Sprintf("%s:%d", cfg.host, cfg.port)
	slog.Info("listen and serve", "scheme", cfg.scheme, "host", cfg.host, "port", cfg.port)

	j := NewJamd(cfg)

	switch strings.ToLower(cfg.scheme) {
	case "http":
		err = http.ListenAndServe(hostPort, j)
	case "https":
		err = http.ListenAndServeTLS(hostPort, cfg.certFile, cfg.keyFile, j)
	default:
		return fmt.Errorf("%w: %q", ErrConfigWrongScheme, cfg.scheme)
	}
	if err != nil {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}
