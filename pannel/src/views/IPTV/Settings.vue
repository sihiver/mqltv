<script setup lang="ts">
import { ContentWrap } from '@/components/ContentWrap'
import {
  ElRow,
  ElCol,
  ElCard,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElSwitch,
  ElButton,
  ElMessage,
  ElDivider,
  ElTag
} from 'element-plus'
import { ref, onMounted } from 'vue'
import request from '@/axios'
import { useAppStore } from '@/store/modules/app'

const appStore = useAppStore()

// System settings
const systemSettings = ref({
  server_name: 'IPTV Panel',
  server_url: 'http://localhost:8080',
  max_connections_per_user: 3,
  session_timeout: 3600,
  enable_user_registration: false,
  enable_relay_mode: true
})

// FFmpeg settings
const ffmpegSettings = ref({
  ffmpeg_path: '/usr/bin/ffmpeg',
  buffer_size: 2048,
  idle_timeout: 60,
  max_streams: 100,
  enable_hls: true,
  hls_segment_duration: 6
})

// Stream settings
const streamSettings = ref({
  auto_start: true,
  auto_stop: true,
  max_bitrate: 8000,
  enable_transcode: false,
  default_format: 'mpegts'
})

// Account settings
const accountForm = ref({
  current_password: '',
  new_password: '',
  confirm_password: ''
})

const profileForm = ref({
  username: '',
  full_name: '',
  email: ''
})

const loading = ref(false)
const saving = ref(false)

// Load settings
const loadSettings = async () => {
  loading.value = true
  try {
    const res = await request.get({ url: '/api/settings' })
    if (res && res.data) {
      // Map backend data to frontend refs
      if (res.data.system) {
        Object.assign(systemSettings.value, res.data.system)
        // Update app title from server_name
        if (res.data.system.server_name) {
          appStore.setTitle(res.data.system.server_name)
        }
      }
      if (res.data.ffmpeg) {
        Object.assign(ffmpegSettings.value, res.data.ffmpeg)
      }
      if (res.data.stream) {
        Object.assign(streamSettings.value, res.data.stream)
      }
    }
  } catch (error) {
    console.error('Error loading settings:', error)
    ElMessage.error('Failed to load settings')
  } finally {
    loading.value = false
  }
}

// Save system settings
const saveSystemSettings = async () => {
  saving.value = true
  try {
    await request.post({
      url: '/api/settings',
      data: {
        category: 'system',
        settings: systemSettings.value
      }
    })
    // Update app title
    if (systemSettings.value.server_name) {
      appStore.setTitle(systemSettings.value.server_name)
      localStorage.setItem('app_title', systemSettings.value.server_name)
    }
    ElMessage.success('System settings saved successfully')
  } catch (error) {
    ElMessage.error('Failed to save system settings')
  } finally {
    saving.value = false
  }
}

// Save FFmpeg settings
const saveFFmpegSettings = async () => {
  saving.value = true
  try {
    await request.post({
      url: '/api/settings',
      data: {
        category: 'ffmpeg',
        settings: ffmpegSettings.value
      }
    })
    ElMessage.success('FFmpeg settings saved successfully')
  } catch (error) {
    ElMessage.error('Failed to save FFmpeg settings')
  } finally {
    saving.value = false
  }
}

// Save stream settings
const saveStreamSettings = async () => {
  saving.value = true
  try {
    await request.post({
      url: '/api/settings',
      data: {
        category: 'stream',
        settings: streamSettings.value
      }
    })
    ElMessage.success('Stream settings saved successfully')
  } catch (error) {
    ElMessage.error('Failed to save stream settings')
  } finally {
    saving.value = false
  }
}

// Test FFmpeg
const testFFmpeg = async () => {
  try {
    ElMessage.info('Testing FFmpeg...')
    const res = await request.post({ url: '/api/settings/test-ffmpeg' })
    if (res && res.data) {
      ElMessage.success('FFmpeg is working correctly')
    }
  } catch (error) {
    ElMessage.error('FFmpeg test failed')
  }
}

// Clear cache
const clearCache = async () => {
  try {
    await request.post({ url: '/api/settings/clear-cache' })
    ElMessage.success('Cache cleared successfully')
  } catch (error) {
    ElMessage.error('Failed to clear cache')
  }
}

// Load admin profile
const loadProfile = async () => {
  try {
    const res = await request.get({ url: '/api/auth/profile' })
    if (res && res.data) {
      profileForm.value = {
        username: res.data.username || '',
        full_name: res.data.full_name || '',
        email: res.data.email || ''
      }
    }
  } catch (error) {
    console.error('Error loading profile:', error)
  }
}

// Change password
const changePassword = async () => {
  if (!accountForm.value.current_password || !accountForm.value.new_password) {
    ElMessage.warning('Please fill all password fields')
    return
  }

  if (accountForm.value.new_password !== accountForm.value.confirm_password) {
    ElMessage.error('New passwords do not match')
    return
  }

  if (accountForm.value.new_password.length < 6) {
    ElMessage.error('Password must be at least 6 characters')
    return
  }

  saving.value = true
  try {
    await request.post({
      url: '/api/auth/change-password',
      data: {
        old_password: accountForm.value.current_password,
        new_password: accountForm.value.new_password
      }
    })
    ElMessage.success('Password changed successfully')
    accountForm.value = {
      current_password: '',
      new_password: '',
      confirm_password: ''
    }
  } catch (error: any) {
    ElMessage.error(error.message || 'Failed to change password')
  } finally {
    saving.value = false
  }
}

// Update profile
const updateProfile = async () => {
  if (!profileForm.value.username) {
    ElMessage.warning('Username is required')
    return
  }

  saving.value = true
  try {
    await request.post({
      url: '/api/auth/update-profile',
      data: profileForm.value
    })
    ElMessage.success('Profile updated successfully')
  } catch (error) {
    ElMessage.error('Failed to update profile')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadSettings()
  loadProfile()
})
</script>

<template>
  <ContentWrap title="Settings" message="Configure system settings and preferences">
    <ElRow :gutter="20">
      <!-- System Settings -->
      <ElCol :xs="24" :sm="24" :md="12">
        <ElCard shadow="hover">
          <template #header>
            <div style="display: flex; align-items: center; gap: 8px">
              <Icon icon="ep:setting" />
              <span>System Settings</span>
            </div>
          </template>

          <ElForm :model="systemSettings" label-width="200px" v-loading="loading">
            <ElFormItem label="Server Name">
              <ElInput v-model="systemSettings.server_name" placeholder="IPTV Panel" />
            </ElFormItem>

            <ElFormItem label="Server URL">
              <ElInput v-model="systemSettings.server_url" placeholder="http://localhost:8080" />
            </ElFormItem>

            <ElFormItem label="Max Connections/User">
              <ElInputNumber
                v-model="systemSettings.max_connections_per_user"
                :min="1"
                :max="10"
                style="width: 100%"
              />
            </ElFormItem>

            <ElFormItem label="Session Timeout (seconds)">
              <ElInputNumber
                v-model="systemSettings.session_timeout"
                :min="300"
                :max="86400"
                style="width: 100%"
              />
            </ElFormItem>

            <ElFormItem label="User Registration">
              <ElSwitch v-model="systemSettings.enable_user_registration" />
            </ElFormItem>

            <ElFormItem label="Relay Mode">
              <ElSwitch v-model="systemSettings.enable_relay_mode" />
            </ElFormItem>

            <ElFormItem>
              <ElButton type="primary" @click="saveSystemSettings" :loading="saving">
                <Icon icon="ep:upload" />
                Save System Settings
              </ElButton>
            </ElFormItem>
          </ElForm>
        </ElCard>
      </ElCol>

      <!-- FFmpeg Settings -->
      <ElCol :xs="24" :sm="24" :md="12">
        <ElCard shadow="hover">
          <template #header>
            <div style="display: flex; align-items: center; gap: 8px">
              <Icon icon="ep:video-camera" />
              <span>FFmpeg Settings</span>
            </div>
          </template>

          <ElForm :model="ffmpegSettings" label-width="200px" v-loading="loading">
            <ElFormItem label="FFmpeg Path">
              <ElInput v-model="ffmpegSettings.ffmpeg_path" placeholder="/usr/bin/ffmpeg" />
            </ElFormItem>

            <ElFormItem label="Buffer Size (KB)">
              <ElInputNumber
                v-model="ffmpegSettings.buffer_size"
                :min="512"
                :max="10240"
                style="width: 100%"
              />
            </ElFormItem>

            <ElFormItem label="Idle Timeout (seconds)">
              <ElInputNumber
                v-model="ffmpegSettings.idle_timeout"
                :min="30"
                :max="300"
                style="width: 100%"
              />
            </ElFormItem>

            <ElFormItem label="Max Concurrent Streams">
              <ElInputNumber
                v-model="ffmpegSettings.max_streams"
                :min="10"
                :max="500"
                style="width: 100%"
              />
            </ElFormItem>

            <ElFormItem label="Enable HLS">
              <ElSwitch v-model="ffmpegSettings.enable_hls" />
            </ElFormItem>

            <ElFormItem label="HLS Segment Duration (s)">
              <ElInputNumber
                v-model="ffmpegSettings.hls_segment_duration"
                :min="2"
                :max="10"
                style="width: 100%"
              />
            </ElFormItem>

            <ElFormItem>
              <div style="display: flex; gap: 8px">
                <ElButton type="primary" @click="saveFFmpegSettings" :loading="saving">
                  <Icon icon="ep:upload" />
                  Save FFmpeg Settings
                </ElButton>
                <ElButton @click="testFFmpeg">
                  <Icon icon="ep:video-play" />
                  Test FFmpeg
                </ElButton>
              </div>
            </ElFormItem>
          </ElForm>
        </ElCard>
      </ElCol>

      <!-- Stream Settings -->
      <ElCol :xs="24" :sm="24" :md="12" style="margin-top: 20px">
        <ElCard shadow="hover">
          <template #header>
            <div style="display: flex; align-items: center; gap: 8px">
              <Icon icon="ep:video-play" />
              <span>Stream Settings</span>
            </div>
          </template>

          <ElForm :model="streamSettings" label-width="200px" v-loading="loading">
            <ElFormItem label="Auto Start Streams">
              <ElSwitch v-model="streamSettings.auto_start" />
            </ElFormItem>

            <ElFormItem label="Auto Stop Idle Streams">
              <ElSwitch v-model="streamSettings.auto_stop" />
            </ElFormItem>

            <ElFormItem label="Max Bitrate (kbps)">
              <ElInputNumber
                v-model="streamSettings.max_bitrate"
                :min="1000"
                :max="20000"
                style="width: 100%"
              />
            </ElFormItem>

            <ElFormItem label="Enable Transcode">
              <ElSwitch v-model="streamSettings.enable_transcode" />
            </ElFormItem>

            <ElFormItem label="Default Format">
              <ElInput v-model="streamSettings.default_format" placeholder="mpegts" />
            </ElFormItem>

            <ElFormItem>
              <ElButton type="primary" @click="saveStreamSettings" :loading="saving">
                <Icon icon="ep:upload" />
                Save Stream Settings
              </ElButton>
            </ElFormItem>
          </ElForm>
        </ElCard>
      </ElCol>

      <!-- System Actions -->
      <ElCol :xs="24" :sm="24" :md="12" style="margin-top: 20px">
        <ElCard shadow="hover">
          <template #header>
            <div style="display: flex; align-items: center; gap: 8px">
              <Icon icon="ep:tools" />
              <span>System Actions</span>
            </div>
          </template>

          <div style="padding: 20px">
            <ElFormItem label="Cache Management">
              <div style="display: flex; gap: 8px; align-items: center">
                <ElButton @click="clearCache">
                  <Icon icon="ep:delete" />
                  Clear HLS Cache
                </ElButton>
                <span style="color: #909399; font-size: 12px"> Clear all cached HLS segments </span>
              </div>
            </ElFormItem>

            <ElDivider />

            <ElFormItem label="System Information">
              <div style="display: flex; flex-direction: column; gap: 8px">
                <div style="display: flex; justify-content: space-between">
                  <span>Version:</span>
                  <ElTag>v1.0.0</ElTag>
                </div>
                <div style="display: flex; justify-content: space-between">
                  <span>Go Version:</span>
                  <ElTag type="success">1.21+</ElTag>
                </div>
                <div style="display: flex; justify-content: space-between">
                  <span>Database:</span>
                  <ElTag type="info">SQLite</ElTag>
                </div>
                <div style="display: flex; justify-content: space-between">
                  <span>FFmpeg:</span>
                  <ElTag type="warning">Installed</ElTag>
                </div>
              </div>
            </ElFormItem>
          </div>
        </ElCard>
      </ElCol>

      <!-- Account Settings - Change Password -->
      <ElCol :xs="24" :sm="24" :md="12" style="margin-top: 20px">
        <ElCard shadow="hover">
          <template #header>
            <div style="display: flex; align-items: center; gap: 8px">
              <Icon icon="ep:lock" />
              <span>Change Password</span>
            </div>
          </template>

          <ElForm :model="accountForm" label-width="200px">
            <ElFormItem label="Current Password">
              <ElInput
                v-model="accountForm.current_password"
                type="password"
                placeholder="Enter current password"
                show-password
              />
            </ElFormItem>

            <ElFormItem label="New Password">
              <ElInput
                v-model="accountForm.new_password"
                type="password"
                placeholder="At least 6 characters"
                show-password
              />
            </ElFormItem>

            <ElFormItem label="Confirm Password">
              <ElInput
                v-model="accountForm.confirm_password"
                type="password"
                placeholder="Re-enter new password"
                show-password
              />
            </ElFormItem>

            <ElFormItem>
              <ElButton type="primary" @click="changePassword" :loading="saving">
                <Icon icon="ep:check" />
                Change Password
              </ElButton>
            </ElFormItem>
          </ElForm>
        </ElCard>
      </ElCol>

      <!-- Account Settings - Profile -->
      <ElCol :xs="24" :sm="24" :md="12" style="margin-top: 20px">
        <ElCard shadow="hover">
          <template #header>
            <div style="display: flex; align-items: center; gap: 8px">
              <Icon icon="ep:user" />
              <span>Profile Settings</span>
            </div>
          </template>

          <ElForm :model="profileForm" label-width="200px">
            <ElFormItem label="Username">
              <ElInput v-model="profileForm.username" placeholder="Admin username" disabled />
            </ElFormItem>

            <ElFormItem label="Full Name">
              <ElInput v-model="profileForm.full_name" placeholder="Your full name" />
            </ElFormItem>

            <ElFormItem label="Email">
              <ElInput v-model="profileForm.email" placeholder="your.email@example.com" />
            </ElFormItem>

            <ElFormItem>
              <ElButton type="primary" @click="updateProfile" :loading="saving">
                <Icon icon="ep:upload" />
                Update Profile
              </ElButton>
            </ElFormItem>
          </ElForm>
        </ElCard>
      </ElCol>
    </ElRow>
  </ContentWrap>
</template>

<style scoped>
.el-card {
  margin-bottom: 20px;
}

.el-form-item {
  margin-bottom: 18px;
}
</style>
