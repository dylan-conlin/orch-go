/**
 * Test Plugin: Verify access to timing and transcript data
 *
 * Purpose: Probe technical feasibility for orchestrator coaching plugin
 * Tests whether plugins can access:
 * - Tool execution timing (start/end)
 * - Message timing
 * - Transcript content (parts)
 */

import type { Plugin } from "@opencode-ai/plugin"

const LOG_PREFIX = "[test-timing]"

export const TestTimingPlugin: Plugin = async ({ directory }) => {
  console.log(LOG_PREFIX, "Plugin initialized")

  return {
    /**
     * Test 1: Can we access tool timing from tool.execute.after?
     */
    "tool.execute.after": async (input: any, output: any) => {
      const tool = input.tool
      const sessionId = input.sessionID
      const callId = input.callID

      console.log(LOG_PREFIX, "Tool executed:", {
        tool,
        sessionId: sessionId?.substring(0, 8),
        callId: callId?.substring(0, 8),
        hasOutput: !!output,
        outputKeys: output ? Object.keys(output) : [],
      })

      // Check if timing data is available in output
      if (output?.metadata) {
        console.log(LOG_PREFIX, "Metadata available:", output.metadata)
      }
    },

    /**
     * Test 2: Can we access message parts and timing?
     */
    "experimental.chat.messages.transform": async (input: any, output: any) => {
      if (!output?.messages || output.messages.length === 0) {
        return
      }

      // Check last message for timing data
      const lastMsg = output.messages[output.messages.length - 1]
      const info = lastMsg.info
      const parts = lastMsg.parts

      console.log(LOG_PREFIX, "Message transform:", {
        role: info.role,
        hasTime: !!info.time,
        timeKeys: info.time ? Object.keys(info.time) : [],
        partCount: parts?.length || 0,
      })

      // Check if parts have timing
      if (parts && parts.length > 0) {
        const toolParts = parts.filter((p: any) => p.type === "tool")
        const textParts = parts.filter((p: any) => p.type === "text")

        console.log(LOG_PREFIX, "Part breakdown:", {
          toolParts: toolParts.length,
          textParts: textParts.length,
        })

        // Check first tool part for timing
        if (toolParts.length > 0) {
          const toolPart = toolParts[0]
          console.log(LOG_PREFIX, "Tool part structure:", {
            type: toolPart.type,
            tool: toolPart.tool,
            status: toolPart.state?.status,
            hasTime: !!toolPart.state?.time,
            timeKeys: toolPart.state?.time ? Object.keys(toolPart.state.time) : [],
          })
        }

        // Check text part for timing
        if (textParts.length > 0) {
          const textPart = textParts[0]
          console.log(LOG_PREFIX, "Text part structure:", {
            type: textPart.type,
            hasTime: !!textPart.time,
            timeKeys: textPart.time ? Object.keys(textPart.time) : [],
            textLength: textPart.text?.length || 0,
          })
        }
      }
    },
  }
}
