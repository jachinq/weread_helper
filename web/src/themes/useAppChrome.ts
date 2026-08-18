import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { fetchSettings, fetchSyncStatus, triggerSync } from '../api'
import { applySiteTitle } from '../site'
import { applyThemeFromSettings } from './apply'
import type { SyncStatus } from '../types'

export function navClassFor(path: string, current: string, prefix?: boolean) {
  const active = prefix ? current.startsWith(path) : current === path
  return { active }
}

function pad(n: number) {
  return String(n).padStart(2, '0')
}

export function formatAbsoluteSyncTime(ts: number) {
  if (!ts) return '尚未同步'
  const d = new Date(ts * 1000)
  return `${d.getMonth() + 1}/${d.getDate()} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function formatRelativeSyncTime(ts: number) {
  if (!ts) return '尚未同步'
  const diffSec = Math.max(0, Math.floor(Date.now() / 1000 - ts))
  if (diffSec < 60) return '刚刚'
  const mins = Math.floor(diffSec / 60)
  if (mins < 60) return `${mins} 分钟前`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours} 小时前`
  const days = Math.floor(hours / 24)
  return `${days} 天前`
}

export function formatElapsedSince(ts: number) {
  if (!ts) return ''
  const diffSec = Math.max(0, Math.floor(Date.now() / 1000 - ts))
  const mins = Math.floor(diffSec / 60)
  if (mins < 1) return '不到 1 分钟'
  if (mins < 60) return `${mins} 分钟`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours} 小时`
  return `${Math.floor(hours / 24)} 天`
}

export function useAppChrome() {
  const route = useRoute()
  const status = ref<SyncStatus | null>(null)
  const syncingClick = ref(false)
  const hasApiKey = ref(false)
  const settingsReady = ref(false)
  const nudgeDismissed = ref(false)
  let timer: number | undefined

  function navClass(path: string, prefix?: boolean) {
    return navClassFor(path, route.path, prefix)
  }

  function stopPolling() {
    if (timer) {
      window.clearInterval(timer)
      timer = undefined
    }
  }

  async function refreshStatus() {
    try {
      status.value = await fetchSyncStatus()
    } catch {
      /* ignore */
    }
  }

  function startPollingIfRunning() {
    if (status.value?.state !== 'running') {
      stopPolling()
      return
    }
    if (timer) return
    timer = window.setInterval(async () => {
      await refreshStatus()
      if (status.value?.state !== 'running') {
        stopPolling()
      }
    }, 2500)
  }

  const showNudge = computed(() => {
    if (nudgeDismissed.value) return false
    if (!settingsReady.value || !status.value) return false
    if (status.value.state === 'running') return false
    if (!hasApiKey.value) return true
    return Boolean(status.value.stale)
  })

  const nudgeKind = computed<'key' | 'never' | 'stale' | null>(() => {
    if (!showNudge.value) return null
    if (!hasApiKey.value) return 'key'
    if (!status.value?.lastOkAt) return 'never'
    return 'stale'
  })

  const nudgeText = computed(() => {
    switch (nudgeKind.value) {
      case 'key':
        return '尚未配置 API Key，请到设置页填写后再同步'
      case 'never':
        return '尚未同步，请点击同步以写入本地库'
      case 'stale':
        return `距上次同步已 ${formatElapsedSince(status.value?.lastOkAt || 0)}，建议同步以更新笔记`
      default:
        return ''
    }
  })

  function dismissNudge() {
    nudgeDismissed.value = true
  }

  async function onSync() {
    syncingClick.value = true
    try {
      const st = await triggerSync(false)
      status.value = st
      nudgeDismissed.value = true
      startPollingIfRunning()
    } finally {
      syncingClick.value = false
    }
  }

  onMounted(() => {
    void refreshStatus().then(() => startPollingIfRunning())
    fetchSettings()
      .then((s) => {
        hasApiKey.value = Boolean(s.apiKeyMasked)
        applySiteTitle(s.siteTitle)
        applyThemeFromSettings(s)
      })
      .catch(() => {
        hasApiKey.value = false
      })
      .finally(() => {
        settingsReady.value = true
      })
  })

  onUnmounted(() => {
    stopPolling()
  })

  return {
    route,
    status,
    syncingClick,
    navClass,
    onSync,
    formatRelativeSyncTime,
    formatAbsoluteSyncTime,
    showNudge,
    nudgeKind,
    nudgeText,
    dismissNudge,
  }
}
