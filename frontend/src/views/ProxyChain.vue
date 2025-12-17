<script lang="ts" setup>
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRulesStore } from '../stores/rules'
import type { Rule, Chain, Hop, Environment } from '../types'

const store = useRulesStore()

// Chain dialog
const chainDialogVisible = ref(false)
const isChainEdit = ref(false)
const currentChain = ref<Chain | null>(null)

const chainForm = ref({
  name: '',
  description: '',
  hops: [{ name: '', addr: '', protocol: 'socks5', auth: undefined }] as Hop[]
})

// Rule dialog
const ruleDialogVisible = ref(false)
const isRuleEdit = ref(false)
const currentRule = ref<Rule | null>(null)

const ruleForm = ref({
  environment: 'CUSTOM' as Environment,
  name: '',
  localPort: 0,
  protocol: 'tcp' as const,
  targetHost: '',
  targetPort: 0,
  chainId: '',
  description: '',
  remark: ''
})

const chains = computed(() => store.chains)
const chainRules = computed(() => store.chainRules)

// Chain operations
function openCreateChainDialog() {
  isChainEdit.value = false
  currentChain.value = null
  chainForm.value = {
    name: '',
    description: '',
    hops: [{ name: '', addr: '', protocol: 'socks5', auth: undefined }]
  }
  chainDialogVisible.value = true
}

function openEditChainDialog(chain: Chain) {
  isChainEdit.value = true
  currentChain.value = chain
  chainForm.value = {
    name: chain.name,
    description: chain.description || '',
    hops: chain.hops.length > 0 ? [...chain.hops] : [{ name: '', addr: '', protocol: 'socks5', auth: undefined }]
  }
  chainDialogVisible.value = true
}

function addHop() {
  chainForm.value.hops.push({ name: '', addr: '', protocol: 'socks5', auth: undefined })
}

function removeHop(index: number) {
  if (chainForm.value.hops.length > 1) {
    chainForm.value.hops.splice(index, 1)
  }
}

async function saveChain() {
  if (!chainForm.value.name) {
    ElMessage.error('Please fill in the chain name')
    return
  }

  const validHops = chainForm.value.hops.filter(h => h.addr.trim() !== '')
  if (validHops.length === 0) {
    ElMessage.error('Please add at least one hop')
    return
  }

  // Add default names to hops
  validHops.forEach((hop, i) => {
    if (!hop.name) {
      hop.name = `hop-${i + 1}`
    }
  })

  try {
    if (isChainEdit.value && currentChain.value) {
      const updatedChain: Chain = {
        ...currentChain.value,
        name: chainForm.value.name,
        description: chainForm.value.description,
        hops: validHops
      }
      await store.updateChain(updatedChain)
      ElMessage.success('Chain updated')
    } else {
      const newChain: Chain = {
        id: '',
        name: chainForm.value.name,
        description: chainForm.value.description,
        hops: validHops,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString()
      }
      await store.createChain(newChain)
      ElMessage.success('Chain created')
    }
    chainDialogVisible.value = false
  } catch (e: any) {
    ElMessage.error(e.message || 'Operation failed')
  }
}

async function deleteChain(chain: Chain) {
  try {
    await ElMessageBox.confirm(`Delete chain "${chain.name}"?`, 'Confirm', {
      type: 'warning'
    })
    await store.deleteChain(chain.id)
    ElMessage.success('Chain deleted')
  } catch (e) {
    // User cancelled
  }
}

// Rule operations
function openCreateRuleDialog() {
  if (chains.value.length === 0) {
    ElMessage.warning('Please create a chain first')
    return
  }
  isRuleEdit.value = false
  currentRule.value = null
  ruleForm.value = {
    environment: 'CUSTOM',
    name: '',
    localPort: 0,
    protocol: 'tcp',
    targetHost: '',
    targetPort: 0,
    chainId: chains.value[0]?.id || '',
    description: '',
    remark: ''
  }
  ruleDialogVisible.value = true
}

function openEditRuleDialog(rule: Rule) {
  isRuleEdit.value = true
  currentRule.value = rule
  ruleForm.value = {
    environment: rule.environment || 'CUSTOM',
    name: rule.name,
    localPort: rule.localPort || 0,
    protocol: rule.protocol as any,
    targetHost: rule.targetHost || '',
    targetPort: rule.targetPort || 0,
    chainId: rule.chainId || '',
    description: rule.description || '',
    remark: rule.remark || ''
  }
  ruleDialogVisible.value = true
}

async function saveRule() {
  if (!ruleForm.value.name) {
    ElMessage.error('Please fill in the rule name')
    return
  }
  if (ruleForm.value.localPort <= 0 || ruleForm.value.localPort > 65535) {
    ElMessage.error('Please enter a valid local port (1-65535)')
    return
  }
  if (!ruleForm.value.targetHost) {
    ElMessage.error('Please enter the target host')
    return
  }
  if (ruleForm.value.targetPort <= 0 || ruleForm.value.targetPort > 65535) {
    ElMessage.error('Please enter a valid target port (1-65535)')
    return
  }
  if (!ruleForm.value.chainId) {
    ElMessage.error('Please select a chain')
    return
  }

  const selectedChain = chains.value.find(c => c.id === ruleForm.value.chainId)

  try {
    if (isRuleEdit.value && currentRule.value) {
      const updatedRule: Rule = {
        ...currentRule.value,
        environment: ruleForm.value.environment,
        name: ruleForm.value.name,
        localPort: ruleForm.value.localPort,
        protocol: ruleForm.value.protocol,
        targetHost: ruleForm.value.targetHost,
        targetPort: ruleForm.value.targetPort,
        chainId: selectedChain?.id,
        description: ruleForm.value.description,
        remark: ruleForm.value.remark
      }
      await store.updateRule(updatedRule)
      ElMessage.success('Rule updated')
    } else {
      const newRule = await store.createNewRule(ruleForm.value.name, 'chain')
      if (newRule) {
        newRule.environment = ruleForm.value.environment
        newRule.localPort = ruleForm.value.localPort
        newRule.protocol = ruleForm.value.protocol
        newRule.targetHost = ruleForm.value.targetHost
        newRule.targetPort = ruleForm.value.targetPort
        newRule.chainId = selectedChain?.id
        newRule.description = ruleForm.value.description
        newRule.remark = ruleForm.value.remark
        await store.createRule(newRule)
        ElMessage.success('Rule created')
      }
    }
    ruleDialogVisible.value = false
  } catch (e: any) {
    ElMessage.error(e.message || 'Operation failed')
  }
}

async function deleteRule(rule: Rule) {
  try {
    await ElMessageBox.confirm(`Delete rule "${rule.name}"?`, 'Confirm', {
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

function getProtocolLabel(protocol: string): string {
  const labels: Record<string, string> = {
    socks5: 'SOCKS5',
    http: 'HTTP',
    https: 'HTTPS',
    ss: 'Shadowsocks',
    sni: 'SNI'
  }
  return labels[protocol] || protocol.toUpperCase()
}

function getChainName(chainId?: string): string {
  if (!chainId) return ''
  const chain = chains.value.find(c => c.id === chainId)
  return chain?.name || ''
}

// Environment options
const environmentOptions = [
  { value: 'TRUNK', label: 'TRUNK' },
  { value: 'PRE-PROD', label: 'PRE-PROD' },
  { value: 'PRODUCTION', label: 'PRODUCTION' },
  { value: 'CUSTOM', label: 'CUSTOM' }
]

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
</script>

<template>
  <div class="proxy-chain">
    <!-- Chains Section -->
    <el-card class="chains-card">
      <template #header>
        <div class="card-header">
          <span>Proxy Chains</span>
          <el-button type="primary" @click="openCreateChainDialog">
            <el-icon><Plus /></el-icon>
            Add Chain
          </el-button>
        </div>
      </template>

      <div v-if="chains.length === 0 && !store.loading" class="empty-chains">
        <el-empty description="No proxy chains">
          <el-button type="primary" @click="openCreateChainDialog">Create First Chain</el-button>
        </el-empty>
      </div>

      <div v-else class="chains-grid">
        <el-card
          v-for="chain in chains"
          :key="chain.id"
          class="chain-card"
          shadow="hover"
        >
          <template #header>
            <div class="chain-header">
              <span class="chain-name">{{ chain.name }}</span>
              <el-dropdown trigger="click">
                <el-button text>
                  <el-icon><MoreFilled /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item @click="openEditChainDialog(chain)">
                      <el-icon><Edit /></el-icon> Edit
                    </el-dropdown-item>
                    <el-dropdown-item @click="deleteChain(chain)" divided>
                      <el-icon><Delete /></el-icon> Delete
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </template>

          <div class="chain-hops">
            <div v-for="(hop, index) in chain.hops" :key="index" class="hop-item">
              <div class="hop-info">
                <el-tag size="small" :type="hop.protocol === 'socks5' ? '' : 'success'">
                  {{ getProtocolLabel(hop.protocol) }}
                </el-tag>
                <span class="hop-addr">{{ hop.addr }}</span>
              </div>
              <el-icon v-if="index < chain.hops.length - 1" class="hop-arrow"><ArrowDown /></el-icon>
            </div>
          </div>

          <div v-if="chain.description" class="chain-desc">
            {{ chain.description }}
          </div>
        </el-card>
      </div>
    </el-card>

    <!-- Chain Rules Section -->
    <el-card class="rules-card">
      <template #header>
        <div class="card-header">
          <span>Chain Rules</span>
          <el-button type="primary" @click="openCreateRuleDialog">
            <el-icon><Plus /></el-icon>
            Add Rule
          </el-button>
        </div>
      </template>

      <el-table :data="chainRules" style="width: 100%" v-loading="store.loading">
        <el-table-column prop="environment" label="Env" width="100">
          <template #default="{ row }">
            <el-tag :type="getEnvTagType(row.environment)" size="small">
              {{ row.environment || 'CUSTOM' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="Name" min-width="150" />
        <el-table-column prop="protocol" label="Protocol" width="80">
          <template #default="{ row }">
            <el-tag size="small">{{ row.protocol?.toUpperCase() }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="localPort" label="Local Port" width="100">
          <template #default="{ row }">
            <el-tag size="small" type="info">:{{ row.localPort }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Target" min-width="180">
          <template #default="{ row }">
            <span>{{ row.targetHost }}:{{ row.targetPort }}</span>
          </template>
        </el-table-column>
        <el-table-column label="Chain" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.chainId" type="warning" size="small">
              {{ getChainName(row.chainId) }}
            </el-tag>
            <span v-else>-</span>
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
              <el-button size="small" @click="openEditRuleDialog(row)">Edit</el-button>
              <el-button size="small" type="danger" @click="deleteRule(row)">Delete</el-button>
            </el-button-group>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="chainRules.length === 0 && !store.loading" description="No chain rules">
        <el-button type="primary" @click="openCreateRuleDialog">Create First Rule</el-button>
      </el-empty>
    </el-card>

    <!-- Chain Dialog -->
    <el-dialog
      v-model="chainDialogVisible"
      :title="isChainEdit ? 'Edit Chain' : 'Add Chain'"
      width="600px"
    >
      <el-form :model="chainForm" label-width="100px">
        <el-form-item label="Name" required>
          <el-input v-model="chainForm.name" placeholder="Chain name" />
        </el-form-item>
        <el-form-item label="Hops" required>
          <div v-for="(hop, index) in chainForm.hops" :key="index" class="hop-row">
            <div class="hop-number">{{ index + 1 }}</div>
            <el-select v-model="hop.protocol" style="width: 120px">
              <el-option label="SOCKS5" value="socks5" />
              <el-option label="HTTP" value="http" />
              <el-option label="HTTPS" value="https" />
              <el-option label="Shadowsocks" value="ss" />
            </el-select>
            <el-input
              v-model="hop.addr"
              placeholder="127.0.0.1:1080"
              style="flex: 1; margin: 0 8px"
            />
            <el-button
              v-if="chainForm.hops.length > 1"
              type="danger"
              circle
              @click="removeHop(index)"
            >
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
          <el-button type="primary" link @click="addHop" style="margin-top: 8px">
            <el-icon><Plus /></el-icon> Add Hop
          </el-button>
        </el-form-item>
        <el-form-item label="Description">
          <el-input v-model="chainForm.description" type="textarea" placeholder="Optional" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="chainDialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="saveChain">Save</el-button>
      </template>
    </el-dialog>

    <!-- Rule Dialog -->
    <el-dialog
      v-model="ruleDialogVisible"
      :title="isRuleEdit ? 'Edit Rule' : 'Add Rule'"
      width="550px"
    >
      <el-form :model="ruleForm" label-width="110px">
        <el-form-item label="Environment" required>
          <el-select v-model="ruleForm.environment" style="width: 100%">
            <el-option
              v-for="opt in environmentOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="Name" required>
          <el-input v-model="ruleForm.name" placeholder="Rule name" />
        </el-form-item>
        <el-form-item label="Protocol" required>
          <el-select v-model="ruleForm.protocol" style="width: 100%">
            <el-option label="TCP" value="tcp" />
            <el-option label="UDP" value="udp" />
          </el-select>
        </el-form-item>
        <el-form-item label="Local Port" required>
          <el-input-number
            v-model="ruleForm.localPort"
            :min="1"
            :max="65535"
            style="width: 100%"
            placeholder="e.g., 8080"
          />
        </el-form-item>
        <el-form-item label="Target Host" required>
          <el-input v-model="ruleForm.targetHost" placeholder="IP or hostname" />
        </el-form-item>
        <el-form-item label="Target Port" required>
          <el-input-number
            v-model="ruleForm.targetPort"
            :min="1"
            :max="65535"
            style="width: 100%"
            placeholder="e.g., 80"
          />
        </el-form-item>
        <el-form-item label="Chain" required>
          <el-select v-model="ruleForm.chainId" style="width: 100%" placeholder="Select chain">
            <el-option
              v-for="chain in chains"
              :key="chain.id"
              :label="chain.name"
              :value="chain.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="Remark">
          <el-input v-model="ruleForm.remark" type="textarea" placeholder="Optional notes" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="saveRule">Save</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.proxy-chain {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.chains-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.chain-card {
  transition: transform 0.2s;
}

.chain-card:hover {
  transform: translateY(-2px);
}

.chain-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.chain-name {
  font-weight: 600;
}

.chain-hops {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.hop-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
}

.hop-info {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 4px;
  width: 100%;
}

.hop-addr {
  font-family: monospace;
  font-size: 13px;
}

.hop-arrow {
  color: #909399;
  margin: 4px 0;
}

.chain-desc {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #ebeef5;
  color: #909399;
  font-size: 13px;
}

.hop-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.hop-number {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: #409eff;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
}

.empty-chains {
  padding: 40px 0;
}
</style>
