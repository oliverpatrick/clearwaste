package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type DevelopmentAccountConfig struct {
	Email       string
	Password    string
	AccountID   uint64
	CharacterID uint64
}

type WorldEntryConfig struct {
	Ticket string
	Host   string
	Port   int
}

type AccountConfig struct {
	HTTPAddress        string
	DevelopmentAccount DevelopmentAccountConfig
	WorldEntry         WorldEntryConfig
}

func LoadAccount() (AccountConfig, error) {
	email := strings.ToLower(strings.TrimSpace(os.Getenv("ACCOUNT_DEV_EMAIL")))
	if email == "" {
		return AccountConfig{}, fmt.Errorf("ACCOUNT_DEV_EMAIL is required")
	}
	password := os.Getenv("ACCOUNT_DEV_PASSWORD")
	if password == "" {
		return AccountConfig{}, fmt.Errorf("ACCOUNT_DEV_PASSWORD is required")
	}
	ticket := os.Getenv("WORLD_DEV_LOGIN_TICKET")
	if ticket == "" {
		return AccountConfig{}, fmt.Errorf("WORLD_DEV_LOGIN_TICKET is required")
	}
	accountID, accountSet, err := envOptionalUint64("WORLD_DEV_ACCOUNT_ID")
	if err != nil || !accountSet || accountID == 0 {
		return AccountConfig{}, fmt.Errorf("invalid WORLD_DEV_ACCOUNT_ID")
	}
	characterID, characterSet, err := envOptionalUint64("WORLD_DEV_CHARACTER_ID")
	if err != nil || !characterSet || characterID == 0 {
		return AccountConfig{}, fmt.Errorf("invalid WORLD_DEV_CHARACTER_ID")
	}
	host := strings.TrimSpace(envOrDefault("WORLD_PUBLIC_HOST", "127.0.0.1"))
	if host == "" {
		return AccountConfig{}, fmt.Errorf("invalid WORLD_PUBLIC_HOST")
	}
	_, portText, err := net.SplitHostPort(envOrDefault("WORLD_TCP_ADDR", ":7777"))
	if err != nil {
		return AccountConfig{}, fmt.Errorf("invalid WORLD_TCP_ADDR")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return AccountConfig{}, fmt.Errorf("invalid WORLD_TCP_ADDR")
	}
	return AccountConfig{
		HTTPAddress: envOrDefault("ACCOUNT_HTTP_ADDR", ":8080"),
		DevelopmentAccount: DevelopmentAccountConfig{
			Email: email, Password: password, AccountID: accountID, CharacterID: characterID,
		},
		WorldEntry: WorldEntryConfig{Ticket: ticket, Host: host, Port: port},
	}, nil
}
