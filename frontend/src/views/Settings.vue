<script lang="ts" setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRulesStore } from '../stores/rules'
import { ExportData, ImportData, ClearAllData } from '../../wailsjs/go/main/App'
import { OpenFileDialog } from '../../wailsjs/go/main/App'

const store = useRulesStore()
const importDialogVisible = ref(false)
const importMode = ref<'replace' | 'merge'>('replace')
const importData = ref('')
const dataLoading = ref(false)

const config = ref({
  logLevel: 'info',
  autoStart: false,
  trayEnabled: true,
  hotkeyEnabled: true,
  hotkeyModifiers: 'cmd+shift',
  hotkeyKey: 'p'
})

const serviceLoading = ref(false)
const systemInfo = ref({ os: '', arch: '', version: '' })

// Service status polling interval
let statusInterval: number | null = null

onMounted(async () => {
  await loadConfig()
  await loadSystemInfo()
  // Refresh service status immediately
  await store.fetchServiceStatus()
  // Poll service status every 3 seconds
  statusInterval = window.setInterval(() => {
    store.fetchServiceStatus()
  }, 3000)
})

onUnmounted(() => {
  if (statusInterval) {
    clearInterval(statusInterval)
    statusInterval = null
  }
})

async function loadSystemInfo() {
  try {
    const { GetSystemInfo } = await import('../../wailsjs/go/main/App')
    const info = await GetSystemInfo()
    systemInfo.value = {
      os: info['os'] || '',
      arch: info['arch'] || '',
      version: info['version'] || ''
    }
  } catch (e) {
    console.error('Failed to load system info:', e)
  }
}

function getPlatformDisplay(): string {
  const os = systemInfo.value.os
  const arch = systemInfo.value.arch
  if (!os) return 'Loading...'

  const osNames: Record<string, string> = {
    darwin: 'macOS',
    windows: 'Windows',
    linux: 'Linux'
  }
  const archNames: Record<string, string> = {
    amd64: 'x64',
    arm64: 'ARM64 (Apple Silicon)',
    '386': 'x86'
  }
  return `${osNames[os] || os} ${archNames[arch] || arch}`
}

function getHotkeyDisplay(): string {
  const mods = config.value.hotkeyModifiers
  const key = config.value.hotkeyKey.toUpperCase()
  const isMac = systemInfo.value.os === 'darwin'

  const modMap: Record<string, string> = {
    'cmd+shift': isMac ? '⌘ + ⇧' : 'Ctrl + Shift',
    'ctrl+shift': 'Ctrl + ⇧',
    'cmd+alt': isMac ? '⌘ + ⌥' : 'Ctrl + Alt',
    'ctrl+alt': 'Ctrl + Alt'
  }

  return `${modMap[mods] || mods.toUpperCase()} + ${key}`
}

async function loadConfig() {
  try {
    const appConfig = store.getConfig()
    if (appConfig) {
      config.value.logLevel = appConfig.logLevel || 'info'
      config.value.autoStart = appConfig.autoStart || false
      config.value.trayEnabled = appConfig.trayEnabled !== false
      config.value.hotkeyEnabled = appConfig.hotkeyEnabled !== false
      config.value.hotkeyModifiers = appConfig.hotkeyModifiers || 'cmd+shift'
      config.value.hotkeyKey = appConfig.hotkeyKey || 'p'
    }
  } catch (e) {
    console.error('Failed to load config:', e)
  }
}

async function saveConfig() {
  try {
    const currentConfig = store.getConfig()
    await store.saveConfig({
      ...currentConfig,
      logLevel: config.value.logLevel,
      autoStart: config.value.autoStart,
      trayEnabled: config.value.trayEnabled,
      hotkeyEnabled: config.value.hotkeyEnabled,
      hotkeyModifiers: config.value.hotkeyModifiers,
      hotkeyKey: config.value.hotkeyKey
    } as any)
    ElMessage.success('Settings saved. Restart app for changes to take effect.')
  } catch (e: any) {
    ElMessage.error(e.message || 'Failed to save')
  }
}

async function installService() {
  try {
    await ElMessageBox.confirm(
      'Install Port Forward Manager as system service? This requires administrator privileges.',
      'Install Service',
      { type: 'info' }
    )
    serviceLoading.value = true
    await store.installService()
    ElMessage.success('Service installed')
    await store.refreshServiceStatus()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.message || 'Installation failed')
    }
  } finally {
    serviceLoading.value = false
  }
}

async function uninstallService() {
  try {
    await ElMessageBox.confirm(
      'Uninstall the system service? Running rules will be stopped.',
      'Uninstall Service',
      { type: 'warning' }
    )
    serviceLoading.value = true
    await store.uninstallService()
    ElMessage.success('Service uninstalled')
    await store.refreshServiceStatus()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.message || 'Uninstallation failed')
    }
  } finally {
    serviceLoading.value = false
  }
}

function getServiceStatusType(status: string) {
  switch (status) {
    case 'running':
      return 'success'
    case 'stopped':
      return 'info'
    case 'not_installed':
      return 'warning'
    default:
      return 'danger'
  }
}

function getServiceStatusText(status: string) {
  switch (status) {
    case 'running':
      return 'Running'
    case 'stopped':
      return 'Stopped'
    case 'not_installed':
      return 'Not Installed'
    default:
      return 'Unknown'
  }
}

// Data Management Functions
async function exportRules() {
  try {
    dataLoading.value = true
    const data = await ExportData()
    if (!data) {
      ElMessage.warning('No data to export')
      return
    }

    // Create and download file
    const blob = new Blob([data], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    const timestamp = new Date().toISOString().slice(0, 10)
    link.download = `pfm-backup-${timestamp}.json`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)

    ElMessage.success('Data exported successfully')
  } catch (e: any) {
    ElMessage.error(e.message || 'Export failed')
  } finally {
    dataLoading.value = false
  }
}

function openImportDialog() {
  importData.value = ''
  importMode.value = 'replace'
  importDialogVisible.value = true
}

async function handleFileSelect() {
  try {
    const content = await OpenFileDialog()
    if (content) {
      importData.value = content
      ElMessage.success('File loaded')
    }
  } catch (e: any) {
    ElMessage.error(e.message || 'Failed to open file')
  }
}

async function confirmImport() {
  if (!importData.value.trim()) {
    ElMessage.warning('Please select or paste import data')
    return
  }

  // Validate JSON
  try {
    JSON.parse(importData.value)
  } catch {
    ElMessage.error('Invalid JSON format')
    return
  }

  try {
    dataLoading.value = true
    const merge = importMode.value === 'merge'
    await ImportData(importData.value, merge)
    await store.init() // Refresh all data
    importDialogVisible.value = false
    ElMessage.success(merge ? 'Data merged successfully' : 'Data imported successfully')
  } catch (e: any) {
    ElMessage.error(e.message || 'Import failed')
  } finally {
    dataLoading.value = false
  }
}

async function clearAllData() {
  try {
    await ElMessageBox.confirm(
      'This will permanently delete ALL rules and chains. This action cannot be undone!',
      'Clear All Data',
      {
        type: 'error',
        confirmButtonText: 'Delete All',
        confirmButtonClass: 'el-button--danger'
      }
    )

    // Double confirmation
    await ElMessageBox.prompt(
      'Type "DELETE" to confirm:',
      'Final Confirmation',
      {
        confirmButtonText: 'Confirm',
        cancelButtonText: 'Cancel',
        inputPattern: /^DELETE$/,
        inputErrorMessage: 'Please type DELETE exactly'
      }
    )

    dataLoading.value = true
    await ClearAllData()
    await store.init() // Refresh all data
    ElMessage.success('All data has been cleared')
  } catch (e: any) {
    if (e !== 'cancel' && e?.message !== 'cancel') {
      ElMessage.error(e.message || 'Failed to clear data')
    }
  } finally {
    dataLoading.value = false
  }
}
</script>

<template>
  <div class="settings">
    <!-- Service Management -->
    <el-card class="settings-card">
      <template #header>
        <div class="card-header">
          <span>Service Management</span>
          <el-tag :type="getServiceStatusType(store.serviceStatus)" size="small">
            {{ getServiceStatusText(store.serviceStatus) }}
          </el-tag>
        </div>
      </template>

      <div class="service-section">
        <el-alert
          type="info"
          :closable="false"
          show-icon
          style="margin-bottom: 20px"
        >
          <template #title>
            Installing as a system service allows port forwarding to run in the background
            and start automatically on boot. The service is self-managing and will auto-restart if needed.
          </template>
        </el-alert>

        <div class="service-actions">
          <template v-if="store.serviceStatus === 'not_installed'">
            <el-button
              type="primary"
              :loading="serviceLoading"
              @click="installService"
            >
              <el-icon><Download /></el-icon>
              Install Service
            </el-button>
          </template>

          <template v-else>
            <el-button
              type="danger"
              plain
              :loading="serviceLoading"
              @click="uninstallService"
            >
              <el-icon><Delete /></el-icon>
              Uninstall Service
            </el-button>
          </template>
        </div>

        <el-descriptions :column="2" border style="margin-top: 20px">
          <el-descriptions-item label="Service Name">PortForwardManager</el-descriptions-item>
          <el-descriptions-item label="Status">
            <el-tag :type="getServiceStatusType(store.serviceStatus)" size="small">
              {{ getServiceStatusText(store.serviceStatus) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Running Mode">
            {{ store.serviceStatus === 'running' ? 'Service' : 'Embedded' }}
          </el-descriptions-item>
          <el-descriptions-item label="Active Rules">
            {{ store.runningRules.length }} / {{ store.rules.length }}
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-card>

    <!-- General Settings -->
    <el-card class="settings-card">
      <template #header>
        <span>General Settings</span>
      </template>

      <el-form :model="config" label-width="140px" style="max-width: 600px">
        <el-form-item label="Log Level">
          <el-select v-model="config.logLevel" style="width: 200px">
            <el-option label="Debug" value="debug" />
            <el-option label="Info" value="info" />
            <el-option label="Warning" value="warn" />
            <el-option label="Error" value="error" />
          </el-select>
        </el-form-item>

        <el-form-item label="Auto Start">
          <el-switch v-model="config.autoStart" />
          <span class="form-tip">Start enabled rules automatically when service starts</span>
        </el-form-item>

        <el-divider content-position="left">Quick Access</el-divider>

        <el-form-item label="System Tray">
          <el-switch v-model="config.trayEnabled" />
          <span class="form-tip">Show icon in system tray / menu bar</span>
        </el-form-item>

        <el-form-item label="Global Hotkey">
          <el-switch v-model="config.hotkeyEnabled" />
          <span class="form-tip">Enable global hotkey to show window</span>
        </el-form-item>

        <el-form-item label="Hotkey Combo" v-if="config.hotkeyEnabled">
          <div class="hotkey-combo">
            <el-select v-model="config.hotkeyModifiers" style="width: 140px">
              <el-option label="Cmd + Shift" value="cmd+shift" />
              <el-option label="Ctrl + Shift" value="ctrl+shift" />
              <el-option label="Cmd + Alt" value="cmd+alt" />
              <el-option label="Ctrl + Alt" value="ctrl+alt" />
            </el-select>
            <span class="hotkey-plus">+</span>
            <el-select v-model="config.hotkeyKey" style="width: 80px">
              <el-option label="P" value="p" />
              <el-option label="F" value="f" />
              <el-option label="M" value="m" />
              <el-option label="O" value="o" />
              <el-option label="W" value="w" />
            </el-select>
          </div>
          <div class="hotkey-display">
            Current: {{ getHotkeyDisplay() }}
          </div>
        </el-form-item>

        <el-alert
          v-if="config.hotkeyEnabled && systemInfo.os === 'darwin'"
          type="warning"
          :closable="false"
          style="margin-bottom: 16px"
        >
          <template #title>
            macOS requires Accessibility permission for global hotkeys.
            Go to System Settings → Privacy & Security → Accessibility.
          </template>
        </el-alert>

        <el-form-item>
          <el-button type="primary" @click="saveConfig">Save Settings</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- About -->
    <el-card class="settings-card">
      <template #header>
        <span>About</span>
      </template>

      <el-descriptions :column="1" border>
        <el-descriptions-item label="Application">Port Forward Manager</el-descriptions-item>
        <el-descriptions-item label="Version">1.0.0</el-descriptions-item>
        <el-descriptions-item label="Core Engine">gost (go-gost/x)</el-descriptions-item>
        <el-descriptions-item label="Framework">Wails v2 + Vue 3</el-descriptions-item>
        <el-descriptions-item label="Platform">
          {{ getPlatformDisplay() }}
        </el-descriptions-item>
        <el-descriptions-item label="Go Version">
          {{ systemInfo.version }}
        </el-descriptions-item>
      </el-descriptions>

      <div class="about-links" style="margin-top: 16px">
        <el-link type="primary" href="https://github.com/go-gost/x" target="_blank">
          <el-icon><Link /></el-icon>
          gost Documentation
        </el-link>
        <el-divider direction="vertical" />
        <el-link type="primary" href="https://wails.io" target="_blank">
          <el-icon><Link /></el-icon>
          Wails Documentation
        </el-link>
      </div>
    </el-card>

    <!-- Data Management -->
    <el-card class="settings-card">
      <template #header>
        <span>Data Management</span>
      </template>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        style="margin-bottom: 20px"
      >
        <template #title>
          Export/Import functions allow you to backup and restore your configurations.
        </template>
      </el-alert>

      <el-space wrap>
        <el-button type="primary" :loading="dataLoading" @click="exportRules">
          <el-icon><Upload /></el-icon>
          Export Data
        </el-button>
        <el-button type="success" :loading="dataLoading" @click="openImportDialog">
          <el-icon><Download /></el-icon>
          Import Data
        </el-button>
        <el-button type="danger" plain :loading="dataLoading" @click="clearAllData">
          <el-icon><Delete /></el-icon>
          Clear All Data
        </el-button>
      </el-space>

      <el-descriptions :column="2" border style="margin-top: 20px">
        <el-descriptions-item label="Total Rules">{{ store.rules.length }}</el-descriptions-item>
        <el-descriptions-item label="Total Chains">{{ store.chains.length }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <!-- Import Dialog -->
    <el-dialog
      v-model="importDialogVisible"
      title="Import Data"
      width="550px"
    >
      <el-form label-width="100px">
        <el-form-item label="Import Mode">
          <el-radio-group v-model="importMode">
            <el-radio value="replace">Replace All</el-radio>
            <el-radio value="merge">Merge</el-radio>
          </el-radio-group>
          <div class="import-mode-tip">
            <span v-if="importMode === 'replace'">Replace mode will overwrite all existing data.</span>
            <span v-else>Merge mode will add new items and update existing ones by ID.</span>
          </div>
        </el-form-item>

        <el-form-item label="Data Source">
          <div class="import-source">
            <el-button type="primary" plain @click="handleFileSelect">
              <el-icon><FolderOpened /></el-icon>
              Select File
            </el-button>
            <span class="file-tip">or paste JSON below</span>
          </div>
        </el-form-item>

        <el-form-item label="JSON Data">
          <el-input
            v-model="importData"
            type="textarea"
            :rows="10"
            placeholder='{"rules": [...], "chains": [...]}'
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="importDialogVisible = false">Cancel</el-button>
        <el-button type="primary" :loading="dataLoading" @click="confirmImport">
          Import
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.settings {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.settings-card {
  width: 100%;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.service-section {
  padding: 0 4px;
}

.service-actions {
  display: flex;
  align-items: center;
}

.form-tip {
  margin-left: 12px;
  color: #909399;
  font-size: 13px;
}

.about-links {
  display: flex;
  align-items: center;
}

.import-mode-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 8px;
}

.import-source {
  display: flex;
  align-items: center;
  gap: 12px;
}

.file-tip {
  color: #909399;
  font-size: 13px;
}

.hotkey-combo {
  display: flex;
  align-items: center;
  gap: 8px;
}

.hotkey-plus {
  font-weight: bold;
  color: #606266;
}

.hotkey-display {
  margin-top: 8px;
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 4px;
  font-family: monospace;
  font-size: 14px;
  color: #409eff;
}
</style>
