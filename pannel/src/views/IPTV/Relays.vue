<script setup lang="ts">
import { ContentWrap } from '@/components/ContentWrap'
import { Icon } from '@iconify/vue'
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

type RelayRow = {
  id: number
  name: string
  source_urls: string
  output_path: string
  active: boolean
}

const relays = ref<RelayRow[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)

const relayForm = reactive({
  name: '',
  output_path: '',
  source_urls_text: ''
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
  const urls = relayForm.source_urls_text
    .split('\n')
    .map((v) => v.trim())
    .filter(Boolean)

  if (!relayForm.output_path || urls.length === 0) {
    ElMessage.warning('Output Path and Source URLs are required')
    return
  }

  try {
    await request.post({
      url: '/api/relays',
      data: {
        name: relayForm.name,
        output_path: relayForm.output_path,
        source_urls: urls
      }
    })
    ElMessage.success('Relay created successfully')
    showCreateDialog.value = false
    relayForm.name = ''
    relayForm.output_path = ''
    relayForm.source_urls_text = ''
    loadRelays()
  } catch (error) {
    ElMessage.error('Failed to create relay')
  }
}

const deleteRelay = async (relay: RelayRow) => {
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

const copyRelayURL = (relay: RelayRow) => {
  const url = `${window.location.origin}/stream/${relay.output_path}`
  navigator.clipboard.writeText(url)
  ElMessage.success('URL copied to clipboard')
}

const getSourcesCount = (relay: RelayRow) => {
  try {
    const arr = JSON.parse(relay.source_urls)
    return Array.isArray(arr) ? arr.length : 0
  } catch {
    return 0
  }
}

onMounted(() => {
  loadRelays()
})
</script>

<template>
  <ContentWrap title="Relays" message="Manage stream relays">
    <ElButton type="primary" @click="showCreateDialog = true" style="margin-bottom: 16px">
      <template #icon>
        <Icon icon="ep:plus" />
      </template>
      Create New Relay
    </ElButton>

    <ElTable :data="relays" v-loading="loading" style="width: 100%">
      <ElTableColumn prop="name" label="Name" min-width="150" />

      <ElTableColumn prop="output_path" label="Output Path" min-width="180" show-overflow-tooltip />

      <ElTableColumn label="Sources" width="120">
        <template #default="{ row }">
          <ElTag type="info">{{ getSourcesCount(row) }}</ElTag>
        </template>
      </ElTableColumn>

      <ElTableColumn label="Status" width="120">
        <template #default="{ row }">
          <ElTag v-if="row.active" type="success"> <Icon icon="ep:video-play" /> Active </ElTag>
          <ElTag v-else type="info"> <Icon icon="ep:video-pause" /> Inactive </ElTag>
        </template>
      </ElTableColumn>

      <ElTableColumn label="Actions" width="200" fixed="right">
        <template #default="{ row }">
          <ElButton type="primary" size="small" text @click="copyRelayURL(row)">
            <template #icon>
              <Icon icon="ep:document-copy" />
            </template>
            Copy URL
          </ElButton>
          <ElButton type="danger" size="small" @click="deleteRelay(row)">
            <template #icon>
              <Icon icon="ep:delete" />
            </template>
          </ElButton>
        </template>
      </ElTableColumn>
    </ElTable>

    <!-- Create Dialog -->
    <ElDialog v-model="showCreateDialog" title="Create New Relay" width="560px">
      <ElForm :model="relayForm" label-width="120px">
        <ElFormItem label="Name">
          <ElInput v-model="relayForm.name" placeholder="Relay name" />
        </ElFormItem>
        <ElFormItem label="Output Path">
          <ElInput v-model="relayForm.output_path" placeholder="channel-123" />
        </ElFormItem>
        <ElFormItem label="Source URLs">
          <ElInput
            v-model="relayForm.source_urls_text"
            type="textarea"
            :rows="3"
            placeholder="http://example.com/stream1.ts\nhttp://example.com/stream2.ts"
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
