<script setup lang="ts">
import { ContentWrap } from '@/components/ContentWrap'
import {
  ElTable,
  ElTableColumn,
  ElButton,
  ElTag,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElMessageBox
} from 'element-plus'
import { ref, reactive, onMounted } from 'vue'
import request from '@/axios'

const relays = ref([])
const loading = ref(false)
const showCreateDialog = ref(false)

const relayForm = reactive({
  name: '',
  source_url: ''
})

const loadRelays = async () => {
  loading.value = true
  try {
    const res = await request.get({ url: '/api/relays' })
    if (res && res.data) {
      relays.value = res.data
    }
  } catch (error) {
    console.error('Error loading relays:', error)
    ElMessage.error('Failed to load relays')
  } finally {
    loading.value = false
  }
}

const createRelay = async () => {
  if (!relayForm.name || !relayForm.source_url) {
    ElMessage.warning('Name and Source URL are required')
    return
  }

  try {
    await request.post({ url: '/api/relays', data: relayForm })
    ElMessage.success('Relay created successfully')
    showCreateDialog.value = false
    relayForm.name = ''
    relayForm.source_url = ''
    loadRelays()
  } catch (error) {
    ElMessage.error('Failed to create relay')
  }
}

const deleteRelay = async (relay: any) => {
  try {
    await ElMessageBox.confirm(`Delete relay "${relay.name}"?`, 'Confirm', {
      confirmButtonText: 'Delete',
      cancelButtonText: 'Cancel',
      type: 'warning'
    })

    await request.delete({ url: `/api/relays/${relay.id}` })
    ElMessage.success('Relay deleted successfully')
    loadRelays()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('Failed to delete relay')
    }
  }
}

const copyRelayURL = (relay: any) => {
  const url = `${window.location.origin}/relay/${relay.id}/playlist.m3u8`
  navigator.clipboard.writeText(url)
  ElMessage.success('URL copied to clipboard')
}

onMounted(() => {
  loadRelays()
})
</script>

<template>
  <ContentWrap title="Relays" message="Manage stream relays">
    <ElButton type="primary" @click="showCreateDialog = true" style="margin-bottom: 16px">
      <Icon icon="ep:plus" />
      Create New Relay
    </ElButton>

    <ElTable :data="relays" v-loading="loading" style="width: 100%">
      <ElTableColumn prop="name" label="Name" min-width="150" />

      <ElTableColumn prop="source_url" label="Source URL" min-width="300" show-overflow-tooltip />

      <ElTableColumn label="Status" width="120">
        <template #default="{ row }">
          <ElTag v-if="row.active" type="success"> <Icon icon="ep:video-play" /> Active </ElTag>
          <ElTag v-else type="info"> <Icon icon="ep:video-pause" /> Inactive </ElTag>
        </template>
      </ElTableColumn>

      <ElTableColumn label="Actions" width="200" fixed="right">
        <template #default="{ row }">
          <ElButton type="primary" size="small" text @click="copyRelayURL(row)">
            <Icon icon="ep:document-copy" />
            Copy URL
          </ElButton>
          <ElButton type="danger" size="small" @click="deleteRelay(row)">
            <Icon icon="ep:delete" />
          </ElButton>
        </template>
      </ElTableColumn>
    </ElTable>

    <!-- Create Dialog -->
    <ElDialog v-model="showCreateDialog" title="Create New Relay" width="500px">
      <ElForm :model="relayForm" label-width="120px">
        <ElFormItem label="Name">
          <ElInput v-model="relayForm.name" placeholder="Relay name" />
        </ElFormItem>
        <ElFormItem label="Source URL">
          <ElInput
            v-model="relayForm.source_url"
            type="textarea"
            :rows="3"
            placeholder="http://example.com/stream.m3u8"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="showCreateDialog = false">Cancel</ElButton>
        <ElButton type="primary" @click="createRelay">Create</ElButton>
      </template>
    </ElDialog>
  </ContentWrap>
</template>
