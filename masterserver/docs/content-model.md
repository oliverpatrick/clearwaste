# Content Model and Runtime IDs

## Goal

Content should be pleasant for humans to author and efficient for the server/client to consume.

Those are separate representations.

## Authoring representation

Readable formats such as JSON are appropriate for source content.

Examples:

```text
content/
├── items/
├── npcs/
├── objects/
├── maps/
└── quests/
```

Authoring data should favour:

- readability;
- validation;
- clear schemas;
- stable symbolic names where helpful to authors;
- source-control-friendly diffs.

## Runtime representation

Build/compile content into a compact representation when performance or load time justifies it.

Runtime systems should prefer numeric IDs.

Examples:

```go
type ItemID uint32
type NPCID uint32
type ObjectID uint32
```

The content registry maps authored identifiers to stable numeric IDs.

Do not repeatedly hash or compare long string names in simulation hot paths if a stable numeric ID is available.

## Stable IDs

IDs become part of data contracts once used in:

- saves;
- network protocol;
- compiled map data;
- drop tables;
- NPC definitions;
- quests.

Therefore:

- never renumber IDs casually;
- keep deleted IDs reserved when compatibility requires it;
- validate duplicate IDs during content build/load.

## Definitions vs runtime entities

A definition describes a type of thing.

Example:

```text
NPCID 17 -> Generic Man definition
```

A runtime entity is one instance:

```text
EntityID 58291
  NPC component { DefinitionID: 17 }
  Position { ... }
  Health { ... }
```

Never use `NPCID` as the runtime entity ID.

## Maps/world content

The logical world is tile-based and may be divided into authoring/runtime regions or chunks.

Chunking/regions are content/indexing conveniences.

They are not simulation ownership boundaries.

Map data may describe:

- terrain;
- collision;
- cardinal edge walls;
- objects;
- NPC spawns;
- ground items;
- triggers.

## Collision

Authoring may store edge information in a readable form.

Runtime should compile it into compact flags suitable for fast traversal checks.

For example:

```text
bit 0 -> North
bit 1 -> East
bit 2 -> South
bit 3 -> West
```

Validate neighbouring edge consistency during content build/load where possible.

## Content validation

Validation should fail early on:

- duplicate IDs;
- references to missing definitions;
- invalid enum values;
- invalid tile coordinates;
- inconsistent collision edges;
- impossible quantities/ranges;
- unsupported content format versions.

Prefer rejecting invalid content at startup/build time rather than discovering it during a live tick.

## Client/server sharing

The server remains authoritative for gameplay rules.

The client may consume compiled content needed for rendering, prediction, or UI.

Do not expose server-only secrets/rules merely because the same content pipeline exists for both sides.

Keep the format/versioning explicit if server and client consume the same compiled artifact.
