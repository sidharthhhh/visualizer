import { useEffect, useCallback, useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useAppStore } from '../lib/store';
import { api, Edge } from '../lib/api';
import { useWebSocket } from '../hooks/useWebSocket';
import Header from '../components/layout/Header';
import Topology3D from '../components/graph3d/Topology3D';
import SidePanel from '../components/panels/SidePanel';
import ConnectionBadge from '../components/layout/ConnectionBadge';

export default function Topology3DPage() {
  const { token } = useAuth();
  const {
    selectedOrg,
    selectedConnection,
    containers,
    networks,
    setContainers,
    setNetworks,
    setSelectedContainer,
  } = useAppStore();

  const [edges, setEdges] = useState<Edge[]>([]);

  const wsUrl = selectedOrg && selectedConnection
    ? `ws://localhost:8080/ws/orgs/${selectedOrg.id}/connections/${selectedConnection.id}`
    : '';

  const handleWsMessage = useCallback((data: unknown) => {
    const msg = data as { type: string; payload: unknown };
    
    if (msg.type === 'topology_update' || msg.type === 'container_add' || msg.type === 'container_del' || msg.type === 'container_update') {
      if (token && selectedOrg && selectedConnection) {
        api.topology.get(token, selectedOrg.id, selectedConnection.id)
          .then((data) => {
            setContainers(data.containers);
            setNetworks(data.networks);
            setEdges(data.edges);
          })
          .catch(console.error);
      }
    }
  }, [token, selectedOrg, selectedConnection, setContainers, setNetworks]);

  const { isConnected } = useWebSocket({
    url: wsUrl,
    onMessage: handleWsMessage,
    onOpen: () => console.log('WebSocket connected'),
    onClose: () => console.log('WebSocket disconnected'),
  });

  useEffect(() => {
    if (token && selectedOrg && selectedConnection) {
      const fetchTopology = () => {
        api.topology.get(token, selectedOrg.id, selectedConnection.id)
          .then((data) => {
            setContainers(data.containers);
            setNetworks(data.networks);
            setEdges(data.edges);
          })
          .catch(console.error);
      };

      fetchTopology();
    }
  }, [token, selectedOrg, selectedConnection, setContainers, setNetworks]);

  const hasData = selectedOrg && selectedConnection;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
      <Header />
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        <div style={{ flex: 1, position: 'relative' }}>
          {hasData ? (
            <>
              <Topology3D
                containers={containers}
                edges={edges}
                onContainerClick={setSelectedContainer}
              />
              <div style={{
                position: 'absolute',
                bottom: 16,
                left: 16,
                display: 'flex',
                alignItems: 'center',
                gap: 12,
              }}>
                <span style={{ fontSize: 12, color: 'var(--color-text-muted)' }}>
                  {containers.length} containers · {networks.length} networks · {edges.length} edges
                </span>
                <ConnectionBadge isConnected={isConnected} />
              </div>
              <div style={{
                position: 'absolute',
                top: 16,
                left: 16,
                padding: '6px 12px',
                background: 'var(--color-surface)',
                border: '1px solid var(--color-border)',
                borderRadius: 4,
                fontSize: 12,
                color: 'var(--color-text-muted)',
              }}>
                3D View · Drag to rotate · Scroll to zoom
              </div>
            </>
          ) : (
            <div style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              color: 'var(--color-text-muted)',
            }}>
              <div style={{ textAlign: 'center' }}>
                <p style={{ fontSize: 48, marginBottom: 16 }}>⬡</p>
                <p style={{ fontSize: 16 }}>Select an organization and connection</p>
                <p style={{ fontSize: 13, marginTop: 8 }}>
                  Deploy an agent to start visualizing your infrastructure in 3D
                </p>
              </div>
            </div>
          )}
        </div>
        <SidePanel />
      </div>
    </div>
  );
}
