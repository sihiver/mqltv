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
  ElSelect,
  ElOption,
  ElSwitch,
  ElMessage,
  ElMessageBox
} from 'element-plus'
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import request from '@/axios'

type RelayRow = {
  id: number
  name: string
  source_urls: string
  output_path: string
  active: boolean
}

const relays = ref<RelayRow[]>([])
const channels = ref<any[]>([])
const streamMap = ref<Record<string, any>>({})
const loading = ref(false)
const showDialog = ref(false)
const isEditing = ref(false)
const selectedChannelId = ref<number | undefined>(undefined)

const relayForm = reactive({
  id: 0,
  name: '',
  output_path: '',
  active: true,
  sources: [''] as string[]
})

const parseSources = (relay: RelayRow): string[] => {
  try {
    const arr = JSON.parse(relay.source_urls)
    return Array.isArray(arr) ? arr.filter(Boolean) : []
  } catch {
    return relay.source_urls ? [relay.source_urls] : []
  }
}

const loadRelays = async () => {
  loading.value = true
  try {
    const res = await request.get({ url: '/api/relays' })
    if (res && res.data) {
      relays.value = res.data || []
    }
  } catch (error) {
    console.error('Error loading relays:', error)
    ElMessage.error('Failed to load relays')
  } finally {
    loading.value = false
  }
}

const loadChannels = async () => {
  try {
    const res = await request.get({ url: '/api/channels/search', params: { q: '' } })
    if (res && Array.isArray(res.data)) {
      channels.value = res.data
    }
  } catch (error) {
    console.error('Error loading channels:', error)
  }
}

const loadStreamStatus = async () => {
  try {
    const res = await request.get({ url: '/api/streams/status' })
    if (res && res.data && Array.isArray(res.data.streams)) {
      const map: Record<string, any> = {}
      res.data.streams.forEach((s: any) => {
        if (s && s.id) {
          map[s.id] = s
        }
      })
      streamMap.value = map
    }
  } catch (error) {
    // ignore polling errors
  }
}

const getRelayLiveStatus = (relay: RelayRow) => {
  const path = relay.output_path || ''
  if (streamMap.value[path]) return streamMap.value[path]
  if (streamMap.value[path + '_hls']) return streamMap.value[path + '_hls']
  if (path.startsWith('channel-')) {
    const idPart = path.replace('channel-', '')
    if (streamMap.value['channel_' + idPart]) return streamMap.value['channel_' + idPart]
    if (streamMap.value['channel_' + idPart + '_hls'])
      return streamMap.value['channel_' + idPart + '_hls']
  }
  return null
}

const openCreateDialog = () => {
  isEditing.value = false
  selectedChannelId.value = undefined
  relayForm.id = 0
  relayForm.name = ''
  relayForm.output_path = ''
  relayForm.active = true
  relayForm.sources = ['', '']
  showDialog.value = true
}

const openEditDialog = (relay: RelayRow) => {
  isEditing.value = true
  selectedChannelId.value = undefined
  relayForm.id = relay.id
  relayForm.name = relay.name
  relayForm.output_path = relay.output_path
  relayForm.active = Boolean(relay.active)
  const parsed = parseSources(relay)
  relayForm.sources = parsed.length > 0 ? [...parsed] : ['']
  showDialog.value = true
}

const onSelectChannel = (chId: number) => {
  const ch = channels.value.find((c: any) => c.id === chId)
  if (!ch) return
  relayForm.name = ch.name
  relayForm.output_path = `channel-${ch.id}`
  if (!relayForm.sources[0]) {
    relayForm.sources[0] = ch.url
  } else if (!relayForm.sources.includes(ch.url)) {
    relayForm.sources[0] = ch.url
  }
}

const addSourceRow = () => {
  relayForm.sources.push('')
}

const removeSourceRow = (idx: number) => {
  if (relayForm.sources.length <= 1) return
  relayForm.sources.splice(idx, 1)
}

const moveSourceUp = (idx: number) => {
  if (idx <= 0) return
  const tmp = relayForm.sources[idx - 1]
  relayForm.sources[idx - 1] = relayForm.sources[idx]
  relayForm.sources[idx] = tmp
}

const moveSourceDown = (idx: number) => {
  if (idx >= relayForm.sources.length - 1) return
  const tmp = relayForm.sources[idx + 1]
  relayForm.sources[idx + 1] = relayForm.sources[idx]
  relayForm.sources[idx] = tmp
}

const saveRelay = async () => {
  const urls = relayForm.sources.map((v) => v.trim()).filter(Boolean)

  if (!relayForm.output_path.trim() || urls.length === 0) {
    ElMessage.warning('Output Path dan minimal 1 Source URL wajib diisi')
    return
  }

  try {
    if (isEditing.value) {
      await request.put({
        url: `/api/relays/${relayForm.id}`,
        data: {
          name: relayForm.name.trim() || relayForm.output_path.trim(),
          output_path: relayForm.output_path.trim(),
          source_urls: urls,
          active: relayForm.active
        }
      })
      ElMessage.success('Multi-Source Relay berhasil diperbarui')
    } else {
      await request.post({
        url: '/api/relays',
        data: {
          name: relayForm.name.trim() || relayForm.output_path.trim(),
          output_path: relayForm.output_path.trim(),
          source_urls: urls
        }
      })
      ElMessage.success('Multi-Source Relay berhasil dibuat')
    }
    showDialog.value = false
    loadRelays()
    loadStreamStatus()
  } catch (error) {
    ElMessage.error(isEditing.value ? 'Gagal memperbarui relay' : 'Gagal membuat relay')
  }
}

const switchActiveSource = async (relay: RelayRow, idx: number) => {
  try {
    await request.post({
      url: `/api/relays/${relay.id}/switch-source`,
      data: { source_index: idx }
    })
    ElMessage.success(
      `Berhasil pindah ke ${idx === 0 ? 'Source Utama' : 'Backup #' + idx}`
    )
    setTimeout(() => {
      loadStreamStatus()
    }, 800)
  } catch (error) {
    ElMessage.error('Gagal mengganti source relay')
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

let statusInterval: any = null

onMounted(() => {
  loadRelays()
  loadChannels()
  loadStreamStatus()
  statusInterval = setInterval(() => {
    loadStreamStatus()
  }, 5000)
})

onUnmounted(() => {
  if (statusInterval) {
    clearInterval(statusInterval)
  }
})
</script>

<template>
  <ContentWrap
    title="Multi-Source Relays & Failover"
    message="Kelola stream utama (misal Receiver Ku-Band) dan URL cadangan (Backup Failover otomatis saat gangguan/hujan)"
  >
    <div
      style="
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex-wrap: wrap;
        gap: 12px;
        margin-bottom: 16px;
      "
    >
      <ElButton type="primary" @click="openCreateDialog">
        <template #icon>
          <Icon icon="ep:plus" />
        </template>
        Create Multi-Source Relay
      </ElButton>

      <ElTag type="info" size="default">
        Tip: Source #1 = Utama (Prioritas 1), Source #2 dst = Backup Failover Otomatis
      </ElTag>
    </div>

    <ElTable :data="relays" v-loading="loading" style="width: 100%">
      <ElTableColumn prop="name" label="Name" min-width="150" />

      <ElTableColumn prop="output_path" label="Output Path" min-width="140" show-overflow-tooltip />

      <ElTableColumn label="Multi-Source URLs (Utama & Backup Failover)" min-width="400">
        <template #default="{ row }">
          <div style="display: flex; flex-direction: column; gap: 6px; padding: 4px 0">
            <div
              v-for="(srcUrl, idx) in parseSources(row)"
              :key="idx"
              style="
                display: flex;
                align-items: center;
                gap: 8px;
                background: #f8f9fa;
                padding: 4px 8px;
                border-radius: 6px;
              "
            >
              <ElTag :type="idx === 0 ? 'primary' : 'warning'" size="small" style="flex-shrink: 0">
                {{ idx === 0 ? 'Utama' : 'Backup #' + idx }}
              </ElTag>

              <ElTag
                v-if="
                  getRelayLiveStatus(row)?.active &&
                  (getRelayLiveStatus(row)?.active_source_index ?? 0) === idx
                "
                type="success"
                size="small"
                effect="dark"
                style="flex-shrink: 0"
              >
                LIVE
              </ElTag>

              <span
                style="
                  font-size: 12px;
                  color: #606266;
                  overflow: hidden;
                  text-overflow: ellipsis;
                  white-space: nowrap;
                  flex: 1;
                "
                :title="srcUrl"
              >
                {{ srcUrl }}
              </span>

              <ElButton
                v-if="
                  parseSources(row).length > 1 &&
                  (!getRelayLiveStatus(row)?.active ||
                    (getRelayLiveStatus(row)?.active_source_index ?? 0) !== idx)
                "
                size="small"
                type="primary"
                plain
                style="padding: 2px 8px; height: 22px; font-size: 11px"
                @click="switchActiveSource(row, idx)"
              >
                Pakai
              </ElButton>
            </div>
          </div>
        </template>
      </ElTableColumn>

      <ElTableColumn label="Status" width="170">
        <template #default="{ row }">
          <div style="display: flex; flex-direction: column; gap: 4px">
            <ElTag v-if="row.active" type="success" size="small">
              <Icon icon="ep:video-play" /> Enabled
            </ElTag>
            <ElTag v-else type="info" size="small">
              <Icon icon="ep:video-pause" /> Disabled
            </ElTag>

            <ElTag
              v-if="getRelayLiveStatus(row)?.active"
              :type="(getRelayLiveStatus(row)?.active_source_index ?? 0) > 0 ? 'warning' : 'success'"
              size="small"
            >
              {{
                (getRelayLiveStatus(row)?.active_source_index ?? 0) > 0
                  ? 'Failover #' + getRelayLiveStatus(row)?.active_source_index
                  : 'Streaming'
              }}
              ({{ getRelayLiveStatus(row)?.clients || 0 }} view)
            </ElTag>
          </div>
        </template>
      </ElTableColumn>

      <ElTableColumn label="Actions" width="250" fixed="right">
        <template #default="{ row }">
          <ElButton type="primary" size="small" @click="openEditDialog(row)">
            <template #icon>
              <Icon icon="ep:edit" />
            </template>
            Sources ({{ parseSources(row).length }})
          </ElButton>
          <ElButton type="primary" size="small" text @click="copyRelayURL(row)">
            <template #icon>
              <Icon icon="ep:document-copy" />
            </template>
            Copy
          </ElButton>
          <ElButton type="danger" size="small" @click="deleteRelay(row)">
            <template #icon>
              <Icon icon="ep:delete" />
            </template>
          </ElButton>
        </template>
      </ElTableColumn>
    </ElTable>

    <!-- Create / Edit Multi-Source Dialog -->
    <ElDialog
      v-model="showDialog"
      :title="isEditing ? 'Edit Multi-Source Relay' : 'Create Multi-Source Relay'"
      width="680px"
    >
      <ElForm :model="relayForm" label-width="130px">
        <ElFormItem v-if="!isEditing && channels.length > 0" label="Pilih Channel">
          <ElSelect
            v-model="selectedChannelId"
            placeholder="(Opsional) Pilih dari Channel yang sudah ada..."
            filterable
            clearable
            style="width: 100%"
            @change="onSelectChannel"
          >
            <ElOption
              v-for="ch in channels"
              :key="ch.id"
              :label="`${ch.name} (channel-${ch.id})`"
              :value="ch.id"
            />
          </ElSelect>
        </ElFormItem>

        <ElFormItem label="Name" required>
          <ElInput v-model="relayForm.name" placeholder="Contoh: RCTI HD" />
        </ElFormItem>

        <ElFormItem label="Output Path" required>
          <ElInput v-model="relayForm.output_path" placeholder="Contoh: channel-12" />
        </ElFormItem>

        <ElFormItem v-if="isEditing" label="Status Relay">
          <ElSwitch v-model="relayForm.active" active-text="Active" inactive-text="Inactive" />
        </ElFormItem>

        <ElFormItem label="Multi-Source URLs" required>
          <div style="width: 100%; display: flex; flex-direction: column; gap: 10px">
            <div
              v-for="(_, idx) in relayForm.sources"
              :key="idx"
              style="display: flex; align-items: center; gap: 8px; width: 100%"
            >
              <ElTag
                :type="idx === 0 ? 'primary' : 'warning'"
                style="width: 90px; text-align: center; flex-shrink: 0"
              >
                {{ idx === 0 ? 'Utama #1' : 'Backup #' + idx }}
              </ElTag>

              <ElInput
                v-model="relayForm.sources[idx]"
                :placeholder="
                  idx === 0
                    ? 'URL Utama (contoh: http://ip-receiver-kuband:8000/stream)'
                    : 'URL Cadangan / Backup (contoh: http://backup-cband-atau-hls/index.m3u8)'
                "
                style="flex: 1"
              />

              <ElButton
                size="small"
                :disabled="idx === 0"
                title="Naikkan prioritas"
                @click="moveSourceUp(idx)"
              >
                ↑
              </ElButton>
              <ElButton
                size="small"
                :disabled="idx === relayForm.sources.length - 1"
                title="Turunkan prioritas"
                @click="moveSourceDown(idx)"
              >
                ↓
              </ElButton>
              <ElButton
                type="danger"
                size="small"
                plain
                :disabled="relayForm.sources.length <= 1"
                title="Hapus source ini"
                @click="removeSourceRow(idx)"
              >
                <Icon icon="ep:delete" />
              </ElButton>
            </div>

            <div style="display: flex; align-items: center; justify-content: space-between; margin-top: 4px">
              <ElButton type="success" plain size="small" @click="addSourceRow">
                <template #icon>
                  <Icon icon="ep:plus" />
                </template>
                Tambah Source Backup (Failover)
              </ElButton>
              <span style="font-size: 12px; color: #909399">
                Otomatis pindah ke Backup jika Source Utama gangguan/hujan
              </span>
            </div>
          </div>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="showDialog = false">Cancel</ElButton>
        <ElButton type="primary" @click="saveRelay">
          {{ isEditing ? 'Save Changes' : 'Create Relay' }}
        </ElButton>
      </template>
    </ElDialog>
  </ContentWrap>
</template>
