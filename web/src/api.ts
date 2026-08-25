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
  SearchNotesResponse,
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

export function fetchOnThisDayHighlights() {
  return getJson<RandomHighlightsResponse>('/api/highlights/on-this-day')
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

export function searchNotes(query: string, kind = 'all', limit = 40) {
  const q = new URLSearchParams({ q: query.trim(), kind, limit: String(limit) })
  return getJson<SearchNotesResponse>(`/api/search?${q}`)
}

export function fetchStarredHighlights(limit = 80) {
  return getJson<RandomHighlightsResponse>(`/api/highlights/starred?limit=${limit}`)
}

export function setHighlightStarred(bookmarkId: string, starred: boolean) {
  return fetch('/api/highlights/star', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ bookmarkId, starred }),
  }).then(async (res) => {
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {
      const msg = (data as { error?: string }).error || `请求失败 (${res.status})`
      throw new Error(msg)
    }
    return data as { bookmarkId: string; starred: boolean }
  })
}

export async function fetchNotesExport(opts?: { bookId?: string; format?: 'md' | 'json' }) {
  const format = opts?.format || 'md'
  const url = opts?.bookId
    ? `/api/books/${encodeURIComponent(opts.bookId)}/export?format=${format}`
    : `/api/export?format=${format}`
  const res = await fetch(url)
  if (!res.ok) {
    const data = await res.json().catch(() => ({}))
    throw new Error((data as { error?: string }).error || `导出失败 (${res.status})`)
  }
  return res
}

export async function downloadNotesExport(opts?: { bookId?: string; format?: 'md' | 'json'; filename?: string }) {
  const format = opts?.format || 'md'
  const res = await fetchNotesExport(opts)
  const blob = await res.blob()
  const fallback = opts?.bookId ? `笔记.${format}` : `纸间笔记-全部.${format}`
  const name = sanitizeDownloadName(opts?.filename || filenameFromDisposition(res.headers.get('Content-Disposition') || ''), format) || fallback
  const a = document.createElement('a')
  const href = URL.createObjectURL(blob)
  a.href = href
  a.download = name
  document.body.appendChild(a)
  a.click()
  a.remove()
  window.setTimeout(() => URL.revokeObjectURL(href), 1000)
}

function filenameFromDisposition(disp: string) {
  const star = /filename\*\s*=\s*(?:UTF-8'')?([^;]+)/i.exec(disp)
  if (star?.[1]) {
    try {
      return decodeURIComponent(star[1].trim().replace(/^UTF-8''/i, '').replace(/"/g, ''))
    } catch {
      /* ignore */
    }
  }
  const quoted = /filename\s*=\s*"([^"]+)"/i.exec(disp)
  if (quoted?.[1] && /^[\x20-\x7e]+$/.test(quoted[1])) return quoted[1]
  return ''
}

function sanitizeDownloadName(name: string, format: string) {
  const cleaned = name.replace(/[\\/:*?"<>|]/g, '_').trim()
  if (!cleaned) return ''
  const ext = `.${format}`
  return cleaned.toLowerCase().endsWith(ext) ? cleaned : `${cleaned}${ext}`
}

export async function fetchStats(
  mode: string,
  period?: { year?: number; month?: string; week?: string },
) {
  const q = new URLSearchParams({ mode })
  if (mode === 'annually' && period?.year) q.set('year', String(period.year))
  if (mode === 'monthly' && period?.month) q.set('month', period.month)
  if (mode === 'weekly' && period?.week) q.set('week', period.week)
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

export async function fetchStatsSnapshot(mode: string, period: { year?: number; month?: string; week?: string }) {
  const q = new URLSearchParams({ mode })
  if (mode === 'annually' && period.year) q.set('year', String(period.year))
  if (mode === 'monthly' && period.month) q.set('month', period.month)
  if (mode === 'weekly' && period.week) q.set('week', period.week)
  const res = await fetch(`/api/stats/fetch?${q.toString()}`, {
    method: 'POST',
    cache: 'no-store',
  })
  const data = await res.json().catch(() => ({}))
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
  highlightDisplay: string
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
