export type ServiceKind = 'http' | 'worker' | 'static';
export type EnvironmentState = 'pending' | 'resolving' | 'deploying' | 'healthy' | 'degraded' | 'failed' | 'destroying' | 'destroyed';
export type DeploymentState = 'pending' | 'building' | 'running' | 'failed' | 'stopped';
export type ResourceKind = 'container' | 'network' | 'traefik_router';
export type SelectionStrategy = 'matched' | 'fallback_main' | 'fallback_latest_stable' | 'fallback_auto_create' | 'fallback_previous_compatible';
export type IntegrationKind = 'github_app' | 'buddy' | 'cloudflare' | 'registry' | 'webhook' | 'custom';

export interface Repository {
  id: string;
  name: string;
  full_name: string;
  default_branch?: string;
  service_kind: ServiceKind;
  expose_port?: number;
  buddy_project?: string;
  buddy_pipeline?: string;
  metadata?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface Feature {
  id: string;
  slug: string;
  display_name?: string;
  created_at: string;
  last_seen_at: string;
}

export interface Environment {
  id: string;
  short_id: string;
  feature_id: string;
  state: EnvironmentState;
  generation: number;
  desired_topology: any;
  docker_network: string;
  base_domain: string;
  ttl_seconds?: number;
  last_event_at: string;
  last_reconciled_at: string;
  suspended: boolean;
  created_at: string;
  updated_at: string;
  destroyed_at?: string | null;
}

export interface Deployment {
  id: string;
  environment_id: string;
  repository_id: string;
  generation: number;
  branch: string;
  commit_sha: string;
  image_ref: string;
  state: DeploymentState;
  container_name: string;
  container_id: string;
  env_vars: Record<string, string>;
  selection_strategy: SelectionStrategy;
  buddy_run_id?: string;
  last_status_at: string;
  created_at: string;
}

export interface Domain {
  id: string;
  environment_id: string;
  repository_id: string;
  hostname: string;
  target_port: number;
  tls: boolean;
}

export interface RuntimeResource {
  id: string;
  environment_id: string;
  kind: ResourceKind;
  external_id: string;
  name: string;
  state: string;
  last_seen_at: string;
}

export interface Event {
  id: string;
  source: string;
  delivery_id: string;
  event_type: string;
  action: string;
  repository: string;
  payload?: any;
  process_error?: string;
  received_at: string;
  processed_at?: string;
  attempt_count: number;
}

export interface RepositoryDependency {
  repository_id: string;
  depends_on_id: string;
  depends_on_name: string;
  depends_on_full_name: string;
  required: boolean;
}

export interface Integration {
  id: string;
  kind: IntegrationKind;
  name: string;
  config: Record<string, any>;
  has_secret: boolean;
  last_verified_at?: string | null;
  last_error?: string;
  created_at: string;
  updated_at: string;
}

export interface EnvironmentDetail {
  environment: Environment;
  deployments: Deployment[];
  domains: Domain[];
  runtime: RuntimeResource[];
}

export interface APIError {
  error: string;
}
