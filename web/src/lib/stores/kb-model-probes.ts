import { writable } from 'svelte/store'

const API_BASE = 'https://localhost:3348'

export type ModelProbeStatus = 'needs_review' | 'stale' | 'well_validated' | 'active'
export type ProbeVerdict = 'confirms' | 'extends' | 'contradicts'

export interface ModelProbeSummary {
  models_total: number
  probes_total: number
  needs_review: number
  stale: number
  well_validated: number
}

export interface ModelProbeCounts {
  confirms: number
  extends: number
  contradicts: number
}

export interface ModelProbe {
  probe_path: string
  title?: string
  model: string
  verdict: ProbeVerdict
  date: string
  claim: string
  merged: boolean
}

export interface ModelProbeItem {
  name: string
  path: string
  last_updated: string
  status: ModelProbeStatus
  probe_counts: ModelProbeCounts
  unmerged_count: number
  last_probe_at?: string
  probes: ModelProbe[]
}

export interface KBModelProbesResponse {
  summary: ModelProbeSummary
  queue: ModelProbe[]
  models: ModelProbeItem[]
  project_dir?: string
  error?: string
}

function createKBModelProbesStore() {
  const { subscribe, set, update } = writable<KBModelProbesResponse | null>(null)
  let controller: AbortController | null = null
  let request = 0

  return {
    subscribe,
    set,
    update,
    async fetch(projectDir?: string, staleDays: number = 30): Promise<void> {
      request += 1
      const id = request

      if (controller) {
        controller.abort()
      }

      controller = new AbortController()

      try {
        const params = new URLSearchParams()
        if (projectDir) {
          params.set('project_dir', projectDir)
        }
        params.set('stale_days', String(staleDays))

        const url = `${API_BASE}/api/kb/model-probes${params.toString() ? '?' + params.toString() : ''}`
        const response = await fetch(url, { signal: controller.signal })

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`)
        }

        const data = await response.json()
        if (id !== request) {
          return
        }

        set(data)
      } catch (error) {
        if (id !== request) {
          return
        }
        if (error instanceof Error && error.name === 'AbortError') {
          return
        }

        console.error('Failed to fetch KB model probes:', error)
        set({
          summary: {
            models_total: 0,
            probes_total: 0,
            needs_review: 0,
            stale: 0,
            well_validated: 0,
          },
          queue: [],
          models: [],
          error: String(error),
        })
      } finally {
        if (id === request) {
          controller = null
        }
      }
    },
  }
}

export const kbModelProbes = createKBModelProbesStore()
