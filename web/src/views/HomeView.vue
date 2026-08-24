<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { fetchOnThisDayHighlights, fetchRandomHighlights, fetchSettings, refreshRandomHighlights } from '../api'
import HighlightFigure from '../highlights/HighlightFigure.vue'
import HighlightLightbox from '../highlights/HighlightLightbox.vue'
import { HIGHLIGHT_DISPLAYS, normalizeHighlightDisplay, type HighlightDisplay } from '../highlights/types'
import type { RandomHighlight } from '../types'

type LightboxKind = 'today' | 'otd'

const pool = ref<RandomHighlight[]>([])
const otdPool = ref<RandomHighlight[]>([])
const loading = ref(false)
const error = ref('')
const displayCount = 5
const drawKey = ref(0)
const otdDrawKey = ref(0)
const display = ref<HighlightDisplay>('card')

const items = computed(() => pool.value.slice(0, displayCount))
const otdItems = computed(() => otdPool.value.slice(0, displayCount))

const focused = ref<number | null>(null)
const lightboxKind = ref<LightboxKind | null>(null)
const lightboxOpen = computed(() => lightboxKind.value != null && focused.value != null)
const todaySlipEls = new Map<number, HTMLElement>()
const otdSlipEls = new Map<number, HTMLElement>()
let lastFocus: HTMLElement | null = null

const activeItems = computed(() => (lightboxKind.value === 'otd' ? otdItems.value : items.value))
const focusedItem = computed(() => {
  if (focused.value == null) return null
  return activeItems.value[focused.value] ?? null
})
const sourceEl = computed(() => {
  if (focused.value == null || lightboxKind.value == null) return null
  const map = lightboxKind.value === 'otd' ? otdSlipEls : todaySlipEls
  return map.get(focused.value) ?? null
})

async function load(refresh = false) {
  if (lightboxKind.value === 'today') onClosed()
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

async function loadOnThisDay() {
  try {
    const data = await fetchOnThisDayHighlights()
    otdPool.value = data.items || []
    otdDrawKey.value += 1
  } catch {
    otdPool.value = []
  }
}

function scatter(i: number, rotSpan: number, xSpan: number, ySpan: number, seedBase: number) {
  const seed = seedBase * 19 + i * 47 + displayCount * 3
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

function tileStyle(i: number, seedBase: number) {
  if (display.value === 'card') return scatter(i, 4, 28, 18, seedBase)
  if (display.value === 'polaroid') return scatter(i, 2, 10, 8, seedBase)
  return { '--delay': `${i * 90}ms`, '--slip-rot': '0deg', '--slip-x': '0px', '--slip-y': '0px' }
}

const displayLabel = computed(
  () => HIGHLIGHT_DISPLAYS.find((x) => x.id === display.value)?.label ?? '藏书票',
)

function setSlipEl(map: Map<number, HTMLElement>, i: number, el: Element | null) {
  if (el instanceof HTMLElement) map.set(i, el)
  else map.delete(i)
}

function openAt(kind: LightboxKind, i: number) {
  if (lightboxKind.value != null) return
  const map = kind === 'otd' ? otdSlipEls : todaySlipEls
  const source = map.get(i)
  if (!source) return
  lastFocus = document.activeElement instanceof HTMLElement ? document.activeElement : source
  focused.value = i
  lightboxKind.value = kind
}

function onClosed() {
  lightboxKind.value = null
  focused.value = null
  lastFocus?.focus()
}

function go(delta: number) {
  if (focused.value == null || !activeItems.value.length) return
  focused.value = (focused.value + delta + activeItems.value.length) % activeItems.value.length
}

onMounted(() => {
  void fetchSettings()
    .then((s) => {
      display.value = normalizeHighlightDisplay(s.highlightDisplay)
    })
    .catch(() => {
      display.value = 'card'
    })
  void load(false)
  void loadOnThisDay()
})

onUnmounted(() => {
  document.body.classList.remove('slip-lightbox-lock')
})
</script>

<template>
  <section class="home" :aria-busy="loading">
    <div v-if="otdItems.length" class="home-otd">
      <header class="home-head">
        <div>
          <p class="home-kicker">On this day · 那年今日</p>
          <h2 class="page-title">那年今日</h2>
          <p class="muted">往年同一天写下的划线，样式与点开交互同「{{ displayLabel }}」</p>
        </div>
      </header>
      <div class="home-spread" :key="otdDrawKey" :data-count="otdItems.length" :data-display="display">
        <button
          v-for="(h, i) in otdItems"
          :key="h.bookmarkId"
          :ref="(el) => setSlipEl(otdSlipEls, i, el as Element | null)"
          class="hl-tile"
          :class="['slip-' + ((i % 5) + 1), { 'is-origin': lightboxKind === 'otd' && focused === i }]"
          :style="tileStyle(i, otdDrawKey)"
          type="button"
          :aria-label="`展开那年今日《${h.title || '未命名'}》的划线`"
          @click="openAt('otd', i)"
        >
          <HighlightFigure :item="h" :display="display" variant="tile" />
        </button>
      </div>
    </div>

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
      :data-count="displayCount"
      :data-display="display"
      aria-hidden="true"
    >
      <div v-for="n in displayCount" :key="n" class="hl-tile skeleton" :class="'slip-' + n">
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
        :ref="(el) => setSlipEl(todaySlipEls, i, el as Element | null)"
        class="hl-tile"
        :class="['slip-' + ((i % 5) + 1), { 'is-origin': lightboxKind === 'today' && focused === i }]"
        :style="tileStyle(i, drawKey)"
        type="button"
        :aria-label="`展开《${h.title || '未命名'}》的划线`"
        @click="openAt('today', i)"
      >
        <HighlightFigure :item="h" :display="display" variant="tile" />
      </button>
    </div>

    <Teleport to="body">
      <HighlightLightbox
        v-if="lightboxOpen && focusedItem"
        :display="display"
        :item="focusedItem"
        :total="activeItems.length"
        :source-el="sourceEl"
        @closed="onClosed"
        @go="go"
      />
    </Teleport>
  </section>
</template>
