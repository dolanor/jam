package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	err := run("http", "localhost", 44444)
	if err != nil {
		slog.Error("run", "error", err)
		os.Exit(1)
	}
}

func run(scheme, host string, port int) error {
	hostPort := fmt.Sprintf("%s:%d", host, port)
	slog.Info("listen and serve", "scheme", scheme, "host", host, "port", port)

	err := http.ListenAndServe(hostPort, nil)
	if err != nil {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}
