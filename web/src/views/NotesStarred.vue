<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { fetchStarredHighlights, setHighlightStarred } from '../api'
import type { RandomHighlight } from '../types'
import NotesSubnav from './NotesSubnav.vue'
import StarMark from './StarMark.vue'

const items = ref<RandomHighlight[]>([])
const loading = ref(false)
const error = ref('')

function formatNoteDate(ts: number) {
  if (!ts) return ''
  const d = new Date(ts * 1000)
  if (Number.isNaN(d.getTime())) return ''
  return `${d.getFullYear()}.${String(d.getMonth() + 1).padStart(2, '0')}.${String(d.getDate()).padStart(2, '0')}`
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const data = await fetchStarredHighlights()
    items.value = data.items || []
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

async function toggleStar(item: RandomHighlight) {
  try {
    await setHighlightStarred(item.bookmarkId, false)
    items.value = items.value.filter((x) => x.bookmarkId !== item.bookmarkId)
  } catch (e) {
    error.value = e instanceof Error ? e.message : '取消星标失败'
  }
}

onMounted(load)
</script>

<template>
  <section :aria-busy="loading">
    <h2 class="page-title">金句</h2>
    <p class="muted">星标过的划线留在这里。同步不会冲掉，删掉原划线后才会消失。</p>
    <NotesSubnav />
    <div v-if="error" class="error" role="alert">{{ error }}</div>
    <p v-else-if="loading" class="muted">正在展开金句…</p>
    <p v-else-if="!items.length" class="empty">还没有星标。在笔记或首页摘抄里点星，句子会出现在这里。</p>
    <div v-else class="search-hits">
      <article v-for="item in items" :key="item.bookmarkId" class="note-card is-mark">
        <header class="note-meta">
          <span class="note-kind">划线</span>
          <span class="note-meta-actions">
            <time v-if="item.createTime">{{ formatNoteDate(item.createTime) }}</time>
            <StarMark :starred="true" @toggle="toggleStar(item)" />
          </span>
        </header>
        <blockquote class="mark-text">{{ item.markText }}</blockquote>
        <RouterLink class="hit-book" :to="`/notes/${item.bookId}#hl-${encodeURIComponent(item.bookmarkId)}`">
          {{ item.title || '未命名' }}
          <span v-if="item.chapterTitle"> · {{ item.chapterTitle }}</span>
          <span v-if="item.author" class="meta"> · {{ item.author }}</span>
        </RouterLink>
      </article>
    </div>
  </section>
</template>
