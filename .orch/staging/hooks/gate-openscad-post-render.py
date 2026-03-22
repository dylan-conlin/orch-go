#!/usr/bin/env python3
"""
Gate: Auto-run geometry-check.sh after openscad renders (advisory).

== WHY THIS GATE EXISTS ==

After any successful `openscad -o` command, the geometry gate should run
automatically so the agent gets immediate feedback on manifold status,
polygon budget, and build plate fit. This is advisory — failures produce
warnings but do not block, since the agent may be iterating with Manifold
before a final CGAL validation pass.

== HOW IT WORKS ==

This is a Claude Code PostToolUse hook on Bash. After a Bash command
completes successfully:
1. Check if the command contained `openscad -o`
2. Extract the source .scad file path from the command
3. Run `gates/geometry-check.sh` on it (found relative to project root)
4. Report results as a message (warn on failure, don't block)

Global hook: fires on openscad command presence. Harmless in non-OpenSCAD
projects since the command pattern simply never matches.

== CONFIGURATION ==

- To disable: set SKIP_OPENSCAD_POST_RENDER_GATE=1 in your environment

== HOOK PROTOCOL ==

Input (stdin): JSON with tool_name, tool_input.command, tool_result
Output (stdout): JSON with hookSpecificOutput.message for advisory feedback
Exit 0 always (hooks should not crash Claude Code)
"""
import json
import os
import re
import subprocess
import sys


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


def extract_scad_path(command: str) -> str | None:
    """Extract the source .scad file path from an openscad command."""
    matches = re.findall(r"""(?:["']([^"']*\.scad)["']|(\S*\.scad))""", command)
    if not matches:
        return None

    paths = [q or u for q, u in matches]

    source = None
    for path in paths:
        escaped = re.escape(path)
        if re.search(rf'-o\s+["\']?{escaped}["\']?', command):
            continue
        source = path

    return source


def main():
    if os.environ.get("SKIP_OPENSCAD_POST_RENDER_GATE", "") == "1":
        sys.exit(0)

    try:
        input_data = json.load(sys.stdin)
    except json.JSONDecodeError:
        sys.exit(0)

    if input_data.get("tool_name") != "Bash":
        sys.exit(0)

    command = input_data.get("tool_input", {}).get("command", "")

    if not re.search(r'\bopenscad\b', command):
        sys.exit(0)
    if not re.search(r'\s-o\s', command):
        sys.exit(0)

    tool_result = input_data.get("tool_result", {})
    if tool_result.get("exitCode", 1) != 0:
        sys.exit(0)

    scad_path = extract_scad_path(command)
    if not scad_path:
        sys.exit(0)

    # Find geometry-check.sh relative to project root
    project_root = get_project_root()
    if not project_root:
        sys.exit(0)

    gate_script = os.path.join(project_root, "gates", "geometry-check.sh")

    if not os.path.isfile(gate_script):
        sys.exit(0)

    try:
        result = subprocess.run(
            ["bash", gate_script, scad_path],
            capture_output=True,
            text=True,
            timeout=120,
        )
        if result.returncode == 0:
            message = (
                f"Geometry gate PASSED for {scad_path}\n"
                f"{result.stdout.strip()}"
            )
        else:
            message = (
                f"WARNING: Geometry gate failed for {scad_path} "
                f"(advisory — not blocking)\n"
                f"{result.stdout.strip()}"
            )
            if result.stderr.strip():
                message += f"\n{result.stderr.strip()}"
    except subprocess.TimeoutExpired:
        message = (
            f"WARNING: Geometry gate timed out for {scad_path} "
            f"(advisory — not blocking)"
        )
    except Exception as e:
        message = (
            f"WARNING: Geometry gate error for {scad_path}: {e} "
            f"(advisory — not blocking)"
        )

    output = {
        "hookSpecificOutput": {
            "hookEventName": "PostToolUse",
            "message": message,
        }
    }
    print(json.dumps(output))
    sys.exit(0)


if __name__ == "__main__":
    main()
