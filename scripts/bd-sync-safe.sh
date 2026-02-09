#!/usr/bin/env bash

set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
local_db="$repo_root/.beads/beads.db"
sync_timeout_seconds="${BD_SYNC_SAFE_TIMEOUT_SECONDS:-60}"

run_with_timeout() {
	python3 - "$sync_timeout_seconds" "$@" <<'PY'
import subprocess
import sys

try:
	timeout = float(sys.argv[1])
except ValueError:
	sys.stderr.write(f"bd-sync-safe: invalid timeout: {sys.argv[1]}\n")
	sys.exit(2)

cmd = sys.argv[2:]
proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)

try:
	output, _ = proc.communicate(timeout=timeout)
except subprocess.TimeoutExpired:
	proc.kill()
	output, _ = proc.communicate()
	if output:
		sys.stdout.write(output)
	sys.stderr.write(f"bd-sync-safe: command timed out after {int(timeout)}s: {' '.join(cmd)}\n")
	sys.exit(124)

if output:
	sys.stdout.write(output)
sys.exit(proc.returncode)
PY
}

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

# Keep sync local-only by default. Orchestrator workflow handles git pull/push
# explicitly, and running bd sync pull in a dirty tree fails under active swarm work.
sync_args=(sync --no-daemon --sqlite --no-pull --no-push "$@")

output=""
if output="$(run_with_timeout bd "${sync_args[@]}" 2>&1)"; then
	printf '%s\n' "$output"
	exit 0
else
	status=$?
fi

printf '%s\n' "$output" >&2

if [ "$status" -eq 124 ] && [[ "$output" == *"JSONL content differs from last sync"* ]] && [[ "$output" == *"Importing JSONL first to prevent stale DB from overwriting changes..."* ]]; then
	echo "→ Timed out in JSONL hash-mismatch import path; retrying with explicit import-only..." >&2
	run_with_timeout bd sync --no-daemon --sqlite --import-only "$@"
	run_with_timeout bd "${sync_args[@]}"
	exit 0
fi

exit "$status"
