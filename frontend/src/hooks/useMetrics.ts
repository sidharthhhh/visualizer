import { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useAppStore } from '../lib/store';

export interface MetricSample {
  timestamp: number;
  value: number;
}

export interface ContainerMetrics {
  cpu: MetricSample[];
  mem: MetricSample[];
  net_rx: MetricSample[];
  net_tx: MetricSample[];
  disk_r: MetricSample[];
  disk_w: MetricSample[];
}

const API_BASE = '/api/v1';

export function useContainerMetrics(runtimeId: string | null, interval: number = 15000) {
  const { token } = useAuth();
  const { selectedOrg, selectedConnection } = useAppStore();
  const [metrics, setMetrics] = useState<ContainerMetrics | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const fetchMetrics = useCallback(async () => {
    if (!token || !selectedOrg || !selectedConnection || !runtimeId) return;

    try {
      setIsLoading(true);
      const response = await fetch(
        `${API_BASE}/orgs/${selectedOrg.id}/connections/${selectedConnection.id}/metrics/instant?runtime_id=${runtimeId}`,
        { headers: { Authorization: `Bearer ${token}` } }
      );

      if (response.ok) {
        const data = await response.json();
        const now = Date.now() / 1000;

        setMetrics((prev) => {
          const newMetrics: ContainerMetrics = {
            cpu: [...(prev?.cpu || []), { timestamp: now, value: parseFloat(data.cpu?.[1] || '0') }].slice(-20),
            mem: [...(prev?.mem || []), { timestamp: now, value: parseFloat(data.mem?.[1] || '0') }].slice(-20),
            net_rx: [...(prev?.net_rx || []), { timestamp: now, value: parseFloat(data.net_rx?.[1] || '0') }].slice(-20),
            net_tx: [...(prev?.net_tx || []), { timestamp: now, value: parseFloat(data.net_tx?.[1] || '0') }].slice(-20),
            disk_r: [...(prev?.disk_r || []), { timestamp: now, value: parseFloat(data.disk_r?.[1] || '0') }].slice(-20),
            disk_w: [...(prev?.disk_w || []), { timestamp: now, value: parseFloat(data.disk_w?.[1] || '0') }].slice(-20),
          };
          return newMetrics;
        });
      }
    } catch (err) {
      console.error('Failed to fetch metrics:', err);
    } finally {
      setIsLoading(false);
    }
  }, [token, selectedOrg, selectedConnection, runtimeId]);

  useEffect(() => {
    if (runtimeId) {
      fetchMetrics();
      const id = setInterval(fetchMetrics, interval);
      return () => clearInterval(id);
    }
  }, [runtimeId, interval, fetchMetrics]);

  return { metrics, isLoading, refetch: fetchMetrics };
}

export interface ContainerMetricSummary {
  runtime_id: string;
  name: string;
  cpu: number;
  mem: number;
}

export function useTopConsumers(limit: number = 10) {
  const { token } = useAuth();
  const { selectedOrg, selectedConnection, containers } = useAppStore();
  const [topCpu, setTopCpu] = useState<ContainerMetricSummary[]>([]);
  const [topMem, setTopMem] = useState<ContainerMetricSummary[]>([]);

  useEffect(() => {
    if (!token || !selectedOrg || !selectedConnection || containers.length === 0) return;

    const fetchAll = async () => {
      const results: ContainerMetricSummary[] = [];

      for (const ctr of containers.slice(0, 20)) {
        try {
          const response = await fetch(
            `${API_BASE}/orgs/${selectedOrg.id}/connections/${selectedConnection.id}/metrics/instant?runtime_id=${ctr.runtime_id}`,
            { headers: { Authorization: `Bearer ${token}` } }
          );

          if (response.ok) {
            const data = await response.json();
            results.push({
              runtime_id: ctr.runtime_id,
              name: ctr.name,
              cpu: parseFloat(data.cpu?.[1] || '0'),
              mem: parseFloat(data.mem?.[1] || '0'),
            });
          }
        } catch {
          // skip failed
        }
      }

      setTopCpu([...results].sort((a, b) => b.cpu - a.cpu).slice(0, limit));
      setTopMem([...results].sort((a, b) => b.mem - a.mem).slice(0, limit));
    };

    fetchAll();
    const id = setInterval(fetchAll, 30000);
    return () => clearInterval(id);
  }, [token, selectedOrg, selectedConnection, containers, limit]);

  return { topCpu, topMem };
}
