/**
 * Plugin: Orchestrator Coaching - Behavioral Pattern Detection
 *
 * Purpose: Track orchestrator behavioral patterns to detect Level 1→2 patterns:
 * - Option theater (low action ratio)
 * - Missing strategic reasoning (low context-gathering ratio)
 * - Analysis paralysis (tool repetition sequences)
 *
 * Hypothesis: Do quantified metrics drive orchestrator behavior change?
 *
 * Architecture:
 * - Track tool usage patterns via tool.execute.after
 * - Calculate behavioral metrics (ratios, sequences)
 * - Write to ~/.orch/coaching-metrics.jsonl
 * - Exposed via /api/coaching endpoint
 * - Displayed in dashboard
 *
 * Reference: docs/designs/2026-01-10-orchestrator-coaching-plugin.md
 */

import type { Plugin } from "@opencode-ai/plugin"
import { appendFileSync, mkdirSync, existsSync, readFileSync, writeFileSync } from "fs"
import { homedir } from "os"
import { join, dirname } from "path"

const LOG_PREFIX = "[coaching]"
const DEBUG = process.env.ORCH_PLUGIN_DEBUG === "1"
const METRICS_PATH = join(homedir(), ".orch", "coaching-metrics.jsonl")
const MAX_LINES = 1000 // Keep last 1000 lines

function log(...args: any[]) {
  if (DEBUG) console.log(LOG_PREFIX, ...args)
}

interface CoachingMetric {
  timestamp: string
  session_id?: string
  metric_type: string
  value: number
  details?: any
}

/**
 * Tool categories for behavioral analysis.
 */
const TOOL_CATEGORIES = {
  // Context-gathering tools
  context: ["bash"], // kb context calls detected via bash
  // Reading/exploration tools
  read: ["read", "grep", "glob"],
  // Action tools (making changes)
  action: ["edit", "write"],
  // Execution tools
  execute: ["bash"],
}

/**
 * Detect if a bash command is a context check.
 */
function isContextCheck(command: string): boolean {
  if (!command) return false
  return command.includes("kb context") || command.includes("kb search")
}

/**
 * Detect if a bash command is a spawn.
 */
function isSpawn(command: string): boolean {
  if (!command) return false
  return command.includes("orch spawn")
}

/**
 * Write metric to JSONL file.
 */
function writeMetric(metric: CoachingMetric): void {
  try {
    mkdirSync(dirname(METRICS_PATH), { recursive: true })
    appendFileSync(METRICS_PATH, JSON.stringify(metric) + "\n")
    log("Wrote metric:", metric.metric_type, "=", metric.value)
  } catch (err) {
    if (DEBUG) console.error(LOG_PREFIX, "Failed to write metric:", err)
  }
}

/**
 * Prune old metrics to keep file size manageable.
 */
function pruneMetrics(): void {
  try {
    if (!existsSync(METRICS_PATH)) return

    const content = readFileSync(METRICS_PATH, "utf-8")
    const lines = content.split("\n").filter((line) => line.trim() !== "")

    if (lines.length > MAX_LINES) {
      const keep = lines.slice(-MAX_LINES)
      writeFileSync(METRICS_PATH, keep.join("\n") + "\n")
      log(`Pruned metrics file: ${lines.length} → ${keep.length} lines`)
    }
  } catch (err) {
    if (DEBUG) console.error(LOG_PREFIX, "Failed to prune metrics:", err)
  }
}

/**
 * Session state tracker.
 */
interface SessionState {
  sessionId: string
  contextChecks: number
  spawns: number
  reads: number
  actions: number
  toolWindow: string[] // Last 10 tools
  lastFlush: number
}

/**
 * Detect tool repetition sequence (analysis paralysis signal).
 */
function detectSequence(tools: string[]): number {
  if (tools.length < 3) return 0

  let maxSequence = 0
  let currentTool = tools[0]
  let currentCount = 1

  for (let i = 1; i < tools.length; i++) {
    if (tools[i] === currentTool) {
      currentCount++
    } else {
      if (currentCount >= 3) {
        maxSequence = Math.max(maxSequence, currentCount)
      }
      currentTool = tools[i]
      currentCount = 1
    }
  }

  // Check final sequence
  if (currentCount >= 3) {
    maxSequence = Math.max(maxSequence, currentCount)
  }

  return maxSequence
}

/**
 * Calculate and flush metrics for a session.
 */
function flushMetrics(state: SessionState): void {
  const now = new Date().toISOString()

  // Context ratio: context checks per spawn
  if (state.spawns > 0) {
    const contextRatio = state.contextChecks / state.spawns
    writeMetric({
      timestamp: now,
      session_id: state.sessionId,
      metric_type: "context_ratio",
      value: parseFloat(contextRatio.toFixed(2)),
      details: {
        context_checks: state.contextChecks,
        spawns: state.spawns,
      },
    })
  }

  // Action ratio: actions per reads
  if (state.reads > 0) {
    const actionRatio = state.actions / state.reads
    writeMetric({
      timestamp: now,
      session_id: state.sessionId,
      metric_type: "action_ratio",
      value: parseFloat(actionRatio.toFixed(2)),
      details: {
        actions: state.actions,
        reads: state.reads,
      },
    })
  }

  // Tool repetition sequence (analysis paralysis)
  const sequence = detectSequence(state.toolWindow)
  if (sequence >= 3) {
    writeMetric({
      timestamp: now,
      session_id: state.sessionId,
      metric_type: "analysis_paralysis",
      value: sequence,
      details: {
        window: state.toolWindow.slice(-10),
      },
    })
  }

  state.lastFlush = Date.now()
  log("Flushed metrics for session:", state.sessionId)
}

/**
 * OpenCode plugin that tracks orchestrator behavioral patterns.
 */
export const CoachingPlugin: Plugin = async ({ directory }) => {
  log("Plugin initialized, directory:", directory)

  // Prune old metrics on startup
  pruneMetrics()

  // Session state (Map<sessionID, SessionState>)
  const sessions = new Map<string, SessionState>()

  // Tool call counter for periodic flush
  let toolCallCounter = 0

  return {
    /**
     * Track all tool executions and update session state.
     */
    "tool.execute.after": async (input: any, output: any) => {
      const tool = input.tool?.toLowerCase()
      const sessionId = input.sessionID

      if (!tool || !sessionId) return

      // Get or create session state
      let state = sessions.get(sessionId)
      if (!state) {
        state = {
          sessionId,
          contextChecks: 0,
          spawns: 0,
          reads: 0,
          actions: 0,
          toolWindow: [],
          lastFlush: Date.now(),
        }
        sessions.set(sessionId, state)
        log("Created session state:", sessionId)
      }

      // Update tool window (keep last 10)
      state.toolWindow.push(tool)
      if (state.toolWindow.length > 10) {
        state.toolWindow.shift()
      }

      // Track tool categories
      if (TOOL_CATEGORIES.read.includes(tool)) {
        state.reads++
      }

      if (TOOL_CATEGORIES.action.includes(tool)) {
        state.actions++
      }

      // Special handling for bash commands
      if (tool === "bash") {
        const command = (input as any).args?.command || ""

        if (isContextCheck(command)) {
          state.contextChecks++
          log("Context check detected:", command.substring(0, 50))
        }

        if (isSpawn(command)) {
          state.spawns++
          log("Spawn detected:", command.substring(0, 50))
        }
      }

      // Periodic flush: every 10 tool calls
      toolCallCounter++
      if (toolCallCounter >= 10) {
        // Flush all active sessions
        for (const [sid, s] of sessions.entries()) {
          // Only flush if there's activity to report
          if (s.spawns > 0 || s.reads > 0 || s.actions > 0) {
            flushMetrics(s)
          }
        }
        toolCallCounter = 0
      }

      // Also flush if session has been active for 5+ minutes since last flush
      const now = Date.now()
      if (now - state.lastFlush > 5 * 60 * 1000) {
        if (state.spawns > 0 || state.reads > 0 || state.actions > 0) {
          flushMetrics(state)
        }
      }
    },
  }
}
