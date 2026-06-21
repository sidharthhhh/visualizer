interface ConnectionBadgeProps {
  isConnected: boolean;
}

export default function ConnectionBadge({ isConnected }: ConnectionBadgeProps) {
  return (
    <div style={{
      display: 'inline-flex',
      alignItems: 'center',
      gap: 6,
      padding: '3px 10px',
      borderRadius: 12,
      fontSize: 11,
      background: isConnected ? '#00ff8822' : '#ff444422',
      color: isConnected ? 'var(--color-success)' : 'var(--color-danger)',
    }}>
      <div style={{
        width: 6,
        height: 6,
        borderRadius: '50%',
        background: isConnected ? 'var(--color-success)' : 'var(--color-danger)',
      }} />
      {isConnected ? 'Live' : 'Disconnected'}
    </div>
  );
}
