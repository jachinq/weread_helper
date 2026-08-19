<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { fetchReportSnapshot, fetchReportYears, fetchStats } from '../api'
import type { StatsResponse } from '../types'

use([CanvasRenderer, BarChart, GridComponent, TooltipComponent])

const modes = [
  { id: 'weekly', label: '周' },
  { id: 'monthly', label: '月' },
  { id: 'annually', label: '年' },
  { id: 'overall', label: '累计' },
] as const

type ModeId = (typeof modes)[number]['id']

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const error = ref('')
const stats = ref<StatsResponse | null>(null)
const missingYear = ref(false)
const fetchingYear = ref(false)
const years = ref<number[]>([])
let loadGen = 0
const brokenCover = ref<Record<string, boolean>>({})

const mode = computed<ModeId>(() => {
  const q = String(route.query.mode || 'monthly')
  return modes.some((m) => m.id === q) ? (q as ModeId) : 'monthly'
})

function queryYearNum(): number | null {
  const raw = route.query.year
  const s = Array.isArray(raw) ? raw[0] : raw
  const n = Number(s)
  if (Number.isFinite(n) && n >= 2015) return n
  return null
}

const year = computed(() => queryYearNum() ?? years.value[0] ?? new Date().getFullYear())

function setMode(id: ModeId) {
  if (id === mode.value) return
  const query: Record<string, string> = { mode: id }
  if (id === 'annually') query.year = String(year.value)
  router.replace({ query })
}

function setYear(y: number) {
  if (y === year.value && queryYearNum() === y) return
  router.replace({ query: { mode: 'annually', year: String(y) } })
}

const yearsAsc = computed(() => [...years.value].sort((a, b) => a - b))

const canPrevYear = computed(() => yearsAsc.value.some((y) => y < year.value))

const canNextYear = computed(() => yearsAsc.value.some((y) => y > year.value))

function stepYear(delta: number) {
  const list = yearsAsc.value
  if (!list.length) return
  const candidates = delta < 0 ? list.filter((y) => y < year.value) : list.filter((y) => y > year.value)
  if (!candidates.length) return
  setYear(delta < 0 ? candidates[candidates.length - 1] : candidates[0])
}

function asNum(v: unknown) {
  if (typeof v === 'number' && Number.isFinite(v)) return v
  if (typeof v === 'string') return Number(v) || 0
  return 0
}

function asStr(v: unknown) {
  return typeof v === 'string' ? v : v == null ? '' : String(v)
}

function obj(v: unknown): Record<string, unknown> | null {
  return v && typeof v === 'object' && !Array.isArray(v) ? (v as Record<string, unknown>) : null
}

type DurationPart = { n: string; unit: string }

function durationParts(sec: number): DurationPart[] {
  if (sec <= 0) return [{ n: '0', unit: '分钟' }]
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const parts: DurationPart[] = []
  if (h > 0) parts.push({ n: String(h), unit: '小时' })
  if (m > 0 || h === 0) parts.push({ n: String(m), unit: '分钟' })
  return parts
}

function parseDurationParts(formatted: string | undefined, sec: number): DurationPart[] {
  if (formatted) {
    const parts: DurationPart[] = []
    const re = /(\d+)\s*(小时|分钟)/g
    let m: RegExpExecArray | null
    while ((m = re.exec(formatted))) {
      parts.push({ n: m[1], unit: m[2] })
    }
    if (parts.length) return parts
  }
  return durationParts(sec)
}

function formatSeconds(sec: number) {
  return durationParts(sec)
    .map((p) => `${p.n} ${p.unit}`)
    .join(' ')
}

function axisLabel(ts: number, i: number) {
  if (!ts) return String(i + 1)
  const d = new Date(ts * 1000)
  if (Number.isNaN(d.getTime())) return String(i + 1)
  const y = d.getFullYear()
  const mo = d.getMonth() + 1
  const day = d.getDate()
  if (mode.value === 'overall') return `${y}`
  if (mode.value === 'annually') return `${mo}月`
  return `${mo}/${day}`
}

function seriesFromMap(raw: unknown): { names: string[]; minutes: number[] } {
  const names: string[] = []
  const minutes: number[] = []
  if (Array.isArray(raw)) {
    raw.forEach((item, i) => {
      if (typeof item === 'number') {
        names.push(String(i + 1))
        minutes.push(Math.round(item / 60))
        return
      }
      const row = obj(item)
      if (!row) return
      const sec = asNum(row.readTime ?? row.time ?? row.count ?? row.value)
      names.push(asStr(row.date ?? row.day ?? row.baseTime) || String(i + 1))
      minutes.push(Math.round(sec / 60))
    })
    return { names, minutes }
  }
  const map = obj(raw)
  if (!map) return { names, minutes }
  Object.entries(map)
    .sort(([a], [b]) => asNum(a) - asNum(b))
    .forEach(([k, v], i) => {
      const ts = asNum(k)
      names.push(ts ? axisLabel(ts, i) : k)
      minutes.push(Math.round(asNum(v) / 60))
    })
  return { names, minutes }
}

const daily = computed(() => {
  const s = stats.value
  if (!s) return { names: [] as string[], minutes: [] as number[] }
  const fromTimes = seriesFromMap(s.readTimes)
  if (fromTimes.names.length) return fromTimes
  return seriesFromMap(s.dailyReadTimes)
})

const chartUnit = computed(() => (mode.value === 'annually' || mode.value === 'overall' ? '小时' : '分钟'))

const chartOption = computed(() => {
  const { names, minutes } = daily.value
  const data = chartUnit.value === '小时' ? minutes.map((m) => Math.round((m / 60) * 10) / 10) : minutes
  return {
    color: ['#e8b86d'],
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#221a14',
      borderColor: '#e8b86d',
      textStyle: { color: '#f8f1e3' },
    },
    textStyle: { color: '#c4b7a3' },
    grid: { left: 48, right: 16, top: 28, bottom: 36 },
    xAxis: {
      type: 'category',
      data: names,
      axisLine: { lineStyle: { color: '#c4b7a3' } },
      axisLabel: { color: '#c4b7a3' },
    },
    yAxis: {
      type: 'value',
      name: chartUnit.value,
      axisLine: { show: false },
      splitLine: { lineStyle: { color: 'rgba(243,226,191,0.16)', type: 'dashed' } },
      axisLabel: { color: '#c4b7a3' },
    },
    series: [{ type: 'bar', data, barMaxWidth: 18, name: `阅读${chartUnit.value}` }],
  }
})

const preferCats = computed(() => {
  const raw = stats.value?.preferCategory
  if (!Array.isArray(raw)) return []
  return raw
    .map((item) => {
      const row = obj(item)
      if (!row) return null
      const readingTime = asNum(row.readingTime ?? row.readTime ?? row.value)
      const readingCount = asNum(row.readingCount ?? row.count)
      if (readingTime <= 0 && readingCount <= 0) return null
      return {
        name: asStr(row.categoryTitle ?? row.categoryName ?? row.name ?? row.title) || '分类',
        readingTime,
        readingCount,
      }
    })
    .filter((x): x is { name: string; readingTime: number; readingCount: number } => !!x)
    .sort((a, b) => b.readingTime - a.readingTime)
})

const catChartOption = computed(() => {
  const cats = [...preferCats.value].slice(0, 8).reverse()
  return {
    color: ['#c45c26'],
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#221a14',
      borderColor: '#e8b86d',
      textStyle: { color: '#f8f1e3' },
      formatter: (params: unknown) => {
        const p = Array.isArray(params) ? params[0] : params
        const row = obj(p)
        if (!row) return ''
        const name = asStr(row.name)
        const hours = asNum(row.value)
        const cat = preferCats.value.find((c) => c.name === name)
        const extra = cat?.readingCount ? ` · ${cat.readingCount} 本` : ''
        return `${name}<br/>${hours} 小时${extra}`
      },
    },
    textStyle: { color: '#c4b7a3' },
    grid: { left: 88, right: 24, top: 8, bottom: 24 },
    xAxis: {
      type: 'value',
      name: '小时',
      splitLine: { lineStyle: { color: 'rgba(243,226,191,0.16)', type: 'dashed' } },
      axisLabel: { color: '#c4b7a3' },
    },
    yAxis: {
      type: 'category',
      data: cats.map((c) => c.name),
      axisLine: { lineStyle: { color: '#c4b7a3' } },
      axisLabel: { color: '#c4b7a3' },
    },
    series: [
      {
        type: 'bar',
        data: cats.map((c) => Math.round((c.readingTime / 3600) * 10) / 10),
        barMaxWidth: 14,
        name: '阅读小时',
      },
    ],
  }
})

const hourSeries = computed(() => {
  const raw = stats.value?.preferTime
  if (!Array.isArray(raw)) return []
  return raw.map((v, i) => ({ hour: i, minutes: Math.round(asNum(v) / 60) }))
})

const hourChartOption = computed(() => ({
  color: ['#f0d18a'],
  tooltip: {
    trigger: 'axis',
    backgroundColor: '#221a14',
    borderColor: '#e8b86d',
    textStyle: { color: '#f8f1e3' },
  },
  textStyle: { color: '#c4b7a3' },
  grid: { left: 48, right: 12, top: 16, bottom: 32 },
  xAxis: {
    type: 'category',
    data: hourSeries.value.map((h) => `${h.hour}时`),
    axisLine: { lineStyle: { color: '#c4b7a3' } },
    axisLabel: { color: '#c4b7a3', interval: 2 },
  },
  yAxis: {
    type: 'value',
    name: '分钟',
    splitLine: { lineStyle: { color: 'rgba(243,226,191,0.16)', type: 'dashed' } },
    axisLabel: { color: '#c4b7a3' },
  },
  series: [{ type: 'bar', data: hourSeries.value.map((h) => h.minutes), barMaxWidth: 12, name: '该时段累计' }],
}))

const compareText = computed(() => {
  const v = stats.value?.compare
  if (typeof v !== 'number' || Number.isNaN(v)) return ''
  const pct = Math.round(Math.abs(v) * 100)
  if (pct === 0) return '较上周期日均持平'
  return v > 0 ? `较上周期日均 +${pct}%` : `较上周期日均 −${pct}%`
})

const compareUp = computed(() => typeof stats.value?.compare === 'number' && stats.value.compare > 0)

const rankText = computed(() => {
  const r = obj(stats.value?.rank)
  return r ? asStr(r.text) : ''
})

const readStats = computed(() => {
  const raw = stats.value?.readStat
  if (!Array.isArray(raw)) return []
  return raw
    .map((item) => {
      const row = obj(item)
      if (!row) return null
      return { label: asStr(row.stat), value: asStr(row.counts) }
    })
    .filter((x): x is { label: string; value: string } => !!x && !!x.label)
})

const longest = computed(() => {
  const raw = stats.value?.readLongest
  if (!Array.isArray(raw)) return []
  return raw
    .map((item) => {
      const row = obj(item)
      if (!row) return null
      const book = obj(row.book) || obj(row.albumInfo) || {}
      const bookId = asStr(book.bookId)
      const title = asStr(book.title)
      if (!title) return null
      return {
        bookId,
        title,
        author: asStr(book.author),
        cover: asStr(book.cover),
        readTime: asNum(row.readTime),
        tags: Array.isArray(row.tags) ? row.tags.map(asStr).filter(Boolean) : [],
      }
    })
    .filter((x): x is NonNullable<typeof x> => !!x)
})

const preferBooks = computed(() => {
  const raw = stats.value?.preferBooks
  if (!Array.isArray(raw)) return []
  return raw
    .map((item) => {
      const row = obj(item)
      if (!row) return null
      const book = obj(row.bookInfo) || obj(row.book)
      if (!book) return null
      const title = asStr(book.title)
      if (!title) return null
      return {
        label: asStr(row.title) || asStr(row.reason),
        bookId: asStr(book.bookId),
        title,
        author: asStr(book.author),
        cover: asStr(book.cover),
        reason: asStr(row.reason),
      }
    })
    .filter((x): x is NonNullable<typeof x> => !!x)
})

const preferAuthors = computed(() => {
  const raw = stats.value?.preferAuthor
  if (!Array.isArray(raw)) return []
  return raw
    .map((item) => {
      const row = obj(item)
      if (!row) return null
      return {
        name: asStr(row.name),
        count: asNum(row.count),
        readTime: asStr(row.readTime),
      }
    })
    .filter((x): x is { name: string; count: number; readTime: string } => x != null && !!x.name)
})

const preferPublishers = computed(() => {
  const raw = stats.value?.preferPublisher
  if (!Array.isArray(raw)) return []
  return raw
    .map((item) => {
      const row = obj(item)
      if (!row) return null
      return { name: asStr(row.name), count: asNum(row.count) }
    })
    .filter((x): x is { name: string; count: number } => x != null && !!x.name)
    .slice(0, 8)
})

const medals = computed(() => {
  const raw = stats.value?.medals
  if (!Array.isArray(raw)) return []
  const seen = new Set<string>()
  const out: { title: string; rankText: string }[] = []
  for (const item of raw) {
    const row = obj(item)
    if (!row) continue
    const title = asStr(row.displayText || row.title || row.name || row.hint)
    if (!title || seen.has(title)) continue
    seen.add(title)
    out.push({ title, rankText: asStr(row.rankText) })
    if (out.length >= 8) break
  }
  return out
})

const wrRead = computed(() => durationParts(asNum(stats.value?.wrReadTime)))
const wrListen = computed(() => durationParts(asNum(stats.value?.wrListenTime)))
const hasWr = computed(() => asNum(stats.value?.wrReadTime) > 0 || asNum(stats.value?.wrListenTime) > 0)
const totalReadParts = computed(() =>
  parseDurationParts(stats.value?.totalReadTimeFormatted, asNum(stats.value?.totalReadTime)),
)
const dayAvgParts = computed(() =>
  parseDurationParts(stats.value?.dayAverageReadTimeFormatted, asNum(stats.value?.dayAverageReadTime)),
)

async function loadYears() {
  try {
    const meta = await fetchReportYears()
    years.value = meta.years?.length ? meta.years : [meta.current]
  } catch {
    years.value = [new Date().getFullYear()]
  }
}

async function load() {
  const seq = ++loadGen
  const selectedMode = mode.value
  const selectedYear = year.value
  loading.value = true
  fetchingYear.value = false
  error.value = ''
  missingYear.value = false
  try {
    let data = await fetchStats(selectedMode, selectedMode === 'annually' ? selectedYear : undefined)
    if (seq !== loadGen) return
    if (data && 'missing' in data && data.missing && selectedMode === 'annually') {
      fetchingYear.value = true
      await fetchReportSnapshot(selectedYear)
      if (seq !== loadGen) return
      data = await fetchStats(selectedMode, selectedYear)
      if (seq !== loadGen) return
      if (data && 'missing' in data && data.missing) {
        missingYear.value = true
        stats.value = null
        error.value = `${selectedYear} 年快照拉取后仍不可用`
        return
      }
    }
    stats.value = data
    brokenCover.value = {}
  } catch (e) {
    if (seq !== loadGen) return
    error.value = e instanceof Error ? e.message : '加载失败'
    stats.value = null
    missingYear.value = selectedMode === 'annually'
  } finally {
    if (seq !== loadGen) return
    loading.value = false
    fetchingYear.value = false
  }
}

watch(
  () => [mode.value, year.value] as const,
  () => {
    void load()
  },
)
onMounted(async () => {
  await loadYears()
  await load()
})
</script>

<template>
  <section :aria-busy="loading">
    <h2 class="page-title">阅读统计</h2>
    <p class="muted">时长按秒换算。周 / 月 / 累计读本地快照；切换年份时若本地没有该年数据，会向官方拉取并保存。</p>
    <div class="modes" role="group" aria-label="统计周期">
      <button
        v-for="m in modes"
        :key="m.id"
        class="btn"
        type="button"
        :aria-pressed="mode === m.id"
        :class="{ active: mode === m.id }"
        @click="setMode(m.id)"
      >
        {{ m.label }}
      </button>
    </div>
    <div v-if="mode === 'annually' && years.length" class="year-step" role="group" aria-label="选择年份">
      <button class="btn" type="button" :disabled="!canPrevYear" aria-label="上一年" @click="stepYear(-1)">‹</button>
      <span class="year-step-label">{{ year }}</span>
      <button class="btn" type="button" :disabled="!canNextYear" aria-label="下一年" @click="stepYear(1)">›</button>
    </div>
    <div v-if="error" class="error" role="alert">{{ error }}</div>
    <p v-if="fetchingYear" class="muted">正在从官方拉取 {{ year }} 年阅读数据…</p>
    <p v-else-if="loading" class="muted">正在汇总…</p>
    <div v-else-if="missingYear" class="stats-block">
      <p>未能取得 {{ year }} 年的官方阅读快照。</p>
      <button class="btn btn-solid" type="button" :disabled="loading || fetchingYear" @click="load">
        再试一次
      </button>
    </div>
    <template v-else-if="stats">
      <p v-if="compareText || rankText || stats.preferCategoryWord || stats.preferTimeWord" class="stats-lead">
        <span v-if="compareText" :class="compareUp ? 'up' : 'down'">{{ compareText }}</span>
        <span v-if="rankText">{{ rankText }}</span>
        <span v-if="asStr(stats.preferCategoryWord)">{{ asStr(stats.preferCategoryWord) }}</span>
        <span v-if="asStr(stats.preferTimeWord)">{{ asStr(stats.preferTimeWord) }}</span>
      </p>
      <div class="stats-row">
        <div class="stat">
          总阅读时长
          <b>
            <span v-for="(p, i) in totalReadParts" :key="i" class="stat-metric">
              {{ p.n }}<span class="stat-unit">{{ p.unit }}</span>
            </span>
          </b>
        </div>
        <div class="stat">
          阅读天数
          <b>{{ stats.readDays ?? '—' }}</b>
        </div>
        <div class="stat">
          日均时长
          <b>
            <span v-for="(p, i) in dayAvgParts" :key="i" class="stat-metric">
              {{ p.n }}<span class="stat-unit">{{ p.unit }}</span>
            </span>
          </b>
        </div>
        <div v-if="hasWr" class="stat">
          文字阅读
          <b>
            <span v-for="(p, i) in wrRead" :key="i" class="stat-metric">
              {{ p.n }}<span class="stat-unit">{{ p.unit }}</span>
            </span>
          </b>
        </div>
        <div v-if="hasWr" class="stat">
          听书
          <b>
            <span v-for="(p, i) in wrListen" :key="i" class="stat-metric">
              {{ p.n }}<span class="stat-unit">{{ p.unit }}</span>
            </span>
          </b>
        </div>
        <div v-if="typeof stats.readRate === 'number'" class="stat">
          阅读超越
          <b>{{ stats.readRate }}<span class="stat-unit">%</span></b>
        </div>
        <div v-if="typeof stats.authorCount === 'number'" class="stat">
          读过作者
          <b>{{ stats.authorCount }}</b>
        </div>
      </div>
      <div v-if="readStats.length" class="read-stat">
        <div v-for="item in readStats" :key="item.label" class="read-stat-item">
          <b>{{ item.value }}</b>
          <span>{{ item.label }}</span>
        </div>
      </div>
      <div v-if="daily.names.length" class="chart">
          <VChart :key="mode + '-' + year + '-month'" :option="chartOption" autoresize />
        <table class="sr-only">
          <caption>阅读时长趋势（{{ chartUnit }}）</caption>
          <thead>
            <tr>
              <th>时段</th>
              <th>{{ chartUnit }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(name, i) in daily.names" :key="name + i">
              <td>{{ name }}</td>
              <td>{{ daily.minutes[i] }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <section v-if="preferCats.length" class="stats-block">
        <h3 class="page-title">偏好分类</h3>
        <p v-if="asStr(stats.preferCategoryWord)" class="muted">{{ asStr(stats.preferCategoryWord) }}</p>
        <div class="chart chart-h">
          <VChart :option="catChartOption" autoresize />
        </div>
      </section>
      <section v-if="hourSeries.length" class="stats-block">
        <h3 class="page-title">阅读时段</h3>
        <p v-if="asStr(stats.preferTimeWord)" class="muted">{{ asStr(stats.preferTimeWord) }}</p>
        <div class="chart">
          <VChart :option="hourChartOption" autoresize />
        </div>
      </section>
      <section v-if="longest.length" class="stats-block">
        <h3 class="page-title">读得最久</h3>
        <ol class="longest-list">
          <li v-for="(book, i) in longest" :key="book.bookId || book.title + i">
            <component
              :is="book.bookId ? RouterLink : 'div'"
              class="longest-card"
              :to="book.bookId ? `/notes/${book.bookId}` : undefined"
            >
              <img
                v-if="book.cover && !brokenCover[book.cover]"
                class="cover"
                :src="book.cover"
                :alt="book.title"
                @error="brokenCover[book.cover] = true"
              />
              <div v-else class="cover">封面</div>
              <div>
                <h4>{{ book.title }}</h4>
                <p class="meta">{{ book.author }}</p>
                <p class="meta">{{ formatSeconds(book.readTime) }}</p>
                <div v-if="book.tags.length" class="counts">
                  <span v-for="tag in book.tags" :key="tag">{{ tag }}</span>
                </div>
              </div>
            </component>
          </li>
        </ol>
      </section>
      <section v-if="preferBooks.length" class="stats-block">
        <h3 class="page-title">偏好书籍</h3>
        <ul class="prefer-books">
          <li v-for="book in preferBooks" :key="book.label + book.bookId">
            <component
              :is="book.bookId ? RouterLink : 'div'"
              class="prefer-ticket"
              :to="book.bookId ? `/notes/${book.bookId}` : undefined"
            >
              <p class="kicker">{{ book.label }}</p>
              <h4>{{ book.title }}</h4>
              <p class="meta">{{ book.author }}</p>
            </component>
          </li>
        </ul>
      </section>
      <div v-if="preferAuthors.length || preferPublishers.length" class="stats-split">
        <section v-if="preferAuthors.length" class="stats-block">
          <h3 class="page-title">偏好作者</h3>
          <ol class="rank-list">
            <li v-for="a in preferAuthors" :key="a.name">
              <span>{{ a.name }}</span>
              <em>{{ a.count }} 本 · {{ a.readTime }}</em>
            </li>
          </ol>
        </section>
        <section v-if="preferPublishers.length" class="stats-block">
          <h3 class="page-title">偏好出版社</h3>
          <ol class="rank-list">
            <li v-for="p in preferPublishers" :key="p.name">
              <span>{{ p.name }}</span>
              <em>{{ p.count }} 本</em>
            </li>
          </ol>
        </section>
      </div>
      <section v-if="medals.length" class="stats-block">
        <h3 class="page-title">勋章</h3>
        <ul class="medal-list">
          <li v-for="m in medals" :key="m.title">
            <b>{{ m.title }}</b>
            <span>{{ m.rankText }}</span>
          </li>
        </ul>
      </section>
    </template>
  </section>
</template>
