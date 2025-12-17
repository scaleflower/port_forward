<script lang="ts" setup>
import { computed, ref, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useRulesStore } from '../stores/rules'
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()
const store = useRulesStore()

// Search
const searchText = ref('')

// Pagination
const currentPage = ref(1)
const pageSize = ref(10)

// Stats refresh interval
let statsInterval: number | null = null

const stats = computed(() => [
  {
    title: '端口转发',
    value: store.forwardRules.length,
    running: store.forwardRules.filter(r => r.status === 'running').length,
    icon: 'Switch',
    color: '#409eff',
    path: '/forward'
  },
  {
    title: '反向代理',
    value: store.reverseRules.length,
    running: store.reverseRules.filter(r => r.status === 'running').length,
    icon: 'RefreshRight',
    color: '#67c23a',
    path: '/reverse'
  },
  {
    title: '代理链',
    value: store.chains.length,
    running: store.chainRules.filter(r => r.status === 'running').length,
    icon: 'Connection',
    color: '#e6a23c',
    path: '/chains'
  },
  {
    title: '错误',
    value: store.errorRules.length,
    running: 0,
    icon: 'Warning',
    color: '#f56c6c',
    path: ''
  }
])

// Filtered rules based on search
const filteredRules = computed(() => {
  if (!searchText.value.trim()) {
    return store.rules
  }

  const search = searchText.value.trim().toLowerCase()
  return store.rules.filter(rule => {
    // Match name
    if (rule.name?.toLowerCase().includes(search)) return true
    // Match local port
    if (rule.localPort?.toString().includes(search)) return true
    // Match target host
    if (rule.targetHost?.toLowerCase().includes(search)) return true
    // Match target port
    if (rule.targetPort?.toString().includes(search)) return true
    // Match remark
    if (rule.remark?.toLowerCase().includes(search)) return true
    return false
  })
})

// Paginated rules
const paginatedRules = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredRules.value.slice(start, end)
})

// Reset to first page when search changes
watch(searchText, () => {
  currentPage.value = 1
})

// Running and stopped counts (based on filtered results)
const runningCount = computed(() => filteredRules.value.filter(r => r.status === 'running').length)
const stoppedCount = computed(() => filteredRules.value.filter(r => r.status !== 'running').length)

// Total traffic
const totalBytesIn = computed(() => {
  let total = 0
  for (const ruleId in store.ruleStats) {
    total += store.ruleStats[ruleId]?.bytesIn || 0
  }
  return total
})

const totalBytesOut = computed(() => {
  let total = 0
  for (const ruleId in store.ruleStats) {
    total += store.ruleStats[ruleId]?.bytesOut || 0
  }
  return total
})

// Total connections
const totalConnections = computed(() => {
  let total = 0
  for (const ruleId in store.ruleStats) {
    total += store.ruleStats[ruleId]?.connections || 0
  }
  return total
})

const totalActiveConns = computed(() => {
  let total = 0
  for (const ruleId in store.ruleStats) {
    total += store.ruleStats[ruleId]?.activeConns || 0
  }
  return total
})

onMounted(() => {
  // Refresh stats every 3 seconds
  statsInterval = window.setInterval(() => {
    store.fetchStats()
  }, 3000)
})

onUnmounted(() => {
  if (statsInterval) {
    clearInterval(statsInterval)
  }
})

function navigateTo(path: string) {
  if (path) router.push(path)
}

async function toggleRule(id: string, running: boolean) {
  if (running) {
    await store.stopRule(id)
  } else {
    await store.startRule(id)
  }
  // Refresh stats after toggle
  await store.fetchStats()
}

// Start all filtered rules
async function startAllRules() {
  const stoppedRules = filteredRules.value.filter(r => r.status !== 'running')
  if (stoppedRules.length === 0) {
    ElMessage.info('所有规则已在运行中')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要启动所有 ${stoppedRules.length} 个未运行的规则吗？`,
      '一键启动',
      { confirmButtonText: '启动', cancelButtonText: '取消', type: 'info' }
    )

    let successCount = 0
    let failCount = 0

    for (const rule of stoppedRules) {
      try {
        await store.startRule(rule.id)
        successCount++
      } catch (e) {
        failCount++
      }
    }

    if (failCount === 0) {
      ElMessage.success(`成功启动 ${successCount} 个规则`)
    } else {
      ElMessage.warning(`启动完成: ${successCount} 成功, ${failCount} 失败`)
    }
  } catch {
    // User cancelled
  }
}

// Stop all filtered rules
async function stopAllRules() {
  const runningRules = filteredRules.value.filter(r => r.status === 'running')
  if (runningRules.length === 0) {
    ElMessage.info('没有正在运行的规则')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要停止所有 ${runningRules.length} 个运行中的规则吗？`,
      '一键停止',
      { confirmButtonText: '停止', cancelButtonText: '取消', type: 'warning' }
    )

    let successCount = 0
    let failCount = 0

    for (const rule of runningRules) {
      try {
        await store.stopRule(rule.id)
        successCount++
      } catch (e) {
        failCount++
      }
    }

    if (failCount === 0) {
      ElMessage.success(`成功停止 ${successCount} 个规则`)
    } else {
      ElMessage.warning(`停止完成: ${successCount} 成功, ${failCount} 失败`)
    }
  } catch {
    // User cancelled
  }
}

function handlePageChange(page: number) {
  currentPage.value = page
}

function handleSizeChange(size: number) {
  pageSize.value = size
  currentPage.value = 1
}

function clearSearch() {
  searchText.value = ''
}

// Format listen address
function formatListenAddr(row: any): string {
  if (row.localPort) {
    return `:${row.localPort}`
  }
  return '-'
}

// Format target address
function formatTarget(row: any): string {
  if (row.targetHost && row.targetPort) {
    return `${row.targetHost}:${row.targetPort}`
  }
  if (row.targets && row.targets.length > 0 && row.targets[0].addr) {
    return row.targets[0].addr
  }
  return '-'
}

// Get additional targets count
function getExtraTargetsCount(row: any): number {
  if (row.targets && row.targets.length > 1) {
    return row.targets.length - 1
  }
  return 0
}

// Format bytes to human readable
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// Get stats for a rule
function getRuleStats(ruleId: string) {
  return store.ruleStats[ruleId] || { bytesIn: 0, bytesOut: 0, connections: 0, activeConns: 0 }
}
</script>

<template>
  <div class="dashboard">
    <!-- Stats Cards -->
    <el-row :gutter="20" class="stats-row">
      <el-col :span="6" v-for="stat in stats" :key="stat.title">
        <el-card class="stat-card" shadow="hover" @click="navigateTo(stat.path)">
          <div class="stat-content">
            <div class="stat-icon" :style="{ backgroundColor: stat.color }">
              <el-icon :size="24"><component :is="stat.icon" /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stat.value }}</div>
              <div class="stat-label">{{ stat.title }}</div>
              <div class="stat-running" v-if="stat.running > 0">
                {{ stat.running }} 运行中
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Traffic Summary -->
    <el-row :gutter="20" class="traffic-row">
      <el-col :span="8">
        <el-card class="traffic-card" shadow="hover">
          <div class="traffic-content">
            <el-icon :size="32" color="#67c23a"><Download /></el-icon>
            <div class="traffic-info">
              <div class="traffic-value">{{ formatBytes(totalBytesOut) }}</div>
              <div class="traffic-label">总下载流量</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card class="traffic-card" shadow="hover">
          <div class="traffic-content">
            <el-icon :size="32" color="#409eff"><Upload /></el-icon>
            <div class="traffic-info">
              <div class="traffic-value">{{ formatBytes(totalBytesIn) }}</div>
              <div class="traffic-label">总上传流量</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card class="traffic-card" shadow="hover">
          <div class="traffic-content">
            <el-icon :size="32" color="#e6a23c"><Connection /></el-icon>
            <div class="traffic-info">
              <div class="traffic-value">{{ totalActiveConns }} / {{ totalConnections }}</div>
              <div class="traffic-label">当前/总连接数</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- All Rules -->
    <el-card class="rules-card">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <span class="header-title">所有规则 ({{ filteredRules.length }}<span v-if="searchText"> / {{ store.rules.length }}</span>)</span>
          </div>
          <div class="header-center">
            <el-input
              v-model="searchText"
              placeholder="搜索名称/端口/IP地址..."
              clearable
              class="search-input"
              @clear="clearSearch"
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
          </div>
          <div class="header-actions">
            <el-button
              type="success"
              size="small"
              @click="startAllRules"
              :disabled="stoppedCount === 0"
            >
              <el-icon><VideoPlay /></el-icon>
              一键启动 ({{ stoppedCount }})
            </el-button>
            <el-button
              type="danger"
              size="small"
              @click="stopAllRules"
              :disabled="runningCount === 0"
            >
              <el-icon><VideoPause /></el-icon>
              一键停止 ({{ runningCount }})
            </el-button>
            <el-button type="primary" size="small" @click="router.push('/forward')">
              <el-icon><Plus /></el-icon>
              添加规则
            </el-button>
          </div>
        </div>
      </template>

      <el-table :data="paginatedRules" style="width: 100%">
        <el-table-column prop="name" label="名称" min-width="100" />
        <el-table-column prop="type" label="类型" width="90" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.type === 'forward' ? '' : row.type === 'reverse' ? 'success' : 'warning'">
              {{ row.type === 'forward' ? '端口转发' : row.type === 'reverse' ? '反向代理' : '代理链' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="监听地址" width="100" align="center">
          <template #default="{ row }">
            <span class="addr-text">{{ formatListenAddr(row) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="目标" min-width="160">
          <template #default="{ row }">
            <span class="addr-text">
              {{ formatTarget(row) }}
              <el-tag v-if="getExtraTargetsCount(row) > 0" size="small" type="info" class="extra-tag">
                +{{ getExtraTargetsCount(row) }}
              </el-tag>
            </span>
          </template>
        </el-table-column>
        <el-table-column label="下载" width="90" align="center">
          <template #default="{ row }">
            <span class="traffic-text download">
              <el-icon><Download /></el-icon>
              {{ formatBytes(getRuleStats(row.id).bytesOut) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="上传" width="90" align="center">
          <template #default="{ row }">
            <span class="traffic-text upload">
              <el-icon><Upload /></el-icon>
              {{ formatBytes(getRuleStats(row.id).bytesIn) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="连接数" width="80" align="center">
          <template #default="{ row }">
            <span class="conn-text">
              <el-icon><Connection /></el-icon>
              {{ getRuleStats(row.id).activeConns }}/{{ getRuleStats(row.id).connections }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 'running' ? 'success' : row.status === 'error' ? 'danger' : 'info'">
              {{ row.status === 'running' ? '运行中' : row.status === 'error' ? '错误' : '已停止' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" width="80" align="center">
          <template #default="{ row }">
            <span>{{ row.remark || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80" align="center">
          <template #default="{ row }">
            <el-button
              size="small"
              :type="row.status === 'running' ? 'danger' : 'success'"
              @click="toggleRule(row.id, row.status === 'running')"
            >
              {{ row.status === 'running' ? '停止' : '启动' }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- Pagination -->
      <div class="pagination-wrapper" v-if="filteredRules.length > pageSize">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredRules.length"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>

      <div v-if="filteredRules.length === 0 && searchText" class="empty-state">
        <el-empty :description="`未找到匹配 '${searchText}' 的规则`">
          <el-button type="primary" @click="clearSearch">清除搜索</el-button>
        </el-empty>
      </div>

      <div v-if="store.rules.length === 0 && !searchText" class="empty-state">
        <el-empty description="暂无规则">
          <el-button type="primary" @click="router.push('/forward')">创建第一个规则</el-button>
        </el-empty>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.stats-row {
  margin-bottom: 0;
}

.traffic-row {
  margin-bottom: 0;
}

.stat-card {
  cursor: pointer;
  transition: transform 0.2s;
}

.stat-card:hover {
  transform: translateY(-4px);
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #303133;
}

.stat-label {
  font-size: 14px;
  color: #909399;
}

.stat-running {
  font-size: 12px;
  color: #67c23a;
  margin-top: 4px;
}

.traffic-card {
  cursor: default;
}

.traffic-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.traffic-info {
  flex: 1;
}

.traffic-value {
  font-size: 24px;
  font-weight: 600;
  color: #303133;
}

.traffic-label {
  font-size: 14px;
  color: #909399;
}

.rules-card {
  flex: 1;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.header-left {
  flex-shrink: 0;
}

.header-title {
  font-weight: 500;
}

.header-center {
  flex: 1;
  max-width: 320px;
}

.search-input {
  width: 100%;
}

.header-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.addr-text {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 13px;
}

.extra-tag {
  margin-left: 4px;
}

.traffic-text {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

.traffic-text.download {
  color: #67c23a;
}

.traffic-text.upload {
  color: #409eff;
}

.conn-text {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  gap: 2px;
  color: #e6a23c;
}

.pagination-wrapper {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.empty-state {
  padding: 40px 0;
}
</style>
