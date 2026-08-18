<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { fetchNotes } from '../api'
import type { NotesResponse } from '../types'

const route = useRoute()
const loading = ref(true)
const error = ref('')
const data = ref<NotesResponse | null>(null)
const coverBroken = ref(false)

const book = computed(() => data.value?.book || {})
const title = computed(() => String(book.value['title'] || book.value['bookId'] || '笔记详情'))
const author = computed(() => String(book.value['author'] || ''))
const cover = computed(() => String(book.value['cover'] || ''))

onMounted(async () => {
  try {
    data.value = await fetchNotes(String(route.params.bookId))
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <section :aria-busy="loading">
    <RouterLink class="back" to="/notes">返回书单</RouterLink>
    <div v-if="error" class="error" role="alert">{{ error }}</div>
    <p v-if="loading" class="muted">正在展开书页…</p>
    <template v-else-if="data">
      <div class="book-hero">
        <img
          v-if="cover && !coverBroken"
          :src="cover"
          :alt="title"
          @error="coverBroken = true"
        />
        <div v-else class="cover" aria-hidden="true">书</div>
        <div>
          <h2 class="page-title">{{ title }}</h2>
          <p class="muted">{{ author }} · {{ data.chapters.length }} 个有笔记的章节</p>
        </div>
      </div>
      <article v-for="ch in data.chapters" :key="ch.chapterUid" class="chapter">
        <h3>{{ ch.title }}</h3>
        <p v-for="h in ch.highlights" :key="h.bookmarkId" class="quote">{{ h.markText }}</p>
        <div v-for="r in ch.reviews" :key="r.reviewId" class="thought">
          <p v-if="r.abstract" class="quote">{{ r.abstract }}</p>
          <p>{{ r.content }}</p>
        </div>
      </article>
      <p v-if="data.chapters.length === 0" class="empty">这本书还没有可展示的划线或想法。</p>
    </template>
  </section>
</template>
