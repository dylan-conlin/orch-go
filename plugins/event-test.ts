/**
 * Plugin: Event Testing
 *
 * Test plugin to observe and log OpenCode events for reliability testing.
 * Logs all events to ~/.orch/event-test.jsonl with timestamps.
 *
 * Tests:
 * 1. file.edited event reliability - does it fire for Edit tool calls?
 * 2. session.idle timing - how long until it fires?
 * 3. Multiple plugin interaction - event ordering
 */

import type { Plugin } from "@opencode-ai/plugin"
import { appendFileSync, mkdirSync } from "fs"
import { homedir } from "os"
import { join, dirname } from "path"

const LOG_PATH = join(homedir(), ".orch", "event-test.jsonl")

interface EventLog {
  timestamp: string
  event_type: string
  session_id?: string
  properties?: any
  source: string
}

function logEvent(event: EventLog): void {
  try {
    mkdirSync(dirname(LOG_PATH), { recursive: true })
    appendFileSync(LOG_PATH, JSON.stringify(event) + "\n")
  } catch (err) {
    console.error("[event-test] Failed to log:", err)
  }
}

// Track session start times for idle timing measurement
const sessionStartTimes = new Map<string, number>()
const sessionLastActivity = new Map<string, number>()

export const EventTestPlugin: Plugin = async ({
  project,
  client,
  $,
  directory,
  worktree,
}) => {
  logEvent({
    timestamp: new Date().toISOString(),
    event_type: "plugin.initialized",
    source: "event-test",
    properties: {
      project: project?.id,
      directory,
      worktree,
    },
  })

  return {
    // Track ALL events via the event hook
    event: async ({ event }) => {
      const now = Date.now()
      const eventType = event.type
      const props = (event as any).properties || {}
      const sessionId = props.sessionID

      // Track session timing
      if (eventType === "session.created" && sessionId) {
        sessionStartTimes.set(sessionId, now)
        sessionLastActivity.set(sessionId, now)
      }

      // For session.idle, calculate timing
      let idleTiming: { sinceStart?: number; sinceLastActivity?: number } = {}
      if (eventType === "session.idle" && sessionId) {
        const startTime = sessionStartTimes.get(sessionId)
        const lastActivity = sessionLastActivity.get(sessionId)
        if (startTime) {
          idleTiming.sinceStart = now - startTime
        }
        if (lastActivity) {
          idleTiming.sinceLastActivity = now - lastActivity
        }
      }

      // Update last activity on non-idle events
      if (eventType !== "session.idle" && sessionId) {
        sessionLastActivity.set(sessionId, now)
      }

      logEvent({
        timestamp: new Date().toISOString(),
        event_type: eventType,
        session_id: sessionId,
        source: "event-test-event-hook",
        properties: {
          ...props,
          ...(Object.keys(idleTiming).length > 0 ? { idleTiming } : {}),
        },
      })
    },

    // Track file.edited specifically via tool.execute.after for Edit tool
    "tool.execute.after": async (input, output) => {
      const tool = input.tool?.toLowerCase()

      // Log all tool executions for completeness
      logEvent({
        timestamp: new Date().toISOString(),
        event_type: `tool.executed.${tool}`,
        session_id: input.sessionID,
        source: "event-test-tool-hook",
        properties: {
          tool,
          callID: input.callID,
          title: output.title,
          hasOutput: !!output.output,
          outputLength: output.output?.length || 0,
        },
      })

      // Update session activity
      if (input.sessionID) {
        sessionLastActivity.set(input.sessionID, Date.now())
      }
    },
  }
}
