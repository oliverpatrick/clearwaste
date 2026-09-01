#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

docker compose up --build -d world account
trap 'docker compose down' EXIT INT TERM

echo "Account server: http://127.0.0.1:8080"
echo "World server:   127.0.0.1:7777"
echo "Launching Godot client; press Ctrl-C to stop all services."

if [[ -n "${GODOT_BIN:-}" ]]; then
	godot_bin="$GODOT_BIN"
elif [[ -x "/Applications/Godot.app/Contents/MacOS/Godot" ]]; then
	godot_bin="/Applications/Godot.app/Contents/MacOS/Godot"
else
	godot_bin="godot"
fi

GAME_CONTENT_ROOT="$root_dir/content_data" "$godot_bin" --path "$root_dir/client"
