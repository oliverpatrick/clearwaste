package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"master/clearwaste/config"
	"master/clearwaste/internal/account"
	"master/clearwaste/internal/character"
	"master/clearwaste/internal/engine/network"
	"master/clearwaste/internal/engine/network/transport"
	"master/clearwaste/internal/game/interaction"
	"master/clearwaste/internal/game/movement"
	"master/clearwaste/internal/world"
	"master/clearwaste/internal/world/bootstrap"
	"master/clearwaste/internal/world/login"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	registry := network.NewRegistry()
	if err := registerCodecs(registry); err != nil {
		log.Fatal(err)
	}
	validator := login.NewDevelopmentValidator("", network.Identity{})
	if development := cfg.DevelopmentLogin; development != nil {
		validator = login.NewDevelopmentValidator(development.Ticket, network.Identity{
			AccountID:   account.ID(development.AccountID),
			CharacterID: character.ID(development.CharacterID),
		})
	}
	runtimeState, err := world.NewState(cfg.ContentRoot)
	if err != nil {
		log.Fatal(err)
	}
	handler := login.NewHandler(cfg.ProtocolVersion, validator, runtimeState)
	listener, err := transport.ListenTCP(cfg.TCPAddress)
	if err != nil {
		log.Fatal(err)
	}
	server := network.NewServer(listener, registry, network.ConnectionConfig{
		MaxPayloadSize:        cfg.MaxPayloadSize,
		InboundQueueCapacity:  cfg.InboundQueueCapacity,
		OutboundQueueCapacity: cfg.OutboundQueueCapacity,
	}, handler)
	defer server.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()

	if err := server.Serve(); err != nil {
		log.Printf("server stopped: %v", err)
	}
}

func registerCodecs(registry *network.Registry) error {
	if err := login.RegisterCodecs(registry); err != nil {
		return err
	}
	if err := movement.RegisterCodecs(registry); err != nil {
		return err
	}
	if err := interaction.RegisterCodecs(registry); err != nil {
		return err
	}
	return bootstrap.RegisterCodecs(registry)
}
