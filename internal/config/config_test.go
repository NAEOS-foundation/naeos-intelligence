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
			cfg := Config{App: AppConfig{Environment: profile, LogLevel: "info"}}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}
