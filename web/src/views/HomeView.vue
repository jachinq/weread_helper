<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { fetchRandomHighlights, refreshRandomHighlights } from '../api'
import type { RandomHighlight } from '../types'

const pool = ref<RandomHighlight[]>([])
const loading = ref(false)
const error = ref('')
const count = ref(5)
const drawKey = ref(0)
let media640: MediaQueryList | undefined
let media960: MediaQueryList | undefined

const items = computed(() => pool.value.slice(0, count.value))

const focused = ref<number | null>(null)
const lightboxOpen = ref(false)
const motionLock = ref(false)
const slideName = ref('slip-slide-next')
const allowSlide = ref(false)
const stageEl = ref<HTMLElement | null>(null)
const closeBtn = ref<HTMLButtonElement | null>(null)
const slipEls = new Map<number, HTMLElement>()
let lastFocus: HTMLElement | null = null
let swipeX = 0
let swipeActive = false
let swipeMoved = false

const focusedItem = computed(() => (focused.value == null ? null : items.value[focused.value] ?? null))

function prefersReducedMotion() {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function pickCount() {
  if (window.matchMedia('(max-width: 639px)').matches) return 3
  if (window.matchMedia('(max-width: 959px)').matches) return 4
  return 5
}

async function load(refresh = false) {
  if (lightboxOpen.value) await closeLightbox()
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

function slipScatter(i: number) {
  const seed = drawKey.value * 19 + i * 47 + count.value * 3
  const rot = ((seed % 81) / 10) - 4
  const x = ((seed * 5) % 57) - 28
  const y = ((seed * 11) % 73) - 18
  return {
    '--delay': `${i * 90}ms`,
    '--slip-rot': `${rot}deg`,
    '--slip-x': `${x}px`,
    '--slip-y': `${y}px`,
  }
}

function formatDate(ts: number) {
  if (!ts) return ''
  const d = new Date(ts * 1000)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}.${m}.${day}`
}

function setSlipEl(i: number, el: Element | null) {
  if (el instanceof HTMLElement) slipEls.set(i, el)
  else slipEls.delete(i)
}

function easeOpen() {
  return 'cubic-bezier(0.22, 1, 0.36, 1)'
}

function easeClose() {
  return 'cubic-bezier(0.4, 0, 0.2, 1)'
}

function flipFromTo(el: HTMLElement, from: DOMRect, to: DOMRect, duration: number, easing: string) {
  const dx = from.left + from.width / 2 - (to.left + to.width / 2)
  const dy = from.top + from.height / 2 - (to.top + to.height / 2)
  const sx = from.width / to.width
  const sy = from.height / to.height
  return el.animate(
    [
      { transform: `translate(${dx}px, ${dy}px) scale(${sx}, ${sy})` },
      { transform: 'none' },
    ],
    { duration, easing, fill: 'both' },
  ).finished
}

async function openAt(i: number) {
  if (motionLock.value || lightboxOpen.value) return
  const source = slipEls.get(i)
  if (!source) return
  lastFocus = document.activeElement instanceof HTMLElement ? document.activeElement : source
  const from = source.getBoundingClientRect()
  allowSlide.value = false
  focused.value = i
  lightboxOpen.value = true
  document.body.classList.add('slip-lightbox-lock')
  await nextTick()
  closeBtn.value?.focus()
  const dest = stageEl.value
  if (!dest || prefersReducedMotion()) {
    allowSlide.value = true
    return
  }
  motionLock.value = true
  dest.style.visibility = 'visible'
  const to = dest.getBoundingClientRect()
  try {
    await flipFromTo(dest, from, to, 520, easeOpen())
  } catch {
    /* animation cancelled */
  } finally {
    dest.getAnimations().forEach((a) => a.cancel())
    motionLock.value = false
    allowSlide.value = true
  }
}

async function closeLightbox() {
  if (!lightboxOpen.value || motionLock.value) return
  const i = focused.value
  const dest = stageEl.value
  const source = i == null ? undefined : slipEls.get(i)
  if (dest && source && !prefersReducedMotion()) {
    motionLock.value = true
    const from = dest.getBoundingClientRect()
    const to = source.getBoundingClientRect()
    try {
      await dest.animate(
        [
          { transform: 'none' },
          {
            transform: `translate(${to.left + to.width / 2 - (from.left + from.width / 2)}px, ${
              to.top + to.height / 2 - (from.top + from.height / 2)
            }px) scale(${to.width / from.width}, ${to.height / from.height})`,
          },
        ],
        { duration: 420, easing: easeClose(), fill: 'both' },
      ).finished
    } catch {
      /* cancelled */
    } finally {
      dest.getAnimations().forEach((a) => a.cancel())
      motionLock.value = false
    }
  }
  lightboxOpen.value = false
  focused.value = null
  document.body.classList.remove('slip-lightbox-lock')
  lastFocus?.focus()
}

function go(delta: number) {
  if (focused.value == null || !items.value.length || motionLock.value) return
  const next = (focused.value + delta + items.value.length) % items.value.length
  slideName.value = delta > 0 ? 'slip-slide-next' : 'slip-slide-prev'
  focused.value = next
}

function onKey(e: KeyboardEvent) {
  if (!lightboxOpen.value) return
  if (e.key === 'Escape') {
    e.preventDefault()
    closeLightbox()
  } else if (e.key === 'ArrowRight') {
    e.preventDefault()
    go(1)
  } else if (e.key === 'ArrowLeft') {
    e.preventDefault()
    go(-1)
  }
}

function onPointerDown(e: PointerEvent) {
  if (e.pointerType === 'mouse' && e.button !== 0) return
  swipeActive = true
  swipeMoved = false
  swipeX = e.clientX
}

function onPointerMove(e: PointerEvent) {
  if (!swipeActive) return
  if (Math.abs(e.clientX - swipeX) > 12) swipeMoved = true
}

function onPointerUp(e: PointerEvent) {
  if (!swipeActive) return
  swipeActive = false
  const dx = e.clientX - swipeX
  if (swipeMoved && Math.abs(dx) > 56) go(dx < 0 ? 1 : -1)
}

onMounted(() => {
  count.value = pickCount()
  media640 = window.matchMedia('(max-width: 639px)')
  media960 = window.matchMedia('(max-width: 959px)')
  media640.addEventListener('change', onBreakpoint)
  media960.addEventListener('change', onBreakpoint)
  window.addEventListener('keydown', onKey)
  load(false)
})

onUnmounted(() => {
  media640?.removeEventListener('change', onBreakpoint)
  media960?.removeEventListener('change', onBreakpoint)
  window.removeEventListener('keydown', onKey)
  document.body.classList.remove('slip-lightbox-lock')
})
</script>

<template>
  <section class="home" :aria-busy="loading">
    <header class="home-head">
      <div>
        <p class="home-kicker">Commonplace · 灯下抽签</p>
        <h2 class="page-title">今日摘抄</h2>
        <p class="muted">从历史划线里抽出几张藏书票，点开可细读，左右翻下一张</p>
      </div>
      <button class="btn home-redraw" type="button" :disabled="loading" @click="load(true)">
        {{ loading ? '抽取中…' : '换一批' }}
      </button>
    </header>

    <div v-if="error" class="error" role="alert">{{ error }}</div>

    <div v-else-if="loading && !items.length" class="home-spread" data-count="3" aria-hidden="true">
      <div v-for="n in 3" :key="n" class="slip skeleton" :class="'slip-' + n">
        <div class="line" />
        <div class="line" />
        <div class="line short" />
      </div>
    </div>

    <p v-else-if="!items.length" class="home-empty">
      还没有划线。先点右上角同步，再回来抽一张纸。
    </p>

    <div v-else class="home-spread" :key="drawKey" :data-count="items.length">
      <button
        v-for="(h, i) in items"
        :key="h.bookmarkId"
        :ref="(el) => setSlipEl(i, el as Element | null)"
        class="slip"
        :class="['slip-' + ((i % 5) + 1), { 'is-origin': lightboxOpen && focused === i }]"
        :style="slipScatter(i)"
        type="button"
        :aria-label="`展开《${h.title || '未命名'}》的划线`"
        @click="openAt(i)"
      >
        <span class="slip-seal" aria-hidden="true">摘</span>
        <img v-if="h.cover" class="slip-cover" :src="h.cover" alt="" loading="lazy" />
        <blockquote class="slip-quote">{{ h.markText }}</blockquote>
        <footer class="slip-meta">
          <div>
            <cite>{{ h.title || '未命名' }}</cite>
            <span v-if="h.author">{{ h.author }}</span>
          </div>
          <time v-if="h.createTime">{{ formatDate(h.createTime) }}</time>
        </footer>
      </button>
    </div>

    <Teleport to="body">
      <div
        v-if="lightboxOpen && focusedItem"
        class="slip-lightbox"
        role="dialog"
        aria-modal="true"
        aria-labelledby="slip-focus-title"
      >
        <button class="slip-lightbox-scrim" type="button" aria-label="关闭摘抄" @click="closeLightbox" />
        <button
          ref="closeBtn"
          class="slip-lightbox-close"
          type="button"
          aria-label="关闭"
          @click="closeLightbox"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M6 6l12 12M18 6 6 18" fill="none" stroke="currentColor" stroke-width="1.8" />
          </svg>
        </button>
        <button
          class="slip-lightbox-nav prev"
          type="button"
          aria-label="上一张"
          :disabled="items.length < 2"
          @click="go(-1)"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M15 5 8 12l7 7" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round" />
          </svg>
        </button>
        <button
          class="slip-lightbox-nav next"
          type="button"
          aria-label="下一张"
          :disabled="items.length < 2"
          @click="go(1)"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="m9 5 7 7-7 7" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round" />
          </svg>
        </button>

        <div
          ref="stageEl"
          class="slip-stage"
          @pointerdown="onPointerDown"
          @pointermove="onPointerMove"
          @pointerup="onPointerUp"
          @pointercancel="onPointerUp"
        >
          <Transition :name="allowSlide ? slideName : undefined">
            <article :key="focusedItem.bookmarkId" class="slip slip-focus">
              <span class="slip-seal" aria-hidden="true">摘</span>
              <img v-if="focusedItem.cover" class="slip-cover" :src="focusedItem.cover" alt="" />
              <blockquote class="slip-quote">{{ focusedItem.markText }}</blockquote>
              <footer class="slip-meta">
                <div>
                  <RouterLink
                    id="slip-focus-title"
                    class="slip-title-link"
                    :to="`/notes/${focusedItem.bookId}`"
                    :aria-label="`查看《${focusedItem.title || '未命名'}》的笔记`"
                    @pointerdown.stop
                  >
                    <cite>{{ focusedItem.title || '未命名' }}</cite>
                  </RouterLink>
                  <span v-if="focusedItem.author">{{ focusedItem.author }}</span>
                </div>
                <div class="slip-focus-actions">
                  <time v-if="focusedItem.createTime">{{ formatDate(focusedItem.createTime) }}</time>
                  <RouterLink class="slip-notes-link" :to="`/notes/${focusedItem.bookId}`">全书笔记</RouterLink>
                </div>
              </footer>
            </article>
          </Transition>
        </div>
        <p class="slip-lightbox-hint">左右滑动或用方向键切换 · Esc 关闭</p>
      </div>
    </Teleport>
  </section>
</template>
