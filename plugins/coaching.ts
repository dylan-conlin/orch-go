/**
 * Plugin: Orchestrator Coaching - Behavioral Pattern Detection
 *
 * Purpose: Track orchestrator behavioral patterns to detect Level 1→2 patterns:
 * - Option theater (low action ratio)
 * - Missing strategic reasoning (low context-gathering ratio)
 * - Analysis paralysis (tool repetition sequences)
 * - Behavioral variation (3+ debugging attempts without strategic pause)
 *
 * Hypothesis: Do quantified metrics drive orchestrator behavior change?
 *
 * Architecture:
 * - Track tool usage patterns via tool.execute.after
 * - Calculate behavioral metrics (ratios, sequences, variations)
 * - Write to ~/.orch/coaching-metrics.jsonl
 * - Exposed via /api/coaching endpoint
 * - Displayed in dashboard
 *
 * Phase 1 (Behavioral Variation Detection):
 * - Semantic tool grouping: bash commands grouped by domain (process_mgmt, git, etc.)
 * - Variation counter: track consecutive attempts in same semantic group
 * - Strategic pause heuristic: 30s no tools = pause (resets variation counter)
 * - Emit behavioral_variation metric when 3+ variations detected
 *
 * Reference: docs/designs/2026-01-10-orchestrator-coaching-plugin.md
 * Reference: .kb/investigations/2026-01-10-inv-probe-pattern-retrospective-validate-detection.md
 */

import type { Plugin } from "@opencode-ai/plugin"
import { appendFileSync, mkdirSync, existsSync, readFileSync, writeFileSync } from "fs"
import { homedir } from "os"
import { join, dirname } from "path"

const LOG_PREFIX = "[coaching]"
const DEBUG = process.env.ORCH_PLUGIN_DEBUG === "1"
const METRICS_PATH = join(homedir(), ".orch", "coaching-metrics.jsonl")
const MAX_LINES = 1000 // Keep last 1000 lines
const STRATEGIC_PAUSE_MS = 30 * 1000 // 30 seconds = strategic pause
const VARIATION_THRESHOLD = 3 // 3+ variations triggers behavioral_variation metric

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
 * Semantic groups for bash commands.
 * Commands in the same group are considered "similar" for variation detection.
 * Example: overmind start, overmind status, overmind restart = all "process_mgmt"
 */
type SemanticGroup =
  | "process_mgmt" // overmind, tmux, launchd, launchctl, systemctl
  | "git" // git commands
  | "build" // make, go build, npm, bun
  | "test" // go test, npm test, jest, pytest
  | "knowledge" // kb, bd commands
  | "orch" // orch spawn, orch status, etc.
  | "file_ops" // ls, cat, mkdir, cp, mv, rm
  | "network" // curl, wget, nc, ssh
  | "other" // fallback

/**
 * Patterns for semantic command classification.
 * Order matters - first match wins.
 * More specific patterns should come before general ones.
 */
const SEMANTIC_PATTERNS: Array<{ group: SemanticGroup; patterns: RegExp[] }> = [
  {
    group: "process_mgmt",
    patterns: [
      /\bovermind\b/,
      /\btmux\b/,
      /\blaunchd\b/,
      /\blaunchctl\b/,
      /\bsystemctl\b/,
      /\bservice\b/,
      /\bkill\b/,
      /\bpkill\b/,
      /\bps\b.*aux/,
      /\bpgrep\b/,
      /\blsof\b/,
    ],
  },
  {
    group: "git",
    patterns: [/\bgit\b/],
  },
  // Test patterns must come BEFORE build patterns (npm test vs npm install)
  {
    group: "test",
    patterns: [/\bgo test\b/, /\bnpm test\b/, /\bjest\b/, /\bpytest\b/, /\bvitest\b/],
  },
  {
    group: "build",
    patterns: [/\bmake\b/, /\bgo build\b/, /\bgo install\b/, /\bnpm\b/, /\bbun\b/, /\bcargo\b/],
  },
  {
    group: "knowledge",
    patterns: [/\bkb\b/, /\bbd\b/],
  },
  // Orch pattern: must be command start, not part of a path like ~/.orch/
  {
    group: "orch",
    patterns: [/^orch\b/, /\borch spawn\b/, /\borch status\b/, /\borch complete\b/],
  },
  {
    group: "file_ops",
    patterns: [/\bls\b/, /\bcat\b/, /\bmkdir\b/, /\bcp\b/, /\bmv\b/, /\brm\b/, /\bfind\b/, /\brg\b/],
  },
  {
    group: "network",
    patterns: [/\bcurl\b/, /\bwget\b/, /\bnc\b/, /\bssh\b/, /\bhttp\b/],
  },
]

/**
 * Classify a bash command into a semantic group.
 */
function classifyBashCommand(command: string): SemanticGroup {
  if (!command) return "other"

  for (const { group, patterns } of SEMANTIC_PATTERNS) {
    for (const pattern of patterns) {
      if (pattern.test(command)) {
        return group
      }
    }
  }

  return "other"
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
 * Variation tracking state for behavioral variation detection.
 */
interface VariationState {
  currentGroup: SemanticGroup | null // Current semantic group being worked on
  variationCount: number // Consecutive variations in same group
  lastToolTimestamp: number // When last tool was called (for pause detection)
  variationHistory: Array<{ group: SemanticGroup; command: string; timestamp: number }> // Recent commands
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
  // Phase 1: Behavioral variation detection
  variation: VariationState
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
          variation: {
            currentGroup: null,
            variationCount: 0,
            lastToolTimestamp: Date.now(),
            variationHistory: [],
          },
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

        // Phase 1: Behavioral variation detection
        const now = Date.now()
        const timeSinceLastTool = now - state.variation.lastToolTimestamp

        // Strategic pause detection: 30s+ without tools resets variation counter
        if (timeSinceLastTool >= STRATEGIC_PAUSE_MS) {
          if (state.variation.variationCount > 0) {
            log(
              "Strategic pause detected (",
              Math.round(timeSinceLastTool / 1000),
              "s), resetting variation counter from",
              state.variation.variationCount
            )
          }
          state.variation.currentGroup = null
          state.variation.variationCount = 0
        }

        // Classify command into semantic group
        const group = classifyBashCommand(command)
        state.variation.lastToolTimestamp = now

        // Track variation history (keep last 20 commands)
        state.variation.variationHistory.push({ group, command, timestamp: now })
        if (state.variation.variationHistory.length > 20) {
          state.variation.variationHistory.shift()
        }

        // Update variation counter
        if (group !== "other") {
          if (state.variation.currentGroup === group) {
            // Same group - increment variation counter
            state.variation.variationCount++
            log(
              `Variation ${state.variation.variationCount} in ${group}: ${command.substring(0, 60)}`
            )

            // Emit behavioral_variation metric when threshold reached
            if (state.variation.variationCount >= VARIATION_THRESHOLD) {
              const recentHistory = state.variation.variationHistory
                .filter((h) => h.group === group)
                .slice(-VARIATION_THRESHOLD)

              writeMetric({
                timestamp: new Date().toISOString(),
                session_id: state.sessionId,
                metric_type: "behavioral_variation",
                value: state.variation.variationCount,
                details: {
                  group,
                  commands: recentHistory.map((h) => h.command.substring(0, 100)),
                  threshold: VARIATION_THRESHOLD,
                },
              })

              log(
                `⚠️ BEHAVIORAL VARIATION DETECTED: ${state.variation.variationCount} variations in ${group} without strategic pause`
              )
            }
          } else {
            // Different group - reset counter
            if (state.variation.currentGroup !== null) {
              log(
                `Group switch: ${state.variation.currentGroup} → ${group}, resetting variation counter`
              )
            }
            state.variation.currentGroup = group
            state.variation.variationCount = 1
          }
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
