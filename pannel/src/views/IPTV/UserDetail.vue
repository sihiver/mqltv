<script setup lang="ts">
import { ContentWrap } from '@/components/ContentWrap'
import {
  ElButton,
  ElTag,
  ElTable,
  ElTableColumn,
  ElCard,
  ElDescriptions,
  ElDescriptionsItem,
  ElMessage,
  ElMessageBox,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput
} from 'element-plus'
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import request from '@/axios'

const route = useRoute()
const router = useRouter()
const userId = route.params.id

const userDetail = ref<any>(null)
const loading = ref(false)
const channels = ref([])
const showExtendDialog = ref(false)
const extendForm = ref({ days: 30 })

const loadUserDetail = async () => {
  loading.value = true
  try {
    const res = await request.get({ url: `/api/users/${userId}` })
    if (res && res.data) {
      userDetail.value = res.data
      // Use channels from API response (only user's channels)
      if (res.data.channels) {
        channels.value = res.data.channels
      }
    }
  } catch (error) {
    console.error('Error loading user detail:', error)
    ElMessage.error('Failed to load user details')
  } finally {
    loading.value = false
  }
}

const copyToClipboard = (text: string) => {
  navigator.clipboard
    .writeText(text)
    .then(() => {
      ElMessage.success('Copied to clipboard!')
    })
    .catch(() => {
      ElMessage.error('Failed to copy')
    })
}

const getFullUrl = (path: string) => {
  return `${window.location.protocol}//${window.location.host}${path}`
}

const getStatusType = (expiry: string) => {
  if (!expiry) return 'info'
  const expiryDate = new Date(expiry)
  const now = new Date()

  if (expiryDate < now) return 'danger'

  const daysLeft = Math.floor((expiryDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
  if (daysLeft <= 7) return 'warning'

  return 'success'
}

const formatExpiry = (expiry: string) => {
  if (!expiry) return 'No expiry'
  const date = new Date(expiry)
  const now = new Date()

  if (date < now) return 'Expired'

  return date.toLocaleDateString()
}

const extendSubscription = async () => {
  try {
    await request.post({
      url: `/api/users/${userId}/extend`,
      data: { days: extendForm.value.days }
    })
    ElMessage.success('Subscription extended successfully')
    showExtendDialog.value = false
    loadUserDetail()
  } catch (error) {
    ElMessage.error('Failed to extend subscription')
  }
}

const resetPassword = async () => {
  try {
    const result = await ElMessageBox.prompt(
      `Enter new password for user "${userDetail.value?.user.username}"`,
      'Reset Password',
      {
        confirmButtonText: 'Reset',
        cancelButtonText: 'Cancel',
        inputPattern: /.+/,
        inputErrorMessage: 'Password is required'
      }
    )

    await request.post({
      url: `/api/users/${userId}/reset-password`,
      data: { password: result.value }
    })
    ElMessage.success('Password reset successfully')
    loadUserDetail()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('Failed to reset password')
    }
  }
}

onMounted(() => {
  loadUserDetail()
})
</script>

<template>
  <ContentWrap title="User Details" message="View user information, playlist and channels">
    <div style="margin-bottom: 16px">
      <ElButton @click="router.push('/users/index')">
        <Icon icon="ep:back" />
        Back to Users
      </ElButton>
      <ElButton type="primary" @click="showExtendDialog = true" style="margin-left: 8px">
        <Icon icon="ep:calendar" />
        Extend Subscription
      </ElButton>
      <ElButton type="warning" @click="resetPassword" style="margin-left: 8px">
        <Icon icon="ep:lock" />
        Reset Password
      </ElButton>
    </div>

    <div v-if="userDetail" v-loading="loading">
      <!-- User Information -->
      <ElCard style="margin-bottom: 16px">
        <template #header>
          <div style="display: flex; justify-content: space-between; align-items: center">
            <span><strong>User Information</strong></span>
            <ElTag :type="getStatusType(userDetail.user.expires_at)">
              {{ formatExpiry(userDetail.user.expires_at) }}
            </ElTag>
          </div>
        </template>
        <ElDescriptions :column="2" border>
          <ElDescriptionsItem label="Username">{{ userDetail.user.username }}</ElDescriptionsItem>
          <ElDescriptionsItem label="User ID">{{ userDetail.user.id }}</ElDescriptionsItem>
          <ElDescriptionsItem label="Password">{{ userDetail.user.password }}</ElDescriptionsItem>
          <ElDescriptionsItem label="Max Connections">
            {{ userDetail.user.max_connections }}
          </ElDescriptionsItem>
          <ElDescriptionsItem label="Expiry Date">
            {{
              userDetail.user.expires_at
                ? new Date(userDetail.user.expires_at).toLocaleString()
                : 'No expiry set'
            }}
          </ElDescriptionsItem>
          <ElDescriptionsItem label="Created At">
            {{ new Date(userDetail.user.created_at).toLocaleString() }}
          </ElDescriptionsItem>
          <ElDescriptionsItem label="Days Remaining" v-if="userDetail.user.expires_at">
            <ElTag :type="userDetail.user.days_remaining < 0 ? 'danger' : 'success'">
              {{ userDetail.user.days_remaining }} days
            </ElTag>
          </ElDescriptionsItem>
          <ElDescriptionsItem label="Status">
            <ElTag :type="userDetail.user.is_active ? 'success' : 'danger'">
              {{ userDetail.user.is_active ? 'Active' : 'Inactive' }}
            </ElTag>
          </ElDescriptionsItem>
        </ElDescriptions>
      </ElCard>

      <!-- Playlist Information -->
      <ElCard style="margin-bottom: 16px">
        <template #header>
          <span><strong>Playlist Information</strong></span>
        </template>
        <ElDescriptions :column="1" border>
          <ElDescriptionsItem label="Status">
            <ElTag :type="userDetail.playlist.generated ? 'success' : 'warning'">
              {{ userDetail.playlist.generated ? 'Generated' : 'Not Generated Yet' }}
            </ElTag>
          </ElDescriptionsItem>
          <ElDescriptionsItem label="Playlist URL" v-if="userDetail.playlist.generated">
            <div style="display: flex; align-items: center; gap: 8px">
              <code style="flex: 1; padding: 8px; background: #f5f5f5; border-radius: 4px">
                {{ getFullUrl(userDetail.playlist.url) }}
              </code>
              <ElButton
                type="primary"
                size="small"
                @click="copyToClipboard(getFullUrl(userDetail.playlist.url))"
              >
                <Icon icon="ep:document-copy" />
                Copy
              </ElButton>
            </div>
          </ElDescriptionsItem>
          <ElDescriptionsItem label="Filename" v-if="userDetail.playlist.generated">
            {{ userDetail.playlist.filename }}
          </ElDescriptionsItem>
          <ElDescriptionsItem label="File Size" v-if="userDetail.playlist.generated">
            {{ Math.round(userDetail.playlist.size / 1024) }} KB
          </ElDescriptionsItem>
          <ElDescriptionsItem label="Generated At" v-if="userDetail.playlist.generated">
            {{ new Date(userDetail.playlist.generated_at).toLocaleString() }}
          </ElDescriptionsItem>
          <ElDescriptionsItem label="Total Active Channels">
            <ElTag type="primary">{{ userDetail.total_channels }} channels</ElTag>
          </ElDescriptionsItem>
        </ElDescriptions>
      </ElCard>

      <!-- Available Playlists Summary -->
      <ElCard style="margin-bottom: 16px" v-if="userDetail.playlist.available_playlists">
        <template #header>
          <span><strong>Available Playlists</strong></span>
        </template>
        <div
          v-for="playlist in userDetail.playlist.available_playlists"
          :key="playlist.id"
          style="margin-bottom: 12px; padding: 12px; background: #f5f5f5; border-radius: 4px"
        >
          <div style="display: flex; justify-content: space-between; align-items: center">
            <div>
              <strong>{{ playlist.name }}</strong>
            </div>
            <ElTag type="info">{{ playlist.channel_count }} channels</ElTag>
          </div>
        </div>
      </ElCard>

      <!-- Channels Table -->
      <ElCard>
        <template #header>
          <span
            ><strong>Available Channels ({{ channels.length }})</strong></span
          >
        </template>
        <ElTable :data="channels" style="width: 100%" max-height="600">
          <ElTableColumn type="index" label="#" width="60" />
          <ElTableColumn prop="name" label="Channel Name" min-width="200" />
          <ElTableColumn prop="playlist_name" label="Playlist" width="180" />
          <ElTableColumn prop="category" label="Category" width="150" />
          <ElTableColumn label="Status" width="100">
            <template #default="{ row }">
              <ElTag :type="row.enabled ? 'success' : 'danger'" size="small">
                {{ row.enabled ? 'Active' : 'Inactive' }}
              </ElTag>
            </template>
          </ElTableColumn>
        </ElTable>
      </ElCard>
    </div>

    <!-- Extend Subscription Dialog -->
    <ElDialog v-model="showExtendDialog" title="Extend Subscription" width="400px">
      <ElForm :model="extendForm" label-width="120px">
        <ElFormItem label="Days to Extend">
          <ElInput v-model.number="extendForm.days" type="number" placeholder="30" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="showExtendDialog = false">Cancel</ElButton>
        <ElButton type="primary" @click="extendSubscription">Extend</ElButton>
      </template>
    </ElDialog>
  </ContentWrap>
</template>
