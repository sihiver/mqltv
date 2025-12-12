import { createRouter, createWebHashHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import type { App } from 'vue'
import { Layout } from '@/utils/routerHelper'
import { NO_RESET_WHITE_LIST } from '@/constants'

export const constantRouterMap: AppRouteRecordRaw[] = [
  {
    path: '/',
    component: Layout,
    redirect: '/dashboard',
    name: 'Root',
    meta: {
      hidden: true
    }
  },
  {
    path: '/redirect',
    component: Layout,
    name: 'RedirectWrap',
    children: [
      {
        path: '/redirect/:path(.*)',
        name: 'Redirect',
        component: () => import('@/views/Redirect/Redirect.vue'),
        meta: {}
      }
    ],
    meta: {
      hidden: true,
      noTagsView: true
    }
  },
  {
    path: '/login',
    component: () => import('@/views/Login/Login.vue'),
    name: 'Login',
    meta: {
      hidden: true,
      title: 'Login',
      noTagsView: true
    }
  },
  {
    path: '/404',
    component: () => import('@/views/Error/404.vue'),
    name: 'NoFind',
    meta: {
      hidden: true,
      title: '404',
      noTagsView: true
    }
  }
]

export const asyncRouterMap: AppRouteRecordRaw[] = [
  {
    path: '/dashboard',
    component: Layout,
    name: 'Dashboard',
    meta: {},
    children: [
      {
        path: 'index',
        component: () => import('@/views/IPTV/Dashboard.vue'),
        name: 'IPTVDashboard',
        meta: {
          title: 'Dashboard',
          icon: 'vi-ant-design:dashboard-filled',
          noCache: true,
          affix: true
        }
      }
    ]
  },
  {
    path: '/playlists',
    component: Layout,
    name: 'Playlists',
    meta: {},
    children: [
      {
        path: 'index',
        component: () => import('@/views/IPTV/Playlists.vue'),
        name: 'PlaylistsManagement',
        meta: {
          title: 'Playlists',
          icon: 'vi-ep:menu'
        }
      }
    ]
  },
  {
    path: '/channels',
    component: Layout,
    name: 'Channels',
    meta: {},
    children: [
      {
        path: 'index',
        component: () => import('@/views/IPTV/Channels.vue'),
        name: 'ChannelsManagement',
        meta: {
          title: 'Channels',
          icon: 'vi-ep:video-camera'
        }
      }
    ]
  },
  {
    path: '/users',
    component: Layout,
    name: 'Users',
    meta: {},
    children: [
      {
        path: 'index',
        component: () => import('@/views/IPTV/Users.vue'),
        name: 'UsersManagement',
        meta: {
          title: 'Users',
          icon: 'vi-ep:user'
        }
      },
      {
        path: 'detail/:id',
        component: () => import('@/views/IPTV/UserDetail.vue'),
        name: 'UserDetail',
        meta: {
          title: 'User Detail',
          hidden: true,
          noTagsView: false
        }
      }
    ]
  },
  {
    path: '/relays',
    component: Layout,
    name: 'Relays',
    meta: {},
    children: [
      {
        path: 'index',
        component: () => import('@/views/IPTV/Relays.vue'),
        name: 'RelaysManagement',
        meta: {
          title: 'Relays',
          icon: 'vi-ep:video-play'
        }
      }
    ]
  },
  {
    path: '/generate',
    component: Layout,
    name: 'Generate',
    meta: {},
    children: [
      {
        path: 'playlist',
        component: () => import('@/views/IPTV/GeneratePlaylist.vue'),
        name: 'GeneratePlaylist',
        meta: {
          title: 'Generate Playlist',
          icon: 'vi-ep:document-add'
        }
      }
    ]
  },
  {
    path: '/import',
    component: Layout,
    name: 'Import',
    meta: {},
    children: [
      {
        path: 'm3u',
        component: () => import('@/views/IPTV/ImportM3U.vue'),
        name: 'ImportM3U',
        meta: {
          title: 'Import M3U',
          icon: 'vi-ep:upload'
        }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  strict: true,
  routes: constantRouterMap as RouteRecordRaw[],
  scrollBehavior: () => ({ left: 0, top: 0 })
})

export const resetRouter = (): void => {
  router.getRoutes().forEach((route) => {
    const { name } = route
    if (name && !NO_RESET_WHITE_LIST.includes(name as string)) {
      router.hasRoute(name) && router.removeRoute(name)
    }
  })
}

export const setupRouter = (app: App<Element>) => {
  app.use(router)
}

export default router
