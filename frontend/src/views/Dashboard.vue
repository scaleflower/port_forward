<script lang="ts" setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useRulesStore } from '../stores/rules'

const router = useRouter()
const store = useRulesStore()

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

function navigateTo(path: string) {
  if (path) router.push(path)
}

async function toggleRule(id: string, running: boolean) {
  if (running) {
    await store.stopRule(id)
  } else {
    await store.startRule(id)
  }
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

    <!-- Recent Rules -->
    <el-card class="rules-card">
      <template #header>
        <div class="card-header">
          <span>最近规则</span>
          <el-button type="primary" size="small" @click="router.push('/forward')">
            添加规则
          </el-button>
        </div>
      </template>
      <el-table :data="store.rules.slice(0, 10)" style="width: 100%">
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="type" label="类型" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.type === 'forward' ? '' : row.type === 'reverse' ? 'success' : 'warning'">
              {{ row.type === 'forward' ? '端口转发' : row.type === 'reverse' ? '反向代理' : '代理链' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="listenAddr" label="监听地址" width="120" />
        <el-table-column label="目标" width="200">
          <template #default="{ row }">
            <span v-if="row.targets && row.targets.length > 0">
              {{ row.targets[0].addr }}
              <span v-if="row.targets.length > 1">+{{ row.targets.length - 1 }}</span>
            </span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 'running' ? 'success' : row.status === 'error' ? 'danger' : 'info'">
              {{ row.status === 'running' ? '运行中' : row.status === 'error' ? '错误' : '已停止' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120">
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
      <div v-if="store.rules.length === 0" class="empty-state">
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

.rules-card {
  flex: 1;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.empty-state {
  padding: 40px 0;
}
</style>
