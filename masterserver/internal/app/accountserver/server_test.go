package accountserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"master/clearwaste/config"
)

func TestNewWiresLoginRoute(t *testing.T) {
	cfg := config.AccountConfig{
		HTTPAddress: ":0",
		DevelopmentAccount: config.DevelopmentAccountConfig{
			Email: "dev@example.com", Password: "development-only", AccountID: 41, CharacterID: 73,
		},
		WorldEntry: config.WorldEntryConfig{Ticket: "opaque-value", Host: "127.0.0.1", Port: 7777},
	}
	server := New(cfg)
	request := httptest.NewRequest(http.MethodPost, "/v1/login", strings.NewReader(`{"email":"dev@example.com","password":"development-only"}`))
	response := httptest.NewRecorder()
	server.Handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
