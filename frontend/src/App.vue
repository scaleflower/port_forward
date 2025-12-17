<script lang="ts" setup>
import { onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useRulesStore } from './stores/rules'

const router = useRouter()
const route = useRoute()
const store = useRulesStore()

const menuItems = [
  { path: '/', icon: 'DataAnalysis', title: '仪表盘' },
  { path: '/forward', icon: 'Switch', title: '端口转发' },
  { path: '/reverse', icon: 'RefreshRight', title: '反向代理' },
  { path: '/chains', icon: 'Connection', title: '代理链' },
  { path: '/logs', icon: 'Document', title: '日志' },
  { path: '/settings', icon: 'Setting', title: '设置' }
]

onMounted(async () => {
  await store.init()
})

function navigateTo(path: string) {
  router.push(path)
}
</script>

<template>
  <el-container class="app-container">
    <!-- Sidebar -->
    <el-aside width="200px" class="sidebar">
      <div class="logo">
        <el-icon :size="24"><Switch /></el-icon>
        <span>端口映射管理器</span>
      </div>
      <el-menu
        :default-active="route.path"
        class="sidebar-menu"
        @select="navigateTo"
      >
        <el-menu-item v-for="item in menuItems" :key="item.path" :index="item.path">
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.title }}</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <!-- Main Content -->
    <el-container>
      <!-- Header -->
      <el-header class="header">
        <div class="header-left">
          <h2>{{ menuItems.find(m => m.path === route.path)?.title || '仪表盘' }}</h2>
        </div>
        <div class="header-right">
          <el-tag v-if="store.status" :type="store.status.running ? 'success' : 'danger'" size="small">
            {{ store.status.running ? '运行中' : '已停止' }}
          </el-tag>
          <el-tag type="info" size="small">
            活跃: {{ store.runningRules.length }} / {{ store.rules.length }}
          </el-tag>
        </div>
      </el-header>

      <!-- Content -->
      <el-main class="main-content">
        <router-view />
      </el-main>

      <!-- Footer -->
      <el-footer class="footer" height="32px">
        <span>Port Forward Manager v1.0.0</span>
        <span v-if="store.serviceStatus !== 'not_installed'">
          服务状态: {{ store.serviceStatus === 'running' ? '运行中' : '已停止' }}
        </span>
      </el-footer>
    </el-container>
  </el-container>
</template>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  width: 100%;
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}

.app-container {
  width: 100%;
  height: 100%;
}

.sidebar {
  background-color: #304156;
  color: #fff;
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: #fff;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.sidebar-menu {
  border-right: none;
  background-color: transparent;
}

.sidebar-menu .el-menu-item {
  color: rgba(255, 255, 255, 0.7);
}

.sidebar-menu .el-menu-item:hover {
  background-color: rgba(255, 255, 255, 0.1);
}

.sidebar-menu .el-menu-item.is-active {
  color: #fff;
  background-color: #409eff;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background-color: #fff;
  border-bottom: 1px solid #e4e7ed;
  padding: 0 20px;
}

.header-left h2 {
  font-size: 18px;
  font-weight: 500;
  color: #303133;
}

.header-right {
  display: flex;
  gap: 8px;
}

.main-content {
  background-color: #f5f7fa;
  padding: 20px;
  overflow-y: auto;
}

.footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background-color: #fff;
  border-top: 1px solid #e4e7ed;
  padding: 0 20px;
  font-size: 12px;
  color: #909399;
}
</style>
