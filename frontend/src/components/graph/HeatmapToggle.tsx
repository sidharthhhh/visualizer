type HeatmapMode = 'none' | 'cpu' | 'memory';

interface HeatmapToggleProps {
  mode: HeatmapMode;
  onChange: (mode: HeatmapMode) => void;
}

export default function HeatmapToggle({ mode, onChange }: HeatmapToggleProps) {
  const options: { id: HeatmapMode; label: string; color: string }[] = [
    { id: 'none', label: 'Normal', color: 'var(--color-text-muted)' },
    { id: 'cpu', label: 'CPU', color: '#00d4ff' },
    { id: 'memory', label: 'Memory', color: '#a855f7' },
  ];

  return (
    <div style={{
      position: 'absolute',
      bottom: 16,
      right: 16,
      display: 'flex',
      background: 'var(--color-surface)',
      border: '1px solid var(--color-border)',
      borderRadius: 6,
      overflow: 'hidden',
    }}>
      {options.map((opt) => (
        <button
          key={opt.id}
          onClick={() => onChange(opt.id)}
          style={{
            padding: '6px 12px',
            fontSize: 11,
            color: mode === opt.id ? '#fff' : 'var(--color-text-muted)',
            background: mode === opt.id ? opt.color : 'transparent',
            borderRight: '1px solid var(--color-border)',
          }}
        >
          {opt.label}
        </button>
      ))}
    </div>
  );
}
