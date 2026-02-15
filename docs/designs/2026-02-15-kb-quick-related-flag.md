# Design: --related Flag for kb quick Commands

**Date:** 2026-02-15
**Status:** Implementation
**Owner:** Worker Agent (orch-go-xgx6)
**Parent:** Phase 5 of `docs/designs/2026-02-15-collaborative-tree-building.md`

## Problem Statement

Quick entries (`kb quick decide/constrain/tried`) currently live in `.kb/quick/entries.jsonl` and are invisible in the knowledge tree. When an orchestrator makes a quick decision during a session related to a parent artifact (like a decision or investigation), there's no way to express that relationship.

The tree should show quick entries attached to their parent artifacts to preserve context and lineage.

## Solution

Add a `--related <path>` flag to the three kb quick commands that create entries with reasoning:
- `kb quick decide`
- `kb quick constrain`  
- `kb quick tried`

(Note: `kb quick question` doesn't need this flag - questions don't attach to specific artifacts)

### Example Usage

```bash
kb quick decide "serialize verification" \
  --reason "Ensures reproducibility and debugging" \
  --related decisions/2026-02-14-verifiability-first-hard-constraint.md
```

This creates a quick decision that appears as a child of the parent decision in the knowledge tree.

## Architecture

### 1. Data Model (kb-cli)

**File:** `~/Documents/personal/kb-cli/cmd/kb/quick.go`

Add new field to `QuickEntry` struct (line 37-70):

```go
type QuickEntry struct {
    ID        string    `json:"id"`
    Type      string    `json:"type"`
    Content   string    `json:"content"`
    // ... existing fields ...
    
    // NEW: Related artifact path
    RelatedArtifact string `json:"related_artifact,omitempty"`
    
    // ... rest of fields ...
}
```

### 2. CLI Flag Addition (kb-cli)

Add `--related` flag to three commands:

**kb quick decide** (line 125-164):
```go
var related string
cmd.Flags().StringVar(&related, "related", "", "Path to related parent artifact")
```

**kb quick constrain** (line 205-247):
```go
var related string
cmd.Flags().StringVar(&related, "related", "", "Path to related parent artifact")
```

**kb quick tried** (line 166-203):
```go
var related string
cmd.Flags().StringVar(&related, "related", "", "Path to related parent artifact")
```

Update the Create* functions to accept and store the related path:
- `CreateDecision(content, reason, tags, scope, related string)`
- `CreateConstraint(content, reason, source, tags, scope, related string)`
- `CreateAttempt(content, failed, tags, scope, related string)`

### 3. Tree Extraction (orch-go)

**File:** `pkg/tree/parser.go`

Add new function to parse quick entries:

```go
// ParseQuickEntries parses quick entries from .kb/quick/entries.jsonl
func ParseQuickEntries(kbDir string) ([]*KnowledgeNode, []Relationship, error)
```

This function will:
1. Read `.kb/quick/entries.jsonl` line by line
2. Parse JSON entries
3. Filter for active entries with `related_artifact` field set
4. Create KnowledgeNode for each quick entry
5. Create Relationship from parent artifact to quick entry

**File:** `pkg/tree/types.go`

Add new node type:

```go
const (
    // ... existing types ...
    NodeTypeQuickDecision   NodeType = "quick_decision"
    NodeTypeQuickConstraint NodeType = "quick_constraint"
    NodeTypeQuickAttempt    NodeType = "quick_attempt"
)
```

**File:** `pkg/tree/tree.go`

Update `BuildKnowledgeTree()` to include quick entries:

```go
// After parsing guides, before building relationship graph:
quickDir := filepath.Join(kbDir, "quick")
if _, err := os.Stat(quickDir); err == nil {
    quickNodes, quickRels, err := ParseQuickEntries(kbDir)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to parse quick entries: %w", err)
    }
    allNodes = append(allNodes, quickNodes...)
    allRelationships = append(allRelationships, quickRels...)
}
```

### 4. Path Resolution

The `--related` flag accepts paths in multiple formats:
- Relative to `.kb/`: `decisions/2026-02-14-something.md`
- Full `.kb/` prefix: `.kb/decisions/2026-02-14-something.md`
- Absolute path: `/full/path/to/.kb/decisions/2026-02-14-something.md`

The tree extraction will normalize these to match artifact IDs using the existing `resolveKbPath()` function.

## Verification Strategy

### Unit Tests (kb-cli)

1. Test `--related` flag sets field correctly
2. Test path normalization
3. Test JSONL storage includes `related_artifact` field

### Integration Tests (orch-go)

1. Test quick entries appear as children in tree output
2. Test entries appear in both knowledge view and timeline view (future)
3. Test path resolution handles all formats

### Manual Verification

```bash
# 1. Create a decision
cd ~/Documents/personal/orch-go
kb quick decide "test related decision" \
  --reason "Testing the related flag functionality" \
  --related decisions/2026-02-14-knowledge-lineage-tree-visualization.md

# 2. Check JSONL contains related_artifact field
cat .kb/quick/entries.jsonl | tail -1 | jq .

# 3. Verify tree extraction shows quick entry as child
orch tree --cluster decisions

# Expected: Quick decision appears under parent decision node
```

## Timeline View (Future Work)

Quick entries will also appear in timeline view (chronological session history). This is Phase 4 of the parent design and requires:
- Session tracking (`orch session label`)
- Timeline view implementation in dashboard
- Correlation of quick entries to session context

For now, we only implement the relationship storage and knowledge tree attachment.

## Dependencies

- Decision `kb-bcd869` (conceptual - the decision to add --related flag)
- Design doc: `docs/designs/2026-02-15-collaborative-tree-building.md`

## Non-Goals

- Validation that related path exists (fail open - orphaned quick entries just don't attach)
- Bidirectional relationship tracking (parent doesn't know about children)
- Timeline view implementation (separate phase)
- Dashboard visualization (separate phase)

## Migration

No migration needed - existing entries without `related_artifact` field continue to work. They just don't appear in the knowledge tree (same as current behavior).

## Open Questions

None - design is straightforward.
