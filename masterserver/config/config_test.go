package config

import (
	"strconv"
	"testing"
	"time"
)

func TestDefaultTick(t *testing.T) {
	t.Setenv("WORLD_TICK_MS", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TickDuration != 600*time.Millisecond {
		t.Fatalf("tick=%s, want 600ms", cfg.TickDuration)
	}
}

func TestTickFromEnvironment(t *testing.T) {
	t.Setenv("WORLD_TICK_MS", "750")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TickDuration != 750*time.Millisecond {
		t.Fatalf("tick=%s, want 750ms", cfg.TickDuration)
	}
}

func TestRejectsTickOutsideSafeRange(t *testing.T) {
	for _, value := range []string{"49", "5001", "not-a-number"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("WORLD_TICK_MS", value)
			if _, err := Load(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDefaultPlayerDeathConfiguration(t *testing.T) {
	for _, name := range []string{"PLAYER_PROTECTED_ITEM_UNITS", "PLAYER_DROP_PRIVATE_TICKS", "PLAYER_DROP_LIFETIME_TICKS", "PLAYER_DEATH_TICKS"} {
		t.Setenv(name, "")
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PlayerProtectedItemUnits != 5 || cfg.PlayerDropPrivateTicks != 200 || cfg.PlayerDropLifetimeTicks != 500 || cfg.PlayerDeathTicks != 5 {
		t.Fatalf("death config=%+v", cfg)
	}
}

func TestPlayerDeathConfigurationFromEnvironment(t *testing.T) {
	t.Setenv("PLAYER_PROTECTED_ITEM_UNITS", "4")
	t.Setenv("PLAYER_DROP_PRIVATE_TICKS", "10")
	t.Setenv("PLAYER_DROP_LIFETIME_TICKS", "20")
	t.Setenv("PLAYER_DEATH_TICKS", "6")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PlayerProtectedItemUnits != 4 || cfg.PlayerDropPrivateTicks != 10 || cfg.PlayerDropLifetimeTicks != 20 || cfg.PlayerDeathTicks != 6 {
		t.Fatalf("death config=%+v", cfg)
	}
}

func TestRejectsInvalidPlayerDeathConfiguration(t *testing.T) {
	for name, value := range map[string]string{
		"PLAYER_PROTECTED_ITEM_UNITS": "-1",
		"PLAYER_DROP_PRIVATE_TICKS":   "-1",
		"PLAYER_DROP_LIFETIME_TICKS":  "0",
		"PLAYER_DEATH_TICKS":          "0",
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(name, value)
			if _, err := Load(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
	t.Run("lifetime not after privacy", func(t *testing.T) {
		t.Setenv("PLAYER_DROP_PRIVATE_TICKS", "20")
		t.Setenv("PLAYER_DROP_LIFETIME_TICKS", "20")
		if _, err := Load(); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestDefaultNetworkConfiguration(t *testing.T) {
	clearNetworkEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxPayloadSize != 64<<10 || cfg.InboundQueueCapacity != 64 || cfg.OutboundQueueCapacity != 256 {
		t.Fatalf("network config=%+v", cfg)
	}
}

func TestNetworkConfigurationFromEnvironment(t *testing.T) {
	t.Setenv("WORLD_MAX_PAYLOAD_BYTES", "4096")
	t.Setenv("WORLD_INBOUND_QUEUE_CAPACITY", "8")
	t.Setenv("WORLD_OUTBOUND_QUEUE_CAPACITY", "16")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxPayloadSize != 4096 || cfg.InboundQueueCapacity != 8 || cfg.OutboundQueueCapacity != 16 {
		t.Fatalf("network config=%+v", cfg)
	}
}

func TestRejectsInvalidNetworkConfiguration(t *testing.T) {
	tests := map[string]string{
		"WORLD_MAX_PAYLOAD_BYTES":       "0",
		"WORLD_INBOUND_QUEUE_CAPACITY":  "-1",
		"WORLD_OUTBOUND_QUEUE_CAPACITY": "not-a-number",
	}
	for name, value := range tests {
		t.Run(name, func(t *testing.T) {
			clearNetworkEnv(t)
			t.Setenv(name, value)
			if _, err := Load(); err == nil {
				t.Fatal("expected error")
			}
		})
	}

	t.Run("payload exceeds uint32", func(t *testing.T) {
		clearNetworkEnv(t)
		t.Setenv("WORLD_MAX_PAYLOAD_BYTES", "4294967296")
		if _, err := Load(); err == nil {
			t.Fatal("expected error")
		}
	})
}

func clearNetworkEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"WORLD_MAX_PAYLOAD_BYTES", "WORLD_INBOUND_QUEUE_CAPACITY", "WORLD_OUTBOUND_QUEUE_CAPACITY"} {
		t.Setenv(name, "")
	}
}

func TestProtocolVersionConfiguration(t *testing.T) {
	clearLoginEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProtocolVersion != 1 {
		t.Fatalf("protocol version=%d", cfg.ProtocolVersion)
	}

	t.Setenv("WORLD_PROTOCOL_VERSION", "7")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProtocolVersion != 7 {
		t.Fatalf("protocol version=%d", cfg.ProtocolVersion)
	}
}

func TestRejectsInvalidProtocolVersion(t *testing.T) {
	for _, value := range []string{"0", "65536", "not-a-number"} {
		t.Run(value, func(t *testing.T) {
			clearLoginEnv(t)
			t.Setenv("WORLD_PROTOCOL_VERSION", value)
			if _, err := Load(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestCompleteDevelopmentLoginConfiguration(t *testing.T) {
	clearLoginEnv(t)
	t.Setenv("WORLD_DEV_LOGIN_TICKET", "opaque")
	t.Setenv("WORLD_DEV_ACCOUNT_ID", "41")
	t.Setenv("WORLD_DEV_CHARACTER_ID", "73")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DevelopmentLogin == nil || cfg.DevelopmentLogin.Ticket != "opaque" || cfg.DevelopmentLogin.AccountID != 41 || cfg.DevelopmentLogin.CharacterID != 73 {
		t.Fatalf("development login=%+v", cfg.DevelopmentLogin)
	}
}

func TestUnconfiguredDevelopmentLoginIsUnavailable(t *testing.T) {
	clearLoginEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DevelopmentLogin != nil {
		t.Fatalf("development login=%+v", cfg.DevelopmentLogin)
	}
}

func TestPartialDevelopmentLoginIsUnavailable(t *testing.T) {
	tests := []map[string]string{
		{"WORLD_DEV_LOGIN_TICKET": "opaque"},
		{"WORLD_DEV_ACCOUNT_ID": "41"},
		{"WORLD_DEV_CHARACTER_ID": "73"},
		{"WORLD_DEV_LOGIN_TICKET": "opaque", "WORLD_DEV_ACCOUNT_ID": "41"},
		{"WORLD_DEV_LOGIN_TICKET": "opaque", "WORLD_DEV_CHARACTER_ID": "73"},
		{"WORLD_DEV_ACCOUNT_ID": "41", "WORLD_DEV_CHARACTER_ID": "73"},
	}
	for i, values := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			clearLoginEnv(t)
			for name, value := range values {
				t.Setenv(name, value)
			}
			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.DevelopmentLogin != nil {
				t.Fatalf("development login=%+v", cfg.DevelopmentLogin)
			}
		})
	}
}

func TestRejectsInvalidDevelopmentLoginIDs(t *testing.T) {
	for name, values := range map[string]map[string]string{
		"zero account":      {"WORLD_DEV_ACCOUNT_ID": "0"},
		"invalid account":   {"WORLD_DEV_ACCOUNT_ID": "invalid"},
		"zero character":    {"WORLD_DEV_CHARACTER_ID": "0"},
		"invalid character": {"WORLD_DEV_CHARACTER_ID": "invalid"},
	} {
		t.Run(name, func(t *testing.T) {
			clearLoginEnv(t)
			for envName, value := range values {
				t.Setenv(envName, value)
			}
			if _, err := Load(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func clearLoginEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"WORLD_PROTOCOL_VERSION", "WORLD_DEV_LOGIN_TICKET", "WORLD_DEV_ACCOUNT_ID", "WORLD_DEV_CHARACTER_ID"} {
		t.Setenv(name, "")
	}
}
