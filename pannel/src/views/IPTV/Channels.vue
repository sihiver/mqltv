<script setup lang="ts">
import { ContentWrap } from '@/components/ContentWrap'
import { Icon } from '@iconify/vue'
import {
  ElTable,
  ElTableColumn,
  ElButton,
  ElTag,
  ElInput,
  ElSelect,
  ElOption,
  ElMessage,
  ElMessageBox,
  ElPagination,
  ElDialog,
  ElForm,
  ElFormItem
} from 'element-plus'
import { ref, computed, onMounted, onUnmounted, watchEffect, watch, nextTick } from 'vue'
import request from '@/axios'
import Hls from 'hls.js'

const channels = ref<any[]>([])
const activeChannelIds = ref<Set<number>>(new Set())
const categories = ref<string[]>([])
const playlists = ref<any[]>([])
const searchQuery = ref('')
const selectedCategory = ref('')
const selectedPlaylist = ref<number | undefined>(undefined)
const selectedChannels = ref<Set<number>>(new Set())
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const tableRef = ref()

// Dialog state
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEditing = ref(false)
const formData = ref({
  id: 0,
  playlist_id: 0,
  name: '',
  url: '',
  logo: '',
  group_name: ''
})

// Rename category dialog state
const renameDialogVisible = ref(false)
const renameForm = ref({
  old_name: '',
  new_name: ''
})

// Playback dialog state
const playbackDialogVisible = ref(false)
const playbackChannel = ref<any>(null)
const playbackUrl = ref('')
const videoRef = ref<HTMLVideoElement | null>(null)
let hls: Hls | null = null

// All filtered channels (before pagination)
const allFilteredChannels = computed(() => {
  let result = channels.value

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(
      (ch: any) =>
        ch.name.toLowerCase().includes(query) ||
        (ch.category && ch.category.toLowerCase().includes(query))
    )
  }

  if (selectedCategory.value) {
    result = result.filter((ch: any) => ch.category === selectedCategory.value)
  }

  if (selectedPlaylist.value !== undefined) {
    result = result.filter((ch: any) => ch.playlist_id === selectedPlaylist.value)
  }

  return result
})

// Paginated channels for display
const filteredChannels = computed(() => {
  const result = allFilteredChannels.value
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return result.slice(start, end)
})

// Update total reactively
watchEffect(() => {
  total.value = allFilteredChannels.value.length
})

const handlePageChange = (page: number) => {
  currentPage.value = page
  // Restore selection state for new page
  syncTableSelection()
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
  syncTableSelection()
}

// Sync table visual selection with selectedChannels Set
const syncTableSelection = () => {
  if (!tableRef.value) return

  setTimeout(() => {
    filteredChannels.value.forEach((ch: any) => {
      const isSelected = selectedChannels.value.has(ch.id)
      tableRef.value.toggleRowSelection(ch, isSelected)
    })
  }, 10)
}

// Reset to page 1 when search or filter changes
watch([searchQuery, selectedCategory, selectedPlaylist], () => {
  currentPage.value = 1
})

const loadChannels = async () => {
  loading.value = true
  try {
    const res = await request.get({ url: '/api/channels/search', params: { q: '' } })
    if (res && res.data) {
      channels.value = res.data

      // Extract categories
      const cats = new Set()
      channels.value.forEach((ch: any) => {
        if (ch.category) cats.add(ch.category)
      })
      categories.value = Array.from(cats).sort() as any
    }
  } catch (error) {
    console.error('Error loading channels:', error)
    ElMessage.error('Failed to load channels')
  } finally {
    loading.value = false
  }
}

const handleSelectionChange = (selection: any[]) => {
  // Element Plus only reports selection for the currently rendered page.
  // Preserve global selection across pagination by only updating IDs on this page.
  const currentPageIds = new Set(filteredChannels.value.map((ch: any) => ch.id))
  const selectedIdsOnPage = new Set(selection.map((ch) => ch.id))

  currentPageIds.forEach((id) => {
    if (!selectedIdsOnPage.has(id)) {
      selectedChannels.value.delete(id)
    }
  })

  selectedIdsOnPage.forEach((id) => {
    selectedChannels.value.add(id)
  })
}

const handleSelectAll = (selection: any[]) => {
  // Header checkbox toggles selection for the current page only by default.
  // We map it to "select all filtered" across pages.
  if (!selection || selection.length === 0) {
    clearSelection()
    return
  }
  selectAllFiltered()
}

const selectAllFiltered = () => {
  allFilteredChannels.value.forEach((ch: any) => {
    selectedChannels.value.add(ch.id)
  })
  // Sync table checkboxes for current page
  if (tableRef.value) {
    filteredChannels.value.forEach((ch: any) => {
      tableRef.value.toggleRowSelection(ch, true)
    })
  }
}

const clearSelection = () => {
  selectedChannels.value.clear()
  // Clear table visual selection
  if (tableRef.value) {
    tableRef.value.clearSelection()
  }
}

const loadPlaylists = async () => {
  try {
    const res = await request.get({ url: '/api/playlists' })
    if (res && res.data) {
      playlists.value = res.data
    }
  } catch (error) {
    console.error('Error loading playlists:', error)
  }
}

const loadActiveChannels = async () => {
  try {
    const res = await request.get({ url: '/api/active-channels' })
    if (res && Array.isArray(res.data)) {
      activeChannelIds.value = new Set(res.data.map((ch: any) => ch.id))
    }
  } catch (error) {
    console.error('Error loading active channels:', error)
  }
}

const handleCreate = () => {
  dialogTitle.value = 'Add New Channel'
  isEditing.value = false
  formData.value = {
    id: 0,
    playlist_id: playlists.value.length > 0 ? playlists.value[0].id : 0,
    name: '',
    url: '',
    logo: '',
    group_name: ''
  }
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  dialogTitle.value = 'Edit Channel'
  isEditing.value = true
  formData.value = {
    id: row.id,
    playlist_id: row.playlist_id,
    name: row.name,
    url: row.url,
    logo: row.logo,
    group_name: row.category || ''
  }
  dialogVisible.value = true
}

const handlePlayback = (row: any) => {
  playbackChannel.value = row
  // Use backend preview proxy (avoids CORS and supports admin session auth)
  playbackUrl.value = `/api/channels/${row.id}/preview`
  playbackDialogVisible.value = true

  // Initialize HLS player after dialog opens
  nextTick(() => {
    initPlayer()
  })
}

const initPlayer = () => {
  const video = videoRef.value
  if (!video) return

  // Clean up existing HLS instance
  if (hls) {
    hls.destroy()
    hls = null
  }

  const streamUrl = playbackUrl.value
  const sourceUrl = playbackChannel.value?.url || ''
  const isHLS = sourceUrl.includes('.m3u8') || sourceUrl.includes('m3u8')

  if (isHLS && Hls.isSupported()) {
    hls = new Hls({
      enableWorker: true,
      lowLatencyMode: true,
      backBufferLength: 90,
      xhrSetup: function (xhr) {
        // Preview endpoint is protected by admin session cookie
        xhr.withCredentials = true
      }
    })

    hls.loadSource(streamUrl)
    hls.attachMedia(video)

    hls.on(Hls.Events.MANIFEST_PARSED, () => {
      video.play().catch((err) => {
        console.warn('Autoplay failed:', err)
      })
    })

    hls.on(Hls.Events.ERROR, (_event, data) => {
      console.error('HLS error:', data)
      if (data.fatal) {
        switch (data.type) {
          case Hls.ErrorTypes.NETWORK_ERROR:
            ElMessage.warning('Network error - trying native player')
            // Fallback to native player
            if (hls) {
              hls.destroy()
              hls = null
            }
            video.src = streamUrl
            video.play().catch(() => {
              ElMessage.error('Playback failed - stream may be unavailable')
            })
            break
          case Hls.ErrorTypes.MEDIA_ERROR:
            ElMessage.error('Media error: Trying to recover')
            hls?.recoverMediaError()
            break
          default:
            ElMessage.error('Fatal error: Cannot play stream')
            hls?.destroy()
            break
        }
      }
    })
  } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
    // Native HLS support (Safari)
    video.src = streamUrl
    video.addEventListener('loadedmetadata', () => {
      video.play().catch((err) => {
        console.warn('Autoplay failed:', err)
      })
    })
  } else {
    // For non-HLS streams or browsers without HLS support, use native player
    video.src = streamUrl
    video.play().catch((err) => {
      ElMessage.error('Playback failed: ' + err.message)
    })
  }
}

const closePlayback = () => {
  // Stop and destroy HLS instance
  if (hls) {
    hls.destroy()
    hls = null
  }

  // Reset video element
  if (videoRef.value) {
    videoRef.value.pause()
    videoRef.value.src = ''
  }

  playbackDialogVisible.value = false
  playbackChannel.value = null
  playbackUrl.value = ''
}

const handleSave = async () => {
  if (!formData.value.name || !formData.value.url) {
    ElMessage.warning('Name and URL are required')
    return
  }

  try {
    if (isEditing.value) {
      // Update existing channel
      const res = await request.put({
        url: `/api/channels/${formData.value.id}`,
        data: {
          name: formData.value.name,
          url: formData.value.url,
          logo: formData.value.logo,
          group_name: formData.value.group_name
        }
      })

      // Update channel in list
      const index = channels.value.findIndex((ch: any) => ch.id === formData.value.id)
      if (index !== -1 && res.data) {
        channels.value[index] = res.data
      }

      ElMessage.success('Channel updated successfully')
    } else {
      // Create new channel
      const res = await request.post({
        url: '/api/channels',
        data: {
          playlist_id: formData.value.playlist_id,
          name: formData.value.name,
          url: formData.value.url,
          logo: formData.value.logo,
          group_name: formData.value.group_name
        }
      })

      // Add new channel to list
      if (res.data) {
        channels.value.unshift(res.data)

        // Update categories if new category
        if (res.data.category && !categories.value.includes(res.data.category)) {
          categories.value.push(res.data.category)
        }
      }

      ElMessage.success('Channel created successfully')
    }

    dialogVisible.value = false
  } catch (error) {
    ElMessage.error(isEditing.value ? 'Failed to update channel' : 'Failed to create channel')
  }
}

const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm(`Delete channel "${row.name}"?`, 'Confirm', {
      confirmButtonText: 'Delete',
      cancelButtonText: 'Cancel',
      type: 'warning'
    })

    await request.delete({ url: `/api/channels/${row.id}` })

    // Instant UI update
    channels.value = channels.value.filter((ch: any) => ch.id !== row.id)

    ElMessage.success('Channel deleted successfully')
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('Failed to delete channel')
    }
  }
}

const handleBatchDelete = async () => {
  if (selectedChannels.value.size === 0) return

  try {
    await ElMessageBox.confirm(
      `Delete ${selectedChannels.value.size} selected channels?`,
      'Confirm',
      {
        confirmButtonText: 'Delete',
        cancelButtonText: 'Cancel',
        type: 'warning'
      }
    )

    const ids = Array.from(selectedChannels.value)
    await request.post({ url: '/api/channels/batch-delete', data: { ids } })

    // Instant UI update - remove deleted channels
    channels.value = channels.value.filter((ch: any) => !ids.includes(ch.id))

    ElMessage.success('Channels deleted successfully')
    selectedChannels.value.clear()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('Failed to delete channels')
    }
  }
}

const openRenameCategory = () => {
  if (!selectedCategory.value) {
    ElMessage.warning('Please select a category first')
    return
  }

  renameForm.value = {
    old_name: selectedCategory.value,
    new_name: selectedCategory.value
  }
  renameDialogVisible.value = true
}

const submitRenameCategory = async () => {
  const oldName = (renameForm.value.old_name || '').trim()
  const newName = (renameForm.value.new_name || '').trim()

  if (!oldName || !newName) {
    ElMessage.warning('Old and new category names are required')
    return
  }

  if (oldName === newName) {
    renameDialogVisible.value = false
    return
  }

  try {
    await ElMessageBox.confirm(`Rename category "${oldName}" to "${newName}"?`, 'Confirm', {
      confirmButtonText: 'Rename',
      cancelButtonText: 'Cancel',
      type: 'warning'
    })

    await request.post({
      url: '/api/channels/rename-category',
      data: { old_name: oldName, new_name: newName }
    })

    // Update local channels list
    channels.value = channels.value.map((ch: any) => {
      if (ch.category === oldName) {
        return { ...ch, category: newName, group_name: newName }
      }
      return ch
    })

    // Update categories list
    categories.value = Array.from(
      new Set(
        (categories.value as any[])
          .filter((c) => c !== oldName)
          .concat([newName])
          .filter((c) => !!c)
      )
    ).sort() as any

    if (selectedCategory.value === oldName) {
      selectedCategory.value = newName
    }

    renameDialogVisible.value = false
    ElMessage.success('Category renamed successfully')
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('Failed to rename category')
    }
  }
}

let activeChannelsInterval: any = null

onMounted(() => {
  loadChannels()
  loadPlaylists()
  loadActiveChannels()

  // Refresh active channels every 5 seconds
  activeChannelsInterval = setInterval(() => {
    loadActiveChannels()
  }, 5000)
})

onUnmounted(() => {
  if (activeChannelsInterval) {
    clearInterval(activeChannelsInterval)
  }

  // Cleanup HLS instance
  if (hls) {
    hls.destroy()
    hls = null
  }
})
</script>

<template>
  <ContentWrap title="Channels" message="Manage IPTV channels">
    <!-- Toolbar -->
    <div style="display: flex; gap: 12px; margin-bottom: 16px; flex-wrap: wrap; align-items: center">
      <ElInput
        v-model="searchQuery"
        placeholder="Search channels..."
        clearable
        style="width: 220px"
      >
        <template #prefix>
          <Icon icon="ep:search" />
        </template>
      </ElInput>

      <ElSelect
        v-model="selectedPlaylist"
        placeholder="All Playlists"
        clearable
        style="width: 200px"
      >
        <ElOption
          v-for="playlist in playlists"
          :key="playlist.id"
          :label="playlist.name"
          :value="playlist.id"
        />
      </ElSelect>

      <ElSelect
        v-model="selectedCategory"
        placeholder="All Categories"
        clearable
        style="width: 200px"
      >
        <ElOption
          v-for="category in categories"
          :key="category"
          :label="category"
          :value="category"
        />
      </ElSelect>

      <div style="margin-left: auto; display: flex; gap: 8px; flex-wrap: wrap">
        <ElButton type="primary" @click="handleCreate">
          <Icon icon="ep:plus" />
          Add Channel
        </ElButton>
        <ElButton :disabled="!selectedCategory" @click="openRenameCategory">
          <Icon icon="ep:edit" />
          Rename Category
        </ElButton>
        <ElButton
          type="danger"
          :disabled="selectedChannels.size === 0"
          @click="handleBatchDelete"
        >
          <Icon icon="ep:delete" />
          Delete ({{ selectedChannels.size }})
        </ElButton>
      </div>
    </div>

    <!-- Channels Table -->
    <ElTable
      ref="tableRef"
      :data="filteredChannels"
      row-key="id"
      v-loading="loading"
      @selection-change="handleSelectionChange"
      @select-all="handleSelectAll"
      style="width: 100%"
    >
      <ElTableColumn type="selection" width="55" />

      <ElTableColumn label="Channel" min-width="250">
        <template #default="{ row }">
          <div style="display: flex; align-items: center; gap: 12px">
            <img
              v-if="row.logo"
              :src="row.logo"
              :alt="row.name"
              style="width: 48px; height: 48px; object-fit: contain; border-radius: 4px"
            />
            <div>
              <div style="font-weight: 600">{{ row.name }}</div>
              <div style="font-size: 12px; color: #909399">{{
                row.category || 'Uncategorized'
              }}</div>
            </div>
          </div>
        </template>
      </ElTableColumn>

      <ElTableColumn prop="playlist_name" label="Playlist" width="150" />

      <ElTableColumn label="Status" width="120">
        <template #default="{ row }">
          <ElTag v-if="activeChannelIds.has(row.id)" type="success">
            <Icon icon="ep:video-play" style="margin-right: 4px" />
            Streaming
          </ElTag>
          <ElTag v-else type="info">
            <Icon icon="ep:video-pause" style="margin-right: 4px" />
            Stopped
          </ElTag>
        </template>
      </ElTableColumn>

      <ElTableColumn label="Actions" width="300" fixed="right">
        <template #default="{ row }">
          <ElButton type="success" size="small" @click="handlePlayback(row)">
            <Icon icon="ep:video-play" />
            Play
          </ElButton>
          <ElButton type="primary" size="small" @click="handleEdit(row)">
            <Icon icon="ep:edit" />
            Edit
          </ElButton>
          <ElButton type="danger" size="small" @click="handleDelete(row)">
            <Icon icon="ep:delete" />
            Delete
          </ElButton>
        </template>
      </ElTableColumn>
    </ElTable>

    <!-- Pagination -->
    <div style="margin-top: 20px; display: flex; justify-content: center">
      <ElPagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handlePageChange"
      />
    </div>

    <!-- Create/Edit Dialog -->
    <ElDialog v-model="dialogVisible" :title="dialogTitle" width="600px">
      <ElForm :model="formData" label-width="120px">
        <ElFormItem label="Playlist">
          <ElSelect
            v-model="formData.playlist_id"
            placeholder="Select Playlist"
            style="width: 100%"
            :disabled="isEditing"
          >
            <ElOption
              v-for="playlist in playlists"
              :key="playlist.id"
              :label="playlist.name"
              :value="playlist.id"
            />
          </ElSelect>
        </ElFormItem>

        <ElFormItem label="Channel Name" required>
          <ElInput v-model="formData.name" placeholder="Enter channel name" />
        </ElFormItem>

        <ElFormItem label="URL" required>
          <ElInput v-model="formData.url" placeholder="http://..." />
        </ElFormItem>

        <ElFormItem label="Logo URL">
          <ElInput v-model="formData.logo" placeholder="http://..." />
        </ElFormItem>

        <ElFormItem label="Category">
          <ElInput v-model="formData.group_name" placeholder="e.g., Sports, News, Entertainment" />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <ElButton @click="dialogVisible = false">Cancel</ElButton>
        <ElButton type="primary" @click="handleSave">Save</ElButton>
      </template>
    </ElDialog>

    <!-- Rename Category Dialog -->
    <ElDialog v-model="renameDialogVisible" title="Rename Category" width="520px">
      <ElForm :model="renameForm" label-width="140px">
        <ElFormItem label="Old Category">
          <ElInput v-model="renameForm.old_name" disabled />
        </ElFormItem>
        <ElFormItem label="New Category" required>
          <ElInput v-model="renameForm.new_name" placeholder="Enter new category name" />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <ElButton @click="renameDialogVisible = false">Cancel</ElButton>
        <ElButton type="primary" @click="submitRenameCategory">Rename</ElButton>
      </template>
    </ElDialog>

    <!-- Playback Dialog -->
    <ElDialog
      v-model="playbackDialogVisible"
      :title="playbackChannel?.name || 'Channel Playback'"
      width="900px"
      @close="closePlayback"
    >
      <div v-if="playbackChannel">
        <div style="margin-bottom: 16px; display: flex; align-items: center; gap: 12px">
          <img
            v-if="playbackChannel.logo"
            :src="playbackChannel.logo"
            :alt="playbackChannel.name"
            style="width: 64px; height: 64px; object-fit: contain; border-radius: 8px"
          />
          <div>
            <div style="font-size: 18px; font-weight: 600">{{ playbackChannel.name }}</div>
            <div style="font-size: 14px; color: #909399">
              {{ playbackChannel.category || 'Uncategorized' }}
            </div>
          </div>
        </div>

        <div
          style="
            background: #000;
            border-radius: 8px;
            overflow: hidden;
            aspect-ratio: 16/9;
            max-height: 500px;
          "
        >
          <video ref="videoRef" controls style="width: 100%; height: 100%; object-fit: contain">
            Your browser does not support the video tag.
          </video>
        </div>

        <div style="margin-top: 16px; font-size: 12px; color: #909399">
          <strong>Stream URL:</strong>
          <code style="padding: 4px 8px; background: #f5f5f5; border-radius: 4px; margin-left: 8px">
            {{ playbackUrl }}
          </code>
        </div>
      </div>

      <template #footer>
        <ElButton @click="closePlayback">Close</ElButton>
      </template>
    </ElDialog>
  </ContentWrap>
</template>
