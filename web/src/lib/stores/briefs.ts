import { writable } from 'svelte/store'
import type { BriefResponse } from './beads'

const API_BASE = 'http://localhost:3348'

export interface BriefListItem {
	beads_id: string
	marked_read: boolean
}

function createBriefsStore() {
	const { subscribe, set, update } = writable<BriefListItem[]>([])

	return {
		subscribe,
		set,
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/briefs`)
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`)
				}
				const data: BriefListItem[] = await response.json()
				set(data)
			} catch (error) {
				console.error('Failed to fetch briefs list:', error)
				set([])
			}
		},
		async fetchBrief(beadsId: string): Promise<BriefResponse | null> {
			try {
				const response = await fetch(`${API_BASE}/api/briefs/${beadsId}`)
				if (!response.ok) return null
				return await response.json()
			} catch (error) {
				console.error('Failed to fetch brief:', error)
				return null
			}
		},
		async markAsRead(beadsId: string): Promise<boolean> {
			try {
				const response = await fetch(`${API_BASE}/api/briefs/${beadsId}`, {
					method: 'POST',
				})
				if (!response.ok) return false
				update(items =>
					items.map(item =>
						item.beads_id === beadsId
							? { ...item, marked_read: true }
							: item
					)
				)
				return true
			} catch (error) {
				console.error('Failed to mark brief as read:', error)
				return false
			}
		},
	}
}

export const briefs = createBriefsStore()
