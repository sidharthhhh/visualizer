import { useState } from 'react';
import { useAppStore } from '../../lib/store';
import { useContainerMetrics } from '../../hooks/useMetrics';
import MetricChart from '../charts/MetricChart';
import Sparkline from '../charts/Sparkline';

type MetricTab = 'overview' | 'cpu' | 'memory' | 'network' | 'disk';

export default function SidePanel() {
  const selectedContainer = useAppStore((s) => s.selectedContainer);
  const setSelectedContainer = useAppStore((s) => s.setSelectedContainer);
  const [activeTab, setActiveTab] = useState<MetricTab>('overview');
  const { metrics } = useContainerMetrics(selectedContainer?.runtime_id || null);

  if (!selectedContainer) return null;

  const stateColor: Record<string, string> = {
    running: 'var(--color-accent)',
    stopped: 'var(--color-text-muted)',
    paused: 'var(--color-warning)',
    restarting: 'var(--color-warning)',
    dead: 'var(--color-danger)',
  };

  const cpuData = metrics?.cpu || [];
  const memData = metrics?.mem || [];
  const netRxData = metrics?.net_rx || [];
  const netTxData = metrics?.net_tx || [];
  const diskRData = metrics?.disk_r || [];
  const diskWData = metrics?.disk_w || [];

  const currentCpu = cpuData.length > 0 ? cpuData[cpuData.length - 1].value : 0;
  const currentMem = memData.length > 0 ? memData[memData.length - 1].value : 0;

  const tabs: { id: MetricTab; label: string }[] = [
    { id: 'overview', label: 'Overview' },
    { id: 'cpu', label: 'CPU' },
    { id: 'memory', label: 'Memory' },
    { id: 'network', label: 'Network' },
    { id: 'disk', label: 'Disk' },
  ];

  return (
    <div style={{
      width: 360,
      background: 'var(--color-surface)',
      borderLeft: '1px solid var(--color-border)',
      overflow: 'auto',
      display: 'flex',
      flexDirection: 'column',
    }}>
      <div style={{
        padding: 16,
        borderBottom: '1px solid var(--color-border)',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
      }}>
        <h3 style={{ fontSize: 16, color: 'var(--color-text)' }}>Container Details</h3>
        <button
          onClick={() => setSelectedContainer(null)}
          style={{ color: 'var(--color-text-muted)', fontSize: 18 }}
        >
          ×
        </button>
      </div>

      <div style={{ padding: 16, borderBottom: '1px solid var(--color-border)' }}>
        <div style={{ marginBottom: 12 }}>
          <div style={{ fontSize: 11, color: 'var(--color-text-muted)', marginBottom: 4 }}>NAME</div>
          <div style={{ fontSize: 14, color: 'var(--color-text)' }}>{selectedContainer.name}</div>
        </div>

        <div style={{ marginBottom: 12 }}>
          <div style={{ fontSize: 11, color: 'var(--color-text-muted)', marginBottom: 4 }}>STATE</div>
          <div style={{
            display: 'inline-block',
            padding: '2px 8px',
            borderRadius: 4,
            fontSize: 12,
            color: stateColor[selectedContainer.state] || 'var(--color-text)',
            background: `${stateColor[selectedContainer.state] || 'var(--color-text)'}22`,
          }}>
            {selectedContainer.state}
          </div>
        </div>

        <div style={{ marginBottom: 12 }}>
          <div style={{ fontSize: 11, color: 'var(--color-text-muted)', marginBottom: 4 }}>IMAGE</div>
          <div style={{ fontSize: 13, color: 'var(--color-text)', wordBreak: 'break-all' }}>
            {selectedContainer.image}
          </div>
        </div>

        <div style={{ display: 'flex', gap: 16 }}>
          <div style={{ flex: 1 }}>
            <div style={{ fontSize: 11, color: 'var(--color-text-muted)', marginBottom: 4 }}>CPU</div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ fontSize: 18, color: 'var(--color-accent)', fontWeight: 600 }}>
                {currentCpu.toFixed(1)}%
              </span>
              <Sparkline data={cpuData.map((s) => s.value)} color="#00d4ff" width={60} height={20} />
            </div>
          </div>
          <div style={{ flex: 1 }}>
            <div style={{ fontSize: 11, color: 'var(--color-text-muted)', marginBottom: 4 }}>MEMORY</div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ fontSize: 18, color: '#a855f7', fontWeight: 600 }}>
                {formatBytes(currentMem)}
              </span>
              <Sparkline data={memData.map((s) => s.value)} color="#a855f7" width={60} height={20} />
            </div>
          </div>
        </div>
      </div>

      <div style={{ display: 'flex', borderBottom: '1px solid var(--color-border)' }}>
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            style={{
              flex: 1,
              padding: '8px 4px',
              fontSize: 11,
              color: activeTab === tab.id ? 'var(--color-accent)' : 'var(--color-text-muted)',
              borderBottom: activeTab === tab.id ? '2px solid var(--color-accent)' : '2px solid transparent',
            }}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div style={{ padding: 16, flex: 1, overflow: 'auto' }}>
        {activeTab === 'overview' && (
          <>
            <MetricChart data={cpuData} title="CPU Usage" color="#00d4ff" unit="%" height={120} />
            <MetricChart data={memData} title="Memory Usage" color="#a855f7" unit="bytes" height={120} />
          </>
        )}

        {activeTab === 'cpu' && (
          <MetricChart data={cpuData} title="CPU Usage" color="#00d4ff" unit="%" height={250} />
        )}

        {activeTab === 'memory' && (
          <MetricChart data={memData} title="Memory Usage" color="#a855f7" unit="bytes" height={250} />
        )}

        {activeTab === 'network' && (
          <>
            <MetricChart data={netRxData} title="Network RX" color="#00ff88" unit="bytes" height={150} />
            <MetricChart data={netTxData} title="Network TX" color="#ff6b6b" unit="bytes" height={150} />
          </>
        )}

        {activeTab === 'disk' && (
          <>
            <MetricChart data={diskRData} title="Disk Read" color="#fbbf24" unit="bytes" height={150} />
            <MetricChart data={diskWData} title="Disk Write" color="#f97316" unit="bytes" height={150} />
          </>
        )}
      </div>

      {selectedContainer.ports && selectedContainer.ports.length > 0 && (
        <div style={{ padding: 16, borderTop: '1px solid var(--color-border)' }}>
          <div style={{ fontSize: 11, color: 'var(--color-text-muted)', marginBottom: 4 }}>PORTS</div>
          {selectedContainer.ports.map((port, i) => (
            <div key={i} style={{ fontSize: 12, color: 'var(--color-text)', marginBottom: 2 }}>
              {port.host_port} → {port.container_port}/{port.protocol}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (bytes >= 1e9) return `${(bytes / 1e9).toFixed(1)} GB`;
  if (bytes >= 1e6) return `${(bytes / 1e6).toFixed(1)} MB`;
  if (bytes >= 1e3) return `${(bytes / 1e3).toFixed(1)} KB`;
  return `${bytes.toFixed(0)} B`;
}
