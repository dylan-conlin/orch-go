/**
 * Shared types, constants, and utilities for the coaching plugin.
 */
import {
  appendFileSync,
  mkdirSync,
  existsSync,
  readFileSync,
  writeFileSync,
} from "fs"
import { homedir } from "os"
import { join, dirname } from "path"

export const LOG_PREFIX = "[coaching]"
export const DEBUG = process.env.ORCH_PLUGIN_DEBUG === "1"
export const METRICS_PATH = join(homedir(), ".orch", "coaching-metrics.jsonl")
export const MAX_LINES = 1000 // Keep last 1000 lines
export const STRATEGIC_PAUSE_MS = 30 * 1000 // 30 seconds = strategic pause
export const VARIATION_THRESHOLD = 5 // 5+ variations triggers behavioral_variation metric
export const COACH_SESSION_ID = process.env.ORCH_COACH_SESSION_ID || "" // Coach session to stream metrics to

// Accretion thresholds (matches pkg/verify/accretion.go)
export const AccretionWarningThreshold = 800   // Files >800 lines trigger warnings
export const AccretionCriticalThreshold = 1500 // Files >1500 lines are CRITICAL

export function log(...args: any[]) {
  if (DEBUG) console.log(LOG_PREFIX, ...args)
}

export interface CoachingMetric {
  timestamp: string
  session_id?: string
  metric_type: string
  value: number
  details?: any
}

export interface InvestigationRecommendation {
  filePath: string
  fileName: string
  date: string
  next: string
  keywords: string[]
}

export type SemanticGroup =
  | "process_mgmt"
  | "git"
  | "build"
  | "test"
  | "knowledge"
  | "orch"
  | "file_ops"
  | "network"
  | "other"

export interface VariationState {
  currentGroup: SemanticGroup | null
  variationCount: number
  lastToolTimestamp: number
  variationHistory: Array<{ group: SemanticGroup; command: string; timestamp: number }>
}

export interface DylanPatternState {
  premiseSkippingCount: number
  premiseSkippingWarningInjected: boolean
  premiseSkippingStrongWarningInjected: boolean
  recentQuestions: string[]
}

export interface FrameCollapseState {
  codeEditCount: number
  lastCodeEditPath: string | null
  warningInjected: boolean
  strongWarningInjected: boolean
}

export interface AccretionState {
  fileEditCounts: Map<string, number>
  fileWarningInjected: Map<string, boolean>
  fileStrongWarningInjected: Map<string, boolean>
}

export interface WorkerHealthState {
  sessionId: string
  sessionStartTime: number
  consecutiveToolFailures: number
  estimatedTokensUsed: number
  lastPhaseUpdate: number
  lastCommitTime: number
  totalToolCalls: number
  totalReadBytes: number
  lastWarningType?: string
  lastWarningValue?: number
}

export interface SessionState {
  sessionId: string
  spawns: number
  lastFlush: number
  variation: VariationState
  dylan: DylanPatternState
  frameCollapse: FrameCollapseState
  accretion: AccretionState
}

/**
 * Write metric to JSONL file.
 */
export function writeMetric(metric: CoachingMetric): void {
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
export function pruneMetrics(): void {
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
 * Create a fresh SessionState for a given session ID.
 */
export function createSessionState(sessionId: string): SessionState {
  return {
    sessionId,
    spawns: 0,
    lastFlush: Date.now(),
    variation: {
      currentGroup: null,
      variationCount: 0,
      lastToolTimestamp: Date.now(),
      variationHistory: [],
    },
    dylan: {
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
    accretion: {
      fileEditCounts: new Map(),
      fileWarningInjected: new Map(),
      fileStrongWarningInjected: new Map(),
    },
  }
}
