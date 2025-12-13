<script setup lang="ts">
import { ElDropdown, ElDropdownMenu, ElDropdownItem } from 'element-plus'
import { useI18n } from '@/hooks/web/useI18n'
import { useDesign } from '@/hooks/web/useDesign'
import { Icon } from '@/components/Icon'
import LockDialog from './components/LockDialog.vue'
import { ref, computed } from 'vue'
import LockPage from './components/LockPage.vue'
import { useLockStore } from '@/store/modules/lock'
import { useUserStore } from '@/store/modules/user'
import { useRouter } from 'vue-router'

const { push } = useRouter()

const userStore = useUserStore()

const lockStore = useLockStore()

const getIsLock = computed(() => lockStore.getLockInfo?.isLock ?? false)

const { getPrefixCls } = useDesign()

const prefixCls = getPrefixCls('user-info')

const { t } = useI18n()

const loginOut = () => {
  userStore.logoutConfirm()
}

const dialogVisible = ref<boolean>(false)

// 锁定屏幕
const lockScreen = () => {
  dialogVisible.value = true
}

const toSettings = () => {
  push('/settings/index')
}

const toDashboard = () => {
  push('/dashboard/index')
}
</script>

<template>
  <ElDropdown class="custom-hover" :class="prefixCls" trigger="click">
    <div class="flex items-center">
      <img
        src="@/assets/imgs/avatar.jpg"
        alt=""
        class="w-[calc(var(--logo-height)-25px)] rounded-[50%]"
      />
      <span class="<lg:hidden text-14px pl-[5px] text-[var(--top-header-text-color)]">{{
        userStore.getUserInfo?.username
      }}</span>
    </div>
    <template #dropdown>
      <ElDropdownMenu>
        <ElDropdownItem>
          <div @click="toDashboard">
            <Icon icon="ep:data-analysis" class="mr-2" />
            Dashboard
          </div>
        </ElDropdownItem>
        <ElDropdownItem>
          <div @click="toSettings">
            <Icon icon="ep:setting" class="mr-2" />
            Settings
          </div>
        </ElDropdownItem>
        <ElDropdownItem divided>
          <div @click="loginOut">
            <Icon icon="ep:switch-button" class="mr-2" />
            Logout
          </div>
        </ElDropdownItem>
      </ElDropdownMenu>
    </template>
  </ElDropdown>

  <LockDialog v-if="dialogVisible" v-model="dialogVisible" />
  <teleport to="body">
    <transition name="fade-bottom" mode="out-in">
      <LockPage v-if="getIsLock" />
    </transition>
  </teleport>
</template>

<style scoped lang="less">
.fade-bottom-enter-active,
.fade-bottom-leave-active {
  transition:
    opacity 0.25s,
    transform 0.3s;
}

.fade-bottom-enter-from {
  opacity: 0;
  transform: translateY(-10%);
}

.fade-bottom-leave-to {
  opacity: 0;
  transform: translateY(10%);
}
</style>
