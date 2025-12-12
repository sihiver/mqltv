<script setup lang="ts">
import { ContentWrap } from '@/components/ContentWrap'
import {
  ElForm,
  ElFormItem,
  ElSelect,
  ElOption,
  ElTransfer,
  ElButton,
  ElMessage,
  ElRow,
  ElCol
} from 'element-plus'
import { ref, onMounted, computed } from 'vue'
import request from '@/axios'

const users = ref<any[]>([])
const channels = ref<any[]>([])
const categories = ref<string[]>([])
const selectedUser = ref('')
const selectedChannels = ref<number[]>([])
const selectedCategory = ref('')
const loading = ref(false)

const transferData = computed(() => {
  return channels.value.map((ch: any) => ({
    key: ch.id,
    label: ch.name,
    disabled: !ch.enabled
  }))
})

const loadUsers = async () => {
  try {
    const res = await request.get({ url: '/api/users' })
    if (res && res.data) {
      users.value = res.data
    }
  } catch (error) {
    console.error('Error loading users:', error)
  }
}

const loadChannels = async () => {
  try {
    const res = await request.get({ url: '/api/channels/search', params: { q: '' } })
    if (res && res.data) {
      channels.value = res.data
      
      // Extract categories
      const cats = new Set<string>()
      channels.value.forEach((ch: any) => {
        if (ch.category) cats.add(ch.category)
      })
      categories.value = Array.from(cats).sort()
    }
  } catch (error) {
    console.error('Error loading channels:', error)
  }
}

const selectByCategory = () => {
  if (!selectedCategory.value) {
    ElMessage.warning('Please select a category first')
    return
  }

  const categoryChannels = channels.value
    .filter((ch: any) => ch.category === selectedCategory.value && ch.enabled)
    .map((ch: any) => ch.id)

  // Add to existing selection (not replace)
  const newSelection = new Set([...selectedChannels.value, ...categoryChannels])
  selectedChannels.value = Array.from(newSelection)

  ElMessage.success(`Added ${categoryChannels.length} channels from ${selectedCategory.value}`)
}

const clearSelection = () => {
  selectedChannels.value = []
  ElMessage.info('Selection cleared')
}

const generatePlaylist = async () => {
  if (!selectedUser.value) {
    ElMessage.warning('Please select a user')
    return
  }

  if (selectedChannels.value.length === 0) {
    ElMessage.warning('Please select at least one channel')
    return
  }

  loading.value = true
  try {
    const res = await request.post({
      url: '/api/generate-playlist',
      data: {
        user_id: selectedUser.value,
        channel_ids: selectedChannels.value
      }
    })

    if (res && res.data) {
      const playlistUrl = res.data.url || res.data.playlist_url
      ElMessage.success('Playlist generated successfully!')

      // Copy to clipboard
      if (playlistUrl) {
        navigator.clipboard.writeText(playlistUrl)
        ElMessage.info('Playlist URL copied to clipboard')
      }
    }
  } catch (error) {
    ElMessage.error('Failed to generate playlist')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadUsers()
  loadChannels()
})
</script>

<template>
  <ContentWrap title="Generate Playlist" message="Create custom user playlists">
    <div style="max-width: 800px">
      <ElForm label-width="120px">
        <ElFormItem label="Select User">
          <ElSelect v-model="selectedUser" placeholder="Choose user" style="width: 100%">
            <ElOption
              v-for="user in users"
              :key="user.id"
              :label="user.username"
              :value="user.id"
            />
          </ElSelect>
        </ElFormItem>

        <ElFormItem label="Select Channels">
          <div style="margin-bottom: 12px">
            <ElRow :gutter="12">
              <ElCol :span="14">
                <ElSelect
                  v-model="selectedCategory"
                  placeholder="Select category to add all channels"
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
              <ElCol :span="10">
                <ElButton type="primary" @click="selectByCategory" style="width: 100%">
                  <Icon icon="ep:plus" />
                  Add Category
                </ElButton>
              </ElCol>
            </ElRow>
            <div style="margin-top: 8px">
              <ElButton size="small" @click="clearSelection">
                <Icon icon="ep:delete" />
                Clear All
              </ElButton>
              <span style="margin-left: 12px; font-size: 13px; color: #909399">
                {{ selectedChannels.length }} channels selected
              </span>
            </div>
          </div>
          
          <ElTransfer
            v-model="selectedChannels"
            :data="transferData"
            filterable
            :titles="['Available Channels', 'Selected Channels']"
            :button-texts="['Remove', 'Add']"
            :format="{
              noChecked: '${total}',
              hasChecked: '${checked}/${total}'
            }"
            style="width: 100%"
          />
        </ElFormItem>

        <ElFormItem>
          <ElButton type="primary" @click="generatePlaylist" :loading="loading">
            <Icon icon="ep:document-add" />
            Generate Playlist
          </ElButton>
        </ElFormItem>
      </ElForm>

      <div style="margin-top: 24px; padding: 16px; background: #f5f7fa; border-radius: 4px">
        <h4 style="margin: 0 0 8px 0; font-size: 14px">Generated Playlist:</h4>
        <p style="margin: 0; font-size: 13px; color: #606266">
          After generation, the playlist URL will be automatically copied to your clipboard. You can
          share this URL with the selected user to access their custom channels.
        </p>
      </div>
    </div>
  </ContentWrap>
</template>
