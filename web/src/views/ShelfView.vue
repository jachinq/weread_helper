<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { fetchShelf } from '../api'
import type { ShelfBook } from '../types'

const books = ref<ShelfBook[]>([])
const loading = ref(false)
const error = ref('')

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
  <section>
    <h2 class="page-title">书架</h2>
    <p class="muted">共 {{ books.length }} 本 · 来自本地库</p>
    <div v-if="error" class="error">{{ error }}</div>
    <p v-if="loading && !books.length" class="muted">载入中…</p>
    <div class="grid">
      <RouterLink v-for="b in books" :key="b.bookId" class="card" :to="`/notes/${b.bookId}`">
        <img :src="b.cover" :alt="b.title" />
        <div>
          <h3>{{ b.title || '未命名' }}</h3>
          <p class="meta">{{ b.author }}</p>
          <div class="counts">
            <span v-if="b.isTop">置顶</span>
            <span v-if="b.finishReading">读完</span>
            <span>进度 {{ b.readingProgress ?? 0 }}%</span>
          </div>
        </div>
      </RouterLink>
    </div>
  </section>
</template>
