# Godot Client Protocol and Content Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the Godot client authenticate against the Go world protocol, then load and render all development regions with numeric content and asset registries.

**Architecture:** Keep the Go six-byte frame and login opcodes as the wire contract. Normalize every region into the existing 65×65 terrain representation, preserving shared world-edge samples. Keep Godot scene bindings in a client-only numeric asset registry and reveal world presentation only after successful login and world authentication.

**Tech Stack:** Godot 4 GDScript, existing Go TCP protocol, JSON content_data, Godot PackedScene assets.

**Spec:** Approved design in conversation for CLIENT-001, PROTO-001, CLIENT-002, CONTENT-003, CLIENT-003.

## Global Constraints

- Do not change the Go wire format or canonical content JSON.
- Do not put Godot resource paths in content_data.
- Preserve the existing 65×65 RegionMeshBuilder input shape.
- World terrain and character presentation must remain hidden until login and world handshake succeed.
- Use the smallest coherent change; no binary content compiler or content validation work.

### Task 1: Replace stale Godot framing with the Go protocol

**Files:**
- Modify: `client/core/network/protocol.gd`
- Modify: `client/autoloads/network_client.gd`
- Delete/retire: stale duplicate four-byte encoder/decoder/protocol files if unreferenced
- Test: `client/tests/content_registry_test.gd` or a focused protocol test script

Implement six-byte big-endian frames and ClientHello/ServerHello/LoginRequest/LoginAccepted/LoginRejected state handling, including fragmented and coalesced TCP reads. Add golden byte assertions for frame and handshake payloads.

### Task 2: Wire authenticated world connection into the app

**Files:**
- Modify: `client/scripts/game/game_app.gd`
- Modify: `client/scenes/game/game.tscn` only if signal/node wiring requires it
- Test: Godot smoke test

Pass the login response world host, port, and opaque ticket to GameNetworkClient. Load and reveal all regions only after the world login handshake emits success; show a useful failure and keep world hidden on rejection.

### Task 3: Load and normalize all development content

**Files:**
- Modify: `client/core/content/definition_registry.gd`
- Modify: `client/core/content/map_loader.gd`
- Modify: `client/scripts/game/world/world_stream.gd`
- Test: Godot content/terrain smoke test

Load manifest, items, mobs, objects, and every `map/region_*.json`. Normalize terrain defaults and additive overrides into world-coordinate 65×65 height grids, ensuring neighboring region border vertices reuse the same samples. Preserve terrain colors and placements as additive normalized fields.

### Task 4: Add numeric Godot-only asset bindings

**Files:**
- Create: `client/core/content/asset_registry.gd`
- Modify: `client/core/content/definition_registry.gd` if a lookup is needed
- Modify: `client/scripts/game/entities/objects/object_registry.gd`
- Modify: `client/scripts/game/entities/player_registry.gd`
- Test: Godot registry smoke test

Map numeric itemId, npcId, objectId, and appearanceId to client PackedScenes/resources. Replace symbolic comparisons and lookups in presentation registries with numeric IDs and a safe fallback for unbound development assets.

### Task 5: Verify and commit

Run `gofmt`, `go test ./... -count=1`, `go vet ./...`, Godot headless content tests, and `git diff --check`; commit the completed ticket set on the feature branch.
