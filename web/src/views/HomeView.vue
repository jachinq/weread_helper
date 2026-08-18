<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { fetchRandomHighlights, refreshRandomHighlights } from '../api'
import type { RandomHighlight } from '../types'

const pool = ref<RandomHighlight[]>([])
const loading = ref(false)
const error = ref('')
const count = ref(5)
const drawKey = ref(0)
let media640: MediaQueryList | undefined
let media960: MediaQueryList | undefined

const items = computed(() => pool.value.slice(0, count.value))

function pickCount() {
  if (window.matchMedia('(max-width: 639px)').matches) return 3
  if (window.matchMedia('(max-width: 959px)').matches) return 4
  return 5
}

async function load(refresh = false) {
  loading.value = true
  error.value = ''
  try {
    const data = refresh ? await refreshRandomHighlights() : await fetchRandomHighlights()
    pool.value = data.items || []
    drawKey.value += 1
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

function onBreakpoint() {
  count.value = pickCount()
}

function formatDate(ts: number) {
  if (!ts) return ''
  const d = new Date(ts * 1000)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}.${m}.${day}`
}

onMounted(() => {
  count.value = pickCount()
  media640 = window.matchMedia('(max-width: 639px)')
  media960 = window.matchMedia('(max-width: 959px)')
  media640.addEventListener('change', onBreakpoint)
  media960.addEventListener('change', onBreakpoint)
  load(false)
})

onUnmounted(() => {
  media640?.removeEventListener('change', onBreakpoint)
  media960?.removeEventListener('change', onBreakpoint)
})
</script>

<template>
  <section class="home">
    <header class="home-head">
      <div>
        <p class="home-kicker">Commonplace · 抽签</p>
        <h2 class="page-title">今日摘抄</h2>
        <p class="muted">从历史划线里随手抽出几张藏书票</p>
      </div>
      <button class="btn home-redraw" :disabled="loading" @click="load(true)">
        {{ loading ? '抽取中…' : '换一批' }}
      </button>
    </header>

    <div v-if="error" class="error">{{ error }}</div>

    <p v-else-if="!loading && !items.length" class="home-empty">
      还没有划线。先去同步，再回来抽一张纸。
    </p>

    <div v-else-if="items.length" class="home-spread" :key="drawKey" :data-count="items.length">
      <RouterLink
        v-for="(h, i) in items"
        :key="h.bookmarkId"
        class="slip"
        :class="'slip-' + ((i % 5) + 1)"
        :style="{ '--delay': i * 90 + 'ms' }"
        :to="`/notes/${h.bookId}`"
      >
        <span class="slip-seal">摘</span>
        <img v-if="h.cover" class="slip-cover" :src="h.cover" alt="" />
        <blockquote class="slip-quote">{{ h.markText }}</blockquote>
        <footer class="slip-meta">
          <div>
            <cite>{{ h.title || '未命名' }}</cite>
            <span v-if="h.author">{{ h.author }}</span>
          </div>
          <time v-if="h.createTime">{{ formatDate(h.createTime) }}</time>
        </footer>
      </RouterLink>
    </div>
  </section>
</template>
