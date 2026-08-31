# World login handshake

The world server starts each session in `Handshake`. The only successful path is:

```text
Handshake
  ClientHello(protocol version)
  ServerHello(accepted, server protocol version)
Login
  LoginRequest(opaque ticket)
  LoginAccepted
Game
```

`ClientHello` and `LoginRequest` are handled by `internal/world/login.Handler`; they never enter the session's gameplay queue. A decoded message reaches `Session.Inbound()` only when it implements `network.GameplayMessage` and the session is in `Game`. The login handler sends responses through `Session.Send`, preserving the connection's single writer goroutine.

The session owns state transitions and the authenticated `AccountID` and `CharacterID`. It never stores the raw ticket. Closing the connection moves the session to `Closed`, after which transitions and inbound enqueueing fail.

## Development validator

The temporary validator is configured through `config.Load`:

- `WORLD_PROTOCOL_VERSION` defaults to `1`.
- `WORLD_DEV_LOGIN_TICKET` has no default.
- `WORLD_DEV_ACCOUNT_ID` has no default and must be a non-zero integer when supplied.
- `WORLD_DEV_CHARACTER_ID` has no default and must be a non-zero integer when supplied.

All three development-login values must be present for validation to be available. Missing or partial configuration lets the server start but rejects every login. A malformed or zero configured ID fails startup. Ticket values are credentials and must not be logged.

The client sends only the opaque ticket. `login.LoginValidator` resolves it to a typed `network.Identity`; failed validation produces a generic `LoginRejected` response. The development implementation maps one configured ticket to one configured identity without database access.

Future account infrastructure should add a signed, short-lived world-ticket implementation of `login.LoginValidator` and replace validator construction in `cmd/server`. Packet decoding, handler state transitions, and the session identity contract do not change.
