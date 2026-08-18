<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { fetchNotebooks } from '../api'
import type { NotebookBook } from '../types'

const books = ref<NotebookBook[]>([])
const loading = ref(false)
const error = ref('')
const hasMore = ref(false)
const lastSort = ref(0)
const totals = ref({ books: 0, notes: 0 })

function asNum(v: unknown, fallback = 0) {
  if (typeof v === 'number') return v
  if (typeof v === 'string' && v !== '') return Number(v)
  return fallback
}

function asStr(v: unknown) {
  return typeof v === 'string' ? v : v == null ? '' : String(v)
}

function normalize(item: unknown): NotebookBook | null {
  if (!item || typeof item !== 'object') return null
  const row = item as Record<string, unknown>
  const nested = (row.book && typeof row.book === 'object' ? row.book : {}) as Record<string, unknown>
  const bookId = asStr(row.bookId || nested.bookId)
  if (!bookId) return null
  return {
    bookId,
    title: asStr(nested.title || row.title) || '未命名',
    author: asStr(nested.author || row.author),
    cover: asStr(nested.cover || row.cover),
    reviewCount: asNum(row.reviewCount),
    noteCount: asNum(row.noteCount),
    bookmarkCount: asNum(row.bookmarkCount),
    readingProgress: asNum(row.readingProgress ?? nested.readingProgress),
    sort: asNum(row.sort),
  }
}

async function load(reset = false) {
  loading.value = true
  error.value = ''
  try {
    const data = await fetchNotebooks(40, reset ? undefined : lastSort.value || undefined)
    const list = (data.books || []).map(normalize).filter((x): x is NotebookBook => !!x)
    books.value = reset ? list : [...books.value, ...list]
    hasMore.value = asNum(data.hasMore) === 1
    totals.value = {
      books: asNum(data.totalBookCount, books.value.length),
      notes: asNum(data.totalNoteCount),
    }
    const last = list[list.length - 1]
    lastSort.value = last?.sort || 0
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(() => load(true))
</script>

<template>
  <section>
    <h2 class="page-title">有笔记的书</h2>
    <p class="muted">
      共 {{ totals.books || books.length }} 本 · 笔记约 {{ totals.notes || '—' }} 条
    </p>
    <div v-if="error" class="error">{{ error }}</div>
    <div class="grid">
      <RouterLink
        v-for="b in books"
        :key="b.bookId"
        class="card"
        :to="`/notes/${b.bookId}`"
      >
        <img :src="b.cover" :alt="b.title" />
        <div>
          <h3>{{ b.title }}</h3>
          <p class="meta">{{ b.author }}</p>
          <div class="counts">
            <span>划线 {{ b.noteCount }}</span>
            <span>想法 {{ b.reviewCount }}</span>
            <span>进度 {{ b.readingProgress }}%</span>
          </div>
        </div>
      </RouterLink>
    </div>
    <div class="load-more">
      <button class="btn" :disabled="loading || !hasMore" @click="load(false)">
        {{ loading ? '载入中…' : hasMore ? '加载更多' : '已全部加载' }}
      </button>
    </div>
  </section>
</template>
