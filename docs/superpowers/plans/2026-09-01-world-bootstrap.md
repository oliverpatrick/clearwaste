# World Bootstrap and Minimal ECS Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Complete the development login flow from ticket validation through a server-authoritative player/region bootstrap rendered by Godot.

**Architecture:** Add a small world runtime package with typed entities and a development character repository. The world login handler creates the player and sends one bootstrap packet containing local entity ID, spawn region, and visible entity snapshots; Godot decodes it and renders static terrain/entities without movement.

**Tech Stack:** Go, existing ECS-adjacent game packages, custom six-byte protocol, Godot 4 GDScript.

**Spec:** Approved bootstrap contract in conversation for WORLD-001..004, CHAR-001, PROTO-002, CLIENT-004..006, M0-INTEGRATION.

## Global Constraints

- Preserve existing login opcodes and six-byte frame format.
- Use stable numeric IDs; no Godot paths in server/content data.
- Keep movement disconnected.
- Do not add persistence or a general ECS framework for this slice.

### Task 1: Minimal server ECS and development character

Create typed runtime entity/component structures and a development character loader for CharacterID 1 with name, appearance ID, and spawn tile.

### Task 2: Region runtime population

Load region_0_0 content and create NPC/object/ground-item entities with positions, kind, definition IDs, and character IDs where applicable.

### Task 3: Bootstrap protocol and login lifecycle

Add one bootstrap opcode/codec, attach the created player EntityID to the authenticated session, and send the local ID, region coordinate, and entity snapshots after LoginAccepted.

### Task 4: Godot bootstrap and rendering

Decode bootstrap, load the referenced local region, render stationary player/NPC/object/item visuals through AssetRegistry, and keep the existing camera hidden until bootstrap succeeds.

### Task 5: Camera and integration verification

Add orbit/pan/zoom input without movement, then run Go tests, Godot headless tests, and an end-to-end development login/bootstrap check.
