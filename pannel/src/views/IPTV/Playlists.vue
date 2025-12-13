<script setup lang="ts">
import { ContentWrap } from '@/components/ContentWrap'
import {
  ElTable,
  ElTableColumn,
  ElButton,
  ElMessageBox,
  ElMessage,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput
} from 'element-plus'
import { ref, onMounted } from 'vue'
import request from '@/axios'

const playlists = ref([])
const loading = ref(false)
const editDialogVisible = ref(false)
const editForm = ref({
  id: 0,
  name: '',
  url: ''
})

const loadPlaylists = async () => {
  loading.value = true
  try {
    const res = await request.get({ url: '/api/playlists' })
    if (res && res.data) {
      playlists.value = res.data
    }
  } catch (error) {
    console.error('Error loading playlists:', error)
    ElMessage.error('Failed to load playlists')
  } finally {
    loading.value = false
  }
}

const openEditDialog = (playlist: any) => {
  editForm.value = {
    id: playlist.id,
    name: playlist.name,
    url: playlist.url
  }
  editDialogVisible.value = true
}

const updatePlaylist = async () => {
  try {
    await request.put({
      url: `/api/playlists/${editForm.value.id}`,
      data: {
        name: editForm.value.name,
        url: editForm.value.url
      }
    })
    ElMessage.success('Playlist updated successfully')
    editDialogVisible.value = false
    loadPlaylists()
  } catch (error) {
    ElMessage.error('Failed to update playlist')
  }
}

const refreshPlaylist = async (id: number, name: string) => {
  try {
    await ElMessageBox.confirm(
      `Re-import playlist "${name}" from its URL? This will update all channels.`,
      'Confirm Refresh',
      {
        confirmButtonText: 'Refresh',
        cancelButtonText: 'Cancel',
        type: 'info'
      }
    )

    loading.value = true
    const res = await request.post({ url: `/api/playlists/${id}/refresh` })
    ElMessage.success(`Playlist refreshed! ${res.data?.channels_count || 0} channels updated`)
    loadPlaylists()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('Failed to refresh playlist')
    }
  } finally {
    loading.value = false
  }
}

const deletePlaylist = async (id: number, name: string) => {
  try {
    await ElMessageBox.confirm(`Delete playlist "${name}" and all its channels?`, 'Confirm', {
      confirmButtonText: 'Delete',
      cancelButtonText: 'Cancel',
      type: 'warning'
    })

    loading.value = true
    await request.delete({ url: `/api/playlists/${id}` })

    // Remove from local array immediately
    playlists.value = playlists.value.filter((p: any) => p.id !== id)

    ElMessage.success('Playlist deleted successfully')
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('Failed to delete playlist')
    }
  } finally {
    loading.value = false
  }
}

const formatDate = (dateString: string) => {
  if (!dateString) return '-'
  const date = new Date(dateString)
  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

onMounted(() => {
  loadPlaylists()
})
</script>

<template>
  <ContentWrap title="Playlists" message="Manage your M3U playlists">
    <ElTable :data="playlists" v-loading="loading" style="width: 100%">
      <ElTableColumn prop="name" label="Playlist Name" min-width="200">
        <template #default="{ row }">
          <div style="display: flex; align-items: center; gap: 8px">
            <Icon icon="ep:folder" />
            <strong>{{ row.name }}</strong>
          </div>
        </template>
      </ElTableColumn>

      <ElTableColumn prop="url" label="URL" min-width="300" show-overflow-tooltip />

      <ElTableColumn prop="channel_count" label="Channels" width="120" align="center">
        <template #default="{ row }">
          <ElTag type="primary">{{ row.channel_count || 0 }}</ElTag>
        </template>
      </ElTableColumn>

      <ElTableColumn prop="created_at" label="Created" width="180">
        <template #default="{ row }">
          {{ formatDate(row.created_at) }}
        </template>
      </ElTableColumn>

      <ElTableColumn label="Actions" width="280" fixed="right" align="center">
        <template #default="{ row }">
          <div style="display: flex; gap: 8px; justify-content: center">
            <ElButton type="primary" size="small" @click="openEditDialog(row)">
              <Icon icon="ep:edit" />
              Edit
            </ElButton>
            <ElButton type="success" size="small" @click="refreshPlaylist(row.id, row.name)">
              <Icon icon="ep:refresh" />
              Refresh
            </ElButton>
            <ElButton type="danger" size="small" @click="deletePlaylist(row.id, row.name)">
              <Icon icon="ep:delete" />
              Delete
            </ElButton>
          </div>
        </template>
      </ElTableColumn>
    </ElTable>

    <!-- Edit Dialog -->
    <ElDialog v-model="editDialogVisible" title="Edit Playlist" width="500px">
      <ElForm :model="editForm" label-width="100px">
        <ElFormItem label="Name">
          <ElInput v-model="editForm.name" placeholder="Playlist name" />
        </ElFormItem>
        <ElFormItem label="URL">
          <ElInput v-model="editForm.url" placeholder="M3U URL" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="editDialogVisible = false">Cancel</ElButton>
        <ElButton type="primary" @click="updatePlaylist">Update</ElButton>
      </template>
    </ElDialog>
  </ContentWrap>
</template>
