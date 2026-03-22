#!/usr/bin/env python3
"""Test strip_quoted_strings fix for gate-git-add-all.py."""
import sys
sys.path.insert(0, '.claude/hooks')

from importlib import util
spec = util.spec_from_file_location('gate', '.claude/hooks/gate-git-add-all.py')
mod = util.module_from_spec(spec)
spec.loader.exec_module(mod)

passed = 0
failed = 0

def check(desc, actual, expected):
    global passed, failed
    if actual == expected:
        passed += 1
        print(f"  PASS: {desc}")
    else:
        failed += 1
        print(f"  FAIL: {desc} (got {actual}, expected {expected})")

# Should BLOCK: actual blanket commands
check("block: git add -A", mod.is_blanket_git_add("git add -A"), True)
check("block: git add .", mod.is_blanket_git_add("git add ."), True)
check("block: git add --all", mod.is_blanket_git_add("git add --all"), True)
check("block: after quotes", mod.is_blanket_git_add('echo "hello" && git add -A'), True)

# Should ALLOW: patterns inside double quotes (false-positive fix)
check("allow: double-quoted -A", mod.is_blanket_git_add('bd create "do not use git add -A"'), False)
check("allow: double-quoted .", mod.is_blanket_git_add('bd comments add x "git add . is blocked"'), False)
check("allow: double-quoted --all", mod.is_blanket_git_add('echo "git add --all is bad"'), False)

# Should ALLOW: patterns inside single quotes
check("allow: single-quoted -A", mod.is_blanket_git_add("bd create 'do not use git add -A'"), False)

# Should ALLOW: explicit file staging
check("allow: explicit files", mod.is_blanket_git_add("git add src/file.go"), False)

# Heredoc test
heredoc_cmd = 'git commit -m "$(cat <<\'EOF\'\nDo not use git add -A or git add .\nEOF\n)"'
check("allow: heredoc content", mod.is_blanket_git_add(heredoc_cmd), False)

print(f"\nResults: {passed} passed, {failed} failed")
sys.exit(1 if failed else 0)
