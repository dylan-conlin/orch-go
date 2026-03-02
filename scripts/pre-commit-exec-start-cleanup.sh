#!/bin/sh
# Pre-commit gates for orch-go
#
# Called by .git/hooks/pre-commit (bd shim) before bd hooks run.
# Gates: compilation + architecture lint.
# Total time: ~2s (cached builds).

set -e

# Only run if Go files are staged
STAGED_GO=$(git diff --cached --name-only --diff-filter=ACM -- '*.go' 2>/dev/null || true)
if [ -z "$STAGED_GO" ]; then
    exit 0
fi

echo "pre-commit: running compilation gate..."
if ! go build ./cmd/orch/ 2>&1; then
    echo ""
    echo "BLOCKED: compilation failed. Fix build errors before committing."
    exit 1
fi

echo "pre-commit: running architecture lint..."
if ! go test -run TestArchitectureLint -count=1 ./cmd/orch/ 2>&1; then
    echo ""
    echo "BLOCKED: architecture lint failed. See test output above."
    echo "Reference: .kb/decisions/2026-02-18-two-lane-agent-discovery.md"
    exit 1
fi

echo "pre-commit: gates passed."
