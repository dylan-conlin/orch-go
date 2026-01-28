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
 * NOTE: Worker detection via session.metadata.role
 * 
 * OpenCode now reliably exposes session.metadata.role='worker' from the
 * x-opencode-env-ORCH_WORKER header sent by orch-go during spawn.
 * 
 * This is set in the session.created event BEFORE any tool calls occur,
 * eliminating the need for complex title-based or tool-path heuristics.
 * 
 * Plugin-level detection (checking process.env.ORCH_WORKER) doesn't work because
 * the plugin runs in the OpenCode server process, not in spawned agent processes.
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
 * Phase 3.5: Dylan pattern tracking state for Phase 3.5.
 */
interface DylanPatternState {
  priorityUncertaintyCount: number // Count of "what's next?" type questions
  compensationKeywords: string[] // Keywords from Dylan's provided context
  premiseSkippingCount: number // Count of "how to X" questions that skip premise validation
  premiseSkippingWarningInjected: boolean // Have we warned about premise-skipping yet?
  premiseSkippingStrongWarningInjected: boolean // Have we warned strongly (2nd+ occurrence)?
  recentQuestions: string[] // Last 5 user questions for pattern tracking
}

/**
 * Frame collapse tracking state for orchestrator code edit detection.
 * Frame collapse = orchestrator editing code files instead of delegating to workers.
 */
interface FrameCollapseState {
  codeEditCount: number // Cumulative code file edits
  lastCodeEditPath: string | null // Most recent code file edited
  warningInjected: boolean // Have we warned yet?
  strongWarningInjected: boolean // Have we strongly warned yet?
}

/**
 * Worker health state for worker-specific metric tracking.
 * Workers need different signals than orchestrators (context budget, tool failures, etc.)
 */
interface WorkerHealthState {
  sessionId: string
  sessionStartTime: number          // When session started (for time_in_phase)
  consecutiveToolFailures: number   // For tool_failure_rate
  estimatedTokensUsed: number       // For context_usage
  lastPhaseUpdate: number           // Timestamp of last "Phase:" comment
  lastCommitTime: number            // Timestamp of last git commit
  totalToolCalls: number            // For token estimation
  totalReadBytes: number            // For token estimation
  lastWarningType?: string          // Type of last health signal injected
  lastWarningValue?: number         // Value of last health signal injected
}


/**
 * Determine if a file path represents code (vs orchestration artifact).
 * Returns true if editing this file indicates frame collapse.
 */
function isCodeFile(filePath: string): boolean {
  if (!filePath) return false

  const lowerPath = filePath.toLowerCase()

  // Orchestration directories - ALLOWED for orchestrators
  const orchestrationPaths = [
    "/.orch/",
    "/.kb/",
    "/.beads/",
    "/skills/",
    "/plugins/",
    "claude.md",
    "skill.md",
    "readme.md",
    "spawn_context.md",
    "synthesis.md",
    "session_handoff.md",
    "workspace.md",
  ]

  for (const orchPath of orchestrationPaths) {
    if (lowerPath.includes(orchPath)) {
      return false // Orchestration artifact, not code
    }
  }

  // Code file extensions - frame collapse indicators
  const codeExtensions = [
    ".go",
    ".ts",
    ".tsx",
    ".js",
    ".jsx",
    ".css",
    ".scss",
    ".less",
    ".sass",
    ".py",
    ".rb",
    ".java",
    ".rs",
    ".c",
    ".cpp",
    ".h",
    ".html",
    ".vue",
    ".svelte",
  ]

  for (const ext of codeExtensions) {
    if (lowerPath.endsWith(ext)) {
      return true // Code file
    }
  }

  return false // Unknown file type, not flagged
}

/**
 * Phase 2: Check if a file path is in the orchestration allowlist.
 * Orchestrators can read these files without filtering.
 * Returns true if file should be accessible to orchestrators.
 */
function isOrchestrationFile(filePath: string): boolean {
  if (!filePath) return false

  const normalizedPath = filePath.toLowerCase()

  // Exact filenames (case-insensitive)
  const allowedFilenames = [
    "claude.md",
    "agents.md",
    "synthesis.md",
    "spawn_context.md",
  ]

  for (const filename of allowedFilenames) {
    if (normalizedPath.endsWith(filename)) {
      return true
    }
  }

  // Directory patterns
  const allowedPaths = [
    "/.kb/",
    "/.orch/",
  ]

  for (const path of allowedPaths) {
    if (normalizedPath.includes(path) && normalizedPath.endsWith(".md")) {
      return true
    }
  }

  return false
}

/**
 * Phase 2: Filter tool output for orchestrator sessions.
 * Implements information hiding by truncating non-orchestration file reads
 * and bash outputs, encouraging delegation to workers instead of direct investigation.
 */
function filterOrchestratorOutput(tool: string, args: any, output: any): void {
  // Only filter successful outputs (don't modify errors)
  if (!output || output.error || output.isError) {
    return
  }

  // Filter read tool outputs
  if (tool === "read" && output.text) {
    const filePath = args?.filePath || args?.file_path || ""
    
    // Skip filtering for orchestration files
    if (isOrchestrationFile(filePath)) {
      return
    }

    // Truncate to first 20 lines
    const lines = output.text.split("\n")
    if (lines.length > 20) {
      const truncatedLines = lines.slice(0, 20)
      const warningMessage = `\n\n[... ${lines.length - 20} lines hidden ...]\n\n⚠️ Full file hidden - orchestrator should delegate to worker.\nOrchestrators operate in meta-action space (spawn, monitor, query).\nFor file investigation, use: orch spawn investigation "analyze ${filePath}"`
      
      output.text = truncatedLines.join("\n") + warningMessage
      log(`Filtered read output for orchestrator: ${filePath} (${lines.length} → 20 lines)`)
    }
  }

  // Filter bash outputs
  if (tool === "bash" && output.output) {
    const outputLength = output.output.length
    const TRUNCATE_THRESHOLD = 1000 // Truncate if >1000 chars

    if (outputLength > TRUNCATE_THRESHOLD) {
      const truncated = output.output.substring(0, TRUNCATE_THRESHOLD)
      const hiddenChars = outputLength - TRUNCATE_THRESHOLD
      const warningMessage = `\n\n[... ${hiddenChars} characters hidden ...]\n\n⚠️ Command completed. Full output hidden.\nOrchestrators should delegate detailed investigations to workers.\nFor command analysis, use: orch spawn investigation "analyze command output"`
      
      output.output = truncated + warningMessage
      log(`Filtered bash output for orchestrator: ${outputLength} → ${TRUNCATE_THRESHOLD} chars`)
    }
  }
}

/**
 * Session state tracker.
 */
interface SessionState {
  sessionId: string
  sessionStartTime: number // Track session start for warm-up period
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
  // Frame collapse detection: orchestrator editing code files
  frameCollapse: FrameCollapseState
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
 * Now also injects coaching messages into the session when patterns detected.
 */
async function flushMetrics(state: SessionState, client: any): Promise<void> {
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
  let shouldInjectActionCoaching = false
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

    // Frame 1: Inject coaching when action ratio is low
    // FIX: Add warm-up period (5 min OR 15 tool calls) before alerting
    const sessionAgeMins = (Date.now() - state.sessionStartTime) / 60000
    const totalToolCalls = state.toolWindow.length + state.spawns + state.reads + state.actions
    const hasWarmupPassed = sessionAgeMins > 5 || totalToolCalls > 15
    
    if (actionRatio < 0.5 && state.reads >= 6 && hasWarmupPassed) {
      shouldInjectActionCoaching = true
    }
  }

  // Tool repetition sequence (analysis paralysis)
  const sequence = detectSequence(state.toolWindow)
  let shouldInjectAnalysisParalysis = false
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

    shouldInjectAnalysisParalysis = true
  }

  // Inject coaching messages when thresholds exceeded
  if (shouldInjectActionCoaching) {
    await injectCoachingMessage(client, state.sessionId, "action_ratio", {
      reads: state.reads,
      actions: state.actions,
    })
  }

  if (shouldInjectAnalysisParalysis) {
    await injectCoachingMessage(client, state.sessionId, "analysis_paralysis", {
      sequence,
      toolWindow: state.toolWindow.slice(-10),
    })
  }

  state.lastFlush = Date.now()
  log("Flushed metrics for session:", state.sessionId)
}

/**
 * Frame 1: Inject coaching message directly into orchestrator session.
 * Uses noReply:true pattern from agentlog-inject.ts to avoid blocking.
 */
async function injectCoachingMessage(
  client: any,
  sessionId: string,
  patternType: "action_ratio" | "analysis_paralysis" | "frame_collapse" | "frame_collapse_strong" | "premise_skipping" | "premise_skipping_strong",
  details: any
): Promise<void> {
  try {
    let message = ""

    if (patternType === "action_ratio") {
      message = `## 📊 Orchestrator Coaching

You've done **${details.reads} reads** with only **${details.actions} actions** (ratio: ${(details.actions / details.reads).toFixed(2)}).

**Observation:** Low action-to-read ratio suggests analysis paralysis or investigation without delegation.

**Consider:** Spawning an agent instead of investigating yourself, or taking action on what you've learned.`
    } else if (patternType === "analysis_paralysis") {
      message = `## 📊 Orchestrator Coaching

Tool repetition sequence detected: **${details.sequence} consecutive uses** of the same tool.

**Observation:** Repeated tool use without progress suggests stuck pattern.

**Consider:** Stepping back to reassess approach, or spawning an agent to handle the investigation.`
    } else if (patternType === "frame_collapse") {
      message = `## ⚠️ Frame Collapse Warning

You've edited a code file: \`${details.filePath}\`

**Observation:** Orchestrators delegate implementation to workers. Editing code files directly indicates potential frame collapse.

**Consider:**
1. Is this work you should have spawned to a worker?
2. If an agent already failed, try different parameters (skill, model, --mcp)`
    } else if (patternType === "frame_collapse_strong") {
      message = `## 🚨 Frame Collapse - Multiple Code Edits

You've now made **${details.count} code file edits** in this session.

**Last edited:** \`${details.filePath}\`

**This is a clear frame collapse pattern.** Orchestrators should delegate, not implement.

**Required Action:**
1. **STOP** editing code files
2. Spawn a worker with \`orch spawn feature-impl "your task" --issue BEADS-ID\`
3. If struggling with spawn strategy, consider \`--mcp playwright\` for UI work

**Why this matters:** Frame collapse wastes orchestrator capacity and bypasses quality gates (worker verification, beads tracking).`
    } else if (patternType === "premise_skipping") {
      message = `## 💭 Premise Validation Reminder

Your question: "${details.question.substring(0, 150)}${details.question.length > 150 ? "..." : ""}"

**Observation:** This question assumes a strategic direction ("${details.verb}") without first validating the premise.

**Consider:** Before asking "How do we ${details.verb}?", ask "**Should** we ${details.verb}?"

**Why this matters:** Strategic questions benefit from premise validation first. The Dec 2025 "evolve skills" epic was created from an unvalidated premise and had to be paused when architect review found the premise was wrong. Validating direction before designing solutions avoids wasted work.

**Suggested next step:** Spawn an investigation or architect session to validate the premise first.`
    } else if (patternType === "premise_skipping_strong") {
      message = `## ⚠️ Repeated Premise-Skipping Pattern

You've now asked **${details.count} "how to" questions** without premise validation.

**Recent questions:**
${details.recentQuestions.slice(-3).map((q: string) => `- "${q.substring(0, 100)}..."`).join("\n")}

**Pattern:** Jumping to implementation ("how to ${details.verb}") before validating strategic direction ("should we?").

**Required Action:**
1. **PAUSE** - Before proceeding with implementation questions
2. **VALIDATE** - Ask "Should we do this?" or spawn architect/investigation
3. **THEN PROCEED** - After premise is validated, ask "how to" questions

**Why this matters:** From CLAUDE.md constraint: "Ask 'should we' before 'how do we' for strategic direction changes." The orch-go-erdw epic failure demonstrates the cost of skipping this step.`
    }

    // Inject using noReply:true pattern
    if (client?.session?.prompt) {
      await client.session.prompt({
        path: { id: sessionId },
        body: {
          noReply: true,
          parts: [
            {
              type: "text",
              text: message,
            },
          ],
        },
      })
      log(`✅ Injected ${patternType} coaching into session ${sessionId}`)
    } else {
      log(`⚠️ Cannot inject coaching: client.session.prompt unavailable`)
    }
  } catch (err) {
    if (DEBUG) console.error(LOG_PREFIX, "Failed to inject coaching:", err)
  }
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
      path: { id: COACH_SESSION_ID },
      body: {
        parts: [
          {
            type: "text",
            text: message,
          },
        ],
      },
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
 * Detect premise-skipping question patterns.
 * Returns {matched: true, verb, extracted} if question assumes strategic direction without premise validation.
 * Example: "How do we migrate to X?" → suggests asking "Should we migrate to X?" first
 */
function detectPremiseSkipping(text: string): { matched: boolean; verb?: string; extracted?: string } | null {
  const lowerText = text.toLowerCase()

  // Red-flag strategic verbs that indicate direction changes
  const strategicVerbs = [
    "migrate",
    "evolve",
    "fix",
    "centralize",
    "transition",
    "implement",
    "solve",
    "change",
    "shift",
    "move",
    "refactor",
    "rewrite",
    "rebuild",
  ]

  // Implementation-focused phrasing patterns
  // Match "how to X", "how do we X", "how should we X", "how can we X"
  const howPatterns = [
    /how\s+(?:to|do\s+we|should\s+we|can\s+we)\s+(\w+)/gi,
  ]

  for (const pattern of howPatterns) {
    const matches = lowerText.matchAll(pattern)
    for (const match of matches) {
      const verb = match[1]
      
      // Check if verb is a strategic direction verb
      if (strategicVerbs.includes(verb)) {
        // Additional check: Skip if this is a tactical "I" question (personal pronoun)
        // "How do I migrate this database?" (tactical) vs "How do we migrate to microservices?" (strategic)
        if (lowerText.includes(" i ") || lowerText.includes("my ")) {
          continue // Skip tactical questions
        }

        return {
          matched: true,
          verb,
          extracted: match[0],
        }
      }
    }
  }

  return null
}

/**
 * Phase 3.5: Extract keywords from text for compensation pattern detection.
 * Simple keyword extraction (words >4 chars, not common stopwords).
 * FIX: Added domain stopwords to filter out file system noise (drwxr, usernames, etc.)
 */
function extractKeywordsSimple(text: string): string[] {
  const stopwords = new Set([
    // Common English stopwords
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
    // Domain stopwords: file system noise
    "drwxr",
    "rwxr",
    "total",
    "bytes",
    "permissions",
    // Common usernames/groups
    "staff",
    "wheel",
    "admin",
    "root",
    "dylanconlin",
    // Path fragments
    "users",
    "documents",
    "personal",
    "library",
    // Date fragments (month abbreviations)
    "jan",
    "feb",
    "mar",
    "apr",
    "may",
    "jun",
    "jul",
    "aug",
    "sep",
    "oct",
    "nov",
    "dec",
    // Common CLI output tokens
    "lines",
    "modified",
    "created",
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
 * Estimate token usage for a worker session.
 * Rough approximation: 1 token ≈ 4 chars average
 * Based on: tool calls (~500 tokens each) + read bytes (~1 token / 4 chars)
 * This is intentionally approximate - see architect investigation for rationale.
 */
function estimateWorkerTokenUsage(state: WorkerHealthState): number {
  const TOKENS_PER_TOOL_CALL = 500   // Average tool call overhead
  const CHARS_PER_TOKEN = 4          // Average chars per token

  const toolCallTokens = state.totalToolCalls * TOKENS_PER_TOOL_CALL
  const readTokens = Math.round(state.totalReadBytes / CHARS_PER_TOKEN)

  return toolCallTokens + readTokens
}

/**
 * Inject a health signal (warning) into the agent's context using noReply: true.
 * This provides immediate feedback ("Pain") to the agent when metrics cross thresholds.
 *
 * @param client OpenCode client
 * @param state Worker health state
 * @param metricType Type of metric triggering the signal
 * @param value Current value of the metric
 */
async function injectHealthSignal(
  client: any,
  state: WorkerHealthState,
  metricType: string,
  value: number
): Promise<void> {
  if (!client?.session?.prompt) {
    log("Cannot inject health signal: client.session.prompt unavailable")
    return
  }

  // Avoid duplicate warnings for same type/value
  if (state.lastWarningType === metricType && state.lastWarningValue === value) {
    return
  }

  let prompt = ""
  switch (metricType) {
    case "tool_failure_rate":
      if (value >= 5) {
        prompt = `CRITICAL: You have had ${value} consecutive tool failures. Please STOP and analyze why your tool calls are failing. Check parameters, file paths, and environment state before continuing.`
      } else if (value >= 3) {
        prompt = `Warning: You have had ${value} consecutive tool failures. Consider verifying your assumptions or searching for more context.`
      }
      break
    case "context_usage":
      prompt = `Warning: Your context usage is at ${value}%. You are approaching the token limit. Consider summarizing your findings and focusing on the most relevant files to avoid context exhaustion.`
      break
    case "time_in_phase":
      prompt = `Notice: You have been in the current phase for ${value} minutes. If you are stuck or experiencing analysis paralysis, consider taking a strategic pause or breaking the task into smaller steps.`
      break
    case "commit_gap":
      prompt = `Notice: It has been ${value} minutes since your last commit. If you have made stable changes, consider committing them now to provide a safety net.`
      break
  }

  if (prompt) {
    try {
      await client.session.prompt({
        path: { id: state.sessionId },
        body: {
          noReply: true,
          parts: [
            {
              type: "text",
              text: prompt,
            },
          ],
        },
      })
      log(`Injected health signal for ${metricType}: ${value}`)

      // Update state to prevent repeat
      state.lastWarningType = metricType
      state.lastWarningValue = value
    } catch (err) {
      log(`Failed to inject health signal for ${metricType}:`, err)
    }
  }
}

/**
 * Track worker health metrics and record to coaching-metrics.jsonl.
 * This is called for worker sessions instead of orchestrator metrics.
 */
async function trackWorkerHealth(
  client: any,
  state: WorkerHealthState,
  tool: string,
  success: boolean,
  args: any,
  output: any
): Promise<void> {
  const now = Date.now()
  const timestamp = new Date().toISOString()

  // Update tool call count (for token estimation)
  state.totalToolCalls++

  // Track read bytes for token estimation
  if (tool === "read" && output?.text) {
    state.totalReadBytes += output.text.length
  }

  // 1. tool_failure_rate: Track consecutive failures
  if (!success) {
    state.consecutiveToolFailures++
    if (state.consecutiveToolFailures >= 3) {
      writeMetric({
        timestamp,
        session_id: state.sessionId,
        metric_type: "tool_failure_rate",
        value: state.consecutiveToolFailures,
        details: {
          last_tool: tool,
          consecutive_failures: state.consecutiveToolFailures,
        },
      })
      log(`Worker metric: tool_failure_rate = ${state.consecutiveToolFailures}`)

      // Inject "Pain" signal into agent context
      await injectHealthSignal(client, state, "tool_failure_rate", state.consecutiveToolFailures)
    }
  } else {
    // Reset on success
    state.consecutiveToolFailures = 0
  }

  // 2. context_usage: Estimate tokens and emit metric periodically
  state.estimatedTokensUsed = estimateWorkerTokenUsage(state)
  // Emit every 50 tool calls or when over threshold
  const CONTEXT_WARNING_THRESHOLD = 80000 // ~80% of 100k typical limit
  if (state.totalToolCalls % 50 === 0 || state.estimatedTokensUsed > CONTEXT_WARNING_THRESHOLD) {
    const percentUsed = Math.round((state.estimatedTokensUsed / 100000) * 100)
    writeMetric({
      timestamp,
      session_id: state.sessionId,
      metric_type: "context_usage",
      value: percentUsed,
      details: {
        estimated_tokens: state.estimatedTokensUsed,
        total_tool_calls: state.totalToolCalls,
        total_read_bytes: state.totalReadBytes,
        threshold_percent: 80,
      },
    })
    log(`Worker metric: context_usage = ${percentUsed}% (~${Math.round(state.estimatedTokensUsed / 1000)}k tokens)`)

    // Inject signal if over threshold
    if (percentUsed >= 80) {
      await injectHealthSignal(client, state, "context_usage", percentUsed)
    }
  }

  // 3. time_in_phase: Track time since last phase change
  const minutesInPhase = Math.round((now - state.lastPhaseUpdate) / 60000)
  // Emit every 5 minutes or when over threshold
  const TIME_IN_PHASE_WARNING_MINUTES = 15
  if (state.totalToolCalls % 30 === 0 && minutesInPhase > 5) {
    writeMetric({
      timestamp,
      session_id: state.sessionId,
      metric_type: "time_in_phase",
      value: minutesInPhase,
      details: {
        minutes_in_phase: minutesInPhase,
        threshold_minutes: TIME_IN_PHASE_WARNING_MINUTES,
        session_start: new Date(state.sessionStartTime).toISOString(),
        last_phase_update: new Date(state.lastPhaseUpdate).toISOString(),
      },
    })
    log(`Worker metric: time_in_phase = ${minutesInPhase} minutes`)

    // Inject signal if over threshold
    if (minutesInPhase >= TIME_IN_PHASE_WARNING_MINUTES) {
      await injectHealthSignal(client, state, "time_in_phase", minutesInPhase)
    }
  }

  // 4. commit_gap: Track time since last commit (detect via git commands in bash)
  if (tool === "bash" && args?.command) {
    const command = args.command as string
    // Detect successful git commit
    if (command.includes("git commit") && success) {
      state.lastCommitTime = now
      log(`Worker: git commit detected, updating lastCommitTime`)
    }
  }

  // Emit commit_gap metric periodically if there have been changes
  const minutesSinceCommit = state.lastCommitTime > 0
    ? Math.round((now - state.lastCommitTime) / 60000)
    : Math.round((now - state.sessionStartTime) / 60000)
  const COMMIT_GAP_WARNING_MINUTES = 30

  if (state.totalToolCalls % 30 === 0 && minutesSinceCommit > 10) {
    writeMetric({
      timestamp,
      session_id: state.sessionId,
      metric_type: "commit_gap",
      value: minutesSinceCommit,
      details: {
        minutes_since_commit: minutesSinceCommit,
        threshold_minutes: COMMIT_GAP_WARNING_MINUTES,
        last_commit_time: state.lastCommitTime > 0
          ? new Date(state.lastCommitTime).toISOString()
          : "never",
        session_start: new Date(state.sessionStartTime).toISOString(),
      },
    })
    log(`Worker metric: commit_gap = ${minutesSinceCommit} minutes`)

    // Inject signal if over threshold
    if (minutesSinceCommit >= COMMIT_GAP_WARNING_MINUTES) {
      await injectHealthSignal(client, state, "commit_gap", minutesSinceCommit)
    }
  }
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

  // Worker health state tracking (for worker-specific metrics)
  const workerHealthStates = new Map<string, WorkerHealthState>()

  // Session state (Map<sessionID, SessionState>)
  const sessions = new Map<string, SessionState>()

  // Tool call counter for periodic flush
  let toolCallCounter = 0

  // Store args from tool.execute.before for use in tool.execute.after
  // Map<callID, args> - args only available in before hook via output.args
  const pendingArgs = new Map<string, any>()

  return {
    /**
     * Store args from tool calls for retrieval in after hook.
     * Args are available in output.args in before hook, but not in after hook.
     */
    "tool.execute.before": async (input: any, output: any) => {
      if (input.callID && output.args) {
        pendingArgs.set(input.callID, output.args)
      }
    },
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

      // Check if this is a worker session (already detected in session.created)
      const isWorker = workerSessions.get(sessionId) === true
      
      if (isWorker) {
        // Skip Dylan pattern detection for worker sessions
        return
      }

      // Get or create session state
      let state = sessions.get(sessionId)
      if (!state) {
        state = {
          sessionId,
          sessionStartTime: Date.now(),
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
            premiseSkippingCount: 0,
            premiseSkippingWarningInjected: false,
            premiseSkippingStrongWarningInjected: false,
            recentQuestions: [],
          },
          frameCollapse: {
            codeEditCount: 0,
            lastCodeEditPath: null,
            warningInjected: false,
            strongWarningInjected: false,
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

      // Pattern 4: Premise-Skipping Detection
      // Track recent questions for pattern analysis
      state.dylan.recentQuestions.push(text)
      if (state.dylan.recentQuestions.length > 5) {
        state.dylan.recentQuestions.shift() // Keep last 5
      }

      const premiseSkipResult = detectPremiseSkipping(text)
      if (premiseSkipResult?.matched) {
        state.dylan.premiseSkippingCount++

        const metric = {
          timestamp: new Date().toISOString(),
          session_id: sessionId,
          metric_type: "premise_skipping",
          value: state.dylan.premiseSkippingCount,
          details: {
            question: text.substring(0, 500),
            verb: premiseSkipResult.verb,
            extracted_pattern: premiseSkipResult.extracted,
            recent_questions: state.dylan.recentQuestions,
          },
        }

        writeMetric(metric)
        log(`⚠️ PREMISE-SKIPPING DETECTED: "${premiseSkipResult.extracted}" (verb: ${premiseSkipResult.verb})`)

        // Graduated coaching: first detection → suggestion, 2nd+ → stronger reminder
        if (!state.dylan.premiseSkippingWarningInjected) {
          // First detection - gentle suggestion
          await injectCoachingMessage(client, sessionId, "premise_skipping", {
            question: text.substring(0, 200),
            verb: premiseSkipResult.verb,
            extracted: premiseSkipResult.extracted,
          })
          state.dylan.premiseSkippingWarningInjected = true
        } else if (
          state.dylan.premiseSkippingCount >= 2 &&
          !state.dylan.premiseSkippingStrongWarningInjected
        ) {
          // 2nd+ detection - stronger reminder
          await injectCoachingMessage(client, sessionId, "premise_skipping_strong", {
            question: text.substring(0, 200),
            verb: premiseSkipResult.verb,
            count: state.dylan.premiseSkippingCount,
            recentQuestions: state.dylan.recentQuestions,
          })
          state.dylan.premiseSkippingStrongWarningInjected = true
        }

        // Stream to coach
        streamToCoach(client, sessionId, metric, {
          recentCommands: state.dylan.recentQuestions,
        })
      }
    },

    /**
     * Track all tool executions and update session state.
     */
    "tool.execute.after": async (input: any, output: any) => {
      const tool = input.tool?.toLowerCase()
      const sessionId = input.sessionID

      if (!tool || !sessionId) return

      // Retrieve stored args from before hook
      const args = input.callID ? pendingArgs.get(input.callID) : undefined

      // Clean up stored args to prevent memory leak
      if (input.callID) {
        pendingArgs.delete(input.callID)
      }

      // Check if this is a worker session (already detected in session.created)
      const isWorker = workerSessions.get(sessionId) === true
      
      if (isWorker) {
        // Track worker-specific health metrics instead of orchestrator metrics
        let workerState = workerHealthStates.get(sessionId)
        if (!workerState) {
          const now = Date.now()
          workerState = {
            sessionId,
            sessionStartTime: now,
            consecutiveToolFailures: 0,
            estimatedTokensUsed: 0,
            lastPhaseUpdate: now,        // Assume phase started at session start
            lastCommitTime: 0,           // 0 means no commit yet
            totalToolCalls: 0,
            totalReadBytes: 0,
          }
          workerHealthStates.set(sessionId, workerState)
          log(`Created worker health state for session ${sessionId}`)
        }

        // Determine if tool succeeded (check for error in output)
        const success = !output?.error && !output?.isError

        // Track worker health metrics
        await trackWorkerHealth(client, workerState, tool, success, args, output)

        // Skip orchestrator metrics for workers
        return
      }

      // Phase 2: Information hiding - filter outputs for orchestrator sessions
      // This reduces temptation to "dive in" by hiding details that invite investigation.
      // Orchestrators should delegate detailed work to workers instead.
      filterOrchestratorOutput(tool, args, output)

      // Get or create session state
      let state = sessions.get(sessionId)
      if (!state) {
        state = {
          sessionId,
          sessionStartTime: Date.now(),
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
            premiseSkippingCount: 0,
            premiseSkippingWarningInjected: false,
            premiseSkippingStrongWarningInjected: false,
            recentQuestions: [],
          },
          frameCollapse: {
            codeEditCount: 0,
            lastCodeEditPath: null,
            warningInjected: false,
            strongWarningInjected: false,
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

      // Frame Collapse Detection: Track edit/write on code files
      // Only triggers for orchestrator sessions (NOT worker sessions - already filtered above)
      if (tool === "edit" || tool === "write") {
        const filePath = args?.file_path || args?.filePath || ""

        if (isCodeFile(filePath)) {
          state.frameCollapse.codeEditCount++
          state.frameCollapse.lastCodeEditPath = filePath

          log(`⚠️ FRAME COLLAPSE: Code file edit detected: ${filePath} (count: ${state.frameCollapse.codeEditCount})`)

          // Write metric for tracking
          const metric = {
            timestamp: new Date().toISOString(),
            session_id: state.sessionId,
            metric_type: "frame_collapse",
            value: state.frameCollapse.codeEditCount,
            details: {
              filePath,
              totalEdits: state.frameCollapse.codeEditCount,
            },
          }
          writeMetric(metric)

          // Tiered injection: first warning, then strong warning at 3+
          if (state.frameCollapse.codeEditCount === 1 && !state.frameCollapse.warningInjected) {
            // First code edit - warning
            injectCoachingMessage(client, state.sessionId, "frame_collapse", { filePath })
            state.frameCollapse.warningInjected = true
          } else if (
            state.frameCollapse.codeEditCount >= 3 &&
            !state.frameCollapse.strongWarningInjected
          ) {
            // 3+ code edits - strong warning
            injectCoachingMessage(client, state.sessionId, "frame_collapse_strong", {
              filePath,
              count: state.frameCollapse.codeEditCount,
            })
            state.frameCollapse.strongWarningInjected = true
          }

          // Stream to coach session for investigation
          streamToCoach(client, sessionId, metric, {
            recentCommands: state.toolWindow.slice(-5),
          })
        }
      }

      // Special handling for bash commands
      if (tool === "bash") {
        const command = args?.command || ""

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
      // FIX: Only flush current session, not all sessions (prevents 4x duplicates)
      toolCallCounter++
      if (toolCallCounter >= 10) {
        // Flush current session only
        if (state.spawns > 0 || state.reads > 0 || state.actions > 0) {
          flushMetrics(state, client)
        }
        toolCallCounter = 0
      }

      // Also flush if session has been active for 5+ minutes since last flush
      const now = Date.now()
      if (now - state.lastFlush > 5 * 60 * 1000) {
        if (state.spawns > 0 || state.reads > 0 || state.actions > 0) {
          flushMetrics(state, client)
        }
      }
    },

    /**
     * Early worker detection via session.created event.
     * OpenCode now reliably exposes metadata.role='worker' from the x-opencode-env-ORCH_WORKER header.
     * This is set at session creation time BEFORE any tool calls occur.
     */
    event: async ({ event }) => {
      // Log ALL events for debugging
      log(`Event received: type=${event.type}`)
      
      // Only handle session.created events
      if (event.type !== "session.created") {
        return
      }

      // Extract session info from event properties
      const info = (event as any).properties?.info
      if (!info) {
        log("Event: No info in session.created event properties, skipping")
        return
      }

      const sessionId = info.id
      const sessionTitle = info.title || ""
      const sessionMetadata = info.metadata || {}

      if (!sessionId) {
        log("Event: No sessionID in event properties, skipping")
        return
      }

      // Worker detection: session.metadata.role
      // OpenCode sets this to "worker" when x-opencode-env-ORCH_WORKER header is present
      const isWorker = sessionMetadata.role === "worker"
      
      if (isWorker) {
        workerSessions.set(sessionId, true)
        log(`Worker detected (metadata.role): ${sessionId}, title: ${sessionTitle}`)
        // Also log to stderr for visibility without DEBUG flag
        console.error(`[coaching] Worker detected: ${sessionId} title="${sessionTitle}"`)
      } else {
        log(`Orchestrator session (will receive coaching): ${sessionId}, title: ${sessionTitle}`)
      }
    },
  }
}
