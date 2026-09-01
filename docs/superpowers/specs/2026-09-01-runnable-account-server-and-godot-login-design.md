# Runnable Account Server and Authenticated Godot Boot

## Scope

This design covers APP-001 and CLIENT-001:

- run an HTTP account process with one environment-configured development account and default character;
- authenticate through `POST /v1/login` and return the opaque ticket already accepted by the world server;
- boot Godot into the login screen;
- load and render the existing `region_0_0` terrain only after account login succeeds.

Production authentication, registration, databases, OAuth, character selection, world selection, world-protocol connection, and player spawning remain out of scope. PLAYER-001 was cancelled, so successful login reveals terrain but does not create a character.

## Server architecture

Use explicit domain boundaries selected for this ticket:

- `internal/account` owns the account record and repository contract;
- an in-memory development repository contains the single configured account and its default character;
- `internal/account/auth/login` owns credential authentication and the HTTP handler;
- `internal/world` owns the configured world-entry grant and public endpoint returned to the client;
- `internal/app/accountserver` wires the repository, login service, handler, and HTTP server;
- `cmd/account` only loads configuration, starts the application, and handles graceful shutdown.

The implementation uses only Go's standard library. It does not add storage or authentication dependencies.

## Configuration

The account process loads a dedicated account configuration so missing account credentials do not prevent the world process from starting.

Required account-process values:

- `ACCOUNT_DEV_EMAIL`;
- `ACCOUNT_DEV_PASSWORD`;
- `WORLD_DEV_LOGIN_TICKET`;
- `WORLD_DEV_ACCOUNT_ID`;
- `WORLD_DEV_CHARACTER_ID`.

Endpoint values:

- `ACCOUNT_HTTP_ADDR`, default `:8080`;
- `WORLD_PUBLIC_HOST`, default `127.0.0.1`;
- `WORLD_TCP_ADDR`, whose port is reused for the public world endpoint.

Missing required values, invalid or zero IDs, and an invalid world TCP address fail account-process startup. Example development values are documented in `masterserver/.env.example`. Credentials and tickets must never appear in logs or error messages.

The Godot account URL is stored in the `account/base_url` project setting and defaults to `http://127.0.0.1:8080`.

## Login contract

`POST /v1/login` accepts a bounded JSON body:

```json
{
  "email": "dev@example.com",
  "password": "development-only"
}
```

The service trims and lowercases the email, queries the account repository, and compares the configured password in constant time.

Success returns HTTP 200:

```json
{
  "ticket": "opaque-value",
  "accountId": 1,
  "characterId": 1,
  "world": {
    "host": "127.0.0.1",
    "port": 7777
  }
}
```

Wrong credentials return the same generic HTTP 401 response. Malformed requests return HTTP 400, unsupported methods return HTTP 405, and JSON responses declare their content type. The handler does not log request bodies, passwords, or tickets.

## Godot flow

`project.godot` registers the existing authentication and game-network client scripts as single autoload instances and retains the existing game scene as the main scene.

At startup:

1. the login screen is visible;
2. no region mesh has been instantiated;
3. the world camera is not active;
4. the login screen submits credentials to the authentication autoload using `account/base_url`.

On failure, the login screen remains visible, shows a generic error, and re-enables its form.

On success, the application retains the response fields separately, removes the login screen, loads content, instantiates `region_0_0`, and activates the terrain camera. It does not connect to the world server or spawn a character in these tickets.

The authentication client accepts a non-empty opaque ticket rather than enforcing the obsolete 43-character ticket length. It also validates the account ID, character ID, world host, and world port needed by the approved response contract.

## Verification

Go tests cover:

- account configuration success and failure;
- successful authentication and returned identity/world grant;
- generic rejection for invalid credentials;
- malformed and oversized JSON;
- unsupported HTTP methods;
- responses and errors that do not expose credentials or tickets.

The Godot headless test verifies:

- both autoloads are present exactly once;
- the login screen is visible immediately after boot;
- terrain is absent and the world camera is inactive before login;
- simulated login success removes the login screen, creates the region terrain, and activates the camera.

Final verification runs `go test ./...`, `go vet ./...`, the existing Godot headless test runner, and a short headless boot of the configured main scene.
