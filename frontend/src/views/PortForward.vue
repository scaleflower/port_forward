<script lang="ts" setup>
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRulesStore } from '../stores/rules'
import type { Rule } from '../types'

const store = useRulesStore()

const dialogVisible = ref(false)
const isEdit = ref(false)
const currentRule = ref<Rule | null>(null)

const form = ref({
  name: '',
  targetHost: '',
  targetPort: 0,
  localPort: 0,
  protocol: 'tcp' as const,
  remark: ''
})

const rules = computed(() => store.forwardRules)

function openCreateDialog() {
  isEdit.value = false
  currentRule.value = null
  form.value = {
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
    ElMessage.error('请输入用途/名称')
    return
  }
  if (!form.value.targetHost) {
    ElMessage.error('请输入目标 IP/主机')
    return
  }
  if (form.value.targetPort <= 0 || form.value.targetPort > 65535) {
    ElMessage.error('请输入有效的目标端口 (1-65535)')
    return
  }
  if (form.value.localPort <= 0 || form.value.localPort > 65535) {
    ElMessage.error('请输入有效的本地端口 (1-65535)')
    return
  }

  try {
    if (isEdit.value && currentRule.value) {
      const updatedRule: Rule = {
        ...currentRule.value,
        name: form.value.name,
        targetHost: form.value.targetHost,
        targetPort: form.value.targetPort,
        localPort: form.value.localPort,
        protocol: form.value.protocol,
        remark: form.value.remark
      }
      await store.updateRule(updatedRule)
      ElMessage.success('规则已更新')
    } else {
      const newRule = await store.createNewRule(form.value.name, 'forward')
      if (newRule) {
        newRule.targetHost = form.value.targetHost
        newRule.targetPort = form.value.targetPort
        newRule.localPort = form.value.localPort
        newRule.protocol = form.value.protocol
        newRule.remark = form.value.remark
        await store.createRule(newRule)
        ElMessage.success('规则已创建')
      }
    }
    dialogVisible.value = false
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

async function deleteCurrentRule() {
  if (!currentRule.value) return

  try {
    await ElMessageBox.confirm(
      `确定要删除规则 "${currentRule.value.name}" 吗？此操作不可撤销。`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )
    await store.deleteRule(currentRule.value.id)
    ElMessage.success('规则已删除')
    dialogVisible.value = false
  } catch (e) {
    // User cancelled
  }
}
</script>

<template>
  <div class="port-forward">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>端口转发规则</span>
          <el-button type="primary" @click="openCreateDialog">
            <el-icon><Plus /></el-icon>
            添加规则
          </el-button>
        </div>
      </template>

      <el-table :data="rules" style="width: 100%" v-loading="store.loading">
        <el-table-column prop="name" label="用途" min-width="140" />
        <el-table-column label="目标" min-width="180">
          <template #default="{ row }">
            <code class="target-addr">{{ row.targetHost }}:{{ row.targetPort }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="localPort" label="本地端口" width="100" align="center">
          <template #default="{ row }">
            <el-tag size="small" type="info">:{{ row.localPort }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="protocol" label="协议" width="80" align="center">
          <template #default="{ row }">
            <el-tag size="small">{{ row.protocol?.toUpperCase() }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="85" align="center">
          <template #default="{ row }">
            <el-tag
              size="small"
              :type="row.status === 'running' ? 'success' : row.status === 'error' ? 'danger' : 'info'"
            >
              {{ row.status === 'running' ? '运行中' : row.status === 'error' ? '错误' : '已停止' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="100" show-overflow-tooltip />
        <el-table-column label="操作" width="80" fixed="right" align="center">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="openEditDialog(row)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="rules.length === 0 && !store.loading" description="暂无端口转发规则">
        <el-button type="primary" @click="openCreateDialog">创建第一个规则</el-button>
      </el-empty>
    </el-card>

    <!-- Rule Dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑规则' : '添加规则'"
      width="550px"
    >
      <el-form :model="form" label-width="100px">
        <el-form-item label="用途" required>
          <el-input v-model="form.name" placeholder="例如：MySQL 数据库" />
        </el-form-item>
        <el-form-item label="目标主机" required>
          <el-input v-model="form.targetHost" placeholder="IP 或主机名，例如：192.168.1.100" />
        </el-form-item>
        <el-form-item label="目标端口" required>
          <el-input-number
            v-model="form.targetPort"
            :min="1"
            :max="65535"
            style="width: 100%"
            placeholder="例如：3306"
          />
        </el-form-item>
        <el-form-item label="本地端口" required>
          <el-input-number
            v-model="form.localPort"
            :min="1"
            :max="65535"
            style="width: 100%"
            placeholder="例如：13306"
          />
          <div class="form-tip">本地监听端口，通过 localhost:{{ form.localPort || 'PORT' }} 访问</div>
        </el-form-item>
        <el-form-item label="协议">
          <el-select v-model="form.protocol" style="width: 100%">
            <el-option label="TCP" value="tcp" />
            <el-option label="UDP" value="udp" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" placeholder="可选备注信息" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <div class="footer-left">
            <el-button
              v-if="isEdit"
              type="danger"
              plain
              @click="deleteCurrentRule"
            >
              <el-icon><Delete /></el-icon>
              删除规则
            </el-button>
          </div>
          <div class="footer-right">
            <el-button @click="dialogVisible = false">取消</el-button>
            <el-button type="primary" @click="saveRule">保存</el-button>
          </div>
        </div>
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

.target-addr {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 13px;
  color: #606266;
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 3px;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
}

.dialog-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.footer-left {
  flex: 1;
}

.footer-right {
  display: flex;
  gap: 8px;
}
</style>
