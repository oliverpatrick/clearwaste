# Runnable Account Server and Authenticated Godot Boot Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run one development account login over HTTP and keep the Godot world unrendered until that login succeeds.

**Architecture:** A dedicated account configuration seeds an in-memory account repository, a login service, and a standard-library HTTP handler wired by `internal/app/accountserver`. Godot registers its existing clients as autoloads, submits the visible login form to the account server, and only initializes the existing terrain scene after a valid response.

**Tech Stack:** Go 1.25 standard library, Godot 4.7/GDScript, `net/http`, Godot `HTTPRequest`.

**Spec:** `docs/superpowers/specs/2026-09-01-runnable-account-server-and-godot-login-design.md`

## Global Constraints

- Keep the account process development-only: no database, OAuth, registration, character selection, or production ticket issuer.
- Use only the Go standard library; add no dependencies.
- Reuse `WORLD_DEV_LOGIN_TICKET`, `WORLD_DEV_ACCOUNT_ID`, and `WORLD_DEV_CHARACTER_ID` so the account and world processes share one opaque-ticket contract.
- Never log or include the configured password or ticket in errors.
- Wrong email and wrong password must produce the same generic authentication failure.
- Do not connect Godot to the world protocol or spawn a character; PLAYER-001 was cancelled.
- Preserve the existing uncommitted WORLD-002 terrain work in `client/project.godot`, `client/scenes/game/game.tscn`, `client/scenes/game/game_app.gd`, and `client/tests/content_registry_test.gd`.
- Before isolating execution in a worktree, re-run the existing Godot terrain test and commit those four WORLD-002 files separately as `feat: render initial region terrain`; never discard them.

---

### Task 1: Dedicated Account-Process Configuration

**Files:**
- Create: `masterserver/config/account.go`
- Create: `masterserver/config/account_test.go`
- Modify: `masterserver/.env.example`

**Interfaces:**
- Consumes: existing `envOptionalUint64` and `envOrDefault` helpers from `masterserver/config/config.go`.
- Produces: `config.LoadAccount() (config.AccountConfig, error)` with `HTTPAddress`, `DevelopmentAccount`, and `WorldEntry` fields.

- [ ] **Step 1: Write failing configuration tests**

Create `masterserver/config/account_test.go` with tests equivalent to:

```go
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
```

- [ ] **Step 2: Run the focused tests and confirm the red state**

Run:

```bash
cd masterserver
GOCACHE=/private/tmp/clearwaste-go-cache go test ./config -run LoadAccount -count=1
```

Expected: build failure because `LoadAccount` and the account configuration types do not exist.

- [ ] **Step 3: Implement the minimal account configuration**

Create `masterserver/config/account.go` with these public shapes:

```go
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
```

Keep error messages limited to configuration field names; never interpolate credential or ticket values.

Update `masterserver/.env.example` to include a complete runnable development set:

```dotenv
ACCOUNT_HTTP_ADDR=:8080
ACCOUNT_DEV_EMAIL=dev@example.com
ACCOUNT_DEV_PASSWORD=development-only
WORLD_TCP_ADDR=:7777
WORLD_PUBLIC_HOST=127.0.0.1
WORLD_DEV_LOGIN_TICKET=development-world-ticket
WORLD_DEV_ACCOUNT_ID=1
WORLD_DEV_CHARACTER_ID=1
WORLD_MAX_PAYLOAD_BYTES=65536
WORLD_INBOUND_QUEUE_CAPACITY=64
WORLD_OUTBOUND_QUEUE_CAPACITY=256
```

- [ ] **Step 4: Format and run the configuration tests**

Run:

```bash
cd masterserver
gofmt -w config/account.go config/account_test.go
GOCACHE=/private/tmp/clearwaste-go-cache go test ./config -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the configuration slice**

```bash
git add masterserver/config/account.go masterserver/config/account_test.go masterserver/.env.example
git commit -m "feat: configure development account server"
```

---

### Task 2: Account Repository and Login Service

**Files:**
- Create: `masterserver/internal/account/account.go`
- Create: `masterserver/internal/account/development_repository.go`
- Create: `masterserver/internal/world/entry.go`
- Create: `masterserver/internal/account/auth/login/service.go`
- Create: `masterserver/internal/account/auth/login/service_test.go`

**Interfaces:**
- Consumes: `account.ID`, `character.ID`, and the configuration values produced by Task 1.
- Produces: `account.Repository.FindByEmail(context.Context, string) (account.Record, error)`, `login.NewService(account.Repository, world.EntryGrant) *login.Service`, and `(*login.Service).Authenticate(context.Context, string, string) (login.Result, error)`.

- [ ] **Step 1: Write failing service tests**

Create `masterserver/internal/account/auth/login/service_test.go`:

```go
package login

import (
	"context"
	"errors"
	"testing"

	"master/clearwaste/internal/account"
	"master/clearwaste/internal/character"
	"master/clearwaste/internal/world"
)

func newTestService() *Service {
	repository := account.NewDevelopmentRepository(
		account.ID(41),
		"dev@example.com",
		"development-only",
		character.ID(73),
	)
	return NewService(repository, world.EntryGrant{Ticket: "opaque-value", Host: "127.0.0.1", Port: 7777})
}

func TestAuthenticateReturnsDefaultCharacterAndWorldEntry(t *testing.T) {
	result, err := newTestService().Authenticate(context.Background(), " DEV@example.com ", "development-only")
	if err != nil {
		t.Fatal(err)
	}
	if result.AccountID != account.ID(41) || result.CharacterID != character.ID(73) {
		t.Fatalf("identity=%+v", result)
	}
	if result.World.Ticket != "opaque-value" || result.World.Host != "127.0.0.1" || result.World.Port != 7777 {
		t.Fatalf("world=%+v", result.World)
	}
}

func TestAuthenticateUsesOneGenericFailure(t *testing.T) {
	for _, credentials := range [][2]string{
		{"missing@example.com", "development-only"},
		{"dev@example.com", "wrong-password"},
		{"", ""},
	} {
		_, err := newTestService().Authenticate(context.Background(), credentials[0], credentials[1])
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("credentials=%q error=%v", credentials[0], err)
		}
		if err.Error() != "invalid credentials" {
			t.Fatalf("non-generic error=%q", err)
		}
	}
}
```

- [ ] **Step 2: Run the service tests and confirm the red state**

Run:

```bash
cd masterserver
GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/account/auth/login -count=1
```

Expected: build failure because the repository, entry grant, and service types do not exist.

- [ ] **Step 3: Implement the account repository and world entry value**

Create `masterserver/internal/account/account.go` with:

```go
package account

import (
	"context"
	"crypto/subtle"
	"errors"

	"master/clearwaste/internal/character"
)

var ErrNotFound = errors.New("account not found")

type Record struct {
	ID                 ID
	Email              string
	DefaultCharacterID character.ID
	password           []byte
}

func (r Record) PasswordMatches(password string) bool {
	return subtle.ConstantTimeCompare(r.password, []byte(password)) == 1
}

type Repository interface {
	FindByEmail(context.Context, string) (Record, error)
}
```

Create `masterserver/internal/account/development_repository.go`:

```go
package account

import (
	"context"
	"strings"

	"master/clearwaste/internal/character"
)

type DevelopmentRepository struct {
	record Record
}

func NewDevelopmentRepository(id ID, email, password string, characterID character.ID) *DevelopmentRepository {
	return &DevelopmentRepository{record: Record{
		ID: id, Email: strings.ToLower(strings.TrimSpace(email)),
		DefaultCharacterID: characterID,
		password: append([]byte(nil), password...),
	}}
}

func (r *DevelopmentRepository) FindByEmail(ctx context.Context, email string) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	if email != r.record.Email {
		return Record{}, ErrNotFound
	}
	return r.record, nil
}
```

Create `masterserver/internal/world/entry.go`:

```go
package world

type EntryGrant struct {
	Ticket string
	Host   string
	Port   int
}
```

- [ ] **Step 4: Implement the login service**

Create `masterserver/internal/account/auth/login/service.go` with:

```go
package login

import (
	"context"
	"errors"
	"strings"

	"master/clearwaste/internal/account"
	"master/clearwaste/internal/character"
	"master/clearwaste/internal/world"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Result struct {
	AccountID   account.ID
	CharacterID character.ID
	World       world.EntryGrant
}

type Service struct {
	accounts account.Repository
	entry    world.EntryGrant
}

func NewService(accounts account.Repository, entry world.EntryGrant) *Service {
	return &Service{accounts: accounts, entry: entry}
}

func (s *Service) Authenticate(ctx context.Context, email, password string) (Result, error) {
	record, err := s.accounts.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || !record.PasswordMatches(password) {
		return Result{}, ErrInvalidCredentials
	}
	return Result{AccountID: record.ID, CharacterID: record.DefaultCharacterID, World: s.entry}, nil
}
```

- [ ] **Step 5: Format and run the focused tests**

Run:

```bash
cd masterserver
gofmt -w internal/account/account.go internal/account/development_repository.go internal/world/entry.go internal/account/auth/login/service.go internal/account/auth/login/service_test.go
GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/account/... ./internal/world/... -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit the domain slice**

```bash
git add masterserver/internal/account masterserver/internal/world/entry.go
git commit -m "feat: authenticate development account"
```

---

### Task 3: HTTP Login Endpoint and Runnable Account Process

**Files:**
- Create: `masterserver/internal/account/auth/login/handler.go`
- Create: `masterserver/internal/account/auth/login/handler_test.go`
- Create: `masterserver/internal/app/accountserver/server.go`
- Create: `masterserver/internal/app/accountserver/server_test.go`
- Modify: `masterserver/cmd/account/main.go`

**Interfaces:**
- Consumes: `login.Service` from Task 2 and `config.AccountConfig` from Task 1.
- Produces: `login.NewHandler(*login.Service) http.Handler` and `accountserver.New(config.AccountConfig) *http.Server`; exposes `POST /v1/login`.

- [ ] **Step 1: Write failing HTTP-handler tests**

Create `masterserver/internal/account/auth/login/handler_test.go` using `newTestService()` from Task 2:

```go
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
		method string
		body   string
		status int
	}{
		{method: http.MethodPost, body: `{`, status: http.StatusBadRequest},
		{method: http.MethodPost, body: strings.Repeat("x", 4097), status: http.StatusBadRequest},
		{method: http.MethodGet, status: http.StatusMethodNotAllowed},
	}
	for _, test := range tests {
		request := httptest.NewRequest(test.method, "/v1/login", strings.NewReader(test.body))
		response := httptest.NewRecorder()
		NewHandler(newTestService()).ServeHTTP(response, request)
		if response.Code != test.status {
			t.Fatalf("method=%s status=%d", test.method, response.Code)
		}
	}
}
```

- [ ] **Step 2: Run the handler tests and confirm the red state**

Run:

```bash
cd masterserver
GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/account/auth/login -run Handler -count=1
```

Expected: build failure because `NewHandler` does not exist.

- [ ] **Step 3: Implement the bounded JSON handler**

Create `masterserver/internal/account/auth/login/handler.go`:

```go
package login

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const maxRequestBytes = 4096

type Handler struct {
	service *Service
}

type request struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type response struct {
	Ticket      string        `json:"ticket"`
	AccountID   uint64        `json:"accountId"`
	CharacterID uint64        `json:"characterId"`
	World       worldResponse `json:"world"`
}

type worldResponse struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func NewHandler(service *Service) http.Handler {
	return &Handler{service: service}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var input request
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	result, err := h.service.Authenticate(r.Context(), input.Email, input.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login failed"})
		return
	}
	writeJSON(w, http.StatusOK, response{
		Ticket: result.World.Ticket, AccountID: uint64(result.AccountID), CharacterID: uint64(result.CharacterID),
		World: worldResponse{Host: result.World.Host, Port: result.World.Port},
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
```

- [ ] **Step 4: Write the failing application-wiring test**

Create `masterserver/internal/app/accountserver/server_test.go`:

```go
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
		DevelopmentAccount: config.DevelopmentAccountConfig{Email: "dev@example.com", Password: "development-only", AccountID: 41, CharacterID: 73},
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
```

- [ ] **Step 5: Implement application wiring and process lifecycle**

Create `masterserver/internal/app/accountserver/server.go`:

```go
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
	repository := account.NewDevelopmentRepository(account.ID(cfg.DevelopmentAccount.AccountID), cfg.DevelopmentAccount.Email, cfg.DevelopmentAccount.Password, character.ID(cfg.DevelopmentAccount.CharacterID))
	service := login.NewService(repository, world.EntryGrant{Ticket: cfg.WorldEntry.Ticket, Host: cfg.WorldEntry.Host, Port: cfg.WorldEntry.Port})
	mux := http.NewServeMux()
	mux.Handle("/v1/login", login.NewHandler(service))
	return &http.Server{Addr: cfg.HTTPAddress, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
}
```

Replace the placeholder `masterserver/cmd/account/main.go` with:

```go
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"master/clearwaste/config"
	"master/clearwaste/internal/app/accountserver"
)

func main() {
	cfg, err := config.LoadAccount()
	if err != nil {
		log.Fatal(err)
	}
	server := accountserver.New(cfg)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("account server shutdown: %v", err)
		}
	}()
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
```

- [ ] **Step 6: Format and run account-server tests**

Run:

```bash
cd masterserver
gofmt -w internal/account/auth/login/handler.go internal/account/auth/login/handler_test.go internal/app/accountserver/server.go internal/app/accountserver/server_test.go cmd/account/main.go
GOCACHE=/private/tmp/clearwaste-go-cache go test ./internal/account/auth/login ./internal/app/accountserver ./cmd/account -count=1
```

Expected: PASS, including successful compilation of `cmd/account`.

- [ ] **Step 7: Smoke-test the actual account process**

From `masterserver`, start `go run ./cmd/account` with the `.env.example` values exported, then POST the documented credentials:

```bash
curl --fail-with-body -X POST http://127.0.0.1:8080/v1/login -H 'Content-Type: application/json' -d '{"email":"dev@example.com","password":"development-only"}'
```

Expected: HTTP 200 JSON containing `development-world-ticket`, account ID `1`, character ID `1`, host `127.0.0.1`, and port `7777`. Stop the process with SIGTERM and confirm clean exit.

- [ ] **Step 8: Commit the runnable server slice**

```bash
git add masterserver/internal/account/auth/login/handler.go masterserver/internal/account/auth/login/handler_test.go masterserver/internal/app/accountserver/server.go masterserver/internal/app/accountserver/server_test.go masterserver/cmd/account/main.go
git commit -m "feat: run development account login server"
```

---

### Task 4: Authenticated Godot Boot Gate

**Files:**
- Modify: `client/project.godot`
- Modify: `client/autoloads/auth_client.gd`
- Modify: `client/autoloads/network_client.gd`
- Modify: `client/scenes/game/game.tscn`
- Modify: `client/scenes/game/game_app.gd`
- Modify: `client/tests/content_registry_test.gd`

**Interfaces:**
- Consumes: the Task 3 response fields `ticket`, `accountId`, `characterId`, and `world.host`/`world.port`; existing `DefinitionRegistry`, `WorldStream`, and `LoginScreen.submitted`.
- Produces: singleton nodes `/root/AuthClient` and `/root/GameNetworkClient`, plus authenticated state fields `ticket`, `account_id`, `character_id`, `world_host`, and `world_port` on `game_app.gd`.

- [ ] **Step 1: Change the Godot smoke test to require the login gate**

In `client/tests/content_registry_test.gd`, keep the existing registry/noise assertions, then replace the immediate-terrain scene assertions with:

```gdscript
	assert(root.get_node_or_null("AuthClient") != null)
	assert(root.get_node_or_null("GameNetworkClient") != null)

	var game := GameScene.instantiate()
	root.add_child(game)
	await process_frame
	var login_screen := game.find_child("LoginScreen", true, false)
	assert(login_screen != null and login_screen.visible)
	assert(game.find_child("Region_0_0_0", true, false) == null)
	assert(root.get_camera_3d() == null)

	game._on_login_succeeded({
		"ticket": "opaque-value",
		"accountId": 41,
		"characterId": 73,
		"world": {"host": "127.0.0.1", "port": 7777},
	})
	await process_frame
	assert(game.find_child("LoginScreen", true, false) == null)
	assert(game.find_child("Region_0_0_0", true, false) != null)
	assert(root.get_camera_3d() != null)
	game.free()
	quit()
```

Also preload `res://autoloads/auth_client.gd` and assert its response validator accepts `opaque-value` and rejects an empty ticket or missing world endpoint.

- [ ] **Step 2: Run the Godot test and confirm the red state**

Run from `client`:

```bash
/Applications/Godot.app/Contents/MacOS/Godot --headless --path . --log-file /private/tmp/clearwaste-login-red.log --script res://tests/content_registry_test.gd
```

Expected: failure because the autoloads/login child are absent and terrain loads before authentication.

- [ ] **Step 3: Register the autoloads and account URL**

In `client/project.godot`, add:

```ini
[account]

base_url="http://127.0.0.1:8080"

[autoload]

AuthClient="*res://autoloads/auth_client.gd"
GameNetworkClient="*res://autoloads/network_client.gd"
```

Remove `class_name AuthClient` from `client/autoloads/auth_client.gd` and `class_name GameNetworkClient` from `client/autoloads/network_client.gd` so their global class names do not collide with singleton names.

- [ ] **Step 4: Validate the full login response without a fixed ticket length**

Add this static boundary to `client/autoloads/auth_client.gd`:

```gdscript
static func is_valid_login_response(response: Variant) -> bool:
	if not response is Dictionary:
		return false
	var world: Variant = response.get("world")
	return not str(response.get("ticket", "")).is_empty() \
		and int(response.get("accountId", 0)) > 0 \
		and int(response.get("characterId", 0)) > 0 \
		and world is Dictionary \
		and not str(world.get("host", "")).is_empty() \
		and int(world.get("port", 0)) > 0 \
		and int(world.get("port", 0)) <= 65535
```

Use it in `_on_request_completed`. Any network error, non-200 response, malformed JSON, or invalid response emits a generic login failure and leaves no ticket stored in the autoload.

- [ ] **Step 5: Put the login screen in the main scene and disable pre-login rendering**

In `client/scenes/game/game.tscn`:

- add `res://ui/screens/login/login_screen.tscn` as a `PackedScene` external resource;
- instantiate it as a direct `LoginScreen` child of `Game`;
- change the existing `Camera.current` value from `true` to `false`.

Do not instantiate a player scene.

- [ ] **Step 6: Wire login and move terrain initialization behind success**

Replace the eager `_ready` flow in `client/scenes/game/game_app.gd` with these fields and handlers:

```gdscript
@onready var login_screen: LoginScreen = $LoginScreen
@onready var world_stream: WorldStream = $WorldStream
@onready var camera: Camera3D = $Camera

var ticket := ""
var account_id := 0
var character_id := 0
var world_host := ""
var world_port := 0

func _ready() -> void:
	login_screen.submitted.connect(_on_login_submitted)
	AuthClient.login_succeeded.connect(_on_login_succeeded)
	AuthClient.login_failed.connect(_on_login_failed)

func _on_login_submitted(email: String, password: String) -> void:
	AuthClient.login(str(ProjectSettings.get_setting("account/base_url", "http://127.0.0.1:8080")), email, password)

func _on_login_failed(message: String) -> void:
	login_screen.show_error(message)

func _on_login_succeeded(response: Dictionary) -> void:
	ticket = str(response.ticket)
	account_id = int(response.accountId)
	character_id = int(response.characterId)
	world_host = str(response.world.host)
	world_port = int(response.world.port)
	login_screen.hide()
	login_screen.queue_free()
	_load_world()
```

Move the existing registry load, `world_stream.configure`, `load_region("0:0:0")`, and `camera.look_at` statements into `_load_world()`. Set `camera.current = true` only after the region loads successfully. On content/region failure, keep the camera inactive and report the existing error.

- [ ] **Step 7: Run Godot tests and the main-scene boot**

Run from `client`:

```bash
/Applications/Godot.app/Contents/MacOS/Godot --headless --path . --log-file /private/tmp/clearwaste-login-test.log --script res://tests/content_registry_test.gd
/Applications/Godot.app/Contents/MacOS/Godot --headless --path . --log-file /private/tmp/clearwaste-login-boot.log --quit-after 3
```

Expected: both commands exit 0; the test confirms no terrain/camera before success and terrain/camera after success. Inspect both logs for parser errors, invalid resources, and leaked credentials/tickets.

- [ ] **Step 8: Commit the authenticated client boot**

```bash
git add client/project.godot client/autoloads/auth_client.gd client/autoloads/network_client.gd client/scenes/game/game.tscn client/scenes/game/game_app.gd client/tests/content_registry_test.gd
git commit -m "feat: gate Godot world behind account login"
```

---

### Task 5: Full Verification

**Files:**
- Verify only; no production files should change.

**Interfaces:**
- Consumes: all server and client deliverables from Tasks 1-4.
- Produces: evidence that the repository builds, tests, and boots without protocol or startup regressions.

- [ ] **Step 1: Run Go formatting, tests, and vet**

Run:

```bash
cd masterserver
gofmt -w config/account.go config/account_test.go internal/account/account.go internal/account/development_repository.go internal/account/auth/login/service.go internal/account/auth/login/service_test.go internal/account/auth/login/handler.go internal/account/auth/login/handler_test.go internal/world/entry.go internal/app/accountserver/server.go internal/app/accountserver/server_test.go cmd/account/main.go
GOCACHE=/private/tmp/clearwaste-go-cache go test ./... -count=1
GOCACHE=/private/tmp/clearwaste-go-cache go vet ./...
```

Expected: all commands exit 0.

- [ ] **Step 2: Run final Godot verification**

Run from `client`:

```bash
/Applications/Godot.app/Contents/MacOS/Godot --headless --path . --log-file /private/tmp/clearwaste-login-final-test.log --script res://tests/content_registry_test.gd
/Applications/Godot.app/Contents/MacOS/Godot --headless --path . --log-file /private/tmp/clearwaste-login-final-boot.log --quit-after 3
```

Expected: both commands exit 0 and neither log contains script errors, resource errors, passwords, or tickets.

- [ ] **Step 3: Check the final diff and history**

Run:

```bash
git diff --check
git status --short
git log --oneline -6
```

Expected: no whitespace errors, no unintended files, and separate commits for preserved WORLD-002 work, account configuration, account authentication, runnable HTTP server, and gated Godot boot.
