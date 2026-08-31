# MMO Server Agent Documentation

This directory is a ready-to-copy documentation pack for the Go MMO server.

Start with [`AGENTS.md`](../AGENTS.md).

## Documents

- [`architecture.md`](architecture.md) — process/domain architecture and scaling model.
- [`folder-structure.md`](folder-structure.md) — target package tree and ownership.
- [`simulation.md`](simulation.md) — 600 ms tick, Decide/Apply/Rebuild, concurrency invariants.
- [`network-protocol.md`](network-protocol.md) — framing, queues, codecs, opcode rules.
- [`identity-persistence.md`](identity-persistence.md) — Account/Character/Entity/World identity and `.sav` design.
- [`content-model.md`](content-model.md) — authoring vs compiled runtime content and stable numeric IDs.
- [`testing-conventions.md`](testing-conventions.md) — tests, race checks, benchmarks, invariants.
- [`mvp-scope.md`](mvp-scope.md) — what belongs in the first playable slice and what does not.
- [`agent-task-checklist.md`](agent-task-checklist.md) — compact checklist for individual AI-agent tasks.
