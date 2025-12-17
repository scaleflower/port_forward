<script lang="ts" setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRulesStore } from '../stores/rules'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { LogEntry } from '../types'

const store = useRulesStore()

// Filters
const filterLevel = ref<string>('all')
const filterRule = ref<string>('')
const searchText = ref<string>('')
const autoScroll = ref(true)

// Log refresh interval
let logInterval: number | null = null
const logContainer = ref<HTMLElement | null>(null)

// Filtered logs
const filteredLogs = computed(() => {
  let result = [...store.logs]

  // Filter by level
  if (filterLevel.value !== 'all') {
    result = result.filter(log => log.level === filterLevel.value)
  }

  // Filter by rule
  if (filterRule.value) {
    result = result.filter(log => log.ruleId === filterRule.value)
  }

  // Filter by search text
  if (searchText.value) {
    const search = searchText.value.toLowerCase()
    result = result.filter(log =>
      log.message.toLowerCase().includes(search) ||
      log.ruleName?.toLowerCase().includes(search) ||
      log.details?.toLowerCase().includes(search)
    )
  }

  return result
})

// Unique rules from logs
const logRules = computed(() => {
  const rules = new Map<string, string>()
  store.logs.forEach(log => {
    if (log.ruleId && log.ruleName) {
      rules.set(log.ruleId, log.ruleName)
    }
  })
  return Array.from(rules, ([id, name]) => ({ id, name }))
})

// Log counts by level
const logCounts = computed(() => {
  const counts = { debug: 0, info: 0, warn: 0, error: 0 }
  store.logs.forEach(log => {
    if (log.level in counts) {
      counts[log.level as keyof typeof counts]++
    }
  })
  return counts
})

onMounted(async () => {
  // Fetch initial logs
  await store.fetchLogs(500)

  // Start polling for new logs
  logInterval = window.setInterval(async () => {
    await store.fetchNewLogs()
    if (autoScroll.value && logContainer.value) {
      nextTick(() => {
        logContainer.value!.scrollTop = logContainer.value!.scrollHeight
      })
    }
  }, 2000)
})

onUnmounted(() => {
  if (logInterval) {
    clearInterval(logInterval)
  }
})

function formatTimestamp(timestamp: string): string {
  try {
    const date = new Date(timestamp)
    return date.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  } catch {
    return timestamp
  }
}

function getLevelType(level: string): string {
  switch (level) {
    case 'error': return 'danger'
    case 'warn': return 'warning'
    case 'info': return 'success'
    case 'debug': return 'info'
    default: return 'info'
  }
}

function getLevelText(level: string): string {
  switch (level) {
    case 'error': return '错误'
    case 'warn': return '警告'
    case 'info': return '信息'
    case 'debug': return '调试'
    default: return level
  }
}

async function clearLogs() {
  try {
    await ElMessageBox.confirm(
      '确定要清空所有日志吗？此操作不可撤销。',
      '清空日志',
      { confirmButtonText: '清空', cancelButtonText: '取消', type: 'warning' }
    )
    await store.clearAllLogs()
    ElMessage.success('日志已清空')
  } catch {
    // User cancelled
  }
}

function exportLogs() {
  const data = filteredLogs.value.map(log => ({
    timestamp: log.timestamp,
    level: log.level,
    rule: log.ruleName || '',
    message: log.message,
    details: log.details || ''
  }))

  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `pfm-logs-${new Date().toISOString().slice(0, 10)}.json`
  a.click()
  URL.revokeObjectURL(url)
  ElMessage.success('日志已导出')
}

function scrollToBottom() {
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
}
</script>

<template>
  <div class="logs-page">
    <!-- Toolbar -->
    <el-card class="toolbar-card">
      <div class="toolbar">
        <div class="toolbar-left">
          <el-select v-model="filterLevel" placeholder="日志级别" style="width: 120px">
            <el-option label="全部级别" value="all" />
            <el-option label="调试" value="debug" />
            <el-option label="信息" value="info" />
            <el-option label="警告" value="warn" />
            <el-option label="错误" value="error" />
          </el-select>

          <el-select v-model="filterRule" placeholder="选择规则" clearable style="width: 160px">
            <el-option label="全部规则" value="" />
            <el-option
              v-for="rule in logRules"
              :key="rule.id"
              :label="rule.name"
              :value="rule.id"
            />
          </el-select>

          <el-input
            v-model="searchText"
            placeholder="搜索日志..."
            clearable
            style="width: 200px"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
        </div>

        <div class="toolbar-right">
          <el-tag type="info" size="small">调试: {{ logCounts.debug }}</el-tag>
          <el-tag type="success" size="small">信息: {{ logCounts.info }}</el-tag>
          <el-tag type="warning" size="small">警告: {{ logCounts.warn }}</el-tag>
          <el-tag type="danger" size="small">错误: {{ logCounts.error }}</el-tag>

          <el-divider direction="vertical" />

          <el-checkbox v-model="autoScroll">自动滚动</el-checkbox>

          <el-button size="small" @click="scrollToBottom">
            <el-icon><Bottom /></el-icon>
            滚动到底部
          </el-button>

          <el-button size="small" @click="exportLogs">
            <el-icon><Download /></el-icon>
            导出
          </el-button>

          <el-button size="small" type="danger" @click="clearLogs">
            <el-icon><Delete /></el-icon>
            清空
          </el-button>
        </div>
      </div>
    </el-card>

    <!-- Log List -->
    <el-card class="logs-card">
      <div ref="logContainer" class="logs-container">
        <div
          v-for="log in filteredLogs"
          :key="log.id"
          class="log-entry"
          :class="log.level"
        >
          <span class="log-time">{{ formatTimestamp(log.timestamp) }}</span>
          <el-tag
            :type="getLevelType(log.level)"
            size="small"
            class="log-level"
          >
            {{ getLevelText(log.level) }}
          </el-tag>
          <span v-if="log.ruleName" class="log-rule">
            [{{ log.ruleName }}]
          </span>
          <span class="log-message">{{ log.message }}</span>
          <span v-if="log.details" class="log-details">{{ log.details }}</span>
        </div>

        <div v-if="filteredLogs.length === 0" class="empty-logs">
          <el-empty description="暂无日志">
            <template #image>
              <el-icon :size="64" color="#909399"><Document /></el-icon>
            </template>
          </el-empty>
        </div>
      </div>

      <div class="logs-footer">
        <span>共 {{ filteredLogs.length }} 条日志</span>
        <span v-if="filterLevel !== 'all' || filterRule || searchText">
          (已筛选，总计 {{ store.logs.length }} 条)
        </span>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.logs-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
}

.toolbar-card {
  flex-shrink: 0;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.toolbar-left {
  display: flex;
  gap: 12px;
  align-items: center;
}

.toolbar-right {
  display: flex;
  gap: 8px;
  align-items: center;
}

.logs-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.logs-card :deep(.el-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 0;
  min-height: 0;
}

.logs-container {
  flex: 1;
  overflow-y: auto;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 13px;
  background-color: #1e1e1e;
  color: #d4d4d4;
  padding: 12px;
  min-height: 400px;
}

.log-entry {
  padding: 4px 8px;
  border-radius: 4px;
  margin-bottom: 4px;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  line-height: 1.5;
}

.log-entry:hover {
  background-color: rgba(255, 255, 255, 0.05);
}

.log-entry.error {
  background-color: rgba(245, 108, 108, 0.1);
}

.log-entry.warn {
  background-color: rgba(230, 162, 60, 0.1);
}

.log-time {
  color: #808080;
  white-space: nowrap;
  flex-shrink: 0;
}

.log-level {
  flex-shrink: 0;
}

.log-rule {
  color: #569cd6;
  flex-shrink: 0;
}

.log-message {
  color: #d4d4d4;
  flex: 1;
  word-break: break-word;
}

.log-details {
  color: #808080;
  font-size: 12px;
}

.empty-logs {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
  min-height: 300px;
}

.empty-logs :deep(.el-empty__description) {
  color: #909399;
}

.logs-footer {
  padding: 8px 16px;
  border-top: 1px solid #e4e7ed;
  font-size: 12px;
  color: #909399;
  background-color: #fff;
}
</style>
