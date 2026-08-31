package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultTickMS                = 600
	minTickMS                    = 50
	maxTickMS                    = 5000
	defaultMaxPayloadSize        = 64 << 10
	defaultInboundQueueCapacity  = 64
	defaultOutboundQueueCapacity = 256
	defaultProtocolVersion       = 1
)

type DevelopmentLoginConfig struct {
	Ticket      string
	AccountID   uint64
	CharacterID uint64
}

type Config struct {
	TickDuration             time.Duration
	ContentRoot              string
	TCPAddress               string
	MaxPayloadSize           uint32
	InboundQueueCapacity     int
	OutboundQueueCapacity    int
	ProtocolVersion          uint16
	DevelopmentLogin         *DevelopmentLoginConfig
	HTTPAddress              string
	SaveDir                  string
	AuthDB                   string
	PlayerProtectedItemUnits int
	PlayerDropPrivateTicks   int
	PlayerDropLifetimeTicks  int
	PlayerDeathTicks         int
}

func Load() (Config, error) {
	tickMS := defaultTickMS
	if value := os.Getenv("WORLD_TICK_MS"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return Config{}, fmt.Errorf("WORLD_TICK_MS: %w", err)
		}
		tickMS = parsed
	}
	if tickMS < minTickMS || tickMS > maxTickMS {
		return Config{}, fmt.Errorf("WORLD_TICK_MS=%d outside %d..%d", tickMS, minTickMS, maxTickMS)
	}
	protocolVersion, err := envUint16("WORLD_PROTOCOL_VERSION", defaultProtocolVersion)
	if err != nil || protocolVersion == 0 {
		return Config{}, fmt.Errorf("invalid WORLD_PROTOCOL_VERSION")
	}
	developmentLogin, err := loadDevelopmentLogin()
	if err != nil {
		return Config{}, err
	}
	maxPayloadSize, err := envUint32("WORLD_MAX_PAYLOAD_BYTES", defaultMaxPayloadSize)
	if err != nil || maxPayloadSize == 0 {
		return Config{}, fmt.Errorf("invalid WORLD_MAX_PAYLOAD_BYTES")
	}
	inboundQueueCapacity, err := envInt("WORLD_INBOUND_QUEUE_CAPACITY", defaultInboundQueueCapacity)
	if err != nil || inboundQueueCapacity <= 0 {
		return Config{}, fmt.Errorf("invalid WORLD_INBOUND_QUEUE_CAPACITY")
	}
	outboundQueueCapacity, err := envInt("WORLD_OUTBOUND_QUEUE_CAPACITY", defaultOutboundQueueCapacity)
	if err != nil || outboundQueueCapacity <= 0 {
		return Config{}, fmt.Errorf("invalid WORLD_OUTBOUND_QUEUE_CAPACITY")
	}
	protected, err := envInt("PLAYER_PROTECTED_ITEM_UNITS", 5)
	if err != nil || protected < 0 {
		return Config{}, fmt.Errorf("invalid PLAYER_PROTECTED_ITEM_UNITS")
	}
	privateTicks, err := envInt("PLAYER_DROP_PRIVATE_TICKS", 200)
	if err != nil || privateTicks < 0 {
		return Config{}, fmt.Errorf("invalid PLAYER_DROP_PRIVATE_TICKS")
	}
	lifetimeTicks, err := envInt("PLAYER_DROP_LIFETIME_TICKS", 500)
	if err != nil || lifetimeTicks <= privateTicks {
		return Config{}, fmt.Errorf("invalid PLAYER_DROP_LIFETIME_TICKS")
	}
	deathTicks, err := envInt("PLAYER_DEATH_TICKS", 5)
	if err != nil || deathTicks <= 0 {
		return Config{}, fmt.Errorf("invalid PLAYER_DEATH_TICKS")
	}
	return Config{
		TickDuration:             time.Duration(tickMS) * time.Millisecond,
		ContentRoot:              envOrDefault("GAME_CONTENT_ROOT", "game_content"),
		TCPAddress:               envOrDefault("WORLD_TCP_ADDR", ":7777"),
		MaxPayloadSize:           maxPayloadSize,
		InboundQueueCapacity:     inboundQueueCapacity,
		OutboundQueueCapacity:    outboundQueueCapacity,
		ProtocolVersion:          protocolVersion,
		DevelopmentLogin:         developmentLogin,
		HTTPAddress:              envOrDefault("CONTENT_HTTP_ADDR", ":8080"),
		SaveDir:                  envOrDefault("PLAYER_SAVE_DIR", "data/players"),
		AuthDB:                   envOrDefault("AUTH_DB_PATH", "data/auth.sqlite"),
		PlayerProtectedItemUnits: protected,
		PlayerDropPrivateTicks:   privateTicks,
		PlayerDropLifetimeTicks:  lifetimeTicks,
		PlayerDeathTicks:         deathTicks,
	}, nil
}

func loadDevelopmentLogin() (*DevelopmentLoginConfig, error) {
	accountID, accountSet, err := envOptionalUint64("WORLD_DEV_ACCOUNT_ID")
	if err != nil || accountSet && accountID == 0 {
		return nil, fmt.Errorf("invalid WORLD_DEV_ACCOUNT_ID")
	}
	characterID, characterSet, err := envOptionalUint64("WORLD_DEV_CHARACTER_ID")
	if err != nil || characterSet && characterID == 0 {
		return nil, fmt.Errorf("invalid WORLD_DEV_CHARACTER_ID")
	}
	ticket := os.Getenv("WORLD_DEV_LOGIN_TICKET")
	if ticket == "" || !accountSet || !characterSet {
		return nil, nil
	}
	return &DevelopmentLoginConfig{Ticket: ticket, AccountID: accountID, CharacterID: characterID}, nil
}

func envUint16(name string, fallback uint16) (uint16, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 16)
	return uint16(parsed), err
}

func envOptionalUint64(name string) (uint64, bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return 0, false, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	return parsed, true, err
}

func envUint32(name string, fallback uint32) (uint32, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	return uint32(parsed), err
}

func envInt(name string, fallback int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
