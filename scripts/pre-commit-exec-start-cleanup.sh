#!/usr/bin/env bash
# Reject new exec.Command(...).Start() calls without cleanup scaffolding.
set -euo pipefail

readonly BYPASS_VAR="ORCH_EXEC_START_CLEANUP_BYPASS"

if [[ -n "${ORCH_EXEC_START_CLEANUP_BYPASS:-}" ]]; then
	echo "WARNING: Skipping exec.Start cleanup check (${BYPASS_VAR} set)"
	mkdir -p .orch
	printf '{"time":"%s","action":"exec_start_cleanup_bypass","reason":"%s"}\n' "$(date -Iseconds)" "${ORCH_EXEC_START_CLEANUP_BYPASS}" >>.orch/mode-history.jsonl
	exit 0
fi

mapfile -t STAGED_GO_FILES < <(git diff --cached --name-only --diff-filter=ACMR -- '*.go')

if [[ ${#STAGED_GO_FILES[@]} -eq 0 ]]; then
	exit 0
fi

declare -a VIOLATIONS=()

for file in "${STAGED_GO_FILES[@]}"; do
	patch_text="$(git diff --cached -U0 -- "$file")"
	if ! grep -Eq '^\+[^+].*\.Start\(\)' <<<"$patch_text"; then
		continue
	fi

	staged_content="$(git show ":$file" 2>/dev/null || true)"
	if [[ -z "$staged_content" ]]; then
		continue
	fi

	declare -A EXEC_VARS=()
	while IFS= read -r line; do
		if [[ $line =~ ^[[:space:]]*([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*(:=|=)[[:space:]]*exec\.Command(Context)?\( ]]; then
			EXEC_VARS["${BASH_REMATCH[1]}"]=1
		fi
	done <<<"$staged_content"

	if [[ ${#EXEC_VARS[@]} -eq 0 ]]; then
		continue
	fi

	declare -A START_VARS=()
	while IFS= read -r line; do
		if [[ $line =~ ^\+[^+].*[^A-Za-z0-9_]([A-Za-z_][A-Za-z0-9_]*)\.Start\(\) ]]; then
			var_name="${BASH_REMATCH[1]}"
			if [[ -n "${EXEC_VARS[$var_name]:-}" ]]; then
				START_VARS["$var_name"]=1
			fi
		fi
	done <<<"$patch_text"

	if [[ ${#START_VARS[@]} -eq 0 ]]; then
		continue
	fi

	for var_name in "${!START_VARS[@]}"; do
		has_sysprocattr=false
		has_setpgid=false
		has_shutdown_registration=false

		if grep -Eq "\\b${var_name}\\.SysProcAttr[[:space:]]*=" <<<"$staged_content"; then
			has_sysprocattr=true
		fi
		if grep -Eq 'Setpgid:[[:space:]]*true' <<<"$staged_content"; then
			has_setpgid=true
		fi

		if grep -Eq '(WriteProcessID|[Rr]egister[A-Za-z0-9_]*Process|[Rr]egister[A-Za-z0-9_]*Shutdown|[Tt]rack[A-Za-z0-9_]*Process|[Aa]ddShutdownHook|[Oo]nShutdown)[[:space:]]*\(' <<<"$staged_content"; then
			has_shutdown_registration=true
		elif grep -Eq "defer[[:space:]]+${var_name}\\.Process\\.Kill[[:space:]]*\\(" <<<"$staged_content"; then
			has_shutdown_registration=true
		fi

		if [[ "$has_sysprocattr" != true || "$has_setpgid" != true || "$has_shutdown_registration" != true ]]; then
			missing_parts=()
			if [[ "$has_sysprocattr" != true || "$has_setpgid" != true ]]; then
				missing_parts+=("process-group management (SysProcAttr + Setpgid: true)")
			fi
			if [[ "$has_shutdown_registration" != true ]]; then
				missing_parts+=("shutdown registration")
			fi
			VIOLATIONS+=("$file: ${var_name}.Start() missing ${missing_parts[*]}")
		fi
	done
done

if [[ ${#VIOLATIONS[@]} -gt 0 ]]; then
	echo "BLOCKED: exec.Command(...).Start() requires lifecycle cleanup safeguards"
	echo
	for violation in "${VIOLATIONS[@]}"; do
		echo "  - $violation"
	done
	echo
	echo "Required for each new exec Start call:"
	echo "  1. Process group management: cmd.SysProcAttr + Setpgid: true"
	echo "  2. Shutdown registration: register cleanup/termination on parent shutdown"
	echo
	echo "Emergency bypass (must include reason):"
	echo "  ORCH_EXEC_START_CLEANUP_BYPASS=\"reason\" git commit ..."
	exit 1
fi

exit 0
