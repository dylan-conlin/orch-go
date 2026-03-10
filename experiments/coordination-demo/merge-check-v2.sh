#!/usr/bin/env bash
# Merge conflict analysis v2 — uses specific file staging, explicit baseline
set -euo pipefail

RESULTS_DIR="$(cd "${1:?Usage: merge-check-v2.sh RESULTS_DIR}" && pwd)"
BASELINE="${2:-22073e5e7}"
PROJECT_DIR="/Users/dylanconlin/Documents/personal/orch-go"
MERGE_CSV="$RESULTS_DIR/merge_results.csv"

echo "trial,merge_result,conflict_files,notes" > "$MERGE_CSV"
echo "=== Merge Conflict Analysis (baseline: $BASELINE) ==="

for trial_num in $(seq 1 10); do
  haiku_diff="$RESULTS_DIR/trial-$trial_num/haiku/full_diff.txt"
  opus_diff="$RESULTS_DIR/trial-$trial_num/opus/full_diff.txt"

  if [ ! -s "$haiku_diff" ] || [ ! -s "$opus_diff" ]; then
    echo "Trial $trial_num: SKIP"
    echo "$trial_num,skip,0,missing diffs" >> "$MERGE_CSV"
    continue
  fi

  hwt="/tmp/mc-h-${trial_num}-$$"
  owt="/tmp/mc-o-${trial_num}-$$"
  hb="mc-h-${trial_num}-$$"
  ob="mc-o-${trial_num}-$$"

  # Create haiku worktree and apply
  cd "$PROJECT_DIR"
  git worktree add -b "$hb" "$hwt" "$BASELINE" 2>/dev/null
  cd "$hwt"
  git apply "$haiku_diff" --exclude='.beads/*' 2>/dev/null || true
  git add pkg/display/display.go pkg/display/display_test.go 2>/dev/null || true
  git commit -m "haiku t${trial_num}" --no-verify 2>/dev/null || true

  # Create opus worktree and apply
  cd "$PROJECT_DIR"
  git worktree add -b "$ob" "$owt" "$BASELINE" 2>/dev/null
  cd "$owt"
  git apply "$opus_diff" --exclude='.beads/*' 2>/dev/null || true
  git add pkg/display/display.go pkg/display/display_test.go 2>/dev/null || true
  git commit -m "opus t${trial_num}" --no-verify 2>/dev/null || true

  # Attempt merge in haiku worktree
  cd "$hwt"
  merge_out=$(git merge "$ob" --no-edit 2>&1) || true

  if echo "$merge_out" | grep -q "CONFLICT"; then
    cf=$(echo "$merge_out" | grep -c "CONFLICT")
    echo "Trial $trial_num: CONFLICT ($cf files)"
    echo "$trial_num,conflict,$cf,merge conflict" >> "$MERGE_CSV"
    git merge --abort 2>/dev/null || true
  elif echo "$merge_out" | grep -q "Already up to date"; then
    echo "Trial $trial_num: NO CHANGE"
    echo "$trial_num,no_change,0,no changes" >> "$MERGE_CSV"
  else
    # Test post-merge
    test_out=$(go test ./pkg/display/ -v -run TestFormatBytes 2>&1) || true
    if echo "$test_out" | grep -q "PASS"; then
      echo "Trial $trial_num: CLEAN MERGE"
      echo "$trial_num,success,0,clean merge" >> "$MERGE_CSV"
    else
      echo "Trial $trial_num: SEMANTIC CONFLICT"
      echo "$trial_num,semantic_conflict,0,tests fail" >> "$MERGE_CSV"
    fi
  fi

  # Cleanup
  cd "$PROJECT_DIR"
  git worktree remove "$hwt" --force 2>/dev/null || true
  git worktree remove "$owt" --force 2>/dev/null || true
  git branch -D "$hb" 2>/dev/null || true
  git branch -D "$ob" 2>/dev/null || true
done

echo ""
echo "=== Summary ==="
conflicts=$(grep -c ",conflict," "$MERGE_CSV" 2>/dev/null || echo 0)
successes=$(grep -c ",success," "$MERGE_CSV" 2>/dev/null || echo 0)
semantic=$(grep -c ",semantic_conflict," "$MERGE_CSV" 2>/dev/null || echo 0)
no_change=$(grep -c ",no_change," "$MERGE_CSV" 2>/dev/null || echo 0)
total=$((conflicts + successes + semantic + no_change))
if [ "$total" -gt 0 ]; then
  echo "Conflicts: $conflicts/$total"
  echo "Clean merges: $successes/$total"
  echo "Semantic conflicts: $semantic/$total"
  echo "No change: $no_change/$total"
  if [ "$((conflicts + successes + semantic))" -gt 0 ]; then
    valid=$((conflicts + successes + semantic))
    echo "Conflict rate (valid trials): $(echo "scale=0; $conflicts * 100 / $valid" | bc)%"
  fi
fi
echo ""
cat "$MERGE_CSV"
