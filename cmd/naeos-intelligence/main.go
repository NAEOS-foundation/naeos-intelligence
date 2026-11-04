package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NAEOS-foundation/naeos-intelligence/internal/config"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/httpapi"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/postgres"
)

const applicationName = "NAEOS Intelligence"

func authVerifierFromEnvironment() (httpapi.TokenVerifier, error) {
	issuer := os.Getenv("NAEOS_AUTH_ISSUER")
	audience := os.Getenv("NAEOS_AUTH_AUDIENCE")
	jwksURL := os.Getenv("NAEOS_AUTH_JWKS_URL")
	if issuer == "" && audience == "" && jwksURL == "" {
		return nil, nil // protected API stays fail-closed when auth is not configured
	}
	return httpapi.NewJWKSVerifier(issuer, audience, jwksURL)
}

func run() error {
	cfg, err := config.Load(os.Environ())
	if err != nil { return err }
	verifier, err := authVerifierFromEnvironment()
	if err != nil { return err }

	addr := os.Getenv("NAEOS_INTELLIGENCE_HTTP_ADDR")
	if addr == "" { addr = "127.0.0.1:8080" }

	var store httpapi.Store
	var closeStore func()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		if cfg.App.Environment == "production" { return errors.New("DATABASE_URL is required in production") }
		store = httpapi.NewMemoryStore()
		closeStore = func() {}
	} else {
		startupCtx, startupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		pool, err := postgres.OpenPool(startupCtx, dsn)
		startupCancel()
		if err != nil { return fmt.Errorf("initialize PostgreSQL pool: %w", err) }
		store = postgres.New(pool)
		closeStore = pool.Close
	}
	defer closeStore()

	server := &http.Server{
		Addr: addr,
		Handler: httpapi.NewHandler(store, httpapi.Authenticator{Verifier: verifier}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout: 60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErr := make(chan error, 1)
	go func() {
		fmt.Printf("%s HTTP server starting (environment=%s log_level=%s addr=%s storage=%s)\n",
			applicationName, cfg.App.Environment, cfg.App.LogLevel, addr, storageMode(dsn))
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) { return nil }
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("graceful HTTP shutdown: %w", err)
		}
		return nil
	}
}

func storageMode(dsn string) string { if dsn == "" { return "memory" }; return "postgres" }

func main() {
	if err := run(); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
