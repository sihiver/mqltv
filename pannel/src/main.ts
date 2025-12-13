import 'vue/jsx'

// 引入windi css
import '@/plugins/unocss'

// 导入全局的svg图标
import '@/plugins/svgIcon'

// 初始化多语言
import { setupI18n } from '@/plugins/vueI18n'

// 引入状态管理
import { setupStore } from '@/store'

// 全局组件
import { setupGlobCom } from '@/components'

// 引入element-plus
import { setupElementPlus } from '@/plugins/elementPlus'

// 引入全局样式
import '@/styles/index.less'

// 引入动画
import '@/plugins/animate.css'

// 路由
import { setupRouter } from './router'

// 权限
import { setupPermission } from './directives'

import { createApp } from 'vue'

import App from './App.vue'

import './permission'

import axios from 'axios'

// Load server name from backend
const loadServerName = async () => {
  try {
    const response = await axios.get('/api/settings', { withCredentials: true })
    if (response.data && response.data.code === 0 && response.data.data?.system?.server_name) {
      const title = response.data.data.system.server_name
      document.title = title
      localStorage.setItem('app_title', title)
    }
  } catch (error) {
    console.log('Using default title')
  }
}

// 创建实例
const setupAll = async () => {
  const app = createApp(App)

  await setupI18n(app)

  setupStore(app)

  // Load server name early
  await loadServerName()

  setupGlobCom(app)

  setupElementPlus(app)

  setupRouter(app)

  setupPermission(app)

  app.mount('#app')
}

setupAll()
