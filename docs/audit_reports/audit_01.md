# Audit result

  The repository has a solid Go transport/login foundation, but the playable stationary-world slice is not
  integrated. A protocol-correct test client can authenticate directly with the world server; the shipped Godot
  client cannot. There is no runnable account server, content boot pipeline, ECS world, character spawn, bootstrap
  replication, or runnable Godot application flow.

  No repository files were modified.

  Verification performed:

  - go test ./...: passes with loopback networking enabled.
  - go vet ./...: passes.
  - go build ./cmd/server: succeeds.
  - go build ./cmd/account: fails because main is undeclared.
  - content_data/manifest.json: fails JSON parsing.
  - No Godot tests were found.

  ## 1. Repository map

  Only vertical-slice-relevant paths are shown.

  clearwaste/
  ├── AGENTS.md
  ├── docs/
  │   ├── architecture.md
  │   ├── content.md
  │   ├── protocol.md
  │   └── simulation.md
  │
  ├── content_data/
  │   ├── manifest.json                    # invalid JSON
  │   ├── items/
  │   │   ├── bronze_axe.json
  │   │   ├── bronze_pickaxe.json
  │   │   └── coin.json
  │   ├── mobs/human/
  │   │   ├── human_male_1.json
  │   │   └── human_female_1.json
  │   ├── map/
  │   │   ├── region_0_0.json
  │   │   ├── region_0_1.json
  │   │   ├── region_1_0.json
  │   │   └── region_1_1.json
  │   ├── protocol/
  │   │   └── golden.json                 # obsolete protocol vectors
  │   └── schema/                         # empty
  │
  ├── masterserver/
  │   ├── cmd/
  │   │   ├── account/main.go             # package only; not runnable
  │   │   └── server/
  │   │       ├── main.go                 # runnable TCP/login server
  │   │       └── main_test.go
  │   ├── config/
  │   │   ├── config.go
  │   │   └── config_test.go
  │   └── internal/
  │       ├── account/id.go
  │       ├── character/id.go
  │       ├── engine/network/
  │       │   ├── server.go
  │       │   ├── connection.go
  │       │   ├── session.go
  │       │   ├── registry.go
  │       │   ├── opcode/
  │       │   ├── protocol/
  │       │   └── transport/
  │       ├── game/
  │       │   ├── entity/id.go
  │       │   ├── movement/
  │       │   └── interaction/
  │       ├── world/login/
  │       │   ├── handler.go
  │       │   ├── validator.go
  │       │   ├── packets.go
  │       │   └── codecs.go
  │       ├── content/{definitions,loader}/ # empty
  │       └── engine/ecs/                   # empty
  │
  └── client/
      ├── project.godot                    # no main scene/autoloads
      ├── autoloads/
      │   ├── auth_client.gd
      │   └── network_client.gd
      ├── core/
      │   ├── content/                     # placeholder loaders
      │   └── network/                     # obsolete protocol
      ├── ui/screens/
      │   ├── startup/
      │   └── login/
      ├── scenes/
      │   ├── game/game.tscn
      │   ├── game/game_app.gd             # empty
      │   ├── player/
      │   ├── mobs/
      │   └── items/
      └── scripts/game/
          ├── click_to_move.gd
          ├── world/
          │   ├── world_stream.gd
          │   ├── region_mesh_builder.gd
          │   └── terrain_height.gd
          └── entities/
              ├── player/player_registry.gd
              └── objects/object_registry.gd

  There are no implemented simulation or spatial packages. Several intended account, content, ECS, player, NPC,
  storage, and application packages exist only as empty directories.

  The audit covers the current workspace. Notably, masterserver/cmd/account and masterserver/go.mod are currently
  untracked in the nested masterserver Git working tree.

  ## 2. Existing functionality by target-flow step

     #    Status      Evidence
  ━━━━━  ━━━━━━━━━━  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
     1    MISSING     masterserver/cmd/account/main.go:1 contains only package main. Building it fails with
                      “function main is undeclared.”
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
     2    PARTIAL     masterserver/cmd/server/main.go:19 builds and starts TCP networking, but it boots no content,
                      world, ECS, simulation, or spatial state.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
     3    MISSING     client/project.godot:11 has no run/main_scene or autoload configuration.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
     4    MISSING     Config.ContentRoot exists, but masterserver/cmd/server/main.go:19 never consumes it. The
                      manifest is also invalid JSON.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
     5    MISSING     Item and mob JSON files exist, but masterserver/internal/content/loader and definitions are
                      empty. No object definitions exist.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
     6    MISSING     Four region JSON files exist, but no server map/region loader exists.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
     7    MISSING     No authoritative world state is constructed. internal/engine/ecs is empty.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
     8    PARTIAL     The configured ticket maps to typed IDs through masterserver/config/config.go:111 and
                      masterserver/internal/world/login/validator.go:38. There is no account record, character
                      model, name, spawn, appearance, or credentials.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
     9    MISSING     client/core/content/definition_registry.gd and the other content scripts are placeholders.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    10    MISSING     There is no numeric definition-ID-to-scene registry. Existing presentation code uses
                      hardcoded strings and scene preloads.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    11    MISSING     A login scene exists, but no main scene or coordinator displays it. See client/ui/screens/
                      login/login_screen.tscn:5.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    12    PARTIAL     client/ui/screens/login/login_screen.gd:11 validates fields and emits submitted, but nothing
                      listens to it.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    13    PARTIAL     client/autoloads/auth_client.gd:12 implements POST /v1/login, but it is not instantiated and
                      no endpoint exists.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    14    MISSING     No backend process or login endpoint validates email/password.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    15    MISSING     The nonexistent backend returns nothing. AuthClient only expects a 43-character ticket and
                      does not require account/character identity.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    16    MISSING     The login scene contains a static “World [number]” button, but no selected/configured world
                      is supplied to a coordinator.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    17    PARTIAL     AuthClient can parse a ticket-shaped JSON response, but no issuer exists and its fixed 43-
                      character assumption does not match the world validator’s arbitrary nonempty configured
                      ticket.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    18    PARTIAL     client/autoloads/network_client.gd:21 opens TCP, but is unwired and implements an
                      incompatible protocol.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    19    COMPLETE    For the current development mechanism, masterserver/internal/world/login/validator.go:20
                      validates the opaque configured ticket and returns typed identity without retaining the raw
                      ticket.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    20    MISSING     Successful validation only calls masterserver/internal/engine/network/session.go:90. No
                      character loader exists.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    21    MISSING     No ECS exists and no runtime player entity is created.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    22    MISSING     There is no player entity to remain stationary. The server does correctly avoid applying
                      movement inside networking.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    23    MISSING     No region lookup, spatial index, visibility service, or spawn-region calculation exists.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    24    MISSING     The server opcode contract ends at opcode 8 in masterserver/internal/engine/network/opcode/
                      opcode.go:6. There are no bootstrap/spawn outbound messages.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    25    PARTIAL     client/scripts/game/world/world_stream.gd:11 exists, but expects an undefined normalized
                      bundle and is not wired.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    26    PARTIAL     client/scripts/game/world/region_mesh_builder.gd:7 can generate terrain from a 65×65 height
                      array, but actual region JSON uses default/override height data and has no embedded region
                      coordinates.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    27    PARTIAL     The old client protocol decodes entity messages in client/core/network/protocol.gd:124, but
                      the Go server emits none and the wire protocol is incompatible.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    28    MISSING     Old ENTITY_SPAWN decodes definition_id as a string in client/core/network/protocol.gd:134,
                      rather than using the stable numeric registry contract.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    29    MISSING     client/scripts/game/entities/objects/object_registry.gd:42 branches on strings such as
                      resource.mutated_tree; client/scripts/game/entities/player/player_registry.gd:63 has
                      hardcoded presentation assumptions.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    30    PARTIAL     Player/NPC/object/ground-item presentation handlers exist, but are unwired, use stale
                      packets, and reference missing or incorrect resources.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    31    PARTIAL     client/scenes/player/player_camera.gd:13 implements follow/orbit/zoom, but client/scenes/
                      game/game.tscn:15 does not instantiate it.
  ─────  ──────────  ───────────────────────────────────────────────────────────────────────────────────────────────
    32    PARTIAL     No server movement simulation exists, which preserves stationarity. However, the client
                      contains obsolete client/scripts/game/click_to_move.gd:93 and destination-authoritative
                      movement encoding. It is currently unwired.

  No step required an UNKNOWN classification; the relevant paths were observable.

  ### Existing subsystem inventory

  - Account/login: world-side development ticket validation only.
  - Character model: typed character.ID only.
  - ECS: missing.
  - Simulation/tick: configuration exists; implementation missing.
  - Spatial: missing.
  - Item definitions: three JSON definitions.
  - NPC definitions: two inconsistent JSON definitions.
  - Ground items: empty arrays in region files; no Go runtime support. An obsolete client quantity handler exists.
  - Objects: string-based map placement exists; no definition registry or server runtime support.
  - Schemas: content_data/schema is empty.
  - Protocol foundation: implemented and tested on Go.
  - Login protocol: implemented and tested on Go.
  - Bootstrap/spawn protocol: missing on Go; obsolete draft exists in Godot.
  - Godot tests: none found.

  ## 3. Current actual flow

  ### When the server starts

  cmd/server:

  1. Loads all configuration through masterserver/config/config.go:44.
  2. Registers login, movement, run, and interaction codecs.
  3. Constructs either a configured DevelopmentValidator or an unavailable validator.
  4. Listens on WORLD_TCP_ADDR.
  5. Accepts connections and creates bounded sessions.
  6. Handles framing, handshake, login, outbound writing, disconnects, and gameplay queueing.

  It does not:

  - read ContentRoot;
  - start the configured tick;
  - load definitions or maps;
  - create an ECS;
  - construct a world;
  - drain authenticated gameplay queues;
  - create a player;
  - send world state.

  The account command cannot start.

  ### When the client starts

  There is no configured Godot main scene. AuthClient and GameNetworkClient are named “autoloads” by directory only;
  they are not registered as autoloads in project.godot.

  The game scene contains only lighting/environment. game_app.gd is empty.

  Therefore, the current source does not produce an application boot, startup screen, login screen, or network
  coordinator.

  ### When login is attempted

  If the isolated login scene is opened manually:

  1. The UI checks that email and password are nonempty.
  2. It emits submitted.
  3. No listener calls AuthClient.

  If AuthClient.login is called manually:

  1. It sends JSON to /v1/login.
  2. No Go HTTP endpoint exists.
  3. Login fails as an unreachable/backend error.

  If GameNetworkClient is called manually:

  1. It sends old CONNECT and CONTENT_HELLO frames.
  2. Those frames use a 4-byte header and old payloads.
  3. The Go server expects a 6-byte header and ClientHello.
  4. The connection is rejected/closed as malformed protocol.

  A custom client using the current Go protocol can:

  1. send ClientHello;
  2. receive ServerHello;
  3. send the configured opaque ticket;
  4. enter StateGame;
  5. send gameplay requests into the bounded session queue.

  Nothing consumes those requests.

  ### When the world is loaded

  No actual world-loading path exists.

  The Godot WorldStream/RegionMeshBuilder classes are not instantiated, and the server never supplies a region
  identity or bootstrap. Even manual construction would require a bundle representation that no loader creates and
  that does not match the current region JSON.

  ## 4. Architecture mismatches

  ### Wire protocol split

  The Go and Godot implementations are two incompatible protocols.

   Concern                Go                                        Godot/current golden data
  ━━━━━━━━━━━━━━━━━━━━━  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   Frame header           uint16 opcode + uint32 length, 6 bytes    uint16 opcode + uint16 length, 4 bytes
  ─────────────────────  ────────────────────────────────────────  ─────────────────────────────────────────────────
   Login                  ClientHello, ServerHello, LoginRequest    CONNECT, CONTENT_HELLO, CONNECT_ACK
  ─────────────────────  ────────────────────────────────────────  ─────────────────────────────────────────────────
   Move                   direction byte, opcode 6                  destination coordinates/mode/sequence, opcode
                                                                    0x10
  ─────────────────────  ────────────────────────────────────────  ─────────────────────────────────────────────────
   Run                    explicit opcode 7                         embedded movement mode
  ─────────────────────  ────────────────────────────────────────  ─────────────────────────────────────────────────
   Interaction target     runtime uint64 EntityID, opcode 8         uint32, opcode 0x11
  ─────────────────────  ────────────────────────────────────────  ─────────────────────────────────────────────────
   Definition identity    intended numeric stable ID                string in ENTITY_SPAWN
  ─────────────────────  ────────────────────────────────────────  ─────────────────────────────────────────────────
   Payload max            configurable, default 64 KiB              hardcoded 32 KiB

  Evidence: masterserver/internal/engine/network/protocol/decoder.go:10, client/core/network/protocol.gd:4, and
  content_data/protocol/golden.json:1.

  This directly conflicts with the single wire contract described in docs/protocol.md:1.

  ### Session enters Game too early

  masterserver/internal/world/login/handler.go:41 transitions directly from validated ticket to Game and sends
  LoginAccepted.

  The documented architecture expects ticket validation, character loading, ECS entity creation, simulation
  insertion, and initial world setup before gameplay begins. Today Game only means “identity validated.”

  ### Canonical content is not currently loadable

  - content_data/manifest.json:1 is invalid JSON.
  - content_data/schema is empty.
  - Regions reference nonexistent ../../schemas/region.schema.json.
  - Map files contain no explicit region coordinates.
  - Map NPC/object references are strings, despite the numeric stable-ID contract.
  - game_object_salvage_pile has no object definition.
  - Male and female NPC definitions use inconsistent stat layouts.

  This conflicts with the validation, registry, and cross-reference requirements in docs/content.md:694.

  ### Configured content location does not match the repository

  masterserver/config/config.go:94 defaults to game_content, while the canonical directory is content_data. It is
  currently unused, so the mismatch is latent.

  ### No documented simulation architecture is implemented

  The required authoritative ECS, immutable/read-only snapshot, Decide, Apply, spatial rebuild, and visibility
  phases in docs/simulation.md:11 are absent.

  Networking appropriately does not mutate gameplay state, but there is no consumer beyond Session.Inbound().

  ### Godot presentation is coupled to obsolete symbolic IDs

  client/scripts/game/entities/objects/object_registry.gd:45 recognizes hardcoded strings such as:

  - resource.mutated_tree
  - item.basic_axe

  This conflicts with the stable definition-ID/client asset-registry boundary in docs/content.md:628.

  ### Broken Godot resource references

  Current source references missing paths:

  - res://assets/player/player.gd
  - res://assets/world/mutated_tree.tscn
  - res://assets/models/Mannequin_F_Mannequin.res

  The female mesh actually exists under assets/models/characters/.

  There are also two scripts declaring class_name GameProtocol:

  - client/core/network/protocol.gd:1
  - client/core/network/game_protocol.gd:1

  ### Client movement represents a different authority model

  The old client sends destination coordinates through encode_move, whereas the current documented/server request
  expresses directional intent. This conflicts with the server-authoritative movement rule in docs/protocol.md:294,
  even though the stale code is presently unwired.

  ## 5. Missing dependencies between systems

  The principal broken chain is:

  LoginScreen
     -X-> AuthClient
     -X-> account HTTP server
     -X-> issued world ticket
     -X-> compatible Godot world protocol
     ---> world ticket validator
     -X-> character loader
     -X-> ECS player entity
     -X-> initial-region query
     -X-> bootstrap/entity packets
     -X-> Godot terrain/entity presentation

  Concrete missing connections:

  - The login screen does not call AuthClient.
  - The account server has no executable or HTTP handler.
  - No account operation returns the ticket understood by the world validator.
  - The Godot world client cannot speak the Go wire protocol.
  - World authentication does not invoke character loading.
  - Session identity is not associated with a runtime EntityID.
  - No ECS receives player, NPC, object, or ground-item runtime state.
  - Region definitions are not transformed into authoritative runtime entities.
  - No visibility or initial-region service exists.
  - No server bootstrap or entity-spawn codecs exist.
  - Godot has no region identity from which to load static terrain.
  - Godot’s terrain code has no content bundle and expects a different region structure.
  - Runtime spawn data has no agreed numeric definition-ID contract.
  - Existing client assets are not reachable through a definition registry.
  - Player/object registries are not mounted in the game scene.
  - The camera is never constructed or attached to the local player.
  - There is no integration test covering account → world → bootstrap → Godot presentation.

  ## 6. Ordered implementation tickets

  A simulation tick is intentionally not a prerequisite for this stationary slice. The smallest working slice can
  create initial ECS state and a player synchronously in the world application layer. Introduce the tick pipeline
  when inputs or autonomous systems must be consumed.

  ### 1. Repair the minimum canonical content contract

  - Prerequisites: none.
  - Acceptance:
      - Manifest and slice content parse successfully.
      - Minimal schemas cover manifest, items, NPCs, objects, and regions.
      - Region identity/coordinates are explicit.
      - NPC/object placements reference numeric stable IDs.
      - Every referenced object/NPC definition exists.
      - Male/female NPCs follow one schema.

  - Likely files: content_data/manifest.json, schema/, map/, mobs/, new minimal object/player-presentation
    definitions if required.

  - Out of scope: editors, hot reload, full content catalogue.

  ### 2. Implement server content loading and validation

  - Prerequisites: ticket 1.
  - Acceptance:
      - GAME_CONTENT_ROOT is used.
      - Registries load items, NPCs, objects, and regions.
      - Duplicate IDs, missing references, malformed coordinates, and invalid schema data fail startup.
      - Unit tests cover valid and invalid fixtures.

  - Likely packages: internal/content/loader, internal/content/definitions, cmd/server.
  - Out of scope: ECS construction, client serving, content reload.

  ### 3. Make the development account server runnable

  - Prerequisites: none.
  - Acceptance:
      - cmd/account builds and listens on configured HTTP address.
      - One environment-configured development credential maps to AccountID and default CharacterID.
      - /v1/login returns generic authentication failures.
      - Successful response contains the opaque configured world ticket, identity IDs, and configured world address.
      - Credentials/ticket are never logged.

  - Likely files: cmd/account, config, internal/account/auth/login.
  - Out of scope: OAuth, registration, databases, multiple characters/worlds.

  ### 4. Expose the client-safe content subset

  - Prerequisites: tickets 1–3.
  - Acceptance:
      - Account/backend process serves the validated manifest, required definitions, and region data needed by
        Godot.

      - Server-only fields are omitted where applicable.
      - Responses carry a stable content version/hash.

  - Likely packages: internal/content, internal/app/accountserver.
  - Out of scope: CDN, patching, archive formats, caching infrastructure.

  ### 5. Establish a runnable Godot boot scene

  - Prerequisites: none.
  - Acceptance:
      - project.godot has one main scene.
      - Startup transitions to the login screen.
      - Auth/network clients are instantiated exactly once.
      - A smoke test or headless boot check catches script/resource errors.

  - Likely files: project.godot, scenes/game/game_app.gd, startup/login scenes.
  - Out of scope: polished UI, registration, world selection.

  ### 6. Wire the login screen to the backend

  - Prerequisites: tickets 3 and 5.
  - Acceptance:
      - Submitted credentials call AuthClient.
      - Failure restores the form and shows a generic error.
      - Success retains account/character IDs and the opaque ticket separately.
      - Ticket length is not hardcoded to 43.

  - Likely files: login_screen.gd, auth_client.gd, game_app.gd.
  - Out of scope: token refresh, remembered credentials, OAuth.

  ### 7. Replace the obsolete Godot framing/login protocol

  - Prerequisites: ticket 5.
  - Acceptance:
      - Godot uses the Go 6-byte big-endian frame.
      - Opcode values 1–8 match Go.
      - ClientHello → ServerHello → LoginRequest → LoginAccepted works.
      - Fragmented/multiple frames are handled.
      - Old CONNECT, CONTENT_HELLO, and duplicate GameProtocol definitions are removed or retired.
      - Shared golden vectors test Go and Godot against the same bytes.

  - Likely files: client/core/network, network_client.gd, content_data/protocol/golden.json.
  - Out of scope: gameplay movement, compression, encryption.

  ### 8. Connect backend login success to the world client

  - Prerequisites: tickets 6–7.
  - Acceptance:
      - The configured world address and opaque ticket flow from backend response to GameNetworkClient.
      - Protocol mismatch and login rejection return safely to login UI.
      - No world connection begins before backend login succeeds.

  - Likely files: game_app.gd, auth_client.gd, network_client.gd.
  - Out of scope: selectable world list, reconnect, character selection.

  ### 9. Implement the minimal Godot content loader

  - Prerequisites: tickets 1 and 4–5.
  - Acceptance:
      - Client loads the manifest, initial region, and only required presentation definitions.
      - Region defaults/overrides are normalized into the representation used by terrain code.
      - Version/hash and malformed-content failures are visible before entering the world.

  - Likely files: client/core/content/*.
  - Out of scope: modding, cache invalidation, binary packs.

  ### 10. Add the numeric client asset registry

  - Prerequisites: tickets 1 and 9.
  - Acceptance:
      - Stable numeric IDs map to existing player, NPC, item, and object scenes/assets.
      - Unknown IDs use one explicit fallback or reject cleanly.
      - No canonical content file contains Godot resource paths.

  - Likely files: client/core/content/definition_registry.gd, a small Godot-only registry resource/script, scene
    paths.

  - Out of scope: skins, equipment composition, animation variants.

  ### 11. Add a minimal ECS runtime store

  - Prerequisites: ticket 2.
  - Acceptance:
      - Runtime entities have unique nonzero EntityID.
      - Minimal components cover position, entity kind, stable definition/appearance ID, and player ownership where
        applicable.

      - Creation/query/destruction tests exist.

  - Likely package: internal/engine/ecs plus minimal world component types.
  - Out of scope: worker pools, movement, combat, inventories, persistence.

  ### 12. Construct initial runtime world entities from regions

  - Prerequisites: tickets 2 and 11.
  - Acceptance:
      - World boot converts configured NPC/object/ground-item placements into ECS entities.
      - Static terrain remains content data, not ECS terrain entities.
      - Runtime existence is authoritative after construction.
      - Empty ground-item lists require no speculative ground-item system.

  - Likely packages: internal/world, internal/content, internal/engine/ecs, cmd/server.
  - Out of scope: respawning, NPC AI, interaction logic, spatial optimization.

  ### 13. Add the development character-loading seam

  - Prerequisites: tickets 3 and 11.
  - Acceptance:
      - A small CharacterLoader-style boundary resolves authenticated CharacterID to the configured development
        character.

      - Result includes spawn tile and stable presentation/appearance ID.
      - Invalid characters fail without entering Game.
      - No database is introduced.

  - Likely packages: internal/character, internal/world, config.
  - Out of scope: saves, inventories, multiple characters, appearance customization.

  ### 14. Complete the world login lifecycle

  - Prerequisites: tickets 11 and 13.
  - Acceptance:
      - Ticket validation is followed by character load and ECS player creation.
      - Session stores the resulting runtime EntityID.
      - Game/LoginAccepted occurs only after the player is ready.
      - Disconnect removes or marks the runtime player according to one tested policy.

  - Likely files: internal/world/login, new small world application service, session.
  - Out of scope: reconnect grace periods, duplicate-login policy, persistence.

  ### 15. Define the initial bootstrap wire contract

  - Prerequisites: tickets 1 and 11.
  - Acceptance:
      - Exact codecs carry region identity and authoritative entity snapshots.
      - Each entity has runtime EntityID, kind, numeric definition/appearance ID, and position.
      - Local player identity is unambiguous.
      - Protocol round trips and opcode uniqueness are tested.

  - Likely packages: neutral opcode contract plus feature-owned world/bootstrap codecs.
  - Out of scope: deltas, movement, visibility updates, inventories.

  ### 16. Generate and send the initial bootstrap

  - Prerequisites: tickets 12, 14, and 15.
  - Acceptance:
      - Player spawn determines the initial region.
      - Initial snapshot includes the local player and relevant runtime NPC/object/ground-item entities.
      - Bootstrap uses the normal bounded outbound queue.
      - Static terrain is not serialized as dynamic entities.

  - Likely packages: internal/world, world application service, cmd/server.
  - Out of scope: ongoing visibility, region crossing, spatial index optimization.

  ### 17. Render the authoritative initial region in Godot

  - Prerequisites: tickets 9 and 15–16.
  - Acceptance:
      - Client receives region identity from bootstrap.
      - It loads matching static terrain through the content loader.
      - Actual default/override height and terrain data produce the region mesh.
      - No static NPC/object/ground-item placement is independently spawned by the client.

  - Likely files: world_stream.gd, region_mesh_builder.gd, game scene/coordinator.
  - Out of scope: streaming adjacent regions, collision gameplay, LOD.

  ### 18. Present authoritative runtime entities

  - Prerequisites: tickets 10 and 15–17.
  - Acceptance:
      - Local player, configured NPCs, and configured objects are instantiated from server snapshots.
      - Ground items are instantiated only if present in the server snapshot.
      - Numeric IDs resolve through the asset registry.
      - Broken scene paths and stale string-ID branches are removed.

  - Likely files: player/object registries, player scenes, game scene.
  - Out of scope: animations, health, interactions, movement interpolation.

  ### 19. Attach the stationary-player camera

  - Prerequisites: ticket 18.
  - Acceptance:
      - PlayerCamera is instantiated and configured from local_player_ready.
      - Orbit/pan/zoom works.
      - No movement input is wired into the milestone scene.

  - Likely files: game.tscn, game_app.gd, player_camera.gd.
  - Out of scope: click-to-move, camera collision, cinematic controls.

  ### 20. Add one vertical-slice integration check

  - Prerequisites: tickets 1–19.
  - Acceptance:
      - A documented command starts account server, world server, and headless Godot client.
      - Development credentials produce a world ticket.
      - World creates exactly one player and returns bootstrap.
      - Client loads the expected region and presents the player plus configured runtime entities.
      - Both servers cleanly remove the connection on shutdown.

  - Likely files: integration tests and a minimal development startup script.
  - Out of scope: deployment tooling, containers, load testing, production authentication.