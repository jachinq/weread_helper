<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { RandomHighlight } from '../types'
import { formatHighlightDate } from './format'
import type { HighlightDisplay } from './types'

defineProps<{
  item: RandomHighlight
  display: HighlightDisplay
  variant: 'tile' | 'focus'
  titleId?: string
}>()
</script>

<template>
  <div v-if="display === 'card'" class="slip" :class="{ 'slip-focus': variant === 'focus' }">
    <span class="slip-seal" aria-hidden="true">摘</span>
    <img v-if="item.cover" class="slip-cover" :src="item.cover" alt="" :loading="variant === 'tile' ? 'lazy' : undefined" />
    <blockquote class="slip-quote">{{ item.markText }}</blockquote>
    <footer class="slip-meta">
      <div>
        <RouterLink
          v-if="variant === 'focus'"
          :id="titleId"
          class="slip-title-link"
          :to="`/notes/${item.bookId}`"
          :aria-label="`查看《${item.title || '未命名'}》的笔记`"
          @pointerdown.stop
        >
          <cite>{{ item.title || '未命名' }}</cite>
        </RouterLink>
        <cite v-else>{{ item.title || '未命名' }}</cite>
        <span v-if="item.author">{{ item.author }}</span>
      </div>
      <div v-if="variant === 'focus'" class="slip-focus-actions">
        <time v-if="item.createTime">{{ formatHighlightDate(item.createTime) }}</time>
        <RouterLink class="slip-notes-link" :to="`/notes/${item.bookId}`">全书笔记</RouterLink>
      </div>
      <time v-else-if="item.createTime">{{ formatHighlightDate(item.createTime) }}</time>
    </footer>
  </div>

  <div v-else-if="display === 'poster'" class="hl-poster" :class="{ 'hl-poster--tile': variant === 'tile' }">
    <img v-if="item.cover" class="hl-poster-bg" :src="item.cover" alt="" :loading="variant === 'tile' ? 'lazy' : undefined" />
    <div class="hl-poster-veil" />
    <p class="hl-poster-kicker">摘抄</p>
    <blockquote class="hl-poster-quote">{{ item.markText }}</blockquote>
    <footer class="hl-poster-meta">
      <RouterLink
        v-if="variant === 'focus'"
        :id="titleId"
        class="slip-title-link"
        :to="`/notes/${item.bookId}`"
        @pointerdown.stop
      >
        <cite>{{ item.title || '未命名' }}</cite>
      </RouterLink>
      <cite v-else>{{ item.title || '未命名' }}</cite>
      <span v-if="item.author">{{ item.author }}</span>
      <time v-if="item.createTime">{{ formatHighlightDate(item.createTime) }}</time>
      <RouterLink v-if="variant === 'focus'" class="slip-notes-link" :to="`/notes/${item.bookId}`">全书笔记</RouterLink>
    </footer>
  </div>

  <div v-else-if="display === 'reader'" class="hl-reader" :class="{ 'hl-reader--tile': variant === 'tile' }">
    <p class="hl-reader-kicker">划线</p>
    <blockquote class="hl-reader-quote">{{ item.markText }}</blockquote>
    <footer class="hl-reader-meta">
      <RouterLink
        v-if="variant === 'focus'"
        :id="titleId"
        class="slip-title-link"
        :to="`/notes/${item.bookId}`"
        @pointerdown.stop
      >
        <cite>{{ item.title || '未命名' }}</cite>
      </RouterLink>
      <cite v-else>{{ item.title || '未命名' }}</cite>
      <span v-if="item.author">{{ item.author }}</span>
      <time v-if="item.createTime">{{ formatHighlightDate(item.createTime) }}</time>
      <RouterLink v-if="variant === 'focus'" class="slip-notes-link" :to="`/notes/${item.bookId}`">全书笔记</RouterLink>
    </footer>
  </div>

  <div v-else-if="display === 'polaroid'" class="hl-polaroid" :class="{ 'hl-polaroid--tile': variant === 'tile' }">
    <div class="hl-polaroid-photo">
      <img v-if="item.cover" :src="item.cover" alt="" :loading="variant === 'tile' ? 'lazy' : undefined" />
      <span v-else class="hl-polaroid-empty">{{ (item.title || '摘').slice(0, 1) }}</span>
    </div>
    <blockquote class="hl-polaroid-quote">{{ item.markText }}</blockquote>
    <footer class="hl-polaroid-meta">
      <RouterLink
        v-if="variant === 'focus'"
        :id="titleId"
        class="slip-title-link"
        :to="`/notes/${item.bookId}`"
        @pointerdown.stop
      >
        <cite>{{ item.title || '未命名' }}</cite>
      </RouterLink>
      <cite v-else>{{ item.title || '未命名' }}</cite>
      <span>{{ item.author }}{{ item.createTime ? ' · ' + formatHighlightDate(item.createTime) : '' }}</span>
      <RouterLink v-if="variant === 'focus'" class="slip-notes-link" :to="`/notes/${item.bookId}`">全书笔记</RouterLink>
    </footer>
  </div>

  <div v-else class="hl-share" :class="{ 'hl-share--tile': variant === 'tile' }">
    <p class="hl-share-mark">摘</p>
    <blockquote class="hl-share-quote">{{ item.markText }}</blockquote>
    <footer class="hl-share-meta">
      <img v-if="item.cover" class="hl-share-cover" :src="item.cover" alt="" :loading="variant === 'tile' ? 'lazy' : undefined" />
      <div>
        <RouterLink
          v-if="variant === 'focus'"
          :id="titleId"
          class="slip-title-link"
          :to="`/notes/${item.bookId}`"
          @pointerdown.stop
        >
          <cite>{{ item.title || '未命名' }}</cite>
        </RouterLink>
        <cite v-else>{{ item.title || '未命名' }}</cite>
        <span v-if="item.author">{{ item.author }}</span>
        <time v-if="item.createTime">{{ formatHighlightDate(item.createTime) }}</time>
      </div>
    </footer>
    <RouterLink v-if="variant === 'focus'" class="slip-notes-link hl-share-notes" :to="`/notes/${item.bookId}`">全书笔记</RouterLink>
  </div>
</template>
