import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  LayoutDashboard, 
  BarChart3, 
  Box, 
  Network, 
  Shield, 
  Bell, 
  Settings, 
  ChevronLeft,
  ChevronRight,
  Search,
  Server,
  Activity,
  Globe,
  Cpu,
  Star,
  Clock,
  HelpCircle,
  LogOut
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { cn } from '@/lib/utils';
import { useAuth } from '@/contexts/AuthContext';

interface SidebarProps {
  collapsed: boolean;
  onToggle: () => void;
  activeItem: string;
  onItemClick: (item: string) => void;
}

const mainNavigation = [
  { 
    id: 'dashboard', 
    label: 'Dashboard', 
    icon: LayoutDashboard,
    description: 'Overview & Analytics'
  },
  { 
    id: 'topology', 
    label: 'Topology', 
    icon: Network,
    description: 'Infrastructure Map'
  },
  { 
    id: 'containers', 
    label: 'Containers', 
    icon: Box, 
    badge: '24',
    description: 'Container Management'
  },
  { 
    id: 'analytics', 
    label: 'Analytics', 
    icon: BarChart3,
    description: 'Performance Metrics'
  },
  { 
    id: 'flows', 
    label: 'Network Flows', 
    icon: Globe,
    description: 'Traffic Analysis'
  },
];

const securityNavigation = [
  { 
    id: 'security', 
    label: 'Security', 
    icon: Shield, 
    badge: '3',
    description: 'Vulnerabilities & Threats'
  },
  { 
    id: 'alerts', 
    label: 'Alerts', 
    icon: Bell, 
    badge: '5',
    description: 'Notifications & Rules'
  },
];

const systemNavigation = [
  { 
    id: 'health', 
    label: 'System Health', 
    icon: Activity,
    description: 'Service Status'
  },
  { 
    id: 'resources', 
    label: 'Resources', 
    icon: Cpu,
    description: 'CPU, Memory, Disk'
  },
  { 
    id: 'settings', 
    label: 'Settings', 
    icon: Settings,
    description: 'Configuration'
  },
];

const favorites = [
  { id: 'dashboard', label: 'Dashboard', icon: Star },
  { id: 'containers', label: 'Containers', icon: Box },
  { id: 'alerts', label: 'Alerts', icon: Bell },
];

const recentPages = [
  { id: 'dashboard', label: 'Dashboard', time: '2m ago' },
  { id: 'containers', label: 'Containers', time: '15m ago' },
  { id: 'security', label: 'Security', time: '1h ago' },
];

export function Sidebar({ collapsed, onToggle, activeItem, onItemClick }: SidebarProps) {
  const { logout } = useAuth();
  const [hoveredItem, setHoveredItem] = useState<string | null>(null);

  const renderNavItem = (item: any, index: number) => {
    const isActive = activeItem === item.id;
    const isHovered = hoveredItem === item.id;

    return (
      <motion.div
        key={item.id}
        initial={{ opacity: 0, x: -20 }}
        animate={{ opacity: 1, x: 0 }}
        transition={{ duration: 0.3, delay: index * 0.05 }}
      >
        <Button
          variant="ghost"
          className={cn(
            'w-full justify-start gap-3 h-10 px-3 relative group',
            collapsed && 'justify-center px-0 h-10',
            isActive && 'bg-primary/10 text-primary',
            !isActive && 'hover:bg-muted/50'
          )}
          onClick={() => onItemClick(item.id)}
          onMouseEnter={() => setHoveredItem(item.id)}
          onMouseLeave={() => setHoveredItem(null)}
        >
          {/* Active Indicator */}
          {isActive && (
            <motion.div
              layoutId="activeIndicator"
              className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-5 bg-primary rounded-r-full"
              transition={{ type: 'spring', stiffness: 500, damping: 30 }}
            />
          )}

          <item.icon className={cn('h-4 w-4 shrink-0', collapsed && 'h-5 w-5')} />
          
          <AnimatePresence mode="wait">
            {!collapsed && (
              <motion.div
                initial={{ opacity: 0, width: 0 }}
                animate={{ opacity: 1, width: 'auto' }}
                exit={{ opacity: 0, width: 0 }}
                className="flex-1 text-left flex items-center justify-between"
              >
                <span className="text-sm">{item.label}</span>
                {item.badge && (
                  <Badge 
                    variant={isActive ? 'default' : 'secondary'} 
                    className="h-5 px-1.5 text-[10px]"
                  >
                    {item.badge}
                  </Badge>
                )}
              </motion.div>
            )}
          </AnimatePresence>

          {/* Tooltip for collapsed state */}
          {collapsed && isHovered && (
            <motion.div
              initial={{ opacity: 0, x: -10 }}
              animate={{ opacity: 1, x: 0 }}
              className="absolute left-full ml-2 px-3 py-2 bg-popover border rounded-lg shadow-lg z-50 whitespace-nowrap"
            >
              <p className="text-sm font-medium">{item.label}</p>
              {item.description && (
                <p className="text-xs text-muted-foreground">{item.description}</p>
              )}
            </motion.div>
          )}
        </Button>
      </motion.div>
    );
  };

  const renderSection = (title: string, items: any[], startIndex: number) => (
    <div className="space-y-1">
      {!collapsed && (
        <motion.p
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="px-3 py-2 text-[11px] font-semibold text-muted-foreground uppercase tracking-wider"
        >
          {title}
        </motion.p>
      )}
      {items.map((item, index) => renderNavItem(item, startIndex + index))}
    </div>
  );

  return (
    <motion.aside
      initial={false}
      animate={{ width: collapsed ? 72 : 280 }}
      transition={{ duration: 0.3, ease: [0.4, 0, 0.2, 1] }}
      className="fixed left-0 top-0 h-screen bg-card border-r border-border z-50 flex flex-col"
    >
      {/* Header */}
      <div className="p-4 flex items-center justify-between">
        <AnimatePresence mode="wait">
          {!collapsed ? (
            <motion.div
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -20 }}
              className="flex items-center gap-3"
            >
              <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-primary to-primary/60 flex items-center justify-center shadow-lg">
                <Server className="w-5 h-5 text-primary-foreground" />
              </div>
              <div>
                <h1 className="font-bold text-sm tracking-tight">ContainerScope</h1>
                <p className="text-[10px] text-muted-foreground font-medium">Enterprise Platform</p>
              </div>
            </motion.div>
          ) : (
            <motion.div
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1 }}
              className="w-9 h-9 rounded-xl bg-gradient-to-br from-primary to-primary/60 flex items-center justify-center shadow-lg mx-auto"
            >
              <Server className="w-5 h-5 text-primary-foreground" />
            </motion.div>
          )}
        </AnimatePresence>

        <Button
          variant="ghost"
          size="icon"
          onClick={onToggle}
          className={cn(
            'h-8 w-8 rounded-lg hover:bg-muted/50',
            collapsed && 'mx-auto mt-2'
          )}
        >
          {collapsed ? (
            <ChevronRight className="h-4 w-4" />
          ) : (
            <ChevronLeft className="h-4 w-4" />
          )}
        </Button>
      </div>

      {/* Search */}
      {!collapsed && (
        <motion.div
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          className="px-4 pb-3"
        >
          <div className="flex items-center gap-2 px-3 py-2 rounded-xl bg-muted/50 text-muted-foreground hover:bg-muted/70 transition-colors cursor-pointer">
            <Search className="h-4 w-4" />
            <span className="text-sm flex-1">Search...</span>
            <kbd className="pointer-events-none inline-flex h-5 select-none items-center gap-1 rounded-md border bg-background px-1.5 font-mono text-[10px] font-medium text-muted-foreground">
              ⌘K
            </kbd>
          </div>
        </motion.div>
      )}

      {/* Content */}
      <ScrollArea className="flex-1 px-3">
        <div className="space-y-4 py-2">
          {/* Favorites */}
          {!collapsed && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.1 }}
            >
              <p className="px-3 py-2 text-[11px] font-semibold text-muted-foreground uppercase tracking-wider flex items-center gap-2">
                <Star className="h-3 w-3" />
                Favorites
              </p>
              <div className="space-y-1">
                {favorites.map((item, index) => renderNavItem(item, index))}
              </div>
            </motion.div>
          )}

          <Separator className={cn('my-2', collapsed && 'mx-2')} />

          {/* Main Navigation */}
          {renderSection('Navigation', mainNavigation, 0)}

          <Separator className={cn('my-2', collapsed && 'mx-2')} />

          {/* Security */}
          {renderSection('Security', securityNavigation, mainNavigation.length)}

          <Separator className={cn('my-2', collapsed && 'mx-2')} />

          {/* System */}
          {renderSection('System', systemNavigation, mainNavigation.length + securityNavigation.length)}

          {/* Recent Pages */}
          {!collapsed && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.3 }}
            >
              <Separator className="my-2" />
              <p className="px-3 py-2 text-[11px] font-semibold text-muted-foreground uppercase tracking-wider flex items-center gap-2">
                <Clock className="h-3 w-3" />
                Recent
              </p>
              <div className="space-y-1">
                {recentPages.map((page) => (
                  <Button
                    key={page.id}
                    variant="ghost"
                    className="w-full justify-start gap-3 h-8 px-3 text-muted-foreground hover:text-foreground"
                    onClick={() => onItemClick(page.id)}
                  >
                    <span className="text-xs flex-1 text-left">{page.label}</span>
                    <span className="text-[10px] text-muted-foreground">{page.time}</span>
                  </Button>
                ))}
              </div>
            </motion.div>
          )}
        </div>
      </ScrollArea>

      {/* Footer */}
      <div className="p-4 border-t border-border">
        <div className={cn(
          'flex items-center gap-3',
          collapsed && 'flex-col'
        )}>
          <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-violet-500 to-violet-600 flex items-center justify-center text-white font-semibold text-sm shadow-lg">
            A
          </div>
          <AnimatePresence mode="wait">
            {!collapsed && (
              <motion.div
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -20 }}
                className="flex-1 min-w-0"
              >
                <p className="text-sm font-semibold truncate">Admin User</p>
                <p className="text-[11px] text-muted-foreground truncate">admin@containerscope.io</p>
              </motion.div>
            )}
          </AnimatePresence>
          {!collapsed && (
            <div className="flex items-center gap-1">
              <Button variant="ghost" size="icon" className="h-8 w-8 rounded-lg">
                <HelpCircle className="h-4 w-4" />
              </Button>
              <Button variant="ghost" size="icon" className="h-8 w-8 rounded-lg" onClick={logout}>
                <LogOut className="h-4 w-4" />
              </Button>
            </div>
          )}
        </div>
      </div>
    </motion.aside>
  );
}
