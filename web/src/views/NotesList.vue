<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { fetchNotebooks } from '../api'
import type { NotebookBook } from '../types'

const books = ref<NotebookBook[]>([])
const loading = ref(false)
const error = ref('')
const hasMore = ref(false)
const lastSort = ref(0)
const totals = ref({ books: 0, notes: 0 })
const broken = ref<Record<string, boolean>>({})

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
  <section :aria-busy="loading">
    <h2 class="page-title">有笔记的书</h2>
    <p class="muted">
      共 {{ totals.books || books.length }} 本 · 笔记约 {{ totals.notes || '—' }} 条
    </p>
    <div v-if="error" class="error" role="alert">{{ error }}</div>
    <div v-if="loading && !books.length" class="grid" aria-hidden="true">
      <div v-for="n in 6" :key="n" class="card skeleton">
        <div class="cover" />
        <div>
          <div class="line" />
          <div class="line short" />
        </div>
      </div>
    </div>
    <p v-else-if="!books.length" class="empty">还没有笔记。点右上角同步后，划线会出现在这里。</p>
    <div v-else class="grid">
      <RouterLink
        v-for="b in books"
        :key="b.bookId"
        class="card"
        :to="`/notes/${b.bookId}`"
      >
        <img
          v-if="b.cover && !broken[b.bookId]"
          :src="b.cover"
          :alt="b.title"
          loading="lazy"
          @error="broken[b.bookId] = true"
        />
        <div v-else class="cover" aria-hidden="true">书</div>
        <div>
          <h3>{{ b.title }}</h3>
          <p class="meta">{{ b.author }}</p>
          <div class="counts">
            <span>划线 {{ b.noteCount }}</span>
            <span>想法 {{ b.reviewCount }}</span>
            <span>进度 {{ b.readingProgress }}%</span>
          </div>
          <div class="progress" aria-hidden="true">
            <i :style="{ width: Math.min(100, Math.max(0, b.readingProgress)) + '%' }" />
          </div>
        </div>
      </RouterLink>
    </div>
    <div v-if="books.length" class="load-more">
      <button class="btn" type="button" :disabled="loading || !hasMore" @click="load(false)">
        {{ loading ? '载入中…' : hasMore ? '加载更多' : '已全部加载' }}
      </button>
    </div>
  </section>
</template>
