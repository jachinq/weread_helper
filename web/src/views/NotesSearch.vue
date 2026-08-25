<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { searchNotes, setHighlightStarred } from '../api'
import type { NoteSearchHit } from '../types'
import NotesSubnav from './NotesSubnav.vue'
import StarMark from './StarMark.vue'

const route = useRoute()
const router = useRouter()
const query = ref('')
const kind = ref('all')
const items = ref<NoteSearchHit[]>([])
const loading = ref(false)
const error = ref('')
const searched = ref(false)
let debounce: ReturnType<typeof setTimeout> | undefined
let seq = 0

function formatNoteDate(ts: number) {
  if (!ts) return ''
  const d = new Date(ts * 1000)
  if (Number.isNaN(d.getTime())) return ''
  return `${d.getFullYear()}.${String(d.getMonth() + 1).padStart(2, '0')}.${String(d.getDate()).padStart(2, '0')}`
}

function hitLink(hit: NoteSearchHit) {
  if (hit.kind === 'highlight' && hit.bookmarkId) {
    return `/notes/${hit.bookId}#hl-${encodeURIComponent(hit.bookmarkId)}`
  }
  if (hit.chapterUid) return `/notes/${hit.bookId}#ch-${hit.chapterUid}`
  return `/notes/${hit.bookId}`
}

async function run() {
  const q = query.value.trim()
  const ticket = ++seq
  if (!q) {
    items.value = []
    searched.value = false
    error.value = ''
    return
  }
  loading.value = true
  error.value = ''
  searched.value = true
  try {
    const data = await searchNotes(q, kind.value, 40)
    if (ticket !== seq) return
    items.value = (data.items || []).slice().sort((a, b) => (b.createTime || 0) - (a.createTime || 0))
  } catch (e) {
    if (ticket !== seq) return
    error.value = e instanceof Error ? e.message : '检索失败'
    items.value = []
  } finally {
    if (ticket === seq) loading.value = false
  }
}

function syncQuery() {
  const q = typeof route.query.q === 'string' ? route.query.q : ''
  const k = typeof route.query.kind === 'string' ? route.query.kind : 'all'
  query.value = q
  kind.value = k === 'highlight' || k === 'review' ? k : 'all'
  void run()
}

watch(query, () => {
  clearTimeout(debounce)
  debounce = setTimeout(() => {
    router.replace({
      query: {
        ...route.query,
        q: query.value.trim() || undefined,
        kind: kind.value === 'all' ? undefined : kind.value,
      },
    })
    void run()
  }, 280)
})

function setKind(next: string) {
  kind.value = next
  router.replace({
    query: {
      ...route.query,
      q: query.value.trim() || undefined,
      kind: next === 'all' ? undefined : next,
    },
  })
  void run()
}

async function toggleStar(hit: NoteSearchHit) {
  if (hit.kind !== 'highlight' || !hit.bookmarkId) return
  const next = !hit.starred
  try {
    await setHighlightStarred(hit.bookmarkId, next)
    hit.starred = next
  } catch (e) {
    error.value = e instanceof Error ? e.message : '星标失败'
  }
}

onMounted(syncQuery)
</script>

<template>
  <section :aria-busy="loading">
    <h2 class="page-title">检索划线与想法</h2>
    <p class="muted">在本地库里找句子，不请求微信读书。点进结果会跳到原书对应章。</p>
    <NotesSubnav />
    <div class="toolbar">
      <label class="sr-only" for="note-fts">检索原文或想法</label>
      <input
        id="note-fts"
        v-model="query"
        class="search"
        type="search"
        placeholder="一句划线、一条想法、书名或作者"
        autocomplete="off"
      />
    </div>
    <div class="modes kind-filter" role="tablist" aria-label="检索范围">
      <button class="btn" type="button" :aria-pressed="kind === 'all'" @click="setKind('all')">全部</button>
      <button class="btn" type="button" :aria-pressed="kind === 'highlight'" @click="setKind('highlight')">划线</button>
      <button class="btn" type="button" :aria-pressed="kind === 'review'" @click="setKind('review')">想法</button>
    </div>
    <div v-if="error" class="error" role="alert">{{ error }}</div>
    <p v-else-if="loading" class="muted">正在翻找…</p>
    <p v-else-if="!query.trim()" class="empty">输入几个字，从全部历史划线和想法里找。</p>
    <p v-else-if="searched && !items.length" class="empty">没有匹配「{{ query.trim() }}」的句子。</p>
    <div v-else class="search-hits">
      <article
        v-for="hit in items"
        :key="(hit.kind === 'highlight' ? hit.bookmarkId : hit.reviewId) || hit.bookId + hit.createTime"
        class="note-card"
        :class="hit.kind === 'highlight' ? 'is-mark' : 'is-idea'"
      >
        <header class="note-meta">
          <span class="note-kind" :class="{ idea: hit.kind === 'review' }">{{ hit.kind === 'review' ? '想法' : '划线' }}</span>
          <span class="note-meta-actions">
            <time v-if="hit.createTime">{{ formatNoteDate(hit.createTime) }}</time>
            <StarMark
              v-if="hit.kind === 'highlight' && hit.bookmarkId"
              :starred="!!hit.starred"
              @toggle="toggleStar(hit)"
            />
          </span>
        </header>
        <blockquote v-if="hit.kind === 'highlight'" class="mark-text">{{ hit.markText }}</blockquote>
        <template v-else>
          <blockquote v-if="hit.abstract" class="idea-abstract">
            <p>{{ hit.abstract }}</p>
          </blockquote>
          <p class="idea-body">{{ hit.content }}</p>
        </template>
        <RouterLink class="hit-book" :to="hitLink(hit)">
          {{ hit.title || '未命名' }}
          <span v-if="hit.chapterTitle"> · {{ hit.chapterTitle }}</span>
          <span v-if="hit.author" class="meta"> · {{ hit.author }}</span>
        </RouterLink>
      </article>
    </div>
  </section>
</template>
