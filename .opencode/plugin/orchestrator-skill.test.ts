import { describe, it, expect, beforeEach, mock } from "bun:test"
import { OrchestratorSkillPlugin } from "./orchestrator-skill"
import { readFileSync } from "fs"
import { homedir } from "os"
import { join } from "path"

const SKILL_PATH = join(
  homedir(),
  ".claude",
  "skills",
  "meta",
  "orchestrator",
  "SKILL.md",
)

// Minimal mock client with session.get
function createMockClient(sessions: Record<string, any> = {}) {
  return {
    session: {
      get: mock(async ({ path }: { path: { id: string } }) => {
        const session = sessions[path.id]
        return { data: session || { metadata: {} } }
      }),
      prompt: mock(async () => ({})),
      promptAsync: mock(async () => ({})),
      list: mock(async () => ({ data: [] })),
    },
    app: {
      log: mock(async () => {}),
    },
    find: {},
    file: {},
  } as any
}

function createPluginInput(clientOverride?: any) {
  return {
    client: clientOverride || createMockClient(),
    project: { name: "orch-go" } as any,
    directory: "/Users/test/orch-go",
    worktree: "/Users/test/orch-go",
    serverUrl: new URL("http://localhost:4096"),
    $: {} as any,
  }
}

describe("OrchestratorSkillPlugin", () => {
  it("initializes and returns hooks", async () => {
    const input = createPluginInput()
    const hooks = await OrchestratorSkillPlugin(input)
    expect(hooks["experimental.chat.system.transform"]).toBeDefined()
  })

  it("injects skill for non-worker session", async () => {
    const client = createMockClient({
      "session-123": { metadata: { role: "user" } },
    })
    const input = createPluginInput(client)
    const hooks = await OrchestratorSkillPlugin(input)

    const system: string[] = ["existing system prompt"]
    await hooks["experimental.chat.system.transform"]!(
      { sessionID: "session-123", model: {} as any },
      { system },
    )

    expect(system.length).toBe(2)
    // Second element should be the orchestrator skill content
    const skillContent = readFileSync(SKILL_PATH, "utf-8")
    expect(system[1]).toBe(skillContent)
  })

  it("skips injection for worker session", async () => {
    const client = createMockClient({
      "worker-456": { metadata: { role: "worker" } },
    })
    const input = createPluginInput(client)
    const hooks = await OrchestratorSkillPlugin(input)

    const system: string[] = ["existing system prompt"]
    await hooks["experimental.chat.system.transform"]!(
      { sessionID: "worker-456", model: {} as any },
      { system },
    )

    expect(system.length).toBe(1)
    expect(system[0]).toBe("existing system prompt")
  })

  it("skips injection when no sessionID", async () => {
    const input = createPluginInput()
    const hooks = await OrchestratorSkillPlugin(input)

    const system: string[] = ["existing system prompt"]
    await hooks["experimental.chat.system.transform"]!(
      { sessionID: undefined, model: {} as any },
      { system },
    )

    expect(system.length).toBe(1)
  })

  it("caches worker status across calls", async () => {
    const client = createMockClient({
      "session-789": { metadata: {} },
    })
    const input = createPluginInput(client)
    const hooks = await OrchestratorSkillPlugin(input)

    const system1: string[] = []
    await hooks["experimental.chat.system.transform"]!(
      { sessionID: "session-789", model: {} as any },
      { system: system1 },
    )

    const system2: string[] = []
    await hooks["experimental.chat.system.transform"]!(
      { sessionID: "session-789", model: {} as any },
      { system: system2 },
    )

    // client.session.get should only be called once (cached)
    expect(client.session.get).toHaveBeenCalledTimes(1)
    // Both calls should inject
    expect(system1.length).toBe(1)
    expect(system2.length).toBe(1)
  })

  it("injects for session with no metadata", async () => {
    const client = createMockClient({
      "session-no-meta": {},
    })
    const input = createPluginInput(client)
    const hooks = await OrchestratorSkillPlugin(input)

    const system: string[] = []
    await hooks["experimental.chat.system.transform"]!(
      { sessionID: "session-no-meta", model: {} as any },
      { system },
    )

    // No metadata.role = not a worker, should inject
    expect(system.length).toBe(1)
  })

  it("handles session.get failure gracefully", async () => {
    const client = createMockClient()
    client.session.get = mock(async () => {
      throw new Error("Network error")
    })
    const input = createPluginInput(client)
    const hooks = await OrchestratorSkillPlugin(input)

    const system: string[] = []
    await hooks["experimental.chat.system.transform"]!(
      { sessionID: "failing-session", model: {} as any },
      { system },
    )

    // Should default to non-worker (inject skill)
    expect(system.length).toBe(1)
  })
})
