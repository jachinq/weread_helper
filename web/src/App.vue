<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { fetchSettings, fetchSyncStatus, triggerSync } from './api'
import { applySiteTitle, siteTitle } from './site'
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

function navClass(path: string, prefix?: boolean) {
  const active = prefix ? route.path.startsWith(path) : route.path === path
  return { active }
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
    .then((s) => applySiteTitle(s.siteTitle))
    .catch(() => {})
  timer = window.setInterval(refreshStatus, 2500)
})

onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <div class="shell">
    <a class="skip-link" href="#main">跳到正文</a>
    <div class="lamp" aria-hidden="true" />
    <header class="masthead">
      <RouterLink class="brand" to="/" :aria-label="siteTitle + '首页'">
        <span class="seal" aria-hidden="true">读</span>
        <div>
          <p class="kicker">Lamp-lit library</p>
          <h1>{{ siteTitle }}</h1>
        </div>
      </RouterLink>
      <div class="mast-actions">
        <nav class="site-nav" aria-label="主导航">
          <RouterLink to="/" :class="navClass('/')" :aria-current="route.path === '/' ? 'page' : undefined">首页</RouterLink>
          <RouterLink
            to="/notes"
            :class="navClass('/notes', true)"
            :aria-current="route.path.startsWith('/notes') ? 'page' : undefined"
          >笔记</RouterLink>
          <RouterLink
            to="/shelf"
            :class="navClass('/shelf', true)"
            :aria-current="route.path.startsWith('/shelf') ? 'page' : undefined"
          >书架</RouterLink>
          <RouterLink
            to="/stats"
            :class="navClass('/stats', true)"
            :aria-current="route.path.startsWith('/stats') ? 'page' : undefined"
          >统计</RouterLink>
          <RouterLink
            to="/settings"
            :class="navClass('/settings', true)"
            :aria-current="route.path.startsWith('/settings') ? 'page' : undefined"
          >设置</RouterLink>
        </nav>
        <div class="sync-box">
          <span class="sync-meta">{{ formatTime(status?.lastOkAt || 0) }}</span>
          <button
            class="btn btn-solid"
            type="button"
            :disabled="status?.state === 'running' || syncingClick"
            :aria-busy="status?.state === 'running' || syncingClick"
            @click="onSync"
          >
            {{ status?.state === 'running' || syncingClick ? '同步中…' : '同步' }}
          </button>
        </div>
      </div>
    </header>
    <p v-if="status?.state === 'running'" class="banner" role="status" aria-live="polite">
      正在把官方数据写入本地库
      <template v-if="status.phase"> · {{ status.phase }}</template>
      <template v-if="status.dirtyTotal"> · {{ status.dirtyDone }}/{{ status.dirtyTotal }} 本</template>
      <template v-if="status.elapsedSec"> · {{ status.elapsedSec }}s</template>
    </p>
    <p v-if="status?.lastError" class="error" role="alert">上次同步失败：{{ status.lastError }}</p>
    <main id="main" tabindex="-1">
      <RouterView v-slot="{ Component }">
        <Transition name="page" mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>
    <footer class="colophon">{{ siteTitle }} · 日常只读本地库，点同步才会请求微信读书</footer>
    <nav class="site-nav dock" aria-label="移动端导航">
      <RouterLink to="/" :class="navClass('/')" :aria-current="route.path === '/' ? 'page' : undefined">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 11.5 12 4l8 7.5V20a1 1 0 0 1-1 1h-5v-6H10v6H5a1 1 0 0 1-1-1z" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/></svg>
        首页
      </RouterLink>
      <RouterLink
        to="/notes"
        :class="navClass('/notes', true)"
        :aria-current="route.path.startsWith('/notes') ? 'page' : undefined"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M7 4h8l4 4v12H7z" fill="none" stroke="currentColor" stroke-width="1.7"/><path d="M15 4v4h4M9 12h8M9 16h6" fill="none" stroke="currentColor" stroke-width="1.7"/></svg>
        笔记
      </RouterLink>
      <RouterLink
        to="/shelf"
        :class="navClass('/shelf', true)"
        :aria-current="route.path.startsWith('/shelf') ? 'page' : undefined"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 19h16M6 19V7h4v12M12 19V5h4v14M18 19v-8h2v8" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/></svg>
        书架
      </RouterLink>
      <RouterLink
        to="/stats"
        :class="navClass('/stats', true)"
        :aria-current="route.path.startsWith('/stats') ? 'page' : undefined"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 19V10h3v9H5zm6 0V5h3v14h-3zm6 0v-7h3v7h-3z" fill="none" stroke="currentColor" stroke-width="1.7"/></svg>
        统计
      </RouterLink>
      <RouterLink
        to="/settings"
        :class="navClass('/settings', true)"
        :aria-current="route.path.startsWith('/settings') ? 'page' : undefined"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="3.2" fill="none" stroke="currentColor" stroke-width="1.7"/><path d="M12 3.5v2.2M12 18.3v2.2M3.5 12h2.2M18.3 12h2.2M6.1 6.1l1.6 1.6M16.3 16.3l1.6 1.6M17.9 6.1l-1.6 1.6M7.7 16.3l-1.6 1.6" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"/></svg>
        设置
      </RouterLink>
    </nav>
  </div>
</template>
