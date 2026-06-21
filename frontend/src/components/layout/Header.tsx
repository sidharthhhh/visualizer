import { useAuth } from '../../contexts/AuthContext';
import { useAppStore } from '../../lib/store';

export default function Header() {
  const { user, logout } = useAuth();
  const {
    orgs,
    selectedOrg,
    connections,
    selectedConnection,
    setSelectedOrg,
    setSelectedConnection,
  } = useAppStore();

  return (
    <header style={{
      height: 56,
      background: '#12121a',
      borderBottom: '1px solid #1e1e2e',
      display: 'flex',
      alignItems: 'center',
      padding: '0 16px',
      gap: 16,
    }}>
      <h1 style={{ fontSize: 18, color: '#00d4ff', marginRight: 16 }}>ContainerScope</h1>

      <select
        value={selectedOrg?.id || ''}
        onChange={(e) => {
          const org = orgs.find((o) => o.id === e.target.value) || null;
          setSelectedOrg(org);
          setSelectedConnection(null);
        }}
        style={{
          padding: '6px 10px',
          background: '#0a0a0f',
          border: '1px solid #1e1e2e',
          borderRadius: 4,
          color: '#e0e0e8',
          fontSize: 13,
        }}
      >
        <option value="">Select Organization</option>
        {orgs.map((org) => (
          <option key={org.id} value={org.id}>{org.name}</option>
        ))}
      </select>

      <select
        value={selectedConnection?.id || ''}
        onChange={(e) => {
          const conn = connections.find((c) => c.id === e.target.value) || null;
          setSelectedConnection(conn);
        }}
        style={{
          padding: '6px 10px',
          background: '#0a0a0f',
          border: '1px solid #1e1e2e',
          borderRadius: 4,
          color: '#e0e0e8',
          fontSize: 13,
        }}
      >
        <option value="">Select Connection</option>
        {connections.map((conn) => (
          <option key={conn.id} value={conn.id}>
            {conn.name} ({conn.status})
          </option>
        ))}
      </select>

      <div style={{ flex: 1 }} />

      <span style={{ fontSize: 13, color: '#8888a0' }}>{user?.name}</span>
      <button
        onClick={logout}
        style={{
          padding: '6px 12px',
          background: '#0a0a0f',
          border: '1px solid #1e1e2e',
          borderRadius: 4,
          color: '#8888a0',
          fontSize: 13,
          cursor: 'pointer',
        }}
      >
        Logout
      </button>
    </header>
  );
}
