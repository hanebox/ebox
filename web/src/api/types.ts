export interface User {
  id: number
  name: string
  email: string
  created_at: string
  updated_at: string
}

export interface SessionInfo {
  id: number
  ip: string
  user_agent: string
  last_seen_at: string
  created_at: string
  expires_at: string
  current: boolean
}

export interface SetupStatus {
  initialized: boolean
}

export interface AuthResponse {
  user: User
  csrf_token: string
}

export interface PanelSettings {
  panel_domain: string
  acme_email: string
  timezone: string
}

export interface HostSample {
  at: string
  cpu_percent: number
  mem_used: number
  mem_total: number
  mem_percent: number
  swap_used: number
  swap_total: number
  load1: number
  load5: number
  load15: number
  disk_used: number
  disk_total: number
  disk_percent: number
  net_rx_rate: number
  net_tx_rate: number
  uptime: number
}

export interface HostMetrics {
  current: HostSample
  history: HostSample[]
}

export interface DockerInfo {
  available: boolean
  version?: string
  error?: string
}

export interface SystemInfo {
  hostname: string
  platform: string
  platform_version: string
  kernel_version: string
  arch: string
  cpus: number
  mem_total: number
  uptime: number
  go_version: string
  panel_version: string
  panel_commit: string
  docker: DockerInfo
}

export interface StreamEvent {
  topic: string
  type: string
  data?: unknown
  at: string
}

export interface Project {
  id: number
  uuid: string
  name: string
  description: string
  created_at: string
  updated_at: string
}

export type SourceType = 'github_app' | 'git'
export type SourceAuthType = 'none' | 'token' | 'ssh'

export interface Source {
  id: number
  uuid: string
  name: string
  type: SourceType
  app_id: number
  app_slug: string
  installation_id: string
  html_url: string
  git_url: string
  auth_type: SourceAuthType
  created_at: string
  updated_at: string
}

export interface Repository {
  full_name: string
  name: string
  private: boolean
  default_branch: string
  html_url: string
  clone_url: string
  ssh_url: string
  description: string
  updated_at: string
}

export interface RepositoryPage {
  repositories: Repository[]
  total: number
  page: number
}

export interface Branch {
  name: string
  protected: boolean
  default: boolean
}

export interface GithubInstallation {
  id: number
  account: string
  type: string
  html_url: string
}

export interface AccessToken {
  id: number
  name: string
  last_used_at: string | null
  expires_at: string | null
  created_at: string
}

export interface AccessTokenCreated {
  token: string
  record: AccessToken
}

export interface GithubManifestStart {
  action_url: string
  manifest: Record<string, unknown>
  state: string
}

export type BuildPack = 'dockerfile' | 'static' | 'image' | 'compose'
export type ApplicationKind = 'git' | 'image'
export type ApplicationStatus = 'idle' | 'deploying' | 'failed'

export type DeploymentStatus =
  | 'queued'
  | 'building'
  | 'deploying'
  | 'success'
  | 'failed'
  | 'cancelled'

export interface ResourceLimits {
  memory_mb?: number
  cpus?: number
}

export interface ApplicationConfig {
  base_directory: string
  dockerfile_path: string
  build_command: string
  publish_directory: string
  image: string
  expose_port: number
  limits?: ResourceLimits
}

export interface Application {
  id: number
  uuid: string
  project_id: number
  source_id: number | null
  name: string
  kind: ApplicationKind
  repository: string
  branch: string
  build_pack: BuildPack
  config: ApplicationConfig
  auto_deploy: boolean
  status: ApplicationStatus
  project?: Project
  source?: Source
  created_at: string
  updated_at: string
}

export interface ApplicationRuntime {
  state: string
  health?: string
  container_id?: string
  container_name?: string
  image?: string
  started_at?: string
  latest_deployment?: Deployment
}

export interface ApplicationWithRuntime extends Application {
  runtime: ApplicationRuntime
}

export interface ApplicationDetail extends ApplicationWithRuntime {
  deploy_url?: string
  domains: Domain[]
}

export interface Domain {
  id: number
  resource_type: string
  resource_id: number
  host: string
  port: number
  https: boolean
  is_primary: boolean
  created_at: string
  updated_at: string
}

export interface EnvVar {
  id: number
  resource_type: string
  resource_id: number
  key: string
  value: string
  is_build_time: boolean
  created_at: string
  updated_at: string
}

export interface Deployment {
  id: number
  uuid: string
  application_id: number
  status: DeploymentStatus
  trigger: string
  commit: string
  commit_message: string
  image_tag: string
  container_id: string
  container_name: string
  error: string
  started_at: string | null
  finished_at: string | null
  duration_ms: number
  created_at: string
  updated_at: string
  application_name?: string
  application_uuid?: string
}

export interface DeploymentLog {
  seq: number
  lines: string[]
  status: DeploymentStatus
}

export interface ContainerStats {
  cpu_percent: number
  memory_usage: number
  memory_limit: number
  memory_percent: number
}

export interface ContainerInfo {
  id: string
  name: string
  image: string
  state: string
  status: string
  health?: string
  labels?: Record<string, string>
  stats?: ContainerStats
}

export interface TerminalTarget {
  id: string
  name: string
  kind: 'host' | 'container'
  state: string
  image?: string
  resource?: string
}

export interface ProxyStatus {
  installed: boolean
  running: boolean
  state: string
  container_id?: string
  container?: string
  image?: string
  version?: string
  ports?: string[]
  api_reachable: boolean
  error?: string
}

export interface ProxySettings {
  panel_domain: string
  acme_email: string
  http_port: number
  https_port: number
}

export interface ProxyRoute {
  name: string
  rule: string
  service: string
  entrypoints: string[]
  status: string
  tls: boolean
  provider: string
  priority?: number
}

export interface ProxyCertificate {
  domain: string
  sans: string[]
  not_after: string
  days_left: number
  issuer: string
  resolved_via: string
}

export interface ProxyMiddleware {
  name: string
  type: string
  config: Record<string, unknown>
  description?: string
  provider: string
  editable: boolean
}

export interface UsageSection {
  count: number
  active: number
  size: number
  reclaimable: number
}

export interface DiskUsage {
  images: UsageSection
  containers: UsageSection
  volumes: UsageSection
  build_cache: UsageSection
  total_size: number
}

export interface PruneResult {
  images: number
  containers: number
  volumes: number
  build_cache: number
  space_reclaimed: number
}

export interface DatabaseDefinition {
  type: string
  label: string
  image: string
  versions: string[]
  internal_port: number
  volume_path: string
}

export interface DatabaseConnection {
  host: string
  port: number
  public_host?: string
  public_port?: number
  username: string
  password?: string
  database: string
  url?: string
  public_url?: string
  environment: Record<string, string>
  internal_host: string
  exposed: boolean
}

export interface DatabaseRecord {
  id: number
  uuid: string
  project_id: number
  name: string
  type: string
  version: string
  port: number
  limits?: ResourceLimits
  status: string
  status_message?: string
  project: Project
  created_at: string
  updated_at: string
}

export interface DatabaseRuntimeStatus {
  state: string
  health?: string
  container_id?: string
  container_name?: string
  image?: string
  uptime?: string
}

export interface DatabaseResponse extends DatabaseRecord {
  runtime: DatabaseRuntimeStatus
  definition?: DatabaseDefinition
  connection?: DatabaseConnection
}

export interface ComposeServiceInfo {
  name: string
  image: string
  build: boolean
  ports: string[]
  depends_on: string[]
}

export interface ServiceRecord {
  id: number
  uuid: string
  project_id: number
  source_id?: number
  name: string
  source: 'inline' | 'git'
  repository: string
  branch: string
  compose_path: string
  compose: string
  environment: string
  entry_service: string
  entry_port: number
  status: string
  last_error?: string
  last_deployed_at?: string
  project: Project
  source_record?: Source
  created_at: string
  updated_at: string
}

export interface ServiceResponse extends Omit<ServiceRecord, 'entry_service' | 'entry_port'> {
  runtime: {
    state: string
    services: Array<{
      name: string
      state: string
      health?: string
      container_id?: string
      container_name?: string
      image?: string
      ports?: string[]
    }>
  }
  definitions?: ComposeServiceInfo[]
  entry_service?: string
  entry_port?: number
}

export interface StackContainer {
  id: string
  name: string
  service: string
  state: string
  health?: string
  image: string
  ports?: string[]
}

export interface ComposePreview {
  services: ComposeServiceInfo[]
  entry_service: string
}

export interface NotificationChannel {
  id: number
  uuid: string
  name: string
  type: NotificationChannelType
  enabled: boolean
  config?: Record<string, string>
  subscribed_events: string[]
  created_at: string
  updated_at: string
}

export type NotificationChannelType = 'email' | 'discord' | 'slack' | 'telegram' | 'webhook'

export interface NotificationDelivery {
  id: number
  channel_id: number
  channel: string
  event: string
  title: string
  body: string
  status: 'sent' | 'failed'
  error?: string
  attempts: number
  created_at: string
}

export interface BackupRecord {
  id: number
  uuid: string
  project_id?: number
  name: string
  target: 'database' | 'volume'
  resource_id: string
  volume_name: string
  schedule: string
  retention: number
  enabled: boolean
  stop_during_backup: boolean
  last_run_at?: string
  last_status?: string
  last_error?: string
  project?: Project
  next_run_at?: string
  schedule_hint?: string
  created_at: string
  updated_at: string
}

export interface BackupRun {
  id: number
  backup_id: number
  status: 'pending' | 'running' | 'success' | 'failed'
  trigger: string
  file: string
  size_bytes: number
  duration_ms: number
  error?: string
  started_at?: string
  finished_at?: string
  created_at: string
}

export interface BackupDetail extends BackupRecord {
  runs: BackupRun[]
}

export interface BackupTargets {
  databases: Array<{ uuid: string; name: string; type: string; project_id: number; port: number }>
  volumes: Array<{ name: string; driver: string; mountpoint?: string; created_at?: string; labels?: Record<string, string> }>
}

export interface UpdateStatus {
  current_version: string
  current_commit: string
  build_date: string
  latest?: {
    version: string
    name: string
    notes?: string
    url?: string
    published_at?: string
    asset_name?: string
    asset_size?: number
  }
  update_available: boolean
  check_error?: string
  checked_at: string
  managed: boolean
  self_updateable: boolean
  reason?: string
  asset_name?: string
}
