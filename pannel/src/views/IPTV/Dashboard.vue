<script setup lang="ts">
import { ContentWrap } from '@/components/ContentWrap'
import { ElRow, ElCol, ElCard, ElStatistic, ElTable, ElTableColumn, ElTag } from 'element-plus'
import { ref, onMounted, onUnmounted } from 'vue'
import request from '@/axios'

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

// Recent channels
const recentChannels = ref([])
const loading = ref(false)

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

// Load bandwidth (mock for now since backend doesn't have this endpoint)
const loadBandwidth = async () => {
  bandwidth.value = {
    downloadMbps: 0,
    uploadMbps: 0,
    totalDownloadMB: 0,
    totalUploadMB: 0
  }
}

// Load recent channels
const loadRecentChannels = async () => {
  loading.value = true
  try {
    const res = await request.get({ url: '/api/recently-watched' })
    if (res && res.data) {
      recentChannels.value = res.data
    }
  } catch (error) {
    console.error('Error loading channels:', error)
    recentChannels.value = []
  } finally {
    loading.value = false
  }
}

let bandwidthInterval: any = null

onMounted(() => {
  loadStats()
  loadBandwidth()
  loadRecentChannels()

  // Update bandwidth every 3 seconds
  bandwidthInterval = setInterval(() => {
    loadBandwidth()
  }, 3000)
})

onUnmounted(() => {
  if (bandwidthInterval) {
    clearInterval(bandwidthInterval)
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
      <ElRow :gutter="15">
        <ElCol :xs="12" :sm="6">
          <ElStatistic title="Download (Mbps)" :value="bandwidth.downloadMbps" :precision="2" />
        </ElCol>
        <ElCol :xs="12" :sm="6">
          <ElStatistic title="Upload (Mbps)" :value="bandwidth.uploadMbps" :precision="2" />
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

    <!-- Recent Channels -->
    <ElCard>
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <Icon icon="ep:video-play" />
          <span>Recently Watched Channels</span>
        </div>
      </template>
      <ElTable :data="recentChannels" v-loading="loading" style="width: 100%">
        <ElTableColumn prop="name" label="Channel Name" min-width="200" />
        <ElTableColumn prop="category" label="Category" width="150" />
        <ElTableColumn label="Status" width="100">
          <template #default="{ row }">
            <ElTag v-if="row.enabled" type="success">Active</ElTag>
            <ElTag v-else type="info">Inactive</ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="playlist_name" label="Playlist" width="150" />
      </ElTable>
    </ElCard>
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
