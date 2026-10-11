import type {
  AccessToken,
  ComposePreview,
  ContainerInfo,
  DatabaseConnection,
  DatabaseDefinition,
  DatabaseRecord,
  DatabaseResponse,
  BackupDetail,
  BackupRecord,
  BackupRun,
  BackupTargets,
  NotificationChannel,
  NotificationDelivery,
  ServiceRecord,
  ServiceResponse,
  StackContainer,
  UpdateStatus,
  ContainerStats,
  DiskUsage,
  PruneResult,
  ProxyCertificate,
  ProxyMiddleware,
  ProxyRoute,
  ProxySettings,
  ProxyStatus,
  TerminalTarget,
  AccessTokenCreated,
  Application,
  ApplicationConfig,
  ApplicationDetail,
  ApplicationWithRuntime,
  AuthResponse,
  Branch,
  BuildPack,
  Deployment,
  DeploymentLog,
  Domain,
  EnvVar,
  GithubInstallation,
  GithubManifestStart,
  HostMetrics,
  PanelSettings,
  Project,
  RepositoryPage,
  SessionInfo,
  SetupStatus,
  Source,
  SourceAuthType,
  SystemInfo,
  User,
} from '@/api/types'
import { apiFetch } from '@/lib/api'

export const api = {
  setupStatus: () => apiFetch<SetupStatus>('/setup/status'),

  setup: (payload: {
    name: string
    email: string
    password: string
    panel_domain?: string
    acme_email?: string
  }) => apiFetch<AuthResponse>('/setup', { method: 'POST', body: payload }),

  login: (email: string, password: string) =>
    apiFetch<AuthResponse>('/auth/login', { method: 'POST', body: { email, password } }),

  me: () => apiFetch<AuthResponse>('/auth/me'),

  logout: () => apiFetch<{ message: string }>('/auth/logout', { method: 'POST' }),

  sessions: () => apiFetch<SessionInfo[]>('/sessions'),

  revokeSession: (id: number) =>
    apiFetch<{ message: string }>(`/sessions/${id}`, { method: 'DELETE' }),

  updateAccount: (payload: { name: string; email: string }) =>
    apiFetch<User>('/account', { method: 'PUT', body: payload }),

  updatePassword: (payload: { current_password: string; password: string }) =>
    apiFetch<{ message: string }>('/account/password', { method: 'PUT', body: payload }),

  settings: () => apiFetch<PanelSettings>('/settings'),

  updateSettings: (payload: Partial<PanelSettings>) =>
    apiFetch<PanelSettings>('/settings', { method: 'PUT', body: payload }),

  hostMetrics: (window: '20m' | '24h' = '20m') =>
    apiFetch<HostMetrics>(`/metrics/host?window=${window}`),

  systemInfo: () => apiFetch<SystemInfo>('/system/info'),

  projects: () => apiFetch<Project[]>('/projects'),

  project: (uuid: string) => apiFetch<Project>(`/projects/${uuid}`),

  createProject: (payload: { name: string; description?: string }) =>
    apiFetch<Project>('/projects', { method: 'POST', body: payload }),

  updateProject: (uuid: string, payload: { name: string; description?: string }) =>
    apiFetch<Project>(`/projects/${uuid}`, { method: 'PUT', body: payload }),

  deleteProject: (uuid: string) =>
    apiFetch<{ message: string }>(`/projects/${uuid}`, { method: 'DELETE' }),

  sources: () => apiFetch<Source[]>('/sources'),

  source: (uuid: string) => apiFetch<Source>(`/sources/${uuid}`),

  createGitSource: (payload: {
    name: string
    git_url: string
    auth_type: SourceAuthType
    token?: string
    ssh_key?: string
    test_connection?: boolean
  }) => apiFetch<Source>('/sources', { method: 'POST', body: { type: 'git', ...payload } }),

  createGithubSource: (payload: {
    name: string
    app_id: number
    private_key: string
    webhook_secret?: string
    installation_id?: string
  }) =>
    apiFetch<Source>('/sources', { method: 'POST', body: { type: 'github_app', ...payload } }),

  updateSource: (uuid: string, payload: { name: string }) =>
    apiFetch<Source>(`/sources/${uuid}`, { method: 'PUT', body: payload }),

  setSourceInstallation: (uuid: string, installationId: string) =>
    apiFetch<Source>(`/sources/${uuid}/installation`, {
      method: 'PUT',
      body: { installation_id: installationId },
    }),

  deleteSource: (uuid: string) =>
    apiFetch<{ message: string }>(`/sources/${uuid}`, { method: 'DELETE' }),

  testSource: (uuid: string) =>
    apiFetch<{ message: string }>(`/sources/${uuid}/test`, { method: 'POST' }),

  repositories: (uuid: string, page = 1) =>
    apiFetch<RepositoryPage>(`/sources/${uuid}/repositories?page=${page}`),

  branches: (uuid: string, repo: string, defaultBranch?: string) => {
    const params = new URLSearchParams({ repo })
    if (defaultBranch) {
      params.set('default_branch', defaultBranch)
    }

    return apiFetch<Branch[]>(`/sources/${uuid}/branches?${params.toString()}`)
  },

  githubInstallations: (uuid: string) =>
    apiFetch<GithubInstallation[]>(`/sources/${uuid}/github/installations`),

  startGithubManifest: (payload: { origin: string; name?: string }) =>
    apiFetch<GithubManifestStart>('/github/manifest', { method: 'POST', body: payload }),

  tokens: () => apiFetch<AccessToken[]>('/tokens'),

  createToken: (payload: { name: string; expires_in_days?: number }) =>
    apiFetch<AccessTokenCreated>('/tokens', { method: 'POST', body: payload }),

  deleteToken: (id: number) => apiFetch<{ message: string }>(`/tokens/${id}`, { method: 'DELETE' }),

  applications: (project?: string) =>
    apiFetch<ApplicationWithRuntime[]>(
      project ? `/applications?project=${encodeURIComponent(project)}` : '/applications',
    ),

  application: (uuid: string) => apiFetch<ApplicationDetail>(`/applications/${uuid}`),

  createApplication: (payload: {
    project: string
    name: string
    source?: string
    repository?: string
    branch?: string
    build_pack: BuildPack
    config?: Partial<ApplicationConfig>
    auto_deploy?: boolean
    domains?: string[]
    env?: { key: string; value: string; is_build_time?: boolean }[]
    deploy?: boolean
  }) => apiFetch<Application>('/applications', { method: 'POST', body: payload }),

  updateApplication: (
    uuid: string,
    payload: {
      project?: string
      name: string
      source?: string
      repository?: string
      branch?: string
      build_pack: BuildPack
      config?: Partial<ApplicationConfig>
      auto_deploy?: boolean
    },
  ) => apiFetch<Application>(`/applications/${uuid}`, { method: 'PUT', body: payload }),

  deleteApplication: (uuid: string) =>
    apiFetch<{ message: string }>(`/applications/${uuid}`, { method: 'DELETE' }),

  deployApplication: (uuid: string, commit?: string) =>
    apiFetch<Deployment>(`/applications/${uuid}/deploy`, { method: 'POST', body: { commit } }),

  rollbackApplication: (uuid: string, deployment: string) =>
    apiFetch<Deployment>(`/applications/${uuid}/rollback`, { method: 'POST', body: { deployment } }),

  applicationAction: (uuid: string, action: 'start' | 'stop' | 'restart') =>
    apiFetch<{ message: string }>(`/applications/${uuid}/${action}`, { method: 'POST' }),

  applicationLogs: (uuid: string, tail = 200) =>
    apiFetch<{ lines: string[]; container: unknown }>(`/applications/${uuid}/logs?tail=${tail}`),

  regenerateDeployToken: (uuid: string) =>
    apiFetch<{ deploy_token: string; deploy_url: string }>(
      `/applications/${uuid}/token/regenerate`,
      { method: 'POST' },
    ),

  applicationEnv: (uuid: string) => apiFetch<EnvVar[]>(`/applications/${uuid}/env`),

  createEnvVar: (uuid: string, payload: { key: string; value: string; is_build_time?: boolean }) =>
    apiFetch<EnvVar>(`/applications/${uuid}/env`, { method: 'POST', body: payload }),

  updateEnvVar: (
    uuid: string,
    id: number,
    payload: { key: string; value: string; is_build_time?: boolean },
  ) => apiFetch<EnvVar>(`/applications/${uuid}/env/${id}`, { method: 'PUT', body: payload }),

  deleteEnvVar: (uuid: string, id: number) =>
    apiFetch<{ message: string }>(`/applications/${uuid}/env/${id}`, { method: 'DELETE' }),

  importEnvVars: (uuid: string, content: string) =>
    apiFetch<{ imported: number }>(`/applications/${uuid}/env/import`, {
      method: 'POST',
      body: { content },
    }),

  exportEnvVars: (uuid: string) =>
    apiFetch<{ content: string }>(`/applications/${uuid}/env/export`),

  resourceEnv: (resource: 'applications' | 'services', uuid: string) =>
    apiFetch<EnvVar[]>(`/${resource}/${uuid}/env`),

  createResourceEnv: (
    resource: 'applications' | 'services',
    uuid: string,
    payload: { key: string; value: string; is_build_time?: boolean },
  ) => apiFetch<EnvVar>(`/${resource}/${uuid}/env`, { method: 'POST', body: payload }),

  updateResourceEnv: (
    resource: 'applications' | 'services',
    uuid: string,
    id: number,
    payload: { key: string; value: string; is_build_time?: boolean },
  ) => apiFetch<EnvVar>(`/${resource}/${uuid}/env/${id}`, { method: 'PUT', body: payload }),

  deleteResourceEnv: (resource: 'applications' | 'services', uuid: string, id: number) =>
    apiFetch<{ message: string }>(`/${resource}/${uuid}/env/${id}`, { method: 'DELETE' }),

  importResourceEnv: (resource: 'applications' | 'services', uuid: string, content: string) =>
    apiFetch<{ imported: number }>(`/${resource}/${uuid}/env/import`, {
      method: 'POST',
      body: { content },
    }),

  exportResourceEnv: (resource: 'applications' | 'services', uuid: string) =>
    apiFetch<{ content: string }>(`/${resource}/${uuid}/env/export`),

  resourceDomains: (resource: 'applications' | 'services', uuid: string) =>
    apiFetch<Domain[]>(`/${resource}/${uuid}/domains`),

  createResourceDomain: (
    resource: 'applications' | 'services',
    uuid: string,
    payload: { host: string; https?: boolean; is_primary?: boolean },
  ) => apiFetch<Domain>(`/${resource}/${uuid}/domains`, { method: 'POST', body: payload }),

  updateResourceDomain: (
    resource: 'applications' | 'services',
    uuid: string,
    id: number,
    payload: { host?: string; https?: boolean; is_primary?: boolean },
  ) => apiFetch<Domain>(`/${resource}/${uuid}/domains/${id}`, { method: 'PUT', body: payload }),

  deleteResourceDomain: (resource: 'applications' | 'services', uuid: string, id: number) =>
    apiFetch<{ message: string }>(`/${resource}/${uuid}/domains/${id}`, { method: 'DELETE' }),

  applicationDomains: (uuid: string) => apiFetch<Domain[]>(`/applications/${uuid}/domains`),

  addDomain: (uuid: string, payload: { host: string; https?: boolean; is_primary?: boolean }) =>
    apiFetch<Domain>(`/applications/${uuid}/domains`, { method: 'POST', body: payload }),

  updateDomain: (
    uuid: string,
    id: number,
    payload: { host: string; https?: boolean; is_primary?: boolean },
  ) => apiFetch<Domain>(`/applications/${uuid}/domains/${id}`, { method: 'PUT', body: payload }),

  deleteDomain: (uuid: string, id: number) =>
    apiFetch<{ message: string }>(`/applications/${uuid}/domains/${id}`, { method: 'DELETE' }),

  deployments: (application?: string, limit = 25) => {
    const params = new URLSearchParams({ limit: String(limit) })
    if (application) {
      params.set('application', application)
    }

    return apiFetch<Deployment[]>(`/deployments?${params.toString()}`)
  },

  deployment: (uuid: string) => apiFetch<Deployment>(`/deployments/${uuid}`),

  deploymentLogs: (uuid: string, since = 0) =>
    apiFetch<DeploymentLog>(`/deployments/${uuid}/logs?since=${since}`),

  cancelDeployment: (uuid: string) =>
    apiFetch<{ message: string }>(`/deployments/${uuid}/cancel`, { method: 'POST' }),

  containers: () => apiFetch<ContainerInfo[]>('/containers'),

  databases: (project?: string) =>
    apiFetch<DatabaseResponse[]>(`/databases${project ? `?project=${project}` : ''}`),

  databaseDefinitions: () => apiFetch<DatabaseDefinition[]>('/databases/definitions'),

  database: (uuid: string) => apiFetch<DatabaseResponse>(`/databases/${uuid}`),

  databaseCredentials: (uuid: string) =>
    apiFetch<DatabaseConnection>(`/databases/${uuid}/credentials`),

  createDatabase: (payload: {
    project: string
    name: string
    type: string
    version: string
    port?: number
    limits?: { memory_mb?: number; cpus?: number }
  }) => apiFetch<DatabaseResponse>('/databases', { method: 'POST', body: payload }),

  updateDatabase: (
    uuid: string,
    payload: {
      name: string
      version: string
      port?: number
      limits?: { memory_mb?: number; cpus?: number }
    },
  ) => apiFetch<DatabaseRecord>(`/databases/${uuid}`, { method: 'PUT', body: payload }),

  deleteDatabase: (uuid: string) =>
    apiFetch<{ message: string }>(`/databases/${uuid}`, { method: 'DELETE' }),

  databaseLifecycle: (uuid: string, action: 'start' | 'stop' | 'restart') =>
    apiFetch<{ message: string }>(`/databases/${uuid}/${action}`, { method: 'POST' }),

  attachDatabase: (uuid: string, application: string) =>
    apiFetch<{ message: string; variables: string[] }>(`/databases/${uuid}/attach`, {
      method: 'POST',
      body: { application },
    }),

  services: (project?: string) =>
    apiFetch<ServiceResponse[]>(`/services${project ? `?project=${project}` : ''}`),

  service: (uuid: string) => apiFetch<ServiceResponse>(`/services/${uuid}`),

  createService: (payload: {
    project: string
    name: string
    source: 'inline' | 'git'
    compose?: string
    source_uuid?: string
    repository?: string
    branch?: string
    compose_path?: string
    environment?: string
    entry_service?: string
    entry_port?: number
  }) => apiFetch<ServiceRecord>('/services', { method: 'POST', body: payload }),

  updateService: (
    uuid: string,
    payload: {
      name: string
      source: 'inline' | 'git'
      compose?: string
      source_uuid?: string
      repository?: string
      branch?: string
      compose_path?: string
      environment?: string
      entry_service?: string
      entry_port?: number
    },
  ) => apiFetch<ServiceRecord>(`/services/${uuid}`, { method: 'PUT', body: payload }),

  deleteService: (uuid: string) =>
    apiFetch<{ message: string }>(`/services/${uuid}`, { method: 'DELETE' }),

  serviceLifecycle: (uuid: string, action: 'deploy' | 'start' | 'stop' | 'restart') =>
    apiFetch<{ message: string }>(`/services/${uuid}/${action}`, { method: 'POST' }),

  serviceContainers: (uuid: string) => apiFetch<StackContainer[]>(`/services/${uuid}/containers`),

  applicationCompose: (uuid: string) => apiFetch<ComposePreview>(`/applications/${uuid}/compose`),

  notificationChannels: () => apiFetch<NotificationChannel[]>('/notifications/channels'),

  notificationEvents: () => apiFetch<{ events: string[] }>('/notifications/events'),

  notificationHistory: (limit = 50) =>
    apiFetch<NotificationDelivery[]>(`/notifications/history?limit=${limit}`),

  createNotificationChannel: (payload: {
    name: string
    type: string
    enabled: boolean
    config: Record<string, string>
    events: string[]
  }) => apiFetch<NotificationChannel>('/notifications/channels', { method: 'POST', body: payload }),

  updateNotificationChannel: (
    uuid: string,
    payload: {
      name: string
      type: string
      enabled: boolean
      config: Record<string, string>
      events: string[]
    },
  ) => apiFetch<NotificationChannel>(`/notifications/channels/${uuid}`, { method: 'PUT', body: payload }),

  deleteNotificationChannel: (uuid: string) =>
    apiFetch<{ message: string }>(`/notifications/channels/${uuid}`, { method: 'DELETE' }),

  toggleNotificationChannel: (uuid: string, enabled: boolean) =>
    apiFetch<{ enabled: boolean }>(`/notifications/channels/${uuid}/toggle`, {
      method: 'POST',
      body: { enabled },
    }),

  testNotificationChannel: (uuid: string) =>
    apiFetch<{ message: string }>(`/notifications/channels/${uuid}/test`, { method: 'POST' }),

  sendNotification: (title: string, body: string) =>
    apiFetch<{ message: string }>('/notifications/send', { method: 'POST', body: { title, body } }),

  backups: (project?: string) =>
    apiFetch<BackupRecord[]>(`/backups${project ? `?project=${project}` : ''}`),

  backupTargets: () => apiFetch<BackupTargets>('/backups/targets'),

  backup: (uuid: string) => apiFetch<BackupDetail>(`/backups/${uuid}`),

  createBackup: (payload: {
    project?: string
    name: string
    target: 'database' | 'volume'
    resource_id?: string
    volume_name?: string
    schedule: string
    retention: number
    enabled: boolean
    stop_during_backup?: boolean
  }) => apiFetch<BackupRecord>('/backups', { method: 'POST', body: payload }),

  updateBackup: (
    uuid: string,
    payload: {
      name: string
      target: 'database' | 'volume'
      resource_id?: string
      volume_name?: string
      schedule: string
      retention: number
      enabled: boolean
      stop_during_backup?: boolean
    },
  ) => apiFetch<BackupRecord>(`/backups/${uuid}`, { method: 'PUT', body: payload }),

  deleteBackup: (uuid: string) => apiFetch<{ message: string }>(`/backups/${uuid}`, { method: 'DELETE' }),

  runBackup: (uuid: string) => apiFetch<BackupRun>(`/backups/${uuid}/run`, { method: 'POST' }),

  restoreBackup: (uuid: string, run: number) =>
    apiFetch<{ message: string }>(`/backups/${uuid}/runs/${run}/restore`, { method: 'POST' }),

  deleteBackupRun: (uuid: string, run: number) =>
    apiFetch<{ message: string }>(`/backups/${uuid}/runs/${run}`, { method: 'DELETE' }),

  updateStatus: (check = false) => apiFetch<UpdateStatus>(`/system/update?check=${check ? '1' : '0'}`),

  applyUpdate: (force = false) =>
    apiFetch<{ message: string; version: string }>('/system/update', { method: 'POST', body: { force } }),

  rollbackUpdate: () =>
    apiFetch<{ message: string }>('/system/update/rollback', { method: 'POST' }),

  applicationStats: (uuid: string) =>
    apiFetch<{ container: ContainerInfo | null; stats: ContainerStats | null }>(
      `/applications/${uuid}/stats`,
    ),

  terminalTargets: () => apiFetch<TerminalTarget[]>('/terminal/targets'),

  proxyStatus: () => apiFetch<{ status: ProxyStatus; settings: ProxySettings }>('/proxy'),

  proxyRoutes: () => apiFetch<ProxyRoute[]>('/proxy/routes'),

  proxyCertificates: () => apiFetch<ProxyCertificate[]>('/proxy/certificates'),

  proxyMiddlewares: () => apiFetch<ProxyMiddleware[]>('/proxy/middlewares'),

  saveProxyMiddleware: (payload: {
    name: string
    type: string
    config: Record<string, unknown>
  }) => apiFetch<ProxyMiddleware>('/proxy/middlewares', { method: 'POST', body: payload }),

  deleteProxyMiddleware: (name: string) =>
    apiFetch<{ message: string }>(`/proxy/middlewares/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    }),

  proxyLifecycle: (action: 'ensure' | 'start' | 'stop' | 'restart') =>
    apiFetch<{ message: string }>(`/proxy/${action}`, { method: 'POST' }),

  updateProxySettings: (payload: { panel_domain: string; acme_email: string }) =>
    apiFetch<{ message: string }>('/proxy/settings', { method: 'PUT', body: payload }),

  proxyLogs: (tail = 200) => apiFetch<{ lines: string[] }>(`/proxy/logs?tail=${tail}`),

  diskUsage: () => apiFetch<DiskUsage>('/system/disk'),

  prune: (targets: string[]) =>
    apiFetch<PruneResult>('/system/prune', { method: 'POST', body: { targets } }),
}

export function terminalWebSocketURL(target: string, cols: number, rows: number): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'

  return `${protocol}//${window.location.host}/api/v1/terminal/ws?target=${encodeURIComponent(target)}&cols=${cols}&rows=${rows}`
}

export function logStreamURL(applicationId: string, tail: number, container?: string): string {
  const suffix = container ? `&container=${encodeURIComponent(container)}` : ''

  return `/api/v1/applications/${applicationId}/logs/stream?tail=${tail}${suffix}`
}

export function logDownloadURL(applicationId: string, tail: number, container?: string): string {
  const suffix = container ? `&container=${encodeURIComponent(container)}` : ''

  return `/api/v1/applications/${applicationId}/logs/download?tail=${tail}${suffix}`
}

export function resourceLogStreamURL(
  resource: 'applications' | 'databases' | 'services',
  uuid: string,
  tail: number,
  container?: string,
): string {
  const suffix = container ? `&container=${encodeURIComponent(container)}` : ''

  return `/api/v1/${resource}/${uuid}/logs/stream?tail=${tail}${suffix}`
}

export function resourceLogDownloadURL(
  resource: 'applications' | 'databases' | 'services',
  uuid: string,
  tail: number,
  container?: string,
): string {
  const suffix = container ? `&container=${encodeURIComponent(container)}` : ''

  return `/api/v1/${resource}/${uuid}/logs/download?tail=${tail}${suffix}`
}

export function backupDownloadURL(uuid: string, run: number): string {
  return `/api/v1/backups/${uuid}/runs/${run}/download`
}
