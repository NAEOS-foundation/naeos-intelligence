// Package config provides validated, startup-only application configuration.
package config

import (
	"fmt"
	"strings"
)

// Config is an immutable-by-convention configuration snapshot for the initial
// application bootstrap. Only non-sensitive fields are currently supported.
type Config struct {
	App AppConfig
}

// AppConfig contains the minimal approved application settings.
type AppConfig struct {
	Environment string
	LogLevel    string
}

const (
	envPrefix       = "NAEOS_INTELLIGENCE_"
	environmentKey  = envPrefix + "APP_ENVIRONMENT"
	logLevelKey     = envPrefix + "APP_LOG_LEVEL"
	defaultEnv      = "development"
	defaultLogLevel = "info"
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

	for key := range values {
		if strings.HasPrefix(key, envPrefix) && key != environmentKey && key != logLevelKey {
			return Config{}, fmt.Errorf("configuration %s: unknown setting", key)
		}
	}

	cfg := Config{App: AppConfig{
		Environment: defaultEnv,
		LogLevel:    defaultLogLevel,
	}}
	if value, ok := values[environmentKey]; ok {
		cfg.App.Environment = strings.TrimSpace(value)
	}
	if value, ok := values[logLevelKey]; ok {
		cfg.App.LogLevel = strings.TrimSpace(value)
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
	return nil
}
