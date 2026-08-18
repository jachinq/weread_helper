<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { fetchShelf } from '../api'
import type { ShelfBook } from '../types'

type Filter = 'all' | 'top' | 'done'

const MONTHS = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12] as const

const books = ref<ShelfBook[]>([])
const loading = ref(false)
const error = ref('')
const broken = ref<Record<string, boolean>>({})
const filter = ref<Filter>('all')
const query = ref('')
const year = ref<number | null>(null)
const month = ref<number | null>(null)

function readDate(b: ShelfBook): Date | null {
  const ts = b.readUpdateTime ?? 0
  if (ts <= 0) return null
  return new Date(ts * 1000)
}

const years = computed(() => {
  const set = new Set<number>()
  for (const b of books.value) {
    const d = readDate(b)
    if (d) set.add(d.getFullYear())
  }
  return [...set].sort((a, b) => b - a)
})

const visible = computed(() => {
  const q = query.value.trim().toLowerCase()
  const y = year.value
  const m = month.value
  return books.value.filter((b) => {
    if (filter.value === 'top' && !b.isTop) return false
    if (filter.value === 'done' && !b.finishReading) return false
    if (y != null) {
      const d = readDate(b)
      if (!d) return false
      if (d.getFullYear() !== y) return false
      if (m != null && d.getMonth() + 1 !== m) return false
    }
    if (!q) return true
    return (b.title || '').toLowerCase().includes(q) || (b.author || '').toLowerCase().includes(q)
  })
})

const countLabel = computed(() => {
  const total = books.value.length
  const shown = visible.value.length
  if (!total || shown === total) return `共 ${total} 本 · 来自本地库`
  return `共 ${total} 本 · 显示 ${shown} 本`
})

watch(year, (y) => {
  if (y == null) month.value = null
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const data = await fetchShelf()
    books.value = data.books || []
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <section :aria-busy="loading">
    <h2 class="page-title">书架</h2>
    <p class="muted">{{ countLabel }}</p>
    <div class="toolbar">
      <label class="sr-only" for="shelf-search">筛选书架</label>
      <input
        id="shelf-search"
        v-model="query"
        class="search"
        type="search"
        placeholder="书名或作者"
        autocomplete="off"
      />
      <label class="sr-only" for="shelf-year">按年筛选</label>
      <select
        id="shelf-year"
        class="date-filter"
        :value="year ?? ''"
        @change="year = ($event.target as HTMLSelectElement).value ? Number(($event.target as HTMLSelectElement).value) : null"
      >
        <option value="">全部年份</option>
        <option v-for="y in years" :key="y" :value="y">{{ y }} 年</option>
      </select>
      <label class="sr-only" for="shelf-month">按月筛选</label>
      <select
        id="shelf-month"
        class="date-filter"
        :disabled="year == null"
        :value="month ?? ''"
        @change="month = ($event.target as HTMLSelectElement).value ? Number(($event.target as HTMLSelectElement).value) : null"
      >
        <option value="">全部月份</option>
        <option v-for="mo in MONTHS" :key="mo" :value="mo">{{ mo }} 月</option>
      </select>
      <div class="filters" role="group" aria-label="书架筛选">
        <button class="chip" type="button" :aria-pressed="filter === 'all'" @click="filter = 'all'">全部</button>
        <button class="chip" type="button" :aria-pressed="filter === 'top'" @click="filter = 'top'">置顶</button>
        <button class="chip" type="button" :aria-pressed="filter === 'done'" @click="filter = 'done'">读完</button>
      </div>
    </div>
    <div v-if="error" class="error" role="alert">{{ error }}</div>
    <div v-if="loading && !books.length" class="grid" aria-hidden="true">
      <div v-for="n in 8" :key="n" class="card skeleton">
        <div class="cover" />
        <div>
          <div class="line" />
          <div class="line short" />
        </div>
      </div>
    </div>
    <p v-else-if="!books.length" class="empty">书架还是空的。先同步，再回来翻一翻。</p>
    <p v-else-if="!visible.length" class="empty">没有符合当前筛选的书。</p>
    <div v-else class="shelf-grid">
      <RouterLink v-for="b in visible" :key="b.bookId" class="spine" :to="`/notes/${b.bookId}`">
        <img
          v-if="b.cover && !broken[b.bookId]"
          :src="b.cover"
          :alt="b.title || '未命名'"
          loading="lazy"
          @error="broken[b.bookId] = true"
        />
        <div v-else class="cover" aria-hidden="true">书</div>
        <h3 :title="b.title || '未命名'">{{ b.title || '未命名' }}</h3>
        <p class="meta" :title="b.author">{{ b.author }}</p>
        <div class="counts">
          <span v-if="b.isTop">置顶</span>
          <span v-if="b.finishReading">读完</span>
          <span>{{ b.readingProgress ?? 0 }}%</span>
        </div>
      </RouterLink>
    </div>
  </section>
</template>
