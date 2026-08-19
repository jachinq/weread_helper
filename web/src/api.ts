import type {
  NotesResponse,
  NotebooksResponse,
  StatsResponse,
  SyncStatus,
  ShelfResponse,
  RandomHighlightsResponse,
  AppSettings,
  YearReport,
  ReportYears,
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

export function fetchNotebooks(count = 40, lastSort?: number, query?: string) {
  const q = new URLSearchParams({ count: String(count) })
  if (lastSort) q.set('lastSort', String(lastSort))
  const keyword = query?.trim()
  if (keyword) q.set('q', keyword)
  return getJson<NotebooksResponse>(`/api/notebooks?${q}`)
}

export function fetchNotes(bookId: string) {
  return getJson<NotesResponse>(`/api/books/${encodeURIComponent(bookId)}/notes`)
}

export async function fetchStats(mode: string, year?: number) {
  const q = new URLSearchParams({ mode })
  if (mode === 'annually' && year) q.set('year', String(year))
  const res = await fetch(`/api/stats?${q.toString()}`, { cache: 'no-store' })
  const data = await res.json().catch(() => ({}))
  if (res.status === 404 && (data as { missing?: boolean }).missing) {
    return data as StatsResponse & { missing: true }
  }
  if (!res.ok) {
    const msg = (data as { error?: string }).error || `请求失败 (${res.status})`
    throw new Error(msg)
  }
  return data as StatsResponse
}

export async function fetchReport(year: number) {
  const res = await fetch(`/api/report?year=${encodeURIComponent(String(year))}`)
  const data = await res.json().catch(() => ({}))
  if (res.status === 404 && (data as { missing?: boolean }).missing) {
    return data as YearReport & { missing: true }
  }
  if (!res.ok) {
    const msg = (data as { error?: string }).error || `请求失败 (${res.status})`
    throw new Error(msg)
  }
  return data as YearReport
}

export function fetchReportYears() {
  return getJson<ReportYears>('/api/report/years')
}

export async function fetchReportSnapshot(year: number) {
  const res = await fetch(`/api/report/fetch?year=${encodeURIComponent(String(year))}`, {
    method: 'POST',
    cache: 'no-store',
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg = (data as { error?: string }).error || `请求失败 (${res.status})`
    throw new Error(msg)
  }
  return data as YearReport
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

export function fetchSettings() {
  return getJson<AppSettings>('/api/settings')
}

export function saveSettings(body: {
  apiKey: string
  skillVersion: string
  gatewayUrl: string
  syncInterval: string
  siteTitle: string
  theme: string
  colorScheme: string
}) {
  return fetch('/api/settings', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  }).then(async (res) => {
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      const msg = (data as { error?: string }).error || `请求失败 (${res.status})`
      throw new Error(msg)
    }
    return data as AppSettings
  })
}
