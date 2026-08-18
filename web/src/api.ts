import type {
  NotesResponse,
  NotebooksResponse,
  StatsResponse,
  SyncStatus,
  ShelfResponse,
  RandomHighlightsResponse,
} from './types'

async function getJson<T>(url: string): Promise<T> {
  const res = await fetch(url)
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg = (data as { error?: string }).error || `请求失败 (${res.status})`
    throw new Error(msg)
  }
  return data as T
}

export function fetchRandomHighlights() {
  return getJson<RandomHighlightsResponse>('/api/highlights/random')
}

export function refreshRandomHighlights() {
  return fetch('/api/highlights/random', { method: 'POST' }).then(async (res) => {
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      const msg = (data as { error?: string }).error || `请求失败 (${res.status})`
      throw new Error(msg)
    }
    return data as RandomHighlightsResponse
  })
}

export function fetchNotebooks(count = 40, lastSort?: number) {
  const q = new URLSearchParams({ count: String(count) })
  if (lastSort) q.set('lastSort', String(lastSort))
  return getJson<NotebooksResponse>(`/api/notebooks?${q}`)
}

export function fetchNotes(bookId: string) {
  return getJson<NotesResponse>(`/api/books/${encodeURIComponent(bookId)}/notes`)
}

export function fetchStats(mode: string) {
  return getJson<StatsResponse>(`/api/stats?mode=${encodeURIComponent(mode)}`)
}

export function fetchShelf() {
  return getJson<ShelfResponse>('/api/shelf')
}

export function fetchSyncStatus() {
  return getJson<SyncStatus>('/api/sync/status')
}

export function triggerSync(force = false) {
  const q = force ? '?force=1' : ''
  return fetch(`/api/sync${q}`, { method: 'POST' }).then(async (res) => {
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      const msg = (data as { error?: string }).error || `请求失败 (${res.status})`
      throw new Error(msg)
    }
    return data as SyncStatus & { started?: boolean }
  })
}
