import { useMemo } from 'react';
import { motion } from 'framer-motion';

interface SparklineProps {
  data: number[];
  width?: number;
  height?: number;
  color?: string;
  gradient?: boolean;
  animated?: boolean;
  className?: string;
}

export function Sparkline({ 
  data, 
  width = 120, 
  height = 40, 
  color = 'hsl(var(--primary))',
  gradient = true,
  animated = true,
  className = '' 
}: SparklineProps) {
  const points = useMemo(() => {
    if (!data.length) return '';

    const max = Math.max(...data);
    const min = Math.min(...data);
    const range = max - min || 1;
    const padding = 2;

    return data
      .map((value, index) => {
        const x = (index / (data.length - 1)) * (width - padding * 2) + padding;
        const y = height - ((value - min) / range) * (height - padding * 2) - padding;
        return `${x},${y}`;
      })
      .join(' ');
  }, [data, width, height]);

  const areaPath = useMemo(() => {
    if (!points) return '';
    const firstPoint = points.split(' ')[0];
    const lastPoint = points.split(' ').pop();
    return `M ${firstPoint} L ${points} L ${lastPoint?.split(',')[0]},${height} L ${firstPoint?.split(',')[0]},${height} Z`;
  }, [points, height]);

  const gradientId = useMemo(() => `sparkline-${Math.random().toString(36).substr(2, 9)}`, []);

  if (!data.length) return null;

  return (
    <motion.svg
      width={width}
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      className={className}
      initial={animated ? { opacity: 0 } : undefined}
      animate={animated ? { opacity: 1 } : undefined}
      transition={{ duration: 0.5 }}
    >
      {gradient && (
        <defs>
          <linearGradient id={gradientId} x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" stopColor={color} stopOpacity={0.3} />
            <stop offset="100%" stopColor={color} stopOpacity={0} />
          </linearGradient>
        </defs>
      )}
      
      {gradient && (
        <motion.path
          d={areaPath}
          fill={`url(#${gradientId})`}
          initial={animated ? { opacity: 0 } : undefined}
          animate={animated ? { opacity: 1 } : undefined}
          transition={{ duration: 1, delay: 0.3 }}
        />
      )}
      
      <motion.polyline
        points={points}
        fill="none"
        stroke={color}
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
        initial={animated ? { pathLength: 0 } : undefined}
        animate={animated ? { pathLength: 1 } : undefined}
        transition={{ duration: 1.5, ease: 'easeOut' }}
      />

      {/* End dot */}
      {data.length > 0 && (
        <motion.circle
          cx={(data.length - 1) / (data.length - 1) * (width - 4) + 2}
          cy={height - ((data[data.length - 1] - Math.min(...data)) / (Math.max(...data) - Math.min(...data) || 1)) * (height - 4) - 2}
          r={3}
          fill={color}
          initial={animated ? { scale: 0 } : undefined}
          animate={animated ? { scale: 1 } : undefined}
          transition={{ duration: 0.3, delay: 1.5 }}
        />
      )}
    </motion.svg>
  );
}
