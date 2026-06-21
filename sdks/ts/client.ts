/**
 * ContainerScope TypeScript SDK
 */

export interface Container {
  id: string;
  connection_id: string;
  runtime_id: string;
  name: string;
  image: string;
  state: string;
  labels?: Record<string, string>;
  ports?: Port[];
  created_at: string;
}

export interface Port {
  host_port: string;
  container_port: string;
  protocol: string;
}

export interface Network {
  id: string;
  connection_id: string;
  name: string;
  driver: string;
  scope: string;
  subnet: string;
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

export interface Topology {
  containers: Container[];
  networks: Network[];
  edges: Edge[];
}

export interface MetricSample {
  timestamp: string;
  value: number;
}

export interface Alert {
  id: string;
  rule_id: string;
  rule_name: string;
  severity: string;
  status: string;
  labels?: Record<string, string>;
  starts_at: string;
  ends_at?: string;
  fired_count: number;
}

export interface Connection {
  id: string;
  org_id: string;
  type: string;
  name: string;
  status: string;
  created_at: string;
}

export interface ClientOptions {
  baseUrl: string;
  apiKey?: string;
}

export class ContainerScopeClient {
  private baseUrl: string;
  private apiKey?: string;

  constructor(options: ClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/$/, '');
    this.apiKey = options.apiKey;
  }

  async getTopology(orgId: string, connectionId: string): Promise<Topology> {
    return this.get(`/api/v1/orgs/${orgId}/connections/${connectionId}/topology`);
  }

  async listContainers(orgId: string, connectionId: string): Promise<Container[]> {
    const topology = await this.getTopology(orgId, connectionId);
    return topology.containers;
  }

  async listConnections(orgId: string): Promise<Connection[]> {
    return this.get(`/api/v1/orgs/${orgId}/connections`);
  }

  async getMetrics(
    orgId: string,
    connectionId: string,
    runtimeId: string,
    metric: string = 'cpu'
  ): Promise<MetricSample[]> {
    const response = await this.get(
      `/api/v1/orgs/${orgId}/connections/${connectionId}/metrics?runtime_id=${runtimeId}&metric=${metric}`
    );
    return response.results || [];
  }

  async listAlerts(orgId: string, connectionId: string): Promise<Alert[]> {
    return this.get(`/api/v1/orgs/${orgId}/connections/${connectionId}/alerts`);
  }

  async listFlows(orgId: string, connectionId: string): Promise<any[]> {
    const response = await this.get(
      `/api/v1/orgs/${orgId}/connections/${connectionId}/flows`
    );
    return response.flows || [];
  }

  async health(): Promise<{ status: string; db: string }> {
    return this.get('/healthz');
  }

  async version(): Promise<{ version: string }> {
    return this.get('/version');
  }

  private async get<T>(path: string): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (this.apiKey) {
      headers['X-API-Key'] = this.apiKey;
    }

    const response = await fetch(`${this.baseUrl}${path}`, {
      method: 'GET',
      headers,
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`API error ${response.status}: ${error}`);
    }

    return response.json();
  }

  private async post<T>(path: string, body?: any): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (this.apiKey) {
      headers['X-API-Key'] = this.apiKey;
    }

    const response = await fetch(`${this.baseUrl}${path}`, {
      method: 'POST',
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`API error ${response.status}: ${error}`);
    }

    return response.json();
  }
}

export default ContainerScopeClient;
