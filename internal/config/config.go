// Package config provides validated, startup-only application configuration.
package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Config is an immutable-by-convention configuration snapshot for the initial
// application bootstrap. Only non-sensitive fields are supported here;
// sensitive values such as DATABASE_URL remain in main.go.
type Config struct {
	App       AppConfig
	HTTP      HTTPConfig
	RateLimit RateLimitConfig
	OpenAPI   OpenAPIConfig
	Core      CoreConfig
}

// AppConfig contains the minimal approved application settings.
type AppConfig struct {
	Environment string
	LogLevel    string
}

// HTTPConfig contains HTTP server settings.
type HTTPConfig struct {
	Addr string
}

// RateLimitConfig contains rate limiter settings.
type RateLimitConfig struct {
	Rate  float64
	Burst int
}

// OpenAPIConfig contains OpenAPI spec path settings.
type OpenAPIConfig struct {
	SpecPath string
}

// CoreConfig contains non-sensitive NAEOS Core connection settings.
// Credentials are read separately by the application bootstrap.
type CoreConfig struct {
	BaseURL string
}

const (
	envPrefix          = "NAEOS_INTELLIGENCE_"
	environmentKey     = envPrefix + "APP_ENVIRONMENT"
	logLevelKey        = envPrefix + "APP_LOG_LEVEL"
	httpAddrKey        = envPrefix + "HTTP_ADDR"
	rateLimitRateKey   = envPrefix + "RATE_LIMIT_RATE"
	rateLimitBurstKey  = envPrefix + "RATE_LIMIT_BURST"
	openAPISpecPathKey = envPrefix + "OPENAPI_SPEC_PATH"
	coreBaseURLKey     = envPrefix + "CORE_API_BASE_URL"
	defaultEnv         = "development"
	defaultLogLevel    = "info"
	defaultHTTPAddr    = "127.0.0.1:8080"
	defaultRateLimit   = 10.0
	defaultRateBurst   = 20
	defaultSpecPath    = "api/openapi.yaml"
	defaultCoreBaseURL = "https://api.naeos.dev/api"
)

// Load reads the supplied process environment and returns a validated snapshot.
// Configuration precedence is environment variables over safe code defaults.
func Load(environ []string) (Config, error) {
	values := make(map[string]string, len(environ))
	for _, entry := range environ {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		values[key] = value
	}

	knownKeys := map[string]struct{}{
		environmentKey:     {},
		logLevelKey:        {},
		httpAddrKey:        {},
		rateLimitRateKey:   {},
		rateLimitBurstKey:  {},
		openAPISpecPathKey: {},
		coreBaseURLKey:     {},
	}
	for key := range values {
		if strings.HasPrefix(key, envPrefix) {
			if _, ok := knownKeys[key]; !ok {
				return Config{}, fmt.Errorf("configuration %s: unknown setting", key)
			}
		}
	}

	cfg := Config{
		App: AppConfig{
			Environment: defaultEnv,
			LogLevel:    defaultLogLevel,
		},
		HTTP: HTTPConfig{
			Addr: defaultHTTPAddr,
		},
		RateLimit: RateLimitConfig{
			Rate:  defaultRateLimit,
			Burst: defaultRateBurst,
		},
		OpenAPI: OpenAPIConfig{
			SpecPath: defaultSpecPath,
		},
		Core: CoreConfig{BaseURL: defaultCoreBaseURL},
	}
	if value, ok := values[environmentKey]; ok {
		cfg.App.Environment = strings.TrimSpace(value)
	}
	if value, ok := values[logLevelKey]; ok {
		cfg.App.LogLevel = strings.TrimSpace(value)
	}
	if value, ok := values[httpAddrKey]; ok {
		cfg.HTTP.Addr = strings.TrimSpace(value)
	}
	if value, ok := values[rateLimitRateKey]; ok {
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && parsed > 0 {
			cfg.RateLimit.Rate = parsed
		}
	}
	if value, ok := values[rateLimitBurstKey]; ok {
		if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed > 0 {
			cfg.RateLimit.Burst = parsed
		}
	}
	if value, ok := values[openAPISpecPathKey]; ok {
		cfg.OpenAPI.SpecPath = strings.TrimSpace(value)
	}
	if value, ok := values[coreBaseURLKey]; ok {
		cfg.Core.BaseURL = strings.TrimSpace(value)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks supported values without including supplied values in errors.
func (c Config) Validate() error {
	switch c.App.Environment {
	case "development", "test", "staging", "production":
	default:
		return fmt.Errorf("configuration %s: must be development, test, staging, or production", environmentKey)
	}

	switch c.App.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("configuration %s: must be debug, info, warn, or error", logLevelKey)
	}

	if strings.TrimSpace(c.HTTP.Addr) == "" {
		return fmt.Errorf("configuration %s: must not be empty", httpAddrKey)
	}

	if c.RateLimit.Rate <= 0 {
		return fmt.Errorf("configuration %s: must be greater than 0", rateLimitRateKey)
	}

	if c.RateLimit.Burst <= 0 {
		return fmt.Errorf("configuration %s: must be greater than 0", rateLimitBurstKey)
	}

	if strings.TrimSpace(c.OpenAPI.SpecPath) == "" {
		return fmt.Errorf("configuration %s: must not be empty", openAPISpecPathKey)
	}

	if strings.TrimSpace(c.Core.BaseURL) != "" {
		coreURL, err := url.Parse(c.Core.BaseURL)
		if err != nil || coreURL.Host == "" || coreURL.User != nil || coreURL.RawQuery != "" || coreURL.Fragment != "" ||
			(coreURL.Scheme != "https" && !(coreURL.Scheme == "http" && (coreURL.Hostname() == "localhost" || coreURL.Hostname() == "127.0.0.1" || (c.App.Environment == "development" && coreURL.Hostname() == "host.docker.internal")))) {
			return fmt.Errorf("configuration %s: must be an https URL (http is allowed only for localhost)", coreBaseURLKey)
		}
	}

	return nil
}
