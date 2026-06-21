import { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useAppStore } from '../lib/store';
import Header from '../components/layout/Header';

interface Alert {
  id: string;
  rule_id: string;
  rule_name: string;
  severity: string;
  status: string;
  labels: Record<string, string>;
  annotations: Record<string, string>;
  starts_at: string;
  ends_at?: string;
  fired_count: number;
}

export default function AlertsPage() {
  const { token } = useAuth();
  const { selectedOrg, selectedConnection } = useAppStore();
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [filterStatus, setFilterStatus] = useState<string>('all');

  useEffect(() => {
    if (!token || !selectedOrg || !selectedConnection) return;

    const fetchAlerts = async () => {
      try {
        const response = await fetch(
          `/api/v1/orgs/${selectedOrg.id}/connections/${selectedConnection.id}/alerts`,
          { headers: { Authorization: `Bearer ${token}` } }
        );

        if (response.ok) {
          const data = await response.json();
          setAlerts(Array.isArray(data) ? data : []);
        }
      } catch (err) {
        console.error('Failed to fetch alerts:', err);
      }
    };

    fetchAlerts();
    const interval = setInterval(fetchAlerts, 30000);
    return () => clearInterval(interval);
  }, [token, selectedOrg, selectedConnection]);

  const severityColor: Record<string, string> = {
    critical: '#ff4444',
    high: '#ff8800',
    medium: '#ffaa00',
    low: '#ffdd00',
  };

  const statusColor: Record<string, string> = {
    firing: '#ff4444',
    resolved: '#00ff88',
    silenced: '#8888a0',
  };

  const filteredAlerts = alerts.filter((alert) => {
    if (filterStatus !== 'all' && alert.status !== filterStatus) return false;
    return true;
  });

  const firingCount = alerts.filter((a) => a.status === 'firing').length;
  const resolvedCount = alerts.filter((a) => a.status === 'resolved').length;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
      <Header />
      <div style={{ flex: 1, padding: 24, background: '#0a0a0f', overflow: 'auto' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
          <h2 style={{ fontSize: 20, color: '#e0e0e8' }}>Alerts</h2>
          <div style={{ display: 'flex', gap: 12 }}>
            <span style={{ fontSize: 14, color: '#ff4444' }}>
              🔥 {firingCount} firing
            </span>
            <span style={{ fontSize: 14, color: '#00ff88' }}>
              ✓ {resolvedCount} resolved
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
              <div style={{ fontSize: 48, marginBottom: 16 }}>🔔</div>
              <div style={{ fontSize: 16 }}>Select an organization and connection to view alerts</div>
            </div>
          </div>
        ) : (
          <>
            <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
              <select
                value={filterStatus}
                onChange={(e) => setFilterStatus(e.target.value)}
                style={{
                  padding: '6px 10px',
                  background: '#0a0a0f',
                  border: '1px solid #1e1e2e',
                  borderRadius: 4,
                  color: '#e0e0e8',
                  fontSize: 13,
                }}
              >
                <option value="all">All Status</option>
                <option value="firing">Firing</option>
                <option value="resolved">Resolved</option>
                <option value="silenced">Silenced</option>
              </select>
            </div>

            <div style={{
              background: '#12121a',
              border: '1px solid #1e1e2e',
              borderRadius: 8,
              overflow: 'hidden',
            }}>
              {filteredAlerts.length === 0 ? (
                <div style={{ padding: 40, textAlign: 'center', color: '#8888a0' }}>
                  No alerts
                </div>
              ) : (
                filteredAlerts.map((alert) => (
                  <div
                    key={alert.id}
                    style={{
                      padding: '12px 16px',
                      borderBottom: '1px solid #1e1e2e',
                    }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                      <div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                          <span style={{
                            padding: '2px 6px',
                            borderRadius: 3,
                            fontSize: 10,
                            color: '#fff',
                            background: severityColor[alert.severity] || '#8888a0',
                          }}>
                            {alert.severity}
                          </span>
                          <span style={{
                            padding: '2px 6px',
                            borderRadius: 3,
                            fontSize: 10,
                            color: '#fff',
                            background: statusColor[alert.status] || '#8888a0',
                          }}>
                            {alert.status}
                          </span>
                          <span style={{ fontSize: 14, color: '#e0e0e8' }}>
                            {alert.rule_name}
                          </span>
                        </div>
                        <div style={{ fontSize: 12, color: '#8888a0' }}>
                          {alert.annotations?.description || 'No description'}
                        </div>
                        <div style={{ fontSize: 11, color: '#8888a0', marginTop: 4 }}>
                          Started: {new Date(alert.starts_at).toLocaleString()}
                          {alert.ends_at && ` · Resolved: ${new Date(alert.ends_at).toLocaleString()}`}
                          · Fired {alert.fired_count} times
                        </div>
                      </div>
                    </div>
                  </div>
                ))
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
