<script setup lang="ts">
import { ContentWrap } from '@/components/ContentWrap'
import { ElForm, ElFormItem, ElInput, ElButton, ElMessage } from 'element-plus'
import { ref, reactive } from 'vue'
import request from '@/axios'
const loading = ref(false)

const importForm = reactive({
  name: '',
  url: ''
})

const importPlaylist = async () => {
  if (!importForm.name || !importForm.url) {
    ElMessage.warning('Name and URL are required')
    return
  }

  loading.value = true
  try {
    const res = await request.post({
      url: '/api/playlists/import',
      data: importForm
    })

    ElMessage.success(`Playlist imported successfully! ${res.data?.channels || 0} channels added`)
    importForm.name = ''
    importForm.url = ''
  } catch (error) {
    ElMessage.error('Failed to import playlist')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <ContentWrap title="Import M3U Playlist" message="Import channels from M3U URL">
    <div style="max-width: 600px">
      <ElForm :model="importForm" label-width="120px">
        <ElFormItem label="Playlist Name">
          <ElInput v-model="importForm.name" placeholder="e.g., My Channels" />
        </ElFormItem>

        <ElFormItem label="M3U URL">
          <ElInput
            v-model="importForm.url"
            type="textarea"
            :rows="4"
            placeholder="https://example.com/playlist.m3u"
          />
        </ElFormItem>

        <ElFormItem>
          <ElButton type="primary" @click="importPlaylist" :loading="loading">
            <Icon icon="ep:upload" />
            Import Playlist
          </ElButton>
        </ElFormItem>
      </ElForm>

      <div style="margin-top: 24px; padding: 16px; background: #f5f7fa; border-radius: 4px">
        <h4 style="margin: 0 0 8px 0; font-size: 14px">Supported Formats:</h4>
        <ul style="margin: 0; padding-left: 20px; font-size: 13px; color: #606266">
          <li>Standard M3U/M3U8 playlists</li>
          <li>EXTINF format with channel names and logos</li>
          <li>Group titles for categories</li>
          <li>Both HTTP and HTTPS URLs</li>
        </ul>
      </div>
    </div>
  </ContentWrap>
</template>
