<template>
  <div class="monitor-page">
    <div class="monitor-toolbar">
      <div>
        <div class="text-h6">
          {{ t('platform.metrics.title') }}
        </div>
        <div class="text-caption text-medium-emphasis">
          {{ t('platform.metrics.lastUpdated') }}: {{ lastUpdatedText }}
        </div>
      </div>
      <div class="toolbar-actions">
        <v-select
          v-model="range"
          :items="rangeOptions"
          density="compact"
          hide-details
          item-title="label"
          item-value="value"
          :style="{ width: mobile ? '100%' : '150px' }"
          :label="t('platform.metrics.timeRange')"
          @update:model-value="onRangeChange"
        />
        <v-switch
          v-model="autoRefresh"
          density="compact"
          hide-details
          color="primary"
          :label="t('platform.metrics.autoRefresh')"
        />
        <v-btn
          :loading="refreshing"
          color="primary"
          variant="tonal"
          prepend-icon="mdi-refresh"
          @click="refreshAll"
        >
          {{ t('platform.metrics.refresh') }}
        </v-btn>
      </div>
    </div>

    <v-alert
      v-if="errorMessage"
      type="error"
      variant="tonal"
      density="compact"
      closable
      class="mb-4"
      @click:close="errorMessage = ''"
    >
      {{ t('platform.metrics.errorLoad') }}: {{ errorMessage }}
    </v-alert>

    <v-alert
      v-if="current && !current.available"
      type="info"
      variant="tonal"
      density="compact"
      class="mb-4"
    >
      {{ t('platform.metrics.collecting') }}
    </v-alert>

    <v-alert
      v-else-if="current && current.status === 'degraded'"
      type="warning"
      variant="tonal"
      density="compact"
      class="mb-4"
    >
      {{ t('platform.metrics.corrupted') }}
      <div
        v-for="warning in warnings"
        :key="warning.source"
        class="text-caption mt-1"
      >
        {{ warning.source }}: {{ warning.message }}
      </div>
    </v-alert>

    <v-row dense>
      <v-col
        v-for="card in summaryCards"
        :key="card.key"
        cols="12"
        sm="6"
        lg="3"
      >
        <v-card
          class="metric-card h-100"
          variant="outlined"
        >
          <v-card-text class="pa-4">
            <div class="d-flex align-center justify-space-between">
              <span class="text-caption text-medium-emphasis">{{ card.label }}</span>
              <v-icon
                :color="card.color"
                size="20"
              >
                {{ card.icon }}
              </v-icon>
            </div>
            <div class="metric-value">
              {{ card.value }}
            </div>
            <div class="metric-sub text-caption text-medium-emphasis">
              {{ card.sub }}
            </div>
            <v-progress-linear
              v-if="card.percent !== null"
              :model-value="card.percent"
              :color="card.color"
              height="5"
              rounded
              class="mt-2"
            />
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-row
      dense
      class="mt-1"
    >
      <v-col cols="12">
        <v-card
          class="chart-card h-100"
          variant="outlined"
        >
          <v-card-title class="text-subtitle-1">
            {{ t('platform.metrics.cpuUsage') }}
          </v-card-title>
          <v-card-text>
            <sc-echarts
              :option="cpuChartOption"
              :height="chartHeight"
            />
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-row
      dense
      class="mt-1"
    >
      <v-col
        v-for="core in coreIndexes"
        :key="core"
        cols="12"
        sm="6"
        lg="3"
      >
        <v-card
          class="chart-card h-100"
          variant="outlined"
        >
          <v-card-title class="text-subtitle-2 d-flex align-center">
            CPU {{ core + 1 }}
            <v-spacer />
            <span class="text-caption text-medium-emphasis">
              {{ formatNumber(coreLatest(core)) }}%
            </span>
          </v-card-title>
          <v-card-text>
            <sc-echarts
              :option="perCoreOption(core)"
              height="17vh"
            />
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
    <div
      v-if="cpuCores > coreIndexes.length"
      class="text-caption text-medium-emphasis mt-1"
    >
      {{ t('platform.metrics.perCoreMore', { n: cpuCores - coreIndexes.length }) }}
    </div>

    <v-row dense>
      <v-col
        cols="12"
        md="6"
      >
        <v-card
          class="chart-card h-100"
          variant="outlined"
        >
          <v-card-title class="text-subtitle-1">
            {{ t('platform.metrics.stealIowait') }}
          </v-card-title>
          <v-card-text>
            <sc-echarts
              :option="stealChartOption"
              :height="chartHeight"
            />
          </v-card-text>
        </v-card>
      </v-col>
      <v-col
        cols="12"
        md="6"
      >
        <v-card
          class="chart-card h-100"
          variant="outlined"
        >
          <v-card-title class="text-subtitle-1">
            {{ t('platform.metrics.memorySwap') }}
          </v-card-title>
          <v-card-text>
            <sc-echarts
              :option="memoryChartOption"
              :height="chartHeight"
            />
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-row dense>
      <v-col
        cols="12"
        md="6"
      >
        <v-card
          class="chart-card h-100"
          variant="outlined"
        >
          <v-card-title class="text-subtitle-1">
            {{ t('platform.metrics.diskIo') }}
          </v-card-title>
          <v-card-text>
            <sc-echarts
              :option="diskChartOption"
              :height="chartHeight"
            />
          </v-card-text>
        </v-card>
      </v-col>
      <v-col
        cols="12"
        md="6"
      >
        <v-card
          class="chart-card h-100"
          variant="outlined"
        >
          <v-card-title class="text-subtitle-1">
            {{ t('platform.metrics.netTraffic') }}
            <span class="text-caption text-medium-emphasis ml-2">Mbps</span>
          </v-card-title>
          <v-card-text>
            <sc-echarts
              :option="networkChartOption"
              :height="chartHeight"
            />
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-card
      class="chart-card mt-1"
      variant="outlined"
    >
      <v-card-title class="text-subtitle-1">
        {{ t('platform.metrics.dstResources') }}
      </v-card-title>
      <v-card-text>
        <sc-echarts
          :option="dstChartOption"
          height="300px"
        />
      </v-card-text>
    </v-card>

    <v-card
      class="chart-card mt-4"
      variant="outlined"
    >
      <v-card-title class="d-flex flex-wrap align-center ga-2">
        <span>{{ t('platform.metrics.processTitle') }}</span>
        <v-chip
          size="small"
          variant="tonal"
        >
          {{ t('platform.metrics.processesTotal', { total: processTotal }) }}
        </v-chip>
        <v-chip
          v-if="processData.partial"
          size="small"
          color="warning"
          variant="tonal"
        >
          {{ t('platform.metrics.partial') }}
        </v-chip>
        <v-spacer />
        <v-text-field
          v-model="processSearch"
          density="compact"
          hide-details
          clearable
          prepend-inner-icon="mdi-magnify"
          :placeholder="t('platform.metrics.searchPlaceholder')"
          class="process-search"
          @update:model-value="onProcessSearch"
        />
        <v-select
          v-model="processSort"
          :items="sortOptions"
          density="compact"
          hide-details
          item-title="label"
          item-value="value"
          :label="t('platform.metrics.sort')"
          class="process-sort"
          @update:model-value="loadProcesses"
        />
        <v-btn-toggle
          v-model="processOrder"
          mandatory
          density="compact"
          variant="outlined"
          divided
          @update:model-value="loadProcesses"
        >
          <v-btn
            value="desc"
            size="small"
          >
            {{ t('platform.metrics.desc') }}
          </v-btn>
          <v-btn
            value="asc"
            size="small"
          >
            {{ t('platform.metrics.asc') }}
          </v-btn>
        </v-btn-toggle>
        <v-btn
          icon="mdi-refresh"
          size="small"
          variant="text"
          :loading="loadingProcesses"
          :aria-label="t('platform.metrics.refresh')"
          @click="loadProcesses"
        />
      </v-card-title>
      <v-card-text>
        <v-data-table
          :headers="processHeaders"
          :items="processes"
          :loading="loadingProcesses"
          :items-per-page="15"
          :items-per-page-options="[15, 25, 50, 100]"
          :sort-by="processSortBy"
          density="compact"
          class="monitor-table"
          :no-data-text="t('platform.metrics.noProcess')"
          @update:sort-by="onProcessSort"
        >
          <template #item.name="{ item }">
            <div class="d-flex align-center ga-2">
              <v-icon
                :color="roleColor(item.role)"
                size="10"
              >
                mdi-circle
              </v-icon>
              <span class="process-name">{{ item.name }}</span>
            </div>
          </template>
          <template #item.role="{ item }">
            <v-chip
              :color="roleColor(item.role)"
              size="x-small"
              variant="tonal"
            >
              {{ roleLabel(item) }}
            </v-chip>
          </template>
          <template #item.cpu="{ item }">
            <span :class="{ 'text-error': item.cpu >= 90 }">
              {{ formatNumber(item.cpu) }}%
            </span>
          </template>
          <template #item.rss="{ item }">
            {{ formatBytes(item.rss) }}
          </template>
          <template #item.memoryPercent="{ item }">
            {{ formatNumber(item.memoryPercent) }}%
          </template>
          <template #item.status="{ item }">
            <v-chip
              size="x-small"
              variant="text"
            >
              {{ item.status }}
            </v-chip>
          </template>
          <template #item.uptime="{ item }">
            {{ formatUptime(item.uptime) }}
          </template>
        </v-data-table>
        <div class="text-caption text-medium-emphasis mt-2">
          {{ t('platform.metrics.processesMatched', { matched: processData.matched }) }}
        </div>
      </v-card-text>
    </v-card>
  </div>
</template>

<script setup>
import platformApi from '@/api/platform.js'
import { useDisplay, useTheme  } from 'vuetify/framework'
import { useI18n } from 'vue-i18n'

const { mobile } = useDisplay()
const { t } = useI18n()
const theme = useTheme()

const range = ref('15m')
const autoRefresh = ref(true)
const current = ref(null)
const history = ref({ points: [], available: false, downsampled: false })
const processData = ref({ processes: [], total: 0, matched: 0, partial: false })
const processSearch = ref('')
const processSort = ref('cpu')
const processOrder = ref('desc')
const loadingCurrent = ref(false)
const loadingHistory = ref(false)
const loadingProcesses = ref(false)
const errorMessage = ref('')
const lastUpdated = ref(null)

let currentTimer
let historyTimer
let processTimer
let searchTimer
let historyRequest = 0
let processRequest = 0

const rangeOptions = computed(() => [
  { label: t('platform.metrics.range.m15'), value: '15m' },
  { label: t('platform.metrics.range.h1'), value: '1h' },
  { label: t('platform.metrics.range.h24'), value: '24h' },
])

const sortOptions = computed(() => [
  { label: t('platform.metrics.sortCpu'), value: 'cpu' },
  { label: t('platform.metrics.sortMemory'), value: 'memory' },
  { label: t('platform.metrics.sortName'), value: 'name' },
])

const processHeaders = computed(() => [
  { title: t('platform.metrics.pid'), key: 'pid', width: 96, minWidth: 96, sortable: false },
  { title: t('platform.metrics.name'), key: 'name', minWidth: 220, sortable: true },
  { title: t('platform.metrics.role'), key: 'role', width: 150, sortable: false },
  { title: t('platform.metrics.cpu'), key: 'cpu', width: 100, align: 'end', sortable: true },
  { title: t('platform.metrics.memory'), key: 'rss', width: 120, align: 'end', sortable: true },
  { title: `${t('platform.metrics.memory')} %`, key: 'memoryPercent', width: 110, align: 'end', sortable: false },
  { title: t('platform.metrics.statusLabel'), key: 'status', width: 100, sortable: false },
  { title: t('platform.metrics.uptime'), key: 'uptime', width: 120, align: 'end', sortable: false },
])

const processSortBy = computed(() => [{ key: processSort.value === 'memory' ? 'rss' : processSort.value, order: processOrder.value }])
const processes = computed(() => processData.value?.processes || [])
const processTotal = computed(() => processData.value?.total || 0)
const snapshot = computed(() => current.value?.snapshot || null)
const warnings = computed(() => snapshot.value?.warnings || [])
const points = computed(() => history.value?.points || [])
const isDark = computed(() => theme.global.current.value.dark)
const chartHeight = computed(() => (mobile.value ? '220px' : '280px'))
const refreshing = computed(() => loadingCurrent.value || loadingHistory.value || loadingProcesses.value)

const lastUpdatedText = computed(() => {
  if (!lastUpdated.value) return '—'

  return formatDateTime(Math.floor(lastUpdated.value.getTime() / 1000))
})

const cpuCores = computed(() => snapshot.value?.cpu.cores || points.value[0]?.perCore?.length || 0)
const chartText = computed(() => (isDark.value ? '#cfd8dc' : '#546e7a'))
const chartMuted = computed(() => (isDark.value ? '#90a4ae' : '#78909c'))
const gridColor = computed(() => (isDark.value ? 'rgba(255,255,255,0.12)' : 'rgba(0,0,0,0.08)'))
const axisLineColor = computed(() => (isDark.value ? 'rgba(255,255,255,0.24)' : 'rgba(0,0,0,0.16)'))
const palette = ['#409EFF', '#67C23A', '#E6A23C', '#F56C6C', '#8C57FF', '#36CE9E', '#edb00d', '#909399']

const summaryCards = computed(() => {
  const s = snapshot.value
  const disk = s?.disks?.find(item => item.mountpoint === '/') || s?.disks?.[0]
  const memory = s?.memory
  const swap = s?.swap
  const network = s?.network
  const cpu = s?.cpu

  return [
    {
      key: 'cpu',
      label: t('platform.metrics.overview.cpu'),
      value: s ? `${formatNumber(cpu.usage)}%` : '—',
      sub: s ? `${cpu.cores} vCPU · ${t('platform.metrics.load')} ${formatNumber(cpu.load1)}` : t('platform.metrics.collecting'),
      percent: s ? cpu.usage : null,
      color: s ? valueColor(cpu.usage) : 'grey',
      icon: 'mdi-cpu-64-bit',
    },
    {
      key: 'memory',
      label: t('platform.metrics.overview.memory'),
      value: s ? `${formatNumber(memory.usedPercent)}%` : '—',
      sub: s ? `${formatBytes(memory.used)} / ${formatBytes(memory.total)} · ${t('platform.metrics.buffCache')} ${formatBytes(memory.buffCache)}` : '',
      percent: s ? memory.usedPercent : null,
      color: s ? valueColor(memory.usedPercent) : 'grey',
      icon: 'mdi-memory',
    },
    {
      key: 'swap',
      label: t('platform.metrics.overview.swap'),
      value: s ? `${formatNumber(swap.usedPercent)}%` : '—',
      sub: s ? `${formatBytes(swap.used)} / ${formatBytes(swap.total)}` : '',
      percent: s ? swap.usedPercent : null,
      color: s ? valueColor(swap.usedPercent) : 'grey',
      icon: 'mdi-swap-horizontal',
    },
    {
      key: 'disk',
      label: t('platform.metrics.overview.disk'),
      value: disk ? `${formatNumber(disk.usedPercent)}%` : '—',
      sub: disk ? `${disk.mountpoint} · ${formatBytes(disk.used)} / ${formatBytes(disk.total)}` : '',
      percent: disk ? disk.usedPercent : null,
      color: disk ? valueColor(disk.usedPercent) : 'grey',
      icon: 'mdi-harddisk',
    },
    {
      key: 'upload',
      label: t('platform.metrics.overview.upload'),
      value: network ? `${formatNumber(network.upMbps)} Mbps` : '—',
      sub: network ? `${formatNumber(network.upMBps)} MB/s` : '',
      percent: null,
      color: 'info',
      icon: 'mdi-upload',
    },
    {
      key: 'download',
      label: t('platform.metrics.overview.download'),
      value: network ? `${formatNumber(network.downMbps)} Mbps` : '—',
      sub: network ? `${formatNumber(network.downMBps)} MB/s` : '',
      percent: null,
      color: 'success',
      icon: 'mdi-download',
    },
    {
      key: 'uptime',
      label: t('platform.metrics.overview.uptime'),
      value: s ? formatUptime(s.uptime) : '—',
      sub: s ? `${s.host.hostname || ''} ${s.host.platform || ''}`.trim() : '',
      percent: null,
      color: 'default',
      icon: 'mdi-clock-outline',
    },
    {
      key: 'processes',
      label: t('platform.metrics.overview.processes'),
      value: s ? String(s.processTotal) : '—',
      sub: s?.processPartial ? t('platform.metrics.partial') : '',
      percent: null,
      color: 'default',
      icon: 'mdi-application-cog',
    },
  ]
})

function formatNumber(value, digits = 2) {
  const number = Number(value)
  if (!Number.isFinite(number)) return '0'

  return number.toFixed(digits)
}

function formatBytes(bytes) {
  const number = Number(bytes)
  if (!Number.isFinite(number) || number <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = number
  let index = 0
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024
    index++
  }

  return `${value.toFixed(value >= 100 ? 0 : 1)} ${units[index]}`
}

function formatUptime(seconds) {
  const total = Math.max(0, Math.floor(Number(seconds) || 0))
  const days = Math.floor(total / 86400)
  const hours = Math.floor((total % 86400) / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  if (days > 0) return `${days}d ${hours}h ${minutes}m`
  if (hours > 0) return `${hours}h ${minutes}m`

  return `${minutes}m`
}

function formatDateTime(timestamp) {
  if (!timestamp) return '—'
  const date = new Date(timestamp * 1000)

  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

function formatAxisTime(timestamp) {
  const date = new Date(Number(timestamp))
  if (range.value === '24h') {
    return `${date.getMonth() + 1}/${date.getDate()} ${pad(date.getHours())}:${pad(date.getMinutes())}`
  }

  return `${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function pad(value) {
  return String(value).padStart(2, '0')
}

function valueColor(value) {
  if (value >= 90) return 'error'
  if (value >= 70) return 'warning'

  return 'success'
}

function roleColor(role) {
  switch (role) {
    case 'dmp':
      return 'primary'
    case 'dst-master':
      return 'success'
    case 'dst-caves':
      return 'info'
    case 'dst':
      return 'warning'
    default:
      return 'grey'
  }
}

function roleLabel(process) {
  const map = {
    'dmp': t('platform.metrics.roleDmp'),
    'dst-master': t('platform.metrics.roleMaster'),
    'dst-caves': t('platform.metrics.roleCaves'),
    'dst': t('platform.metrics.roleDst'),
    'other': t('platform.metrics.roleOther'),
  }

  const label = map[process.role] || process.role || t('platform.metrics.roleOther')

  return `${label}${process.cluster ? ` · ${process.cluster}` : ''}`
}

function series(name, data, color, options = {}) {
  return {
    name,
    type: 'line',
    showSymbol: false,
    smooth: true,
    sampling: 'lttb',
    yAxisIndex: options.yAxisIndex || 0,
    data,
    lineStyle: {
      width: 2,
      color,
      type: options.dashed ? 'dashed' : 'solid',
    },
    itemStyle: { color },
    areaStyle: options.area ? { color, opacity: 0.1 } : undefined,
  }
}

function percentAxis() {
  return {
    type: 'value',
    min: 0,
    max: 100,
    axisLabel: { color: chartMuted.value, fontSize: 10, formatter: '{value}%' },
    splitLine: { lineStyle: { color: gridColor.value } },
  }
}

function valueAxis(unit = '') {
  return {
    type: 'value',
    min: 0,
    axisLabel: {
      color: chartMuted.value,
      fontSize: 10,
      formatter: value => `${value}${unit}`,
    },
    splitLine: { lineStyle: { color: gridColor.value } },
  }
}

function baseOption(title, seriesList, yAxis, tooltipFormatter) {
  return {
    backgroundColor: 'transparent',
    animation: false,
    title: {
      text: title,
      left: 0,
      top: 0,
      textStyle: { color: chartText.value, fontSize: 13, fontWeight: 600 },
    },
    tooltip: {
      trigger: 'axis',
      confine: true,
      backgroundColor: isDark.value ? 'rgba(30,30,30,0.96)' : 'rgba(255,255,255,0.98)',
      borderColor: isDark.value ? '#546e7a' : '#cfd8dc',
      textStyle: { color: chartText.value, fontSize: 12 },
      formatter: tooltipFormatter,
    },
    legend: {
      type: 'scroll',
      top: 24,
      left: 0,
      right: 0,
      itemWidth: 12,
      itemHeight: 8,
      textStyle: { color: chartText.value, fontSize: 11 },
    },
    grid: { left: 6, right: 10, top: 62, bottom: 4, containLabel: true },
    xAxis: {
      type: 'time',
      axisLine: { lineStyle: { color: axisLineColor.value } },
      axisLabel: { color: chartMuted.value, fontSize: 10, hideOverlap: true, formatter: formatAxisTime },
      splitLine: { show: false },
    },
    yAxis,
    series: seriesList,
  }
}

function tooltipRows(params, unit = '%') {
  if (!params?.length) return ''
  let html = formatDateTime(Math.floor(params[0].value[0] / 1000))
  for (const item of params) {
    const value = Number(item.value[1])

    html += `<br/>${item.marker}${item.seriesName}: ${formatNumber(value)}${unit}`
  }

  return html
}

const cpuChartOption = computed(() => {
  const list = points.value

  return baseOption(
    t('platform.metrics.cpuUsage'),
    [
      series(t('platform.metrics.cpuUsage'), list.map(item => [item.timestamp * 1000, item.cpu]), palette[0], { area: true }),
      series(t('platform.metrics.load1'), list.map(item => [item.timestamp * 1000, item.load1]), palette[2], { yAxisIndex: 1 }),
    ],
    [percentAxis(), valueAxis()],
    params => {
      if (!params?.length) return ''
      const loadName = t('platform.metrics.load1')
      let html = formatDateTime(Math.floor(params[0].value[0] / 1000))
      for (const item of params) {
        const unit = item.seriesName === loadName ? '' : '%'

        html += `<br/>${item.marker}${item.seriesName}: ${formatNumber(item.value[1])}${unit}`
      }

      return html
    },
  )
})

const coreIndexes = computed(() => {
  const count = Math.min(4, Math.max(cpuCores.value, points.value[0]?.perCore?.length || 0))

  return Array.from({ length: count }, (_, index) => index)
})

function coreLatest(index) {
  const last = points.value.at(-1)

  return last?.perCore?.[index] ?? 0
}

function perCoreOption(index) {
  const list = points.value

  const option = baseOption(
    '',
    [series(`CPU ${index + 1}`, list.map(item => [item.timestamp * 1000, item.perCore?.[index] ?? 0]), palette[index % palette.length], { area: true })],
    [percentAxis()],
    params => tooltipRows(params, '%'),
  )

  option.legend.show = false
  option.grid.top = 10

  return option
}

const stealChartOption = computed(() => {
  const list = points.value

  return baseOption(
    t('platform.metrics.stealIowait'),
    [
      series(t('platform.metrics.iowait'), list.map(item => [item.timestamp * 1000, item.iowait]), palette[2], { area: true }),
      series(t('platform.metrics.steal'), list.map(item => [item.timestamp * 1000, item.steal]), palette[3]),
    ],
    [percentAxis()],
    params => tooltipRows(params, '%'),
  )
})

const memoryChartOption = computed(() => {
  const list = points.value

  return baseOption(
    t('platform.metrics.memorySwap'),
    [
      series(t('platform.metrics.memoryPct'), list.map(item => [item.timestamp * 1000, item.memoryPercent]), palette[1], { area: true }),
      series(t('platform.metrics.swapPct'), list.map(item => [item.timestamp * 1000, item.swapPercent]), palette[4]),
    ],
    [percentAxis()],
    params => tooltipRows(params, '%'),
  )
})

const diskChartOption = computed(() => {
  const list = points.value

  return baseOption(
    t('platform.metrics.diskIo'),
    [
      series(t('platform.metrics.read'), list.map(item => [item.timestamp * 1000, item.diskReadBps / 1024 / 1024]), palette[0], { area: true }),
      series(t('platform.metrics.write'), list.map(item => [item.timestamp * 1000, item.diskWriteBps / 1024 / 1024]), palette[3]),
    ],
    [valueAxis(' MB/s')],
    params => tooltipRows(params, ' MB/s'),
  )
})

const networkChartOption = computed(() => {
  const list = points.value

  return baseOption(
    t('platform.metrics.netTraffic'),
    [
      series(t('platform.metrics.netUp'), list.map(item => [item.timestamp * 1000, item.netUpBps * 8 / 1_000_000]), palette[2], { area: true }),
      series(t('platform.metrics.netDown'), list.map(item => [item.timestamp * 1000, item.netDownBps * 8 / 1_000_000]), palette[1], { area: true }),
    ],
    [valueAxis(' Mbps')],
    params => tooltipRows(params, ' Mbps'),
  )
})

const dstChartOption = computed(() => {
  const list = points.value

  const cpuSeries = [
    series(t('platform.metrics.master') + ' CPU', list.map(item => [item.timestamp * 1000, item.masterCpu]), palette[1]),
    series(t('platform.metrics.caves') + ' CPU', list.map(item => [item.timestamp * 1000, item.cavesCpu]), palette[2]),
    series(t('platform.metrics.dmp') + ' CPU', list.map(item => [item.timestamp * 1000, item.dmpCpu]), palette[0]),
  ]

  const memorySeries = [
    series(t('platform.metrics.master') + ' RSS', list.map(item => [item.timestamp * 1000, item.masterRss / 1024 / 1024]), palette[1], { dashed: true, yAxisIndex: 1 }),
    series(t('platform.metrics.caves') + ' RSS', list.map(item => [item.timestamp * 1000, item.cavesRss / 1024 / 1024]), palette[2], { dashed: true, yAxisIndex: 1 }),
    series(t('platform.metrics.dmp') + ' RSS', list.map(item => [item.timestamp * 1000, item.dmpRss / 1024 / 1024]), palette[0], { dashed: true, yAxisIndex: 1 }),
  ]

  return baseOption(
    t('platform.metrics.dstResources'),
    [...cpuSeries, ...memorySeries],
    [percentAxis(), valueAxis(' MB')],
    params => {
      if (!params?.length) return ''
      let html = formatDateTime(Math.floor(params[0].value[0] / 1000))
      for (const item of params) {
        const value = Number(item.value[1])
        const unit = item.seriesName.endsWith('RSS') ? ' MB' : '%'

        html += `<br/>${item.marker}${item.seriesName}: ${formatNumber(value)}${unit}`
      }

      return html
    },
  )
})

function mergeCurrentIntoHistory(data) {
  if (range.value === '24h' || !data?.snapshot) return
  const s = data.snapshot
  const list = history.value?.points || []
  if (list.length && list[list.length - 1].timestamp >= s.timestamp) return
  let masterCpu = 0
  let masterRss = 0
  let cavesCpu = 0
  let cavesRss = 0
  let dmpCpu = 0
  let dmpRss = 0
  for (const process of s.processes || []) {
    if (process.role === 'dst-master') {
      masterCpu += process.cpu
      masterRss += process.rss
    } else if (process.role === 'dst-caves') {
      cavesCpu += process.cpu
      cavesRss += process.rss
    } else if (process.role === 'dmp') {
      dmpCpu += process.cpu
      dmpRss += process.rss
    }
  }

  const point = {
    timestamp: s.timestamp,
    cpu: s.cpu.usage,
    perCore: s.cpu.perCore,
    user: s.cpu.user,
    system: s.cpu.system,
    idle: s.cpu.idle,
    iowait: s.cpu.iowait,
    steal: s.cpu.steal,
    load1: s.cpu.load1,
    memoryUsed: s.memory.used,
    memoryPercent: s.memory.usedPercent,
    swapUsed: s.swap.used,
    swapPercent: s.swap.usedPercent,
    diskReadBps: s.diskIo.readBytesPerSec,
    diskWriteBps: s.diskIo.writeBytesPerSec,
    netUpBps: s.network.upBps,
    netDownBps: s.network.downBps,
    masterCpu,
    masterRss,
    cavesCpu,
    cavesRss,
    dmpCpu,
    dmpRss,
  }

  const maxPoints = range.value === '15m' ? 360 : 480
  const next = [...list, point]
  if (next.length > maxPoints) next.splice(0, next.length - maxPoints)
  history.value = { ...history.value, points: next, available: true }
}

async function loadCurrent() {
  if (loadingCurrent.value) return
  loadingCurrent.value = true
  try {
    const response = await platformApi.monitorCurrent.get()

    current.value = response.data || null
    lastUpdated.value = new Date()
    errorMessage.value = ''
    mergeCurrentIntoHistory(response.data)
  } catch (error) {
    errorMessage.value = error?.message || String(error)
  } finally {
    loadingCurrent.value = false
  }
}

async function loadHistory() {
  loadingHistory.value = true

  const requestedRange = range.value
  const requestId = ++historyRequest
  try {
    const response = await platformApi.monitorHistory.get({ range: requestedRange })
    if (requestId !== historyRequest || requestedRange !== range.value) return
    history.value = response.data || { points: [], available: false, downsampled: false }
  } catch (error) {
    if (requestId === historyRequest) errorMessage.value = error?.message || String(error)
  } finally {
    if (requestId === historyRequest) loadingHistory.value = false
  }
}

async function loadProcesses() {
  const requestId = ++processRequest

  loadingProcesses.value = true
  try {
    const response = await platformApi.monitorProcesses.get({
      sort: processSort.value,
      order: processOrder.value,
      q: processSearch.value || '',
      limit: 300,
    })

    if (requestId !== processRequest) return
    processData.value = response.data || { processes: [], total: 0, matched: 0, partial: false }
  } catch (error) {
    if (requestId === processRequest) errorMessage.value = error?.message || String(error)
  } finally {
    if (requestId === processRequest) loadingProcesses.value = false
  }
}

function onRangeChange() {
  history.value = { ...history.value, points: [], available: false }
  loadHistory()
}

function onProcessSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(loadProcesses, 350)
}

function onProcessSort(sortBy) {
  const next = Array.isArray(sortBy) ? sortBy[0] : sortBy
  if (!next?.key) return
  if (next.key === 'rss' || next.key === 'memoryPercent') {
    processSort.value = 'memory'
  } else if (next.key === 'name') {
    processSort.value = 'name'
  } else {
    processSort.value = 'cpu'
  }
  processOrder.value = next.order === 'asc' ? 'asc' : 'desc'
  loadProcesses()
}

function refreshAll() {
  loadCurrent()
  loadHistory()
  loadProcesses()
}

function startTimers() {
  clearInterval(currentTimer)
  clearInterval(historyTimer)
  clearInterval(processTimer)
  currentTimer = setInterval(() => {
    if (autoRefresh.value && !document.hidden) loadCurrent()
  }, 5000)
  historyTimer = setInterval(() => {
    if (autoRefresh.value && !document.hidden) loadHistory()
  }, 10000)
  processTimer = setInterval(() => {
    if (autoRefresh.value && !document.hidden) loadProcesses()
  }, 10000)
}

function onVisibilityChange() {
  if (!document.hidden && autoRefresh.value) refreshAll()
}

onMounted(() => {
  refreshAll()
  startTimers()
  document.addEventListener('visibilitychange', onVisibilityChange)
})

onUnmounted(() => {
  clearInterval(currentTimer)
  clearInterval(historyTimer)
  clearInterval(processTimer)
  clearTimeout(searchTimer)
  document.removeEventListener('visibilitychange', onVisibilityChange)
})
</script>

<style scoped>
.monitor-page {
  padding-bottom: 24px;
}

.monitor-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.toolbar-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.metric-card,
.chart-card {
  border-radius: 10px;
}

.metric-value {
  margin-top: 6px;
  font-size: 1.45rem;
  font-weight: 600;
  line-height: 1.3;
}

.metric-sub {
  min-height: 18px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chart-card :deep(.v-card-title) {
  padding-bottom: 0;
}

.process-search {
  max-width: 240px;
  min-width: 180px;
}

.process-sort {
  max-width: 140px;
  min-width: 110px;
}

.process-name {
  display: inline-block;
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.monitor-table :deep(th) {
  white-space: nowrap;
}

.monitor-table :deep(td:first-child) {
  white-space: nowrap;
}

@media (max-width: 600px) {
  .monitor-toolbar {
    align-items: stretch;
  }

  .toolbar-actions {
    width: 100%;
  }

  .process-search,
  .process-sort {
    max-width: none;
    width: 100%;
  }

  .process-name {
    max-width: 150px;
  }
}
</style>
