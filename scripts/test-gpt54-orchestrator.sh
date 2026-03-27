#!/usr/bin/env bash
# Test GPT-5.4 protocol compliance with full orchestrator skill
# Creates OpenCode sessions with GPT-5.4, injects orchestrator skill, measures stall rate
#
# Usage: ./scripts/test-gpt54-orchestrator.sh [num_tasks]
# Default: 3 tasks

set -euo pipefail

API="http://127.0.0.1:4096"
MODEL="openai/gpt-5.4"
PROJECT_DIR="/Users/dylanconlin/Documents/personal/orch-go"
SKILL_PATH="$HOME/.claude/skills/meta/orchestrator/SKILL.md"
NUM_TASKS="${1:-3}"
STALL_TIMEOUT=600  # 10 minutes per task
POLL_INTERVAL=5

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() { echo -e "[$(date +%H:%M:%S)] $*"; }
ok()  { echo -e "${GREEN}[PASS]${NC} $*"; }
fail(){ echo -e "${RED}[FAIL]${NC} $*"; }
warn(){ echo -e "${YELLOW}[WARN]${NC} $*"; }

# Verify prerequisites
if ! curl -s --max-time 2 "$API/session" >/dev/null 2>&1; then
    fail "OpenCode API not reachable at $API"
    exit 1
fi

if [ ! -f "$SKILL_PATH" ]; then
    fail "Orchestrator skill not found at $SKILL_PATH"
    exit 1
fi

SKILL_CONTENT=$(cat "$SKILL_PATH")
SKILL_TOKENS=$(echo "$SKILL_CONTENT" | wc -w | tr -d ' ')
log "Orchestrator skill: ~${SKILL_TOKENS} words (~$(( SKILL_TOKENS * 4 / 3 )) tokens estimated)"

# Define realistic orchestration tasks
TASKS=(
    "You are an orchestrator agent for orch-go. Review the following completion summary and provide Three-Layer Reconnection synthesis:\n\nAgent orch-go-test1 completed a systematic-debugging task: 'Fix stall tracker false negatives when agent produces output but never reports Phase: Complete'. The agent found that the stall tracker was using a 5-second threshold but some agents take 5.8-8.8 seconds between messages. Fixed by increasing threshold to 10s with adaptive backoff. Tests pass. 2 files changed.\n\nProvide:\n1. Frame - what was the original problem from Dylan's perspective\n2. Resolution - what changed and why it matters\n3. Placement - how this connects to the larger daemon reliability thread\n4. An open question for Dylan"

    "You are an orchestrator agent for orch-go. Triage the following issue and produce a well-enriched beads issue specification:\n\nDylan says: 'The dashboard is showing agents as active when they finished 20 minutes ago. The status seems stale.'\n\nProvide:\n1. Issue type classification (bug/feature/investigation)\n2. Skill selection with reasoning (use the decision tree)\n3. Label taxonomy (skill:, area:, effort:)\n4. A structured description including what's known, what's not known, and constraints\n5. The bd create command you would run"

    "You are an orchestrator agent for orch-go. Synthesize the following three investigation findings into a coherent understanding:\n\n1. Investigation A found that 69% of daemon routing falls to coarse type-based inference because issues lack skill: labels.\n2. Investigation B found that GPT-5.4 achieves 89% first-attempt completion on reasoning-heavy skills, up from 67% on GPT-5.2.\n3. Investigation C found that the daemon's InferModelFromSkill() only hard-pins Opus for reasoning skills and leaves feature-impl to defaults.\n\nProvide:\n1. What pattern connects these three findings?\n2. What does Dylan now understand differently?\n3. What thread does this form or extend?\n4. What's the next highest-value action?"
)

# Results tracking
declare -a SESSION_IDS
declare -a TASK_NAMES
declare -a RESULTS
declare -a DURATIONS
declare -a TOKEN_COUNTS

TASK_LABELS=("completion-synthesis" "issue-triage" "multi-finding-synthesis")

log "Starting GPT-5.4 orchestrator protocol compliance test"
log "Tasks: $NUM_TASKS | Model: $MODEL | Stall timeout: ${STALL_TIMEOUT}s"
echo "---"

# Create and run tasks
for i in $(seq 0 $(( NUM_TASKS - 1 ))); do
    idx=$(( i % ${#TASKS[@]} ))
    task_label="${TASK_LABELS[$idx]}"
    task_content="${TASKS[$idx]}"

    log "[$task_label] Creating session..."

    # Create session WITHOUT ORCH_WORKER header so skill can load naturally
    SESSION_RESP=$(curl -s -X POST "$API/session" \
        -H "Content-Type: application/json" \
        -H "x-opencode-directory: $PROJECT_DIR" \
        -d "$(python3 -c "
import json
print(json.dumps({
    'title': 'GPT-5.4 Orchestrator Test: $task_label',
    'directory': '$PROJECT_DIR',
    'model': '$MODEL',
    'metadata': {'test': 'gpt54-protocol', 'task': '$task_label'},
    'time_ttl': 900
}))
")")

    SESSION_ID=$(echo "$SESSION_RESP" | python3 -c "import json,sys; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

    if [ -z "$SESSION_ID" ]; then
        fail "[$task_label] Failed to create session: $SESSION_RESP"
        RESULTS+=("ERROR")
        DURATIONS+=("0")
        TOKEN_COUNTS+=("0/0")
        continue
    fi

    SESSION_IDS+=("$SESSION_ID")
    TASK_NAMES+=("$task_label")
    log "[$task_label] Session: $SESSION_ID"

    # Build prompt with full orchestrator skill injected
    PROMPT=$(python3 -c "
import json
skill = open('$SKILL_PATH').read()
task = '''$(echo -e "$task_content")'''
prompt = f'''## ORCHESTRATOR SKILL (Full Context)

{skill}

---

## TASK

{task}

---

Respond following the orchestrator skill's behavioral norms and synthesis protocols. Use the decision trees and frameworks from the skill to structure your response.'''
print(json.dumps(prompt))
")

    # Send prompt
    log "[$task_label] Sending prompt (~${SKILL_TOKENS} words of skill + task)..."
    START_TIME=$(date +%s)

    SEND_RESP=$(curl -s -X POST "$API/session/$SESSION_ID/prompt_async" \
        -H "Content-Type: application/json" \
        -H "x-opencode-directory: $PROJECT_DIR" \
        -d "{\"parts\": [{\"type\": \"text\", \"text\": $PROMPT}], \"agent\": \"build\", \"model\": {\"providerID\": \"openai\", \"modelID\": \"gpt-5.4\"}}")

    # Poll for completion
    log "[$task_label] Waiting for response (timeout: ${STALL_TIMEOUT}s)..."
    ELAPSED=0
    STATUS="unknown"
    LAST_TOKEN_CHECK=0
    STALL_COUNT=0

    while [ $ELAPSED -lt $STALL_TIMEOUT ]; do
        sleep $POLL_INTERVAL
        ELAPSED=$(( $(date +%s) - START_TIME ))

        # Check session status - OpenCode only shows actively running sessions
        STATUS_RESP=$(curl -s "$API/session/status" 2>/dev/null || echo "{}")
        SESSION_STATUS=$(echo "$STATUS_RESP" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    for sid, info in data.items():
        if sid == '$SESSION_ID':
            print(info.get('status', 'unknown'))
            break
    else:
        # Not in active sessions — check if it completed
        print('not_active')
except: print('error')
" 2>/dev/null)

        if [ "$SESSION_STATUS" = "idle" ] || [ "$SESSION_STATUS" = "not_active" ]; then
            # Check if we got actual output or an error
            MSG_RESP=$(curl -s "$API/session/$SESSION_ID/message" 2>/dev/null || echo "[]")
            MSG_ANALYSIS=$(echo "$MSG_RESP" | python3 -c "
import json, sys
try:
    msgs = json.load(sys.stdin)
    for m in msgs:
        info = m.get('info', {})
        error = info.get('error', {})
        if error:
            print(f'error:{error.get(\"name\",\"unknown\")}:{error.get(\"data\",{}).get(\"message\",\"\")}')
            sys.exit(0)
        if info.get('role') == 'assistant':
            parts = m.get('parts', [])
            has_text = any(p.get('type') == 'text' and p.get('text','').strip() for p in parts)
            if has_text:
                print('completed')
                sys.exit(0)
    print('no_output')
except Exception as e:
    print(f'parse_error:{e}')
" 2>/dev/null)

            if [[ "$MSG_ANALYSIS" == completed ]]; then
                STATUS="completed"
                break
            elif [[ "$MSG_ANALYSIS" == error:* ]]; then
                STATUS="error"
                ERROR_MSG="${MSG_ANALYSIS#error:}"
                fail "[$task_label] Provider error: $ERROR_MSG"
                break
            else
                # Idle but no messages — might be initializing
                if [ $ELAPSED -gt 60 ]; then
                    STATUS="stalled_no_output"
                    break
                fi
            fi
        elif [ "$SESSION_STATUS" = "busy" ]; then
            STATUS="running"
            # Check for token progress
            TOKENS=$(curl -s "$API/session/$SESSION_ID/message" 2>/dev/null | python3 -c "
import json, sys
try:
    msgs = json.load(sys.stdin)
    total_out = sum(m.get('tokens', {}).get('output', 0) for m in msgs)
    total_in = sum(m.get('tokens', {}).get('input', 0) for m in msgs)
    print(f'{total_in}/{total_out}')
except: print('0/0')
" 2>/dev/null)

            if [ "$TOKENS" = "$LAST_TOKEN_CHECK" ] && [ $ELAPSED -gt 120 ]; then
                STALL_COUNT=$(( STALL_COUNT + 1 ))
                if [ $STALL_COUNT -ge 6 ]; then  # 30s of no token progress after 2min
                    STATUS="stalled_no_progress"
                    break
                fi
            else
                STALL_COUNT=0
                LAST_TOKEN_CHECK="$TOKENS"
            fi

            if [ $(( ELAPSED % 30 )) -lt $POLL_INTERVAL ]; then
                log "[$task_label] Running... ${ELAPSED}s elapsed, tokens: $TOKENS"
            fi
        fi
    done

    END_TIME=$(date +%s)
    DURATION=$(( END_TIME - START_TIME ))

    # Get final token counts
    FINAL_TOKENS=$(curl -s "$API/session/$SESSION_ID/message" 2>/dev/null | python3 -c "
import json, sys
try:
    msgs = json.load(sys.stdin)
    total_out = sum(m.get('tokens', {}).get('output', 0) for m in msgs)
    total_in = sum(m.get('tokens', {}).get('input', 0) for m in msgs)
    print(f'{total_in}/{total_out}')
except: print('0/0')
" 2>/dev/null)

    if [ $ELAPSED -ge $STALL_TIMEOUT ] && [ "$STATUS" != "completed" ]; then
        STATUS="timeout"
    fi

    RESULTS+=("$STATUS")
    DURATIONS+=("$DURATION")
    TOKEN_COUNTS+=("$FINAL_TOKENS")

    case "$STATUS" in
        completed) ok "[$task_label] Completed in ${DURATION}s | Tokens: $FINAL_TOKENS" ;;
        stalled_*) fail "[$task_label] Stalled ($STATUS) after ${DURATION}s | Tokens: $FINAL_TOKENS" ;;
        timeout)   fail "[$task_label] Timeout after ${DURATION}s | Tokens: $FINAL_TOKENS" ;;
        *)         warn "[$task_label] Unknown status: $STATUS after ${DURATION}s" ;;
    esac

    echo "---"
done

# Summary
echo ""
echo "=========================================="
echo "GPT-5.4 ORCHESTRATOR PROTOCOL COMPLIANCE"
echo "=========================================="
echo ""

COMPLETED=0
STALLED=0
ERRORS=0

for i in $(seq 0 $(( ${#RESULTS[@]} - 1 ))); do
    status="${RESULTS[$i]}"
    case "$status" in
        completed) COMPLETED=$((COMPLETED + 1)) ;;
        stalled_*|timeout) STALLED=$((STALLED + 1)) ;;
        *) ERRORS=$((ERRORS + 1)) ;;
    esac

    label="${TASK_NAMES[$i]:-task-$i}"
    duration="${DURATIONS[$i]:-?}"
    tokens="${TOKEN_COUNTS[$i]:-?}"
    echo "  $label: $status (${duration}s, tokens: $tokens)"
done

TOTAL=${#RESULTS[@]}
echo ""
echo "Results: $COMPLETED/$TOTAL completed | $STALLED stalled | $ERRORS errors"

if [ $TOTAL -gt 0 ]; then
    COMPLETION_RATE=$(( COMPLETED * 100 / TOTAL ))
    STALL_RATE=$(( STALLED * 100 / TOTAL ))
    echo "Completion rate: ${COMPLETION_RATE}%"
    echo "Stall rate: ${STALL_RATE}%"
    echo ""

    if [ $COMPLETION_RATE -ge 80 ]; then
        ok "GPT-5.4 PASSES protocol compliance threshold (>= 80%)"
    elif [ $COMPLETION_RATE -ge 50 ]; then
        warn "GPT-5.4 MARGINAL on protocol compliance (50-79%)"
    else
        fail "GPT-5.4 FAILS protocol compliance threshold (< 50%)"
    fi
fi

echo ""
echo "Session IDs for manual inspection:"
for i in $(seq 0 $(( ${#SESSION_IDS[@]} - 1 ))); do
    echo "  ${TASK_NAMES[$i]:-task-$i}: ${SESSION_IDS[$i]}"
done
