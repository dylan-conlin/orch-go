## Summary (D.E.K.N.)

**Delta:** C1/C2 enforcement logic existed locally, but CI had no resource-guardrail workflow and C2 could miss direct `exec.Command(...).Start()` additions.

**Evidence:** Repository had only `.github/workflows/bloat-check.yml`; `scripts/pre-commit-exec-start-cleanup.sh` operated only on staged changes and variable-based `.Start()` usage.

**Knowledge:** We can enforce constraints immediately without paying baseline lint debt by running bounded-lifetime checks in delta mode (`--new-from-rev`) and reusing the same C2 detector in CI with a base-revision mode.

**Next:** Add PR workflow for C1/C2, extend C2 script with `--diff-base`, and include direct `exec.Command(...).Start()` detection in added lines.

**Authority:** implementation

---

# Investigation: C1/C2 Automated Enforcement CI Wiring

**Question:** What is missing to make C1/C2 truly blocking automated checks (including CI), and what minimal changes close those gaps?

**Started:** 2026-02-07
**Updated:** 2026-02-07
**Owner:** OpenCode worker
**Phase:** Complete
**Status:** Complete

## Findings

### Finding 1: C1 analyzer implementation already exists and is wired into custom golangci plugin config

**Evidence:** `pkg/lint/boundedlifetime/analyzer.go`, `pkg/lint/boundedlifetime/plugin.go`, `.custom-gcl.yml`, and `.golangci.yml` are present and tested by `pkg/lint/boundedlifetime/plugin_test.go`.

**Source:** `pkg/lint/boundedlifetime/analyzer.go`, `pkg/lint/boundedlifetime/plugin_test.go`, `.custom-gcl.yml`, `.golangci.yml`.

**Significance:** C1 did not require net-new analyzer architecture; enforcement gaps were mainly integration/gating.

---

### Finding 2: CI had no workflow to run C1/C2 checks on pull requests

**Evidence:** Only `.github/workflows/bloat-check.yml` existed in repo.

**Source:** `.github/workflows/bloat-check.yml` and workflow directory listing.

**Significance:** Constraints could be bypassed by skipping local hooks or contributor environment drift.

---

### Finding 3: Full lint is not currently viable as a hard gate because of large baseline debt

**Evidence:** `make lint` reported broad pre-existing issues from multiple linters.

**Source:** `make lint` output during this session.

**Significance:** New C1 gate should run in delta mode (`--new-from-rev`) to block regressions immediately without requiring full-repo lint cleanup first.

---

### Finding 4: C2 script lacked CI mode and inline direct-call coverage

**Evidence:** Script used `git diff --cached` only and tracked `.Start()` primarily through variable assignments to `exec.Command*`.

**Source:** `scripts/pre-commit-exec-start-cleanup.sh` before changes.

**Significance:** CI could not reuse the detector, and one-line `exec.Command(...).Start()` additions could evade the variable-based path.

---

## Synthesis

The fastest durable path is to keep existing C1/C2 detectors but harden integration: run C1 as a PR delta check through custom golangci (`--enable-only boundedlifetime --new-from-rev`) and run C2 detector in CI with an explicit base revision. This satisfies automated blocking enforcement while avoiding immediate full-repo lint remediation.
