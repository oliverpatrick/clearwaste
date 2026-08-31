package main

import (
	"testing"

	"master/clearwaste/internal/engine/network"
	"master/clearwaste/internal/game/entity"
	"master/clearwaste/internal/game/interaction"
	"master/clearwaste/internal/game/movement"
	"master/clearwaste/internal/world/login"
)

func TestRegisterCodecsIncludesLoginAndGameplayRequests(t *testing.T) {
	registry := network.NewRegistry()
	if err := registerCodecs(registry); err != nil {
		t.Fatal(err)
	}
	messages := []network.Message{
		login.ClientHello{ProtocolVersion: 1},
		movement.MoveRequest{Direction: movement.North},
		movement.SetRunEnabled{Enabled: true},
		interaction.InteractRequest{TargetID: entity.ID(1), Action: interaction.Chop},
	}
	for _, message := range messages {
		if _, err := registry.Encode(message); err != nil {
			t.Fatalf("%T is not registered: %v", message, err)
		}
	}
}
