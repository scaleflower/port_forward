import { createRouter, createWebHashHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'Dashboard',
    component: () => import('../views/Dashboard.vue')
  },
  {
    path: '/forward',
    name: 'PortForward',
    component: () => import('../views/PortForward.vue')
  },
  {
    path: '/reverse',
    name: 'ReverseProxy',
    component: () => import('../views/ReverseProxy.vue')
  },
  {
    path: '/chains',
    name: 'ProxyChain',
    component: () => import('../views/ProxyChain.vue')
  },
  {
    path: '/logs',
    name: 'Logs',
    component: () => import('../views/Logs.vue')
  },
  {
    path: '/settings',
    name: 'Settings',
    component: () => import('../views/Settings.vue')
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router
