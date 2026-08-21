<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import type { RandomHighlight } from '../types'
import HighlightFigure from './HighlightFigure.vue'
import type { HighlightDisplay } from './types'

const props = defineProps<{
  display: HighlightDisplay
  item: RandomHighlight
  total: number
  sourceEl: HTMLElement | null
}>()

const emit = defineEmits<{
  closed: []
  go: [delta: number]
}>()

const stageEl = ref<HTMLElement | null>(null)
const closeBtn = ref<HTMLButtonElement | null>(null)
const motionLock = ref(false)
const slideName = ref('slip-slide-next')
const allowSlide = ref(false)
let swipeX = 0
let swipeActive = false
let swipeMoved = false

function prefersReducedMotion() {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
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
  return el
    .animate(
      [
        { transform: `translate(${dx}px, ${dy}px) scale(${sx}, ${sy})` },
        { transform: 'none' },
      ],
      { duration, easing, fill: 'both' },
    )
    .finished
}

async function playOpen() {
  await nextTick()
  closeBtn.value?.focus()
  const dest = stageEl.value
  const source = props.sourceEl
  if (props.display !== 'card' || !dest || !source || prefersReducedMotion()) {
    allowSlide.value = true
    return
  }
  motionLock.value = true
  dest.style.visibility = 'visible'
  const from = source.getBoundingClientRect()
  const to = dest.getBoundingClientRect()
  try {
    await flipFromTo(dest, from, to, 520, easeOpen())
  } catch {
    /* cancelled */
  } finally {
    dest.getAnimations().forEach((a) => a.cancel())
    motionLock.value = false
    allowSlide.value = true
  }
}

async function requestClose() {
  if (motionLock.value) return
  const dest = stageEl.value
  const source = props.sourceEl
  if (props.display === 'card' && dest && source && !prefersReducedMotion()) {
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
  emit('closed')
}

function go(delta: number) {
  if (props.total < 2 || motionLock.value) return
  slideName.value = delta > 0 ? 'slip-slide-next' : 'slip-slide-prev'
  emit('go', delta)
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    void requestClose()
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
  document.body.classList.add('slip-lightbox-lock')
  window.addEventListener('keydown', onKey)
  void playOpen()
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKey)
  document.body.classList.remove('slip-lightbox-lock')
})

watch(
  () => props.item.bookmarkId,
  () => {
    allowSlide.value = true
  },
)
</script>

<template>
  <div class="slip-lightbox" :data-display="display" role="dialog" aria-modal="true" aria-labelledby="slip-focus-title">
    <button class="slip-lightbox-scrim" type="button" aria-label="关闭摘抄" @click="requestClose" />
    <button ref="closeBtn" class="slip-lightbox-close" type="button" aria-label="关闭" @click="requestClose">
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M6 6l12 12M18 6 6 18" fill="none" stroke="currentColor" stroke-width="1.8" />
      </svg>
    </button>
    <button class="slip-lightbox-nav prev" type="button" aria-label="上一张" :disabled="total < 2" @click="go(-1)">
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M15 5 8 12l7 7" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round" />
      </svg>
    </button>
    <button class="slip-lightbox-nav next" type="button" aria-label="下一张" :disabled="total < 2" @click="go(1)">
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
        <div :key="display + '-' + item.bookmarkId" class="hl-focus-frame">
          <HighlightFigure :item="item" :display="display" variant="focus" title-id="slip-focus-title" />
        </div>
      </Transition>
    </div>
    <p class="slip-lightbox-hint">左右滑动或用方向键切换 · Esc 关闭</p>
  </div>
</template>
