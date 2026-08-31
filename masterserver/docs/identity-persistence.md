# Identity and Persistence

## Identity hierarchy

The system deliberately separates account, character, runtime entity, world, and connection identity.

```text
AccountID
  user/customer identity

CharacterID
  persistent playable identity

WorldID
  selectable server/world instance

EntityID
  temporary runtime ECS identity

ConnectionID
  temporary network connection identity
```

Example:

```text
Account 42
├── Character 1001 "foo"
├── Character 1002 "bar"
└── Character 1003 "foobar"

Character 1001 enters World 2
  ↓
runtime Entity 58291
```

Never persist an `EntityID` as the permanent identity of a character.

Never use a mutable character name as a foreign key.

## Account

`account/` owns:

- login identity;
- password/OAuth provider linkage;
- account-level status;
- account-level subscription/billing status where applicable.

OAuth providers should resolve to a stable internal `AccountID`.

The rest of the game should not need to know whether the user authenticated using Google, Apple, or password.

## Character

`character/` owns:

- `CharacterID`;
- `AccountID` ownership;
- display name;
- creation metadata;
- character session/online lock;
- save loading/storing.

A character remains meaningful while offline.

## Runtime Player

`game/player/` represents a loaded character participating in one world simulation.

A player component may contain the persistent character identity:

```go
type Player struct {
    CharacterID character.ID
}
```

Runtime gameplay state then lives in ECS components such as:

```text
Position
Movement
Inventory
Equipment
Health
Combat
Animation
Skills
```

## Membership/subscription placement

Store membership at the level where the business rule actually applies.

If membership grants benefits to every character on one account:

```text
Account -> subscription
```

If a product genuinely supports per-character membership:

```text
Character -> membership
```

Do not duplicate the same meaning in both account and character rows.

## Database vs `.sav`

Use the database for queryable/control-plane data.

Examples:

```text
characters
  id
  account_id
  name
  created_at
  last_login_at
  save_version
  current_save_key
  current_save_generation
  save_checksum
```

Potential additional denormalised fields may be stored when the application needs efficient queries without loading every save, such as:

- combat level;
- total level;
- last world;
- status flags.

Use `.sav` for heavier gameplay state.

Examples:

```text
position
skills
inventory
equipment
bank
quests
appearance
unlocks
game variables
```

## Save format

The save model is an explicit persistence DTO/schema, not an ECS dump.

Conceptually:

```go
type Save struct {
    Version   uint16
    Position  PositionSave
    Skills    SkillsSave
    Inventory InventorySave
    Equipment EquipmentSave
    // ...
}
```

This format may be binary for efficiency.

Keep the format versioned.

## Save migration

Old saves should be migrated through explicit version steps.

Example:

```text
v1
 ↓
v2
 ↓
v3
 ↓
current model
```

Avoid scattering legacy-version checks through live gameplay systems.

Migration belongs under `character/save`.

## Atomic save generations

Prefer generation-based saves over overwriting one file in place.

Example:

```text
characters/1001/928.sav
characters/1001/929.sav
```

Safe write flow:

```text
DB points at generation 928
  ↓
write 929.sav
  ↓
flush/verify/checksum
  ↓
transactionally update DB pointer -> 929
```

If the process fails during the new write, the previous generation remains usable.

Retention policy can delete old generations later.

## Save repository abstraction

Game code should not assume saves live on a local disk.

Use a repository boundary roughly equivalent to:

```go
type Repository interface {
    Load(ctx context.Context, characterID CharacterID) (Save, error)
    Store(ctx context.Context, characterID CharacterID, save Save) error
}
```

An initial implementation may use local files.

A later implementation may use shared/object storage when multiple world hosts need the same saves.

## Character session lock

Only one world should actively own a character at a time.

Before loading a character into a world, acquire an online/session lock.

Concept:

```text
character:1001 -> world:2
```

The exact storage may later be Redis, database locking, or another coordination mechanism.

This is a correctness boundary, not merely a social presence feature.

## World tickets

Credentials should be handled by the account/auth boundary.

The game world should receive a short-lived, scoped world-entry ticket rather than the user's Google token/password.

A world ticket should identify enough context to validate entry, for example:

```text
AccountID
CharacterID
WorldID
Expiry
Nonce
Signature/MAC
```

The target world validates:

- ticket integrity;
- expiry;
- intended `WorldID`;
- account owns character;
- world capacity;
- character not already active elsewhere.

## Social and leaderboard identity

If characters appear as distinct in-game identities, use `CharacterID` for:

- friends;
- ignore lists;
- clan membership;
- presence;
- leaderboards.

Use `AccountID` for customer/account-level concerns such as:

- authentication;
- account sanctions;
- billing/subscription;
- account-wide settings.
