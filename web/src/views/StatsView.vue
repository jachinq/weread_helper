<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart, LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { fetchStats } from '../api'
import type { StatsResponse } from '../types'

use([CanvasRenderer, LineChart, BarChart, GridComponent, TooltipComponent])

const modes = [
  { id: 'weekly', label: '本周' },
  { id: 'monthly', label: '本月' },
  { id: 'annually', label: '本年' },
  { id: 'overall', label: '累计' },
] as const

const mode = ref<(typeof modes)[number]['id']>('monthly')
const loading = ref(false)
const error = ref('')
const stats = ref<StatsResponse | null>(null)

function asNum(v: unknown) {
  if (typeof v === 'number') return v
  if (typeof v === 'string') return Number(v) || 0
  return 0
}

function dailySeries(raw: unknown): { names: string[]; minutes: number[] } {
  const names: string[] = []
  const minutes: number[] = []
  if (Array.isArray(raw)) {
    raw.forEach((item, i) => {
      if (typeof item === 'number') {
        names.push(String(i + 1))
        minutes.push(Math.round(item / 60))
        return
      }
      if (item && typeof item === 'object') {
        const row = item as Record<string, unknown>
        const sec = asNum(row.readTime ?? row.time ?? row.count ?? row.value)
        names.push(String(row.date ?? row.day ?? row.baseTime ?? i + 1))
        minutes.push(Math.round(sec / 60))
      }
    })
  } else if (raw && typeof raw === 'object') {
    Object.entries(raw as Record<string, unknown>).forEach(([k, v]) => {
      names.push(k)
      minutes.push(Math.round(asNum(v) / 60))
    })
  }
  return { names, minutes }
}

const daily = computed(() => dailySeries(stats.value?.dailyReadTimes))

const chartOption = computed(() => {
  const { names, minutes } = daily.value
  return {
    color: ['#b3392b'],
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 16, top: 24, bottom: 32 },
    xAxis: {
      type: 'category',
      data: names,
      axisLine: { lineStyle: { color: '#5a5046' } },
      axisLabel: { color: '#5a5046' },
    },
    yAxis: {
      type: 'value',
      name: '分钟',
      axisLine: { show: false },
      splitLine: { lineStyle: { color: '#c9b89a', type: 'dashed' } },
      axisLabel: { color: '#5a5046' },
    },
    series: [{ type: 'bar', data: minutes, barMaxWidth: 18 }],
  }
})

const preferCats = computed(() => {
  const raw = stats.value?.preferCategory
  if (!Array.isArray(raw)) return []
  return raw
    .map((item) => {
      if (!item || typeof item !== 'object') return null
      const row = item as Record<string, unknown>
      return {
        name: String(row.categoryName ?? row.name ?? row.title ?? '分类'),
        value: asNum(row.count ?? row.readTime ?? row.value),
      }
    })
    .filter((x): x is { name: string; value: number } => !!x)
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    stats.value = await fetchStats(mode.value)
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
    stats.value = null
  } finally {
    loading.value = false
  }
}

watch(mode, load)
onMounted(load)
</script>

<template>
  <section>
    <h2 class="page-title">阅读统计</h2>
    <p class="muted">时长按秒换算，数据来自本地快照。</p>
    <div class="modes">
      <button
        v-for="m in modes"
        :key="m.id"
        class="btn"
        :class="{ active: mode === m.id }"
        @click="mode = m.id"
      >
        {{ m.label }}
      </button>
    </div>
    <div v-if="error" class="error">{{ error }}</div>
    <p v-if="loading" class="muted">正在汇总…</p>
    <template v-else-if="stats">
      <div class="stats-row">
        <div class="stat">
          总阅读时长
          <b>{{ stats.totalReadTimeFormatted || '—' }}</b>
        </div>
        <div class="stat">
          阅读天数
          <b>{{ stats.readDays ?? '—' }}</b>
        </div>
        <div class="stat">
          日均时长
          <b>{{ stats.dayAverageReadTimeFormatted || '—' }}</b>
        </div>
      </div>
      <div v-if="daily.names.length" class="chart">
        <VChart :option="chartOption" autoresize />
      </div>
      <div v-if="preferCats.length">
        <h3 class="page-title">偏好分类</h3>
        <div class="counts">
          <span v-for="c in preferCats" :key="c.name">{{ c.name }}</span>
        </div>
      </div>
    </template>
  </section>
</template>
