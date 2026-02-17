# Probe: Knowledge Tree Shows Duplicate Items Across Phase Groups

**Status:** Active
**Model:** dashboard-architecture
**Date:** 2026-02-16

## Question

Why do items like 'Audit Model/Probe/Investigation Claims' appear under BOTH Phase 3 Review and Phase 4 Review clusters in the knowledge tree?

## What I Tested

1. Read the investigation file that appears duplicated: `.kb/investigations/2026-02-13-inv-audit-model-probe-investigation-claims.md`
2. Examined its Prior-Work table (lines 31-35)
3. Traced through tree building code in `pkg/tree/tree.go`, `pkg/tree/cluster.go`, `pkg/tree/parser.go`
4. Analyzed how relationships are built from Prior-Work tables

## What I Observed

**Root Cause:** Investigations reference multiple models in their Prior-Work tables, creating parent-child relationships with BOTH models.

**Example from `.kb/investigations/2026-02-13-inv-audit-model-probe-investigation-claims.md`:**
```markdown
| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/PHASE4_REVIEW.md | extends | yes | ... |
| .kb/models/PHASE3_REVIEW.md | extends | yes | ... |
```

**Code Flow:**
1. `parseInvestigation()` (parser.go:128-148) parses Prior-Work tables
2. For each row, creates relationship: `From: targetPath, To: investigationPath` (line 143)
3. `BuildRelationshipGraph()` (cluster.go:110-129) converts relationships to parent.Children
4. BOTH PHASE3_REVIEW and PHASE4_REVIEW get the investigation as a child
5. When clusters are built, `buildClusterTree()` (tree.go:127-168) includes the investigation under BOTH parents
6. `cloneNodeRecursive()` (tree.go:215-218) preserves ALL children without deduplication

**The Problem:** There's no deduplication logic. If an investigation references multiple models, it appears as a child of ALL of them.

## Model Impact

**Extends** the dashboard-architecture model's understanding of tree building:

The model doesn't document this multi-parent scenario. The tree building logic assumes a simple parent-child hierarchy, but Prior-Work tables create many-to-many relationships (one investigation can reference multiple models, creating multiple parent relationships).

**New Constraint:** Tree rendering must deduplicate nodes that have multiple parents, or implement a "primary parent" selection strategy.

**Failure Mode Addition:** When investigations reference multiple models in Prior-Work tables, those investigations appear duplicated in the tree view, once under each parent model cluster.
