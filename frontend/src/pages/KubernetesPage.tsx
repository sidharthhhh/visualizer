import { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useAppStore } from '../lib/store';

interface K8sNamespace {
  name: string;
  labels: Record<string, string>;
}

interface K8sDeployment {
  name: string;
  namespace: string;
  replicas: number;
  ready: number;
  labels: Record<string, string>;
}

interface K8sPod {
  name: string;
  namespace: string;
  owner_name: string;
  node_name: string;
  phase: string;
  ip: string;
  containers: Array<{ name: string; image: string; ready: boolean }>;
}

interface K8sNode {
  name: string;
  addresses: string[];
  capacity: Record<string, string>;
}

interface K8sService {
  name: string;
  namespace: string;
  type: string;
  cluster_ip: string;
  ports: Array<{ name: string; port: number; target_port: number; protocol: string }>;
}

export default function KubernetesPage() {
  const { token } = useAuth();
  const { selectedOrg, selectedConnection } = useAppStore();
  const [namespaces, setNamespaces] = useState<K8sNamespace[]>([]);
  const [deployments, setDeployments] = useState<K8sDeployment[]>([]);
  const [pods, setPods] = useState<K8sPod[]>([]);
  const [nodes, setNodes] = useState<K8sNode[]>([]);
  const [services, setServices] = useState<K8sService[]>([]);
  const [selectedNs, setSelectedNs] = useState<string>('all');
  const [expandedDeployment, setExpandedDeployment] = useState<string | null>(null);

  useEffect(() => {
    if (!token || !selectedOrg || !selectedConnection) return;

    const fetchK8s = async () => {
      try {
        const response = await fetch(
          `/api/v1/orgs/${selectedOrg.id}/connections/${selectedConnection.id}/topology`,
          { headers: { Authorization: `Bearer ${token}` } }
        );

        if (response.ok) {
          const data = await response.json();
          setNamespaces(data.k8s?.namespaces || []);
          setDeployments(data.k8s?.deployments || []);
          setPods(data.k8s?.pods || []);
          setNodes(data.k8s?.nodes || []);
          setServices(data.k8s?.services || []);
        }
      } catch (err) {
        console.error('Failed to fetch K8s topology:', err);
      }
    };

    fetchK8s();
  }, [token, selectedOrg, selectedConnection]);

  const filteredDeployments = selectedNs === 'all'
    ? deployments
    : deployments.filter((d) => d.namespace === selectedNs);

  const filteredPods = selectedNs === 'all'
    ? pods
    : pods.filter((p) => p.namespace === selectedNs);

  const phaseColor: Record<string, string> = {
    Running: 'var(--color-success)',
    Pending: 'var(--color-warning)',
    Succeeded: 'var(--color-text-muted)',
    Failed: 'var(--color-danger)',
    Unknown: 'var(--color-text-muted)',
  };

  if (!selectedOrg || !selectedConnection) {
    return (
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100%',
        color: 'var(--color-text-muted)',
      }}>
        Select an organization and connection to view Kubernetes topology
      </div>
    );
  }

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <h2 style={{ fontSize: 20, color: 'var(--color-text)' }}>Kubernetes Topology</h2>
        <div style={{ display: 'flex', gap: 12 }}>
          <select
            value={selectedNs}
            onChange={(e) => setSelectedNs(e.target.value)}
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
              <option key={ns.name} value={ns.name}>{ns.name}</option>
            ))}
          </select>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: 16, marginBottom: 24 }}>
        <StatCard title="Namespaces" value={namespaces.length.toString()} color="var(--color-accent)" />
        <StatCard title="Deployments" value={filteredDeployments.length.toString()} color="#a855f7" />
        <StatCard title="Pods" value={filteredPods.length.toString()} color="#00ff88" />
        <StatCard title="Nodes" value={nodes.length.toString()} color="#fbbf24" />
        <StatCard title="Services" value={services.length.toString()} color="#f97316" />
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 24 }}>
        <div style={{
          background: 'var(--color-surface)',
          border: '1px solid var(--color-border)',
          borderRadius: 8,
          padding: 16,
        }}>
          <h3 style={{ fontSize: 14, marginBottom: 16, color: 'var(--color-text)' }}>Deployments</h3>
          {filteredDeployments.length === 0 ? (
            <div style={{ color: 'var(--color-text-muted)', textAlign: 'center', padding: 20 }}>No deployments</div>
          ) : (
            filteredDeployments.map((dep) => (
              <div
                key={`${dep.namespace}/${dep.name}`}
                style={{
                  padding: '10px 12px',
                  borderBottom: '1px solid var(--color-border)',
                  cursor: 'pointer',
                }}
                onClick={() => setExpandedDeployment(
                  expandedDeployment === dep.name ? null : dep.name
                )}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <div>
                    <div style={{ fontSize: 13, color: 'var(--color-text)' }}>{dep.name}</div>
                    <div style={{ fontSize: 11, color: 'var(--color-text-muted)' }}>{dep.namespace}</div>
                  </div>
                  <div style={{
                    padding: '2px 8px',
                    borderRadius: 4,
                    fontSize: 12,
                    color: dep.ready === dep.replicas ? 'var(--color-success)' : 'var(--color-warning)',
                    background: dep.ready === dep.replicas ? '#00ff8822' : '#ffaa0022',
                  }}>
                    {dep.ready}/{dep.replicas}
                  </div>
                </div>
                {expandedDeployment === dep.name && (
                  <div style={{ marginTop: 8, paddingLeft: 12 }}>
                    {filteredPods
                      .filter((p) => p.owner_name === dep.name)
                      .map((pod) => (
                        <div key={pod.name} style={{ padding: '4px 0', fontSize: 12 }}>
                          <span style={{ color: phaseColor[pod.phase] || 'var(--color-text)' }}>
                            ●
                          </span>
                          <span style={{ color: 'var(--color-text)', marginLeft: 8 }}>
                            {pod.name}
                          </span>
                        </div>
                      ))
                    }
                  </div>
                )}
              </div>
            ))
          )}
        </div>

        <div style={{
          background: 'var(--color-surface)',
          border: '1px solid var(--color-border)',
          borderRadius: 8,
          padding: 16,
        }}>
          <h3 style={{ fontSize: 14, marginBottom: 16, color: 'var(--color-text)' }}>Nodes</h3>
          {nodes.length === 0 ? (
            <div style={{ color: 'var(--color-text-muted)', textAlign: 'center', padding: 20 }}>No nodes</div>
          ) : (
            nodes.map((node) => (
              <div key={node.name} style={{ padding: '10px 12px', borderBottom: '1px solid var(--color-border)' }}>
                <div style={{ fontSize: 13, color: 'var(--color-text)' }}>{node.name}</div>
                <div style={{ fontSize: 11, color: 'var(--color-text-muted)', marginTop: 4 }}>
                  CPU: {node.capacity.cpu} · Memory: {node.capacity.memory}
                </div>
                <div style={{ fontSize: 11, color: 'var(--color-text-muted)' }}>
                  {node.addresses.join(', ')}
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

function StatCard({ title, value, color }: { title: string; value: string; color: string }) {
  return (
    <div style={{
      background: 'var(--color-surface)',
      border: '1px solid var(--color-border)',
      borderRadius: 8,
      padding: 16,
    }}>
      <div style={{ fontSize: 12, color: 'var(--color-text-muted)', marginBottom: 8 }}>{title}</div>
      <div style={{ fontSize: 28, fontWeight: 700, color }}>{value}</div>
    </div>
  );
}
