import { onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { fetchSettings, fetchSyncStatus, triggerSync } from '../api'
import { applySiteTitle } from '../site'
import { applyThemeFromSettings } from './apply'
import type { SyncStatus } from '../types'

export function navClassFor(path: string, current: string, prefix?: boolean) {
  const active = prefix ? current.startsWith(path) : current === path
  return { active }
}

export function formatSyncTime(ts: number) {
  if (!ts) return '尚未同步'
  const d = new Date(ts * 1000)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getMonth() + 1}/${d.getDate()} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function useAppChrome() {
  const route = useRoute()
  const status = ref<SyncStatus | null>(null)
  const syncingClick = ref(false)
  let timer: number | undefined

  function navClass(path: string, prefix?: boolean) {
    return navClassFor(path, route.path, prefix)
  }

  async function refreshStatus() {
    try {
      status.value = await fetchSyncStatus()
    } catch {
      /* ignore */
    }
  }

  async function onSync() {
    syncingClick.value = true
    try {
      await triggerSync(false)
      await refreshStatus()
    } finally {
      syncingClick.value = false
    }
  }

  onMounted(() => {
    refreshStatus()
    fetchSettings()
      .then((s) => {
        applySiteTitle(s.siteTitle)
        applyThemeFromSettings(s)
      })
      .catch(() => {})
    timer = window.setInterval(refreshStatus, 2500)
  })

  onUnmounted(() => {
    if (timer) window.clearInterval(timer)
  })

  return { route, status, syncingClick, navClass, onSync, formatSyncTime }
}
