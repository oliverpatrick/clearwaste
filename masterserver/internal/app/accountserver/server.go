package accountserver

import (
	"net/http"
	"time"

	"master/clearwaste/config"
	"master/clearwaste/internal/account"
	"master/clearwaste/internal/account/auth/login"
	"master/clearwaste/internal/character"
	"master/clearwaste/internal/world"
)

func New(cfg config.AccountConfig) *http.Server {
	repository := account.NewDevelopmentRepository(
		account.ID(cfg.DevelopmentAccount.AccountID),
		cfg.DevelopmentAccount.Email,
		cfg.DevelopmentAccount.Password,
		character.ID(cfg.DevelopmentAccount.CharacterID),
	)
	if second := cfg.SecondDevelopmentAccount; second != nil { repository.Add(account.ID(second.AccountID), second.Email, second.Password, character.ID(second.CharacterID)) }
	service := login.NewService(repository, world.EntryGrant{
		Ticket: cfg.WorldEntry.Ticket,
		Host:   cfg.WorldEntry.Host,
		Port:   cfg.WorldEntry.Port,
	})
	mux := http.NewServeMux()
	mux.Handle("/v1/login", login.NewHandler(service))
	return &http.Server{Addr: cfg.HTTPAddress, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
}
