#!/usr/bin/env python3
"""Analyze investigation stance contrastive experiment results."""

import json
import sys
from pathlib import Path

results_dir = Path("evidence/2026-03-06-investigation-stance-contrastive/results")

variants = {}
for f in ["bare.json", "without-stance.json", "with-stance.json"]:
    with open(results_dir / f) as fh:
        variants[f.replace(".json", "")] = json.load(fh)

# Cross-variant comparison by scenario
print("=" * 70)
print("INVESTIGATION STANCE CONTRASTIVE EXPERIMENT - RESULTS")
print("=" * 70)
print()

scenarios = {}
for vname, vdata in variants.items():
    for s in vdata["scenarios"]:
        if s["name"] not in scenarios:
            scenarios[s["name"]] = {}
        scenarios[s["name"]][vname] = s

for sname, sdata in scenarios.items():
    print(f"### {sname}")
    print()
    print(f"{'Variant':<20} {'Scores':<30} {'Median':>7} {'Pass':>6} {'Mean':>6}")
    print("-" * 70)
    for vname in ["bare", "without-stance", "with-stance"]:
        s = sdata[vname]
        scores = s["all_scores"]
        median = s["median_score"]
        pass_count = s["pass_count"]
        mean = sum(scores) / len(scores)
        print(f"{vname:<20} {str(scores):<30} {median:>5}/8 {pass_count:>4}/6 {mean:>5.1f}")
    print()

    # Per-indicator comparison
    print(f"  {'Indicator':<30} {'Bare':>6} {'NoStance':>10} {'Stance':>8} {'Delta':>7}")
    print("  " + "-" * 62)

    bare_indicators = {i["id"]: i for i in sdata["bare"].get("indicators", [])}
    nostance_indicators = {i["id"]: i for i in sdata["without-stance"].get("indicators", [])}
    stance_indicators = {i["id"]: i for i in sdata["with-stance"].get("indicators", [])}

    for ind_id in bare_indicators:
        bi = bare_indicators[ind_id]
        ni = nostance_indicators.get(ind_id, {})
        si = stance_indicators.get(ind_id, {})
        b_count = bi.get("detected_count", 0)
        n_count = ni.get("detected_count", 0)
        s_count = si.get("detected_count", 0)
        w = bi.get("weight", 1)
        delta = s_count - b_count
        disc = "YES" if abs(delta) >= 2 else "no"
        print(f"  {ind_id} (w{w}){' ' * max(0, 28 - len(ind_id) - 4)}  {b_count}/6  {n_count:>6}/6  {s_count:>4}/6  {delta:>+3} {disc}")
    print()

# Summary statistics
print("=" * 70)
print("CROSS-SCENARIO SUMMARY")
print("=" * 70)
print()
print(f"{'Scenario':<30} {'Bare Med':>9} {'NoSt Med':>9} {'Stance Med':>11} {'Lift':>6}")
print("-" * 70)
for sname, sdata in scenarios.items():
    bm = sdata["bare"]["median_score"]
    nm = sdata["without-stance"]["median_score"]
    sm = sdata["with-stance"]["median_score"]
    lift = sm - bm
    print(f"{sname:<30} {bm:>7}/8 {nm:>7}/8 {sm:>9}/8 {lift:>+4}")

print()
print("Stance lift = with-stance median - bare median")
print("Knowledge lift = without-stance median - bare median")
