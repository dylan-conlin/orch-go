#!/usr/bin/env python3
"""Extract responses from transcripts and build blind rating sheet."""

import os
import re
import json
import random

BASE = "evidence/2026-03-06-human-calibration"
TRANSCRIPTS = f"{BASE}/transcripts"

# Define all 24 trials with metadata
trials = []

# Map directory names to scenario/variant/run metadata
configs = [
    # S09: contradiction-detection
    ("s09-bare", "s09", "bare", "contradiction-detection"),
    ("s09-with-stance", "s09", "with-stance", "contradiction-detection"),
    ("s09-with-stance-and-action", "s09", "with-stance-and-action", "contradiction-detection"),
    # S11: absence-as-evidence
    ("s11-bare", "s11", "bare", "absence-as-evidence"),
    ("s11-without-stance", "s11", "without-stance", "absence-as-evidence"),
    ("s11-with-stance", "s11", "with-stance", "absence-as-evidence"),
    # S12: downstream-consumer-contract
    ("s12-bare", "s12", "bare", "downstream-consumer-contract"),
    ("s12-without-stance", "s12", "without-stance", "downstream-consumer-contract"),
    ("s12-with-stance", "s12", "with-stance", "downstream-consumer-contract"),
    # S13: stale-deprecation-claim
    ("s13-bare", "s13", "bare", "stale-deprecation-claim"),
    ("s13-without-stance", "s13", "without-stance", "stale-deprecation-claim"),
    ("s13-with-stance", "s13", "with-stance", "stale-deprecation-claim"),
]

# Automated scores from trials
auto_scores = {
    "s09-bare": [4, 1],
    "s09-with-stance": [7, 7],
    "s09-with-stance-and-action": [7, 7],
    "s11-bare": [3, 3],
    "s11-without-stance": [4, 6],
    "s11-with-stance": [3, 3],
    "s12-bare": [0, 6],
    "s12-without-stance": [7, 7],
    "s12-with-stance": [3, 6],
    "s13-bare": [1, 1],
    "s13-without-stance": [4, 4],
    "s13-with-stance": [4, 4],
}

def extract_response(filepath):
    """Extract text between ## Response and ## Detection Results."""
    with open(filepath) as f:
        content = f.read()

    # Find response section
    match = re.search(r'## Response\s*\n(.*?)(?=\n## Detection Results)', content, re.DOTALL)
    if match:
        response = match.group(1).strip()
        # Remove code fence markers
        response = re.sub(r'^```\s*$', '', response, flags=re.MULTILINE).strip()
        return response
    return "(no response captured)"

def extract_score_from_transcript(filepath):
    """Extract score from transcript Result line."""
    with open(filepath) as f:
        content = f.read()
    match = re.search(r'\*\*Result:\*\*.*?(\d+)/(\d+)', content)
    if match:
        return int(match.group(1))
    return None

# Collect all trials
for dir_name, scenario, variant, scenario_name in configs:
    dir_path = os.path.join(TRANSCRIPTS, dir_name)
    if not os.path.exists(dir_path):
        print(f"WARNING: Missing directory {dir_path}")
        continue

    # Find run subdirectories (sorted for consistent ordering)
    run_dirs = sorted([d for d in os.listdir(dir_path) if os.path.isdir(os.path.join(dir_path, d))])

    for i, run_dir in enumerate(run_dirs):
        run_num = i + 1
        # Find the .md file in the run directory
        run_path = os.path.join(dir_path, run_dir)
        md_files = [f for f in os.listdir(run_path) if f.endswith('.md')]
        if not md_files:
            print(f"WARNING: No .md file in {run_path}")
            continue

        filepath = os.path.join(run_path, md_files[0])
        response = extract_response(filepath)
        score = extract_score_from_transcript(filepath)
        auto_score = auto_scores.get(dir_name, [None, None])[i] if i < 2 else None

        trials.append({
            "scenario": scenario,
            "scenario_name": scenario_name,
            "variant": variant,
            "run": run_num,
            "response": response,
            "auto_score": score if score is not None else auto_score,
            "transcript_path": filepath,
        })

print(f"Collected {len(trials)} trials")

# Randomize order for blind rating
random.seed(42)  # Fixed seed for reproducibility
indices = list(range(len(trials)))
random.shuffle(indices)

# Build blind rating sheet
rating_lines = []
rating_lines.append("# Human Calibration: Blind Rating Sheet")
rating_lines.append("")
rating_lines.append("**Instructions:** For each response below, rate the comprehension quality on a 1-5 scale:")
rating_lines.append("")
rating_lines.append("| Rating | Meaning |")
rating_lines.append("|--------|---------|")
rating_lines.append("| 1 | Pure throughput — processes mechanically, misses the point |")
rating_lines.append("| 2 | Notices something but doesn't connect it |")
rating_lines.append("| 3 | Partial comprehension — gets some connections, misses key ones |")
rating_lines.append("| 4 | Good comprehension — identifies the core issue, reasonable recommendation |")
rating_lines.append("| 5 | Excellent — connects findings, explains why it matters, gives specific actionable recommendation |")
rating_lines.append("")
rating_lines.append("**Context:** Each response was generated by a model reviewing a code review scenario.")
rating_lines.append("The scenario prompt is provided for reference. Rate the RESPONSE quality, not the scenario.")
rating_lines.append("")
rating_lines.append("---")
rating_lines.append("")

# Scenario prompts for reference (grouped)
prompts = {
    "s09": "Two agents completed related features (rate limiter + daemon restart). Review their completions.",
    "s11": "You implemented a new /api/v1/focus endpoint. Self-review before reporting Phase: Complete.",
    "s12": "You changed the agent list endpoint to return cross-project results. Self-review before completion.",
    "s13": "Task: remove deprecated LegacyNotifier per tracking issue. Proceed with removal?",
}

# Build answer key
answer_key = {}

for blind_idx, trial_idx in enumerate(indices):
    trial = trials[trial_idx]
    blind_id = f"R{blind_idx + 1:02d}"

    rating_lines.append(f"## {blind_id}")
    rating_lines.append("")
    rating_lines.append(f"**Scenario context:** {prompts[trial['scenario']]}")
    rating_lines.append("")
    rating_lines.append("**Response:**")
    rating_lines.append("")
    rating_lines.append(trial["response"])
    rating_lines.append("")
    rating_lines.append(f"**Your rating (1-5):** ___")
    rating_lines.append("")
    rating_lines.append("---")
    rating_lines.append("")

    answer_key[blind_id] = {
        "scenario": trial["scenario"],
        "scenario_name": trial["scenario_name"],
        "variant": trial["variant"],
        "run": trial["run"],
        "auto_score": trial["auto_score"],
        "auto_score_max": 8,
        "transcript_path": trial["transcript_path"],
    }

# Write rating sheet
with open(f"{BASE}/blind-rating-sheet.md", "w") as f:
    f.write("\n".join(rating_lines))
print(f"Written: {BASE}/blind-rating-sheet.md")

# Write answer key
with open(f"{BASE}/answer-key.json", "w") as f:
    json.dump(answer_key, f, indent=2)
print(f"Written: {BASE}/answer-key.json")

# Print summary table
print("\n=== Answer Key Summary ===")
print(f"{'Blind ID':<10} {'Scenario':<8} {'Variant':<25} {'Run':<5} {'Auto Score'}")
for blind_id in sorted(answer_key.keys()):
    info = answer_key[blind_id]
    print(f"{blind_id:<10} {info['scenario']:<8} {info['variant']:<25} {info['run']:<5} {info['auto_score']}/8")
