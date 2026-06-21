const API_BASE = '/api/v1';

interface RequestOptions {
  method?: string;
  body?: unknown;
  token?: string;
}

let isRefreshing = false;
let refreshPromise: Promise<string | null> | null = null;

async function tryRefreshToken(): Promise<string | null> {
  const refreshToken = localStorage.getItem('refreshToken');
  if (!refreshToken) return null;

  try {
    const response = await fetch(`${API_BASE}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (response.ok) {
      const data = await response.json();
      localStorage.setItem('token', data.access_token);
      localStorage.setItem('refreshToken', data.refresh_token);
      return data.access_token;
    }
  } catch {
    // Refresh failed
  }

  localStorage.removeItem('token');
  localStorage.removeItem('refreshToken');
  return null;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, token } = options;

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  let response = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  // If 401, try to refresh token
  if (response.status === 401 && token) {
    if (!isRefreshing) {
      isRefreshing = true;
      refreshPromise = tryRefreshToken();
    }

    const newToken = await refreshPromise;
    isRefreshing = false;
    refreshPromise = null;

    if (newToken) {
      headers['Authorization'] = `Bearer ${newToken}`;
      response = await fetch(`${API_BASE}${path}`, {
        method,
        headers,
        body: body ? JSON.stringify(body) : undefined,
      });
    }
  }

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }));
    throw new Error(error.error || `HTTP ${response.status}`);
  }

  return response.json();
}

export interface User {
  id: string;
  email: string;
  name: string;
  created_at: string;
}

export interface Org {
  id: string;
  name: string;
  slug: string;
  plan: string;
  created_at: string;
}

export interface Connection {
  id: string;
  org_id: string;
  type: string;
  name: string;
  status: string;
  agent_token?: string;
  last_seen_at?: string;
  created_at: string;
}

export interface Container {
  id: string;
  connection_id: string;
  runtime_id: string;
  name: string;
  image: string;
  state: string;
  labels?: Record<string, string>;
  ports?: Array<{ host_port: string; container_port: string; protocol: string }>;
  created_at: string;
  updated_at: string;
}

export interface Network {
  id: string;
  connection_id: string;
  name: string;
  driver: string;
  scope: string;
  subnet: string;
  created_at: string;
}

export interface Edge {
  id: string;
  connection_id: string;
  src_container_id: string;
  dst_container_id?: string;
  dst_ip: string;
  dst_port: number;
  protocol: string;
  first_seen: string;
  last_seen: string;
}

export interface TopologyData {
  containers: Container[];
  networks: Network[];
  edges: Edge[];
}

export interface FlowEvent {
  timestamp: string;
  connection_id: string;
  src_ip: string;
  dst_ip: string;
  src_port: number;
  dst_port: number;
  protocol: string;
  bytes: number;
  packets: number;
  latency_ms: number;
}

export interface VulnScan {
  id: string;
  connection_id: string;
  image: string;
  image_id?: string;
  scan_time: string;
  critical_count: number;
  high_count: number;
  medium_count: number;
  low_count: number;
  total_count: number;
  created_at: string;
}

export interface Vulnerability {
  id: string;
  scan_id: string;
  vuln_id: string;
  severity: string;
  package: string;
  version: string;
  fixed_in?: string;
  title?: string;
  description?: string;
}

export interface Alert {
  id: string;
  rule_id: string;
  rule_name: string;
  severity: string;
  status: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  starts_at: string;
  ends_at?: string;
  fired_count: number;
}

export interface AlertRule {
  id: string;
  name: string;
  description: string;
  severity: string;
  condition: {
    type: string;
    metric: string;
    threshold: number;
    operator: string;
    duration?: string;
  };
  channels: string[];
  enabled: boolean;
}

export interface HealthService {
  name: string;
  status: string;
  latency_ms: number;
  message?: string;
}

export interface HealthResponse {
  status: string;
  services: HealthService[];
}

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
}

export interface MemberWithUser {
  user_id: string;
  org_id: string;
  role: string;
  created_at: string;
  user: User;
}

export const api = {
  auth: {
    register: (email: string, name: string, password: string) =>
      request<User>('/auth/register', { method: 'POST', body: { email, name, password } }),

    login: (email: string, password: string) =>
      request<TokenResponse>('/auth/login', { method: 'POST', body: { email, password } }),

    refresh: (refreshToken: string) =>
      request<TokenResponse>('/auth/refresh', { method: 'POST', body: { refresh_token: refreshToken } }),

    me: (token: string) =>
      request<User>('/auth/me', { token }),
  },

  orgs: {
    list: (token: string) =>
      request<Org[]>('/orgs', { token }),

    create: (token: string, name: string, slug: string) =>
      request<{ org: Org }>('/orgs', { method: 'POST', body: { name, slug }, token }),

    get: (token: string, orgId: string) =>
      request<Org>(`/orgs/${orgId}`, { token }),

    listMembers: (token: string, orgId: string) =>
      request<MemberWithUser[]>(`/orgs/${orgId}/members`, { token }),
  },

  connections: {
    list: (token: string, orgId: string) =>
      request<Connection[]>(`/orgs/${orgId}/connections`, { token }),

    create: (token: string, orgId: string, type: string, name: string) =>
      request<Connection>(`/orgs/${orgId}/connections`, {
        method: 'POST',
        body: { type, name },
        token,
      }),

    get: (token: string, orgId: string, connectionId: string) =>
      request<Connection>(`/orgs/${orgId}/connections/${connectionId}`, { token }),
  },

  topology: {
    get: (token: string, orgId: string, connectionId: string) =>
      request<TopologyData>(`/orgs/${orgId}/connections/${connectionId}/topology`, { token }),
  },

  flows: {
    list: (token: string, orgId: string, connectionId: string, limit = 500) =>
      request<{ flows: FlowEvent[] }>(`/orgs/${orgId}/connections/${connectionId}/flows?limit=${limit}`, { token }),
  },

  vulns: {
    list: (token: string, orgId: string, connectionId: string) =>
      request<VulnScan[]>(`/orgs/${orgId}/connections/${connectionId}/vulns`, { token }),
    get: (token: string, orgId: string, connectionId: string, scanId: string) =>
      request<Vulnerability[]>(`/orgs/${orgId}/connections/${connectionId}/vulns/${scanId}`, { token }),
    dashboard: (token: string, orgId: string, connectionId: string) =>
      request<{ total_scans: number; critical: number; high: number; medium: number; low: number; affected_images: Record<string, number> }>(
        `/orgs/${orgId}/connections/${connectionId}/vulns/dashboard`, { token }
      ),
  },

  alerts: {
    list: (token: string, orgId: string, connectionId: string) =>
      request<Alert[]>(`/orgs/${orgId}/connections/${connectionId}/alerts`, { token }),
    firing: (token: string, orgId: string, connectionId: string) =>
      request<Alert[]>(`/orgs/${orgId}/connections/${connectionId}/alerts/firing`, { token }),
    rules: (token: string, orgId: string, connectionId: string) =>
      request<AlertRule[]>(`/orgs/${orgId}/connections/${connectionId}/alerts/rules`, { token }),
    channels: (token: string, orgId: string, connectionId: string) =>
      request<any[]>(`/orgs/${orgId}/connections/${connectionId}/alerts/channels`, { token }),
    silence: (token: string, orgId: string, connectionId: string, alertId: string) =>
      request<{ status: string }>(`/orgs/${orgId}/connections/${connectionId}/alerts/silence`, {
        method: 'POST',
        body: { alert_id: alertId },
        token,
      }),
  },

  metrics: {
    instant: (token: string, orgId: string, connectionId: string, runtimeId: string) =>
      request<Record<string, [number, string]>>(
        `/orgs/${orgId}/connections/${connectionId}/metrics/instant?runtime_id=${runtimeId}`, { token }
      ),
  },

  containers: {
    stats: (token: string, orgId: string, connectionId: string, containerId: string) =>
      request<{
        container_id: string;
        cpu_percent: number;
        memory_usage_mb: number;
        memory_limit_mb: number;
        memory_percent: number;
        network_rx_bytes: number;
        network_tx_bytes: number;
        disk_read_bytes: number;
        disk_write_bytes: number;
        pids: number;
        timestamp: string;
      }>(`/orgs/${orgId}/connections/${connectionId}/containers/${containerId}/stats`, { token }),
    logs: (token: string, orgId: string, connectionId: string, containerId: string, limit = 50) =>
      request<{ container_id: string; container_name: string; logs: Array<{ timestamp: string; level: string; message: string }>; total: number }>(
        `/orgs/${orgId}/connections/${connectionId}/containers/${containerId}/logs?limit=${limit}`, { token }
      ),
  },

  health: {
    services: () => request<HealthResponse>('/healthz/services'),
  },
};
