import { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useAppStore } from '../lib/store';
import Header from '../components/layout/Header';

interface FlowEvent {
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

export default function FlowExplorerPage() {
  const { token } = useAuth();
  const { selectedOrg, selectedConnection } = useAppStore();
  const [flows, setFlows] = useState<FlowEvent[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [filter, setFilter] = useState('');

  useEffect(() => {
    if (!token || !selectedOrg || !selectedConnection) return;

    const fetchFlows = async () => {
      setIsLoading(true);
      try {
        const response = await fetch(
          `/api/v1/orgs/${selectedOrg.id}/connections/${selectedConnection.id}/flows?limit=500`,
          { headers: { Authorization: `Bearer ${token}` } }
        );

        if (response.ok) {
          const data = await response.json();
          setFlows(data.flows || []);
        }
      } catch (err) {
        console.error('Failed to fetch flows:', err);
      } finally {
        setIsLoading(false);
      }
    };

    fetchFlows();
    const interval = setInterval(fetchFlows, 10000);
    return () => clearInterval(interval);
  }, [token, selectedOrg, selectedConnection]);

  const filteredFlows = flows.filter((flow) => {
    if (!filter) return true;
    const search = filter.toLowerCase();
    return (
      flow.src_ip.toLowerCase().includes(search) ||
      flow.dst_ip.toLowerCase().includes(search) ||
      flow.protocol.toLowerCase().includes(search) ||
      flow.dst_port.toString().includes(search)
    );
  });

  const formatBytes = (bytes: number): string => {
    if (bytes >= 1e9) return `${(bytes / 1e9).toFixed(1)} GB`;
    if (bytes >= 1e6) return `${(bytes / 1e6).toFixed(1)} MB`;
    if (bytes >= 1e3) return `${(bytes / 1e3).toFixed(1)} KB`;
    return `${bytes} B`;
  };

  const formatTime = (timestamp: string): string => {
    return new Date(timestamp).toLocaleTimeString();
  };

  const protocolColor: Record<string, string> = {
    tcp: '#00d4ff',
    udp: '#a855f7',
    http: '#00ff88',
    https: '#00ff88',
    dns: '#fbbf24',
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
      <Header />
      <div style={{ flex: 1, padding: 24, background: '#0a0a0f', overflow: 'auto' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
          <h2 style={{ fontSize: 20, color: '#e0e0e8' }}>Flow Explorer</h2>
          <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
            <input
              type="text"
              placeholder="Filter by IP, port, protocol..."
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              style={{
                width: 250,
                padding: '6px 12px',
                background: '#0a0a0f',
                border: '1px solid #1e1e2e',
                borderRadius: 4,
                color: '#e0e0e8',
                fontSize: 13,
              }}
            />
            <span style={{ fontSize: 12, color: '#8888a0' }}>
              {filteredFlows.length} flows
            </span>
          </div>
        </div>

        {!selectedOrg || !selectedConnection ? (
          <div style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            height: '50vh',
            color: '#8888a0',
          }}>
            <div style={{ textAlign: 'center' }}>
              <div style={{ fontSize: 48, marginBottom: 16 }}>🌊</div>
              <div style={{ fontSize: 16 }}>Select an organization and connection to view flows</div>
            </div>
          </div>
        ) : (
          <div style={{
            background: '#12121a',
            border: '1px solid #1e1e2e',
            borderRadius: 8,
            overflow: 'hidden',
          }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid #1e1e2e' }}>
                  <th style={{ padding: '10px 12px', textAlign: 'left', fontSize: 11, color: '#8888a0', fontWeight: 500 }}>TIME</th>
                  <th style={{ padding: '10px 12px', textAlign: 'left', fontSize: 11, color: '#8888a0', fontWeight: 500 }}>SOURCE</th>
                  <th style={{ padding: '10px 12px', textAlign: 'left', fontSize: 11, color: '#8888a0', fontWeight: 500 }}>DESTINATION</th>
                  <th style={{ padding: '10px 12px', textAlign: 'left', fontSize: 11, color: '#8888a0', fontWeight: 500 }}>PROTO</th>
                  <th style={{ padding: '10px 12px', textAlign: 'right', fontSize: 11, color: '#8888a0', fontWeight: 500 }}>BYTES</th>
                  <th style={{ padding: '10px 12px', textAlign: 'right', fontSize: 11, color: '#8888a0', fontWeight: 500 }}>PACKETS</th>
                  <th style={{ padding: '10px 12px', textAlign: 'right', fontSize: 11, color: '#8888a0', fontWeight: 500 }}>LATENCY</th>
                </tr>
              </thead>
              <tbody>
                {isLoading && flows.length === 0 ? (
                  <tr>
                    <td colSpan={7} style={{ padding: 40, textAlign: 'center', color: '#8888a0' }}>
                      Loading flows...
                    </td>
                  </tr>
                ) : filteredFlows.length === 0 ? (
                  <tr>
                    <td colSpan={7} style={{ padding: 40, textAlign: 'center', color: '#8888a0' }}>
                      No flows found
                    </td>
                  </tr>
                ) : (
                  filteredFlows.map((flow, index) => (
                    <tr
                      key={index}
                      style={{
                        borderBottom: '1px solid #1e1e2e',
                        cursor: 'pointer',
                      }}
                      onMouseEnter={(e) => {
                        e.currentTarget.style.background = '#0a0a0f';
                      }}
                      onMouseLeave={(e) => {
                        e.currentTarget.style.background = 'transparent';
                      }}
                    >
                      <td style={{ padding: '8px 12px', fontSize: 12, color: '#8888a0' }}>
                        {formatTime(flow.timestamp)}
                      </td>
                      <td style={{ padding: '8px 12px', fontSize: 12, color: '#e0e0e8', fontFamily: 'monospace' }}>
                        {flow.src_ip}:{flow.src_port}
                      </td>
                      <td style={{ padding: '8px 12px', fontSize: 12, color: '#e0e0e8', fontFamily: 'monospace' }}>
                        {flow.dst_ip}:{flow.dst_port}
                      </td>
                      <td style={{ padding: '8px 12px' }}>
                        <span style={{
                          padding: '2px 6px',
                          borderRadius: 3,
                          fontSize: 11,
                          fontWeight: 500,
                          color: protocolColor[flow.protocol.toLowerCase()] || '#e0e0e8',
                          background: `${protocolColor[flow.protocol.toLowerCase()] || '#e0e0e8'}22`,
                        }}>
                          {flow.protocol}
                        </span>
                      </td>
                      <td style={{ padding: '8px 12px', fontSize: 12, color: '#e0e0e8', textAlign: 'right' }}>
                        {formatBytes(flow.bytes)}
                      </td>
                      <td style={{ padding: '8px 12px', fontSize: 12, color: '#e0e0e8', textAlign: 'right' }}>
                        {flow.packets}
                      </td>
                      <td style={{ padding: '8px 12px', fontSize: 12, color: '#e0e0e8', textAlign: 'right' }}>
                        {flow.latency_ms > 0 ? `${flow.latency_ms.toFixed(1)}ms` : '-'}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
