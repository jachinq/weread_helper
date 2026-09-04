<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { fetchNotes, downloadNotesExport, fetchNotesExport, fetchSettings, setHighlightStarred } from '../api'
import HighlightLightbox from '../highlights/HighlightLightbox.vue'
import { nextHighlightDisplay, normalizeHighlightDisplay, type HighlightDisplay } from '../highlights/types'
import type { ChapterNotes, Highlight, NotesResponse, RandomHighlight, Review } from '../types'
import StarMark from './StarMark.vue'

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
const exporting = ref(false)
const copying = ref(false)
const exportError = ref('')
const copyHint = ref('')
let copyHintTimer: number | undefined
const starring = ref('')
const highlightDisplay = ref<HighlightDisplay>('card')
const focusedNote = ref<number | null>(null)
const noteEls = new Map<string, HTMLElement>()
let lastNoteFocus: HTMLElement | null = null
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

async function scrollToHighlight(bookmarkId: string) {
  await nextTick()
  const el = document.getElementById(`hl-${bookmarkId}`)
  if (!el) return
  await animateScroll(window, window.scrollY + el.getBoundingClientRect().top - headerOffset())
  el.classList.add('is-flash')
  window.setTimeout(() => el.classList.remove('is-flash'), 1400)
}

function hashTarget() {
  const hash = route.hash || window.location.hash
  const hl = /^#hl-(.+)$/.exec(hash)
  if (hl) {
    try {
      return { type: 'hl' as const, id: decodeURIComponent(hl[1]) }
    } catch {
      return { type: 'hl' as const, id: hl[1] }
    }
  }
  const ch = /^#ch-(\d+)$/.exec(hash)
  if (ch) return { type: 'ch' as const, id: Number(ch[1]) }
  return null
}

async function exportBook(format: 'md' | 'json') {
  const bookId = String(route.params.bookId || '')
  if (!bookId || exporting.value || copying.value) return
  exporting.value = true
  exportError.value = ''
  copyHint.value = ''
  try {
    await downloadNotesExport({ bookId, format, filename: title.value })
  } catch (e) {
    exportError.value = e instanceof Error ? e.message : '导出失败'
  } finally {
    exporting.value = false
  }
}

async function writeClipboard(text: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text)
    return
  }
  const ta = document.createElement('textarea')
  ta.value = text
  ta.setAttribute('readonly', '')
  ta.style.position = 'fixed'
  ta.style.left = '-9999px'
  document.body.appendChild(ta)
  ta.select()
  const ok = document.execCommand('copy')
  ta.remove()
  if (!ok) throw new Error('浏览器不允许写入剪贴板')
}

async function copyBookMarkdown() {
  const bookId = String(route.params.bookId || '')
  if (!bookId || exporting.value || copying.value) return
  copying.value = true
  exportError.value = ''
  copyHint.value = ''
  try {
    const res = await fetchNotesExport({ bookId, format: 'md' })
    const text = await res.text()
    await writeClipboard(text)
    copyHint.value = '已复制 Markdown 到剪贴板'
    if (copyHintTimer) window.clearTimeout(copyHintTimer)
    copyHintTimer = window.setTimeout(() => {
      if (copyHint.value === '已复制 Markdown 到剪贴板') copyHint.value = ''
    }, 2400)
  } catch (e) {
    exportError.value = e instanceof Error ? e.message : '复制失败'
  } finally {
    copying.value = false
  }
}

async function toggleStar(highlight: Highlight) {
  if (!highlight.bookmarkId || starring.value) return
  starring.value = highlight.bookmarkId
  const next = !highlight.starred
  try {
    await setHighlightStarred(highlight.bookmarkId, next)
    highlight.starred = next
  } finally {
    starring.value = ''
  }
}

onUnmounted(() => {
  if (flashTimer) window.clearTimeout(flashTimer)
  if (copyHintTimer) window.clearTimeout(copyHintTimer)
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

function blockGalleryId(block: NoteBlock) {
  return block.kind === 'highlight' ? block.highlight.bookmarkId : `review:${block.review.reviewId}`
}

function setNoteEl(id: string, el: Element | null) {
  if (el instanceof HTMLElement) noteEls.set(id, el)
  else noteEls.delete(id)
}

const gallery = computed((): RandomHighlight[] => {
  if (!data.value) return []
  const bookId = String(route.params.bookId || data.value.bookId || '')
  const items: RandomHighlight[] = []
  for (const ch of data.value.chapters) {
    for (const block of chapterBlocks(ch)) {
      if (block.kind === 'highlight') {
        items.push({
          bookmarkId: block.highlight.bookmarkId,
          bookId,
          markText: block.highlight.markText,
          createTime: block.highlight.createTime,
          title: title.value,
          author: author.value,
          cover: cover.value,
          starred: block.highlight.starred,
          chapterTitle: ch.title,
        })
      } else {
        items.push({
          bookmarkId: `review:${block.review.reviewId}`,
          bookId,
          markText: (block.review.abstract || block.review.content || '').trim(),
          createTime: block.review.createTime,
          title: title.value,
          author: author.value,
          cover: cover.value,
          starred: false,
          chapterTitle: ch.title,
        })
      }
    }
  }
  return items
})

const focusedItem = computed(() => {
  if (focusedNote.value == null) return null
  return gallery.value[focusedNote.value] ?? null
})
const lightboxOpen = computed(() => focusedItem.value != null)
const lightboxSourceEl = computed(() => {
  const item = focusedItem.value
  if (!item) return null
  return noteEls.get(item.bookmarkId) ?? null
})
const focusedCanStar = computed(() => !!focusedItem.value && !focusedItem.value.bookmarkId.startsWith('review:'))

function openNote(block: NoteBlock, event: MouseEvent) {
  if (focusedNote.value != null) return
  const id = blockGalleryId(block)
  const i = gallery.value.findIndex((item) => item.bookmarkId === id)
  if (i < 0) return
  lastNoteFocus = document.activeElement instanceof HTMLElement ? document.activeElement : (event.currentTarget as HTMLElement)
  focusedNote.value = i
}

function closeNoteLightbox() {
  focusedNote.value = null
  lastNoteFocus?.focus()
}

function goNote(delta: number) {
  const n = gallery.value.length
  if (focusedNote.value == null || n < 2) return
  focusedNote.value = (focusedNote.value + delta + n) % n
}

function cycleNoteDisplay() {
  highlightDisplay.value = nextHighlightDisplay(highlightDisplay.value)
}

function onLightboxStarred(bookmarkId: string, starred: boolean) {
  if (!data.value) return
  for (const ch of data.value.chapters) {
    const hit = ch.highlights.find((h) => h.bookmarkId === bookmarkId)
    if (hit) hit.starred = starred
  }
}

function onNoteCardKey(block: NoteBlock, event: KeyboardEvent) {
  if (event.key !== 'Enter' && event.key !== ' ') return
  event.preventDefault()
  const i = gallery.value.findIndex((item) => item.bookmarkId === blockGalleryId(block))
  if (i < 0) return
  lastNoteFocus = event.currentTarget instanceof HTMLElement ? event.currentTarget : null
  focusedNote.value = i
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
    focusedNote.value = null
    lockScroll(false)
    try {
      data.value = await fetchNotes(bookId)
    } catch (e) {
      error.value = e instanceof Error ? e.message : '加载失败'
    } finally {
      loading.value = false
      await nextTick()
      const target = hashTarget()
      if (target?.type === 'hl') await scrollToHighlight(target.id)
      else if (target?.type === 'ch') await scrollToChapter(target.id)
    }
  },
  { immediate: true },
)

onMounted(() => {
  void fetchSettings()
    .then((s) => {
      highlightDisplay.value = normalizeHighlightDisplay(s.highlightDisplay)
    })
    .catch(() => {
      highlightDisplay.value = 'card'
    })
})
</script>

<template>
  <section :aria-busy="loading">
    <RouterLink class="back" to="/notes">← 返回书单</RouterLink>
    <div class="toolbar note-export">
      <button class="btn" type="button" :disabled="exporting || copying || loading" @click="exportBook('md')">
        {{ exporting ? '导出中…' : '导出本书 Markdown' }}
      </button>
      <button class="btn" type="button" :disabled="exporting || copying || loading" @click="copyBookMarkdown">
        {{ copying ? '复制中…' : '复制 Markdown 到剪贴板' }}
      </button>
      <button class="btn" type="button" :disabled="exporting || copying || loading" @click="exportBook('json')">
        导出 JSON
      </button>
    </div>
    <div v-if="exportError" class="error" role="alert">{{ exportError }}</div>
    <p v-if="copyHint" class="muted" role="status">{{ copyHint }}</p>
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
                :id="block.kind === 'highlight' ? 'hl-' + block.highlight.bookmarkId : undefined"
                :ref="(el) => setNoteEl(blockGalleryId(block), el as Element | null)"
                class="note-card"
                :class="block.kind === 'highlight' ? 'is-mark' : 'is-idea'"
                role="button"
                tabindex="0"
                :aria-label="block.kind === 'highlight' ? '展开划线卡片' : '展开想法卡片'"
                @click="openNote(block, $event)"
                @keydown="onNoteCardKey(block, $event)"
              >
                <template v-if="block.kind === 'highlight'">
                  <header class="note-meta">
                    <span class="note-kind">划线</span>
                    <span class="note-meta-actions">
                      <time
                        v-if="block.highlight.createTime"
                        :datetime="dateAttr(block.highlight.createTime)"
                      >
                        {{ formatNoteDate(block.highlight.createTime) }}
                      </time>
                      <StarMark
                        :starred="!!block.highlight.starred"
                        :busy="starring === block.highlight.bookmarkId"
                        @toggle="toggleStar(block.highlight)"
                      />
                    </span>
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
      <Teleport to="body">
        <HighlightLightbox
          v-if="lightboxOpen && focusedItem"
          :display="highlightDisplay"
          :item="focusedItem"
          :total="gallery.length"
          :source-el="lightboxSourceEl"
          :can-star="focusedCanStar"
          @closed="closeNoteLightbox"
          @go="goNote"
          @starred="onLightboxStarred"
          @cycle-display="cycleNoteDisplay"
        />
      </Teleport>
    </template>
  </section>
</template>
