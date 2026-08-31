package login

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"master/clearwaste/internal/engine/network"
	"master/clearwaste/internal/game/entity"
	"master/clearwaste/internal/game/interaction"
	"master/clearwaste/internal/game/movement"
)

func TestConcreteGameplayRequestsRequireGameState(t *testing.T) {
	requests := []network.Message{
		movement.MoveRequest{Direction: movement.North},
		movement.SetRunEnabled{Enabled: true},
		interaction.InteractRequest{TargetID: entity.ID(4812), Action: interaction.Chop},
	}
	for _, request := range requests {
		t.Run(reflect.TypeOf(request).Name(), func(t *testing.T) {
			connection, peer, registry := startLoginConnection(t, configuredValidator())
			sendLoginMessage(t, peer, registry, request)
			if err := connection.Wait(); !errors.Is(err, ErrGameplayBeforeLogin) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestConcreteGameplayRequestsOnlyEnterAuthenticatedInbound(t *testing.T) {
	connection, peer, registry := startLoginConnection(t, configuredValidator())
	completeLogin(t, connection, peer, registry)

	want := []network.Message{
		movement.MoveRequest{Direction: movement.NorthEast},
		movement.SetRunEnabled{Enabled: true},
		interaction.InteractRequest{TargetID: entity.ID(4812), Action: interaction.Mine},
	}
	identityBefore, ok := connection.Session().Identity()
	if !ok {
		t.Fatal("authenticated session has no identity")
	}
	for _, request := range want {
		sendLoginMessage(t, peer, registry, request)
	}
	for _, expected := range want {
		select {
		case got := <-connection.Session().Inbound():
			if !reflect.DeepEqual(got, expected) {
				t.Fatalf("got=%#v want=%#v", got, expected)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for %T", expected)
		}
	}
	identityAfter, ok := connection.Session().Identity()
	if !ok || identityAfter != identityBefore || connection.Session().State() != network.StateGame {
		t.Fatalf("request changed session identity/state: identity=%+v state=%v", identityAfter, connection.Session().State())
	}
}

func TestGameplayQueueOverflowClosesOnlyOffendingConnection(t *testing.T) {
	noisy, noisyPeer, noisyRegistry := startLoginConnectionWithCapacity(t, configuredValidator(), 1)
	healthy, healthyPeer, healthyRegistry := startLoginConnectionWithCapacity(t, configuredValidator(), 1)
	completeLogin(t, noisy, noisyPeer, noisyRegistry)
	completeLogin(t, healthy, healthyPeer, healthyRegistry)

	sendLoginMessage(t, noisyPeer, noisyRegistry, movement.MoveRequest{Direction: movement.North})
	sendLoginMessage(t, noisyPeer, noisyRegistry, movement.SetRunEnabled{Enabled: true})
	if err := noisy.Wait(); !errors.Is(err, network.ErrInboundBackpressure) {
		t.Fatalf("noisy error=%v", err)
	}

	want := interaction.InteractRequest{TargetID: entity.ID(99), Action: interaction.Chop}
	sendLoginMessage(t, healthyPeer, healthyRegistry, want)
	select {
	case got := <-healthy.Session().Inbound():
		if got != want {
			t.Fatalf("healthy message=%#v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("healthy request was not delivered")
	}
	select {
	case <-healthy.Done():
		t.Fatal("healthy connection closed")
	default:
	}
}
