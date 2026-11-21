package config

import (
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load(nil) error = %v", err)
	}
	if cfg.App.Environment != "development" {
		t.Errorf("Environment = %q, want development", cfg.App.Environment)
	}
	if cfg.App.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.App.LogLevel)
	}
}

func TestLoadEnvironmentOverridesDefaults(t *testing.T) {
	cfg, err := Load([]string{
		"NAEOS_INTELLIGENCE_APP_ENVIRONMENT=staging",
		"NAEOS_INTELLIGENCE_APP_LOG_LEVEL=warn",
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.App.Environment != "staging" || cfg.App.LogLevel != "warn" {
		t.Fatalf("unexpected config: %+v", cfg.App)
	}
}

func TestLoadRejectsInvalidEnvironment(t *testing.T) {
	const secretLikeValue = "production-password-do-not-leak"
	_, err := Load([]string{"NAEOS_INTELLIGENCE_APP_ENVIRONMENT=" + secretLikeValue})
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
	if strings.Contains(err.Error(), secretLikeValue) {
		t.Errorf("error leaked supplied value: %v", err)
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	_, err := Load([]string{"NAEOS_INTELLIGENCE_APP_LOG_LEVEL=verbose"})
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
}

func TestLoadRejectsUnknownPrefixedSetting(t *testing.T) {
	_, err := Load([]string{"NAEOS_INTELLIGENCE_DATABASE_PASSWORD=secret"})
	if err == nil {
		t.Fatal("Load() error = nil, want unknown-setting error")
	}
	if strings.Contains(err.Error(), "secret") {
		t.Errorf("error leaked secret value: %v", err)
	}
}

func TestValidateEnvironmentProfiles(t *testing.T) {
	for _, profile := range []string{"development", "test", "staging", "production"} {
		t.Run(profile, func(t *testing.T) {
			cfg := Config{
				App:       AppConfig{Environment: profile, LogLevel: "info"},
				HTTP:      HTTPConfig{Addr: "127.0.0.1:8080"},
				RateLimit: RateLimitConfig{Rate: 10.0, Burst: 20},
				OpenAPI:   OpenAPIConfig{SpecPath: "api/openapi.yaml"},
			}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestLoadHTTPConfig(t *testing.T) {
	cfg, err := Load([]string{"NAEOS_INTELLIGENCE_HTTP_ADDR=0.0.0.0:9090"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.Addr != "0.0.0.0:9090" {
		t.Errorf("HTTP.Addr = %q, want 0.0.0.0:9090", cfg.HTTP.Addr)
	}
}

func TestLoadRateLimitConfig(t *testing.T) {
	cfg, err := Load([]string{
		"NAEOS_INTELLIGENCE_RATE_LIMIT_RATE=5.5",
		"NAEOS_INTELLIGENCE_RATE_LIMIT_BURST=10",
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.RateLimit.Rate != 5.5 {
		t.Errorf("RateLimit.Rate = %v, want 5.5", cfg.RateLimit.Rate)
	}
	if cfg.RateLimit.Burst != 10 {
		t.Errorf("RateLimit.Burst = %v, want 10", cfg.RateLimit.Burst)
	}
}

func TestLoadOpenAPISpecPath(t *testing.T) {
	cfg, err := Load([]string{"NAEOS_INTELLIGENCE_OPENAPI_SPEC_PATH=/app/api/openapi.yaml"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.OpenAPI.SpecPath != "/app/api/openapi.yaml" {
		t.Errorf("OpenAPI.SpecPath = %q, want /app/api/openapi.yaml", cfg.OpenAPI.SpecPath)
	}
}

func TestValidateRejectsInvalidHTTPAddr(t *testing.T) {
	cfg := Config{HTTP: HTTPConfig{Addr: ""}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want validation error for empty HTTP addr")
	}
}

func TestValidateRejectsInvalidRateLimit(t *testing.T) {
	cfg := Config{
		HTTP:      HTTPConfig{Addr: "127.0.0.1:8080"},
		RateLimit: RateLimitConfig{Rate: 0, Burst: 20},
		OpenAPI:   OpenAPIConfig{SpecPath: "api/openapi.yaml"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want validation error for zero rate")
	}
	cfg.RateLimit.Rate = 10.0
	cfg.RateLimit.Burst = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want validation error for zero burst")
	}
}

func TestValidateRejectsInvalidOpenAPISpecPath(t *testing.T) {
	cfg := Config{
		HTTP:      HTTPConfig{Addr: "127.0.0.1:8080"},
		RateLimit: RateLimitConfig{Rate: 10.0, Burst: 20},
		OpenAPI:   OpenAPIConfig{SpecPath: ""},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want validation error for empty spec path")
	}
}

func TestCoreURLAllowsDockerGatewayOnlyInDevelopment(t *testing.T) {
	_, err := Load([]string{"NAEOS_INTELLIGENCE_CORE_API_BASE_URL=http://host.docker.internal:8081/api"})
	if err != nil {
		t.Fatalf("development Docker gateway URL rejected: %v", err)
	}

	_, err = Load([]string{
		"NAEOS_INTELLIGENCE_APP_ENVIRONMENT=production",
		"NAEOS_INTELLIGENCE_CORE_API_BASE_URL=http://host.docker.internal:8081/api",
	})
	if err == nil {
		t.Fatal("production accepted insecure Docker gateway URL")
	}
}
