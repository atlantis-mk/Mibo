import { useAuthStore } from '@/stores/auth-store'

export type ApiErrorShape = {
  code: string
  message: string
}

type Envelope<T> = {
  request_id: string
  data?: T
  error?: ApiErrorShape
}

export type User = {
  id: number
  username: string
  role: string
  roles: string[]
  avatar_url?: string
  has_pin?: boolean
  requires_pin_change?: boolean
  created_at: string
  updated_at: string
}

export type AdminUser = User

export type AdminRole = {
  id: number
  name: string
  allow_library_tags?: string[]
  deny_library_tags?: string[]
  created_at: string
  updated_at: string
}

export type CreateAdminUserInput = {
  username: string
  password: string
  role: 'user' | 'admin'
  pin?: string
  avatar_url?: string
}

export type UpdateAdminUserInput = {
  username?: string
  password?: string
  role?: 'user' | 'admin'
  pin?: string
  avatar_url?: string
}

export type CreateAdminRoleInput = {
  name: string
  allow_library_tags?: string[]
  deny_library_tags?: string[]
}

export type UpdateAdminRoleInput = {
  name?: string
  allow_library_tags?: string[]
  deny_library_tags?: string[]
}

export type LoginResult = {
  token: string
  expires_at: string
  user: User
}

export type LoginUserProfile = {
  id: number
  username: string
  avatar_url: string
  has_pin: boolean
  updated_at: string
}

export type UserSettingsAppearance = {
  theme: 'system' | 'light' | 'dark'
  locale: string
}

export type UserSettingsPlayback = {
  autoplay_next_episode: boolean
  prefer_direct_play: boolean
  default_subtitle_mode: 'auto' | 'always' | 'never'
  preferred_audio_language: string
  preferred_subtitle_language: string
}

export type UserSettingsSecurity = {
  session_timeout: '12h' | '24h' | '7d' | '30d'
  login_protection_level: 'standard' | 'strict' | 'local_only'
  auto_clear_invalid_token: boolean
  require_dangerous_action_confirmation: boolean
}

export type UserSettings = {
  appearance: UserSettingsAppearance
  playback: UserSettingsPlayback
  security: UserSettingsSecurity
}

export type UserSettingsInput = UserSettings

export type LiveTVRefreshStatus = {
  status: string
  error?: string
}

export type LiveTVSource = {
  id: number
  name: string
  source_type: string
  format_hint: 'auto' | 'm3u' | 'txt'
  url: string
  user_agent: string
  referrer: string
  tuner_count: number
  import_groups: string
  import_guide_data: boolean
  channel_image_source: 'm3u' | 'guide'
  allow_guide_mapping_by_number: boolean
  channel_tags: string
  enabled: boolean
  last_refresh_at?: string
  refresh: LiveTVRefreshStatus
  channel_count: number
  created_at: string
  updated_at: string
}

export type CreateLiveTVSourceInput = {
  name: string
  url: string
  format_hint: 'auto' | 'm3u' | 'txt'
  user_agent?: string
  referrer?: string
  tuner_count?: number
  import_groups?: string
  import_guide_data?: boolean
  channel_image_source?: 'm3u' | 'guide'
  allow_guide_mapping_by_number?: boolean
  channel_tags?: string
  enabled?: boolean
}

export type UpdateLiveTVSourceInput = {
  name?: string
  url?: string
  format_hint?: 'auto' | 'm3u' | 'txt'
  user_agent?: string
  referrer?: string
  tuner_count?: number
  import_groups?: string
  import_guide_data?: boolean
  channel_image_source?: 'm3u' | 'guide'
  allow_guide_mapping_by_number?: boolean
  channel_tags?: string
  enabled?: boolean
}

export type LiveTVChannel = {
  id: number
  source_id: number
  name: string
  group_name: string
  logo_url: string
  tvg_id: string
  tvg_name: string
  tags: string[]
  current_program?: LiveTVProgram
  enabled: boolean
  sort_order: number
  raw_attributes?: Record<string, string>
  created_at: string
  updated_at: string
}

export type LiveTVChannelGroup = {
  name: string
  channel_count: number
}

export type LiveTVProgram = {
  id: number
  source_id?: number
  channel_id: number
  channel_name?: string
  group_name?: string
  guide_channel_id: string
  title: string
  subtitle?: string
  description?: string
  start_at: string
  end_at: string
}

export type LiveTVProgramList = {
  items: LiveTVProgram[]
  total: number
  limit: number
  offset: number
  has_more: boolean
}

export type LiveTVPlaybackSource = {
  channel_id: number
  title: string
  group_name?: string
  logo_url?: string
  tags?: string[]
  current_program?: LiveTVProgram
  type: string
  container: string
  url: string
  direct: boolean
  playable: boolean
  stream_mode: string
}

export type SetupStatus = {
  initialized: boolean
  can_enter_app: boolean
  has_users: boolean
  has_media_sources: boolean
  has_libraries: boolean
  user_count: number
  media_source_count: number
  library_count: number
}

export type SetupDatabaseConnection = {
  driver: 'sqlite' | 'postgres' | 'mysql'
  sqlite_path?: string
  host?: string
  port?: number
  database?: string
  username?: string
  ssl_mode?: string
  password_configured: boolean
}

export type SetupDatabaseState = {
  active_driver: 'sqlite' | 'postgres' | 'mysql'
  active_source: 'default' | 'bootstrap_file' | 'environment'
  active_connection: SetupDatabaseConnection
  draft_connection: SetupDatabaseConnection
  defaults: {
    sqlite_path: string
    postgres_port: number
    mysql_port: number
    ssl_mode: string
  }
  edit_locked: boolean
  edit_lock_reason?: string
  initialization_locked: boolean
  initialization_lock_reason?: string
  restart_required: boolean
}

export type SetupDatabaseInput = {
  driver: 'sqlite' | 'postgres' | 'mysql'
  sqlite_path?: string
  host?: string
  port?: number
  database?: string
  username?: string
  password?: string
  ssl_mode?: string
}

export type SetupDatabaseValidation = {
  valid: boolean
  normalized: SetupDatabaseConnection
  message: string
}

export type SetupDatabaseApplyResult = {
  status: 'unchanged' | 'restarting'
  restart_required: boolean
  normalized: SetupDatabaseConnection
  message: string
}

export type AppSession = {
  setup: SetupStatus
  authenticated: boolean
  user: User | null
}

export type ConsoleStatus =
  | 'ok'
  | 'warning'
  | 'error'
  | 'unknown'
  | 'unavailable'
  | 'not_configured'

export type ConsoleUpdateStatus =
  | 'disabled'
  | 'unknown'
  | 'up_to_date'
  | 'update_available'
  | 'check_failed'

export type ConsoleUpdateSummary = {
  status: ConsoleUpdateStatus
  current_version: string
  latest_version?: string
  release_url?: string
  download_url?: string
  asset_name?: string
  checked_at?: string
  message?: string
  staged?: ConsolePrepareUpdateResult
}

export type ConsoleServerSummary = {
  name: string
  service: string
  status: ConsoleStatus
  version: string
  update_status: ConsoleUpdateStatus
  update: ConsoleUpdateSummary
  api_address: string
  port: number
  uptime_seconds: number
  storage_provider: string
  storage_root: string
  database_driver: string
}

export type ConsoleAccessAddress = {
  kind: 'local' | 'lan' | 'remote'
  label: string
  url?: string
  status: ConsoleStatus | 'available'
  route?: string
  message?: string
  copyable: boolean
}

export type ConsoleMediaSummary = {
  libraries: number
  media_sources: number
  metadata_items: number
  inventory_files: number
  movies: number
  series: number
  episodes: number
  people: number
  active_jobs: number
  failed_jobs: number
  schedules: number
  enabled_schedules: number
  warnings: number
  ingest?: ConsoleIngestSummary
}

export type ConsoleIngestSummary = {
  organizing: number
  failed: number
  stale: number
  review_required: number
  retry_eligible: number
}

export type IngestDiagnosticsResult = {
  summary: ConsoleIngestSummary
  stages: IngestDiagnosticStage[]
}

export type IngestDiagnosticStage = {
  id: number
  unit_key: string
  library_id: number
  library_name?: string
  inventory_file_id?: number
  storage_path?: string
  metadata_item_id?: number
  metadata_item_title?: string
  condition_type: string
  status: string
  reason?: string
  message?: string
  severity?: string
  attempts: number
  job_id?: number
  metadata_operation_id?: number
  retry_eligible: boolean
  stale: boolean
  updated_at: string
  last_transition_at?: string
}

export type IngestRetryResult = {
  condition_id: number
  status: string
  message: string
}

export type IngestResolveReviewResult = {
  condition_id: number
  status: string
  message: string
}

export type ConsoleSectionStatus = {
  status: ConsoleStatus
  message?: string
}

export type ConsoleModuleStatus = {
  name: string
  status: ConsoleStatus
  message?: string
}

export type ConsoleActivityEvent = {
  id: string
  type: string
  severity: 'info' | 'warning' | 'error'
  message: string
  user?: string
  device?: string
  media_title?: string
  timestamp: string
}

export type ConsoleDeviceSummary = {
  id: string
  name: string
  client_type?: string
  user?: string
  state?: string
  media_title?: string
  last_seen_at: string
}

export type ConsoleQuickAction = {
  id: string
  label: string
  description: string
  kind: 'route' | 'mutation' | 'unsupported'
  route?: string
  method?: string
  endpoint?: string
  disabled: boolean
  disabled_reason?: string
  risk: 'safe' | 'expensive' | 'danger'
  confirm: boolean
}

export type ConsoleSectionWarning = {
  section: string
  message: string
}

export type ConsoleSummary = {
  server: ConsoleServerSummary
  access: {
    addresses: ConsoleAccessAddress[]
  }
  media: ConsoleMediaSummary
  health: {
    database: ConsoleSectionStatus
    storage: ConsoleSectionStatus
    modules: ConsoleModuleStatus[]
  }
  devices: ConsoleDeviceSummary[]
  quick_actions: ConsoleQuickAction[]
  activity: ConsoleActivityEvent[]
  warnings: ConsoleSectionWarning[]
}

export type ConsoleActionResult = Record<string, unknown>

export type ConsoleRestartActionResult = {
  status: 'restarting'
  message: string
}

export type ConsolePrepareUpdateResult = {
  status: 'staged'
  current_version: string
  latest_version: string
  asset_name: string
  download_url: string
  staged_directory: string
  staged_binary: string
  sha256: string
  message: string
}

export type ConsoleApplyUpdateResult = {
  status: 'applied'
  previous_binary: string
  backup_binary: string
  applied_binary: string
  latest_version: string
  sha256: string
  restart_required: boolean
  restart_scheduled: boolean
  restart_helper_pid?: number
  message: string
}

export type LoginSession = {
  id: number
  user_agent: string
  remote_addr: string
  device_name: string
  client_type: string
  expires_at: string
  last_used_at?: string
  created_at: string
  updated_at: string
  is_current: boolean
}

export type AdminLogFile = {
  name: string
  modified_at: string
  size_bytes: number
  kind: string
}

export type AdminLogContent = {
  name: string
  content: string
  truncated: boolean
  size_bytes: number
  max_bytes: number
}

export type AdminLogSettings = {
  include_server_logs: boolean
  include_transcode_logs: boolean
  max_preview_bytes: number
}

export type AdminLogSettingsInput = Partial<AdminLogSettings>

export type Library = {
  id: number
  name: string
  media_source_id: number
  root_path: string
  visibility_mode: 'default_open' | 'allow_list_only'
  status: string
  scanner_enabled: boolean
  probe_status: string
  probe_summary_json?: string
  access_tags?: LibraryAccessTag[]
  paths?: LibraryPath[]
  policies?: LibraryPolicies
  probe_summary?: SourceProbeSummary
  collections?: SourceCollection[]
}

export type LibraryDetail = Library & {
  metadata_items_count: number
  inventory_files_count: number
}

export type LibraryAccessTag = {
  id: number
  name: string
}

export type InventoryFileListItem = {
  id: number
  library_id: number
  storage_path: string
  storage_provider: string
  thumbnail_url?: string
  excluded?: boolean
  exclusion_id?: number
  filename_rule_id?: number
}

export type InventoryFileListResponse = {
  items: InventoryFileListItem[]
  total: number
  page: number
  limit: number
  has_more: boolean
}

export type LibraryPath = {
  id: number
  library_id: number
  media_source_id: number
  root_path: string
  display_name: string
  enabled: boolean
}

export type SourceContentClass = 'video' | 'audio' | 'text' | 'image' | 'other'

export type SourceProbeSummary = {
  status: string
  dominant_class: SourceContentClass | ''
  uncertain: boolean
  budget_limited: boolean
  sampled_objects: number
  sampled_files: number
  sampled_dirs: number
  max_objects: number
  max_depth: number
  classes: Record<SourceContentClass, number>
  error?: string
}

export type SourceCollection = {
  content_class: SourceContentClass
  label: string
  count: number
}

export type LibraryScanPolicy = {
  scanner_enabled: boolean
  realtime_monitor_enabled: boolean
  scheduled_refresh_enabled: boolean
  refresh_interval_hours: number
  ignore_hidden_files: boolean
  ignore_file_extensions: string[]
  min_file_size_bytes: number
  sample_ignore_size_bytes: number
  inventory_probe_batch_enabled: boolean
  configurable_exclusion_rules: boolean
}

export type LibraryMetadataPolicy = {
  preferred_metadata_language: string
  local_metadata_enabled: boolean
  metadata_profile_id?: number
  metadata_profile_name?: string
}

export type LibraryMetadataStrategy = {
  library_id: number
  template_profile_id?: number
  template_profile_name?: string
  search_provider_ids: number[]
  search_provider_refs?: string[]
  detail_provider_ids: number[]
  detail_provider_refs?: string[]
  preferred_metadata_language?: string
}

export type LibraryMetadataStrategyInput = {
  template_profile_id?: number
  search_provider_ids: number[]
  search_provider_refs?: string[]
  detail_provider_ids: number[]
  detail_provider_refs?: string[]
  preferred_metadata_language?: string
}

export type LibraryPlaybackPolicy = {
  resume_enabled: boolean
  max_resume_pct: number
  min_resume_duration_seconds: number
}

export type LibrarySubtitlePolicy = {
  external_sidecars_enabled: boolean
  preferred_languages: string[]
  tolerate_unavailable_subtitles: boolean
}

export type LibraryPolicies = {
  scan: LibraryScanPolicy
  metadata: LibraryMetadataPolicy
  playback: LibraryPlaybackPolicy
  subtitle: LibrarySubtitlePolicy
}

export type ScanExclusion = {
  id: number
  library_id: number
  library_name?: string
  storage_provider: string
  stable_identity_key?: string
  storage_path: string
  reason: string
  enabled: boolean
  created_by_user_id?: number
  disabled_at?: string
  disabled_by_user_id?: number
  created_at: string
  updated_at: string
}

export type FilenameExclusionFile = {
  id: number
  storage_path: string
  stable_identity_key?: string
  status: string
  restored: boolean
}

export type FilenameExclusionRule = {
  id: number
  normalized_filename: string
  reason: string
  enabled: boolean
  created_by_user_id?: number
  updated_by_user_id?: number
  disabled_at?: string
  disabled_by_user_id?: number
  created_at: string
  updated_at: string
  affected_count: number
  affected_files: FilenameExclusionFile[]
}

export type ScanExclusionsView = {
  manual_exclusions: ScanExclusion[]
  filename_rules: FilenameExclusionRule[]
}

export type FilenameExclusionPreview = {
  library_id: number
  library_name: string
  storage_provider: string
  normalized_filename: string
  affected_count: number
  affected_files: FilenameExclusionFile[]
}

export type FilenameExclusionRestore = {
  id: number
  rule_id: number
  stable_identity_key?: string
  storage_path: string
  created_by_user_id?: number
  created_at: string
  updated_at: string
}

export type ScanExclusionRule = {
  id: number
  key: string
  library_id?: number
  name: string
  description: string
  rule_type: 'filename_token' | 'directory_segment' | 'path_pattern'
  value: string
  reason: string
  enabled: boolean
  system: boolean
  created_by_user_id?: number
  updated_by_user_id?: number
  disabled_at?: string
  created_at: string
  updated_at: string
}

export type ScanExclusionRuleInput = {
  library_id?: number
  name: string
  description?: string
  rule_type: ScanExclusionRule['rule_type']
  value: string
  reason: string
  enabled?: boolean
}

export type OpenListMediaSourceConfig = {
  base_url: string
  username?: string
  password?: string
  token?: string
  timeout?: string
  scan_interval?: string
  insecure_skip?: boolean
}

export type MediaSourceConfigInput = {
  openlist?: OpenListMediaSourceConfig
}

export type OpenListMediaSourceConfigView = {
  base_url: string
  username?: string
  timeout?: string
  scan_interval?: string
  insecure_skip: boolean
  has_password: boolean
  has_token: boolean
}

export type MediaSourceConfigView = {
  openlist?: OpenListMediaSourceConfigView
}

export type PluginProtocolVersion = '1.0' | string

export type PluginDeploymentKind = 'remote' | 'local_companion' | string

export type PluginCapability =
  | 'metadata.search'
  | 'metadata.detail'
  | 'subtitle.search'
  | 'storage.browse'
  | 'storage.resolve'
  | 'storage.link'
  | string

export type PluginHealthAvailability =
  | 'available'
  | 'unavailable'
  | 'cooldown'
  | string

export type PluginOperationEndpoint = {
  path: string
  method?: string
  timeout?: string
  authenticated?: boolean
}

export type PluginCapabilityDeclaration = {
  capability: PluginCapability
  endpoint: PluginOperationEndpoint
}

export type PluginConfigurationFieldType =
  | 'string'
  | 'secret'
  | 'number'
  | 'boolean'
  | 'select'
  | 'url'
  | 'duration'
  | string

export type PluginConfigurationFieldDisplay = {
  label?: string
  description?: string
  help_text?: string
  placeholder?: string
}

export type PluginConfigurationSelectOption = {
  value: string
  label?: string
}

export type PluginConfigurationField = {
  key: string
  type: PluginConfigurationFieldType
  required?: boolean
  default?: unknown
  display?: PluginConfigurationFieldDisplay
  options?: PluginConfigurationSelectOption[]
  minimum?: number
  maximum?: number
}

export type PluginConfigurationSchema = {
  fields?: PluginConfigurationField[]
}

export type PluginManifest = {
  id: string
  name: string
  version: string
  protocol_version: PluginProtocolVersion
  description?: string
  homepage_url?: string
  capabilities?: PluginCapabilityDeclaration[]
  health: {
    path: string
  }
  configuration_schema?: PluginConfigurationSchema
}

export type PluginProviderInstance = {
  id: number
  name: string
  deployment_kind: PluginDeploymentKind
  endpoint: string
  plugin_id: string
  plugin_name: string
  plugin_version: string
  protocol_version: PluginProtocolVersion
  capabilities: PluginCapability[]
  enabled: boolean
  availability_status: PluginHealthAvailability
  failure_reason?: string
  cooldown_until?: string
  last_checked_at?: string
  manifest: PluginManifest
  configuration?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export type PluginUsageReference = {
  kind: 'metadata_profile' | 'library_metadata_strategy' | 'media_source'
  id: number
  name: string
  description?: string
  stage?: string
}

export type PluginUsageSummary = {
  provider_instance_id: number
  metadata_profiles: PluginUsageReference[]
  library_metadata_strategies: PluginUsageReference[]
  media_sources: PluginUsageReference[]
  active_reference_count: number
}

export type PluginProviderDetail = {
  instance: PluginProviderInstance
  usage: PluginUsageSummary
}

export type InternalPlugin = {
  id: string
  name: string
  kind: 'metadata' | 'storage' | 'subtitle' | string
  provider_key: string
  provider_ref?: string
  description?: string
  capabilities: PluginCapability[]
  enabled: boolean
  availability_status: PluginHealthAvailability
  configured: boolean
  local_subtitle?: {
    external_file_enabled: boolean
    embedded_extraction_enabled: boolean
  }
  usage?: PluginUsageReference[]
}

export type OpenSubtitlesSettings = {
  configured: boolean
  api_key_masked: boolean
  api_key_count: number
  base_url: string
  languages: string
  timeout: string
  source: string
}

export type OpenSubtitlesSettingsInput = {
  api_key?: string
  api_keys?: string
  clear_api_key?: boolean
  base_url: string
  languages: string
  timeout: string
}

export type OpenSubtitlesProviderSettingsInput = {
  api_key?: string
  clear_api_key?: boolean
  base_url: string
  languages: string
  timeout: string
}

export type GeneralConfigSettings = {
  http: {
    shutdown_timeout: string
  }
  web: {
    dist_dir: string
  }
  cors: {
    allowed_origins: string
  }
  access: {
    disable_library_visibility_enforcement: boolean
  }
  ffmpeg: {
    enabled: boolean
    path: string
    timeout: string
    artwork_root_path: string
    transcode_root_path: string
    transcode_idle_timeout: string
  }
  ffprobe: {
    enabled: boolean
    path: string
    timeout: string
  }
  worker: {
    enabled: boolean
    poll_interval: string
    probe_workers: number
    scan_directory_workers: number
    workflow_poll_interval: string
    workflow_lease_duration: string
    workflow_task_timeout: string
    workflow_max_concurrent: number
  }
  effective_status: {
    source: string
    restart_required: boolean
    message: string
  }
}

export type GeneralConfigInput = Omit<GeneralConfigSettings, 'effective_status'>

export type SubtitleProviderInstance = {
  id: number
  name: string
  provider_type: 'opensubtitles' | string
  system_managed: boolean
  locked: boolean
  enabled: boolean
  availability_status: string
  failure_reason?: string
  cooldown_until?: string
  configured: boolean
  opensubtitles?: OpenSubtitlesSettings
}

export type SubtitleProviderInstanceInput = {
  name: string
  provider_type: 'opensubtitles' | string
  enabled?: boolean
  availability_status?: string
  failure_reason?: string
  cooldown_until?: string
  opensubtitles?: OpenSubtitlesProviderSettingsInput
}

export type LocalPluginInstallation = {
  id: number
  plugin_id: string
  name: string
  version: string
  source_kind: string
  source: string
  install_path?: string
  enabled: boolean
  install_state: string
  process_state: string
  resolved_endpoint?: string
  failure_reason?: string
  installed_at: string
  updated_at: string
  started_at?: string
  stopped_at?: string
}

export type LocalPluginInstallInput = {
  name?: string
  plugin_id: string
  version?: string
  source_kind?: string
  source: string
  install_path?: string
  enabled?: boolean
}

export type PluginCatalogSource = {
  id: number
  name: string
  url: string
  trust_level: string
  enabled: boolean
  last_sync_status?: string
  last_sync_at?: string
  created_at: string
  updated_at: string
}

export type PluginCompatibilityResult = {
  compatible: boolean
  reasons: string[]
}

export type PluginCatalogEntry = {
  id: number
  source_id: number
  plugin_id: string
  name: string
  version: string
  protocol_version: string
  mibo_version_range?: string
  platforms?: string[]
  capabilities?: PluginCapability[]
  checksum?: string
  signature_status?: string
  homepage_url?: string
  release_notes?: string
  trust_level?: string
  compatibility?: PluginCompatibilityResult
  created_at: string
  updated_at: string
}

export type PluginCatalogOverview = {
  sources: PluginCatalogSource[]
  entries: PluginCatalogEntry[]
}

export type RemotePluginProviderInput = {
  name?: string
  endpoint: string
  configuration?: Record<string, unknown>
  enabled?: boolean
}

export type MediaSource = {
  id: number
  name: string
  provider: string
  provider_label?: string
  root_path: string
  config?: MediaSourceConfigView
  capabilities_json: string
  plugin_provider?: PluginProviderInstance
  created_at: string
  updated_at: string
}

export type StorageBrowseItem = {
  name: string
  path: string
  is_dir: boolean
  size: number
  modified?: string
}

export type StorageBrowseResult = {
  provider: string
  root_path: string
  current_path: string
  parent_path?: string
  items: StorageBrowseItem[]
}

export type OpenListTestResult = {
  status: string
  provider: string
  message: string
  root_path: string
}

export type Track = {
  codec: string
  language: string
  title: string
  channels?: number
  file_id?: number
  stream_index?: number
  url?: string
  external?: boolean
  available?: boolean
  provider_id?: number
  provider_name?: string
}

export type SubtitleSearchProvider = {
  id: number
  name: string
  provider_type: string
}

export type SubtitleSearchResult = {
  provider: SubtitleSearchProvider
  tracks: Track[]
}

export type CatalogSelectedImage = {
  id?: number
  inventory_file_id?: number
  image_type: string
  url: string
  language?: string
  width?: number
  height?: number
}

export type CatalogMetadataVideo = {
  provider?: string
  site: string
  key: string
  name?: string
  type?: string
  official?: boolean
  language?: string
  published?: string
  watch_url?: string
  embed_url?: string
  thumbnail?: string
  sort_order?: number
}

export type CatalogExternalIdentity = {
  provider: string
  provider_type: string
  external_id: string
  is_primary: boolean
  source?: string
  confidence?: number
}

export type CatalogSourceEvidence = {
  source_type: string
  source_name: string
  language?: string
  external_id?: string
  confidence?: number
  fetched_at: string
  expires_at?: string
  summary?: unknown
}

export type CatalogFieldState = {
  field_key: string
  source_id?: number
  value?: unknown
  is_locked: boolean
  lock_reason?: string
  edited_by_user_id?: number
  edited_at?: string
}

export type CatalogChildSummary = {
  child_count: number
  available_count: number
  missing_count: number
  unaired_count: number
  played_count: number
  in_progress_count: number
  latest_air_date?: string
  latest_added_at?: string
}

export type MediaResourceLink = {
  metadata_item_id: number
  role: string
  segment_index: number
  start_seconds?: number
  end_seconds?: number
  confidence?: number
  source?: string
}

export type CatalogEpisodeParentContext = {
  series?: {
    id: number
    title: string
    selected_images?: CatalogSelectedImage[]
  }
  season?: {
    id: number
    title: string
    number?: number
    selected_images?: CatalogSelectedImage[]
  }
  season_number?: number
  episode_number?: number
  episode_number_end?: number
  incomplete_hierarchy: boolean
}

export type MediaResourceFileSummary = {
  file_id: number
  role: string
  part_index: number
  storage_provider: string
  storage_path?: string
  stable_identity_key?: string
  size_bytes: number
  container?: string
  status: string
  modified_at?: string
}

export type CatalogMediaStreamSummary = {
  file_id: number
  stream_index: number
  stream_type: string
  codec?: string
  profile?: string
  level?: number
  language?: string
  title?: string
  width?: number
  height?: number
  avg_frame_rate?: string
  r_frame_rate?: string
  field_order?: string
  color_space?: string
  bit_depth?: number
  pixel_format?: string
  reference_frames?: number
  channels?: number
  channel_layout?: string
  sample_rate?: number
  bit_rate?: number
  duration_seconds?: number
  default?: boolean
  forced?: boolean
  hearing_impaired?: boolean
  external?: boolean
}

export type CatalogPersonDetail = {
  id?: number
  name: string
  role?: string
  avatar_url?: string
}

export type CatalogPersonPageDetail = {
  id: number
  name: string
  sort_name?: string
  avatar_url?: string
  biography?: string
  birthday?: string
  deathday?: string
  place_of_birth?: string
  known_for_department?: string
  external_identities?: CatalogExternalIdentity[]
  related_items?: CatalogListItem[]
}

export type CatalogTagDetail = {
  kind: string
  name: string
}

export type CatalogListItem = {
  id: number
  metadata_item_id?: number
  library_id: number
  resource_count?: number
  available_count?: number
  missing_count?: number
  source_kind?: 'catalog' | 'inventory_file'
  inventory_file_id?: number
  maturity_state?:
    | 'discovered'
    | 'classified'
    | 'enriched'
    | 'review_required'
    | string
  organizing?: boolean
  organizing_summary?: CatalogOrganizingSummary
  storage_path?: string
  type: string
  title: string
  original_title?: string
  local_title?: string
  overview?: string
  year?: number
  end_year?: number
  runtime_seconds?: number
  index_number?: number
  index_number_end?: number
  parent_index_number?: number
  episode_label?: string
  community_rating?: number
  official_rating?: string
  series_status?: string
  availability_status: string
  governance_status: string
  release_date?: string
  first_air_date?: string
  last_air_date?: string
  child_summary?: CatalogChildSummary
  selected_images?: CatalogSelectedImage[]
  external_identities?: CatalogExternalIdentity[]
  tags?: CatalogTagDetail[]
  genres?: string[]
  directors?: CatalogPersonDetail[]
}

export type CatalogOrganizingSummary = {
  state:
    | 'organizing'
    | 'partial_ready'
    | 'ready'
    | 'failed'
    | 'review_required'
    | string
  message: string
  stage?: string
  severity?: 'info' | 'warning' | 'error' | string
  conditions?: CatalogOrganizingCondition[]
}

export type CatalogOrganizingCondition = {
  type: string
  status: string
  reason?: string
  message?: string
  severity?: string
}

export type MediaResourceDetail = {
  id: number
  resource_id?: number
  library_id: number
  resource_type: string
  file_name?: string
  token_title?: string
  edition?: string
  duration_seconds?: number
  status: string
  probe_status: string
  file_ids: number[]
  files?: MediaResourceFileSummary[]
  streams?: CatalogMediaStreamSummary[]
  links: MediaResourceLink[]
}

export type MetadataResourceDetail = {
  id: number
  library_id?: number
  resource_type: string
  resource_shape: string
  file_name?: string
  token_title?: string
  edition?: string
  duration_seconds?: number
  size_bytes?: number
  status: string
  probe_status: string
  role: string
  segment_index?: number
  review_state?: string
}

export type ResourceMetadataLinkInput = {
  target_metadata_item_id?: number
  source_metadata_item_id?: number
  library_id?: number
  mode?: 'copy' | 'move'
  role?: string
  segment_index?: number
  start_seconds?: number
  end_seconds?: number
}

export type ResourceMetadataLinkUpdateInput = {
  library_id?: number
  role?: string
  segment_index?: number
  new_role?: string
  review_state?: string
}

export type MetadataMergeInput = {
  target_metadata_item_id: number
  library_id?: number
}

export type MetadataSplitInput = {
  target_metadata_item_id: number
  resource_ids: number[]
  library_id?: number
}

export type ProjectionVisibilityInput = {
  library_id: number
  hidden: boolean
}

export type CatalogEpisodeShelfItem = {
  id: number
  library_id: number
  type: string
  title: string
  label?: string
  overview?: string
  season_number?: number
  episode_number?: number
  episode_number_end?: number
  runtime_seconds?: number
  inventory_file_id?: number
  availability_status: string
  governance_status: string
  release_date?: string
  first_air_date?: string
  selected_images?: CatalogSelectedImage[]
  external_identities?: CatalogExternalIdentity[]
  current: boolean
  progress?: ProgressState
}

export type CatalogEpisodeDetail = {
  id: number
  library_id: number
  type: string
  title: string
  overview?: string
  year?: number
  parent_index_number?: number
  index_number?: number
  index_number_end?: number
  absolute_number?: number
  runtime_seconds?: number
  inventory_file_id?: number
  availability_status: string
  governance_status: string
  release_date?: string
  first_air_date?: string
  selected_images?: CatalogSelectedImage[]
  external_identities?: CatalogExternalIdentity[]
  source_evidence?: CatalogSourceEvidence[]
  field_states?: CatalogFieldState[]
  resources?: MediaResourceDetail[]
}

export type CatalogSeasonDetail = {
  id: number
  library_id: number
  type: string
  title: string
  overview?: string
  year?: number
  index_number?: number
  runtime_seconds?: number
  availability_status: string
  governance_status: string
  child_summary?: CatalogChildSummary
  selected_images?: CatalogSelectedImage[]
  external_identities?: CatalogExternalIdentity[]
  source_evidence?: CatalogSourceEvidence[]
  field_states?: CatalogFieldState[]
  episodes?: CatalogEpisodeDetail[]
}

export type CatalogSeriesPlaybackTarget = {
  episode_metadata_item_id: number
  resource_id?: number
  title: string
  label?: string
  selection_reason: string
}

export type CatalogItemDetail = {
  id: number
  metadata_item_id?: number
  library_id: number
  resource_count?: number
  available_count?: number
  missing_count?: number
  type: string
  title: string
  original_title?: string
  local_title?: string
  overview?: string
  year?: number
  end_year?: number
  runtime_seconds?: number
  community_rating?: number
  official_rating?: string
  series_status?: string
  availability_status: string
  governance_status: string
  release_date?: string
  first_air_date?: string
  last_air_date?: string
  child_summary?: CatalogChildSummary
  selected_images?: CatalogSelectedImage[]
  image_candidates?: CatalogSelectedImage[]
  videos?: CatalogMetadataVideo[]
  external_identities?: CatalogExternalIdentity[]
  tags?: CatalogTagDetail[]
  genres?: string[]
  source_evidence?: CatalogSourceEvidence[]
  field_states?: CatalogFieldState[]
  cast?: CatalogPersonDetail[]
  directors?: CatalogPersonDetail[]
  seasons?: CatalogSeasonDetail[]
  episodes?: CatalogEpisodeDetail[]
  episode_context?: CatalogEpisodeParentContext
  series_playback_target?: CatalogSeriesPlaybackTarget
  same_season_episodes?: CatalogEpisodeShelfItem[]
  resources?: MediaResourceDetail[]
  related_items?: CatalogListItem[]
}

export type CatalogMetadataPlanProviderSummary = {
  id: number
  name: string
  provider_type: string
  enabled: boolean
  configured: boolean
  operational: boolean
  availability_status?: string
  cooldown_until?: string
}

export type CatalogMetadataExecutionPlanSummary = {
  library_id: number
  strategy_id: number
  metadata_profile_id?: number
  metadata_profile_name?: string
  preferred_metadata_language?: string
  search_providers?: CatalogMetadataPlanProviderSummary[]
  detail_providers?: CatalogMetadataPlanProviderSummary[]
  local_evidence_enabled: boolean
}

export type CatalogMetadataProviderAttempt = {
  stage: string
  provider_instance_id: number
  provider_instance_name: string
  provider_type: string
  outcome: string
  error_class?: string
  error_message?: string
  status_code?: number
  candidate_count?: number
  selected: boolean
}

export type CatalogMetadataAppliedField = {
  metadata_item_id: number
  field_key: string
  source_id?: number
  apply_mode: string
  confidence?: number
}

export type CatalogMetadataSkippedField = {
  metadata_item_id: number
  field_key: string
  reason: string
}

export type CatalogMetadataOperationWarning = {
  code: string
  message: string
}

export type CatalogMetadataAffectedScope = {
  metadata_item_ids?: number[]
  library_id: number
  metadata_root_id?: number
}

export type CatalogMetadataOperation = {
  operation: string
  origin_metadata_item_id?: number
  target_metadata_item_id?: number
  target_type: string
  status: string
  governance_status?: string
  plan: CatalogMetadataExecutionPlanSummary
  provider_attempts?: CatalogMetadataProviderAttempt[]
  metadata_source_ids?: number[]
  applied_fields?: CatalogMetadataAppliedField[]
  skipped_fields?: CatalogMetadataSkippedField[]
  affected_scope: CatalogMetadataAffectedScope
  warnings?: CatalogMetadataOperationWarning[]
}

export type CatalogClassificationEvidence = {
  kind: string
  source?: string
  value?: string
  weight?: number
}

export type CatalogClassificationAlternative = {
  type: string
  role?: string
  target_kind?: string
  target_key?: string
  confidence?: number
  reason?: string
}

export type CatalogClassificationCorrection = {
  action: string
  label: string
  description?: string
}

export type CatalogClassificationDecision = {
  id: number
  source_path: string
  decision_type: string
  role?: string
  candidate_type?: string
  target_kind?: string
  target_key?: string
  status: string
  confidence?: number
  alternatives: CatalogClassificationAlternative[]
  evidence: CatalogClassificationEvidence[]
  affected_files: string[]
  correction_actions: CatalogClassificationCorrection[]
  reason?: string
  warnings: string[]
  created_at: string
  updated_at: string
  resolved_at?: string
}

export type CatalogClassificationRuleSummary = {
  id: number
  library_id: number
  key: string
  name: string
  path_pattern: string
  rule_type: string
  role?: string
  candidate_type?: string
  series_title?: string
  season_number?: number
  numbering_source?: string
  enabled: boolean
}

export type CatalogGovernanceWorkspace = {
  metadata_item_id: number
  library_id: number
  type: string
  title: string
  original_title?: string
  local_title?: string
  overview?: string
  year?: number
  release_date?: string
  availability_status: string
  governance_status: string
  selected_images?: CatalogSelectedImage[]
  image_candidates?: CatalogSelectedImage[]
  videos?: CatalogMetadataVideo[]
  external_identities?: CatalogExternalIdentity[]
  review_candidates?: CatalogMetadataSearchCandidate[]
  source_evidence?: CatalogSourceEvidence[]
  field_states?: CatalogFieldState[]
  resources?: MediaResourceDetail[]
  classification_decisions?: CatalogClassificationDecision[]
  classification_rules?: CatalogClassificationRuleSummary[]
  recommended_children?: CatalogListItem[]
  metadata_operation?: CatalogMetadataOperation
}

export type CatalogMetadataSearchCandidate = {
  provider: string
  media_type: string
  external_id: string
  title: string
  original_title: string
  overview: string
  poster_url: string
  backdrop_url: string
  release_date: string
  year?: number
  confidence: number
  matched_query?: string
  matched_language?: string
  match_score?: number
  auto_match_eligible?: boolean
  match_score_breakdown?: {
    title_score: number
    year_score: number
    media_type_score: number
    sequence_score: number
    search_signal_score: number
    uniqueness_score: number
    total: number
    hard_block?: string
    reason?: string
  }
  reason_summary?: string
}

export type CatalogMetadataSearchResponse = {
  candidates: CatalogMetadataSearchCandidate[]
}

export type ProgressState = {
  user_id: number
  metadata_item_id?: number
  resource_id?: number
  preferred_resource_id?: number
  position_seconds: number
  duration_seconds?: number
  played_percentage?: number
  progress_frame_url?: string
  play_count?: number
  watched: boolean
  favorite?: boolean
  completed_at?: string
  last_played_at?: string
}

export type CatalogUserItemEntry = ProgressState & {
  favorite: boolean
  item: CatalogListItem
  display_item?: CatalogListItem
  play_item?: CatalogListItem
}

export type HomeContentSection = {
  key: string
  title: string
  items: CatalogListItem[]
}

export type HomeMediaSectionSummary = {
  key: string
  title: string
  count: number
  items: CatalogListItem[]
}

export type HomeMediaOverview = {
  sections: HomeMediaSectionSummary[]
}

export type HealthSeverity = 'info' | 'warning' | 'error' | 'blocking'

export type OperationsStatus = 'healthy' | 'attention' | 'degraded' | 'blocked'

export type OperationsTaskKind =
  | 'storage_access_required'
  | 'scan_blocked'
  | 'classification_review_required'
  | 'metadata_review_required'
  | 'projection_stale'
  | 'maintenance_backlog'

export type OperationsActionType =
  | 'open_url'
  | 'validate_media_source'
  | 'scan_library'
  | 'retry_ingest_stage'
  | 'retry_probe_file'
  | 'resolve_review_stage'

export type OperationsIssueKind =
  | 'metadata'
  | 'classification'
  | 'probe'
  | 'workflow'
  | 'storage'
  | 'projection'

export type OperationsIssueScopeKind =
  | 'library'
  | 'media_source'
  | 'folder'
  | 'inventory_file'
  | 'resource'
  | 'metadata_item'
  | 'series'
  | 'season'
  | 'episode'

export type OperationsIssueLifecycleStatus =
  | 'active'
  | 'in_progress'
  | 'resolved'
  | 'reopened'
  | 'ignored'

export type OperationsIssueActionType =
  | 'retry'
  | 'apply_candidate'
  | 'mark_governed'
  | 'accept_classification'
  | 'correct_classification'
  | 'relink_resource'
  | 'unlink_resource'
  | 'exclude'
  | 'ignore'

export type OperationsOverview = {
  status: OperationsStatus
  affected_libraries: number
  affected_files: number
  affected_items: number
  pending_reviews: number
  active_jobs: number
  failed_jobs: number
  sections: OperationsSectionStatus[]
}

export type OperationsSectionStatus = {
  key: string
  label: string
  status: OperationsStatus
  count: number
  description: string
}

export type OperationsTask = {
  id: string
  kind: OperationsTaskKind
  lifecycle_status?: 'active' | 'resolved'
  severity: HealthSeverity
  title: string
  summary: string
  impact: OperationsImpact
  affected: OperationsAffected
  recommended_actions: OperationsRecommendedAction[]
  evidence: OperationsEvidence[]
  first_seen_at?: string
  last_seen_at?: string
}

export type OperationsTaskListResult = {
  items: OperationsTask[]
  total: number
  page: number
  page_size: number
}

export type OperationsImpact = {
  blocks_scan: boolean
  blocks_home_visibility: boolean
  blocks_playback: boolean
  affected_libraries: number
  affected_files: number
  affected_items: number
}

export type OperationsAffected = {
  media_sources: OperationsMediaSourceRef[]
  libraries: OperationsLibraryRef[]
  files: OperationsInventoryRef[]
  items: OperationsMetadataRef[]
}

export type OperationsMediaSourceRef = {
  id: number
  name: string
  provider: string
  root_path: string
  admin_url?: string
}

export type OperationsLibraryRef = {
  id: number
  name: string
  type: string
  status: string
  media_source_id: number
  root_path: string
}

export type OperationsInventoryRef = {
  id: number
  library_id: number
  storage_path: string
  scan_state: string
}

export type OperationsMetadataRef = {
  id: number
  title: string
  type: string
}

export type OperationsRecommendedAction = {
  id?: string
  type: OperationsActionType
  label: string
  description?: string
  href?: string
}

export type OperationsEvidence = {
  kind: string
  label: string
  description?: string
  value?: string
}

export type OperationsPipeline = {
  stages: OperationsPipelineStage[]
}

export type OperationsPipelineStage = {
  key: string
  label: string
  status: OperationsStatus
  pending: number
  running: number
  failed: number
  stale: number
  review_required: number
  retry_eligible: number
  samples: OperationsPipelineStageEntry[]
}

export type OperationsPipelineStageEntry = {
  id: number
  condition_type: string
  status: string
  reason?: string
  message?: string
  severity?: string
  library_id: number
  library_name?: string
  inventory_file_id?: number
  metadata_item_id?: number
  updated_at: string
  last_transition_at?: string
}

export type OperationsActionResult = {
  action_id: string
  status: string
  message: string
  results?: OperationsActionTargetResult[]
}

export type OperationsActionTargetResult = {
  target_type: string
  target_key: string
  status: string
  message?: string
}

export type OperationsIssue = {
  id: number
  fingerprint: string
  library_id: number
  kind: OperationsIssueKind
  scope_kind: OperationsIssueScopeKind
  scope_key: string
  lifecycle_status: OperationsIssueLifecycleStatus
  severity: HealthSeverity
  title: string
  summary: string
  occurrence_count: number
  target_count: number
  impact: OperationsImpact
  library?: OperationsLibraryRef
  targets?: OperationsIssueTarget[]
  occurrences?: OperationsIssueOccurrence[]
  actions?: OperationsIssueAction[]
  events?: OperationsIssueEvent[]
  samples?: OperationsIssueTarget[]
  first_seen_at?: string
  last_seen_at?: string
  resolved_at?: string
}

export type OperationsIssueTarget = {
  id: number
  target_type: string
  target_key: string
  library_id: number
  media_source_id?: number
  inventory_file_id?: number
  resource_id?: number
  metadata_item_id?: number
  parent_metadata_id?: number
  root_metadata_id?: number
  label?: string
  description?: string
  sample_rank: number
  count_hint: number
}

export type OperationsIssueOccurrence = {
  id: number
  source_type: string
  source_key: string
  library_id: number
  inventory_file_id?: number
  resource_id?: number
  metadata_item_id?: number
  workflow_task_id?: number
  ingest_condition_id?: number
  status?: string
  reason?: string
  message?: string
  observed_at?: string
  resolved_at?: string
}

export type OperationsIssueAction = {
  id: number
  action_type: OperationsIssueActionType
  action_key: string
  label: string
  description?: string
  eligible: boolean
  target_count: number
  confirmation_message?: string
  parameters_json?: string
  sort_order: number
}

export type OperationsIssueEvent = {
  id: number
  event_type: string
  action_type?: OperationsIssueActionType
  user_id?: number
  status?: string
  message?: string
  details_json?: string
  created_at?: string
}

export type OperationsIssueListResult = {
  items: OperationsIssue[]
  total: number
  page: number
  page_size: number
}

export type ExecuteIssueActionInput = {
  action_key: string
  external_id?: string
  reason?: string
  confirmation?: boolean
  target_metadata_item_id?: number
  source_metadata_item_id?: number
  resource_id?: number
  metadata_item_id?: number
  role?: string
  mode?: string
  segment_index?: number
  start_seconds?: number
  end_seconds?: number
  classification_target_kind?: string
  classification_target_key?: string
  classification_role?: string
  classification_file_ids?: number[]
}

export type MediaSourceValidationResult = {
  media_source_id: number
  status: string
  message: string
}

export type MetadataProviderSettings = {
  configured: boolean
  api_key_masked: boolean
  base_url: string
  image_base_url?: string
  language: string
  timeout: string
  retry_count: number
  source: string
  implementation: string
  upstream_provider_filter?: string
  fallback_enabled?: boolean
}

export type MetadataProviderInstance = {
  id: number
  name: string
  provider_type: string
  system_managed: boolean
  locked: boolean
  enabled: boolean
  availability_status: string
  failure_reason?: string
  cooldown_until?: string
  configured: boolean
  tmdb?: MetadataProviderSettings
  tvdb?: MetadataProviderSettings
  metatube?: MetadataProviderSettings
}

export type MetadataProviderInstanceInput = {
  name: string
  provider_type: string
  enabled?: boolean
  availability_status?: string
  failure_reason?: string
  cooldown_until?: string
  tmdb?: MetadataProviderInput
  tvdb?: MetadataProviderInput
  metatube?: MetadataProviderInput
}

export type MetadataProfile = {
  id: number
  name: string
  description?: string
  system: boolean
  locked: boolean
  search_provider_ids: number[]
  search_provider_refs?: string[]
  detail_provider_ids: number[]
  detail_provider_refs?: string[]
  preferred_metadata_language?: string
  fallback_enabled: boolean
}

export type MetadataProfileInput = {
  name: string
  description?: string
  search_provider_ids: number[]
  search_provider_refs?: string[]
  detail_provider_ids: number[]
  detail_provider_refs?: string[]
  preferred_metadata_language?: string
  fallback_enabled?: boolean
}

export type MetadataProviderInput = {
  api_key?: string
  clear_api_key?: boolean
  base_url?: string
  image_base_url?: string
  language?: string
  timeout?: string
  retry_count?: number
  upstream_provider_filter?: string
  fallback_enabled?: boolean
}

export type NetworkCertificatePasswordState = {
  configured: boolean
  masked: boolean
}

export type NetworkSettingsStatus = {
  source: string
  restart_required_fields: string[]
  future_runtime_fields: string[]
  automatic_port_mapping_active: boolean
  message: string
}

export type NetworkSettings = {
  local_networks: string[]
  local_ip_address: string
  local_http_port: number
  local_https_port: number
  allow_remote_access: boolean
  remote_ip_filter: string[]
  remote_ip_filter_mode: 'allow' | 'block'
  public_http_port: number
  public_https_port: number
  external_domain: string
  trust_proxy_headers: boolean
  ssl_certificate_path: string
  certificate_password: NetworkCertificatePasswordState
  secure_connection_mode: 'disabled' | 'preferred' | 'required'
  automatic_port_mapping: boolean
  max_video_streams: 'unlimited' | '1' | '2' | '4' | '8'
  remote_streaming_bitrate_limit:
    | 'unlimited'
    | '4mbps'
    | '8mbps'
    | '12mbps'
    | '20mbps'
  network_request_protocol: 'auto' | 'ipv4' | 'ipv6'
  effective_status: NetworkSettingsStatus
}

export type NetworkSettingsInput = Omit<
  NetworkSettings,
  'certificate_password' | 'effective_status'
> & {
  certificate_password?: string
  clear_certificate_password?: boolean
}

export type CatalogDiscoverySort =
  | 'recent'
  | 'imdb_rating'
  | 'last_episode_release_date'
  | 'last_episode_added_date'
  | 'added_date'
  | 'release_date'
  | 'parental_rating'
  | 'director'
  | 'year'
  | 'critic_rating'
  | 'played_date'
  | 'runtime'
  | 'title'
  | 'random'
  | 'audience_rating'
  | 'watch_status'

export type DiscoveryQuery = {
  scope?: 'all' | 'library'
  library_id?: number
  q?: string
  type?: 'all' | 'movie' | 'show' | 'episode'
  genre?: string
  region?: string
  year?: number
  min_rating?: number
  watched_state?: 'all' | 'unwatched' | 'in_progress' | 'watched'
  organizing_state?: 'all' | 'organized' | 'unorganized'
  sort?: CatalogDiscoverySort
  sort_direction?: 'asc' | 'desc'
  limit?: number
  offset?: number
}

export type CatalogDiscoveryResult = CatalogListItem

export type CatalogDiscoveryResponse = {
  items: CatalogListItem[]
  total: number
  limit: number
  offset: number
  has_more: boolean
  sort: CatalogDiscoverySort
  sort_direction: 'asc' | 'desc'
}

export type LibraryHierarchyQuery = Omit<DiscoveryQuery, 'scope'> & {
  path?: string
}

export type LibraryHierarchyNodeKind = 'library' | 'folder' | 'item'

export type LibraryHierarchyContext = {
  node_kind: LibraryHierarchyNodeKind
  node_id: string
  library_id?: number
  library_name?: string
  path?: string
  parent_node_id?: string
}

export type LibraryHierarchyBreadcrumb = {
  node_kind: LibraryHierarchyNodeKind
  node_id: string
  library_id?: number
  library_name?: string
  path?: string
  title: string
}

export type LibraryHierarchyNode = {
  node_kind: LibraryHierarchyNodeKind
  node_id: string
  parent_node_id?: string
  library_id: number
  library_name?: string
  path?: string
  title: string
  child_count?: number
  item_count?: number
  item?: CatalogListItem
}

export type LibraryHierarchyResponse = {
  items: LibraryHierarchyNode[]
  total: number
  limit: number
  offset: number
  has_more: boolean
  current_node: LibraryHierarchyContext
  breadcrumbs: LibraryHierarchyBreadcrumb[]
}

export type SearchHistoryEntry = {
  id: number
  query: string
  type_filter: string
  genre: string
  region: string
  year?: number
  min_rating?: number
  watched_state: string
  sort: CatalogDiscoverySort
  last_used_at: string
}

export type ClientProfile = 'web' | 'mobile' | 'tv'

export type PlaybackCheck = {
  code: string
  status: string
  message: string
}

export type DecisionReason = {
  code: string
  category: string
  message: string
}

export type PlaybackDecision = {
  kind: 'direct' | 'fallback' | 'unplayable'
  client_profile: ClientProfile
  selected_by: string
  fallback_kind?: string
  reasons: DecisionReason[]
}

export type PlaybackPart = {
  part_index: number
  file_id: number
  title: string
  container: string
  url: string
  direct: boolean
  size_bytes: number
  duration_seconds?: number
}

export type PlaybackVariantKind =
  | 'original'
  | 'audio-repair'
  | 'quality'
  | 'unavailable'

export type PlaybackVariant = {
  id: string
  label: string
  kind: PlaybackVariantKind
  available: boolean
  requires_ffmpeg: boolean
  width?: number
  height?: number
  video_bitrate?: string
  audio_codec?: string
  video_codec?: string
  reason?: string
}

export type SelectedPlaybackVariant = {
  id: string
  label: string
  kind: PlaybackVariantKind
  hls: boolean
  manifest?: string
}

export type PlaybackSource = {
  metadata_item_id?: number
  resource_id?: number
  file_id?: number
  title: string
  type: string
  container: string
  url: string
  direct: boolean
  size_bytes: number
  runtime_seconds?: number
  segment_index?: number
  start_seconds?: number
  end_seconds?: number
  edition?: string
  video_codec: string
  width?: number
  height?: number
  probe_status?: string
  audio_tracks: Track[]
  subtitle_tracks: Track[]
  subtitle_search_providers?: SubtitleSearchProvider[]
  parts?: PlaybackPart[]
  variants?: PlaybackVariant[]
  selected_variant?: SelectedPlaybackVariant
  selected_audio_stream_index?: number
  playback_mode?: string
  hls_manifest_url?: string
  checks: PlaybackCheck[]
  playable: boolean
  decision: PlaybackDecision
}

export type WorkflowRun = {
  id: number
  run_key: string
  library_id: number
  reason: string
  status: string
  priority: number
  scope_key: string
  payload_json: string
  error_message: string
  started_at?: string
  finished_at?: string
  cancelled_at?: string
  created_at: string
  updated_at: string
}

export type WorkflowTask = {
  id: number
  run_id: number
  library_id: number
  task_key: string
  task_type: string
  stage: string
  status: string
  priority: number
  scope_key: string
  payload_json: string
  resource_json: string
  blocked_by: number
  attempts: number
  max_attempts: number
  available_at: string
  lease_owner: string
  lease_until?: string
  error_message: string
  resource_wait_key: string
  started_at?: string
  finished_at?: string
  created_at: string
  updated_at: string
}

export type WorkflowTaskStatusCount = {
  stage: string
  status: string
  count: number
}

export type WorkflowResourceWaitCount = {
  resource_key: string
  count: number
}

export type WorkflowRunStatusView = {
  run: WorkflowRun
  task_counts: WorkflowTaskStatusCount[] | null
  resource_waits: WorkflowResourceWaitCount[] | null
  recent_tasks: WorkflowTask[] | null
}

export type WorkflowDiagnostics = {
  active_runs: number
  running_tasks: number
  blocked_tasks: number
  expired_leases: number
  resource_budgets: Array<{
    id: number
    resource_key: string
    max_concurrency: number
    enabled: boolean
  }> | null
  resource_usage: Array<{
    id: number
    resource_key: string
    task_id: number
    run_id: number
    library_id: number
    units: number
    lease_until: string
  }> | null
}

export type ScheduleFrequencyKind = 'daily' | 'weekly' | 'monthly'

export type ScheduleScopeKind = 'global' | 'library'

export type ScheduleRunStatus = 'queued' | 'running' | 'completed' | 'failed'

export type ScheduleFrequency = {
  kind: ScheduleFrequencyKind
  time_of_day: string
  weekday?: number
  day_of_month?: number
}

export type ScheduleRun = {
  id: number
  schedule_id: number
  status: ScheduleRunStatus
  job_id?: number
  error_summary: string
  started_at?: string
  finished_at?: string
  created_at: string
  updated_at: string
}

export type Schedule = {
  id: number
  name: string
  kind: string
  scope_kind: ScheduleScopeKind
  library_id?: number
  frequency: ScheduleFrequency
  enabled: boolean
  next_run_at?: string
  latest_run_status?: ScheduleRunStatus | ''
  latest_run_message: string
  latest_job_id?: number
  latest_run_started_at?: string
  latest_run_finished_at?: string
  recent_runs?: ScheduleRun[]
  created_at: string
  updated_at: string
}

export type ScheduleMutationInput = {
  name: string
  kind: string
  scope_kind: ScheduleScopeKind
  library_id?: number
  enabled?: boolean
  frequency: ScheduleFrequency
}

export type ScheduleRunNowResult = {
  run: ScheduleRun
}

export type AcceptedResult = {
  queued: boolean
}

type ApiOptions = {
  baseUrl: string
  token?: string | null
}

let isRedirectingToLogin = false
let isRedirectingToSetup = false

export class ApiError extends Error {
  status: number
  code: string

  constructor(status: number, error: ApiErrorShape) {
    super(error.message)
    this.name = 'ApiError'
    this.status = status
    this.code = error.code
  }
}

export function getApiBaseUrl(): string {
  const envBaseUrl = (
    import.meta.env.VITE_API_BASE_URL as string | undefined
  )?.replace(/\/$/, '')
  const defaultBaseUrl = 'http://127.0.0.1:8080'
  const devProxyBaseUrl = ''

  if (typeof window === 'undefined') {
    return envBaseUrl ?? defaultBaseUrl
  }

  const storedBaseUrl = window.localStorage
    .getItem('mibo-web-api-base-url')
    ?.trim()
    .replace(/\/$/, '')

  if (storedBaseUrl || envBaseUrl) {
    return storedBaseUrl || envBaseUrl || defaultBaseUrl
  }

  if (import.meta.env.DEV) {
    return devProxyBaseUrl
  }

  return window.location.origin || defaultBaseUrl
}

export function buildApiUrl(pathname: string) {
  if (pathname.startsWith('http://') || pathname.startsWith('https://')) {
    return pathname
  }

  return `${getApiBaseUrl()}${pathname.startsWith('/') ? pathname : `/${pathname}`}`
}

function handleUnauthorizedResponse(pathname: string, token?: string | null) {
  if (typeof window === 'undefined') {
    return
  }

  if (
    pathname === '/api/v1/auth/login' ||
    pathname === '/api/v1/auth/register'
  ) {
    return
  }

  const { pathname: currentPathname, search, hash } = window.location

  if (token && shouldAutoClearInvalidToken()) {
    useAuthStore.getState().auth.clearSession()
  }

  if (currentPathname === '/sign-in' || isRedirectingToLogin) {
    return
  }

  isRedirectingToLogin = true

  const redirect = `${currentPathname}${search}${hash}`
  const loginUrl = new URL('/sign-in', window.location.origin)
  loginUrl.searchParams.set('redirect', redirect)
  window.location.replace(loginUrl.toString())
}

function shouldAutoClearInvalidToken() {
  if (typeof window === 'undefined') {
    return true
  }
  return (
    window.localStorage.getItem('mibo-auto-clear-invalid-token') !== 'false'
  )
}

function handleSetupRequiredResponse() {
  if (typeof window === 'undefined') {
    return
  }

  useAuthStore.getState().auth.clearSession()

  if (window.location.pathname === '/setup' || isRedirectingToSetup) {
    return
  }

  isRedirectingToSetup = true
  window.location.replace(new URL('/setup', window.location.origin).toString())
}

export function createMiboApi(options: ApiOptions) {
  const baseUrl = options.baseUrl.replace(/\/$/, '')

  async function request<T>(pathname: string, init?: RequestInit): Promise<T> {
    const headers = new Headers(init?.headers)

    if (!headers.has('Content-Type') && init?.body !== undefined) {
      headers.set('Content-Type', 'application/json')
    }

    if (options.token) {
      headers.set('Authorization', `Bearer ${options.token}`)
    }

    let response: Response
    try {
      response = await fetch(`${baseUrl}${pathname}`, {
        ...init,
        headers,
        credentials: 'include',
      })
    } catch {
      throw new ApiError(0, {
        code: 'network_error',
        message: '无法连接后端服务，请确认 Mibo 服务已启动。',
      })
    }

    if (response.status === 401) {
      handleUnauthorizedResponse(pathname, options.token)
    }

    let payload: Envelope<T> | null = null
    try {
      payload = (await response.json()) as Envelope<T>
    } catch {
      if (!response.ok) {
        throw new ApiError(response.status, {
          code: 'request_failed',
          message: `请求失败，状态码 ${response.status}`,
        })
      }
    }

    if (payload?.error?.code === 'setup_required') {
      handleSetupRequiredResponse()
    }

    if (!response.ok || payload?.error) {
      throw new ApiError(
        response.status,
        payload?.error ?? {
          code: 'request_failed',
          message: `请求失败，状态码 ${response.status}`,
        }
      )
    }

    if (payload?.data === undefined) {
      throw new ApiError(response.status, {
        code: 'missing_payload',
        message: '服务端返回了空数据',
      })
    }

    return payload.data
  }

  return {
    getSetupStatus() {
      return request<SetupStatus>('/api/v1/setup/status')
    },
    getSetupDatabaseState() {
      return request<SetupDatabaseState>('/api/v1/setup/database')
    },
    persistSetupDatabaseDraft(input: SetupDatabaseInput) {
      return request<SetupDatabaseValidation>('/api/v1/setup/database/draft', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    validateSetupDatabase(input: SetupDatabaseInput) {
      return request<SetupDatabaseValidation>(
        '/api/v1/setup/database/validate',
        {
          method: 'POST',
          body: JSON.stringify(input),
        }
      )
    },
    applySetupDatabase(input: SetupDatabaseInput) {
      return request<SetupDatabaseApplyResult>('/api/v1/setup/database/apply', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    getAppSession() {
      return request<AppSession>('/api/v1/app/session')
    },
    register(username: string, password: string, pin: string) {
      return request<User>('/api/v1/auth/register', {
        method: 'POST',
        body: JSON.stringify({ username, password, pin }),
      })
    },
    registerSetupAdmin(username: string, password: string, pin: string) {
      return request<User>('/api/v1/setup/register-admin', {
        method: 'POST',
        body: JSON.stringify({ username, password, pin }),
      })
    },
    login(username: string, password: string) {
      return request<LoginResult>('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({ username, password }),
      })
    },
    listLoginUsers() {
      return request<LoginUserProfile[]>('/api/v1/auth/users')
    },
    loginWithPin(userId: number, pin: string) {
      return request<LoginResult>('/api/v1/auth/login/pin', {
        method: 'POST',
        body: JSON.stringify({ user_id: userId, pin }),
      })
    },
    logout() {
      return request<{ status: string }>('/api/v1/auth/logout', {
        method: 'POST',
      })
    },
    listLoginSessions() {
      return request<LoginSession[]>('/api/v1/auth/sessions')
    },
    revokeLoginSession(sessionId: number) {
      return request<{ id: number; status: string }>(
        `/api/v1/auth/sessions/${sessionId}`,
        { method: 'DELETE' }
      )
    },
    revokeOtherLoginSessions() {
      return request<{ status: string }>('/api/v1/auth/sessions/others', {
        method: 'DELETE',
      })
    },
    me() {
      return request<User>('/api/v1/me')
    },
    updateOwnPin(pin: string) {
      return request<User>('/api/v1/me/pin', {
        method: 'PUT',
        body: JSON.stringify({ pin }),
      })
    },
    getUserSettings() {
      return request<UserSettings>('/api/v1/me/settings')
    },
    updateUserSettings(input: UserSettingsInput) {
      return request<UserSettings>('/api/v1/me/settings', {
        method: 'PUT',
        body: JSON.stringify(input),
      })
    },
    listLiveTVSources() {
      return request<LiveTVSource[]>('/api/v1/live-tv/sources')
    },
    createLiveTVSource(input: CreateLiveTVSourceInput) {
      return request<LiveTVSource>('/api/v1/live-tv/sources', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    updateLiveTVSource(sourceId: number, input: UpdateLiveTVSourceInput) {
      return request<LiveTVSource>(`/api/v1/live-tv/sources/${sourceId}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      })
    },
    deleteLiveTVSource(sourceId: number) {
      return request<{ id: number; status: string; type: string }>(
        `/api/v1/live-tv/sources/${sourceId}`,
        {
          method: 'DELETE',
        }
      )
    },
    refreshLiveTVSource(sourceId: number) {
      return request<LiveTVSource>(
        `/api/v1/live-tv/sources/${sourceId}/refresh`,
        {
          method: 'POST',
        }
      )
    },
    listLiveTVChannels(options?: {
      source_id?: number
      group?: string
      q?: string
      enabled?: boolean
    }) {
      const query = new URLSearchParams()
      if (typeof options?.source_id === 'number' && options.source_id > 0) {
        query.set('source_id', String(options.source_id))
      }
      if (options?.group) {
        query.set('group', options.group)
      }
      if (options?.q) {
        query.set('q', options.q)
      }
      if (typeof options?.enabled === 'boolean') {
        query.set('enabled', String(options.enabled))
      }
      const queryString = query.toString()
      return request<LiveTVChannel[]>(
        `/api/v1/live-tv/channels${queryString ? `?${queryString}` : ''}`
      )
    },
    listLiveTVChannelGroups(options?: {
      source_id?: number
      q?: string
      enabled?: boolean
    }) {
      const query = new URLSearchParams()
      if (typeof options?.source_id === 'number' && options.source_id > 0) {
        query.set('source_id', String(options.source_id))
      }
      if (options?.q) {
        query.set('q', options.q)
      }
      if (typeof options?.enabled === 'boolean') {
        query.set('enabled', String(options.enabled))
      }
      const queryString = query.toString()
      return request<LiveTVChannelGroup[]>(
        `/api/v1/live-tv/channel-groups${queryString ? `?${queryString}` : ''}`
      )
    },
    listLiveTVPrograms(options?: {
      source_id?: number
      q?: string
      current?: boolean
      limit?: number
      offset?: number
    }) {
      const query = new URLSearchParams()
      if (typeof options?.source_id === 'number' && options.source_id > 0) {
        query.set('source_id', String(options.source_id))
      }
      if (options?.q) {
        query.set('q', options.q)
      }
      if (typeof options?.current === 'boolean') {
        query.set('current', String(options.current))
      }
      if (typeof options?.limit === 'number' && options.limit > 0) {
        query.set('limit', String(options.limit))
      }
      if (typeof options?.offset === 'number' && options.offset > 0) {
        query.set('offset', String(options.offset))
      }
      const queryString = query.toString()
      return request<LiveTVProgramList>(
        `/api/v1/live-tv/programs${queryString ? `?${queryString}` : ''}`
      )
    },
    getLiveTVPlayback(channelId: number) {
      return request<LiveTVPlaybackSource>(
        `/api/v1/live-tv/channels/${channelId}/playback`
      )
    },
    listMediaSources() {
      return request<MediaSource[]>('/api/v1/media-sources')
    },
    browseStorageProvider(provider: string, path?: string, refresh = false) {
      return request<StorageBrowseResult>('/api/v1/storage/providers/browse', {
        method: 'POST',
        body: JSON.stringify({ provider, path, refresh }),
      })
    },
    browsePluginProvider(providerId: number, path?: string, refresh = false) {
      return request<StorageBrowseResult>('/api/v1/storage/plugin/browse', {
        method: 'POST',
        body: JSON.stringify({ provider_id: providerId, path, refresh }),
      })
    },
    browseOpenList(input: {
      path?: string
      refresh?: boolean
      config: OpenListMediaSourceConfig
    }) {
      return request<StorageBrowseResult>('/api/v1/storage/openlist/browse', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    testOpenListConnection(input: { config: OpenListMediaSourceConfig }) {
      return request<OpenListTestResult>('/api/v1/storage/openlist/test', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    createMediaSource(input: {
      provider: string
      name: string
      root_path: string
      config?: MediaSourceConfigInput
    }) {
      return request<MediaSource>('/api/v1/media-sources', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    updateMediaSource(
      mediaSourceId: number,
      input: {
        name: string
        root_path: string
        config?: MediaSourceConfigInput
      }
    ) {
      return request<MediaSource>(`/api/v1/media-sources/${mediaSourceId}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      })
    },
    deleteMediaSource(mediaSourceId: number) {
      return request<{ id: number; status: string; type: string }>(
        `/api/v1/media-sources/${mediaSourceId}`,
        {
          method: 'DELETE',
        }
      )
    },
    browseMediaSource(mediaSourceId: number, path?: string, refresh = false) {
      return request<StorageBrowseResult>('/api/v1/media-sources/browse', {
        method: 'POST',
        body: JSON.stringify({ id: mediaSourceId, path, refresh }),
      })
    },
    listLibraries() {
      return request<Library[]>('/api/v1/libraries')
    },
    listLibraryAccessTags() {
      return request<LibraryAccessTag[]>('/api/v1/library-access-tags')
    },
    listLibraryInventoryFiles(
      libraryId: number,
      options?: { page?: number; limit?: number }
    ) {
      const query = new URLSearchParams()
      if (typeof options?.page === 'number' && options.page > 0) {
        query.set('page', String(options.page))
      }
      if (typeof options?.limit === 'number' && options.limit > 0) {
        query.set('limit', String(options.limit))
      }
      const queryString = query.toString()
      return request<InventoryFileListResponse>(
        `/api/v1/libraries/${libraryId}/inventory-files${queryString ? `?${queryString}` : ''}`
      )
    },
    listInventoryFiles(options?: {
      page?: number
      limit?: number
      library_id?: number
      q?: string
    }) {
      const query = new URLSearchParams()
      if (typeof options?.page === 'number' && options.page > 0) {
        query.set('page', String(options.page))
      }
      if (typeof options?.limit === 'number' && options.limit > 0) {
        query.set('limit', String(options.limit))
      }
      if (typeof options?.library_id === 'number' && options.library_id > 0) {
        query.set('library_id', String(options.library_id))
      }
      if (options?.q?.trim()) {
        query.set('q', options.q.trim())
      }
      const queryString = query.toString()
      return request<InventoryFileListResponse>(
        `/api/v1/inventory-files${queryString ? `?${queryString}` : ''}`
      )
    },
    getOperationsOverview() {
      return request<OperationsOverview>('/api/v1/operations/overview')
    },
    listOperationsTasks(filters?: {
      page?: number
      page_size?: number
      lifecycle_status?: 'active' | 'resolved' | 'all'
      kind?: string
      action_type?: string
      library_id?: number
      q?: string
    }) {
      const query = new URLSearchParams()
      if (typeof filters?.page === 'number' && filters.page > 0) {
        query.set('page', String(filters.page))
      }
      if (typeof filters?.page_size === 'number' && filters.page_size > 0) {
        query.set('page_size', String(filters.page_size))
      }
      if (filters?.lifecycle_status) {
        query.set('lifecycle_status', filters.lifecycle_status)
      }
      if (filters?.kind?.trim()) {
        query.set('kind', filters.kind.trim())
      }
      if (filters?.action_type?.trim()) {
        query.set('action_type', filters.action_type.trim())
      }
      if (
        typeof filters?.library_id === 'number' &&
        Number.isFinite(filters.library_id) &&
        filters.library_id > 0
      ) {
        query.set('library_id', String(filters.library_id))
      }
      if (filters?.q?.trim()) {
        query.set('q', filters.q.trim())
      }
      const queryString = query.toString()
      return request<OperationsTaskListResult>(
        `/api/v1/operations/tasks${queryString ? `?${queryString}` : ''}`
      )
    },
    getOperationsPipeline() {
      return request<OperationsPipeline>('/api/v1/operations/pipeline')
    },
    listOperationsIssues(filters?: {
      page?: number
      page_size?: number
      status?: OperationsIssueLifecycleStatus | 'all'
      kind?: OperationsIssueKind | 'all'
      action_type?: OperationsIssueActionType | 'all'
      library_id?: number
      q?: string
    }) {
      const query = new URLSearchParams()
      if (typeof filters?.page === 'number' && filters.page > 0) {
        query.set('page', String(filters.page))
      }
      if (typeof filters?.page_size === 'number' && filters.page_size > 0) {
        query.set('page_size', String(filters.page_size))
      }
      if (filters?.status) {
        query.set('status', filters.status)
      }
      if (filters?.kind) {
        query.set('kind', filters.kind)
      }
      if (filters?.action_type) {
        query.set('action_type', filters.action_type)
      }
      if (
        typeof filters?.library_id === 'number' &&
        Number.isFinite(filters.library_id) &&
        filters.library_id > 0
      ) {
        query.set('library_id', String(filters.library_id))
      }
      if (filters?.q?.trim()) {
        query.set('q', filters.q.trim())
      }
      const queryString = query.toString()
      return request<OperationsIssueListResult>(
        `/api/v1/operations/issues${queryString ? `?${queryString}` : ''}`
      )
    },
    getOperationsIssue(issueId: number) {
      return request<OperationsIssue>(`/api/v1/operations/issues/${issueId}`)
    },
    listOperationsIssueEvents(issueId: number) {
      return request<OperationsIssueEvent[]>(
        `/api/v1/operations/issues/${issueId}/events`
      )
    },
    executeOperationsIssueAction(
      issueId: number,
      input: ExecuteIssueActionInput
    ) {
      return request<OperationsActionResult>(
        `/api/v1/operations/issues/${issueId}/actions`,
        {
          method: 'POST',
          body: JSON.stringify(input),
        }
      )
    },
    executeOperationsAction(actionId: string) {
      return request<OperationsActionResult>(
        `/api/v1/operations/actions/${encodeURIComponent(actionId)}`,
        { method: 'POST' }
      )
    },
    getConsoleSummary() {
      return request<ConsoleSummary>('/api/v1/admin/console')
    },
    getIngestDiagnostics() {
      return request<IngestDiagnosticsResult>(
        '/api/v1/admin/ingest/diagnostics'
      )
    },
    retryIngestStage(stageId: number) {
      return request<IngestRetryResult>(
        `/api/v1/admin/ingest/stages/${stageId}/retry`,
        {
          method: 'POST',
        }
      )
    },
    resolveIngestReviewStage(stageId: number) {
      return request<IngestResolveReviewResult>(
        `/api/v1/admin/ingest/stages/${stageId}/resolve-review`,
        {
          method: 'POST',
        }
      )
    },
    runConsoleAction(actionId: string) {
      const actionEndpoints: Record<string, string> = {
        'scan-libraries': '/api/v1/admin/console/actions/scan-libraries',
        'refresh-update': '/api/v1/admin/console/actions/refresh-update',
        'prepare-update': '/api/v1/admin/console/actions/prepare-update',
        'apply-update': '/api/v1/admin/console/actions/apply-update',
        restart: '/api/v1/admin/console/actions/restart',
      }
      const endpoint = actionEndpoints[actionId]
      if (!endpoint) {
        throw new Error('unsupported console action')
      }
      return request<
        | ConsoleActionResult
        | ConsoleApplyUpdateResult
        | ConsolePrepareUpdateResult
        | ConsoleRestartActionResult
      >(endpoint, {
        method: 'POST',
      })
    },
    listAdminLogs() {
      return request<AdminLogFile[]>('/api/v1/admin/logs')
    },
    getAdminLogSettings() {
      return request<AdminLogSettings>('/api/v1/admin/logs/settings')
    },
    updateAdminLogSettings(input: AdminLogSettingsInput) {
      return request<AdminLogSettings>('/api/v1/admin/logs/settings', {
        method: 'PUT',
        body: JSON.stringify(input),
      })
    },
    listAdminUsers() {
      return request<AdminUser[]>('/api/v1/admin/users')
    },
    createAdminUser(input: CreateAdminUserInput) {
      return request<AdminUser>('/api/v1/admin/users', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    updateAdminUser(id: number, input: UpdateAdminUserInput) {
      return request<AdminUser>(`/api/v1/admin/users/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      })
    },
    listAdminRoles() {
      return request<AdminRole[]>('/api/v1/admin/roles')
    },
    createAdminRole(input: CreateAdminRoleInput) {
      return request<AdminRole>('/api/v1/admin/roles', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    updateAdminRole(id: number, input: UpdateAdminRoleInput) {
      return request<AdminRole>(`/api/v1/admin/roles/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      })
    },
    deleteAdminRole(id: number) {
      return request<{ id: number; status: string }>(
        `/api/v1/admin/roles/${id}`,
        {
          method: 'DELETE',
        }
      )
    },
    getAdminLog(name: string) {
      return request<AdminLogContent>(
        `/api/v1/admin/logs/${encodeURIComponent(name)}`
      )
    },
    deleteAdminLog(name: string) {
      return request<{ name: string; status: string }>(
        `/api/v1/admin/logs/${encodeURIComponent(name)}`,
        {
          method: 'DELETE',
        }
      )
    },
    releaseTranscodeSession(
      sessionId: string,
      options?: { keepalive?: boolean }
    ) {
      return request<{ id: string; status: string }>(
        `/api/v1/transcodes/${encodeURIComponent(sessionId)}`,
        {
          method: 'DELETE',
          keepalive: options?.keepalive,
        }
      )
    },
    listMetadataProviderInstances() {
      return request<MetadataProviderInstance[]>(
        '/api/v1/settings/metadata/providers'
      )
    },
    createMetadataProviderInstance(input: MetadataProviderInstanceInput) {
      return request<MetadataProviderInstance>(
        '/api/v1/settings/metadata/providers',
        {
          method: 'POST',
          body: JSON.stringify(input),
        }
      )
    },
    updateMetadataProviderInstance(
      providerId: number,
      input: Partial<MetadataProviderInstanceInput>
    ) {
      return request<MetadataProviderInstance>(
        `/api/v1/settings/metadata/providers/${providerId}`,
        {
          method: 'PATCH',
          body: JSON.stringify(input),
        }
      )
    },
    listMetadataProfiles() {
      return request<MetadataProfile[]>('/api/v1/settings/metadata/profiles')
    },
    createMetadataProfile(input: MetadataProfileInput) {
      return request<MetadataProfile>('/api/v1/settings/metadata/profiles', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    updateMetadataProfile(
      profileId: number,
      input: Partial<MetadataProfileInput>
    ) {
      return request<MetadataProfile>(
        `/api/v1/settings/metadata/profiles/${profileId}`,
        {
          method: 'PATCH',
          body: JSON.stringify(input),
        }
      )
    },
    listPluginProviderInstances() {
      return request<PluginProviderInstance[]>(
        '/api/v1/settings/plugin/providers'
      )
    },
    getPluginProviderDetail(providerId: number) {
      return request<PluginProviderDetail>(
        `/api/v1/settings/plugin/providers/${providerId}/detail`
      )
    },
    previewRemotePluginManifest(endpoint: string) {
      return request<PluginManifest>(
        '/api/v1/settings/plugin/providers/preview',
        {
          method: 'POST',
          body: JSON.stringify({ endpoint }),
        }
      )
    },
    createRemotePluginProviderInstance(input: RemotePluginProviderInput) {
      return request<PluginProviderInstance>(
        '/api/v1/settings/plugin/providers',
        {
          method: 'POST',
          body: JSON.stringify(input),
        }
      )
    },
    updateRemotePluginProviderInstance(
      providerId: number,
      input: Partial<RemotePluginProviderInput>
    ) {
      return request<PluginProviderInstance>(
        `/api/v1/settings/plugin/providers/${providerId}`,
        {
          method: 'PATCH',
          body: JSON.stringify(input),
        }
      )
    },
    disablePluginProviderInstance(providerId: number) {
      return request<PluginProviderInstance>(
        `/api/v1/settings/plugin/providers/${providerId}/disable`,
        {
          method: 'POST',
        }
      )
    },
    refreshPluginProviderHealth(providerId: number) {
      return request<PluginProviderInstance>(
        `/api/v1/settings/plugin/providers/${providerId}/refresh-health`,
        {
          method: 'POST',
        }
      )
    },
    listInternalPlugins() {
      return request<InternalPlugin[]>('/api/v1/settings/plugin/internal')
    },
    updateInternalPlugin(
      pluginId: string,
      input: {
        enabled?: boolean
        local_subtitle?: {
          external_file_enabled?: boolean
          embedded_extraction_enabled?: boolean
        }
      }
    ) {
      return request<InternalPlugin>(
        `/api/v1/settings/plugin/internal/${encodeURIComponent(pluginId)}`,
        {
          method: 'PATCH',
          body: JSON.stringify(input),
        }
      )
    },
    getOpenSubtitlesSettings() {
      return request<OpenSubtitlesSettings>(
        '/api/v1/settings/plugin/internal/opensubtitles/config'
      )
    },
    updateOpenSubtitlesSettings(input: OpenSubtitlesSettingsInput) {
      return request<OpenSubtitlesSettings>(
        '/api/v1/settings/plugin/internal/opensubtitles/config',
        {
          method: 'PUT',
          body: JSON.stringify(input),
        }
      )
    },
    getGeneralConfig() {
      return request<GeneralConfigSettings>('/api/v1/settings/general')
    },
    updateGeneralConfig(input: GeneralConfigInput) {
      return request<GeneralConfigSettings>('/api/v1/settings/general', {
        method: 'PUT',
        body: JSON.stringify(input),
      })
    },
    listSubtitleProviderInstances() {
      return request<SubtitleProviderInstance[]>(
        '/api/v1/settings/subtitles/providers'
      )
    },
    createSubtitleProviderInstance(input: SubtitleProviderInstanceInput) {
      return request<SubtitleProviderInstance>(
        '/api/v1/settings/subtitles/providers',
        {
          method: 'POST',
          body: JSON.stringify(input),
        }
      )
    },
    updateSubtitleProviderInstance(
      providerId: number,
      input: Partial<SubtitleProviderInstanceInput>
    ) {
      return request<SubtitleProviderInstance>(
        `/api/v1/settings/subtitles/providers/${providerId}`,
        {
          method: 'PATCH',
          body: JSON.stringify(input),
        }
      )
    },
    listLocalPluginInstallations() {
      return request<LocalPluginInstallation[]>(
        '/api/v1/settings/plugin/local/installations'
      )
    },
    installLocalPlugin(input: LocalPluginInstallInput) {
      return request<LocalPluginInstallation>(
        '/api/v1/settings/plugin/local/installations',
        {
          method: 'POST',
          body: JSON.stringify(input),
        }
      )
    },
    startLocalPluginInstallation(installationId: number) {
      return request<LocalPluginInstallation>(
        `/api/v1/settings/plugin/local/installations/${installationId}/start`,
        { method: 'POST' }
      )
    },
    stopLocalPluginInstallation(installationId: number) {
      return request<LocalPluginInstallation>(
        `/api/v1/settings/plugin/local/installations/${installationId}/stop`,
        { method: 'POST' }
      )
    },
    restartLocalPluginInstallation(installationId: number) {
      return request<LocalPluginInstallation>(
        `/api/v1/settings/plugin/local/installations/${installationId}/restart`,
        { method: 'POST' }
      )
    },
    uninstallLocalPluginInstallation(installationId: number) {
      return request<LocalPluginInstallation>(
        `/api/v1/settings/plugin/local/installations/${installationId}/uninstall`,
        { method: 'POST' }
      )
    },
    getLocalPluginInstallationLogs(installationId: number) {
      return request<{ logs: string[] }>(
        `/api/v1/settings/plugin/local/installations/${installationId}/logs`
      )
    },
    getPluginCatalogOverview() {
      return request<PluginCatalogOverview>('/api/v1/settings/plugin/catalog')
    },
    getNetworkSettings() {
      return request<NetworkSettings>('/api/v1/settings/network')
    },
    updateNetworkSettings(input: NetworkSettingsInput) {
      return request<NetworkSettings>('/api/v1/settings/network', {
        method: 'PUT',
        body: JSON.stringify(input),
      })
    },
    getLibrary(libraryId: number) {
      return request<LibraryDetail>(`/api/v1/libraries/${libraryId}`)
    },
    createLibrary(input: {
      name: string
      media_source_id: number
      root_path: string
      visibility_mode?: 'default_open' | 'allow_list_only'
      access_tags?: string[]
      scan?: LibraryScanPolicy
      metadata?: LibraryMetadataPolicy
      metadata_strategy?: LibraryMetadataStrategyInput
      playback?: LibraryPlaybackPolicy
      subtitle?: LibrarySubtitlePolicy
      scan_exclusion_rules?: ScanExclusionRuleInput[]
    }) {
      return request<{ library: LibraryDetail }>('/api/v1/libraries', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    updateLibraryAccessTags(libraryId: number, accessTags: string[]) {
      return request<{ library_id: number; access_tags: LibraryAccessTag[] }>(
        `/api/v1/libraries/${libraryId}/access-tags`,
        {
          method: 'PUT',
          body: JSON.stringify({ access_tags: accessTags }),
        }
      )
    },
    updateLibraryVisibilityMode(
      libraryId: number,
      visibilityMode: 'default_open' | 'allow_list_only'
    ) {
      return request<{ library_id: number; visibility_mode: string }>(
        `/api/v1/libraries/${libraryId}/visibility-mode`,
        {
          method: 'PUT',
          body: JSON.stringify({ visibility_mode: visibilityMode }),
        }
      )
    },
    listLibraryPaths(libraryId: number) {
      return request<LibraryPath[]>(`/api/v1/libraries/${libraryId}/paths`)
    },
    addLibraryPath(
      libraryId: number,
      input: {
        media_source_id: number
        root_path: string
        display_name?: string
        enabled?: boolean
      }
    ) {
      return request<LibraryPath>(`/api/v1/libraries/${libraryId}/paths`, {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    updateLibraryPath(
      libraryId: number,
      pathId: number,
      input: {
        media_source_id?: number
        root_path?: string
        display_name?: string
        enabled?: boolean
      }
    ) {
      return request<LibraryPath>(
        `/api/v1/libraries/${libraryId}/paths/${pathId}`,
        {
          method: 'PATCH',
          body: JSON.stringify(input),
        }
      )
    },
    getLibraryPolicies(libraryId: number) {
      return request<LibraryPolicies>(`/api/v1/libraries/${libraryId}/policies`)
    },
    getLibraryMetadataStrategy(libraryId: number) {
      return request<LibraryMetadataStrategy>(
        `/api/v1/libraries/${libraryId}/metadata-strategy`
      )
    },
    updateLibraryMetadataStrategy(
      libraryId: number,
      input: LibraryMetadataStrategyInput
    ) {
      return request<LibraryMetadataStrategy>(
        `/api/v1/libraries/${libraryId}/metadata-strategy`,
        {
          method: 'PUT',
          body: JSON.stringify(input),
        }
      )
    },
    updateLibraryScanPolicy(libraryId: number, input: LibraryScanPolicy) {
      return request<LibraryScanPolicy>(
        `/api/v1/libraries/${libraryId}/policies/scan`,
        {
          method: 'PUT',
          body: JSON.stringify(input),
        }
      )
    },
    updateLibraryMetadataPolicy(
      libraryId: number,
      input: LibraryMetadataPolicy
    ) {
      return request<LibraryMetadataPolicy>(
        `/api/v1/libraries/${libraryId}/policies/metadata`,
        {
          method: 'PUT',
          body: JSON.stringify(input),
        }
      )
    },
    updateLibraryPlaybackPolicy(
      libraryId: number,
      input: LibraryPlaybackPolicy
    ) {
      return request<LibraryPlaybackPolicy>(
        `/api/v1/libraries/${libraryId}/policies/playback`,
        {
          method: 'PUT',
          body: JSON.stringify(input),
        }
      )
    },
    updateLibrarySubtitlePolicy(
      libraryId: number,
      input: LibrarySubtitlePolicy
    ) {
      return request<LibrarySubtitlePolicy>(
        `/api/v1/libraries/${libraryId}/policies/subtitle`,
        {
          method: 'PUT',
          body: JSON.stringify(input),
        }
      )
    },
    deleteLibrary(libraryId: number) {
      return request<{ id: number; status: string; type: string }>(
        `/api/v1/libraries/${libraryId}`,
        {
          method: 'DELETE',
        }
      )
    },
    scanLibrary(libraryId: number, mode: 'full' | 'changed' = 'full') {
      return request<{ queued: boolean; mode: 'full' | 'changed' }>(
        `/api/v1/libraries/${libraryId}/scan`,
        {
          method: 'POST',
          body: JSON.stringify({ mode }),
        }
      )
    },
    listScanExclusions(filters?: { libraryId?: number; enabled?: boolean }) {
      const query = new URLSearchParams()
      if (typeof filters?.libraryId === 'number' && filters.libraryId > 0) {
        query.set('library_id', String(filters.libraryId))
      }
      if (typeof filters?.enabled === 'boolean') {
        query.set('enabled', String(filters.enabled))
      }
      const queryString = query.toString()
      return request<ScanExclusionsView>(
        `/api/v1/scan-exclusions${queryString ? `?${queryString}` : ''}`
      )
    },
    setScanExclusionEnabled(exclusionId: number, enabled: boolean) {
      return request<ScanExclusion>(`/api/v1/scan-exclusions/${exclusionId}`, {
        method: 'PATCH',
        body: JSON.stringify({ enabled }),
      })
    },
    restoreFilenameExclusionMatch(ruleId: number, inventoryFileId: number) {
      return request<{ id: number }>(
        `/api/v1/filename-exclusion-rules/${ruleId}/restores`,
        {
          method: 'POST',
          body: JSON.stringify({ inventory_file_id: inventoryFileId }),
        }
      )
    },
    deleteFilenameExclusionRestore(ruleId: number, inventoryFileId: number) {
      return request<{ status: string }>(
        `/api/v1/filename-exclusion-rules/${ruleId}/restores`,
        {
          method: 'DELETE',
          body: JSON.stringify({ inventory_file_id: inventoryFileId }),
        }
      )
    },
    listScanExclusionRules() {
      return request<ScanExclusionRule[]>('/api/v1/scan-exclusion-rules')
    },
    createScanExclusionRule(input: ScanExclusionRuleInput) {
      return request<ScanExclusionRule>('/api/v1/scan-exclusion-rules', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    updateScanExclusionRule(ruleId: number, input: ScanExclusionRuleInput) {
      return request<ScanExclusionRule>(
        `/api/v1/scan-exclusion-rules/${ruleId}`,
        {
          method: 'PATCH',
          body: JSON.stringify(input),
        }
      )
    },
    setScanExclusionRuleEnabled(ruleId: number, enabled: boolean) {
      return request<ScanExclusionRule>(
        `/api/v1/scan-exclusion-rules/${ruleId}`,
        {
          method: 'PATCH',
          body: JSON.stringify({ enabled }),
        }
      )
    },
    deleteScanExclusionRule(ruleId: number) {
      return request<{ status: string }>(
        `/api/v1/scan-exclusion-rules/${ruleId}`,
        {
          method: 'DELETE',
        }
      )
    },
    replaceLibraryScanExclusionRules(
      libraryId: number,
      input: ScanExclusionRuleInput[]
    ) {
      return request<ScanExclusionRule[]>(
        `/api/v1/libraries/${libraryId}/scan-exclusion-rules`,
        {
          method: 'PUT',
          body: JSON.stringify({ rules: input }),
        }
      )
    },
    previewInventoryFileScanExclusion(fileId: number) {
      return request<FilenameExclusionPreview>(
        `/api/v1/inventory-files/${fileId}/scan-exclusion-preview`
      )
    },
    createInventoryFileFilenameExclusionRule(
      fileId: number,
      reason = 'advertisement'
    ) {
      return request<FilenameExclusionRule>(
        `/api/v1/inventory-files/${fileId}/filename-exclusion-rule`,
        {
          method: 'POST',
          body: JSON.stringify({ reason }),
        }
      )
    },
    setFilenameExclusionRuleEnabled(ruleId: number, enabled: boolean) {
      return request<FilenameExclusionRule>(
        `/api/v1/filename-exclusion-rules/${ruleId}`,
        {
          method: 'PATCH',
          body: JSON.stringify({ enabled }),
        }
      )
    },
    listLibraryItems(
      libraryId: number,
      queryOptions?: {
        type?: 'all' | 'movie' | 'show'
        year?: number
        sort?: CatalogDiscoverySort
        limit?: number
      }
    ) {
      const query = new URLSearchParams()

      if (queryOptions?.type) {
        query.set('type', queryOptions.type)
      }
      if (typeof queryOptions?.year === 'number') {
        query.set('year', String(queryOptions.year))
      }
      if (queryOptions?.sort) {
        query.set('sort', queryOptions.sort)
      }
      if (typeof queryOptions?.limit === 'number') {
        query.set('limit', String(queryOptions.limit))
      }

      const queryString = query.toString()
      return request<CatalogListItem[]>(
        `/api/v1/libraries/${libraryId}/items${queryString ? `?${queryString}` : ''}`
      )
    },
    discoverMedia(queryOptions?: DiscoveryQuery) {
      const query = new URLSearchParams()

      if (queryOptions?.scope) query.set('scope', queryOptions.scope)
      if (typeof queryOptions?.library_id === 'number') {
        query.set('library_id', String(queryOptions.library_id))
      }
      if (queryOptions?.q) query.set('q', queryOptions.q)
      if (queryOptions?.type) query.set('type', queryOptions.type)
      if (queryOptions?.genre) query.set('genre', queryOptions.genre)
      if (queryOptions?.region) query.set('region', queryOptions.region)
      if (typeof queryOptions?.year === 'number') {
        query.set('year', String(queryOptions.year))
      }
      if (typeof queryOptions?.min_rating === 'number') {
        query.set('min_rating', String(queryOptions.min_rating))
      }
      if (queryOptions?.watched_state) {
        query.set('watched_state', queryOptions.watched_state)
      }
      if (queryOptions?.organizing_state) {
        query.set('organizing_state', queryOptions.organizing_state)
      }
      if (queryOptions?.sort) query.set('sort', queryOptions.sort)
      if (queryOptions?.sort_direction) {
        query.set('sort_direction', queryOptions.sort_direction)
      }
      if (typeof queryOptions?.limit === 'number') {
        query.set('limit', String(queryOptions.limit))
      }
      if (typeof queryOptions?.offset === 'number') {
        query.set('offset', String(queryOptions.offset))
      }

      const queryString = query.toString()
      return request<CatalogDiscoveryResponse>(
        `/api/v1/discovery${queryString ? `?${queryString}` : ''}`
      )
    },
    browseLibraryHierarchy(queryOptions?: LibraryHierarchyQuery) {
      const query = new URLSearchParams()

      if (typeof queryOptions?.library_id === 'number') {
        query.set('library_id', String(queryOptions.library_id))
      }
      if (queryOptions?.path) query.set('path', queryOptions.path)
      if (queryOptions?.q) query.set('q', queryOptions.q)
      if (queryOptions?.type) query.set('type', queryOptions.type)
      if (queryOptions?.genre) query.set('genre', queryOptions.genre)
      if (queryOptions?.region) query.set('region', queryOptions.region)
      if (typeof queryOptions?.year === 'number') {
        query.set('year', String(queryOptions.year))
      }
      if (typeof queryOptions?.min_rating === 'number') {
        query.set('min_rating', String(queryOptions.min_rating))
      }
      if (queryOptions?.watched_state) {
        query.set('watched_state', queryOptions.watched_state)
      }
      if (queryOptions?.organizing_state) {
        query.set('organizing_state', queryOptions.organizing_state)
      }
      if (queryOptions?.sort) query.set('sort', queryOptions.sort)
      if (queryOptions?.sort_direction) {
        query.set('sort_direction', queryOptions.sort_direction)
      }
      if (typeof queryOptions?.limit === 'number') {
        query.set('limit', String(queryOptions.limit))
      }
      if (typeof queryOptions?.offset === 'number') {
        query.set('offset', String(queryOptions.offset))
      }

      const queryString = query.toString()
      return request<LibraryHierarchyResponse>(
        `/api/v1/library/browse${queryString ? `?${queryString}` : ''}`
      )
    },
    listSearchHistory(limit = 8) {
      return request<SearchHistoryEntry[]>(
        `/api/v1/search/history?limit=${limit}`
      )
    },
    getMetadataItem(itemId: number, options?: { libraryId?: number }) {
      const query = new URLSearchParams()
      if (typeof options?.libraryId === 'number') {
        query.set('library_id', String(options.libraryId))
      }
      const queryString = query.toString()
      return request<CatalogItemDetail>(
        `/api/v1/items/${itemId}${queryString ? `?${queryString}` : ''}`
      )
    },
    listMetadataItemResources(
      itemId: number,
      options?: { libraryId?: number }
    ) {
      const query = new URLSearchParams()
      if (typeof options?.libraryId === 'number') {
        query.set('library_id', String(options.libraryId))
      }
      const queryString = query.toString()
      return request<MetadataResourceDetail[]>(
        `/api/v1/items/${itemId}/resources${queryString ? `?${queryString}` : ''}`
      )
    },
    getInventoryFilePlayback(
      fileId: number,
      options?: {
        clientProfile?: ClientProfile
        variant?: string
        startSeconds?: number
        audioStreamIndex?: number
      }
    ) {
      const params = new URLSearchParams()
      if (options?.clientProfile)
        params.set('client_profile', options.clientProfile)
      if (options?.variant) params.set('variant', options.variant)
      if (typeof options?.startSeconds === 'number') {
        params.set('start_seconds', String(options.startSeconds))
      }
      if (typeof options?.audioStreamIndex === 'number') {
        params.set('audio_stream_index', String(options.audioStreamIndex))
      }
      const queryString = params.toString()
      return request<PlaybackSource>(
        `/api/v1/inventory-files/${fileId}/playback${queryString ? `?${queryString}` : ''}`
      )
    },
    searchInventoryFilePlaybackSubtitles(fileId: number, providerId: number) {
      return request<SubtitleSearchResult>(
        `/api/v1/inventory-files/${fileId}/playback/subtitles/search`,
        {
          method: 'POST',
          body: JSON.stringify({ provider_id: providerId }),
        }
      )
    },
    getCatalogPerson(personId: number) {
      return request<CatalogPersonPageDetail>(`/api/v1/people/${personId}`)
    },
    getCatalogGovernanceWorkspace(itemId: number) {
      return request<CatalogGovernanceWorkspace>(
        `/api/v1/items/${itemId}/governance`
      )
    },
    searchCatalogGovernanceCandidates(
      itemId: number,
      input: {
        title?: string
        year?: number
        imdb_id?: string
        tmdb_id?: string
        tvdb_id?: string
      },
      options?: { libraryId?: number }
    ) {
      const query = new URLSearchParams()
      if (typeof options?.libraryId === 'number') {
        query.set('library_id', String(options.libraryId))
      }
      const queryString = query.toString()
      return request<CatalogMetadataSearchResponse>(
        `/api/v1/items/${itemId}/governance/search${queryString ? `?${queryString}` : ''}`,
        {
          method: 'POST',
          body: JSON.stringify(input),
        }
      )
    },
    applyCatalogGovernanceCandidate(
      itemId: number,
      input: {
        external_id: string
      },
      options?: { libraryId?: number }
    ) {
      const query = new URLSearchParams()
      if (typeof options?.libraryId === 'number') {
        query.set('library_id', String(options.libraryId))
      }
      const queryString = query.toString()
      return request<CatalogGovernanceWorkspace>(
        `/api/v1/items/${itemId}/governance/apply-candidate${queryString ? `?${queryString}` : ''}`,
        {
          method: 'POST',
          body: JSON.stringify(input),
        }
      )
    },
    updateCatalogGovernanceField(
      itemId: number,
      input: {
        field_key: string
        value?: unknown
        lock?: boolean
        lock_reason?: string
        force?: boolean
      },
      options?: { libraryId?: number }
    ) {
      const query = new URLSearchParams()
      if (typeof options?.libraryId === 'number') {
        query.set('library_id', String(options.libraryId))
      }
      const queryString = query.toString()
      return request<CatalogGovernanceWorkspace>(
        `/api/v1/items/${itemId}/governance/fields${queryString ? `?${queryString}` : ''}`,
        {
          method: 'PUT',
          body: JSON.stringify(input),
        }
      )
    },
    selectCatalogGovernanceImage(
      itemId: number,
      input: {
        image_type: string
        url: string
      }
    ) {
      return request<CatalogGovernanceWorkspace>(
        `/api/v1/items/${itemId}/governance/images`,
        {
          method: 'PUT',
          body: JSON.stringify(input),
        }
      )
    },
    linkGovernanceResource(
      metadataItemId: number,
      resourceId: number,
      input: ResourceMetadataLinkInput
    ) {
      return request<CatalogMetadataOperation>(
        `/api/v1/items/${metadataItemId}/governance/resources/${resourceId}/links`,
        {
          method: 'POST',
          body: JSON.stringify(input),
        }
      )
    },
    updateGovernanceResourceLink(
      metadataItemId: number,
      resourceId: number,
      targetMetadataItemId: number,
      input: ResourceMetadataLinkUpdateInput
    ) {
      return request<CatalogMetadataOperation>(
        `/api/v1/items/${metadataItemId}/governance/resources/${resourceId}/links/${targetMetadataItemId}`,
        {
          method: 'PATCH',
          body: JSON.stringify(input),
        }
      )
    },
    unlinkGovernanceResource(
      metadataItemId: number,
      resourceId: number,
      targetMetadataItemId: number,
      options?: { libraryId?: number; role?: string; segmentIndex?: number }
    ) {
      const query = new URLSearchParams()
      if (typeof options?.libraryId === 'number') {
        query.set('library_id', String(options.libraryId))
      }
      if (options?.role) query.set('role', options.role)
      if (typeof options?.segmentIndex === 'number') {
        query.set('segment_index', String(options.segmentIndex))
      }
      const queryString = query.toString()
      return request<CatalogMetadataOperation>(
        `/api/v1/items/${metadataItemId}/governance/resources/${resourceId}/links/${targetMetadataItemId}${queryString ? `?${queryString}` : ''}`,
        { method: 'DELETE' }
      )
    },
    mergeGovernanceMetadata(metadataItemId: number, input: MetadataMergeInput) {
      return request<CatalogMetadataOperation>(
        `/api/v1/items/${metadataItemId}/governance/metadata-merge`,
        {
          method: 'POST',
          body: JSON.stringify(input),
        }
      )
    },
    splitGovernanceMetadata(metadataItemId: number, input: MetadataSplitInput) {
      return request<CatalogMetadataOperation>(
        `/api/v1/items/${metadataItemId}/governance/metadata-split`,
        {
          method: 'POST',
          body: JSON.stringify(input),
        }
      )
    },
    setGovernanceProjectionVisibility(
      metadataItemId: number,
      input: ProjectionVisibilityInput
    ) {
      return request<CatalogMetadataOperation>(
        `/api/v1/items/${metadataItemId}/governance/projection-visibility`,
        {
          method: 'PUT',
          body: JSON.stringify(input),
        }
      )
    },
    correctCatalogEpisodeNumbering(
      itemId: number,
      input: {
        season_number: number
        episode_number: number
        episode_number_end?: number
      }
    ) {
      return request<CatalogGovernanceWorkspace>(
        `/api/v1/items/${itemId}/governance/episode-numbering`,
        {
          method: 'PUT',
          body: JSON.stringify(input),
        }
      )
    },
    markInventoryFileScanExclusion(fileId: number, reason = 'advertisement') {
      return request<ScanExclusion>(
        `/api/v1/inventory-files/${fileId}/scan-exclusion`,
        {
          method: 'POST',
          body: JSON.stringify({ reason }),
        }
      )
    },
    reprobeInventoryFile(fileId: number) {
      return request<AcceptedResult>(
        `/api/v1/inventory-files/${fileId}/probe`,
        {
          method: 'POST',
        }
      )
    },
    listWorkflows(filters?: {
      limit?: number
      offset?: number
      status?: string
      library_id?: number
    }) {
      const query = new URLSearchParams()

      if (typeof filters?.limit === 'number') {
        query.set('limit', String(filters.limit))
      }
      if (typeof filters?.offset === 'number') {
        query.set('offset', String(filters.offset))
      }
      if (filters?.status) {
        query.set('status', filters.status)
      }
      if (typeof filters?.library_id === 'number') {
        query.set('library_id', String(filters.library_id))
      }

      const queryString = query.toString()
      return request<WorkflowRunStatusView[]>(
        `/api/v1/workflows${queryString ? `?${queryString}` : ''}`
      )
    },
    getWorkflow(workflowId: number) {
      return request<WorkflowRunStatusView>(`/api/v1/workflows/${workflowId}`)
    },
    getWorkflowDiagnostics() {
      return request<WorkflowDiagnostics>('/api/v1/workflows/diagnostics')
    },
    listSchedules() {
      return request<Schedule[]>('/api/v1/schedules')
    },
    getSchedule(scheduleId: number) {
      return request<Schedule>(`/api/v1/schedules/${scheduleId}`)
    },
    createSchedule(input: ScheduleMutationInput) {
      return request<Schedule>('/api/v1/schedules', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    updateSchedule(scheduleId: number, input: ScheduleMutationInput) {
      return request<Schedule>(`/api/v1/schedules/${scheduleId}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      })
    },
    toggleSchedule(scheduleId: number, enabled: boolean) {
      return request<Schedule>(`/api/v1/schedules/${scheduleId}/toggle`, {
        method: 'POST',
        body: JSON.stringify({ enabled }),
      })
    },
    runScheduleNow(scheduleId: number) {
      return request<ScheduleRunNowResult>(
        `/api/v1/schedules/${scheduleId}/run`,
        {
          method: 'POST',
        }
      )
    },
    listScheduleHistory(scheduleId: number) {
      return request<ScheduleRun[]>(`/api/v1/schedules/${scheduleId}/history`)
    },
    getCatalogPlayback(
      itemId: number,
      playbackOptions: {
        resourceId?: number
        libraryId?: number
        clientProfile: ClientProfile
        variant?: string
        startSeconds?: number
        audioStreamIndex?: number
      }
    ) {
      const query = new URLSearchParams({
        client_profile: playbackOptions.clientProfile,
      })

      if (typeof playbackOptions.resourceId === 'number') {
        query.set('resource_id', String(playbackOptions.resourceId))
      }
      if (typeof playbackOptions.libraryId === 'number') {
        query.set('library_id', String(playbackOptions.libraryId))
      }
      if (playbackOptions.variant) {
        query.set('variant', playbackOptions.variant)
      }
      if (typeof playbackOptions.startSeconds === 'number') {
        query.set('start_seconds', String(playbackOptions.startSeconds))
      }
      if (typeof playbackOptions.audioStreamIndex === 'number') {
        query.set(
          'audio_stream_index',
          String(playbackOptions.audioStreamIndex)
        )
      }

      return request<PlaybackSource>(
        `/api/v1/items/${itemId}/playback?${query.toString()}`
      )
    },
    searchCatalogPlaybackSubtitles(
      itemId: number,
      input: {
        providerId: number
        resourceId?: number
        libraryId?: number
      }
    ) {
      const query = new URLSearchParams()

      if (typeof input.resourceId === 'number') {
        query.set('resource_id', String(input.resourceId))
      }
      if (typeof input.libraryId === 'number') {
        query.set('library_id', String(input.libraryId))
      }

      const queryString = query.toString()
      return request<SubtitleSearchResult>(
        `/api/v1/items/${itemId}/playback/subtitles/search${queryString ? `?${queryString}` : ''}`,
        {
          method: 'POST',
          body: JSON.stringify({ provider_id: input.providerId }),
        }
      )
    },
    getMetadataPlayback(
      metadataItemId: number,
      playbackOptions: {
        resourceId?: number
        libraryId?: number
        clientProfile: ClientProfile
        variant?: string
        startSeconds?: number
      }
    ) {
      return this.getCatalogPlayback(metadataItemId, playbackOptions)
    },
    getMetadataItemProgress(itemId: number) {
      return request<ProgressState>(`/api/v1/items/${itemId}/progress`)
    },
    updateProgress(input: {
      metadata_item_id?: number
      resource_id?: number
      position_seconds: number
      duration_seconds?: number
      completed?: boolean
      progress_frame_data?: string
    }) {
      return request<ProgressState>('/api/v1/me/progress', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    setPreferredResource(input: {
      metadata_item_id: number
      resource_id: number
    }) {
      return request<ProgressState>('/api/v1/me/preferred-resource', {
        method: 'POST',
        body: JSON.stringify(input),
      })
    },
    continueWatching() {
      return request<CatalogUserItemEntry[]>('/api/v1/me/continue-watching')
    },
    listFavorites() {
      return request<CatalogUserItemEntry[]>('/api/v1/me/favorites')
    },
    addFavorite(itemId: number) {
      return request<CatalogUserItemEntry>(`/api/v1/me/favorites/${itemId}`, {
        method: 'POST',
      })
    },
    removeFavorite(itemId: number) {
      return request<CatalogUserItemEntry>(`/api/v1/me/favorites/${itemId}`, {
        method: 'DELETE',
      })
    },
    homeSections(limit = 12) {
      return request<HomeContentSection[]>(
        `/api/v1/home/sections?limit=${limit}`
      )
    },
    homeMediaOverview(previewLimit = 4) {
      return request<HomeMediaOverview>(
        `/api/v1/home/media-overview?preview_limit=${previewLimit}`
      )
    },
    recentlyAdded(limit = 5) {
      return request<CatalogListItem[]>(
        `/api/v1/home/recently-added?limit=${limit}`
      )
    },
  }
}
