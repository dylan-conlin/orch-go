import { describe, expect, it } from "bun:test"

import {
  getActiveCompoundSignals,
  parsePhaseCommentFromCommand,
  summarizeCompoundSignals,
} from "./coaching"

describe("coaching frustration helpers", () => {
  it("parses phase comments from bd commands", () => {
    expect(
      parsePhaseCommentFromCommand(
        'bd comments add orch-go-123 "Phase: Testing - running boundary checks"',
      ),
    ).toBe("Testing")
  })

  it("summarizes compound frustration signals in stable order", () => {
    expect(
      summarizeCompoundSignals(["circular_pattern", "behavioral_variation", "time_in_phase"]),
    ).toBe("thrashing, stuck, and contradicting prior work")
  })

  it("filters stale frustration signals from the compound window", () => {
    const now = Date.now()
    expect(
      getActiveCompoundSignals(
        {
          behavioral_variation: now - 5 * 60 * 1000,
          time_in_phase: now - 10 * 60 * 1000,
          circular_pattern: now - 25 * 60 * 1000,
        },
        now,
      ),
    ).toEqual(["behavioral_variation", "time_in_phase"])
  })
})
