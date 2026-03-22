#!/usr/bin/env python3
"""
Gate: Block openscad STL exports unless --backend cgal is specified.

== WHY THIS GATE EXISTS ==

Manifold backend silently repairs non-manifold geometry, meaning an STL exported
with Manifold may look valid but contain unprintable geometry. CGAL catches these
issues and fails loudly. Exporting STL without CGAL risks producing parts that
waste print time and filament.

Non-STL outputs (PNG previews, etc.) are allowed with any backend since they
don't produce printable geometry.

== HOW IT WORKS ==

This is a Claude Code PreToolUse hook. When an agent tries to run a Bash command
containing an openscad invocation that outputs an STL file, the script checks
for --backend cgal. If missing, the command is blocked.

Global hook: fires on openscad command presence. Harmless in non-OpenSCAD
projects since the command pattern simply never matches.

== CONFIGURATION ==

- To disable temporarily: set SKIP_OPENSCAD_CGAL_GATE=1 in your environment
- To remove permanently: delete this file and remove its entry from settings.json

== HOOK PROTOCOL ==

Input (stdin): JSON with tool_name and tool_input.command
Output (stdout): JSON with hookSpecificOutput.permissionDecision = "deny" to block
Exit 0 always (hooks should not crash Claude Code)
"""
import json
import os
import re
import sys


def is_stl_export_without_cgal(command: str) -> bool:
    """Detect openscad commands that export STL without --backend cgal."""
    if not re.search(r'\bopenscad\b', command):
        return False

    if not re.search(r'-o\s+["\']?[^\s"\']*\.stl\b', command, re.IGNORECASE):
        return False

    if re.search(r'--backend\s+cgal\b', command):
        return False

    return True


def main():
    if os.environ.get("SKIP_OPENSCAD_CGAL_GATE", "") == "1":
        sys.exit(0)

    try:
        input_data = json.load(sys.stdin)
    except json.JSONDecodeError:
        sys.exit(0)

    if input_data.get("tool_name") != "Bash":
        sys.exit(0)

    command = input_data.get("tool_input", {}).get("command", "")
    if not is_stl_export_without_cgal(command):
        sys.exit(0)

    output = {
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "deny",
            "permissionDecisionReason": (
                "BLOCKED: openscad STL export requires --backend cgal\n\n"
                "Manifold backend silently repairs non-manifold geometry, so an STL\n"
                "exported with Manifold may appear valid but be unprintable.\n\n"
                "Use --backend cgal for STL exports:\n"
                "  openscad -o exports/part.stl --backend cgal parts/part.scad\n\n"
                "Manifold is fine for non-STL outputs (PNG previews, etc.):\n"
                "  openscad -o exports/preview.png --backend manifold parts/part.scad\n\n"
                "To bypass: set SKIP_OPENSCAD_CGAL_GATE=1"
            ),
        }
    }
    print(json.dumps(output))
    sys.exit(0)


if __name__ == "__main__":
    main()
