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
  ElDatePicker,
  ElMessage,
  ElMessageBox,
  ElDropdown,
  ElDropdownMenu,
  ElDropdownItem
} from 'element-plus'
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import request from '@/axios'

const router = useRouter()
const users = ref<any[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const showExtendDialog = ref(false)
const showExpiryDialog = ref(false)
const selectedUser = ref<any>(null)

const userForm = reactive({
  username: '',
  password: '',
  duration_days: 30
})

const extendForm = reactive({
  days: 30
})

const expiryForm = reactive<{ expiresAt: Date }>({
  expiresAt: new Date(Date.now() - 24 * 60 * 60 * 1000)
})

const loadUsers = async () => {
  loading.value = true
  try {
    const res = await request.get({ url: '/api/users' })
    if (res && res.data) {
      // Map is_active to disabled for UI
      users.value = res.data.map((user: any) => ({
        ...user,
        disabled: !user.is_active
      }))
    }
  } catch (error) {
    console.error('Error loading users:', error)
    ElMessage.error('Failed to load users')
  } finally {
    loading.value = false
  }
}

const createUser = async () => {
  if (!userForm.username || !userForm.password) {
    ElMessage.warning('Username and password are required')
    return
  }

  try {
    const res = await request.post({ url: '/api/users', data: userForm })

    // Add new user to the list instantly
    if (res && res.data) {
      users.value.unshift({
        ...res.data,
        disabled: !res.data.is_active
      })
    }

    ElMessage.success('User created successfully')
    showCreateDialog.value = false

    // Reset form
    userForm.username = ''
    userForm.password = ''
    userForm.duration_days = 30
  } catch (error) {
    ElMessage.error('Failed to create user')
  }
}

const deleteUser = async (user: any) => {
  try {
    await ElMessageBox.confirm(`Delete user "${user.username}"?`, 'Confirm', {
      confirmButtonText: 'Delete',
      cancelButtonText: 'Cancel',
      type: 'warning'
    })

    await request.delete({ url: `/api/users/${user.id}` })

    // Instant UI update - remove deleted user
    users.value = users.value.filter((u: any) => u.id !== user.id)

    ElMessage.success('User deleted successfully')
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('Failed to delete user')
    }
  }
}

const openExtendDialog = (user: any) => {
  selectedUser.value = user
  extendForm.days = 30
  showExtendDialog.value = true
}

const openExpiryDialog = (user: any) => {
  selectedUser.value = user

  // Default: if user has expiry, prefill it; otherwise set to yesterday (expired)
  if (user?.expires_at) {
    const parsed = new Date(user.expires_at)
    expiryForm.expiresAt = isNaN(parsed.getTime())
      ? new Date(Date.now() - 24 * 60 * 60 * 1000)
      : parsed
  } else {
    expiryForm.expiresAt = new Date(Date.now() - 24 * 60 * 60 * 1000)
  }

  showExpiryDialog.value = true
}

const setExpiredNow = async (user: any) => {
  try {
    await ElMessageBox.confirm(`Set user "${user.username}" as expired now?`, 'Confirm', {
      confirmButtonText: 'Set Expired',
      cancelButtonText: 'Cancel',
      type: 'warning'
    })

    // Set to 1 minute ago to guarantee expired
    const expiresAt = new Date(Date.now() - 60 * 1000)
    await request.post({
      url: `/api/users/${user.id}/set-expired`,
      data: { expires_at: expiresAt.toISOString() }
    })

    ElMessage.success('User marked as expired')
    loadUsers()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('Failed to set expired')
    }
  }
}

const saveExpiry = async () => {
  if (!selectedUser.value) return

  try {
    await request.post({
      url: `/api/users/${selectedUser.value.id}/set-expired`,
      data: { expires_at: expiryForm.expiresAt.toISOString() }
    })

    ElMessage.success('Expiry updated')
    showExpiryDialog.value = false
    loadUsers()
  } catch (error) {
    ElMessage.error('Failed to update expiry')
  }
}

const clearExpiryForUser = async (user: any, closeDialog = false) => {
  try {
    await ElMessageBox.confirm(
      `Clear expiry for user "${user.username}" (set unlimited)?`,
      'Confirm',
      {
        confirmButtonText: 'Clear Expiry',
        cancelButtonText: 'Cancel',
        type: 'warning'
      }
    )

    await request.post({
      url: `/api/users/${user.id}/set-expired`,
      data: { clear: true }
    })

    ElMessage.success('Expiry cleared (unlimited)')
    if (closeDialog) {
      showExpiryDialog.value = false
    }
    loadUsers()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('Failed to clear expiry')
    }
  }
}

const extendSubscription = async () => {
  if (!selectedUser.value) return

  try {
    await request.post({
      url: `/api/users/${selectedUser.value.id}/extend`,
      data: { days: extendForm.days }
    })
    ElMessage.success('Subscription extended successfully')
    showExtendDialog.value = false
    loadUsers()
  } catch (error) {
    ElMessage.error('Failed to extend subscription')
  }
}

const toggleUserStatus = async (user: any) => {
  const action = user.disabled ? 'enable' : 'disable'
  const actionText = user.disabled ? 'Enable' : 'Disable'

  try {
    await ElMessageBox.confirm(`${actionText} user "${user.username}"?`, `${actionText} User`, {
      confirmButtonText: actionText,
      cancelButtonText: 'Cancel',
      type: 'warning'
    })

    const res = await request.post({
      url: `/api/users/${user.id}/toggle`
    })

    // Update user status instantly in UI from response
    if (res && res.data) {
      user.is_active = res.data.is_active
      user.disabled = !res.data.is_active
    }

    ElMessage.success(`User ${action}d successfully`)
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(`Failed to ${action} user`)
    }
  }
}

const getStatusType = (expiresAt?: string | null) => {
  if (!expiresAt) return 'info'
  const expiryDate = new Date(expiresAt)
  const now = new Date()

  if (expiryDate < now) return 'danger'

  const daysLeft = Math.floor((expiryDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
  if (daysLeft <= 7) return 'warning'

  return 'success'
}

const formatExpiry = (expiresAt?: string | null) => {
  if (!expiresAt) return 'No expiry'
  const date = new Date(expiresAt)
  const now = new Date()

  if (date < now) return 'Expired'

  return date.toLocaleDateString()
}

const showUserDetail = (user: any) => {
  router.push(`/users/detail/${user.id}`)
}

const handleDropdown = (command: string, row: any) => {
  if (command === 'set_expiry') {
    openExpiryDialog(row)
  } else if (command === 'clear_expiry') {
    clearExpiryForUser(row)
  } else if (command === 'set_expired') {
    setExpiredNow(row)
  }
}

onMounted(() => {
  loadUsers()
})
</script>

<template>
  <ContentWrap title="Users" message="Manage user accounts and subscriptions">
    <ElButton type="primary" @click="showCreateDialog = true" style="margin-bottom: 16px">
      <template #icon>
        <Icon icon="ep:plus" />
      </template>
      Create New User
    </ElButton>

    <ElTable :data="users" v-loading="loading" style="width: 100%" stripe :header-cell-style="{ background: '#f5f7fa', color: '#606266', fontWeight: 'bold' }">
      <ElTableColumn prop="username" label="Username" min-width="150">
        <template #default="{ row }">
          <div class="flex items-center gap-2">
            <ElButton type="primary" link @click="showUserDetail(row)" class="font-bold text-lg">
              {{ row.username }}
            </ElButton>
            <ElTag v-if="row.disabled" type="danger" size="small" effect="dark">Disabled</ElTag>
          </div>
        </template>
      </ElTableColumn>

      <ElTableColumn label="Status / Expiry" width="160">
        <template #default="{ row }">
          <ElTag :type="getStatusType(row.expires_at)" effect="light" class="font-semibold" round>
            <div style="display: flex; align-items: center; gap: 4px;">
              <Icon v-if="getStatusType(row.expires_at) === 'success'" icon="ep:circle-check" />
              <Icon v-else-if="getStatusType(row.expires_at) === 'warning'" icon="ep:warning" />
              <Icon v-else-if="getStatusType(row.expires_at) === 'danger'" icon="ep:circle-close" />
              <Icon v-else icon="ep:info-filled" />
              <span>{{ formatExpiry(row.expires_at) }}</span>
            </div>
          </ElTag>
        </template>
      </ElTableColumn>

      <ElTableColumn label="Days Left" width="120" align="center">
        <template #default="{ row }">
          <span v-if="row?.is_expired" style="color: var(--el-color-danger); font-weight: bold;">Expired</span>
          <span v-else-if="!row?.expires_at" style="color: var(--el-color-info); font-weight: bold;">Unlimited</span>
          <span v-else style="font-weight: bold;" :style="{ color: (row?.days_remaining || 0) <= 7 ? 'var(--el-color-warning)' : 'var(--el-color-success)' }">
            {{ row?.days_remaining }} days
          </span>
        </template>
      </ElTableColumn>

      <ElTableColumn prop="created_at" label="Created At" width="180">
        <template #default="{ row }">
          <span style="color: #909399; font-size: 0.9em;">
            <Icon icon="ep:clock" style="vertical-align: middle; margin-right: 4px;" />
            {{ new Date(row.created_at).toLocaleDateString() }}
          </span>
        </template>
      </ElTableColumn>

      <ElTableColumn label="Actions" width="220" fixed="right" align="center">
        <template #default="{ row }">
          <div style="display: flex; justify-content: center; gap: 8px; align-items: center;">
            <ElButton type="primary" size="small" plain @click="openExtendDialog(row)" title="Extend Subscription">
              <template #icon><Icon icon="ep:calendar" /></template>
              Extend
            </ElButton>
            
            <ElButton
              :type="row.disabled ? 'success' : 'warning'"
              size="small"
              circle
              plain
              @click="toggleUserStatus(row)"
              :title="row.disabled ? 'Enable User' : 'Disable User'"
            >
              <template #icon><Icon :icon="row.disabled ? 'ep:video-play' : 'ep:video-pause'" /></template>
            </ElButton>
            
            <ElButton type="danger" size="small" circle plain @click="deleteUser(row)" title="Delete User">
              <template #icon><Icon icon="ep:delete" /></template>
            </ElButton>

            <ElDropdown trigger="click" @command="(cmd) => handleDropdown(cmd, row)">
              <ElButton type="info" size="small" circle plain title="More Actions">
                <template #icon><Icon icon="ep:more" /></template>
              </ElButton>
              <template #dropdown>
                <ElDropdownMenu>
                  <ElDropdownItem command="set_expiry">
                    <Icon icon="ep:date" style="margin-right: 8px;" /> Set Expiry Date
                  </ElDropdownItem>
                  <ElDropdownItem command="clear_expiry">
                    <Icon icon="ep:refresh" style="margin-right: 8px;" /> Clear Expiry (Unlimited)
                  </ElDropdownItem>
                  <ElDropdownItem command="set_expired" divided style="color: var(--el-color-danger)">
                    <Icon icon="ep:warning" style="margin-right: 8px;" /> Mark as Expired Now
                  </ElDropdownItem>
                </ElDropdownMenu>
              </template>
            </ElDropdown>
          </div>
        </template>
      </ElTableColumn>
    </ElTable>

    <!-- Create User Dialog -->
    <ElDialog v-model="showCreateDialog" title="Create New User" width="500px">
      <ElForm :model="userForm" label-width="120px">
        <ElFormItem label="Username">
          <ElInput v-model="userForm.username" placeholder="Username" />
        </ElFormItem>
        <ElFormItem label="Password">
          <ElInput
            v-model="userForm.password"
            type="password"
            placeholder="Password"
            show-password
          />
        </ElFormItem>
        <ElFormItem label="Duration (Days)">
          <ElInput
            v-model.number="userForm.duration_days"
            type="number"
            placeholder="30 (0 for unlimited)"
            style="width: 100%"
          >
            <template #append>days</template>
          </ElInput>
          <div style="font-size: 12px; color: #909399; margin-top: 4px">
            Enter number of days (0 = unlimited)
          </div>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="showCreateDialog = false">Cancel</ElButton>
        <ElButton type="primary" @click="createUser">Create</ElButton>
      </template>
    </ElDialog>

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

    <!-- Set Expiry Dialog -->
    <ElDialog v-model="showExpiryDialog" title="Set Expiry" width="450px">
      <ElForm :model="expiryForm" label-width="140px">
        <ElFormItem label="Expires At">
          <ElDatePicker
            v-model="expiryForm.expiresAt"
            type="datetime"
            placeholder="Select expiry date/time"
            style="width: 100%"
          />
          <div style="font-size: 12px; color: #909399; margin-top: 4px">
            Choose a past date/time to mark user as expired.
          </div>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="showExpiryDialog = false">Cancel</ElButton>
        <ElButton type="warning" @click="selectedUser && clearExpiryForUser(selectedUser, true)">
          Clear Expiry
        </ElButton>
        <ElButton type="primary" @click="saveExpiry">Save</ElButton>
      </template>
    </ElDialog>
  </ContentWrap>
</template>
