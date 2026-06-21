import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

interface MetricChartProps {
  data: Array<{ timestamp: number; value: number }>;
  title: string;
  color?: string;
  unit?: string;
  height?: number;
}

function formatValue(value: number, unit: string): string {
  if (unit === 'bytes') {
    if (value >= 1e9) return `${(value / 1e9).toFixed(1)} GB`;
    if (value >= 1e6) return `${(value / 1e6).toFixed(1)} MB`;
    if (value >= 1e3) return `${(value / 1e3).toFixed(1)} KB`;
    return `${value.toFixed(0)} B`;
  }
  if (unit === '%') return `${value.toFixed(1)}%`;
  return value.toFixed(2);
}

function formatTime(timestamp: number): string {
  const date = new Date(timestamp * 1000);
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

export default function MetricChart({ data, title, color = '#00d4ff', unit = '', height = 200 }: MetricChartProps) {
  return (
    <div style={{ marginBottom: 16 }}>
      <div style={{ fontSize: 12, color: 'var(--color-text-muted)', marginBottom: 8 }}>{title}</div>
      <ResponsiveContainer width="100%" height={height}>
        <LineChart data={data} margin={{ top: 5, right: 5, bottom: 5, left: 5 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#1e1e2e" />
          <XAxis
            dataKey="timestamp"
            tickFormatter={formatTime}
            stroke="#8888a0"
            fontSize={10}
            tickLine={false}
          />
          <YAxis
            stroke="#8888a0"
            fontSize={10}
            tickLine={false}
            tickFormatter={(value) => formatValue(value, unit)}
          />
          <Tooltip
            contentStyle={{
              background: 'var(--color-surface)',
              border: '1px solid var(--color-border)',
              borderRadius: 4,
              fontSize: 12,
            }}
            labelFormatter={(label) => formatTime(label as number)}
            formatter={(value) => [formatValue(Number(value), unit), title]}
          />
          <Line
            type="monotone"
            dataKey="value"
            stroke={color}
            strokeWidth={2}
            dot={false}
            isAnimationActive={false}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
