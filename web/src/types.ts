export type NotebookBook = {
  bookId: string
  title: string
  author: string
  cover: string
  reviewCount: number
  noteCount: number
  bookmarkCount: number
  readingProgress: number
  sort: number
}

export type NotebooksResponse = {
  totalBookCount?: number
  totalNoteCount?: number
  hasMore?: number
  books?: unknown[]
}

export type Highlight = {
  bookmarkId: string
  markText: string
  createTime: number
  colorStyle?: unknown
  range?: string
}

export type Review = {
  reviewId: string
  content: string
  abstract: string
  createTime: number
  star?: unknown
}

export type ChapterNotes = {
  chapterUid: number
  title: string
  chapterIdx: number
  highlights: Highlight[]
  reviews: Review[]
}

export type NotesResponse = {
  bookId: string
  book?: Record<string, unknown>
  chapters: ChapterNotes[]
}

export type StatsResponse = {
  mode?: string
  totalReadTime?: number
  totalReadTimeFormatted?: string
  dayAverageReadTime?: number
  dayAverageReadTimeFormatted?: string
  readDays?: number
  dailyReadTimes?: unknown
  preferCategory?: unknown
  preferAuthor?: unknown
  preferBooks?: unknown
  compare?: unknown
  medals?: unknown
  rank?: unknown
  [key: string]: unknown
}

export type SyncStatus = {
  state: 'idle' | 'running' | string
  lastOkAt: number
  lastError?: string
  stale?: boolean
  phase?: string
  startedAt?: number
  elapsedSec?: number
  dirtyTotal?: number
  dirtyDone?: number
  currentBookId?: string
}

export type ShelfBook = {
  bookId: string
  title: string
  author: string
  cover: string
  category?: string
  readingProgress?: number
  isTop?: boolean
  finishReading?: boolean
  readUpdateTime?: number
}

export type ShelfResponse = {
  books?: ShelfBook[]
  bookCount?: number
}
