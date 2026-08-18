<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { fetchShelf } from '../api'
import type { ShelfBook } from '../types'

type Filter = 'all' | 'top' | 'done'

const books = ref<ShelfBook[]>([])
const loading = ref(false)
const error = ref('')
const broken = ref<Record<string, boolean>>({})
const filter = ref<Filter>('all')
const query = ref('')

const visible = computed(() => {
  const q = query.value.trim().toLowerCase()
  return books.value.filter((b) => {
    if (filter.value === 'top' && !b.isTop) return false
    if (filter.value === 'done' && !b.finishReading) return false
    if (!q) return true
    return (b.title || '').toLowerCase().includes(q) || (b.author || '').toLowerCase().includes(q)
  })
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
    <p class="muted">共 {{ books.length }} 本 · 来自本地库</p>
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
        <h3>{{ b.title || '未命名' }}</h3>
        <p class="meta">{{ b.author }}</p>
        <div class="counts">
          <span v-if="b.isTop">置顶</span>
          <span v-if="b.finishReading">读完</span>
          <span>{{ b.readingProgress ?? 0 }}%</span>
        </div>
      </RouterLink>
    </div>
  </section>
</template>
