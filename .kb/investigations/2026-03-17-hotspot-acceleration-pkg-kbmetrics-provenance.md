---
Status: Complete
Question: Is pkg/kbmetrics/provenance_test.go a genuine hotspot risk requiring extraction?
---

# Hotspot Acceleration: pkg/kbmetrics/provenance_test.go

**TLDR:** File is 366 lines (24% of 1500-line threshold), all from a single initial commit. Not a hotspot risk — the "366 lines in 30 days" metric is misleading because it's entirely from file creation, not incremental growth.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| N/A - novel investigation | - | - | - |

## D.E.K.N. Summary

- **Delta:** provenance_test.go is not a hotspot. Growth was a one-time creation event, not incremental accretion.
- **Evidence:** Single git commit created the file. File is 366 lines (24% of 1500-line threshold). No subsequent modifications.
- **Knowledge:** Hotspot detection should distinguish between file creation and incremental growth. A file created at 366 lines is structurally different from one that grew 366 lines over many commits.
- **Next:** No extraction needed. If the file grows beyond ~600 lines, the shared test fixture setup (temp dir + model.md writing) could be extracted to a `testhelper_test.go` in the same package.

## Findings

### Finding 1: File size analysis

`provenance_test.go` = 366 lines. The accretion boundary is 1500 lines. The file is at 24% of the threshold.

For comparison, other test files in the same package:
- `claims_test.go` = 238 lines
- `confidence_propagation_test.go` = 250 lines
- `orphans_test.go` = 238 lines

The provenance_test.go is the largest but proportionally similar.

### Finding 2: Growth pattern — single commit

```
git log --oneline --follow -- pkg/kbmetrics/provenance_test.go
2a9e37e39 feat: add kb audit provenance command for evidence quality scanning (orch-go-oc1ra)
```

Only one commit. The entire 366 lines came from file creation. There's been zero incremental growth — the "366 lines/30d" metric is entirely from the initial write.

### Finding 3: Test structure review

8 test functions covering:
1. `TestAuditProvenance_FullCoverage` — 100% annotated model
2. `TestAuditProvenance_Unannotated` — missing annotations
3. `TestAuditProvenance_OrphanContradictions` — contradicting probes
4. `TestAuditProvenance_NoOrphanWhenModelUpdatedAfterProbe` — negative case
5. `TestAuditProvenance_LowConfidenceClaims` — single-source/assumed
6. `TestAuditProvenance_EmptyModelsDir` — edge case
7. `TestAuditProvenance_MultipleModels` — sorting behavior
8. `TestAuditProvenance_ClaimHeadings` — `### Claim N` format
9. `TestFormatProvenanceText_Output` — formatter output

Each test creates temp directories and writes markdown fixtures inline. The boilerplate is repetitive but each test's fixture is materially different (different markdown structures testing different parsing behaviors), so extraction would need a builder pattern rather than simple shared constants.

### Finding 4: Potential extraction path (future, not needed now)

If the file grows beyond ~600 lines, the shared pattern is:
```go
dir := t.TempDir()
kbDir := filepath.Join(dir, ".kb")
modelDir := filepath.Join(kbDir, "models", "test-model")
os.MkdirAll(filepath.Join(modelDir, "probes"), 0755)
os.WriteFile(filepath.Join(modelDir, "model.md"), []byte(content), 0644)
```

This could become a helper like:
```go
func setupTestModel(t *testing.T, name, content string) (kbDir string)
```

The `containsStr` helper is already shared via `orphans_test.go` in the same package.

## Test performed

Verified git log shows single-commit creation. Verified line counts via `wc -l`. Cross-referenced with accretion threshold from CLAUDE.md (1500 lines).

## Conclusion

**No action needed.** The hotspot detection flagged this because the metric "366 lines added in 30 days" doesn't distinguish between file creation and incremental growth. The file is at 24% of the threshold with zero subsequent churn. The test structure is reasonable — each test covers a distinct behavior with materially different fixtures.

**Recommendation:** Adjust hotspot detection to discount initial file creation commits, or weight multi-commit growth higher than single-commit creation.
