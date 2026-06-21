import { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useAppStore } from '../lib/store';

interface K8sFlow {
  src_pod: string;
  src_namespace: string;
  dst_pod: string;
  dst_namespace: string;
  dst_ip: string;
  dst_port: number;
  protocol: string;
  allowed: boolean;
  policy_name?: string;
}

interface NetworkPolicy {
  name: string;
  namespace: string;
  pod_selector: Record<string, string>;
  ingress: Array<{ ports: Array<{ port: number; protocol: string }>; from: Array<{ pod_selector?: Record<string, string>; ip_block?: string }> }>;
  egress: Array<{ ports: Array<{ port: number; protocol: string }>; to: Array<{ pod_selector?: Record<string, string>; ip_block?: string }> }>;
}

export default function K8sFlowExplorerPage() {
  const { token } = useAuth();
  const { selectedOrg, selectedConnection } = useAppStore();
  const [flows, setFlows] = useState<K8sFlow[]>([]);
  const [policies, setPolicies] = useState<NetworkPolicy[]>([]);
  const [filterNs, setFilterNs] = useState('all');
  const [filterAllowed, setFilterAllowed] = useState<'all' | 'allowed' | 'denied'>('all');
  const [namespaces, setNamespaces] = useState<string[]>([]);

  useEffect(() => {
    if (!token || !selectedOrg || !selectedConnection) return;

    const fetchFlows = async () => {
      try {
        const response = await fetch(
          `/api/v1/orgs/${selectedOrg.id}/connections/${selectedConnection.id}/topology`,
          { headers: { Authorization: `Bearer ${token}` } }
        );

        if (response.ok) {
          const data = await response.json();
          const k8sFlows = data.k8s?.flows || [];
          const k8sPolicies = data.k8s?.network_policies || [];
          const k8sNamespaces = data.k8s?.namespaces?.map((n: { name: string }) => n.name) || [];

          setFlows(k8sFlows);
          setPolicies(k8sPolicies);
          setNamespaces(k8sNamespaces);
        }
      } catch (err) {
        console.error('Failed to fetch K8s flows:', err);
      }
    };

    fetchFlows();
  }, [token, selectedOrg, selectedConnection]);

  const filteredFlows = flows.filter((flow) => {
    if (filterNs !== 'all' && flow.src_namespace !== filterNs && flow.dst_namespace !== filterNs) {
      return false;
    }
    if (filterAllowed === 'allowed' && !flow.allowed) return false;
    if (filterAllowed === 'denied' && flow.allowed) return false;
    return true;
  });

  if (!selectedOrg || !selectedConnection) {
    return (
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100%',
        color: 'var(--color-text-muted)',
      }}>
        Select an organization and connection to view K8s flows
      </div>
    );
  }

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <h2 style={{ fontSize: 20, color: 'var(--color-text)' }}>K8s Flow Explorer</h2>
        <div style={{ display: 'flex', gap: 12 }}>
          <select
            value={filterNs}
            onChange={(e) => setFilterNs(e.target.value)}
            style={{
              padding: '6px 10px',
              background: 'var(--color-bg)',
              border: '1px solid var(--color-border)',
              borderRadius: 4,
              color: 'var(--color-text)',
              fontSize: 13,
            }}
          >
            <option value="all">All Namespaces</option>
            {namespaces.map((ns) => (
              <option key={ns} value={ns}>{ns}</option>
            ))}
          </select>
          <select
            value={filterAllowed}
            onChange={(e) => setFilterAllowed(e.target.value as 'all' | 'allowed' | 'denied')}
            style={{
              padding: '6px 10px',
              background: 'var(--color-bg)',
              border: '1px solid var(--color-border)',
              borderRadius: 4,
              color: 'var(--color-text)',
              fontSize: 13,
            }}
          >
            <option value="all">All Flows</option>
            <option value="allowed">Allowed</option>
            <option value="denied">Denied</option>
          </select>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 24 }}>
        <div style={{
          background: 'var(--color-surface)',
          border: '1px solid var(--color-border)',
          borderRadius: 8,
          overflow: 'hidden',
        }}>
          <div style={{ padding: 16, borderBottom: '1px solid var(--color-border)' }}>
            <h3 style={{ fontSize: 14, color: 'var(--color-text)' }}>Flows ({filteredFlows.length})</h3>
          </div>
          <div style={{ maxHeight: 400, overflow: 'auto' }}>
            {filteredFlows.length === 0 ? (
              <div style={{ padding: 40, textAlign: 'center', color: 'var(--color-text-muted)' }}>
                No flows found
              </div>
            ) : (
              filteredFlows.map((flow, index) => (
                <div
                  key={index}
                  style={{
                    padding: '10px 16px',
                    borderBottom: '1px solid var(--color-border)',
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                  }}
                >
                  <div>
                    <div style={{ fontSize: 12, color: 'var(--color-text)' }}>
                      <span style={{ color: 'var(--color-accent)' }}>{flow.src_pod}</span>
                      <span style={{ color: 'var(--color-text-muted)' }}> → </span>
                      <span style={{ color: '#a855f7' }}>{flow.dst_pod}</span>
                    </div>
                    <div style={{ fontSize: 11, color: 'var(--color-text-muted)', marginTop: 2 }}>
                      {flow.dst_ip}:{flow.dst_port} · {flow.protocol}
                    </div>
                  </div>
                  <div style={{
                    padding: '2px 8px',
                    borderRadius: 4,
                    fontSize: 11,
                    color: flow.allowed ? 'var(--color-success)' : 'var(--color-danger)',
                    background: flow.allowed ? '#00ff8822' : '#ff444422',
                  }}>
                    {flow.allowed ? 'Allowed' : 'Denied'}
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        <div style={{
          background: 'var(--color-surface)',
          border: '1px solid var(--color-border)',
          borderRadius: 8,
          overflow: 'hidden',
        }}>
          <div style={{ padding: 16, borderBottom: '1px solid var(--color-border)' }}>
            <h3 style={{ fontSize: 14, color: 'var(--color-text)' }}>Network Policies ({policies.length})</h3>
          </div>
          <div style={{ maxHeight: 400, overflow: 'auto' }}>
            {policies.length === 0 ? (
              <div style={{ padding: 40, textAlign: 'center', color: 'var(--color-text-muted)' }}>
                No network policies
              </div>
            ) : (
              policies.map((policy, index) => (
                <div
                  key={index}
                  style={{
                    padding: '10px 16px',
                    borderBottom: '1px solid var(--color-border)',
                  }}
                >
                  <div style={{ fontSize: 13, color: 'var(--color-text)' }}>{policy.name}</div>
                  <div style={{ fontSize: 11, color: 'var(--color-text-muted)' }}>{policy.namespace}</div>
                  <div style={{ fontSize: 11, color: 'var(--color-text-muted)', marginTop: 4 }}>
                    Ingress: {policy.ingress.length} rules · Egress: {policy.egress.length} rules
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
