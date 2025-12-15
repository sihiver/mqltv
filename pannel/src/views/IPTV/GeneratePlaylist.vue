<script setup lang="ts">
import { ContentWrap } from '@/components/ContentWrap'
import { Icon } from '@/components/Icon'
import {
  ElForm,
  ElFormItem,
  ElSelect,
  ElOption,
  ElTransfer,
  ElButton,
  ElMessage,
  ElRow,
  ElCol,
  ElCard
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
    <ElRow :gutter="20">
      <ElCol :xs="24" :md="12">
        <ElCard shadow="hover">
          <template #header>
            <div style="display: flex; align-items: center; gap: 8px">
              <Icon icon="ep:user" />
              <span>Select User</span>
            </div>
          </template>

          <ElForm label-width="120px">
            <ElFormItem label="User">
              <ElSelect v-model="selectedUser" placeholder="Choose user" style="width: 100%">
                <ElOption
                  v-for="user in users"
                  :key="user.id"
                  :label="user.username"
                  :value="user.id"
                />
              </ElSelect>
            </ElFormItem>
          </ElForm>
        </ElCard>
      </ElCol>

      <ElCol :xs="24" :md="12">
        <ElCard shadow="hover">
          <template #header>
            <div style="display: flex; align-items: center; gap: 8px">
              <Icon icon="ep:document-add" />
              <span>Generate</span>
            </div>
          </template>

          <ElButton type="primary" @click="generatePlaylist" :loading="loading" style="width: 100%">
            <Icon icon="ep:document-add" />
            Generate Playlist
          </ElButton>

          <div style="margin-top: 12px; font-size: 13px; color: #909399">
            After generation, the playlist URL will be copied to clipboard.
          </div>
        </ElCard>
      </ElCol>

      <ElCol :xs="24" :md="24" style="margin-top: 20px">
        <ElCard shadow="hover">
          <template #header>
            <div style="display: flex; align-items: center; justify-content: space-between">
              <div style="display: flex; align-items: center; gap: 8px">
                <Icon icon="ep:video-camera" />
                <span>Select Channels</span>
              </div>
              <div style="font-size: 13px; color: #909399">
                {{ selectedChannels.length }} selected
              </div>
            </div>
          </template>

          <div style="margin-bottom: 12px">
            <ElRow :gutter="12">
              <ElCol :xs="24" :sm="16">
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
              <ElCol :xs="24" :sm="8">
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
        </ElCard>
      </ElCol>
    </ElRow>
  </ContentWrap>
</template>
