import { useEffect, useRef, useState } from 'react';
import { motion, useInView } from 'framer-motion';

interface AnimatedCounterProps {
  value: number;
  duration?: number;
  prefix?: string;
  suffix?: string;
  className?: string;
}

export function AnimatedCounter({ 
  value, 
  duration = 2, 
  prefix = '', 
  suffix = '', 
  className = '' 
}: AnimatedCounterProps) {
  const [count, setCount] = useState(0);
  const ref = useRef(null);
  const isInView = useInView(ref, { once: true });

  useEffect(() => {
    if (!isInView) return;

    const end = value;
    const timer = setTimeout(() => {
      const startTime = Date.now();

      const interval = setInterval(() => {
        const now = Date.now();
        const progress = Math.min((now - startTime) / (duration * 1000), 1);
        
        const eased = 1 - Math.pow(1 - progress, 3);
        
        setCount(Math.floor(eased * end));

        if (progress >= 1) {
          clearInterval(interval);
          setCount(end);
        }
      }, 16);

      return () => clearInterval(interval);
    }, 100);

    return () => clearTimeout(timer);
  }, [value, duration, isInView]);

  return (
    <motion.span
      ref={ref}
      initial={{ opacity: 0, y: 10 }}
      animate={isInView ? { opacity: 1, y: 0 } : {}}
      transition={{ duration: 0.5 }}
      className={className}
    >
      {prefix}{count.toLocaleString()}{suffix}
    </motion.span>
  );
}

interface AnimatedPercentageProps {
  value: number;
  decimals?: number;
  className?: string;
  showSign?: boolean;
}

export function AnimatedPercentage({ 
  value, 
  decimals = 1, 
  className = '',
  showSign = true 
}: AnimatedPercentageProps) {
  const isPositive = value >= 0;
  
  return (
    <motion.span
      initial={{ opacity: 0, scale: 0.8 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.5, delay: 0.2 }}
      className={`inline-flex items-center gap-1 ${className}`}
    >
      {showSign && (
        <svg
          className={`h-3 w-3 ${isPositive ? 'text-emerald-500' : 'text-rose-500'}`}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={3}
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d={isPositive ? 'M5 15l7-7 7 7' : 'M19 9l-7 7-7-7'}
          />
        </svg>
      )}
      <span className={isPositive ? 'text-emerald-500' : 'text-rose-500'}>
        {isPositive ? '+' : ''}{value.toFixed(decimals)}%
      </span>
    </motion.span>
  );
}
