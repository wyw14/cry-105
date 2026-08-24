package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type config struct{ listen, stateRoot string }

func loadConfig() (config, error) {
	value := config{}
	flag.StringVar(&value.listen, "listen", envOr("ORBITLINK_LISTEN", "127.0.0.1:19705"), "HTTP listen address")
	flag.StringVar(&value.stateRoot, "state-root", envOr("ORBITLINK_STATE_ROOT", filepath.Join(".", "var", "orbitlink")), "durable state directory")
	flag.Parse()
	if value.listen == "" || value.stateRoot == "" {
		return config{}, fmt.Errorf("listen address and state root are required")
	}
	return value, nil
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
