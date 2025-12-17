import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Rule, Chain, AppConfig, ServiceStatus } from '../types'
import {
  GetRules,
  GetChains,
  CreateRule,
  UpdateRule,
  DeleteRule,
  StartRule,
  StopRule,
  CreateChain,
  UpdateChain,
  DeleteChain,
  GetConfig,
  UpdateConfig,
  GetStatus,
  GetServiceStatus,
  NewRule,
  NewChain,
  InstallService,
  UninstallService,
  StartService,
  StopService,
  RestartService
} from '../../wailsjs/go/main/App'

export const useRulesStore = defineStore('rules', () => {
  // State
  const rules = ref<Rule[]>([])
  const chains = ref<Chain[]>([])
  const config = ref<AppConfig | null>(null)
  const status = ref<ServiceStatus | null>(null)
  const serviceStatus = ref<string>('not_installed')
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Getters
  const runningRules = computed(() => rules.value.filter(r => r.status === 'running'))
  const stoppedRules = computed(() => rules.value.filter(r => r.status === 'stopped'))
  const errorRules = computed(() => rules.value.filter(r => r.status === 'error'))

  const forwardRules = computed(() => rules.value.filter(r => r.type === 'forward'))
  const reverseRules = computed(() => rules.value.filter(r => r.type === 'reverse'))
  const chainRules = computed(() => rules.value.filter(r => r.type === 'chain'))

  // Actions
  async function fetchRules() {
    loading.value = true
    error.value = null
    try {
      const data = await GetRules()
      rules.value = (data || []) as unknown as Rule[]
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch rules'
    } finally {
      loading.value = false
    }
  }

  async function fetchChains() {
    loading.value = true
    error.value = null
    try {
      const data = await GetChains()
      chains.value = data || []
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch chains'
    } finally {
      loading.value = false
    }
  }

  async function fetchConfig() {
    try {
      const data = await GetConfig()
      config.value = data
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch config'
    }
  }

  async function fetchStatus() {
    try {
      const data = await GetStatus()
      status.value = data
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch status'
    }
  }

  async function fetchServiceStatus() {
    try {
      serviceStatus.value = await GetServiceStatus()
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch service status'
    }
  }

  async function createRule(rule: Rule): Promise<boolean> {
    loading.value = true
    error.value = null
    try {
      await CreateRule(rule as any)
      await fetchRules()
      return true
    } catch (e: any) {
      error.value = e.message || 'Failed to create rule'
      return false
    } finally {
      loading.value = false
    }
  }

  async function updateRule(rule: Rule): Promise<boolean> {
    loading.value = true
    error.value = null
    try {
      await UpdateRule(rule as any)
      await fetchRules()
      return true
    } catch (e: any) {
      error.value = e.message || 'Failed to update rule'
      return false
    } finally {
      loading.value = false
    }
  }

  async function deleteRule(id: string): Promise<boolean> {
    loading.value = true
    error.value = null
    try {
      await DeleteRule(id)
      await fetchRules()
      return true
    } catch (e: any) {
      error.value = e.message || 'Failed to delete rule'
      return false
    } finally {
      loading.value = false
    }
  }

  async function startRule(id: string): Promise<boolean> {
    loading.value = true
    error.value = null
    try {
      await StartRule(id)
      await fetchRules()
      return true
    } catch (e: any) {
      error.value = e.message || 'Failed to start rule'
      return false
    } finally {
      loading.value = false
    }
  }

  async function stopRule(id: string): Promise<boolean> {
    loading.value = true
    error.value = null
    try {
      await StopRule(id)
      await fetchRules()
      return true
    } catch (e: any) {
      error.value = e.message || 'Failed to stop rule'
      return false
    } finally {
      loading.value = false
    }
  }

  async function createChain(chain: Chain): Promise<boolean> {
    loading.value = true
    error.value = null
    try {
      await CreateChain(chain as any)
      await fetchChains()
      return true
    } catch (e: any) {
      error.value = e.message || 'Failed to create chain'
      return false
    } finally {
      loading.value = false
    }
  }

  async function updateChain(chain: Chain): Promise<boolean> {
    loading.value = true
    error.value = null
    try {
      await UpdateChain(chain as any)
      await fetchChains()
      return true
    } catch (e: any) {
      error.value = e.message || 'Failed to update chain'
      return false
    } finally {
      loading.value = false
    }
  }

  async function deleteChain(id: string): Promise<boolean> {
    loading.value = true
    error.value = null
    try {
      await DeleteChain(id)
      await fetchChains()
      return true
    } catch (e: any) {
      error.value = e.message || 'Failed to delete chain'
      return false
    } finally {
      loading.value = false
    }
  }

  async function saveConfig(newConfig: AppConfig): Promise<boolean> {
    loading.value = true
    error.value = null
    try {
      await UpdateConfig(newConfig as any)
      config.value = newConfig
      return true
    } catch (e: any) {
      error.value = e.message || 'Failed to save config'
      return false
    } finally {
      loading.value = false
    }
  }

  async function createNewRule(name: string, type: string): Promise<Rule | null> {
    try {
      const result = await NewRule(name, type)
      return result as unknown as Rule
    } catch (e: any) {
      error.value = e.message || 'Failed to create new rule'
      return null
    }
  }

  async function createNewChain(name: string): Promise<Chain | null> {
    try {
      return await NewChain(name)
    } catch (e: any) {
      error.value = e.message || 'Failed to create new chain'
      return null
    }
  }

  // Config helpers
  function getConfig(): AppConfig | null {
    return config.value
  }

  async function refreshServiceStatus(): Promise<void> {
    await fetchServiceStatus()
  }

  // Service management
  async function installService(): Promise<boolean> {
    try {
      await InstallService()
      return true
    } catch (e: any) {
      error.value = e.message || 'Failed to install service'
      throw e
    }
  }

  async function uninstallService(): Promise<boolean> {
    try {
      await UninstallService()
      return true
    } catch (e: any) {
      error.value = e.message || 'Failed to uninstall service'
      throw e
    }
  }

  async function startService(): Promise<boolean> {
    try {
      await StartService()
      return true
    } catch (e: any) {
      error.value = e.message || 'Failed to start service'
      throw e
    }
  }

  async function stopService(): Promise<boolean> {
    try {
      await StopService()
      return true
    } catch (e: any) {
      error.value = e.message || 'Failed to stop service'
      throw e
    }
  }

  async function restartService(): Promise<boolean> {
    try {
      await RestartService()
      return true
    } catch (e: any) {
      error.value = e.message || 'Failed to restart service'
      throw e
    }
  }

  // Initialize
  async function init() {
    await Promise.all([
      fetchRules(),
      fetchChains(),
      fetchConfig(),
      fetchStatus(),
      fetchServiceStatus()
    ])
  }

  return {
    // State
    rules,
    chains,
    config,
    status,
    serviceStatus,
    loading,
    error,
    // Getters
    runningRules,
    stoppedRules,
    errorRules,
    forwardRules,
    reverseRules,
    chainRules,
    // Actions
    fetchRules,
    fetchChains,
    fetchConfig,
    fetchStatus,
    fetchServiceStatus,
    createRule,
    updateRule,
    deleteRule,
    startRule,
    stopRule,
    createChain,
    updateChain,
    deleteChain,
    saveConfig,
    createNewRule,
    createNewChain,
    getConfig,
    refreshServiceStatus,
    installService,
    uninstallService,
    startService,
    stopService,
    restartService,
    init
  }
})
