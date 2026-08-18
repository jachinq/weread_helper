<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { fetchShelf } from '../api'
import type { ShelfBook } from '../types'

const books = ref<ShelfBook[]>([])
const loading = ref(false)
const error = ref('')
const broken = ref<Record<string, boolean>>({})

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
    <div v-else class="grid">
      <RouterLink v-for="b in books" :key="b.bookId" class="card" :to="`/notes/${b.bookId}`">
        <img
          v-if="b.cover && !broken[b.bookId]"
          :src="b.cover"
          :alt="b.title || '未命名'"
          loading="lazy"
          @error="broken[b.bookId] = true"
        />
        <div v-else class="cover" aria-hidden="true">书</div>
        <div>
          <h3>{{ b.title || '未命名' }}</h3>
          <p class="meta">{{ b.author }}</p>
          <div class="counts">
            <span v-if="b.isTop">置顶</span>
            <span v-if="b.finishReading">读完</span>
            <span>进度 {{ b.readingProgress ?? 0 }}%</span>
          </div>
          <div class="progress" aria-hidden="true">
            <i :style="{ width: Math.min(100, Math.max(0, b.readingProgress ?? 0)) + '%' }" />
          </div>
        </div>
      </RouterLink>
    </div>
  </section>
</template>
