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
 * Architecture:
 * - Track tool usage patterns via tool.execute.after
 * - Calculate behavioral metrics (ratios, sequences, variations)
 * - Write to ~/.orch/coaching-metrics.jsonl
 * - Exposed via /api/coaching endpoint
 * - Displayed in dashboard
 *
 * Reference: docs/designs/2026-01-10-orchestrator-coaching-plugin.md
 */

import type { Plugin } from "@opencode-ai/plugin"
import { exec } from "child_process"

import type { SessionState, WorkerHealthState } from "./coaching-types"
import {
  log,
  DEBUG,
  LOG_PREFIX,
  STRATEGIC_PAUSE_MS,
  VARIATION_THRESHOLD,
  AccretionWarningThreshold,
  pruneMetrics,
  writeMetric,
  createSessionState,
} from "./coaching-types"
import { classifyBashCommand, isSpawn } from "./coaching-classification"
import { loadInvestigationRecommendations, detectArchitecturalDecision, findContradiction } from "./coaching-investigation"
import { injectCoachingMessage, streamToCoach } from "./coaching-injection"
import { getFileLineCount, isCodeFile, extractUserMessages, detectPremiseSkipping } from "./coaching-detection"
import { trackWorkerHealth } from "./coaching-worker-health"

/**
 * Promisified exec for async shell commands.
 */
function execAsync(command: string): Promise<string> {
  return new Promise((resolve, reject) => {
    exec(command, { timeout: 10000 }, (error, stdout) => {
      if (error) {
        reject(error)
        return
      }
      resolve(stdout)
    })
  })
}

/**
 * Simple promise-based delay.
 */
function delay(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}

/**
 * Format orch stats JSON into compact health summary for session injection.
 */
function formatHealthSummary(stats: any): string {
  const s = stats.summary
  const lines = [
    `## Orch Health (24h)`,
    ``,
    `**Throughput:** ${s.total_spawns} spawned, ${s.task_completions} complete (${Math.round(s.task_completion_rate)}%), ${s.total_abandonments} abandoned | avg ${Math.round(s.avg_duration_minutes)}m`,
  ]

  if (stats.skill_stats?.length > 0) {
    const topSkills = stats.skill_stats.slice(0, 3).map((sk: any) =>
      `${sk.skill} ${sk.spawns}→${sk.completions}`
    ).join(", ")
    lines.push(`**Skills:** ${topSkills}`)
  }

  if (stats.verification_stats) {
    const v = stats.verification_stats
    lines.push(`**Verification:** ${Math.round(v.pass_rate)}% first-try pass (${v.total_attempts} attempts)`)

    if (v.failures_by_gate?.length > 0) {
      const gates = v.failures_by_gate.map((g: any) => `${g.gate}:${g.fail_count}`).join(", ")
      lines.push(`**Gate failures:** ${gates}`)
    }
  }

  if (stats.coaching_stats) {
    const c = stats.coaching_stats
    const highlights: string[] = []
    if (c.frame_collapse?.count > 0) highlights.push(`frame_collapse:${c.frame_collapse.count}`)
    if (c.behavioral_variation?.count > 0) highlights.push(`variation:${c.behavioral_variation.count}`)
    if (c.circular_pattern?.count > 0) highlights.push(`circular:${c.circular_pattern.count}`)
    if (c.premise_skipping?.count > 0) highlights.push(`premise_skip:${c.premise_skipping.count}`)
    if (highlights.length > 0) {
      lines.push(`**Coaching signals:** ${highlights.join(", ")}`)
    }
  }

  return lines.join("\n")
}

/**
 * Calculate and flush metrics for a session.
 */
async function flushMetrics(state: SessionState, client: any): Promise<void> {
  state.lastFlush = Date.now()
  log("Flushed metrics for session:", state.sessionId)
}

/**
 * OpenCode plugin that tracks orchestrator behavioral patterns.
 */
export const CoachingPlugin: Plugin = async ({ directory, client }) => {
  log("Plugin initialized, directory:", directory)

  // Prune old metrics on startup
  pruneMetrics()

  // Phase 2: Load investigation recommendations for circular pattern detection
  const investigationRecommendations = loadInvestigationRecommendations(directory)
  log(`Loaded ${investigationRecommendations.length} investigation recommendations for circular detection`)

  // Worker session tracking (per-session detection, not plugin-level)
  const workerSessions = new Map<string, boolean>()
  const workerHealthStates = new Map<string, WorkerHealthState>()

  /**
   * Detect if a session is a worker by checking session.metadata.role.
   */
  function detectWorkerSession(sessionId: string, session?: { metadata?: { role?: string } }): boolean {
    const cached = workerSessions.get(sessionId)
    if (cached === true) return true

    if (session?.metadata?.role === 'worker') {
      workerSessions.set(sessionId, true)
      log(`Worker detected (session.metadata.role): session ${sessionId}`)
      return true
    }

    return false
  }

  // Session state
  const sessions = new Map<string, SessionState>()
  let toolCallCounter = 0

  return {
    /**
     * Auto-surface orch stats health summary on orchestrator session start.
     */
    event: async ({ event }) => {
      if (event.type !== "session.created") return

      const sessionId = (event as any).properties?.sessionID
      if (!sessionId) return

      const sessionInfo = (event as any).properties?.info
      if (sessionInfo?.metadata?.role === "worker") {
        workerSessions.set(sessionId, true)
        log(`Health summary: Skipping worker session ${sessionId}`)
        return
      }

      ;(async () => {
        await delay(3000)

        if (workerSessions.get(sessionId) === true) {
          log(`Health summary: Skipping late-detected worker ${sessionId}`)
          return
        }

        try {
          const output = await execAsync("orch stats --days 1 --json")
          const stats = JSON.parse(output)
          const summary = formatHealthSummary(stats)

          if (client?.session?.prompt) {
            await client.session.prompt({
              sessionID: sessionId,
              prompt: summary,
              noReply: true,
            })
            log(`Injected health summary into session ${sessionId}`)
          }
        } catch (err) {
          if (DEBUG) console.error(LOG_PREFIX, "Failed to inject health summary:", err)
        }
      })().catch(err => {
        if (DEBUG) console.error(LOG_PREFIX, "Health summary injection error:", err)
      })
    },

    /**
     * Phase 3.5: Track Dylan's behavioral patterns via message monitoring.
     */
    "experimental.chat.messages.transform": async (input: {}, output: { messages: Array<{ info: any; parts: any[] }> }) => {
      const userMessages = extractUserMessages(output.messages)
      if (userMessages.length === 0) return

      const latestMessage = userMessages[userMessages.length - 1]
      const text = latestMessage.text
      const sessionId = output.messages[0]?.info?.sessionID || "unknown"

      if (workerSessions.get(sessionId) === true) return

      let state = sessions.get(sessionId)
      if (!state) {
        state = createSessionState(sessionId)
        sessions.set(sessionId, state)
        log("Created session state for Dylan patterns:", sessionId)
      }

      // Premise-Skipping Detection
      state.dylan.recentQuestions.push(text)
      if (state.dylan.recentQuestions.length > 5) {
        state.dylan.recentQuestions.shift()
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
        log(`PREMISE-SKIPPING DETECTED: "${premiseSkipResult.extracted}" (verb: ${premiseSkipResult.verb})`)

        if (!state.dylan.premiseSkippingWarningInjected) {
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
          await injectCoachingMessage(client, sessionId, "premise_skipping_strong", {
            question: text.substring(0, 200),
            verb: premiseSkipResult.verb,
            count: state.dylan.premiseSkippingCount,
            recentQuestions: state.dylan.recentQuestions,
          })
          state.dylan.premiseSkippingStrongWarningInjected = true
        }

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

      // Worker session handling
      const isWorker = detectWorkerSession(sessionId, input.session)
      if (isWorker) {
        let workerState = workerHealthStates.get(sessionId)
        if (!workerState) {
          const now = Date.now()
          workerState = {
            sessionId,
            sessionStartTime: now,
            consecutiveToolFailures: 0,
            estimatedTokensUsed: 0,
            lastPhaseUpdate: now,
            lastCommitTime: 0,
            totalToolCalls: 0,
            totalReadBytes: 0,
          }
          workerHealthStates.set(sessionId, workerState)
          log(`Created worker health state for session ${sessionId}`)
        }

        const success = !output?.error && !output?.isError
        await trackWorkerHealth(client, workerState, tool, success, input.args, output)

        // Accretion Detection for workers
        if (tool === "edit" || tool === "write") {
          const filePath = (input as any).args?.file_path || (input as any).args?.filePath || ""

          if (filePath) {
            const lineCount = getFileLineCount(filePath)

            if (lineCount && lineCount > AccretionWarningThreshold) {
              let workerSessionState = sessions.get(sessionId)
              if (!workerSessionState) {
                workerSessionState = createSessionState(sessionId)
                sessions.set(sessionId, workerSessionState)
                log(`Created worker session state for accretion tracking: ${sessionId}`)
              }

              const currentCount = workerSessionState.accretion.fileEditCounts.get(filePath) || 0
              workerSessionState.accretion.fileEditCounts.set(filePath, currentCount + 1)

              log(`ACCRETION: Large file edit detected: ${filePath} (${lineCount} lines, edit ${currentCount + 1})`)

              const metric = {
                timestamp: new Date().toISOString(),
                session_id: sessionId,
                metric_type: "accretion_warning",
                value: lineCount,
                details: {
                  filePath,
                  lineCount,
                  editCount: currentCount + 1,
                  severity: lineCount > 1500 ? "critical" : "warning",
                  threshold: lineCount > 1500 ? 1500 : 800,
                },
              }
              writeMetric(metric)

              const hasWarned = workerSessionState.accretion.fileWarningInjected.get(filePath)
              const hasStrongWarned = workerSessionState.accretion.fileStrongWarningInjected.get(filePath)

              if (!hasWarned) {
                await injectCoachingMessage(client, sessionId, "accretion_warning", {
                  filePath,
                  lineCount,
                })
                workerSessionState.accretion.fileWarningInjected.set(filePath, true)
              } else if (currentCount + 1 >= 3 && !hasStrongWarned) {
                await injectCoachingMessage(client, sessionId, "accretion_strong", {
                  filePath,
                  lineCount,
                  count: currentCount + 1,
                })
                workerSessionState.accretion.fileStrongWarningInjected.set(filePath, true)
              }

              streamToCoach(client, sessionId, metric, {})
            }
          }
        }

        return // Skip orchestrator metrics for workers
      }

      // Orchestrator session handling
      let state = sessions.get(sessionId)
      if (!state) {
        state = createSessionState(sessionId)
        sessions.set(sessionId, state)
        log("Created session state:", sessionId)
      }

      // Frame Collapse Detection
      if (tool === "edit" || tool === "write") {
        const filePath = (input as any).args?.file_path || (input as any).args?.filePath || ""

        if (isCodeFile(filePath)) {
          state.frameCollapse.codeEditCount++
          state.frameCollapse.lastCodeEditPath = filePath

          log(`FRAME COLLAPSE: Code file edit detected: ${filePath} (count: ${state.frameCollapse.codeEditCount})`)

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

          if (state.frameCollapse.codeEditCount === 1 && !state.frameCollapse.warningInjected) {
            injectCoachingMessage(client, state.sessionId, "frame_collapse", { filePath })
            state.frameCollapse.warningInjected = true
          } else if (
            state.frameCollapse.codeEditCount >= 3 &&
            !state.frameCollapse.strongWarningInjected
          ) {
            injectCoachingMessage(client, state.sessionId, "frame_collapse_strong", {
              filePath,
              count: state.frameCollapse.codeEditCount,
            })
            state.frameCollapse.strongWarningInjected = true
          }

          streamToCoach(client, sessionId, metric, {})
        }
      }

      // Bash command handling
      if (tool === "bash") {
        const command = (input as any).args?.command || ""

        if (isSpawn(command)) {
          state.spawns++
          log("Spawn detected:", command.substring(0, 50))
        }

        // Phase 1: Behavioral variation detection
        const now = Date.now()
        const timeSinceLastTool = now - state.variation.lastToolTimestamp

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

        const group = classifyBashCommand(command)
        state.variation.lastToolTimestamp = now

        state.variation.variationHistory.push({ group, command, timestamp: now })
        if (state.variation.variationHistory.length > 20) {
          state.variation.variationHistory.shift()
        }

        if (group !== "other") {
          if (state.variation.currentGroup === group) {
            state.variation.variationCount++
            log(
              `Variation ${state.variation.variationCount} in ${group}: ${command.substring(0, 60)}`
            )

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
                `BEHAVIORAL VARIATION DETECTED: ${state.variation.variationCount} variations in ${group} without strategic pause`
              )

              streamToCoach(client, sessionId, metric, {
                recentCommands: recentHistory.map((h) => h.command),
              })
            }
          } else {
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
              `CIRCULAR PATTERN DETECTED: Command uses ${decisionKeywords.join(", ")} but ${contradiction.fileName} recommended ${contradiction.keywords.join(", ")}`
            )

            streamToCoach(client, sessionId, metric, {
              recommendation: contradiction.next,
            })
          }
        }
      }

      // Periodic flush: every 10 tool calls
      toolCallCounter++
      if (toolCallCounter >= 10) {
        for (const [sid, s] of sessions.entries()) {
          if (s.spawns > 0) {
            flushMetrics(s, client)
          }
        }
        toolCallCounter = 0
      }

      // Also flush if session has been active for 5+ minutes since last flush
      const now = Date.now()
      if (now - state.lastFlush > 5 * 60 * 1000) {
        if (state.spawns > 0) {
          flushMetrics(state, client)
        }
      }
    },
  }
}
