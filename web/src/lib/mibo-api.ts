import { useAuthStore } from "#/stores/auth-store"

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
  created_at: string
  updated_at: string
}

export type AdminUser = User

export type CreateAdminUserInput = {
  username: string
  password: string
  role: "user" | "admin"
}

export type LoginResult = {
  token: string
  expires_at: string
  user: User
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

export type ConsoleStatus =
  | "ok"
  | "warning"
  | "error"
  | "unknown"
  | "unavailable"
  | "not_configured"

export type ConsoleServerSummary = {
  name: string
  service: string
  status: ConsoleStatus
  version: string
  update_status: string
  api_address: string
  port: number
  uptime_seconds: number
  storage_provider: string
  storage_root: string
  database_driver: string
}

export type ConsoleAccessAddress = {
  kind: "local" | "lan" | "remote"
  label: string
  url?: string
  status: ConsoleStatus | "available"
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
  severity: "info" | "warning" | "error"
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
  kind: "route" | "mutation" | "unsupported"
  route?: string
  method?: string
  endpoint?: string
  disabled: boolean
  disabled_reason?: string
  risk: "safe" | "expensive" | "danger"
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
}

export type Library = {
  id: number
  name: string
  media_source_id: number
  root_path: string
  status: string
  scanner_enabled: boolean
  probe_status: string
  probe_summary_json?: string
  paths?: LibraryPath[]
  policies?: LibraryPolicies
  probe_summary?: SourceProbeSummary
  collections?: SourceCollection[]
}

export type LibraryDetail = Library & {
  metadata_items_count: number
  inventory_files_count: number
}

export type LibraryPath = {
  id: number
  library_id: number
  media_source_id: number
  root_path: string
  display_name: string
  enabled: boolean
}

export type SourceContentClass = "video" | "audio" | "text" | "image" | "other"

export type SourceProbeSummary = {
  status: string
  dominant_class: SourceContentClass | ""
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
  preferred_image_language: string
  metadata_country_code: string
  metadata_profile_id?: number
  metadata_profile_name?: string
}

export type LibraryMetadataStrategy = {
  library_id: number
  template_profile_id?: number
  template_profile_name?: string
  search_provider_ids: number[]
  detail_provider_ids: number[]
  image_provider_ids: number[]
  people_provider_ids: number[]
  hierarchy_provider_ids: number[]
  preferred_metadata_language?: string
  preferred_image_language?: string
  metadata_country_code?: string
}

export type LibraryMetadataStrategyInput = {
  template_profile_id?: number
  search_provider_ids: number[]
  detail_provider_ids: number[]
  image_provider_ids: number[]
  people_provider_ids: number[]
  hierarchy_provider_ids: number[]
  preferred_metadata_language?: string
  preferred_image_language?: string
  metadata_country_code?: string
}

export type LibraryPlaybackPolicy = {
  resume_enabled: boolean
  min_resume_pct: number
  max_resume_pct: number
  min_resume_duration_seconds: number
}

export type LibrarySubtitlePolicy = {
  external_sidecars_enabled: boolean
  preferred_languages: string[]
  require_perfect_match: boolean
  save_with_media: boolean
  tolerate_unavailable_subtitles: boolean
  skip_if_embedded_subtitles_present: boolean
  skip_if_audio_track_matches: boolean
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
  rule_type: "filename_token" | "directory_segment" | "path_pattern"
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
  rule_type: ScanExclusionRule["rule_type"]
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
  insecure_skip?: boolean
}

export type MediaSourceConfigInput = {
  openlist?: OpenListMediaSourceConfig
}

export type OpenListMediaSourceConfigView = {
  base_url: string
  username?: string
  timeout?: string
  insecure_skip: boolean
  has_password: boolean
  has_token: boolean
}

export type MediaSourceConfigView = {
  openlist?: OpenListMediaSourceConfigView
}

export type MediaSource = {
  id: number
  name: string
  provider: string
  storage_ref: string
  root_path: string
  config?: MediaSourceConfigView
  capabilities_json: string
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
}

export type CatalogSelectedImage = {
  image_type: string
  url: string
  language?: string
  width?: number
  height?: number
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
  source_kind?: "catalog" | "inventory_file"
  inventory_file_id?: number
  maturity_state?:
    | "discovered"
    | "classified"
    | "enriched"
    | "review_required"
    | string
  organizing?: boolean
  organizing_summary?: CatalogOrganizingSummary
  storage_path?: string
  type: string
  title: string
  original_title?: string
  sort_title?: string
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
}

export type CatalogOrganizingSummary = {
  state:
    | "organizing"
    | "partial_ready"
    | "ready"
    | "failed"
    | "review_required"
    | string
  message: string
  stage?: string
  severity?: "info" | "warning" | "error" | string
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
  display_name?: string
  edition?: string
  quality_label?: string
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
  display_name?: string
  edition?: string
  quality_label?: string
  duration_seconds?: number
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
  mode?: "copy" | "move"
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
  sort_title?: string
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
  preferred_image_language?: string
  search_providers?: CatalogMetadataPlanProviderSummary[]
  detail_providers?: CatalogMetadataPlanProviderSummary[]
  image_providers?: CatalogMetadataPlanProviderSummary[]
  people_providers?: CatalogMetadataPlanProviderSummary[]
  hierarchy_providers?: CatalogMetadataPlanProviderSummary[]
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
  availability_status: string
  governance_status: string
  selected_images?: CatalogSelectedImage[]
  image_candidates?: CatalogSelectedImage[]
  external_identities?: CatalogExternalIdentity[]
  source_evidence?: CatalogSourceEvidence[]
  field_states?: CatalogFieldState[]
	resources?: MediaResourceDetail[]
	classification_decisions?: CatalogClassificationDecision[]
  classification_rules?: CatalogClassificationRuleSummary[]
  recommended_children?: CatalogListItem[]
  metadata_operation?: CatalogMetadataOperation
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

export type HealthSeverity = "info" | "warning" | "error" | "blocking"

export type HealthSummary = {
  status: "healthy" | "warning" | "error" | "blocking"
  issue_count: number
  blocking_count: number
  error_count: number
  warning_count: number
  issues: HealthIssue[]
}

export type HealthIssue = {
  id: string
  severity: HealthSeverity
  reason_code: string
  scope: string
  title: string
  message: string
  impact: HealthImpact
  affected: HealthAffected
  actions: HealthAction[]
  technical_detail: HealthTechnicalDetail
  first_seen_at?: string
  last_seen_at?: string
  latest_job_id?: number
}

export type HealthImpact = {
  blocks_scan: boolean
  blocks_home_visibility: boolean
  blocks_playback: boolean
  blocks_metadata: boolean
  affected_metadata_items: number
  affected_files: number
}

export type HealthAffected = {
  media_sources: HealthMediaSourceRef[] | null
  libraries: HealthLibraryRef[] | null
  jobs: HealthJobRef[] | null
}

export type HealthMediaSourceRef = {
  id: number
  name: string
  provider: string
  root_path: string
}

export type HealthLibraryRef = {
  id: number
  name: string
  type: string
  status: string
  media_source_id: number
  root_path: string
}

export type HealthJobRef = {
  id: number
  kind: string
  status: string
  attempts: number
  created_at: string
  updated_at: string
  finished_at?: string
  payload_json?: string
}

export type HealthAction = {
  type: string
  label: string
  description?: string
  href?: string
  media_source_id?: number
  job_id?: number
  library_ids?: number[]
}

export type HealthTechnicalDetail = {
  job_kind?: string
  job_status?: string
  payload_json?: string
  error_message?: string
}

function normalizeHealthIssues(issues: HealthIssue[]) {
  return issues.map((issue) => ({
    ...issue,
    affected: {
      media_sources: issue.affected?.media_sources ?? [],
      libraries: issue.affected?.libraries ?? [],
      jobs: issue.affected?.jobs ?? [],
    },
  }))
}

export type MediaSourceValidationResult = {
  media_source_id: number
  status: string
  message: string
}

export type HealthIssueRescanResult = {
  issue_id: string
  jobs: HealthJobRef[]
}

export type HealthIssueIgnoreResult = {
  issue_id: string
  status: string
}

export type MetadataProviderSettings = {
  configured: boolean
  api_key_masked: boolean
  base_url: string
  image_base_url?: string
  language: string
  timeout: string
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
  detail_provider_ids: number[]
  image_provider_ids: number[]
  people_provider_ids: number[]
  hierarchy_provider_ids: number[]
  preferred_metadata_language?: string
  preferred_image_language?: string
  fallback_enabled: boolean
}

export type MetadataProfileInput = {
  name: string
  description?: string
  search_provider_ids: number[]
  detail_provider_ids: number[]
  image_provider_ids: number[]
  people_provider_ids: number[]
  hierarchy_provider_ids: number[]
  preferred_metadata_language?: string
  preferred_image_language?: string
  fallback_enabled?: boolean
}

export type MetadataProviderInput = {
  api_key?: string
  clear_api_key?: boolean
  base_url?: string
  image_base_url?: string
  language?: string
  timeout?: string
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
  remote_ip_filter_mode: "allow" | "block"
  public_http_port: number
  public_https_port: number
  external_domain: string
  trust_proxy_headers: boolean
  ssl_certificate_path: string
  certificate_password: NetworkCertificatePasswordState
  secure_connection_mode: "disabled" | "preferred" | "required"
  automatic_port_mapping: boolean
  max_video_streams: "unlimited" | "1" | "2" | "4" | "8"
  remote_streaming_bitrate_limit:
    | "unlimited"
    | "4mbps"
    | "8mbps"
    | "12mbps"
    | "20mbps"
  network_request_protocol: "auto" | "ipv4" | "ipv6"
  effective_status: NetworkSettingsStatus
}

export type NetworkSettingsInput = Omit<
  NetworkSettings,
  "certificate_password" | "effective_status"
> & {
  certificate_password?: string
  clear_certificate_password?: boolean
}

export type DiscoveryQuery = {
  scope?: "all" | "library"
  library_id?: number
  q?: string
  type?: "all" | "movie" | "show" | "episode"
  genre?: string
  region?: string
  year?: number
  min_rating?: number
  watched_state?: "all" | "unwatched" | "in_progress" | "watched"
  organizing_state?: "all" | "organized" | "unorganized"
  sort?: "recent" | "title" | "year" | "watch_status"
  sort_direction?: "asc" | "desc"
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
  sort: "recent" | "title" | "year" | "watch_status"
  sort_direction: "asc" | "desc"
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
  sort: "recent" | "title" | "year" | "watch_status"
  last_used_at: string
}

export type ClientProfile = "web" | "mobile" | "tv"

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
  kind: "direct" | "fallback" | "unplayable"
  client_profile: ClientProfile
  selected_by: string
  fallback_kind?: string
  reasons: DecisionReason[]
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
  quality_label?: string
  edition?: string
  video_codec: string
  width?: number
  height?: number
  audio_tracks: Track[]
  subtitle_tracks: Track[]
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

export type ScheduleFrequencyKind = "daily" | "weekly" | "monthly"

export type ScheduleScopeKind = "global" | "library"

export type ScheduleRunStatus = "queued" | "running" | "completed" | "failed"

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
  latest_run_status?: ScheduleRunStatus | ""
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

export const TOKEN_STORAGE_KEY = "mibo-web-token"

let isRedirectingToLogin = false

export class ApiError extends Error {
  status: number
  code: string

  constructor(status: number, error: ApiErrorShape) {
    super(error.message)
    this.name = "ApiError"
    this.status = status
    this.code = error.code
  }
}

export function getApiBaseUrl() {
  return (
    (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(
      /\/$/,
      ""
    ) ?? ""
  )
}

export function buildApiUrl(pathname: string) {
  if (pathname.startsWith("http://") || pathname.startsWith("https://")) {
    return pathname
  }

  return `${getApiBaseUrl()}${pathname.startsWith("/") ? pathname : `/${pathname}`}`
}

function handleUnauthorizedResponse(token?: string | null) {
  if (!token || typeof window === "undefined") {
    return
  }

  const { pathname, search, hash } = window.location

  useAuthStore.getState().clearSession()

  if (pathname === "/login" || isRedirectingToLogin) {
    return
  }

  isRedirectingToLogin = true

  const redirect = `${pathname}${search}${hash}`
  const loginUrl = new URL("/login", window.location.origin)
  loginUrl.searchParams.set("redirect", redirect)
  window.location.replace(loginUrl.toString())
}

export function createMiboApi(options: ApiOptions) {
  const baseUrl = options.baseUrl.replace(/\/$/, "")

  async function request<T>(pathname: string, init?: RequestInit): Promise<T> {
    const headers = new Headers(init?.headers)

    if (!headers.has("Content-Type") && init?.body !== undefined) {
      headers.set("Content-Type", "application/json")
    }

    if (options.token) {
      headers.set("Authorization", `Bearer ${options.token}`)
    }

    let response: Response
    try {
      response = await fetch(`${baseUrl}${pathname}`, {
        ...init,
        headers,
      })
    } catch {
      throw new ApiError(0, {
        code: "network_error",
        message: "无法连接后端服务，请确认 Mibo 服务已启动。",
      })
    }

    if (response.status === 401) {
      handleUnauthorizedResponse(options.token)
    }

    let payload: Envelope<T> | null = null
    try {
      payload = (await response.json()) as Envelope<T>
    } catch {
      if (!response.ok) {
        throw new ApiError(response.status, {
          code: "request_failed",
          message: `请求失败，状态码 ${response.status}`,
        })
      }
    }

    if (!response.ok || payload?.error) {
      throw new ApiError(
        response.status,
        payload?.error ?? {
          code: "request_failed",
          message: `请求失败，状态码 ${response.status}`,
        }
      )
    }

    if (payload?.data === undefined) {
      throw new ApiError(response.status, {
        code: "missing_payload",
        message: "服务端返回了空数据",
      })
    }

    return payload.data
  }

  return {
    getSetupStatus() {
      return request<SetupStatus>("/api/v1/setup/status")
    },
    register(username: string, password: string) {
      return request<User>("/api/v1/auth/register", {
        method: "POST",
        body: JSON.stringify({ username, password }),
      })
    },
    login(username: string, password: string) {
      return request<LoginResult>("/api/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({ username, password }),
      })
    },
    logout() {
      return request<{ status: string }>("/api/v1/auth/logout", {
        method: "POST",
      })
    },
    listLoginSessions() {
      return request<LoginSession[]>("/api/v1/auth/sessions")
    },
    revokeLoginSession(sessionId: number) {
      return request<{ id: number; status: string }>(
        `/api/v1/auth/sessions/${sessionId}`,
        { method: "DELETE" }
      )
    },
    revokeOtherLoginSessions() {
      return request<{ status: string }>("/api/v1/auth/sessions/others", {
        method: "DELETE",
      })
    },
    me() {
      return request<User>("/api/v1/me")
    },
    listMediaSources() {
      return request<MediaSource[]>("/api/v1/media-sources")
    },
    browseStorageProvider(provider: string, path?: string, refresh = false) {
      return request<StorageBrowseResult>("/api/v1/storage/providers/browse", {
        method: "POST",
        body: JSON.stringify({ provider, path, refresh }),
      })
    },
    browseOpenList(input: {
      path?: string
      refresh?: boolean
      config: OpenListMediaSourceConfig
    }) {
      return request<StorageBrowseResult>("/api/v1/storage/openlist/browse", {
        method: "POST",
        body: JSON.stringify(input),
      })
    },
    testOpenListConnection(input: { config: OpenListMediaSourceConfig }) {
      return request<OpenListTestResult>("/api/v1/storage/openlist/test", {
        method: "POST",
        body: JSON.stringify(input),
      })
    },
    createMediaSource(input: {
      provider: string
      name: string
      root_path: string
      storage_ref?: string
      config?: MediaSourceConfigInput
    }) {
      return request<MediaSource>("/api/v1/media-sources", {
        method: "POST",
        body: JSON.stringify(input),
      })
    },
    updateMediaSource(
      mediaSourceId: number,
      input: {
        name: string
        root_path: string
        storage_ref?: string
        config?: MediaSourceConfigInput
      }
    ) {
      return request<MediaSource>(`/api/v1/media-sources/${mediaSourceId}`, {
        method: "PATCH",
        body: JSON.stringify(input),
      })
    },
    deleteMediaSource(mediaSourceId: number) {
      return request<{ id: number; status: string; type: string }>(
        `/api/v1/media-sources/${mediaSourceId}`,
        {
          method: "DELETE",
        }
      )
    },
    browseMediaSource(mediaSourceId: number, path?: string, refresh = false) {
      return request<StorageBrowseResult>("/api/v1/media-sources/browse", {
        method: "POST",
        body: JSON.stringify({ id: mediaSourceId, path, refresh }),
      })
    },
    validateMediaSource(mediaSourceId: number) {
      return request<MediaSourceValidationResult>(
        `/api/v1/media-sources/${mediaSourceId}/validate`,
        { method: "POST" }
      )
    },
    listLibraries() {
      return request<Library[]>("/api/v1/libraries")
    },
    getHealthSummary() {
      return request<HealthSummary>("/api/v1/health/summary")
    },
    listHealthIssues() {
      return request<HealthIssue[]>("/api/v1/health/issues").then(
        normalizeHealthIssues
      )
    },
    rescanHealthIssueLibraries(issueId: string) {
      return request<HealthIssueRescanResult>(
        `/api/v1/health/issues/${encodeURIComponent(issueId)}/rescan`,
        { method: "POST" }
      )
    },
    ignoreHealthIssue(issueId: string) {
      return request<HealthIssueIgnoreResult>(
        `/api/v1/health/issues/${encodeURIComponent(issueId)}/ignore`,
        { method: "POST" }
      )
    },
    getConsoleSummary() {
      return request<ConsoleSummary>("/api/v1/admin/console")
    },
    getIngestDiagnostics() {
      return request<IngestDiagnosticsResult>(
        "/api/v1/admin/ingest/diagnostics"
      )
    },
    retryIngestStage(stageId: number) {
      return request<IngestRetryResult>(
        `/api/v1/admin/ingest/stages/${stageId}/retry`,
        {
          method: "POST",
        }
      )
    },
    resolveIngestReviewStage(stageId: number) {
      return request<IngestResolveReviewResult>(
        `/api/v1/admin/ingest/stages/${stageId}/resolve-review`,
        {
          method: "POST",
        }
      )
    },
    runConsoleAction(actionId: string) {
      const actionEndpoints: Record<string, string> = {
        "scan-libraries": "/api/v1/admin/console/actions/scan-libraries",
      }
      const endpoint = actionEndpoints[actionId]
      if (!endpoint) {
        throw new Error("unsupported console action")
      }
      return request<ConsoleActionResult>(endpoint, { method: "POST" })
    },
    listAdminLogs() {
      return request<AdminLogFile[]>("/api/v1/admin/logs")
    },
    listAdminUsers() {
      return request<AdminUser[]>("/api/v1/admin/users")
    },
    createAdminUser(input: CreateAdminUserInput) {
      return request<AdminUser>("/api/v1/admin/users", {
        method: "POST",
        body: JSON.stringify(input),
      })
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
          method: "DELETE",
        }
      )
    },
    listMetadataProviderInstances() {
      return request<MetadataProviderInstance[]>(
        "/api/v1/settings/metadata/providers"
      )
    },
    createMetadataProviderInstance(input: MetadataProviderInstanceInput) {
      return request<MetadataProviderInstance>(
        "/api/v1/settings/metadata/providers",
        {
          method: "POST",
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
          method: "PATCH",
          body: JSON.stringify(input),
        }
      )
    },
    listMetadataProfiles() {
      return request<MetadataProfile[]>("/api/v1/settings/metadata/profiles")
    },
    createMetadataProfile(input: MetadataProfileInput) {
      return request<MetadataProfile>("/api/v1/settings/metadata/profiles", {
        method: "POST",
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
          method: "PATCH",
          body: JSON.stringify(input),
        }
      )
    },
    getNetworkSettings() {
      return request<NetworkSettings>("/api/v1/settings/network")
    },
    updateNetworkSettings(input: NetworkSettingsInput) {
      return request<NetworkSettings>("/api/v1/settings/network", {
        method: "PUT",
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
      scan?: LibraryScanPolicy
      metadata?: LibraryMetadataPolicy
      metadata_strategy?: LibraryMetadataStrategyInput
      playback?: LibraryPlaybackPolicy
      subtitle?: LibrarySubtitlePolicy
      scan_exclusion_rules?: ScanExclusionRuleInput[]
    }) {
      return request<{ library: LibraryDetail }>("/api/v1/libraries", {
        method: "POST",
        body: JSON.stringify(input),
      })
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
        method: "POST",
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
          method: "PATCH",
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
          method: "PUT",
          body: JSON.stringify(input),
        }
      )
    },
    updateLibraryScanPolicy(libraryId: number, input: LibraryScanPolicy) {
      return request<LibraryScanPolicy>(
        `/api/v1/libraries/${libraryId}/policies/scan`,
        {
          method: "PUT",
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
          method: "PUT",
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
          method: "PUT",
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
          method: "PUT",
          body: JSON.stringify(input),
        }
      )
    },
    deleteLibrary(libraryId: number) {
      return request<{ id: number; status: string; type: string }>(
        `/api/v1/libraries/${libraryId}`,
        {
          method: "DELETE",
        }
      )
    },
    scanLibrary(libraryId: number, mode: "full" | "changed" = "full") {
      return request<{ queued: boolean; mode: "full" | "changed" }>(
        `/api/v1/libraries/${libraryId}/scan`,
        {
          method: "POST",
          body: JSON.stringify({ mode }),
        }
      )
    },
    listScanExclusions(filters?: { libraryId?: number; enabled?: boolean }) {
      const query = new URLSearchParams()
      if (typeof filters?.libraryId === "number" && filters.libraryId > 0) {
        query.set("library_id", String(filters.libraryId))
      }
      if (typeof filters?.enabled === "boolean") {
        query.set("enabled", String(filters.enabled))
      }
      const queryString = query.toString()
      return request<ScanExclusionsView>(
        `/api/v1/scan-exclusions${queryString ? `?${queryString}` : ""}`
      )
    },
    setScanExclusionEnabled(exclusionId: number, enabled: boolean) {
      return request<ScanExclusion>(`/api/v1/scan-exclusions/${exclusionId}`, {
        method: "PATCH",
        body: JSON.stringify({ enabled }),
      })
    },
    listScanExclusionRules() {
      return request<ScanExclusionRule[]>("/api/v1/scan-exclusion-rules")
    },
    createScanExclusionRule(input: ScanExclusionRuleInput) {
      return request<ScanExclusionRule>("/api/v1/scan-exclusion-rules", {
        method: "POST",
        body: JSON.stringify(input),
      })
    },
    updateScanExclusionRule(ruleId: number, input: ScanExclusionRuleInput) {
      return request<ScanExclusionRule>(
        `/api/v1/scan-exclusion-rules/${ruleId}`,
        {
          method: "PATCH",
          body: JSON.stringify(input),
        }
      )
    },
    setScanExclusionRuleEnabled(ruleId: number, enabled: boolean) {
      return request<ScanExclusionRule>(
        `/api/v1/scan-exclusion-rules/${ruleId}`,
        {
          method: "PATCH",
          body: JSON.stringify({ enabled }),
        }
      )
    },
    deleteScanExclusionRule(ruleId: number) {
      return request<{ status: string }>(
        `/api/v1/scan-exclusion-rules/${ruleId}`,
        {
          method: "DELETE",
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
          method: "PUT",
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
      reason = "advertisement"
    ) {
      return request<FilenameExclusionRule>(
        `/api/v1/inventory-files/${fileId}/filename-exclusion-rule`,
        {
          method: "POST",
          body: JSON.stringify({ reason }),
        }
      )
    },
    setFilenameExclusionRuleEnabled(ruleId: number, enabled: boolean) {
      return request<FilenameExclusionRule>(
        `/api/v1/filename-exclusion-rules/${ruleId}`,
        {
          method: "PATCH",
          body: JSON.stringify({ enabled }),
        }
      )
    },
    restoreFilenameExclusionMatch(ruleId: number, inventoryFileId: number) {
      return request<FilenameExclusionRestore>(
        `/api/v1/filename-exclusion-rules/${ruleId}/restores`,
        {
          method: "POST",
          body: JSON.stringify({ inventory_file_id: inventoryFileId }),
        }
      )
    },
    listLibraryItems(
      libraryId: number,
      queryOptions?: {
        type?: "all" | "movie" | "show"
        year?: number
        sort?: "recent" | "title" | "year" | "watch_status"
        limit?: number
      }
    ) {
      const query = new URLSearchParams()

      if (queryOptions?.type) {
        query.set("type", queryOptions.type)
      }
      if (typeof queryOptions?.year === "number") {
        query.set("year", String(queryOptions.year))
      }
      if (queryOptions?.sort) {
        query.set("sort", queryOptions.sort)
      }
      if (typeof queryOptions?.limit === "number") {
        query.set("limit", String(queryOptions.limit))
      }

      const queryString = query.toString()
      return request<CatalogListItem[]>(
        `/api/v1/libraries/${libraryId}/items${queryString ? `?${queryString}` : ""}`
      )
    },
    discoverMedia(queryOptions?: DiscoveryQuery) {
      const query = new URLSearchParams()

      if (queryOptions?.scope) query.set("scope", queryOptions.scope)
      if (typeof queryOptions?.library_id === "number") {
        query.set("library_id", String(queryOptions.library_id))
      }
      if (queryOptions?.q) query.set("q", queryOptions.q)
      if (queryOptions?.type) query.set("type", queryOptions.type)
      if (queryOptions?.genre) query.set("genre", queryOptions.genre)
      if (queryOptions?.region) query.set("region", queryOptions.region)
      if (typeof queryOptions?.year === "number") {
        query.set("year", String(queryOptions.year))
      }
      if (typeof queryOptions?.min_rating === "number") {
        query.set("min_rating", String(queryOptions.min_rating))
      }
      if (queryOptions?.watched_state) {
        query.set("watched_state", queryOptions.watched_state)
      }
      if (queryOptions?.organizing_state) {
        query.set("organizing_state", queryOptions.organizing_state)
      }
      if (queryOptions?.sort) query.set("sort", queryOptions.sort)
      if (queryOptions?.sort_direction) {
        query.set("sort_direction", queryOptions.sort_direction)
      }
      if (typeof queryOptions?.limit === "number") {
        query.set("limit", String(queryOptions.limit))
      }
      if (typeof queryOptions?.offset === "number") {
        query.set("offset", String(queryOptions.offset))
      }

      const queryString = query.toString()
      return request<CatalogDiscoveryResponse>(
        `/api/v1/discovery${queryString ? `?${queryString}` : ""}`
      )
    },
    listSearchHistory(limit = 8) {
      return request<SearchHistoryEntry[]>(
        `/api/v1/search/history?limit=${limit}`
      )
    },
    getMetadataItem(itemId: number, options?: { libraryId?: number }) {
      const query = new URLSearchParams()
      if (typeof options?.libraryId === "number") {
        query.set("library_id", String(options.libraryId))
      }
      const queryString = query.toString()
      return request<CatalogItemDetail>(
        `/api/v1/items/${itemId}${queryString ? `?${queryString}` : ""}`
      )
    },
    listMetadataItemResources(itemId: number, options?: { libraryId?: number }) {
      const query = new URLSearchParams()
      if (typeof options?.libraryId === "number") {
        query.set("library_id", String(options.libraryId))
      }
      const queryString = query.toString()
      return request<MetadataResourceDetail[]>(
        `/api/v1/items/${itemId}/resources${queryString ? `?${queryString}` : ""}`
      )
    },
    getInventoryFilePlayback(
      fileId: number,
      options?: { clientProfile?: ClientProfile }
    ) {
      const params = new URLSearchParams()
      if (options?.clientProfile)
        params.set("client_profile", options.clientProfile)
      const queryString = params.toString()
      return request<PlaybackSource>(
        `/api/v1/inventory-files/${fileId}/playback${queryString ? `?${queryString}` : ""}`
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
    updateCatalogGovernanceField(
      itemId: number,
      input: {
        field_key: string
        value?: unknown
        lock?: boolean
        lock_reason?: string
        force?: boolean
      }
    ) {
      return request<CatalogGovernanceWorkspace>(
        `/api/v1/items/${itemId}/governance/fields`,
        {
          method: "PUT",
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
          method: "PUT",
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
          method: "POST",
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
          method: "PATCH",
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
      if (typeof options?.libraryId === "number") {
        query.set("library_id", String(options.libraryId))
      }
      if (options?.role) query.set("role", options.role)
      if (typeof options?.segmentIndex === "number") {
        query.set("segment_index", String(options.segmentIndex))
      }
      const queryString = query.toString()
      return request<CatalogMetadataOperation>(
        `/api/v1/items/${metadataItemId}/governance/resources/${resourceId}/links/${targetMetadataItemId}${queryString ? `?${queryString}` : ""}`,
        { method: "DELETE" }
      )
    },
    mergeGovernanceMetadata(metadataItemId: number, input: MetadataMergeInput) {
      return request<CatalogMetadataOperation>(
        `/api/v1/items/${metadataItemId}/governance/metadata-merge`,
        {
          method: "POST",
          body: JSON.stringify(input),
        }
      )
    },
    splitGovernanceMetadata(metadataItemId: number, input: MetadataSplitInput) {
      return request<CatalogMetadataOperation>(
        `/api/v1/items/${metadataItemId}/governance/metadata-split`,
        {
          method: "POST",
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
          method: "PUT",
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
          method: "PUT",
          body: JSON.stringify(input),
        }
      )
    },
    markInventoryFileScanExclusion(fileId: number, reason = "advertisement") {
      return request<ScanExclusion>(
        `/api/v1/inventory-files/${fileId}/scan-exclusion`,
        {
          method: "POST",
          body: JSON.stringify({ reason }),
        }
      )
    },
    reprobeInventoryFile(fileId: number) {
      return request<AcceptedResult>(
        `/api/v1/inventory-files/${fileId}/probe`,
        {
          method: "POST",
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

      if (typeof filters?.limit === "number") {
        query.set("limit", String(filters.limit))
      }
      if (typeof filters?.offset === "number") {
        query.set("offset", String(filters.offset))
      }
      if (filters?.status) {
        query.set("status", filters.status)
      }
      if (typeof filters?.library_id === "number") {
        query.set("library_id", String(filters.library_id))
      }

      const queryString = query.toString()
      return request<WorkflowRunStatusView[]>(
        `/api/v1/workflows${queryString ? `?${queryString}` : ""}`
      )
    },
    getWorkflow(workflowId: number) {
      return request<WorkflowRunStatusView>(`/api/v1/workflows/${workflowId}`)
    },
    getWorkflowDiagnostics() {
      return request<WorkflowDiagnostics>("/api/v1/workflows/diagnostics")
    },
    listSchedules() {
      return request<Schedule[]>("/api/v1/schedules")
    },
    getSchedule(scheduleId: number) {
      return request<Schedule>(`/api/v1/schedules/${scheduleId}`)
    },
    createSchedule(input: ScheduleMutationInput) {
      return request<Schedule>("/api/v1/schedules", {
        method: "POST",
        body: JSON.stringify(input),
      })
    },
    updateSchedule(scheduleId: number, input: ScheduleMutationInput) {
      return request<Schedule>(`/api/v1/schedules/${scheduleId}`, {
        method: "PATCH",
        body: JSON.stringify(input),
      })
    },
    toggleSchedule(scheduleId: number, enabled: boolean) {
      return request<Schedule>(`/api/v1/schedules/${scheduleId}/toggle`, {
        method: "POST",
        body: JSON.stringify({ enabled }),
      })
    },
    runScheduleNow(scheduleId: number) {
      return request<ScheduleRunNowResult>(
        `/api/v1/schedules/${scheduleId}/run`,
        {
          method: "POST",
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
      }
    ) {
      const query = new URLSearchParams({
        client_profile: playbackOptions.clientProfile,
      })

      if (typeof playbackOptions.resourceId === "number") {
        query.set("resource_id", String(playbackOptions.resourceId))
      }
      if (typeof playbackOptions.libraryId === "number") {
        query.set("library_id", String(playbackOptions.libraryId))
      }

      return request<PlaybackSource>(
        `/api/v1/items/${itemId}/playback?${query.toString()}`
      )
    },
    getMetadataPlayback(
      metadataItemId: number,
      playbackOptions: {
        resourceId?: number
        libraryId?: number
        clientProfile: ClientProfile
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
      return request<ProgressState>("/api/v1/me/progress", {
        method: "POST",
        body: JSON.stringify(input),
      })
    },
    setPreferredResource(input: {
      metadata_item_id: number
      resource_id: number
    }) {
      return request<ProgressState>("/api/v1/me/preferred-resource", {
        method: "POST",
        body: JSON.stringify(input),
      })
    },
    continueWatching() {
      return request<CatalogUserItemEntry[]>("/api/v1/me/continue-watching")
    },
    listFavorites() {
      return request<CatalogUserItemEntry[]>("/api/v1/me/favorites")
    },
    addFavorite(itemId: number) {
      return request<CatalogUserItemEntry>(`/api/v1/me/favorites/${itemId}`, {
        method: "POST",
      })
    },
    removeFavorite(itemId: number) {
      return request<CatalogUserItemEntry>(`/api/v1/me/favorites/${itemId}`, {
        method: "DELETE",
      })
    },
    homeSections(limit = 12) {
      return request<HomeContentSection[]>(`/api/v1/home/sections?limit=${limit}`)
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
