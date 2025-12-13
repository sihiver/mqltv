<script setup lang="ts">
import { ContentWrap } from '@/components/ContentWrap'
import { ElRow, ElCol, ElCard, ElStatistic, ElTable, ElTableColumn, ElTag } from 'element-plus'
import { ref, onMounted, onUnmounted, computed } from 'vue'
import request from '@/axios'
import * as echarts from 'echarts'
import type { EChartsOption } from 'echarts'

// Stats data
const stats = ref({
  total_playlists: 0,
  total_channels: 0,
  active_channels: 0,
  total_relays: 0
})

// Bandwidth data
const bandwidth = ref({
  downloadMbps: 0,
  uploadMbps: 0,
  totalDownloadMB: 0,
  totalUploadMB: 0
})

// Previous bandwidth values for rate calculation
let prevBytesRead = 0
let prevBytesWritten = 0
let prevTimestamp = Date.now()

// Chart data
const timeLabels = ref<string[]>([])
const downloadData = ref<number[]>([])
const uploadData = ref<number[]>([])
const maxDataPoints = 20 // Keep last 20 data points

let bandwidthChart: echarts.ECharts | null = null

// Recent channels
const recentChannels = ref([])
const activeChannels = ref([])
const loading = ref(false)
const activeLoading = ref(false)

// Load stats
const loadStats = async () => {
  try {
    const res = await request.get({ url: '/api/stats' })
    if (res && res.data) {
      stats.value = res.data
    }
  } catch (error) {
    console.error('Error loading stats:', error)
    // Set default values on error
    stats.value = {
      total_playlists: 0,
      total_channels: 0,
      active_channels: 0,
      total_relays: 0
    }
  }
}

// Load bandwidth from stream status
const loadBandwidth = async () => {
  try {
    const res = await request.get({ url: '/api/streams/status' })
    if (res && res.data) {
      const bytesRead = res.data.total_bytes_read || 0
      const bytesWritten = res.data.total_bytes_write || 0
      const now = Date.now()

      // Calculate rates (Mbps = megabits per second)
      const timeDelta = (now - prevTimestamp) / 1000 // seconds
      const downloadRate =
        timeDelta > 0 ? ((bytesRead - prevBytesRead) * 8) / 1024 / 1024 / timeDelta : 0
      const uploadRate =
        timeDelta > 0 ? ((bytesWritten - prevBytesWritten) * 8) / 1024 / 1024 / timeDelta : 0

      bandwidth.value = {
        downloadMbps: downloadRate.toFixed(2),
        uploadMbps: uploadRate.toFixed(2),
        totalDownloadMB: (bytesRead / 1024 / 1024).toFixed(2),
        totalUploadMB: (bytesWritten / 1024 / 1024).toFixed(2)
      }

      // Update chart data
      const currentTime = new Date().toLocaleTimeString('en-US', {
        hour12: false,
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
      })

      timeLabels.value.push(currentTime)
      downloadData.value.push(parseFloat(downloadRate.toFixed(2)))
      uploadData.value.push(parseFloat(uploadRate.toFixed(2)))

      // Keep only last N data points
      if (timeLabels.value.length > maxDataPoints) {
        timeLabels.value.shift()
        downloadData.value.shift()
        uploadData.value.shift()
      }

      // Update chart
      updateChart()

      // Update previous values
      prevBytesRead = bytesRead
      prevBytesWritten = bytesWritten
      prevTimestamp = now
    }
  } catch (error) {
    console.error('Error loading bandwidth:', error)
    bandwidth.value = {
      downloadMbps: 0,
      uploadMbps: 0,
      totalDownloadMB: 0,
      totalUploadMB: 0
    }
  }
}

// Load recent channels
const loadRecentChannels = async () => {
  loading.value = true
  try {
    const res = await request.get({ url: '/api/recently-watched' })
    console.log('Recently watched response:', res)
    if (res && res.data) {
      recentChannels.value = Array.isArray(res.data) ? res.data : []
      console.log('Recently watched channels:', recentChannels.value)
    } else {
      recentChannels.value = []
    }
  } catch (error) {
    console.error('Error loading recently watched channels:', error)
    recentChannels.value = []
  } finally {
    loading.value = false
  }
}

// Load currently active channels with viewer counts
const loadActiveChannels = async () => {
  try {
    const res = await request.get({ url: '/api/active-channels' })
    console.log('Active channels response:', res)
    if (res && Array.isArray(res.data)) {
      activeChannels.value = res.data
      console.log('Active channels:', activeChannels.value)
    } else {
      activeChannels.value = []
    }
  } catch (error) {
    console.error('Error loading active channels:', error)
    activeChannels.value = []
  }
}

// Initialize chart
const initChart = () => {
  const chartDom = document.getElementById('bandwidthChart')
  if (chartDom) {
    bandwidthChart = echarts.init(chartDom)
    updateChart()
  }
}

// Update chart
const updateChart = () => {
  if (!bandwidthChart) return

  const option: EChartsOption = {
    title: {
      text: 'Real-time Bandwidth',
      left: 'center',
      textStyle: {
        fontSize: 14
      }
    },
    tooltip: {
      trigger: 'axis',
      formatter: (params: any) => {
        let result = params[0].name + '<br/>'
        params.forEach((item: any) => {
          result += `${item.marker} ${item.seriesName}: ${item.value} Mbps<br/>`
        })
        return result
      }
    },
    legend: {
      data: ['Download', 'Upload'],
      top: 30
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: timeLabels.value,
      axisLabel: {
        rotate: 45,
        fontSize: 10
      }
    },
    yAxis: {
      type: 'value',
      name: 'Mbps',
      axisLabel: {
        formatter: '{value}'
      }
    },
    series: [
      {
        name: 'Download',
        type: 'line',
        smooth: true,
        data: downloadData.value,
        itemStyle: {
          color: '#67C23A'
        },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            {
              offset: 0,
              color: 'rgba(103, 194, 58, 0.3)'
            },
            {
              offset: 1,
              color: 'rgba(103, 194, 58, 0.05)'
            }
          ])
        }
      },
      {
        name: 'Upload',
        type: 'line',
        smooth: true,
        data: uploadData.value,
        itemStyle: {
          color: '#409EFF'
        },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            {
              offset: 0,
              color: 'rgba(64, 158, 255, 0.3)'
            },
            {
              offset: 1,
              color: 'rgba(64, 158, 255, 0.05)'
            }
          ])
        }
      }
    ]
  }

  bandwidthChart.setOption(option)
}

let bandwidthInterval: any = null
let statsInterval: any = null

onMounted(() => {
  loadStats()
  loadBandwidth()
  loadRecentChannels()
  loadActiveChannels()

  // Initialize chart after DOM is ready
  setTimeout(() => {
    initChart()
  }, 100)

  // Update bandwidth every 3 seconds
  bandwidthInterval = setInterval(() => {
    loadBandwidth()
  }, 3000)

  // Update stats and active channels every 5 seconds
  statsInterval = setInterval(() => {
    loadStats()
    loadActiveChannels()
  }, 5000)
})

onUnmounted(() => {
  if (bandwidthInterval) {
    clearInterval(bandwidthInterval)
  }
  if (statsInterval) {
    clearInterval(statsInterval)
  }
  if (bandwidthChart) {
    bandwidthChart.dispose()
  }
})
</script>

<template>
  <ContentWrap title="IPTV Dashboard" message="Monitor and manage your IPTV system">
    <!-- Statistics Cards -->
    <ElRow :gutter="20" style="margin-bottom: 20px">
      <ElCol :xs="12" :sm="12" :md="6">
        <ElCard shadow="hover">
          <ElStatistic title="Total Playlists" :value="stats.total_playlists">
            <template #prefix>
              <Icon icon="ep:folder" />
            </template>
          </ElStatistic>
        </ElCard>
      </ElCol>
      <ElCol :xs="12" :sm="12" :md="6">
        <ElCard shadow="hover">
          <ElStatistic title="Total Channels" :value="stats.total_channels">
            <template #prefix>
              <Icon icon="ep:video-camera" />
            </template>
          </ElStatistic>
        </ElCard>
      </ElCol>
      <ElCol :xs="12" :sm="12" :md="6">
        <ElCard shadow="hover">
          <ElStatistic title="Channels Being Watched" :value="stats.active_channels">
            <template #prefix>
              <Icon icon="ep:video-play" style="color: #67c23a" />
            </template>
          </ElStatistic>
        </ElCard>
      </ElCol>
      <ElCol :xs="12" :sm="12" :md="6">
        <ElCard shadow="hover">
          <ElStatistic title="Total Relays" :value="stats.total_relays">
            <template #prefix>
              <Icon icon="ep:connection" />
            </template>
          </ElStatistic>
        </ElCard>
      </ElCol>
    </ElRow>

    <!-- Bandwidth Monitor -->
    <ElCard style="margin-bottom: 20px">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <Icon icon="ep:odometer" />
          <span>Bandwidth Monitor</span>
        </div>
      </template>

      <!-- Real-time Chart -->
      <div id="bandwidthChart" style="width: 100%; height: 300px; margin-bottom: 20px"></div>

      <!-- Statistics -->
      <ElRow :gutter="15">
        <ElCol :xs="12" :sm="6">
          <ElStatistic title="Download (Mbps)" :value="bandwidth.downloadMbps" :precision="2">
            <template #prefix>
              <Icon icon="ep:download" style="color: #67c23a" />
            </template>
          </ElStatistic>
        </ElCol>
        <ElCol :xs="12" :sm="6">
          <ElStatistic title="Upload (Mbps)" :value="bandwidth.uploadMbps" :precision="2">
            <template #prefix>
              <Icon icon="ep:upload" style="color: #409eff" />
            </template>
          </ElStatistic>
        </ElCol>
        <ElCol :xs="12" :sm="6">
          <ElStatistic
            title="Total Download (MB)"
            :value="bandwidth.totalDownloadMB"
            :precision="2"
          />
        </ElCol>
        <ElCol :xs="12" :sm="6">
          <ElStatistic title="Total Upload (MB)" :value="bandwidth.totalUploadMB" :precision="2" />
        </ElCol>
      </ElRow>
    </ElCard>

    <!-- Active and Recent Channels -->
    <ElRow :gutter="20">
      <ElCol :xs="24" :sm="24" :md="12">
        <ElCard>
          <template #header>
            <div style="display: flex; align-items: center; gap: 8px">
              <Icon icon="ep:video-play" style="color: #67c23a" />
              <span>Currently Being Watched</span>
            </div>
          </template>
          <ElTable :data="activeChannels" v-loading="activeLoading" style="width: 100%">
            <ElTableColumn prop="name" label="Channel Name" min-width="180" />
            <ElTableColumn prop="category" label="Category" width="120" />
            <ElTableColumn label="Viewers" width="100" align="center">
              <template #default="{ row }">
                <ElTag type="success">{{ row.viewer_count }}</ElTag>
              </template>
            </ElTableColumn>
          </ElTable>
        </ElCard>
      </ElCol>
      <ElCol :xs="24" :sm="24" :md="12">
        <ElCard>
          <template #header>
            <div style="display: flex; align-items: center; gap: 8px">
              <Icon icon="ep:video-play" />
              <span>Recently Watched Channels</span>
            </div>
          </template>
          <ElTable :data="recentChannels" v-loading="loading" style="width: 100%">
            <ElTableColumn prop="name" label="Channel Name" min-width="180" />
            <ElTableColumn prop="category" label="Category" width="120" />
            <ElTableColumn label="Status" width="100">
              <template #default="{ row }">
                <ElTag v-if="row.enabled" type="success">Active</ElTag>
                <ElTag v-else type="info">Inactive</ElTag>
              </template>
            </ElTableColumn>
          </ElTable>
        </ElCard>
      </ElCol>
    </ElRow>
  </ContentWrap>
</template>

<style scoped>
.el-card {
  transition: all 0.3s;
}

.el-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}
</style>
