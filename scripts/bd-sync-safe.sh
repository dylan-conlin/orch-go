#!/usr/bin/env bash

set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
local_db="$repo_root/.beads/beads.db"

if [ ! -f "$local_db" ]; then
	echo "→ No local SQLite DB found at $local_db"
	echo "→ Attempting one-time bootstrap via 'bd init --sqlite'..."

	init_output=""
	if ! init_output="$(bd init --sqlite 2>&1)"; then
		case "$init_output" in
		*"already initialized"* | *"Found existing database:"*)
			printf '%s\n' "$init_output"
			echo "→ Reusing existing beads initialization and continuing."
			;;
		*)
			printf '%s\n' "$init_output" >&2
			exit 1
			;;
		esac
	else
		printf '%s\n' "$init_output"
	fi
fi

exec bd sync --no-daemon --sqlite "$@"
