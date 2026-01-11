/**
 * Plugin: Orchestrator Coaching - Behavioral Pattern Detection
 *
 * Purpose: Track orchestrator behavioral patterns to detect Level 1→2 patterns:
 * - Option theater (low action ratio)
 * - Missing strategic reasoning (low context-gathering ratio)
 * - Analysis paralysis (tool repetition sequences)
 * - Behavioral variation (3+ debugging attempts without strategic pause)
 * - Circular patterns (contradicting prior investigation recommendations)
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
 * Phase 2 (Cross-Document Circular Detection):
 * - Parse investigation D.E.K.N. summaries from .kb/investigations/*.md
 * - Extract "Next:" field recommendations (architectural guidance)
 * - Track architectural decisions in session (git commits, bd create, file edits)
 * - Emit circular_pattern metric when decisions contradict prior recommendations
 * - Example: "Next: Use overmind" → session creates launchd plists
 *
 * Reference: docs/designs/2026-01-10-orchestrator-coaching-plugin.md
 * Reference: .kb/investigations/2026-01-10-inv-probe-pattern-retrospective-validate-detection.md
 * Reference: .kb/investigations/2026-01-10-inv-phase-cross-document-parsing-circular.md
 */

import type { Plugin } from "@opencode-ai/plugin"
import {
  appendFileSync,
  mkdirSync,
  existsSync,
  readFileSync,
  writeFileSync,
  readdirSync,
  statSync,
} from "fs"
import { homedir } from "os"
import { join, dirname } from "path"

const LOG_PREFIX = "[coaching]"
const DEBUG = process.env.ORCH_PLUGIN_DEBUG === "1"
const METRICS_PATH = join(homedir(), ".orch", "coaching-metrics.jsonl")
const MAX_LINES = 1000 // Keep last 1000 lines
const STRATEGIC_PAUSE_MS = 30 * 1000 // 30 seconds = strategic pause
const VARIATION_THRESHOLD = 3 // 3+ variations triggers behavioral_variation metric
const COACH_SESSION_ID = process.env.ORCH_COACH_SESSION_ID || "" // Coach session to stream metrics to

function log(...args: any[]) {
  if (DEBUG) console.log(LOG_PREFIX, ...args)
}

/**
 * NOTE: Worker detection moved to per-session detection in tool hooks.
 * Plugin-level detection (checking process.env.ORCH_WORKER) doesn't work because
 * the plugin runs in the OpenCode server process, not in spawned agent processes.
 * See detectWorkerSession() function below for the correct implementation.
 */

interface CoachingMetric {
  timestamp: string
  session_id?: string
  metric_type: string
  value: number
  details?: any
}

/**
 * Phase 2: Investigation recommendation for circular pattern detection.
 */
interface InvestigationRecommendation {
  filePath: string // Path to investigation file
  fileName: string // Just the filename for display
  date: string // From filename YYYY-MM-DD
  next: string // Content of **Next:** field
  keywords: string[] // Extracted keywords for matching (launchd, overmind, etc.)
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
 * Phase 2: Parse D.E.K.N. Summary from investigation markdown.
 * Extracts the **Next:** field which contains architectural recommendations.
 */
function parseDEKNSummary(content: string): string | null {
  // Look for ## Summary (D.E.K.N.) section
  const deknMatch = content.match(/## Summary \(D\.E\.K\.N\.\)([\s\S]*?)(?=\n##|$)/i)
  if (!deknMatch) {
    return null
  }

  const deknSection = deknMatch[1]

  // Extract **Next:** field
  const nextMatch = deknSection.match(/\*\*Next:\*\*\s+(.+?)(?=\n\*\*|$)/s)
  if (!nextMatch) {
    return null
  }

  return nextMatch[1].trim()
}

/**
 * Phase 2: Extract architectural keywords from recommendation text.
 * Keywords are used for contradiction detection.
 */
function extractKeywords(text: string): string[] {
  const lowerText = text.toLowerCase()
  const keywords: string[] = []

  // Architecture patterns
  const patterns = [
    "launchd",
    "overmind",
    "tmux",
    "systemd",
    "docker",
    "kubernetes",
    "procfile",
    "plist",
    "daemon",
    "supervisor",
  ]

  for (const pattern of patterns) {
    if (lowerText.includes(pattern)) {
      keywords.push(pattern)
    }
  }

  return keywords
}

/**
 * Phase 2: Load all investigation recommendations from .kb/investigations/.
 * Returns array of parsed recommendations with Next field and keywords.
 */
function loadInvestigationRecommendations(directory: string): InvestigationRecommendation[] {
  const recommendations: InvestigationRecommendation[] = []
  const invDir = join(directory, ".kb", "investigations")

  if (!existsSync(invDir)) {
    log("Investigation directory not found:", invDir)
    return recommendations
  }

  try {
    const files = readdirSync(invDir)

    for (const file of files) {
      if (!file.endsWith(".md")) continue

      const filePath = join(invDir, file)
      const stat = statSync(filePath)
      if (!stat.isFile()) continue

      // Extract date from filename (YYYY-MM-DD-...)
      const dateMatch = file.match(/^(\d{4}-\d{2}-\d{2})/)
      if (!dateMatch) continue

      const content = readFileSync(filePath, "utf-8")
      const next = parseDEKNSummary(content)

      if (next) {
        const keywords = extractKeywords(next)
        recommendations.push({
          filePath,
          fileName: file,
          date: dateMatch[1],
          next,
          keywords,
        })

        log(`Loaded recommendation from ${file}: ${keywords.join(", ")}`)
      }
    }

    log(`Loaded ${recommendations.length} investigation recommendations`)
  } catch (err) {
    if (DEBUG) console.error(LOG_PREFIX, "Failed to load investigation recommendations:", err)
  }

  return recommendations
}

/**
 * Phase 2: Detect if command represents an architectural decision.
 * Returns extracted keywords if it's a decision, null otherwise.
 */
function detectArchitecturalDecision(command: string): string[] | null {
  if (!command) return null

  const lowerCommand = command.toLowerCase()

  // Git commit messages often contain architectural choices
  if (command.includes("git commit")) {
    // Extract commit message after -m flag
    const messageMatch = command.match(/-m\s+["']([^"']+)["']/i)
    if (messageMatch) {
      return extractKeywords(messageMatch[1])
    }
  }

  // File edits to architecture files (plist, Procfile, etc.)
  if (
    command.includes(".plist") ||
    command.includes("Procfile") ||
    command.includes("launchd") ||
    command.includes("overmind")
  ) {
    return extractKeywords(command)
  }

  // bd create for architectural issues
  if (command.includes("bd create") && (command.includes("--type") || command.includes("-t"))) {
    return extractKeywords(command)
  }

  return null
}

/**
 * Phase 2: Check if decision keywords contradict recommendation keywords.
 * Returns contradicting recommendation if found, null otherwise.
 */
function findContradiction(
  decisionKeywords: string[],
  recommendations: InvestigationRecommendation[]
): InvestigationRecommendation | null {
  if (!decisionKeywords.length) return null

  // Look for recommendations with different keywords in same domain
  // Example: decision has "launchd", recommendation has "overmind"
  const architectureGroups = {
    process_supervision: ["launchd", "overmind", "systemd", "supervisor", "daemon"],
    containerization: ["docker", "kubernetes"],
  }

  for (const group of Object.values(architectureGroups)) {
    const decisionInGroup = decisionKeywords.filter((k) => group.includes(k))
    if (!decisionInGroup.length) continue

    // Find recommendations with DIFFERENT keywords in same group
    for (const rec of recommendations) {
      const recInGroup = rec.keywords.filter((k) => group.includes(k))
      if (!recInGroup.length) continue

      // Check for contradiction: different keywords in same domain
      const hasContradiction = recInGroup.some((rk) => !decisionKeywords.includes(rk))

      if (hasContradiction) {
        log(
          `Potential circular pattern: Decision uses ${decisionInGroup.join(", ")} but ${rec.fileName} recommended ${recInGroup.join(", ")}`
        )
        return rec
      }
    }
  }

  return null
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
 * Dylan pattern tracking state for Phase 3.5.
 */
interface DylanPatternState {
  priorityUncertaintyCount: number // Count of "what's next?" type questions
  compensationKeywords: string[] // Keywords from Dylan's provided context
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
  // Phase 3.5: Dylan pattern detection
  dylan: DylanPatternState
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
 * Phase 3: Stream metric to coach session for investigation.
 * Coach receives metric + context and decides whether to intervene.
 */
async function streamToCoach(
  client: any,
  sessionId: string,
  metric: CoachingMetric,
  context: { recentCommands?: string[]; recommendation?: string }
): Promise<void> {
  if (!COACH_SESSION_ID) {
    return // Coach streaming disabled
  }

  // Avoid infinite loop - don't stream if this IS the coach session
  if (sessionId === COACH_SESSION_ID) {
    log("Skipping coach stream - current session is coach")
    return
  }

  try {
    // Format message for coach investigation
    const message = formatMetricForCoach(metric, context)

    // Stream to coach session asynchronously
    await client.session.promptAsync({
      sessionID: COACH_SESSION_ID,
      parts: [
        {
          type: "text",
          text: message,
        },
      ],
    })

    log(`✓ Streamed ${metric.metric_type} metric to coach session ${COACH_SESSION_ID}`)
  } catch (err) {
    if (DEBUG) console.error(LOG_PREFIX, "Failed to stream to coach:", err)
  }
}

/**
 * Format coaching metric into readable message for coach investigation.
 */
function formatMetricForCoach(metric: CoachingMetric, context: any): string {
  const lines = [
    `## Orchestrator Pattern Detected`,
    ``,
    `**Metric:** ${metric.metric_type}`,
    `**Timestamp:** ${metric.timestamp}`,
    `**Session:** ${metric.session_id}`,
    `**Value:** ${metric.value}`,
    ``,
  ]

  // Add metric-specific details
  if (metric.metric_type === "behavioral_variation") {
    lines.push(
      `**Pattern:** ${metric.value} consecutive variations in ${metric.details.group} without strategic pause`,
      ``,
      `**Recent Commands:**`,
      ...metric.details.commands.map((cmd: string) => `- \`${cmd}\``),
      ``,
      `**Threshold:** ${metric.details.threshold} variations`,
      ``
    )
  } else if (metric.metric_type === "circular_pattern") {
    lines.push(
      `**Pattern:** Decision contradicts prior investigation recommendation`,
      ``,
      `**Decision Command:** \`${metric.details.decision_command}\``,
      `**Decision Keywords:** ${metric.details.decision_keywords.join(", ")}`,
      ``,
      `**Contradicts Investigation:** ${metric.details.contradicts_investigation}`,
      `**Recommendation:** ${metric.details.recommendation}`,
      `**Recommendation Keywords:** ${metric.details.recommendation_keywords.join(", ")}`,
      `**Recommendation Date:** ${metric.details.recommendation_date}`,
      ``
    )
  } else if (metric.metric_type === "dylan_signal_prefix") {
    lines.push(
      `**Pattern:** Dylan used explicit signal prefix: \`${metric.details.prefix}:\``,
      ``,
      `**Message:** "${metric.details.message.substring(0, 200)}${metric.details.message.length > 200 ? "..." : ""}"`,
      ``,
      `**Signal Meaning:**`,
      `- **frame-collapse**: Orchestrator dropped into worker mode (doing spawnable work)`,
      `- **compensation**: Dylan providing context system should have surfaced`,
      `- **focus**: Dylan redirecting to what actually matters`,
      `- **step-back**: Need perspective, pause current thread`,
      ``
    )
  } else if (metric.metric_type === "priority_uncertainty") {
    lines.push(
      `**Pattern:** Dylan asking "what's next?" type questions (${metric.value} times)`,
      ``,
      `**Recent Questions:**`,
      ...metric.details.recent_questions.map((q: string) => `- "${q.substring(0, 150)}..."`),
      ``,
      `**Threshold:** ${metric.details.threshold}+ occurrences indicates orchestrator not providing strategic guidance`,
      ``
    )
  } else if (metric.metric_type === "compensation_pattern") {
    lines.push(
      `**Pattern:** Dylan providing repeated context (${Math.round(metric.value * 100)}% keyword overlap)`,
      ``,
      `**Current Message:** "${metric.details.current_message.substring(0, 200)}..."`,
      ``,
      `**Overlapping Keywords:** ${metric.details.overlapping_keywords.join(", ")}`,
      ``,
      `**Indicates:** System failing to surface knowledge - orchestrator should have run \`kb context\` first`,
      ``
    )
  }

  // Add context if provided
  if (context.recentCommands) {
    lines.push(`**Recent Transcript Context:**`, ...context.recentCommands.map((cmd) => `- ${cmd}`), ``)
  }

  if (context.recommendation) {
    lines.push(`**Investigation Recommendation:** ${context.recommendation}`, ``)
  }

  lines.push(
    `---`,
    ``,
    `**Your Task:** Investigate whether this pattern is a real concern or false positive. Use Read tool to examine .kb/investigations/ for context. Provide observations if intervention needed.`
  )

  return lines.join("\n")
}

/**
 * Phase 3.5: Extract user messages from messages array.
 * Returns array of text content from user messages (role='user').
 */
function extractUserMessages(
  messages: Array<{ info: any; parts: any[] }>
): Array<{ text: string; messageId: string }> {
  const userMessages: Array<{ text: string; messageId: string }> = []

  for (const msg of messages) {
    if (msg.info.role !== "user") continue

    for (const part of msg.parts) {
      if (part.type !== "text" || part.ignored || part.synthetic) continue
      if (part.text && part.text.trim()) {
        userMessages.push({
          text: part.text.trim(),
          messageId: msg.info.id || "unknown",
        })
      }
    }
  }

  return userMessages
}

/**
 * Phase 3.5: Detect Dylan's explicit signal prefixes.
 * Returns prefix type if message starts with known prefix, null otherwise.
 */
function detectSignalPrefix(text: string): string | null {
  const lowerText = text.toLowerCase()

  const prefixes = [
    "frame-collapse:",
    "compensation:",
    "focus:",
    "step-back:",
  ]

  for (const prefix of prefixes) {
    if (lowerText.startsWith(prefix)) {
      return prefix.replace(":", "") // Return without colon for cleaner metric
    }
  }

  return null
}

/**
 * Phase 3.5: Detect priority uncertainty patterns.
 * Returns true if message contains phrases indicating Dylan doesn't know what to do next.
 */
function detectPriorityUncertainty(text: string): boolean {
  const lowerText = text.toLowerCase()

  const patterns = [
    "what's next",
    "what should we focus on",
    "what should i focus on",
    "where should we start",
    "what's the priority",
  ]

  return patterns.some((pattern) => lowerText.includes(pattern))
}

/**
 * Phase 3.5: Extract keywords from text for compensation pattern detection.
 * Simple keyword extraction (words >4 chars, not common stopwords).
 */
function extractKeywordsSimple(text: string): string[] {
  const stopwords = new Set([
    "this",
    "that",
    "with",
    "from",
    "have",
    "been",
    "were",
    "they",
    "what",
    "when",
    "where",
    "which",
    "while",
    "should",
    "could",
    "would",
    "there",
  ])

  const words = text.toLowerCase().match(/\b\w+\b/g) || []
  const keywords: string[] = []

  for (const word of words) {
    if (word.length > 4 && !stopwords.has(word)) {
      keywords.push(word)
    }
  }

  return keywords
}

/**
 * Phase 3.5: Detect compensation pattern (Dylan providing repeated context).
 * Returns keyword overlap ratio if significant (>0.3), null otherwise.
 */
function detectCompensation(
  newKeywords: string[],
  priorKeywords: string[]
): number | null {
  if (newKeywords.length === 0 || priorKeywords.length === 0) return null

  // Count overlapping keywords
  const overlap = newKeywords.filter((k) => priorKeywords.includes(k)).length
  const ratio = overlap / Math.max(newKeywords.length, priorKeywords.length)

  // Significant overlap if >30% keywords repeat
  return ratio > 0.3 ? ratio : null
}

/**
 * OpenCode plugin that tracks orchestrator behavioral patterns.
 */
export const CoachingPlugin: Plugin = async ({ directory, client }) => {
  log("Plugin initialized, directory:", directory)
  log("Coach session ID:", COACH_SESSION_ID || "not set (coach streaming disabled)")

  // Prune old metrics on startup
  pruneMetrics()

  // Phase 2: Load investigation recommendations for circular pattern detection
  const investigationRecommendations = loadInvestigationRecommendations(directory)
  log(`Loaded ${investigationRecommendations.length} investigation recommendations for circular detection`)

  // Worker session tracking (per-session detection, not plugin-level)
  // Plugin runs in server process, can't see ORCH_WORKER env from spawned agents
  const workerSessions = new Map<string, boolean>() // sessionID -> isWorker

  /**
   * Detect if a session is a worker by examining tool args.
   * Returns true if worker detected, false otherwise.
   * Caches result in workerSessions Map to avoid repeated checks.
   */
  function detectWorkerSession(sessionId: string, tool: string, args: any): boolean {
    // Check cache first
    const cached = workerSessions.get(sessionId)
    if (cached !== undefined) {
      return cached
    }

    let isWorker = false

    // Detection signal 1: bash tool with workdir in .orch/workspace/
    if (tool === "bash" && args?.workdir) {
      if (args.workdir.includes(".orch/workspace/")) {
        log(`Worker detected (bash workdir): session ${sessionId}, workdir: ${args.workdir}`)
        isWorker = true
      }
    }

    // Detection signal 2: read tool accessing SPAWN_CONTEXT.md
    if (tool === "read" && args?.filePath) {
      if (args.filePath.endsWith("SPAWN_CONTEXT.md")) {
        log(`Worker detected (SPAWN_CONTEXT.md read): session ${sessionId}, file: ${args.filePath}`)
        isWorker = true
      }
    }

    // Detection signal 3: any tool with filePath in .orch/workspace/
    if (args?.filePath && typeof args.filePath === "string") {
      if (args.filePath.includes(".orch/workspace/")) {
        log(`Worker detected (filePath in workspace): session ${sessionId}, file: ${args.filePath}`)
        isWorker = true
      }
    }

    // Cache the result
    workerSessions.set(sessionId, isWorker)

    if (isWorker) {
      log(`Session ${sessionId} marked as worker (will skip metrics)`)
    }

    return isWorker
  }

  // Session state (Map<sessionID, SessionState>)
  const sessions = new Map<string, SessionState>()

  // Tool call counter for periodic flush
  let toolCallCounter = 0

  return {
    /**
     * Phase 3.5: Track Dylan's behavioral patterns via message monitoring.
     * Detects: explicit prefixes, priority uncertainty, compensation patterns.
     */
    "experimental.chat.messages.transform": async (input: {}, output: { messages: Array<{ info: any; parts: any[] }> }) => {
      // Extract user messages
      const userMessages = extractUserMessages(output.messages)
      if (userMessages.length === 0) return

      // Get most recent user message
      const latestMessage = userMessages[userMessages.length - 1]
      const text = latestMessage.text

      // Infer session ID from messages (use first message's info if available)
      const sessionId = output.messages[0]?.info?.sessionID || "unknown"

      // Check if this is a worker session (may not have tool args yet, but check cache)
      const cachedWorkerStatus = workerSessions.get(sessionId)
      if (cachedWorkerStatus === true) {
        // Skip Dylan pattern detection for worker sessions
        return
      }

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
          dylan: {
            priorityUncertaintyCount: 0,
            compensationKeywords: [],
          },
        }
        sessions.set(sessionId, state)
        log("Created session state for Dylan patterns:", sessionId)
      }

      // Pattern 1: Explicit Signal Prefixes
      const prefix = detectSignalPrefix(text)
      if (prefix) {
        const metric = {
          timestamp: new Date().toISOString(),
          session_id: sessionId,
          metric_type: "dylan_signal_prefix",
          value: 1,
          details: {
            prefix,
            message: text.substring(0, 500), // First 500 chars
          },
        }

        writeMetric(metric)
        log(`⚠️ DYLAN SIGNAL PREFIX DETECTED: ${prefix}`)

        // Stream to coach
        streamToCoach(client, sessionId, metric, {})
      }

      // Pattern 2: Priority Uncertainty
      if (detectPriorityUncertainty(text)) {
        state.dylan.priorityUncertaintyCount++

        // Emit metric when threshold reached (2+ occurrences)
        if (state.dylan.priorityUncertaintyCount >= 2) {
          const recentQuestions = userMessages.slice(-5).map((m) => m.text)

          const metric = {
            timestamp: new Date().toISOString(),
            session_id: sessionId,
            metric_type: "priority_uncertainty",
            value: state.dylan.priorityUncertaintyCount,
            details: {
              recent_questions: recentQuestions,
              threshold: 2,
            },
          }

          writeMetric(metric)
          log(`⚠️ PRIORITY UNCERTAINTY DETECTED: ${state.dylan.priorityUncertaintyCount} occurrences`)

          // Stream to coach
          streamToCoach(client, sessionId, metric, {})

          // Reset counter after emitting
          state.dylan.priorityUncertaintyCount = 0
        }
      }

      // Pattern 3: Compensation Pattern (keyword overlap)
      const newKeywords = extractKeywordsSimple(text)
      if (newKeywords.length > 0) {
        const overlapRatio = detectCompensation(newKeywords, state.dylan.compensationKeywords)

        if (overlapRatio !== null) {
          const overlappingKeywords = newKeywords.filter((k) => state.dylan.compensationKeywords.includes(k))

          const metric = {
            timestamp: new Date().toISOString(),
            session_id: sessionId,
            metric_type: "compensation_pattern",
            value: overlapRatio,
            details: {
              current_message: text.substring(0, 500),
              overlapping_keywords: overlappingKeywords,
              overlap_ratio: overlapRatio,
            },
          }

          writeMetric(metric)
          log(
            `⚠️ COMPENSATION PATTERN DETECTED: ${Math.round(overlapRatio * 100)}% keyword overlap (${overlappingKeywords.length} keywords)`
          )

          // Stream to coach
          streamToCoach(client, sessionId, metric, {})
        }

        // Update compensation keywords (keep last 100)
        state.dylan.compensationKeywords.push(...newKeywords)
        if (state.dylan.compensationKeywords.length > 100) {
          state.dylan.compensationKeywords = state.dylan.compensationKeywords.slice(-100)
        }
      }
    },

    /**
     * Track all tool executions and update session state.
     */
    "tool.execute.after": async (input: any, output: any) => {
      const tool = input.tool?.toLowerCase()
      const sessionId = input.sessionID

      if (!tool || !sessionId) return

      // Check if this is a worker session (per-session detection)
      if (detectWorkerSession(sessionId, tool, input.args)) {
        // Skip all metrics tracking for worker sessions
        return
      }

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
          dylan: {
            priorityUncertaintyCount: 0,
            compensationKeywords: [],
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

              const metric = {
                timestamp: new Date().toISOString(),
                session_id: state.sessionId,
                metric_type: "behavioral_variation",
                value: state.variation.variationCount,
                details: {
                  group,
                  commands: recentHistory.map((h) => h.command.substring(0, 100)),
                  threshold: VARIATION_THRESHOLD,
                },
              }

              writeMetric(metric)

              log(
                `⚠️ BEHAVIORAL VARIATION DETECTED: ${state.variation.variationCount} variations in ${group} without strategic pause`
              )

              // Phase 3: Stream to coach session for investigation
              streamToCoach(client, sessionId, metric, {
                recentCommands: recentHistory.map((h) => h.command),
              })
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

        // Phase 2: Circular pattern detection
        const decisionKeywords = detectArchitecturalDecision(command)
        if (decisionKeywords && decisionKeywords.length > 0) {
          const contradiction = findContradiction(decisionKeywords, investigationRecommendations)

          if (contradiction) {
            const metric = {
              timestamp: new Date().toISOString(),
              session_id: state.sessionId,
              metric_type: "circular_pattern",
              value: 1,
              details: {
                decision_command: command.substring(0, 200),
                decision_keywords: decisionKeywords,
                contradicts_investigation: contradiction.fileName,
                recommendation: contradiction.next.substring(0, 200),
                recommendation_keywords: contradiction.keywords,
                recommendation_date: contradiction.date,
              },
            }

            writeMetric(metric)

            log(
              `⚠️ CIRCULAR PATTERN DETECTED: Command uses ${decisionKeywords.join(", ")} but ${contradiction.fileName} recommended ${contradiction.keywords.join(", ")}`
            )

            // Phase 3: Stream to coach session for investigation
            streamToCoach(client, sessionId, metric, {
              recommendation: contradiction.next,
            })
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
