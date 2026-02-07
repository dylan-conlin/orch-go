---
blocks:
  - keywords:
      - "test feature"
      - "sample implementation"
---

# Decision: Test Decision Gate

**Date:** 2026-01-28
**Status:** Test Fixture
**Context:** Test fixture for decision gate functionality

## Summary

This is a test decision file used to verify that the decision gate correctly detects and blocks spawns containing blocked keywords. Do not proceed with test feature work without explicit acknowledgment.

## The Problem

Decision gates need test coverage to ensure they work correctly.

## The Decision

This decision blocks any spawn containing "test feature" or "sample implementation" keywords unless explicitly acknowledged with `--acknowledge-decision 2026-01-28-test-decision-gate`.

## Why This Design

This is a test fixture to validate the decision gate implementation.

## Evidence

- Test file: `cmd/orch/decision_gate_test.go`

## Related Decisions

N/A - This is a test fixture
