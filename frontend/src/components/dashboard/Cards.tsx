import { motion } from 'framer-motion';
import { 
  TrendingUp, 
  TrendingDown, 
  Minus,
  Clock
} from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { cn } from '@/lib/utils';

interface KPICardProps {
  title: string;
  value: string | number;
  change?: number;
  changeLabel?: string;
  icon: React.ReactNode;
  trend?: 'up' | 'down' | 'neutral';
  color?: string;
  delay?: number;
}

export function KPICard({ title, value, change, changeLabel, icon, trend = 'neutral', color = 'text-primary', delay = 0 }: KPICardProps) {
  const trendIcon = trend === 'up' ? TrendingUp : trend === 'down' ? TrendingDown : Minus;
  const trendColor = trend === 'up' ? 'text-emerald-500' : trend === 'down' ? 'text-red-500' : 'text-muted-foreground';
  const TrendIcon = trendIcon;

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5, delay }}
    >
      <Card className="premium-card overflow-hidden">
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            {title}
          </CardTitle>
          <div className={cn('p-2 rounded-lg bg-muted', color)}>
            {icon}
          </div>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{value}</div>
          {change !== undefined && (
            <div className="flex items-center gap-1 mt-1">
              <TrendIcon className={cn('h-3 w-3', trendColor)} />
              <span className={cn('text-xs font-medium', trendColor)}>
                {change > 0 ? '+' : ''}{change}%
              </span>
              {changeLabel && (
                <span className="text-xs text-muted-foreground ml-1">
                  {changeLabel}
                </span>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </motion.div>
  );
}

interface MetricCardProps {
  title: string;
  value: number;
  max: number;
  unit: string;
  icon: React.ReactNode;
  color: string;
  delay?: number;
}

export function MetricCard({ title, value, max, unit, icon, color, delay = 0 }: MetricCardProps) {
  const percentage = (value / max) * 100;

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.5, delay }}
    >
      <Card className="premium-card">
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">
            {title}
          </CardTitle>
          <div className={cn('p-2 rounded-lg bg-muted', color)}>
            {icon}
          </div>
        </CardHeader>
        <CardContent>
          <div className="flex items-baseline gap-2">
            <span className="text-2xl font-bold">{value.toFixed(1)}</span>
            <span className="text-sm text-muted-foreground">{unit}</span>
          </div>
          <Progress value={percentage} className="mt-3" />
          <p className="text-xs text-muted-foreground mt-2">
            {percentage.toFixed(1)}% of {max}{unit} capacity
          </p>
        </CardContent>
      </Card>
    </motion.div>
  );
}

interface StatusCardProps {
  title: string;
  status: 'healthy' | 'warning' | 'critical';
  message: string;
  icon: React.ReactNode;
  delay?: number;
}

export function StatusCard({ title, status, message, icon, delay = 0 }: StatusCardProps) {
  const statusConfig = {
    healthy: { color: 'bg-emerald-500', badge: 'default', label: 'Healthy' },
    warning: { color: 'bg-yellow-500', badge: 'secondary', label: 'Warning' },
    critical: { color: 'bg-red-500', badge: 'destructive', label: 'Critical' },
  };

  const config = statusConfig[status];

  return (
    <motion.div
      initial={{ opacity: 0, x: -20 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.5, delay }}
    >
      <Card className="premium-card">
        <CardContent className="p-4">
          <div className="flex items-center gap-4">
            <div className={cn('p-3 rounded-lg', config.color, 'bg-opacity-10')}>
              {icon}
            </div>
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <h3 className="font-medium">{title}</h3>
                <Badge variant={config.badge as any} className="text-xs">
                  {config.label}
                </Badge>
              </div>
              <p className="text-sm text-muted-foreground mt-1">{message}</p>
            </div>
            <div className={cn('status-dot', status)} />
          </div>
        </CardContent>
      </Card>
    </motion.div>
  );
}

interface ActivityItemProps {
  title: string;
  description: string;
  time: string;
  icon: React.ReactNode;
  delay?: number;
}

export function ActivityItem({ title, description, time, icon, delay = 0 }: ActivityItemProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, delay }}
      className="flex items-start gap-3 p-3 rounded-lg hover:bg-muted/50 transition-colors"
    >
      <div className="p-2 rounded-lg bg-muted">
        {icon}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium">{title}</p>
        <p className="text-xs text-muted-foreground mt-0.5">{description}</p>
      </div>
      <div className="flex items-center gap-1 text-xs text-muted-foreground">
        <Clock className="h-3 w-3" />
        <span>{time}</span>
      </div>
    </motion.div>
  );
}
