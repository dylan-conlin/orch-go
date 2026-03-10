/**
 * Coaching message injection and streaming to coach session.
 */
import type { CoachingMetric } from "./coaching-types"
import { log, DEBUG, LOG_PREFIX, COACH_SESSION_ID } from "./coaching-types"

/**
 * Inject coaching message directly into a session.
 * Uses noReply:true pattern to avoid blocking.
 */
export async function injectCoachingMessage(
  client: any,
  sessionId: string,
  patternType: "frame_collapse" | "frame_collapse_strong" | "premise_skipping" | "premise_skipping_strong" | "accretion_warning" | "accretion_strong",
  details: any
): Promise<void> {
  try {
    let message = ""

    if (patternType === "frame_collapse") {
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
    } else if (patternType === "accretion_warning") {
      message = `## 📏 File Size Warning

You're editing: \`${details.filePath}\` (${details.lineCount} lines)

**Observation:** This file is ${details.lineCount > 1500 ? "CRITICAL" : "large"} (threshold: ${details.lineCount > 1500 ? "1,500" : "800"} lines). Adding features to large files contributes to "accretion gravity" - the pattern where files grow from manageable to unmaintainable through repeated small additions.

**Consider:**
1. Extract logic to a new file/package before adding features
2. See \`.kb/guides/code-extraction-patterns.md\` for extraction workflow
3. Run \`orch hotspot\` to see all files in accretion risk zones

**Why this matters:** Files over ${details.lineCount > 1500 ? "1,500" : "800"} lines are harder to understand, modify, and test. Extraction before addition prevents technical debt accumulation.`
    } else if (patternType === "accretion_strong") {
      message = `## 🚨 Accretion Alert - Multiple Edits to Large File

You've now made **${details.count} edits** to \`${details.filePath}\` (${details.lineCount} lines).

**Pattern detected:** Repeatedly modifying a ${details.lineCount > 1500 ? "CRITICAL" : "large"} file instead of extracting.

**Required Action:**
1. **STOP** adding to this file
2. **EXTRACT** logic to pkg/ or a new module BEFORE adding features
3. **REFERENCE** \`.kb/guides/code-extraction-patterns.md\` for extraction workflow
4. **CHECK** completion gates will ${details.lineCount > 1500 ? "BLOCK" : "WARN"} if you add +50 lines without extraction

**Why this matters:** Accretion gravity is how spawn_cmd.go grew from 200 to 2,332 lines. Each agent added "just one feature" without extracting. Prevention through extraction is cheaper than cleanup later.

**Suggested approach:** Create a new package for this feature's logic, import it from the existing file.`
    }

    if (client?.session?.prompt) {
      await client.session.prompt({
        sessionID: sessionId,
        prompt: message,
        noReply: true,
      })
      log(`Injected ${patternType} coaching into session ${sessionId}`)
    } else {
      log(`Cannot inject coaching: client.session.prompt unavailable`)
    }
  } catch (err) {
    if (DEBUG) console.error(LOG_PREFIX, "Failed to inject coaching:", err)
  }
}

/**
 * Stream metric to coach session for investigation.
 */
export async function streamToCoach(
  client: any,
  sessionId: string,
  metric: CoachingMetric,
  context: { recentCommands?: string[]; recommendation?: string }
): Promise<void> {
  if (!COACH_SESSION_ID) {
    return
  }

  if (sessionId === COACH_SESSION_ID) {
    log("Skipping coach stream - current session is coach")
    return
  }

  try {
    const message = formatMetricForCoach(metric, context)

    await client.session.promptAsync({
      sessionID: COACH_SESSION_ID,
      parts: [
        {
          type: "text",
          text: message,
        },
      ],
    })

    log(`Streamed ${metric.metric_type} metric to coach session ${COACH_SESSION_ID}`)
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
  }

  if (context.recentCommands) {
    lines.push(`**Recent Transcript Context:**`, ...context.recentCommands.map((cmd: string) => `- ${cmd}`), ``)
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
