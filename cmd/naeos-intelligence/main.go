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
	"github.com/NAEOS-foundation/naeos-intelligence/internal/evidence"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/httpapi"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/classifier"
	intctx "github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/context"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/orchestrator"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/intelligence/planner"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/model"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/postgres"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/ratelimit"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/runtime"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/tool"
	"github.com/NAEOS-foundation/naeos-intelligence/internal/verification"
	"github.com/jackc/pgx/v5/pgxpool"
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
	if err != nil {
		return err
	}
	verifier, err := authVerifierFromEnvironment()
	if err != nil {
		return err
	}

	limiter := ratelimit.NewLimiter(cfg.RateLimit.Rate, cfg.RateLimit.Burst)

	cl := classifier.NewRuleClassifier()
	cb := intctx.NewRuleBuilder()
	pl := planner.NewRulePlanner()
	reg := model.NewRegistry()
	tr := tool.NewRegistry()
	rp := runtime.NewProcessor(reg, tr)

	var pool *pgxpool.Pool
	validator, err := newOpenAPIValidator(cfg.OpenAPI.SpecPath)
	if err != nil {
		return fmt.Errorf("initialize OpenAPI validator: %w", err)
	}

	var store httpapi.Store
	var closeStore func()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		if cfg.App.Environment == "production" {
			return errors.New("DATABASE_URL is required in production")
		}
	} else {
		startupCtx, startupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		pool, err = postgres.OpenPool(startupCtx, dsn)
		startupCancel()
		if err != nil {
			return fmt.Errorf("initialize PostgreSQL pool: %w", err)
		}
		closeStore = pool.Close
	}

	var orch *orchestrator.Orchestrator
	if pool != nil {
		es := evidence.New(pool)
		ve := verification.New()
		orch = orchestrator.NewOrchestratorWithEvidence(cl, cb, pl, rp, es, ve)
	} else {
		orch = orchestrator.NewOrchestrator(cl, cb, pl, rp)
	}

	if pool != nil {
		store = postgres.New(pool)
	} else {
		store = httpapi.NewMemoryStore()
	}
	if closeStore != nil {
		defer closeStore()
	}

	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           httpapi.NewHandler(store, httpapi.Authenticator{Verifier: verifier}, limiter, validator, orch),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErr := make(chan error, 1)
	go func() {
		fmt.Printf("%s HTTP server starting (environment=%s log_level=%s addr=%s storage=%s)\n",
			applicationName, cfg.App.Environment, cfg.App.LogLevel, cfg.HTTP.Addr, storageMode(dsn))
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
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

func storageMode(dsn string) string {
	if dsn == "" {
		return "memory"
	}
	return "postgres"
}

func newOpenAPIValidator(specPath string) (*httpapi.OpenAPIValidator, error) {
	v, err := httpapi.NewOpenAPIValidator(specPath)
	if err != nil {
		return nil, fmt.Errorf("load OpenAPI spec %s: %w", specPath, err)
	}
	return v, nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
