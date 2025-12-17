<script lang="ts" setup>
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRulesStore } from '../stores/rules'
import type { Rule, Environment } from '../types'

const store = useRulesStore()

const dialogVisible = ref(false)
const isEdit = ref(false)
const currentRule = ref<Rule | null>(null)

const form = ref({
  environment: 'CUSTOM' as Environment,
  name: '',
  targetHost: '',
  targetPort: 0,
  localPort: 0,
  protocol: 'tcp' as const,
  remark: ''
})

const rules = computed(() => store.forwardRules)

// Environment options
const environmentOptions = [
  { value: 'TRUNK', label: 'TRUNK' },
  { value: 'PRE-PROD', label: 'PRE-PROD' },
  { value: 'PRODUCTION', label: 'PRODUCTION' },
  { value: 'CUSTOM', label: 'CUSTOM' }
]

// Environment tag type mapping
function getEnvTagType(env: string): '' | 'success' | 'warning' | 'danger' | 'info' {
  switch (env) {
    case 'TRUNK':
      return 'info'
    case 'PRE-PROD':
      return 'warning'
    case 'PRODUCTION':
      return 'danger'
    default:
      return ''
  }
}

function openCreateDialog() {
  isEdit.value = false
  currentRule.value = null
  form.value = {
    environment: 'CUSTOM',
    name: '',
    targetHost: '',
    targetPort: 0,
    localPort: 0,
    protocol: 'tcp',
    remark: ''
  }
  dialogVisible.value = true
}

function openEditDialog(rule: Rule) {
  isEdit.value = true
  currentRule.value = rule
  form.value = {
    environment: rule.environment || 'CUSTOM',
    name: rule.name,
    targetHost: rule.targetHost || '',
    targetPort: rule.targetPort || 0,
    localPort: rule.localPort || 0,
    protocol: rule.protocol as any,
    remark: rule.remark || ''
  }
  dialogVisible.value = true
}

async function saveRule() {
  if (!form.value.name) {
    ElMessage.error('Please enter the purpose/name')
    return
  }
  if (!form.value.targetHost) {
    ElMessage.error('Please enter the target IP/host')
    return
  }
  if (form.value.targetPort <= 0 || form.value.targetPort > 65535) {
    ElMessage.error('Please enter a valid target port (1-65535)')
    return
  }
  if (form.value.localPort <= 0 || form.value.localPort > 65535) {
    ElMessage.error('Please enter a valid local port (1-65535)')
    return
  }

  try {
    if (isEdit.value && currentRule.value) {
      const updatedRule: Rule = {
        ...currentRule.value,
        environment: form.value.environment,
        name: form.value.name,
        targetHost: form.value.targetHost,
        targetPort: form.value.targetPort,
        localPort: form.value.localPort,
        protocol: form.value.protocol,
        remark: form.value.remark
      }
      await store.updateRule(updatedRule)
      ElMessage.success('Rule updated')
    } else {
      const newRule = await store.createNewRule(form.value.name, 'forward')
      if (newRule) {
        newRule.environment = form.value.environment
        newRule.targetHost = form.value.targetHost
        newRule.targetPort = form.value.targetPort
        newRule.localPort = form.value.localPort
        newRule.protocol = form.value.protocol
        newRule.remark = form.value.remark
        await store.createRule(newRule)
        ElMessage.success('Rule created')
      }
    }
    dialogVisible.value = false
  } catch (e: any) {
    ElMessage.error(e.message || 'Operation failed')
  }
}

async function deleteRule(rule: Rule) {
  try {
    await ElMessageBox.confirm(`Delete rule "${rule.name}"?`, 'Confirm Delete', {
      type: 'warning'
    })
    await store.deleteRule(rule.id)
    ElMessage.success('Rule deleted')
  } catch (e) {
    // User cancelled
  }
}

async function toggleRule(rule: Rule) {
  try {
    if (rule.status === 'running') {
      await store.stopRule(rule.id)
      ElMessage.success('Rule stopped')
    } else {
      await store.startRule(rule.id)
      ElMessage.success('Rule started')
    }
  } catch (e: any) {
    ElMessage.error(e.message || 'Operation failed')
  }
}
</script>

<template>
  <div class="port-forward">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>Port Forward Rules</span>
          <el-button type="primary" @click="openCreateDialog">
            <el-icon><Plus /></el-icon>
            Add Rule
          </el-button>
        </div>
      </template>

      <el-table :data="rules" style="width: 100%" v-loading="store.loading">
        <el-table-column prop="environment" label="Environment" width="120">
          <template #default="{ row }">
            <el-tag :type="getEnvTagType(row.environment)" size="small">
              {{ row.environment || 'CUSTOM' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="Purpose" min-width="150" />
        <el-table-column label="Target" min-width="180">
          <template #default="{ row }">
            <span>{{ row.targetHost }}:{{ row.targetPort }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="localPort" label="Local Port" width="100">
          <template #default="{ row }">
            <el-tag size="small" type="info">:{{ row.localPort }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="protocol" label="Protocol" width="80">
          <template #default="{ row }">
            <el-tag size="small">{{ row.protocol?.toUpperCase() }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="Status" width="100">
          <template #default="{ row }">
            <el-tag
              :type="row.status === 'running' ? 'success' : row.status === 'error' ? 'danger' : 'info'"
            >
              {{ row.status === 'running' ? 'Running' : row.status === 'error' ? 'Error' : 'Stopped' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="Remark" min-width="120" show-overflow-tooltip />
        <el-table-column label="Actions" width="200" fixed="right">
          <template #default="{ row }">
            <el-button-group>
              <el-button
                size="small"
                :type="row.status === 'running' ? 'danger' : 'success'"
                @click="toggleRule(row)"
              >
                {{ row.status === 'running' ? 'Stop' : 'Start' }}
              </el-button>
              <el-button size="small" @click="openEditDialog(row)">Edit</el-button>
              <el-button size="small" type="danger" @click="deleteRule(row)">Delete</el-button>
            </el-button-group>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="rules.length === 0 && !store.loading" description="No port forward rules">
        <el-button type="primary" @click="openCreateDialog">Create First Rule</el-button>
      </el-empty>
    </el-card>

    <!-- Rule Dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? 'Edit Rule' : 'Add Rule'"
      width="550px"
    >
      <el-form :model="form" label-width="110px">
        <el-form-item label="Environment" required>
          <el-select v-model="form.environment" style="width: 100%">
            <el-option
              v-for="opt in environmentOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="Purpose" required>
          <el-input v-model="form.name" placeholder="e.g., MySQL Database" />
        </el-form-item>
        <el-form-item label="Target Host" required>
          <el-input v-model="form.targetHost" placeholder="IP or hostname, e.g., 192.168.1.100" />
        </el-form-item>
        <el-form-item label="Target Port" required>
          <el-input-number
            v-model="form.targetPort"
            :min="1"
            :max="65535"
            style="width: 100%"
            placeholder="e.g., 3306"
          />
        </el-form-item>
        <el-form-item label="Local Port" required>
          <el-input-number
            v-model="form.localPort"
            :min="1"
            :max="65535"
            style="width: 100%"
            placeholder="e.g., 13306"
          />
          <div class="form-tip">The local port to listen on. Access via localhost:{{ form.localPort || 'PORT' }}</div>
        </el-form-item>
        <el-form-item label="Protocol">
          <el-select v-model="form.protocol" style="width: 100%">
            <el-option label="TCP" value="tcp" />
            <el-option label="UDP" value="udp" />
          </el-select>
        </el-form-item>
        <el-form-item label="Remark">
          <el-input v-model="form.remark" type="textarea" placeholder="Optional notes" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="saveRule">Save</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
}
</style>
