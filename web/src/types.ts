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
  readTimes?: unknown
  wrReadTime?: number
  wrListenTime?: number
  readRate?: number
  authorCount?: number
  preferCategory?: unknown
  preferCategoryWord?: string
  preferTime?: unknown
  preferTimeWord?: string
  preferAuthor?: unknown
  preferPublisher?: unknown
  preferBooks?: unknown
  readLongest?: unknown
  readStat?: unknown
  compare?: unknown
  medals?: unknown
  rank?: unknown
  missing?: boolean
  year?: number
  month?: string
  week?: string
  weekStart?: string
  [key: string]: unknown
}

export type ReportBookCard = {
  bookId?: string
  title: string
  author?: string
  cover?: string
  hint?: string
}

export type YearReport = {
  year: number
  fetchedAt?: number
  missing?: boolean
  overview: {
    totalReadTime: number
    totalReadTimeFormatted: string
    dayAverageReadTime: number
    dayAverageReadTimeFormatted: string
    readDays: number
    booksRead?: string
    booksFinished?: string
    noteCount: number
    highlightCount?: number
    reviewCount?: number
  }
  months: { month: number; seconds: number; label: string }[]
  peakMonth?: { month: number; seconds: number; label: string }
  cheer?: string
  favorite?: ReportBookCard
  firstRead?: ReportBookCard
  mostHighlights?: ReportBookCard
  rarest?: ReportBookCard
  immersed?: ReportBookCard
  lateNight?: ReportBookCard
  holidays?: { name: string; date: string; read: boolean; bookKnown?: boolean; book?: ReportBookCard }[]
  monthBooks?: { month: number; book: ReportBookCard; count: number }[]
  categories?: { name: string; count: number }[]
  hours?: number[]
  hoursUnit?: string
  authors?: { name: string; count: number }[]
  copyrights?: { name: string; count: number }[]
  copyrightSource?: string
}

export type ReportYears = {
  years: number[]
  cached: number[]
  current: number
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

export type RandomHighlight = {
  bookmarkId: string
  bookId: string
  markText: string
  createTime: number
  title: string
  author: string
  cover: string
}

export type RandomHighlightsResponse = {
  date?: string
  items: RandomHighlight[]
}

export type AppSettings = {
  apiKeyMasked: string
  skillVersion: string
  gatewayUrl: string
  syncInterval: string
  siteTitle: string
  theme: string
  colorScheme: string
}
