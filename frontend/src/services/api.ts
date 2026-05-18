import {
  Repository,
  Environment,
  Deployment,
  Domain,
  RuntimeResource,
  Event,
  APIError,
  Feature,
  RepositoryDependency,
  Integration,
  IntegrationKind,
} from '../types';

const BASE_URL = (import.meta as any).env?.VITE_API_URL || 'http://localhost:8080';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!response.ok) {
    const errorBody: APIError = await response.json().catch(() => ({ error: 'Unknown API error' }));
    throw new Error(errorBody.error);
  }

  if (response.status === 204) {
    return {} as T;
  }

  return response.json();
}

export const orchestratorApi = {
  // Health
  checkHealth: () => request<{ status: string }>('/healthz'),

  // Repositories
  getRepositories: () => request<Repository[]>('/repositories'),
  getDependencies: (repoName: string) =>
    request<RepositoryDependency[]>(`/repositories/${encodeURIComponent(repoName)}/dependencies`),
  upsertRepository: (repo: Partial<Repository>) =>
    request<Repository>('/repositories', {
      method: 'POST',
      body: JSON.stringify(repo),
    }),
  addDependency: (repoName: string, dependency: { depends_on: string; required?: boolean }) =>
    request<{ status: string }>(`/repositories/${encodeURIComponent(repoName)}/dependencies`, {
      method: 'POST',
      body: JSON.stringify(dependency),
    }),
  removeDependency: (repoName: string, depName: string) =>
    request<{ status: string }>(
      `/repositories/${encodeURIComponent(repoName)}/dependencies/${encodeURIComponent(depName)}`,
      { method: 'DELETE' },
    ),

  // Environments
  getEnvironments: () => request<Environment[]>('/environments'),
  getEnvironmentDetails: (id: string) =>
    request<{
      environment: Environment;
      deployments: Deployment[];
      domains: Domain[];
      runtime: RuntimeResource[];
    }>(`/environments/${id}`),
  createEnvironment: (data: { feature: string; ttl?: string }) =>
    request<Environment>('/environments', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  deleteEnvironment: (id: string) =>
    request<{ status: string }>(`/environments/${id}`, { method: 'DELETE' }),
  suspendEnvironment: (id: string) =>
    request<{ status: string; id: string; short_id: string }>(`/environments/${id}/suspend`, {
      method: 'POST',
      body: '{}',
    }),
  resumeEnvironment: (id: string) =>
    request<{ status: string; id: string; short_id: string }>(`/environments/${id}/resume`, {
      method: 'POST',
      body: '{}',
    }),

  // Deployment actions
  reconcile: (envId: string) =>
    request<{ status: string }>('/deployments/reconcile', {
      method: 'POST',
      body: JSON.stringify({ environment_id: envId }),
    }),
  restart: (envId: string) =>
    request<{ status: string }>('/deployments/restart', {
      method: 'POST',
      body: JSON.stringify({ environment_id: envId }),
    }),

  // Features
  getFeature: (slug: string) =>
    request<{ feature: Feature; environment: Environment | null }>(`/features/${slug}`),

  // Events
  getEvents: () => request<Event[]>('/events'),

  // Integrations
  getIntegrations: (kind?: IntegrationKind) =>
    request<Integration[]>(`/integrations${kind ? `?kind=${kind}` : ''}`),
  createIntegration: (data: { kind: IntegrationKind; name: string; config?: Record<string, any>; secret?: string }) =>
    request<Integration>('/integrations', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  patchIntegration: (id: string, data: { config?: Record<string, any>; secret?: string }) =>
    request<Integration>(`/integrations/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),
  deleteIntegration: (id: string) =>
    request<{ status: string }>(`/integrations/${id}`, { method: 'DELETE' }),
  verifyIntegration: (id: string, result: { ok: boolean; detail?: string }) =>
    request<Integration>(`/integrations/${id}/verify`, {
      method: 'POST',
      body: JSON.stringify(result),
    }),
};
