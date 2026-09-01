package login

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerReturnsLoginGrant(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/login", strings.NewReader(`{"email":"dev@example.com","password":"development-only"}`))
	response := httptest.NewRecorder()
	NewHandler(newTestService()).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("content type=%q", response.Header().Get("Content-Type"))
	}
	var body struct {
		Ticket      string `json:"ticket"`
		AccountID   uint64 `json:"accountId"`
		CharacterID uint64 `json:"characterId"`
		World       struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		} `json:"world"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Ticket != "opaque-value" || body.AccountID != 41 || body.CharacterID != 73 || body.World.Host != "127.0.0.1" || body.World.Port != 7777 {
		t.Fatalf("body=%+v", body)
	}
}

func TestHandlerRejectsInvalidCredentialsGenerically(t *testing.T) {
	for _, payload := range []string{
		`{"email":"missing@example.com","password":"development-only"}`,
		`{"email":"dev@example.com","password":"wrong-password"}`,
	} {
		request := httptest.NewRequest(http.MethodPost, "/v1/login", strings.NewReader(payload))
		response := httptest.NewRecorder()
		NewHandler(newTestService()).ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized || response.Body.String() != "{\"error\":\"login failed\"}\n" {
			t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
		}
	}
}

func TestHandlerRejectsMalformedOversizedAndWrongMethod(t *testing.T) {
	tests := []struct {
		name   string
		method string
		body   string
		status int
	}{
		{name: "malformed", method: http.MethodPost, body: `{`, status: http.StatusBadRequest},
		{name: "unknown field", method: http.MethodPost, body: `{"email":"dev@example.com","password":"development-only","admin":true}`, status: http.StatusBadRequest},
		{name: "trailing object", method: http.MethodPost, body: `{"email":"dev@example.com","password":"development-only"}{}`, status: http.StatusBadRequest},
		{name: "oversized", method: http.MethodPost, body: strings.Repeat("x", 4097), status: http.StatusBadRequest},
		{name: "wrong method", method: http.MethodGet, status: http.StatusMethodNotAllowed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "/v1/login", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			NewHandler(newTestService()).ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
			}
		})
	}
}
