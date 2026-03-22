#!/usr/bin/env python3
"""
Hook: Block architect agents from editing production domain files.

Triggered by: PreToolUse (Edit|Write)

Purpose: Architect skill should produce decisions and implementation issues,
not modify domain files directly. This hook enforces that boundary by denying
edits to production file patterns when the active skill is "architect".

== DOMAIN DETECTION ==

This is a global hook with a config guard. It reads `.harness/config.yaml`
from the project root to determine whether domain-specific restrictions apply.

- If `domain: openscad` -> protect parts/*.scad and lib/*.scad
- If config missing or domain differs -> pass through (no enforcement)

This allows the hook to live globally in ~/.orch/hooks/ while only activating
in projects that declare their domain.

== DETECTION ==

Looks up the current agent's SPAWN_CONTEXT.md via ORCH_BEADS_ID to determine
the active skill from the "SKILL GUIDANCE" header.

== PRECONDITIONS ==

- CLAUDE_CONTEXT=worker (skips interactive sessions)
- ORCH_BEADS_ID is set (skips non-orchestrated sessions)

== CONFIGURATION ==

Escape hatch: SKIP_ARCHITECT_GATE=1 bypasses the check.
"""
import glob
import json
import os
import re
import subprocess
import sys

# Domain-specific production file patterns
DOMAIN_PATTERNS = {
    "openscad": [
        re.compile(r'parts/.*\.scad$'),
        re.compile(r'lib/.*\.scad$'),
    ],
}

# Skill name to block
BLOCKED_SKILL = "architect"


def get_project_root() -> str | None:
    """Find the project root via git rev-parse or fall back to cwd."""
    try:
        result = subprocess.run(
            ["git", "rev-parse", "--show-toplevel"],
            capture_output=True, text=True, timeout=5,
        )
        if result.returncode == 0:
            return result.stdout.strip()
    except (subprocess.TimeoutExpired, FileNotFoundError):
        pass
    return os.getcwd()


def detect_domain(project_dir: str) -> str | None:
    """Read .harness/config.yaml to determine project domain.

    Returns domain string (e.g., 'openscad') or None if not configured.
    """
    config_path = os.path.join(project_dir, ".harness", "config.yaml")
    try:
        with open(config_path) as f:
            for line in f:
                match = re.match(r'^domain:\s*(.+)$', line.strip())
                if match:
                    return match.group(1).strip().strip('"\'')
    except (OSError, IOError):
        pass
    return None


def find_skill_for_beads_id(beads_id: str, project_dir: str) -> str | None:
    """Find the skill name for the current agent by looking up SPAWN_CONTEXT.md."""
    workspace_base = os.path.join(project_dir, ".orch", "workspace")
    pattern = os.path.join(workspace_base, "*/SPAWN_CONTEXT.md")

    for ctx_path in glob.glob(pattern):
        try:
            with open(ctx_path) as f:
                content = f.read(8000)
        except OSError:
            continue

        if beads_id not in content:
            continue

        match = re.search(r'## SKILL GUIDANCE \(([^)]+)\)', content)
        if match:
            return match.group(1)

    return None


def is_production_file(file_path: str, project_dir: str, patterns: list) -> str | None:
    """Check if a file path matches a production pattern."""
    if not file_path:
        return None

    normalized = os.path.expanduser(file_path)
    try:
        rel_path = os.path.relpath(normalized, project_dir)
    except ValueError:
        rel_path = normalized

    for pattern in patterns:
        if pattern.search(rel_path):
            return pattern.pattern

    return None


def main():
    """Main entry point for PreToolUse hook."""
    context = os.environ.get("CLAUDE_CONTEXT", "")
    if context != "worker":
        sys.exit(0)

    if os.environ.get("SKIP_ARCHITECT_GATE", "") == "1":
        sys.exit(0)

    beads_id = os.environ.get("ORCH_BEADS_ID", "")
    if not beads_id:
        sys.exit(0)

    # Detect project domain from .harness/config.yaml
    project_dir = get_project_root()
    domain = detect_domain(project_dir)
    if not domain or domain not in DOMAIN_PATTERNS:
        sys.exit(0)

    try:
        input_data = json.load(sys.stdin)
    except json.JSONDecodeError:
        sys.exit(0)

    tool_name = input_data.get("tool_name", "")
    if tool_name not in ("Edit", "Write"):
        sys.exit(0)

    file_path = input_data.get("tool_input", {}).get("file_path", "")

    patterns = DOMAIN_PATTERNS[domain]
    matched_pattern = is_production_file(file_path, project_dir, patterns)
    if not matched_pattern:
        sys.exit(0)

    skill = find_skill_for_beads_id(beads_id, project_dir)
    if skill != BLOCKED_SKILL:
        sys.exit(0)

    output = {
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "deny",
            "permissionDecisionReason": (
                f"ARCHITECT PRODUCTION FILE GATE ({domain}): Architect agents cannot "
                "edit production files.\n\n"
                f"Blocked file: {file_path}\n"
                f"Matched pattern: {matched_pattern}\n"
                f"Active skill: {skill}\n"
                f"Domain: {domain}\n\n"
                "Architect skill should produce:\n"
                "  - Decisions (.kb/decisions/)\n"
                "  - Implementation issues (bd create)\n"
                "  - Specs (specs/)\n\n"
                f"NOT direct edits to domain production files.\n\n"
                "If this is blocking legitimate work, the orchestrator should\n"
                "spawn a feature-impl worker to make the changes."
            ),
        }
    }
    print(json.dumps(output))
    sys.exit(0)


if __name__ == "__main__":
    main()
