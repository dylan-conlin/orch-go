#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WEB_DIR="$ROOT_DIR/web"
RUN_DIR="$ROOT_DIR/.orch/run"
PID_FILE="$RUN_DIR/dashboard-web.pid"

mkdir -p "$RUN_DIR"

kill_pid() {
	local pid="$1"

	if [[ ! "$pid" =~ ^[0-9]+$ ]]; then
		return 0
	fi

	if ! kill -0 "$pid" 2>/dev/null; then
		return 0
	fi

	kill "$pid" 2>/dev/null || true

	for _ in {1..20}; do
		if ! kill -0 "$pid" 2>/dev/null; then
			return 0
		fi
		sleep 0.1
	done

	kill -9 "$pid" 2>/dev/null || true
}

kill_stale_bun_in_web_dir() {
	local tracked_pid=""

	if [[ -f "$PID_FILE" ]]; then
		tracked_pid="$(<"$PID_FILE")"
		kill_pid "$tracked_pid"
		rm -f "$PID_FILE"
	fi

	local pid=""
	local cwd=""
	local line=""

	while IFS= read -r pid; do
		if [[ ! "$pid" =~ ^[0-9]+$ ]]; then
			continue
		fi

		cwd=""
		while IFS= read -r line; do
			if [[ "$line" == n* ]]; then
				cwd="${line#n}"
				break
			fi
		done < <(lsof -a -p "$pid" -d cwd -Fn 2>/dev/null || true)

		if [[ "$cwd" == "$WEB_DIR" ]]; then
			kill_pid "$pid"
		fi
	done < <(pgrep -f "bun" 2>/dev/null || true)
}

kill_stale_bun_in_web_dir

cd "$WEB_DIR"
bun run dev &
web_pid=$!
printf '%s\n' "$web_pid" >"$PID_FILE"

forward_signal() {
	kill "$web_pid" 2>/dev/null || true
}

trap forward_signal INT TERM HUP QUIT

set +e
wait "$web_pid"
status=$?
set -e

if [[ -f "$PID_FILE" ]]; then
	current_pid="$(<"$PID_FILE")"
	if [[ "$current_pid" == "$web_pid" ]]; then
		rm -f "$PID_FILE"
	fi
fi

exit "$status"
