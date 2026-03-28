/**
 * Worker health metrics tracking and health signal injection.
 */
import { execFile } from "child_process"

import { classifyBashCommand } from "./coaching-classification"
import { findContradiction, detectArchitecturalDecision } from "./coaching-investigation"
import type { WorkerHealthState } from "./coaching-types"
import type { InvestigationRecommendation } from "./coaching-types"
import { log, writeMetric, VARIATION_THRESHOLD } from "./coaching-types"

const PHASE_COMMENT_PATTERN = /bd\s+comments?\s+add\s+.+phase:/i
const FRUSTRATION_SIGNAL_WINDOW_MS = 15 * 60 * 1000
const FRUSTRATION_TIME_IN_PHASE_MINUTES = 15

function execFileAsync(file: string, args: string[], cwd: string): Promise<void> {
  return new Promise((resolve, reject) => {
    execFile(file, args, { cwd, timeout: 10000 }, (error) => {
      if (error) {
        reject(error)
        return
      }
      resolve()
    })
  })
}

function markBoundarySignal(state: WorkerHealthState, signal: string, detail: string, now: number): void {
  state.activeBoundarySignals.set(signal, { seenAt: now, detail })
}

function clearBoundarySignal(state: WorkerHealthState, signal: string): void {
  state.activeBoundarySignals.delete(signal)
}

function getActiveBoundarySignals(state: WorkerHealthState, now: number): Array<{ signal: string; detail: string }> {
  const active: Array<{ signal: string; detail: string }> = []
  for (const [signal, entry] of state.activeBoundarySignals.entries()) {
    if (now - entry.seenAt <= FRUSTRATION_SIGNAL_WINDOW_MS) {
      active.push({ signal, detail: entry.detail })
    }
  }
  return active
}

async function reportFrustrationBoundary(
  state: WorkerHealthState,
  directory: string,
  activeSignals: Array<{ signal: string; detail: string }>
): Promise<void> {
  if (!state.beadsID || state.frustrationBoundaryTriggered) {
    return
  }

  const summary = activeSignals.map((entry) => entry.detail).join("; ")
  writeMetric({
    timestamp: new Date().toISOString(),
    session_id: state.sessionId,
    metric_type: "frustration_compound",
    value: activeSignals.length,
    details: {
      beads_id: state.beadsID,
      signals: activeSignals.map((entry) => entry.signal),
      summary,
    },
  })

  const comment = `Phase: Boundary - frustration compound detected (${activeSignals.map((entry) => entry.signal).join(", ")}): ${summary}`
  await execFileAsync("bd", ["comments", "add", state.beadsID, comment], directory)
  state.frustrationBoundaryTriggered = true
  log(`Worker frustration boundary triggered for ${state.beadsID}: ${comment}`)
}

/**
 * Estimate token usage for a worker session.
 * Rough approximation: 1 token ~ 4 chars average.
 */
export function estimateWorkerTokenUsage(state: WorkerHealthState): number {
  const TOKENS_PER_TOOL_CALL = 500
  const CHARS_PER_TOKEN = 4

  const toolCallTokens = state.totalToolCalls * TOKENS_PER_TOOL_CALL
  const readTokens = Math.round(state.totalReadBytes / CHARS_PER_TOKEN)

  return toolCallTokens + readTokens
}

/**
 * Inject a health signal (warning) into the agent's context using noReply: true.
 */
export async function injectHealthSignal(
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
        sessionID: state.sessionId,
        prompt,
        noReply: true,
      })
      log(`Injected health signal for ${metricType}: ${value}`)

      state.lastWarningType = metricType
      state.lastWarningValue = value
    } catch (err) {
      log(`Failed to inject health signal for ${metricType}:`, err)
    }
  }
}

/**
 * Track worker health metrics and record to coaching-metrics.jsonl.
 */
export async function trackWorkerHealth(
  client: any,
  state: WorkerHealthState,
  tool: string,
  success: boolean,
  args: any,
  output: any,
  investigationRecommendations: InvestigationRecommendation[],
  directory: string
): Promise<void> {
  const now = Date.now()
  const timestamp = new Date().toISOString()

  state.totalToolCalls++

  if (tool === "read" && output?.text) {
    state.totalReadBytes += output.text.length
  }

  if (tool === "bash" && args?.command && success) {
    const command = args.command as string

    if (PHASE_COMMENT_PATTERN.test(command)) {
      state.lastPhaseUpdate = now
      clearBoundarySignal(state, "stuck")
      log(`Worker phase update detected for ${state.sessionId}`)
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
      } else {
        state.variation.currentGroup = group
        state.variation.variationCount = 1
      }

      if (state.variation.variationCount >= VARIATION_THRESHOLD) {
        const recentHistory = state.variation.variationHistory
          .filter((entry) => entry.group === group)
          .slice(-VARIATION_THRESHOLD)

        const detail = `thrashing on ${group} commands (${state.variation.variationCount} consecutive attempts)`
        writeMetric({
          timestamp,
          session_id: state.sessionId,
          metric_type: "worker_behavioral_variation",
          value: state.variation.variationCount,
          details: {
            group,
            commands: recentHistory.map((entry) => entry.command.substring(0, 100)),
            threshold: VARIATION_THRESHOLD,
          },
        })
        markBoundarySignal(state, "thrashing", detail, now)
      }
    }

    const decisionKeywords = detectArchitecturalDecision(command)
    if (decisionKeywords && decisionKeywords.length > 0) {
      const contradiction = findContradiction(decisionKeywords, investigationRecommendations)
      if (contradiction) {
        const detail = `contradicting prior work from ${contradiction.fileName}`
        writeMetric({
          timestamp,
          session_id: state.sessionId,
          metric_type: "worker_circular_pattern",
          value: 1,
          details: {
            decision_command: command.substring(0, 200),
            decision_keywords: decisionKeywords,
            contradicts_investigation: contradiction.fileName,
            recommendation: contradiction.next.substring(0, 200),
            recommendation_keywords: contradiction.keywords,
            recommendation_date: contradiction.date,
          },
        })
        markBoundarySignal(state, "contradicting_prior_work", detail, now)
      }
    }
  }

  // 1. tool_failure_rate
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
      await injectHealthSignal(client, state, "tool_failure_rate", state.consecutiveToolFailures)
    }
  } else {
    state.consecutiveToolFailures = 0
  }

  // 2. context_usage
  state.estimatedTokensUsed = estimateWorkerTokenUsage(state)
  const CONTEXT_WARNING_THRESHOLD = 80000
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

    if (percentUsed >= 80) {
      await injectHealthSignal(client, state, "context_usage", percentUsed)
    }
  }

  // 3. time_in_phase
  const minutesInPhase = Math.round((now - state.lastPhaseUpdate) / 60000)
  const TIME_IN_PHASE_WARNING_MINUTES = 15
  if (minutesInPhase >= FRUSTRATION_TIME_IN_PHASE_MINUTES) {
    markBoundarySignal(state, "stuck", `stuck in same phase for ${minutesInPhase} minutes`, now)
  } else {
    clearBoundarySignal(state, "stuck")
  }
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

    if (minutesInPhase >= TIME_IN_PHASE_WARNING_MINUTES) {
      await injectHealthSignal(client, state, "time_in_phase", minutesInPhase)
    }
  }

  // 4. commit_gap
  if (tool === "bash" && args?.command) {
    const command = args.command as string
    if (command.includes("git commit") && success) {
      state.lastCommitTime = now
      log(`Worker: git commit detected, updating lastCommitTime`)
    }
  }

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

    if (minutesSinceCommit >= COMMIT_GAP_WARNING_MINUTES) {
      await injectHealthSignal(client, state, "commit_gap", minutesSinceCommit)
    }
  }

  const activeSignals = getActiveBoundarySignals(state, now)
  if (activeSignals.length >= 2) {
    await reportFrustrationBoundary(state, directory, activeSignals)
  }
}
