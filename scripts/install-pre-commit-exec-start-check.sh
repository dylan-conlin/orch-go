#!/usr/bin/env bash
# Ensure .git/hooks/pre-commit executes the exec.Start lifecycle guard.
set -euo pipefail

hook_path=".git/hooks/pre-commit"
guard_line='scripts/pre-commit-exec-start-cleanup.sh'

if [[ ! -f "$hook_path" ]]; then
	echo "ERROR: $hook_path not found. Run 'bd hooks install' first." >&2
	exit 1
fi

if grep -Fq "$guard_line" "$hook_path"; then
	echo "Pre-commit guard already installed in $hook_path"
	exit 0
fi

tmp_file="$(mktemp)"
python3 - "$hook_path" "$tmp_file" <<'PY'
import sys
from pathlib import Path

hook_path = Path(sys.argv[1])
tmp_path = Path(sys.argv[2])
content = hook_path.read_text()

guard_block = '''# Project-specific lifecycle safety checks (best effort)
if [ -x "scripts/pre-commit-exec-start-cleanup.sh" ]; then
    scripts/pre-commit-exec-start-cleanup.sh "$@" || exit $?
fi

'''

needle = 'exec bd hooks run pre-commit "$@"'
if needle in content:
    content = content.replace(needle, guard_block + needle, 1)
else:
    if not content.endswith("\n"):
        content += "\n"
    content += "\n" + guard_block + needle + "\n"

tmp_path.write_text(content)
PY

mv "$tmp_file" "$hook_path"
chmod +x "$hook_path"
echo "Installed exec.Start lifecycle guard into $hook_path"
