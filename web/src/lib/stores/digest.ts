import { writable } from 'svelte/store'

const API_BASE = 'http://localhost:3348'

export interface DigestSource {
	artifact_type: string
	path: string
	change_type: string
	delta_words?: number
}

export interface DigestProduct {
	id: string
	type: 'thread_progression' | 'model_update' | 'model_probe' | 'decision_brief'
	title: string
	summary: string
	significance: 'low' | 'medium' | 'high'
	source: DigestSource
	state: 'new' | 'read' | 'starred' | 'archived'
	created_at: string
	read_at?: string
	starred_at?: string
	archived_at?: string
}

export interface DigestStatsData {
	unread: number
	read: number
	starred: number
	total: number
}

export interface DigestListResponse {
	products: DigestProduct[]
	unread_count: number
	total: number
	error?: string
}

// Digest stats store (used by navbar badge)
function createDigestStatsStore() {
	const { subscribe, set } = writable<DigestStatsData>({ unread: 0, read: 0, starred: 0, total: 0 })

	return {
		subscribe,
		set,
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/digest/stats`)
				if (!response.ok) throw new Error(`HTTP ${response.status}`)
				const data = await response.json()
				set(data)
			} catch (error) {
				console.error('Failed to fetch digest stats:', error)
				set({ unread: 0, read: 0, starred: 0, total: 0 })
			}
		},
	}
}

// Digest products store (used by thinking page)
function createDigestProductsStore() {
	const { subscribe, set, update } = writable<DigestListResponse>({ products: [], unread_count: 0, total: 0 })

	return {
		subscribe,
		set,
		async fetch(state?: string, type?: string): Promise<void> {
			try {
				const params = new URLSearchParams()
				if (state) params.set('state', state)
				if (type) params.set('type', type)
				const url = `${API_BASE}/api/digest${params.toString() ? '?' + params.toString() : ''}`
				const response = await fetch(url)
				if (!response.ok) throw new Error(`HTTP ${response.status}`)
				const data = await response.json()
				set(data)
			} catch (error) {
				console.error('Failed to fetch digest products:', error)
				set({ products: [], unread_count: 0, total: 0, error: String(error) })
			}
		},
		async updateState(id: string, state: 'read' | 'starred' | 'archived'): Promise<boolean> {
			try {
				const response = await fetch(`${API_BASE}/api/digest/update?id=${encodeURIComponent(id)}`, {
					method: 'PATCH',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ state }),
				})
				if (!response.ok) throw new Error(`HTTP ${response.status}`)
				// Optimistically update local state
				update((current) => ({
					...current,
					products: current.products.map((p) =>
						p.id === id ? { ...p, state } : p
					),
				}))
				return true
			} catch (error) {
				console.error('Failed to update digest product:', error)
				return false
			}
		},
		async archiveRead(): Promise<number> {
			try {
				const response = await fetch(`${API_BASE}/api/digest/archive-read`, { method: 'POST' })
				if (!response.ok) throw new Error(`HTTP ${response.status}`)
				const data = await response.json()
				return data.archived || 0
			} catch (error) {
				console.error('Failed to archive read products:', error)
				return 0
			}
		},
	}
}

export const digestStats = createDigestStatsStore()
export const digestProducts = createDigestProductsStore()
