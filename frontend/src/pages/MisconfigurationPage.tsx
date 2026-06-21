import { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useAppStore } from '../lib/store';

interface Finding {
  id: string;
  title: string;
  description: string;
  severity: string;
  resource: string;
  category: string;
  remediation: string;
}

interface MisconfigSummary {
  critical: number;
  high: number;
  medium: number;
  low: number;
  info: number;
  total: number;
  byCategory: Record<string, number>;
  findings: Finding[];
}

export default function MisconfigurationPage() {
  const { token } = useAuth();
  const { selectedOrg, selectedConnection } = useAppStore();
  const [summary, setSummary] = useState<MisconfigSummary | null>(null);
  const [filterSeverity, setFilterSeverity] = useState<string>('all');
  const [filterCategory, setFilterCategory] = useState<string>('all');

  useEffect(() => {
    if (!token || !selectedOrg || !selectedConnection) return;

    const fetchFindings = async () => {
      try {
        const response = await fetch(
          `/api/v1/orgs/${selectedOrg.id}/connections/${selectedConnection.id}/misconfigs`,
          { headers: { Authorization: `Bearer ${token}` } }
        );

        if (response.ok) {
          const data = await response.json();
          setSummary(data);
        }
      } catch (err) {
        console.error('Failed to fetch misconfigurations:', err);
      }
    };

    fetchFindings();
  }, [token, selectedOrg, selectedConnection]);

  const severityColor: Record<string, string> = {
    CRITICAL: '#ff4444',
    HIGH: '#ff8800',
    MEDIUM: '#ffaa00',
    LOW: '#ffdd00',
    INFO: '#8888a0',
  };

  const categoryIcons: Record<string, string> = {
    security: '🔒',
    network: '🌐',
    isolation: '🔲',
    resources: '📊',
    reliability: '🔄',
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
        Select an organization and connection to view misconfigurations
      </div>
    );
  }

  const filteredFindings = summary?.findings.filter((f) => {
    if (filterSeverity !== 'all' && f.severity !== filterSeverity) return false;
    if (filterCategory !== 'all' && f.category !== filterCategory) return false;
    return true;
  }) || [];

  const categories = Object.keys(summary?.byCategory || {});

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <h2 style={{ fontSize: 20, color: 'var(--color-text)' }}>Misconfiguration Detection</h2>
        <div style={{ display: 'flex', gap: 12 }}>
          <select
            value={filterSeverity}
            onChange={(e) => setFilterSeverity(e.target.value)}
            style={{
              padding: '6px 10px',
              background: 'var(--color-bg)',
              border: '1px solid var(--color-border)',
              borderRadius: 4,
              color: 'var(--color-text)',
              fontSize: 13,
            }}
          >
            <option value="all">All Severities</option>
            <option value="CRITICAL">Critical</option>
            <option value="HIGH">High</option>
            <option value="MEDIUM">Medium</option>
            <option value="LOW">Low</option>
            <option value="INFO">Info</option>
          </select>
          <select
            value={filterCategory}
            onChange={(e) => setFilterCategory(e.target.value)}
            style={{
              padding: '6px 10px',
              background: 'var(--color-bg)',
              border: '1px solid var(--color-border)',
              borderRadius: 4,
              color: 'var(--color-text)',
              fontSize: 13,
            }}
          >
            <option value="all">All Categories</option>
            {categories.map((cat) => (
              <option key={cat} value={cat}>{cat}</option>
            ))}
          </select>
        </div>
      </div>

      {summary && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: 16, marginBottom: 24 }}>
          <StatCard title="Total Findings" value={summary.total.toString()} color="var(--color-accent)" />
          <StatCard title="Critical" value={summary.critical.toString()} color={severityColor.CRITICAL} />
          <StatCard title="High" value={summary.high.toString()} color={severityColor.HIGH} />
          <StatCard title="Medium" value={summary.medium.toString()} color={severityColor.MEDIUM} />
          <StatCard title="Low" value={summary.low.toString()} color={severityColor.LOW} />
        </div>
      )}

      <div style={{
        background: 'var(--color-surface)',
        border: '1px solid var(--color-border)',
        borderRadius: 8,
        overflow: 'hidden',
      }}>
        <div style={{ padding: 16, borderBottom: '1px solid var(--color-border)' }}>
          <h3 style={{ fontSize: 14, color: 'var(--color-text)' }}>
            Findings ({filteredFindings.length})
          </h3>
        </div>
        <div style={{ maxHeight: 500, overflow: 'auto' }}>
          {filteredFindings.length === 0 ? (
            <div style={{ padding: 40, textAlign: 'center', color: 'var(--color-text-muted)' }}>
              No findings match the selected filters
            </div>
          ) : (
            filteredFindings.map((finding) => (
              <div
                key={finding.id}
                style={{
                  padding: '12px 16px',
                  borderBottom: '1px solid var(--color-border)',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                  <div style={{ flex: 1 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                      <span style={{
                        padding: '2px 6px',
                        borderRadius: 3,
                        fontSize: 10,
                        color: '#fff',
                        background: severityColor[finding.severity] || 'var(--color-text-muted)',
                      }}>
                        {finding.severity}
                      </span>
                      <span style={{ fontSize: 13, color: 'var(--color-text)' }}>
                        {categoryIcons[finding.category] || '📋'} {finding.title}
                      </span>
                    </div>
                    <div style={{ fontSize: 12, color: 'var(--color-text-muted)', marginBottom: 4 }}>
                      {finding.description}
                    </div>
                    <div style={{ fontSize: 11, color: 'var(--color-text-muted)' }}>
                      Resource: {finding.resource}
                    </div>
                    {finding.remediation && (
                      <div style={{
                        marginTop: 8,
                        padding: '6px 10px',
                        background: '#00ff8811',
                        border: '1px solid #00ff8833',
                        borderRadius: 4,
                        fontSize: 12,
                        color: 'var(--color-success)',
                      }}>
                        Fix: {finding.remediation}
                      </div>
                    )}
                  </div>
                  <div style={{ fontSize: 11, color: 'var(--color-text-muted)', marginLeft: 16 }}>
                    {finding.id}
                  </div>
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
