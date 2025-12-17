// Rule types
export type RuleType = 'forward' | 'reverse' | 'chain'
export type RuleStatus = 'stopped' | 'running' | 'error'
export type Protocol = 'tcp' | 'udp' | 'http' | 'https' | 'socks5' | 'ss'

export interface Target {
  host: string
  port: number
  weight: number
}

export interface Auth {
  username: string
  password: string
}

export interface TLSConfig {
  enabled: boolean
  certFile?: string
  keyFile?: string
  caFile?: string
  serverName?: string
  secure: boolean
}

export interface Rule {
  id: string
  name: string                 // 用途
  type: string
  enabled: boolean
  localPort: number            // 本地映射端口
  protocol: string
  targetHost: string           // 目标 IP/域名
  targetPort: number           // 目标端口
  targets: Target[]            // 保留用于负载均衡场景
  chainId?: string
  chain?: Chain | null         // For UI convenience
  auth?: Auth
  tls?: TLSConfig
  status: string
  errorMsg?: string
  description?: string         // 用途描述
  remark?: string              // 备注
  createdAt: any
  updatedAt: any
}

// Chain types
export interface Hop {
  name: string
  addr: string
  protocol: string
  auth?: Auth
  tls?: TLSConfig
}

export interface Chain {
  id: string
  name: string
  hops: Hop[]
  description?: string
  createdAt: any
  updatedAt: any
}

// Config types
export interface AppConfig {
  logLevel: string
  autoStart: boolean
  startMinimized: boolean
  serviceEnabled: boolean
  servicePort: number
  apiEnabled: boolean
  apiAddr: string
  apiAuth?: Auth
  metricsEnabled: boolean
  metricsAddr: string
}

// Status types
export interface ServiceStatus {
  running: boolean
  embedded?: boolean
  pid?: number
  startTime?: string
  rulesActive: number
  rulesTotal: number
  version: string
}

// Form types
export interface RuleForm {
  name: string                 // 用途
  type: RuleType
  localPort: number            // 本地映射端口
  protocol: Protocol
  targetHost: string           // 目标 IP/域名
  targetPort: number           // 目标端口
  targets: Target[]            // 用于负载均衡
  chainId: string
  authEnabled: boolean
  username: string
  password: string
  tlsEnabled: boolean
  description: string          // 用途描述
  remark: string               // 备注
}

export interface ChainForm {
  name: string
  hops: Hop[]
  description: string
}

// Statistics types
export interface RuleStats {
  ruleId: string
  bytesIn: number
  bytesOut: number
  connections: number
  activeConns: number
  errors: number
  lastActivity?: string
}

// Log types
export type LogLevel = 'debug' | 'info' | 'warn' | 'error'

export interface LogEntry {
  id: number
  timestamp: string
  level: LogLevel
  ruleId?: string
  ruleName?: string
  message: string
  details?: string
}
