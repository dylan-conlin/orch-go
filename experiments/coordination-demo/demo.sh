#!/usr/bin/env bash
# Coordination Demo — Reproduce the finding in three commands
#
# Usage:
#   ./demo.sh setup          # Verify prerequisites
#   ./demo.sh run             # Run 1 trial per condition (~10 min)
#   ./demo.sh results         # Visualize results
#
#   ./demo.sh run --full      # Run full experiment (10 trials × 5 conditions, ~2 hours)
#   ./demo.sh run --condition placement --trials 3   # Custom run

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

case "${1:-help}" in
    setup)
        echo ""
        echo "=== Coordination Demo: Setup ==="
        echo ""

        errors=0

        # Check Go
        if command -v go &>/dev/null; then
            echo "  [ok] Go $(go version | awk '{print $3}')"
        else
            echo "  [!!] Go not found. Install from https://go.dev"
            errors=$((errors + 1))
        fi

        # Check Claude CLI
        if command -v claude &>/dev/null; then
            echo "  [ok] Claude CLI found"
        else
            echo "  [!!] Claude CLI not found. Install: npm install -g @anthropic-ai/claude-code"
            errors=$((errors + 1))
        fi

        # Check jq
        if command -v jq &>/dev/null; then
            echo "  [ok] jq found"
        else
            echo "  [!!] jq not found. Install: brew install jq"
            errors=$((errors + 1))
        fi

        # Check git
        if command -v git &>/dev/null; then
            echo "  [ok] git found"
        else
            echo "  [!!] git not found"
            errors=$((errors + 1))
        fi

        # Check display package builds (the experiment target)
        echo ""
        echo "  Checking pkg/display builds..."
        if (cd "$PROJECT_DIR" && go build ./pkg/display/ 2>/dev/null); then
            echo "  [ok] pkg/display builds successfully"
        else
            echo "  [!!] pkg/display build failed — agents need a clean baseline"
            errors=$((errors + 1))
        fi

        # Check display package tests
        if (cd "$PROJECT_DIR" && go test ./pkg/display/ -count=1 2>/dev/null | grep -q "^ok"); then
            echo "  [ok] pkg/display tests pass"
        else
            echo "  [!!] pkg/display tests failing — agents need a clean baseline"
            errors=$((errors + 1))
        fi

        echo ""
        if [ "$errors" -eq 0 ]; then
            echo "  Ready to run! Next: ./demo.sh run"
        else
            echo "  $errors issue(s) to fix before running."
        fi
        echo ""
        ;;

    run)
        shift
        TRIALS=1
        FULL=false
        EXTRA_ARGS=()

        while [[ $# -gt 0 ]]; do
            case $1 in
                --full) FULL=true; TRIALS=10; shift ;;
                --trials) TRIALS="$2"; shift 2 ;;
                --condition|--task|--model)
                    EXTRA_ARGS+=("$1" "$2"); shift 2 ;;
                *) echo "Unknown arg: $1"; exit 1 ;;
            esac
        done

        echo ""
        echo "=== Coordination Demo: Run ==="
        echo ""

        if [ "$FULL" = true ]; then
            echo "  Running FULL experiment: 5 conditions × 2 tasks × $TRIALS trials"
            echo "  Estimated time: ~2 hours"
        else
            echo "  Running QUICK demo: 5 conditions × 2 tasks × $TRIALS trial(s)"
            echo "  Estimated time: ~10 minutes"
        fi
        echo ""
        echo "  Each trial spawns 2 Claude agents in parallel on isolated git worktrees."
        echo "  Agents implement different features in the same files, then we try to merge."
        echo ""

        # Run the experiment
        cd "$SCRIPT_DIR/redesign"
        bash run.sh --trials "$TRIALS" "${EXTRA_ARGS[@]}"

        # Find the latest results
        LATEST=$(ls -td "$SCRIPT_DIR/redesign/results"/*/ 2>/dev/null | head -1)
        if [ -n "$LATEST" ]; then
            echo ""
            echo "=== Results ready ==="
            echo "  View: ./demo.sh results"
            echo "  Raw:  $LATEST"
        fi
        ;;

    results)
        # Find the latest results directory
        LATEST=$(ls -td "$SCRIPT_DIR/redesign/results"/*/ 2>/dev/null | head -1)

        if [ -n "$LATEST" ] && [ -d "$LATEST" ]; then
            echo ""
            echo "=== Your Results ==="
            bash "$SCRIPT_DIR/visualize.sh" "$LATEST"
            echo ""
            echo -e "\033[2m  Raw results: $LATEST\033[0m"
        else
            echo ""
            echo "  No experiment results found. Showing reference data."
        fi

        echo ""
        echo "=== Reference Results (100 trials) ==="
        bash "$SCRIPT_DIR/visualize.sh"
        ;;

    help|--help|-h|*)
        echo ""
        echo "Coordination Demo — Multi-agent coordination experiment"
        echo ""
        echo "We tested 100 AI agent pairs. Perfect communication. Zero coordination."
        echo "Then one structural change: 100% success."
        echo ""
        echo "Usage:"
        echo "  ./demo.sh setup              Verify prerequisites (Go, Claude CLI, etc.)"
        echo "  ./demo.sh run                Run quick demo (1 trial per condition, ~10 min)"
        echo "  ./demo.sh run --full         Run full experiment (10 trials, ~2 hours)"
        echo "  ./demo.sh results            Visualize results"
        echo ""
        echo "  ./demo.sh run --condition placement --trials 3   Custom run"
        echo ""
        echo "What this tests:"
        echo "  Two AI agents implement different features in the same Go file."
        echo "  Five conditions test whether coordination can be achieved through:"
        echo "    1. no-coord       — Agents work independently"
        echo "    2. context-share  — Each agent sees the other's task"
        echo "    3. messaging      — Agents exchange plans before coding"
        echo "    4. gate           — Agents must verify no conflicts before committing"
        echo "    5. placement      — Orchestrator assigns non-overlapping insertion points"
        echo ""
        echo "  Only condition 5 works. The rest produce 0% merge success."
        echo ""
        ;;
esac
