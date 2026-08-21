<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { fetchRandomHighlights, fetchSettings, refreshRandomHighlights } from '../api'
import HighlightFigure from '../highlights/HighlightFigure.vue'
import HighlightLightbox from '../highlights/HighlightLightbox.vue'
import { HIGHLIGHT_DISPLAYS, normalizeHighlightDisplay, type HighlightDisplay } from '../highlights/types'
import type { RandomHighlight } from '../types'

const pool = ref<RandomHighlight[]>([])
const loading = ref(false)
const error = ref('')
const count = ref(5)
const drawKey = ref(0)
const display = ref<HighlightDisplay>('card')
let media640: MediaQueryList | undefined
let media960: MediaQueryList | undefined

const items = computed(() => pool.value.slice(0, count.value))

const focused = ref<number | null>(null)
const lightboxOpen = ref(false)
const slipEls = new Map<number, HTMLElement>()
let lastFocus: HTMLElement | null = null

const focusedItem = computed(() => (focused.value == null ? null : items.value[focused.value] ?? null))
const sourceEl = computed(() => (focused.value == null ? null : slipEls.get(focused.value) ?? null))

function pickCount() {
  if (window.matchMedia('(max-width: 639px)').matches) return 3
  if (window.matchMedia('(max-width: 959px)').matches) return 4
  return 5
}

async function load(refresh = false) {
  if (lightboxOpen.value) onClosed()
  loading.value = true
  error.value = ''
  try {
    const data = refresh ? await refreshRandomHighlights() : await fetchRandomHighlights()
    pool.value = data.items || []
    drawKey.value += 1
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

function onBreakpoint() {
  count.value = pickCount()
  if (focused.value != null && focused.value >= count.value) {
    focused.value = Math.max(0, count.value - 1)
  }
}

function scatter(i: number, rotSpan: number, xSpan: number, ySpan: number) {
  const seed = drawKey.value * 19 + i * 47 + count.value * 3
  const rot = ((seed % (rotSpan * 20 + 1)) / 10) - rotSpan
  const x = ((seed * 5) % (xSpan * 2 + 1)) - xSpan
  const y = ((seed * 11) % (ySpan * 2 + 1)) - ySpan
  return {
    '--delay': `${i * 90}ms`,
    '--slip-rot': `${rot}deg`,
    '--slip-x': `${x}px`,
    '--slip-y': `${y}px`,
  }
}

function tileStyle(i: number) {
  if (display.value === 'card') return scatter(i, 4, 28, 18)
  if (display.value === 'polaroid') return scatter(i, 2, 10, 8)
  return { '--delay': `${i * 90}ms`, '--slip-rot': '0deg', '--slip-x': '0px', '--slip-y': '0px' }
}

const displayLabel = computed(
  () => HIGHLIGHT_DISPLAYS.find((x) => x.id === display.value)?.label ?? '藏书票',
)

function setSlipEl(i: number, el: Element | null) {
  if (el instanceof HTMLElement) slipEls.set(i, el)
  else slipEls.delete(i)
}

function openAt(i: number) {
  if (lightboxOpen.value) return
  const source = slipEls.get(i)
  if (!source) return
  lastFocus = document.activeElement instanceof HTMLElement ? document.activeElement : source
  focused.value = i
  lightboxOpen.value = true
}

function onClosed() {
  lightboxOpen.value = false
  focused.value = null
  lastFocus?.focus()
}

function go(delta: number) {
  if (focused.value == null || !items.value.length) return
  focused.value = (focused.value + delta + items.value.length) % items.value.length
}

onMounted(() => {
  count.value = pickCount()
  media640 = window.matchMedia('(max-width: 639px)')
  media960 = window.matchMedia('(max-width: 959px)')
  media640.addEventListener('change', onBreakpoint)
  media960.addEventListener('change', onBreakpoint)
  void fetchSettings()
    .then((s) => {
      display.value = normalizeHighlightDisplay(s.highlightDisplay)
    })
    .catch(() => {
      display.value = 'card'
    })
  load(false)
})

onUnmounted(() => {
  media640?.removeEventListener('change', onBreakpoint)
  media960?.removeEventListener('change', onBreakpoint)
  document.body.classList.remove('slip-lightbox-lock')
})
</script>

<template>
  <section class="home" :aria-busy="loading">
    <header class="home-head">
      <div>
        <p class="home-kicker">Commonplace · 灯下抽签</p>
        <h2 class="page-title">今日摘抄</h2>
        <p class="muted">从历史划线里抽出几条，首页与点开均为「{{ displayLabel }}」样式，左右可翻下一张</p>
      </div>
      <button class="btn home-redraw" type="button" :disabled="loading" @click="load(true)">
        {{ loading ? '抽取中…' : '换一批' }}
      </button>
    </header>

    <div v-if="error" class="error" role="alert">{{ error }}</div>

    <div
      v-else-if="loading && !items.length"
      class="home-spread"
      data-count="3"
      :data-display="display"
      aria-hidden="true"
    >
      <div v-for="n in 3" :key="n" class="hl-tile skeleton" :class="'slip-' + n">
        <div class="line" />
        <div class="line" />
        <div class="line short" />
      </div>
    </div>

    <p v-else-if="!items.length" class="home-empty">
      还没有划线。先点右上角同步，再回来抽一张纸。
    </p>

    <div v-else class="home-spread" :key="drawKey" :data-count="items.length" :data-display="display">
      <button
        v-for="(h, i) in items"
        :key="h.bookmarkId"
        :ref="(el) => setSlipEl(i, el as Element | null)"
        class="hl-tile"
        :class="['slip-' + ((i % 5) + 1), { 'is-origin': lightboxOpen && focused === i }]"
        :style="tileStyle(i)"
        type="button"
        :aria-label="`展开《${h.title || '未命名'}》的划线`"
        @click="openAt(i)"
      >
        <HighlightFigure :item="h" :display="display" variant="tile" />
      </button>
    </div>

    <Teleport to="body">
      <HighlightLightbox
        v-if="lightboxOpen && focusedItem"
        :display="display"
        :item="focusedItem"
        :total="items.length"
        :source-el="sourceEl"
        @closed="onClosed"
        @go="go"
      />
    </Teleport>
  </section>
</template>
