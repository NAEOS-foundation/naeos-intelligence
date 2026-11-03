package main

import (
	"fmt"
	"os"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/config"
)

const applicationName = "NAEOS Intelligence"

func run() error {
	cfg, err := config.Load(os.Environ())
	if err != nil {
		return err
	}
	fmt.Printf("%s bootstrap (environment=%s log_level=%s)\n",
		applicationName, cfg.App.Environment, cfg.App.LogLevel)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
