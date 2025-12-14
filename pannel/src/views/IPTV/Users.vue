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
  ElMessageBox
} from 'element-plus'
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import request from '@/axios'

const router = useRouter()
const users = ref<any[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const showExtendDialog = ref(false)
const selectedUser = ref<any>(null)

const userForm = reactive({
  username: '',
  password: '',
  duration_days: 30
})

const extendForm = reactive({
  days: 30
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
      users.value.unshift(res.data)
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
    await ElMessageBox.confirm(
      `${actionText} user "${user.username}"?`,
      `${actionText} User`,
      {
        confirmButtonText: actionText,
        cancelButtonText: 'Cancel',
        type: 'warning'
      }
    )

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

const showUserDetail = (user: any) => {
  router.push(`/users/detail/${user.id}`)
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

    <ElTable :data="users" v-loading="loading" style="width: 100%">
      <ElTableColumn prop="username" label="Username" min-width="150">
        <template #default="{ row }">
          <ElButton type="primary" text @click="showUserDetail(row)" style="padding: 0">
            {{ row.username }}
          </ElButton>
        </template>
      </ElTableColumn>

      <ElTableColumn label="Status" width="120">
        <template #default="{ row }">
          <ElTag :type="getStatusType(row.expiry)">
            {{ formatExpiry(row.expiry) }}
          </ElTag>
        </template>
      </ElTableColumn>

      <ElTableColumn prop="created_at" label="Created" width="180">
        <template #default="{ row }">
          {{ new Date(row.created_at).toLocaleString() }}
        </template>
      </ElTableColumn>

      <ElTableColumn label="Actions" width="300" fixed="right">
        <template #default="{ row }">
          <ElButton type="primary" size="small" text @click="openExtendDialog(row)">
            <template #icon>
              <Icon icon="ep:calendar" />
            </template>
            Extend
          </ElButton>
          <ElButton 
            :type="row.disabled ? 'success' : 'warning'" 
            size="small" 
            text 
            @click="toggleUserStatus(row)"
          >
            <template #icon>
              <Icon :icon="row.disabled ? 'ep:check' : 'ep:close'" />
            </template>
            {{ row.disabled ? 'Enable' : 'Disable' }}
          </ElButton>
          <ElButton type="danger" size="small" @click="deleteUser(row)">
            <template #icon>
              <Icon icon="ep:delete" />
            </template>
          </ElButton>
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
  </ContentWrap>
</template>
