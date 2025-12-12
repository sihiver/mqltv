<script setup lang="ts">
import { ContentWrap } from '@/components/ContentWrap'
import {
  ElRow,
  ElCol,
  ElTable,
  ElTableColumn,
  ElButton,
  ElTag,
  ElInput,
  ElSelect,
  ElOption,
  ElMessage,
  ElMessageBox,
  ElPagination
} from 'element-plus'
import { ref, computed, onMounted, watch } from 'vue'
import request from '@/axios'

const channels = ref([])
const categories = ref([])
const searchQuery = ref('')
const selectedCategory = ref('')
const selectedChannels = ref<Set<number>>(new Set())
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const tableRef = ref()

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

  total.value = result.length
  return result
})

// Paginated channels for display
const filteredChannels = computed(() => {
  const result = allFilteredChannels.value
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return result.slice(start, end)
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
watch([searchQuery, selectedCategory], () => {
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

const toggleChannel = async (channel: any) => {
  try {
    await request.post({ url: `/api/channels/${channel.id}/toggle` })
    channel.enabled = !channel.enabled
    ElMessage.success(`Channel ${channel.enabled ? 'enabled' : 'disabled'}`)
  } catch (error) {
    ElMessage.error('Failed to toggle channel')
  }
}

const deleteChannel = async (channel: any) => {
  try {
    await ElMessageBox.confirm(`Delete channel "${channel.name}"?`, 'Confirm', {
      confirmButtonText: 'Delete',
      cancelButtonText: 'Cancel',
      type: 'warning'
    })

    await request.delete({ url: `/api/channels/${channel.id}` })

    // Instant UI update - remove deleted channel
    channels.value = channels.value.filter((ch: any) => ch.id !== channel.id)

    ElMessage.success('Channel deleted successfully')
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('Failed to delete channel')
    }
  }
}

const handleSelectionChange = (selection: any[]) => {
  selectedChannels.value = new Set(selection.map((ch) => ch.id))
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

onMounted(() => {
  loadChannels()
})
</script>

<template>
  <ContentWrap title="Channels" message="Manage IPTV channels">
    <!-- Toolbar -->
    <ElRow :gutter="12" style="margin-bottom: 16px">
      <ElCol :xs="24" :sm="12" :md="8">
        <ElInput v-model="searchQuery" placeholder="Search channels..." clearable>
          <template #prefix>
            <Icon icon="ep:search" />
          </template>
        </ElInput>
      </ElCol>

      <ElCol :xs="24" :sm="12" :md="8">
        <ElSelect
          v-model="selectedCategory"
          placeholder="All Categories"
          clearable
          style="width: 100%"
        >
          <ElOption
            v-for="category in categories"
            :key="category"
            :label="category"
            :value="category"
          />
        </ElSelect>
      </ElCol>

      <ElCol :xs="24" :sm="24" :md="8">
        <div style="display: flex; gap: 8px; flex-wrap: wrap">
          <ElButton @click="selectAllFiltered">
            <Icon icon="ep:select" />
            Select All ({{ total }})
          </ElButton>
          <ElButton @click="clearSelection" v-if="selectedChannels.size > 0">
            <Icon icon="ep:close" />
            Clear
          </ElButton>
          <ElButton type="danger" :disabled="selectedChannels.size === 0" @click="handleBatchDelete">
            <Icon icon="ep:delete" />
            Delete ({{ selectedChannels.size }})
          </ElButton>
        </div>
      </ElCol>
    </ElRow>

    <!-- Channels Table -->
    <ElTable
      ref="tableRef"
      :data="filteredChannels"
      row-key="id"
      v-loading="loading"
      @selection-change="handleSelectionChange"
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

      <ElTableColumn label="Status" width="100">
        <template #default="{ row }">
          <ElTag v-if="row.enabled" type="success">Active</ElTag>
          <ElTag v-else type="info">Inactive</ElTag>
        </template>
      </ElTableColumn>

      <ElTableColumn label="Actions" width="220" fixed="right">
        <template #default="{ row }">
          <ElButton
            :type="row.enabled ? 'warning' : 'success'"
            size="small"
            @click="toggleChannel(row)"
          >
            {{ row.enabled ? 'Disable' : 'Enable' }}
          </ElButton>
          <ElButton type="danger" size="small" @click="deleteChannel(row)">
            <Icon icon="ep:delete" />
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
  </ContentWrap>
</template>
