<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { fetchSyncStatus, triggerSync } from './api'
import type { SyncStatus } from './types'

const route = useRoute()
const status = ref<SyncStatus | null>(null)
const syncingClick = ref(false)
let timer: number | undefined

function formatTime(ts: number) {
  if (!ts) return '尚未同步'
  const d = new Date(ts * 1000)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getMonth() + 1}/${d.getDate()} ${pad(d.getHours())}:${pad(d.getMinutes())}`
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
  timer = window.setInterval(refreshStatus, 2500)
})

onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <div class="shell">
    <header class="masthead">
      <div class="brand">
        <span class="seal">读</span>
        <div>
          <p class="kicker">WeRead Companion</p>
          <h1>纸间笔记</h1>
        </div>
      </div>
      <div class="mast-actions">
        <nav>
          <RouterLink to="/" :class="{ active: route.path === '/' }">首页</RouterLink>
          <RouterLink to="/notes" :class="{ active: route.path.startsWith('/notes') }">笔记</RouterLink>
          <RouterLink to="/shelf" :class="{ active: route.path.startsWith('/shelf') }">书架</RouterLink>
          <RouterLink to="/stats" :class="{ active: route.path.startsWith('/stats') }">统计</RouterLink>
        </nav>
        <div class="sync-box">
          <span class="sync-meta">{{ formatTime(status?.lastOkAt || 0) }}</span>
          <button
            class="btn"
            :disabled="status?.state === 'running' || syncingClick"
            @click="onSync"
          >
            {{ status?.state === 'running' || syncingClick ? '同步中…' : '同步' }}
          </button>
        </div>
      </div>
    </header>
    <p v-if="status?.state === 'running'" class="banner">
      同步中
      <template v-if="status.phase"> · {{ status.phase }}</template>
      <template v-if="status.dirtyTotal"> · {{ status.dirtyDone }}/{{ status.dirtyTotal }} 本书</template>
      <template v-if="status.elapsedSec"> · 已用 {{ status.elapsedSec }}s</template>
    </p>
    <p v-if="status?.lastError" class="error">上次同步：{{ status.lastError }}</p>
    <main>
      <RouterView />
    </main>
    <footer class="colophon">个人助手 · 列表与详情读本地库，同步时才请求官方接口</footer>
  </div>
</template>
