<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { proxiedCover } from '../cover'
import type { RandomHighlight } from '../types'
import { formatHighlightDate } from './format'
import type { HighlightDisplay } from './types'

defineProps<{
  item: RandomHighlight
  display: HighlightDisplay
  variant: 'tile' | 'focus'
  titleId?: string
}>()

function displayTitle(title: string | undefined, variant: 'tile' | 'focus') {
  const t = title?.trim() || '未命名'
  if (variant !== 'focus') return t
  if (t.startsWith('《') && t.endsWith('》')) return t
  return `《${t}》`
}
</script>

<template>
  <div v-if="display === 'card'" class="slip" :class="{ 'slip-focus': variant === 'focus' }">
    <span class="slip-seal" aria-hidden="true">摘</span>
    <img v-if="item.cover" class="slip-cover" :src="proxiedCover(item.cover)" alt="" :loading="variant === 'tile' ? 'lazy' : undefined" />
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
          <cite>{{ displayTitle(item.title, variant) }}</cite>
        </RouterLink>
        <cite v-else>{{ displayTitle(item.title, variant) }}</cite>
        <span v-if="item.author">{{ item.author }}</span>
      </div>
      <time v-if="item.createTime">{{ formatHighlightDate(item.createTime) }}</time>
    </footer>
  </div>

  <div v-else-if="display === 'poster'" class="hl-poster" :class="{ 'hl-poster--tile': variant === 'tile' }">
    <img v-if="item.cover" class="hl-poster-bg" :src="proxiedCover(item.cover)" alt="" :loading="variant === 'tile' ? 'lazy' : undefined" />
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
        <cite>{{ displayTitle(item.title, variant) }}</cite>
      </RouterLink>
      <cite v-else>{{ displayTitle(item.title, variant) }}</cite>
      <span v-if="item.author">{{ item.author }}</span>
      <time v-if="item.createTime">{{ formatHighlightDate(item.createTime) }}</time>
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
        <cite>{{ displayTitle(item.title, variant) }}</cite>
      </RouterLink>
      <cite v-else>{{ displayTitle(item.title, variant) }}</cite>
      <span v-if="item.author">{{ item.author }}</span>
      <time v-if="item.createTime">{{ formatHighlightDate(item.createTime) }}</time>
    </footer>
  </div>

  <div v-else-if="display === 'polaroid'" class="hl-polaroid" :class="{ 'hl-polaroid--tile': variant === 'tile' }">
    <div class="hl-polaroid-photo">
      <img v-if="item.cover" :src="proxiedCover(item.cover)" alt="" :loading="variant === 'tile' ? 'lazy' : undefined" />
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
        <cite>{{ displayTitle(item.title, variant) }}</cite>
      </RouterLink>
      <cite v-else>{{ displayTitle(item.title, variant) }}</cite>
      <span>{{ item.author }}{{ item.createTime ? ' · ' + formatHighlightDate(item.createTime) : '' }}</span>
    </footer>
  </div>

  <div v-else class="hl-share" :class="{ 'hl-share--tile': variant === 'tile' }">
    <p class="hl-share-mark">摘</p>
    <blockquote class="hl-share-quote">{{ item.markText }}</blockquote>
    <footer class="hl-share-meta">
      <img v-if="item.cover" class="hl-share-cover" :src="proxiedCover(item.cover)" alt="" :loading="variant === 'tile' ? 'lazy' : undefined" />
      <div>
        <RouterLink
          v-if="variant === 'focus'"
          :id="titleId"
          class="slip-title-link"
          :to="`/notes/${item.bookId}`"
          @pointerdown.stop
        >
          <cite>{{ displayTitle(item.title, variant) }}</cite>
        </RouterLink>
        <cite v-else>{{ displayTitle(item.title, variant) }}</cite>
        <span v-if="item.author">{{ item.author }}</span>
        <time v-if="item.createTime">{{ formatHighlightDate(item.createTime) }}</time>
      </div>
    </footer>
  </div>
</template>
