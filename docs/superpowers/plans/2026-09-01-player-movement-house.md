# Player Movement, Pathfinding, House, and Two-User Integration Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Let two development clients click a tile, pathfind locally, submit directional steps, and receive authoritative tick-based movement around a small house.

**Architecture:** The client uses the existing tile collision data for bounded A* and sends one-direction `MoveRequest` packets. The server queues steps per authenticated runtime entity, validates cardinal edges/blocked tiles on each tick, applies valid moves, and broadcasts compact authoritative position updates. The house is built from `wall.tscn` on the Godot side and described by matching collision edges in region content; development login supports two credentials.

**Tech Stack:** Go, Godot 4 GDScript, existing custom binary protocol and region JSON.

**Spec:** Approved movement model in conversation.

## Global Constraints

- Preserve the existing six-byte frame and opcode values 1–9.
- Keep server authority for position and collision.
- Keep movement tile-based, 8-directional, and diagonal corner-safe.
- Do not add persistence, combat, or generalized navigation infrastructure.

### Task 1: Authoritative server movement state

Add collision edge parsing, development character 2, per-entity step queues, deterministic tick application, and position-update snapshots.

### Task 2: Movement wire packets and world tick

Use one-byte directional MoveRequest packets, add a stable position-update opcode/codec, run the world tick, and broadcast accepted positions to connected sessions.

### Task 3: Godot pathfinding and authoritative presentation

Normalize collision data, implement bounded A* with diagonal corner checks, send step packets, and apply only server position updates to player visuals.

### Task 4: House fixture and two development accounts

Render a 9×5 wall.tscn house around the spawn area, add matching collision edges, and configure a second development account/character for concurrent login testing.

### Task 5: Integration verification

Add focused queue/collision/path tests, run Go and Godot checks, and exercise two-client login plus movement where sandbox networking permits.
