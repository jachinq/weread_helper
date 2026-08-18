<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { fetchNotes } from '../api'
import type { ChapterNotes, Highlight, NotesResponse, Review } from '../types'

type NoteBlock =
  | { kind: 'highlight'; key: string; highlight: Highlight; thoughts: Review[] }
  | { kind: 'thought'; key: string; review: Review }

const route = useRoute()
const loading = ref(true)
const error = ref('')
const data = ref<NotesResponse | null>(null)
const coverBroken = ref(false)
const currentUid = ref<number | null>(null)
const flashUid = ref<number | null>(null)
const cardOpen = ref(false)
const cardReady = ref(false)
const heroCover = ref<HTMLElement | null>(null)
const sheetEl = ref<HTMLElement | null>(null)
let flashTimer: number | undefined
let scrollRaf = 0
let cardClosing = false

function prefersReduce() {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function headerOffset() {
  const header = parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--header-h'))
  return (Number.isFinite(header) ? header : 140) + 20
}

function animateScroll(scroller: HTMLElement | Window, to: number, duration = 640): Promise<void> {
  const getTop = () => (scroller instanceof Window ? window.scrollY : scroller.scrollTop)
  const setTop = (value: number) => {
    if (scroller instanceof Window) window.scrollTo(0, value)
    else scroller.scrollTop = value
  }
  const max =
    scroller instanceof Window
      ? Math.max(0, document.documentElement.scrollHeight - window.innerHeight)
      : Math.max(0, scroller.scrollHeight - scroller.clientHeight)
  const target = Math.max(0, Math.min(to, max))
  if (prefersReduce()) {
    setTop(target)
    return Promise.resolve()
  }
  const from = getTop()
  const delta = target - from
  if (Math.abs(delta) < 1) return Promise.resolve()
  if (scrollRaf) cancelAnimationFrame(scrollRaf)
  return new Promise((resolve) => {
    const start = performance.now()
    const ease = (t: number) => 1 - (1 - t) ** 3
    const frame = (now: number) => {
      const p = Math.min(1, (now - start) / duration)
      setTop(from + delta * ease(p))
      if (p < 1) scrollRaf = requestAnimationFrame(frame)
      else resolve()
    }
    scrollRaf = requestAnimationFrame(frame)
  })
}

function flashChapter(uid: number) {
  flashUid.value = null
  window.requestAnimationFrame(() => {
    flashUid.value = uid
  })
  if (flashTimer) window.clearTimeout(flashTimer)
  flashTimer = window.setTimeout(() => {
    if (flashUid.value === uid) flashUid.value = null
  }, 1400)
}

async function scrollToChapter(uid: number) {
  currentUid.value = uid
  flashUid.value = null
  if (flashTimer) window.clearTimeout(flashTimer)
  await nextTick()
  const el = document.getElementById(`ch-${uid}`)
  if (!el) {
    flashChapter(uid)
    return
  }
  await animateScroll(window, window.scrollY + el.getBoundingClientRect().top - headerOffset())
  if (currentUid.value === uid) flashChapter(uid)
}

function onTocClick(_event: MouseEvent, uid: number) {
  scrollToChapter(uid)
  const url = `${window.location.pathname}${window.location.search}#ch-${uid}`
  history.replaceState(history.state, '', url)
}

onUnmounted(() => {
  if (flashTimer) window.clearTimeout(flashTimer)
  if (scrollRaf) cancelAnimationFrame(scrollRaf)
  lockScroll(false)
  window.removeEventListener('keydown', onOverlayKey)
})

const book = computed(() => data.value?.book || {})
const title = computed(() => String(book.value['title'] || book.value['bookId'] || '笔记详情'))
const author = computed(() => String(book.value['author'] || ''))
const cover = computed(() => String(book.value['cover'] || ''))
const intro = computed(() => String(book.value['intro'] || '').trim())
const publisher = computed(() => String(book.value['publisher'] || '').trim())
const category = computed(() => String(book.value['category'] || '').trim())

function wereadUrl() {
  const raw = book.value['deepLink']
  if (typeof raw === 'string' && /^https?:\/\//i.test(raw)) return raw
  const id = String(route.params.bookId || book.value['bookId'] || '')
  return `https://weread.qq.com/web/bookDetail/${encodeURIComponent(id)}`
}

function lockScroll(lock: boolean) {
  document.documentElement.style.overflow = lock ? 'hidden' : ''
}

function flipFromHero() {
  const from = heroCover.value
  const to = sheetEl.value
  if (!from || !to || prefersReduce()) {
    cardReady.value = true
    return
  }
  const a = from.getBoundingClientRect()
  const b = to.getBoundingClientRect()
  const dx = a.left + a.width / 2 - (b.left + b.width / 2)
  const dy = a.top + a.height / 2 - (b.top + b.height / 2)
  const sx = Math.max(0.08, a.width / b.width)
  const sy = Math.max(0.08, a.height / b.height)
  to.style.transition = 'none'
  to.style.transform = `translate(${dx}px, ${dy}px) scale(${sx}, ${sy})`
  to.style.opacity = '1'
  void to.offsetWidth
  to.style.transition = 'transform 520ms var(--ease), opacity 280ms ease'
  to.style.transform = 'translate(0, 0) scale(1)'
  cardReady.value = true
}

async function openBookCard() {
  if (cardOpen.value || cardClosing) return
  cardReady.value = false
  cardOpen.value = true
  lockScroll(true)
  await nextTick()
  flipFromHero()
}

function closeBookCard() {
  if (!cardOpen.value || cardClosing) return
  cardClosing = true
  const to = sheetEl.value
  const from = heroCover.value
  const finish = () => {
    cardOpen.value = false
    cardReady.value = false
    cardClosing = false
    lockScroll(false)
    if (to) {
      to.style.transition = ''
      to.style.transform = ''
      to.style.opacity = ''
    }
  }
  if (!to || !from || prefersReduce()) {
    finish()
    return
  }
  const a = from.getBoundingClientRect()
  const b = to.getBoundingClientRect()
  const dx = a.left + a.width / 2 - (b.left + b.width / 2)
  const dy = a.top + a.height / 2 - (b.top + b.height / 2)
  const sx = Math.max(0.08, a.width / b.width)
  const sy = Math.max(0.08, a.height / b.height)
  to.style.transition = 'transform 380ms var(--ease), opacity 280ms ease'
  to.style.transform = `translate(${dx}px, ${dy}px) scale(${sx}, ${sy})`
  to.style.opacity = '0.35'
  window.setTimeout(finish, 400)
}

function onOverlayKey(e: KeyboardEvent) {
  if (e.key === 'Escape') closeBookCard()
}
const progress = computed(() => {
  const raw = book.value['readingProgress']
  const n = typeof raw === 'number' ? raw : Number(raw)
  return Number.isFinite(n) ? n : 0
})
const highlightCount = computed(
  () => data.value?.chapters.reduce((n, ch) => n + ch.highlights.length, 0) || 0,
)
const reviewCount = computed(
  () => data.value?.chapters.reduce((n, ch) => n + ch.reviews.length, 0) || 0,
)

function formatNoteDate(ts: number) {
  if (!ts) return ''
  const d = new Date(ts * 1000)
  if (Number.isNaN(d.getTime())) return ''
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}.${m}.${day}`
}

function dateAttr(ts: number) {
  if (!ts) return undefined
  const d = new Date(ts * 1000)
  if (Number.isNaN(d.getTime())) return undefined
  return d.toISOString().slice(0, 10)
}

function textsRelate(a: string, b: string) {
  const x = a.trim()
  const y = b.trim()
  if (!x || !y) return false
  return x === y || x.includes(y) || y.includes(x)
}

function chapterBlocks(ch: ChapterNotes) {
  const used = new Set<string>()
  const blocks: NoteBlock[] = []
  for (const highlight of ch.highlights) {
    const thoughts = ch.reviews.filter((review) => {
      if (used.has(review.reviewId) || !review.abstract) return false
      if (!textsRelate(review.abstract, highlight.markText)) return false
      used.add(review.reviewId)
      return true
    })
    blocks.push({
      kind: 'highlight',
      key: highlight.bookmarkId,
      highlight,
      thoughts,
    })
  }
  for (const review of ch.reviews) {
    if (used.has(review.reviewId)) continue
    blocks.push({ kind: 'thought', key: review.reviewId, review })
  }
  return blocks
}

watch(cardOpen, (open) => {
  if (open) window.addEventListener('keydown', onOverlayKey)
  else window.removeEventListener('keydown', onOverlayKey)
})

watch(
  () => String(route.params.bookId),
  async (bookId) => {
    loading.value = true
    error.value = ''
    data.value = null
    coverBroken.value = false
    cardOpen.value = false
    lockScroll(false)
    try {
      data.value = await fetchNotes(bookId)
    } catch (e) {
      error.value = e instanceof Error ? e.message : '加载失败'
    } finally {
      loading.value = false
      await nextTick()
      const match = /^#ch-(\d+)$/.exec(route.hash || window.location.hash)
      if (match) scrollToChapter(Number(match[1]))
    }
  },
  { immediate: true },
)
</script>

<template>
  <section :aria-busy="loading">
    <RouterLink class="back" to="/notes">← 返回书单</RouterLink>
    <div v-if="error" class="error" role="alert">{{ error }}</div>
    <p v-if="loading" class="muted">正在展开书页…</p>
    <template v-else-if="data">
      <div class="detail-layout">
        <aside class="detail-rail">
          <button
            class="book-hero"
            type="button"
            :aria-expanded="cardOpen"
            aria-haspopup="dialog"
            aria-controls="book-intro-card"
            @click="openBookCard"
          >
            <span ref="heroCover" class="hero-cover">
              <img
                v-if="cover && !coverBroken"
                :src="cover"
                alt=""
                @error="coverBroken = true"
              />
              <span v-else class="cover" aria-hidden="true">书</span>
            </span>
            <div>
              <h2 class="page-title">{{ title }}</h2>
              <p class="muted">
                {{ author }}
                · {{ data.chapters.length }} 章
                · 划线 {{ highlightCount }}
                · 想法 {{ reviewCount }}
                · 进度 {{ progress }}%
              </p>
              <div class="progress" aria-hidden="true">
                <i :style="{ width: Math.min(100, Math.max(0, progress)) + '%' }" />
              </div>
            </div>
          </button>
          <nav v-if="data.chapters.length" class="toc" aria-label="章节目录">
            <h3>目录</h3>
            <button
              v-for="ch in data.chapters"
              :key="ch.chapterUid"
              class="toc-item"
              type="button"
              :class="{ active: currentUid === ch.chapterUid }"
              :aria-current="currentUid === ch.chapterUid ? 'true' : undefined"
              @click="onTocClick($event, ch.chapterUid)"
            >
              {{ ch.title }}
            </button>
          </nav>
        </aside>
        <div v-if="data.chapters.length" class="notes-pane">
          <article
            v-for="ch in data.chapters"
            :id="'ch-' + ch.chapterUid"
            :key="ch.chapterUid"
            class="chapter"
            :class="{
              'is-current': currentUid === ch.chapterUid,
              'is-flash': flashUid === ch.chapterUid,
            }"
          >
            <h3>{{ ch.title }}</h3>
            <div class="note-stack">
              <article
                v-for="block in chapterBlocks(ch)"
                :key="block.key"
                class="note-card"
                :class="block.kind === 'highlight' ? 'is-mark' : 'is-idea'"
              >
                <template v-if="block.kind === 'highlight'">
                  <header class="note-meta">
                    <span class="note-kind">划线</span>
                    <time
                      v-if="block.highlight.createTime"
                      :datetime="dateAttr(block.highlight.createTime)"
                    >
                      {{ formatNoteDate(block.highlight.createTime) }}
                    </time>
                  </header>
                  <blockquote class="mark-text">{{ block.highlight.markText }}</blockquote>
                  <div
                    v-for="thought in block.thoughts"
                    :key="thought.reviewId"
                    class="idea-body nested"
                  >
                    <header class="note-meta">
                      <span class="note-kind idea">想法</span>
                      <time
                        v-if="thought.createTime"
                        :datetime="dateAttr(thought.createTime)"
                      >
                        {{ formatNoteDate(thought.createTime) }}
                      </time>
                    </header>
                    <p>{{ thought.content }}</p>
                  </div>
                </template>
                <template v-else>
                  <blockquote v-if="block.review.abstract" class="idea-abstract">
                    <header class="note-meta">
                      <span class="note-kind">划线</span>
                    </header>
                    <p>{{ block.review.abstract }}</p>
                  </blockquote>
                  <div class="idea-body">
                    <header class="note-meta">
                      <span class="note-kind idea">批注</span>
                      <time
                        v-if="block.review.createTime"
                        :datetime="dateAttr(block.review.createTime)"
                      >
                        {{ formatNoteDate(block.review.createTime) }}
                      </time>
                    </header>
                    <p>{{ block.review.content }}</p>
                  </div>
                </template>
              </article>
            </div>
          </article>
        </div>
        <p v-else class="empty">这本书还没有可展示的划线或想法。</p>
      </div>
      <Teleport to="body">
        <div
          v-if="cardOpen"
          id="book-intro-card"
          class="book-overlay"
          :class="{ ready: cardReady }"
          role="dialog"
          aria-modal="true"
          aria-labelledby="book-intro-title"
          @click.self="closeBookCard"
        >
          <article ref="sheetEl" class="book-sheet" @click.stop>
            <button class="sheet-close" type="button" aria-label="关闭简介" @click="closeBookCard">
              关闭
            </button>
            <div class="sheet-cover">
              <img v-if="cover && !coverBroken" :src="cover" alt="" />
              <div v-else class="cover" aria-hidden="true">书</div>
            </div>
            <div class="sheet-body">
              <a
                id="book-intro-title"
                class="sheet-title"
                :href="wereadUrl()"
                target="_blank"
                rel="noopener noreferrer"
                :title="'在微信读书打开《' + title + '》'"
              >
                {{ title }}
              </a>
              <p class="sheet-author">{{ author }}</p>
              <p v-if="category || publisher" class="sheet-meta">
                <span v-if="category">{{ category }}</span>
                <span v-if="category && publisher"> · </span>
                <span v-if="publisher">{{ publisher }}</span>
              </p>
              <p class="sheet-intro">{{ intro || '这本书还没有简介。同步后如果官方有 intro，会显示在这里。' }}</p>
              <p class="sheet-hint">点击书名，前往微信读书</p>
            </div>
          </article>
        </div>
      </Teleport>
    </template>
  </section>
</template>
