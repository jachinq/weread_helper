<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart, RadarChart } from 'echarts/charts'
import { GridComponent, RadarComponent, TooltipComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { fetchReport, fetchReportSnapshot, fetchReportYears } from '../api'
import type { ReportBookCard, YearReport } from '../types'

use([CanvasRenderer, BarChart, RadarChart, GridComponent, RadarComponent, TooltipComponent])

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const fetching = ref(false)
const error = ref('')
const report = ref<YearReport | null>(null)
const missing = ref(false)
const years = ref<number[]>([])
const cached = ref<number[]>([])

const year = computed(() => {
  const n = Number(route.query.year)
  if (Number.isFinite(n) && n >= 2015) return n
  return years.value[0] || new Date().getFullYear()
})

function setYear(y: number) {
  if (y === year.value && route.query.year) return
  router.replace({ query: { ...route.query, year: String(y) } })
}

function formatSeconds(sec: number) {
  if (sec <= 0) return '0 分钟'
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  if (h > 0 && m > 0) return `${h} 小时 ${m} 分钟`
  if (h > 0) return `${h} 小时`
  return `${m} 分钟`
}

const monthChartOption = computed(() => {
  const months = report.value?.months || []
  return {
    color: ['#e8b86d'],
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#221a14',
      borderColor: '#e8b86d',
      textStyle: { color: '#f8f1e3' },
      formatter: (params: unknown) => {
        const p = Array.isArray(params) ? params[0] : params
        const row = p && typeof p === 'object' ? (p as { name?: string; value?: number }) : {}
        return `${row.name || ''}<br/>${row.value ?? 0} 小时`
      },
    },
    textStyle: { color: '#c4b7a3' },
    grid: { left: 48, right: 16, top: 28, bottom: 36 },
    xAxis: {
      type: 'category',
      data: months.map((m) => m.label),
      axisLine: { lineStyle: { color: '#c4b7a3' } },
      axisLabel: { color: '#c4b7a3' },
    },
    yAxis: {
      type: 'value',
      name: '小时',
      splitLine: { lineStyle: { color: 'rgba(243,226,191,0.16)', type: 'dashed' } },
      axisLabel: { color: '#c4b7a3' },
    },
    series: [
      {
        type: 'bar',
        data: months.map((m) => Math.round((m.seconds / 3600) * 10) / 10),
        barMaxWidth: 18,
        name: '阅读小时',
      },
    ],
  }
})

const radarOption = computed(() => {
  const cats = report.value?.categories || []
  const max = Math.max(1, ...cats.map((c) => c.count))
  return {
    color: ['#c45c26'],
    tooltip: { backgroundColor: '#221a14', borderColor: '#e8b86d', textStyle: { color: '#f8f1e3' } },
    radar: {
      indicator: cats.map((c) => ({ name: c.name, max })),
      axisName: { color: '#c4b7a3' },
      splitLine: { lineStyle: { color: 'rgba(243,226,191,0.2)' } },
      splitArea: { areaStyle: { color: ['rgba(243,226,191,0.04)', 'rgba(243,226,191,0.01)'] } },
    },
    series: [{ type: 'radar', data: [{ value: cats.map((c) => c.count), name: '本数' }] }],
  }
})

const hourUnit = computed(() => (report.value?.hoursUnit === 'notes' ? '条笔记' : '分钟'))

const hourChartOption = computed(() => {
  const hours = report.value?.hours || []
  const data = report.value?.hoursUnit === 'notes' ? hours : hours.map((s) => Math.round(s / 60))
  return {
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
      data: hours.map((_, i) => `${i}时`),
      axisLine: { lineStyle: { color: '#c4b7a3' } },
      axisLabel: { color: '#c4b7a3', interval: 2 },
    },
    yAxis: {
      type: 'value',
      name: hourUnit.value,
      splitLine: { lineStyle: { color: 'rgba(243,226,191,0.16)', type: 'dashed' } },
      axisLabel: { color: '#c4b7a3' },
    },
    series: [{ type: 'bar', data, barMaxWidth: 12, name: hourUnit.value }],
  }
})

function cloudStyle(items: { count: number }[], count: number) {
  const max = Math.max(1, ...items.map((x) => x.count))
  const min = Math.min(...items.map((x) => x.count))
  const t = max === min ? 1 : (count - min) / (max - min)
  const size = 13 + t * 22
  const weight = t > 0.6 ? 700 : 500
  return { fontSize: `${size}px`, fontWeight: weight, opacity: 0.72 + t * 0.28 }
}

const awards = computed(() => {
  const r = report.value
  if (!r) return []
  const rows: { key: string; label: string; note?: string; book: ReportBookCard }[] = []
  const push = (key: string, label: string, book: ReportBookCard | undefined, note?: string) => {
    if (book?.title) rows.push({ key, label, note, book })
  }
  push('favorite', '年度最爱', r.favorite)
  push('first', '第一本阅读', r.firstRead, '官方口径，不是「今年第一本读完」')
  push('think', '思考最多', r.mostHighlights)
  push('rare', '最小众', r.rarest)
  push('immerse', '最沉浸阅读', r.immersed, '来自官方年度书票')
  push('night', '读到深夜', r.lateNight, '无官方书票时按 0–5 点笔记时间推断')
  return rows
})

async function loadYears() {
  const meta = await fetchReportYears()
  years.value = meta.years?.length ? meta.years : [meta.current]
  cached.value = meta.cached || []
  if (!route.query.year) setYear(meta.current)
}

async function load() {
  loading.value = true
  error.value = ''
  missing.value = false
  try {
    const data = await fetchReport(year.value)
    if ('missing' in data && data.missing) {
      missing.value = true
      report.value = null
      return
    }
    report.value = data
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
    report.value = null
  } finally {
    loading.value = false
  }
}

async function fetchYear() {
  fetching.value = true
  error.value = ''
  try {
    report.value = await fetchReportSnapshot(year.value)
    missing.value = false
    if (!cached.value.includes(year.value)) cached.value = [year.value, ...cached.value]
  } catch (e) {
    error.value = e instanceof Error ? e.message : '拉取失败'
  } finally {
    fetching.value = false
  }
}

watch(year, load)
onMounted(async () => {
  try {
    await loadYears()
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载年份失败'
  }
  await load()
})
</script>

<template>
  <section :aria-busy="loading">
    <h2 class="page-title">年度报告</h2>
    <p class="muted">读本地官方年报快照与笔记；历史年份需单独拉取，不会在打开页面时打微信读书。</p>
    <div class="modes" role="group" aria-label="选择年份">
      <button
        v-for="y in years"
        :key="y"
        class="btn"
        type="button"
        :aria-pressed="year === y"
        :class="{ active: year === y }"
        @click="setYear(y)"
      >
        {{ y }}
      </button>
    </div>
    <div v-if="error" class="error" role="alert">{{ error }}</div>
    <p v-if="loading" class="muted">正在汇总…</p>
    <div v-else-if="missing" class="stats-block">
      <p>本地还没有 {{ year }} 年的官方阅读快照。</p>
      <button class="btn btn-solid" type="button" :disabled="fetching" @click="fetchYear">
        {{ fetching ? '正在拉取…' : `拉取 ${year} 年` }}
      </button>
    </div>
    <template v-else-if="report">
      <section class="stats-block">
        <h3 class="page-title">概况</h3>
        <div class="stats-row">
          <div class="stat">总阅读时长<b>{{ report.overview.totalReadTimeFormatted }}</b></div>
          <div class="stat">日均时长<b>{{ report.overview.dayAverageReadTimeFormatted }}</b></div>
          <div v-if="report.overview.booksRead" class="stat">读过<b>{{ report.overview.booksRead }}</b></div>
          <div v-if="report.overview.booksFinished" class="stat">读完<b>{{ report.overview.booksFinished }}</b></div>
          <div class="stat">阅读天数<b>{{ report.overview.readDays }}</b></div>
          <div class="stat">笔记<b>{{ report.overview.noteCount }} 条</b></div>
        </div>
      </section>

      <section v-if="report.months.length" class="stats-block">
        <h3 class="page-title">月阅读时长</h3>
        <div class="chart">
          <VChart :option="monthChartOption" autoresize />
        </div>
        <p v-if="report.peakMonth && report.peakMonth.seconds > 0" class="stats-lead">
          {{ report.peakMonth.month }} 月阅读最久 · {{ formatSeconds(report.peakMonth.seconds) }}
        </p>
        <p v-if="report.cheer" class="cheer">{{ report.cheer }}</p>
      </section>

      <section v-if="awards.length" class="stats-block">
        <h3 class="page-title">年度之书</h3>
        <ul class="prefer-books">
          <li v-for="item in awards" :key="item.key">
            <component
              :is="item.book.bookId ? RouterLink : 'div'"
              class="prefer-ticket"
              :to="item.book.bookId ? `/notes/${item.book.bookId}` : undefined"
            >
              <p class="kicker">{{ item.label }}</p>
              <h4>{{ item.book.title }}</h4>
              <p class="meta">{{ item.book.author }}</p>
              <p v-if="item.book.hint" class="meta">{{ item.book.hint }}</p>
              <p v-if="item.note" class="meta muted">{{ item.note }}</p>
            </component>
          </li>
        </ul>
      </section>

      <section v-if="report.holidays?.length" class="stats-block">
        <h3 class="page-title">节假日读书</h3>
        <ul class="holiday-list">
          <li v-for="h in report.holidays" :key="h.date + h.name">
            <b>{{ h.name }}</b>
            <span class="meta">{{ h.date }}</span>
            <span v-if="h.book">{{ h.book.title }}</span>
            <span v-else-if="h.read && !h.bookKnown">读过，书未知</span>
            <span v-else class="muted">未记录</span>
          </li>
        </ul>
      </section>

      <section v-if="report.monthBooks?.length" class="stats-block">
        <h3 class="page-title">月度之书</h3>
        <p class="muted">按当月划线最多；没有划线的月份不展示。</p>
        <ol class="rank-list">
          <li v-for="m in report.monthBooks" :key="m.month">
            <span>{{ m.month }} 月 · {{ m.book.title }}</span>
            <em>{{ m.count }} 条划线</em>
          </li>
        </ol>
      </section>

      <section v-if="report.categories?.length" class="stats-block">
        <h3 class="page-title">偏好阅读</h3>
        <div class="chart chart-h">
          <VChart :option="radarOption" autoresize />
        </div>
      </section>

      <section v-if="report.hours?.length" class="stats-block">
        <h3 class="page-title">偏好阅读时间</h3>
        <p v-if="report.hoursUnit === 'notes'" class="muted">官方年报没有 24 小时时长，图中按该年笔记创建时刻估算。</p>
        <div class="chart">
          <VChart :option="hourChartOption" autoresize />
        </div>
      </section>

      <section v-if="report.authors?.length" class="stats-block">
        <h3 class="page-title">偏好作者</h3>
        <div class="word-cloud">
          <span v-for="a in report.authors" :key="a.name" :style="cloudStyle(report.authors, a.count)">
            {{ a.name }}
          </span>
        </div>
      </section>

      <section v-if="report.copyrights?.length" class="stats-block">
        <h3 class="page-title">偏好版权方</h3>
        <p v-if="report.copyrightSource === 'preferPublisher'" class="muted">官方未返回版权方，以下为出版社。</p>
        <div class="word-cloud">
          <span v-for="a in report.copyrights" :key="a.name" :style="cloudStyle(report.copyrights, a.count)">
            {{ a.name }}
          </span>
        </div>
      </section>
    </template>
  </section>
</template>
