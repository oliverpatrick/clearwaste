package config

import (
	"strings"
	"testing"
)

func setValidAccountEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("ACCOUNT_HTTP_ADDR", ":8090")
	t.Setenv("ACCOUNT_DEV_EMAIL", " Dev@Example.com ")
	t.Setenv("ACCOUNT_DEV_PASSWORD", "development-only")
	t.Setenv("WORLD_DEV_LOGIN_TICKET", "opaque-value")
	t.Setenv("WORLD_DEV_ACCOUNT_ID", "41")
	t.Setenv("WORLD_DEV_CHARACTER_ID", "73")
	t.Setenv("WORLD_PUBLIC_HOST", "game.example.test")
	t.Setenv("WORLD_TCP_ADDR", ":7777")
}

func TestLoadAccount(t *testing.T) {
	setValidAccountEnvironment(t)
	cfg, err := LoadAccount()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddress != ":8090" || cfg.DevelopmentAccount.Email != "dev@example.com" {
		t.Fatalf("account config=%+v", cfg)
	}
	if cfg.DevelopmentAccount.AccountID != 41 || cfg.DevelopmentAccount.CharacterID != 73 {
		t.Fatalf("development account=%+v", cfg.DevelopmentAccount)
	}
	if cfg.WorldEntry.Ticket != "opaque-value" || cfg.WorldEntry.Host != "game.example.test" || cfg.WorldEntry.Port != 7777 {
		t.Fatalf("world entry=%+v", cfg.WorldEntry)
	}
}

func TestLoadAccountEndpointDefaults(t *testing.T) {
	setValidAccountEnvironment(t)
	t.Setenv("ACCOUNT_HTTP_ADDR", "")
	t.Setenv("WORLD_PUBLIC_HOST", "")
	t.Setenv("WORLD_TCP_ADDR", "")
	cfg, err := LoadAccount()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddress != ":8080" || cfg.WorldEntry.Host != "127.0.0.1" || cfg.WorldEntry.Port != 7777 {
		t.Fatalf("endpoint defaults=%+v", cfg)
	}
}

func TestLoadAccountRejectsMissingRequiredValuesWithoutExposingSecrets(t *testing.T) {
	for _, name := range []string{"ACCOUNT_DEV_EMAIL", "ACCOUNT_DEV_PASSWORD", "WORLD_DEV_LOGIN_TICKET", "WORLD_DEV_ACCOUNT_ID", "WORLD_DEV_CHARACTER_ID"} {
		t.Run(name, func(t *testing.T) {
			setValidAccountEnvironment(t)
			t.Setenv(name, "")
			_, err := LoadAccount()
			if err == nil {
				t.Fatal("expected error")
			}
			if strings.Contains(err.Error(), "development-only") || strings.Contains(err.Error(), "opaque-value") {
				t.Fatalf("error exposed a secret: %v", err)
			}
		})
	}
}

func TestLoadAccountRejectsInvalidWorldAddressAndIDs(t *testing.T) {
	tests := map[string]map[string]string{
		"invalid address": {"WORLD_TCP_ADDR": "missing-port"},
		"zero account":    {"WORLD_DEV_ACCOUNT_ID": "0"},
		"zero character":  {"WORLD_DEV_CHARACTER_ID": "0"},
	}
	for name, values := range tests {
		t.Run(name, func(t *testing.T) {
			setValidAccountEnvironment(t)
			for key, value := range values {
				t.Setenv(key, value)
			}
			if _, err := LoadAccount(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
