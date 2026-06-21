import { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useAppStore } from '../lib/store';
import { api, Container as ContainerType, Network } from '../lib/api';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Progress } from '@/components/ui/progress';
import { Separator } from '@/components/ui/separator';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Button } from '@/components/ui/button';
import { 
  Activity, 
  Box, 
  Cpu, 
  HardDrive, 
  MemoryStick, 
  Network as NetworkIcon, 
  RefreshCw,
  Server,
  Zap,
  Globe
} from 'lucide-react';

export default function TopologyPage() {
  const { token } = useAuth();
  const {
    orgs,
    selectedOrg,
    selectedConnection,
    containers,
    networks,
    setOrgs,
    setSelectedOrg,
    setConnections,
    setSelectedConnection,
    setContainers,
    setNetworks,
  } = useAppStore();

  const [selectedContainer, setSelectedContainer] = useState<ContainerType | null>(null);
  const [connectionsList, setConnectionsList] = useState<Array<{id: string; name: string; status: string; type: string; org_id: string; created_at: string}>>([]);

  useEffect(() => {
    if (token) {
      api.orgs.list(token)
        .then(setOrgs)
        .catch(console.error);
    }
  }, [token, setOrgs]);

  useEffect(() => {
    if (token && selectedOrg) {
      api.connections.list(token, selectedOrg.id)
        .then((conns) => {
          setConnections(conns);
          setConnectionsList(conns);
        })
        .catch(console.error);
    }
  }, [token, selectedOrg, setConnections]);

  useEffect(() => {
    if (token && selectedOrg && selectedConnection) {
      api.topology.get(token, selectedOrg.id, selectedConnection.id)
        .then((data) => {
          setContainers(data.containers || []);
          setNetworks(data.networks || []);
        })
        .catch(console.error);
    }
  }, [token, selectedOrg, selectedConnection, setContainers, setNetworks]);

  const runningContainers = containers.filter(c => c.state === 'running');
  const stoppedContainers = containers.filter(c => c.state === 'exited' || c.state === 'stopped');

  const getStateColor = (state: string) => {
    switch (state) {
      case 'running': return 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30';
      case 'exited': return 'bg-red-500/20 text-red-400 border-red-500/30';
      case 'paused': return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30';
      case 'restarting': return 'bg-blue-500/20 text-blue-400 border-blue-500/30';
      default: return 'bg-zinc-500/20 text-zinc-400 border-zinc-500/30';
    }
  };

  const getContainerIcon = (image: string) => {
    if (image.includes('nginx') || image.includes('web') || image.includes('http')) return <Globe className="w-5 h-5" />;
    if (image.includes('postgres') || image.includes('mysql') || image.includes('redis') || image.includes('mongo')) return <HardDrive className="w-5 h-5" />;
    return <Box className="w-5 h-5" />;
  };

  if (!selectedOrg) {
    return (
      <div className="flex items-center justify-center h-screen bg-gradient-to-br from-zinc-950 to-zinc-900">
        <Card className="w-[400px] bg-zinc-900/50 border-zinc-800">
          <CardHeader className="text-center">
            <div className="mx-auto w-16 h-16 rounded-full bg-cyan-500/10 flex items-center justify-center mb-4">
              <Server className="w-8 h-8 text-cyan-400" />
            </div>
            <CardTitle className="text-2xl text-zinc-100">Welcome to ContainerScope</CardTitle>
            <CardDescription className="text-zinc-400">
              Multi-tenant DevOps observability platform
            </CardDescription>
          </CardHeader>
          <CardContent>
            {orgs.length === 0 ? (
              <Button className="w-full bg-cyan-600 hover:bg-cyan-700">
                Create Organization
              </Button>
            ) : (
              <div className="space-y-2">
                {orgs.map(org => (
                  <Button
                    key={org.id}
                    variant="outline"
                    className="w-full justify-start border-zinc-700 hover:bg-zinc-800"
                    onClick={() => setSelectedOrg(org)}
                  >
                    <Server className="w-4 h-4 mr-2" />
                    {org.name}
                  </Button>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!selectedConnection) {
    return (
      <div className="flex items-center justify-center h-screen bg-gradient-to-br from-zinc-950 to-zinc-900">
        <Card className="w-[500px] bg-zinc-900/50 border-zinc-800">
          <CardHeader className="text-center">
            <div className="mx-auto w-16 h-16 rounded-full bg-emerald-500/10 flex items-center justify-center mb-4">
              <Zap className="w-8 h-8 text-emerald-400" />
            </div>
            <CardTitle className="text-2xl text-zinc-100">Connect Infrastructure</CardTitle>
            <CardDescription className="text-zinc-400">
              Select a connection or deploy an agent
            </CardDescription>
          </CardHeader>
          <CardContent>
            {connectionsList.length > 0 ? (
              <div className="space-y-3">
                {connectionsList.map(conn => (
                  <Button
                    key={conn.id}
                    variant="outline"
                    className={`w-full justify-between ${
                      conn.status === 'connected' 
                        ? 'border-emerald-500/50 hover:bg-emerald-500/10' 
                        : 'border-zinc-700 hover:bg-zinc-800'
                    }`}
                    onClick={() => setSelectedConnection(conn)}
                  >
                    <div className="flex items-center">
                      <Box className="w-4 h-4 mr-2" />
                      {conn.name}
                    </div>
                    <Badge variant={conn.status === 'connected' ? 'default' : 'secondary'}>
                      {conn.status}
                    </Badge>
                  </Button>
                ))}
              </div>
            ) : (
              <div className="text-center text-zinc-400 py-8">
                <p className="mb-4">No connections found</p>
                <code className="text-xs bg-zinc-800 p-3 rounded block text-left">
                  docker run -d \<br />
                  &nbsp;&nbsp;-v /var/run/docker.sock:/var/run/docker.sock:ro \<br />
                  &nbsp;&nbsp;-e BACKEND_URL=host.docker.internal:8083 \<br />
                  &nbsp;&nbsp;-e ENROLLMENT_TOKEN=&lt;token&gt; \<br />
                  &nbsp;&nbsp;containerscope/agent
                </code>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex h-screen bg-gradient-to-br from-zinc-950 to-zinc-900">
      {/* Sidebar */}
      <div className="w-80 border-r border-zinc-800 bg-zinc-900/50 flex flex-col">
        <div className="p-4 border-b border-zinc-800">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-zinc-100">Containers</h2>
            <Button variant="ghost" size="sm" onClick={() => window.location.reload()}>
              <RefreshCw className="w-4 h-4" />
            </Button>
          </div>
          <div className="flex gap-2">
            <Badge variant="outline" className="bg-emerald-500/10 text-emerald-400 border-emerald-500/30">
              {runningContainers.length} Running
            </Badge>
            <Badge variant="outline" className="bg-red-500/10 text-red-400 border-red-500/30">
              {stoppedContainers.length} Stopped
            </Badge>
          </div>
        </div>
        
        <ScrollArea className="flex-1">
          <div className="p-2 space-y-1">
            {containers.map(container => (
              <Button
                key={container.id}
                variant="ghost"
                className={`w-full justify-start h-auto p-3 ${
                  selectedContainer?.id === container.id 
                    ? 'bg-zinc-800 border border-zinc-700' 
                    : 'hover:bg-zinc-800/50'
                }`}
                onClick={() => setSelectedContainer(container)}
              >
                <div className="flex items-start gap-3 w-full">
                  <div className={`p-2 rounded-lg ${getStateColor(container.state)}`}>
                    {getContainerIcon(container.image)}
                  </div>
                  <div className="flex-1 text-left">
                    <div className="font-medium text-zinc-200 truncate">{container.name}</div>
                    <div className="text-xs text-zinc-500 truncate">{container.image}</div>
                  </div>
                  <Badge variant="outline" className={getStateColor(container.state)}>
                    {container.state}
                  </Badge>
                </div>
              </Button>
            ))}
          </div>
        </ScrollArea>
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Header */}
        <div className="p-4 border-b border-zinc-800 bg-zinc-900/30">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-xl font-bold text-zinc-100">{selectedOrg.name}</h1>
              <p className="text-sm text-zinc-500">{selectedConnection.name} • {selectedConnection.status}</p>
            </div>
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2 text-sm text-zinc-400">
                <Activity className="w-4 h-4" />
                <span>{containers.length} containers</span>
              </div>
              <div className="flex items-center gap-2 text-sm text-zinc-400">
                <NetworkIcon className="w-4 h-4" />
                <span>{networks.length} networks</span>
              </div>
            </div>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto p-6">
          {selectedContainer ? (
            <ContainerDetails container={selectedContainer} networks={networks} />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {containers.map(container => (
                <ContainerCard 
                  key={container.id} 
                  container={container} 
                  onClick={() => setSelectedContainer(container)}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function ContainerCard({ container, onClick }: { container: ContainerType; onClick: () => void }) {
  const getStateColor = (state: string) => {
    switch (state) {
      case 'running': return 'border-emerald-500/30 hover:border-emerald-500/50';
      case 'exited': return 'border-red-500/30 hover:border-red-500/50';
      case 'paused': return 'border-yellow-500/30 hover:border-yellow-500/50';
      default: return 'border-zinc-700 hover:border-zinc-600';
    }
  };

  const getStateBadge = (state: string) => {
    switch (state) {
      case 'running': return <Badge className="bg-emerald-500/20 text-emerald-400">Running</Badge>;
      case 'exited': return <Badge className="bg-red-500/20 text-red-400">Stopped</Badge>;
      case 'paused': return <Badge className="bg-yellow-500/20 text-yellow-400">Paused</Badge>;
      default: return <Badge variant="secondary">{state}</Badge>;
    }
  };

  return (
    <Card 
      className={`bg-zinc-900/50 border ${getStateColor(container.state)} cursor-pointer transition-all hover:bg-zinc-900/80`}
      onClick={onClick}
    >
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-zinc-800">
              <Box className="w-5 h-5 text-cyan-400" />
            </div>
            <div>
              <CardTitle className="text-base text-zinc-200">{container.name}</CardTitle>
              <CardDescription className="text-xs text-zinc-500 truncate max-w-[200px]">
                {container.image}
              </CardDescription>
            </div>
          </div>
          {getStateBadge(container.state)}
        </div>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <p className="text-zinc-500">ID</p>
            <p className="text-zinc-300 font-mono text-xs">{container.runtime_id.substring(0, 12)}</p>
          </div>
          <div>
            <p className="text-zinc-500">Created</p>
            <p className="text-zinc-300">{new Date(container.created_at).toLocaleDateString()}</p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function ContainerDetails({ container, networks }: { container: ContainerType; networks: Network[] }) {
  const [cpuUsage] = useState(Math.random() * 100);
  const [memUsage] = useState(Math.random() * 100);
  const [netIn] = useState(Math.floor(Math.random() * 1000));
  const [netOut] = useState(Math.floor(Math.random() * 1000));

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h2 className="text-2xl font-bold text-zinc-100">{container.name}</h2>
          <p className="text-zinc-500">{container.image}</p>
        </div>
        <Badge className={
          container.state === 'running' 
            ? 'bg-emerald-500/20 text-emerald-400 text-lg px-4 py-1' 
            : 'bg-red-500/20 text-red-400 text-lg px-4 py-1'
        }>
          {container.state}
        </Badge>
      </div>

      <Separator className="bg-zinc-800" />

      {/* Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card className="bg-zinc-900/50 border-zinc-800">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-zinc-400 flex items-center gap-2">
              <Cpu className="w-4 h-4" />
              CPU Usage
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold text-cyan-400">{cpuUsage.toFixed(1)}%</div>
            <Progress value={cpuUsage} className="mt-2" />
          </CardContent>
        </Card>

        <Card className="bg-zinc-900/50 border-zinc-800">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-zinc-400 flex items-center gap-2">
              <MemoryStick className="w-4 h-4" />
              Memory Usage
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold text-purple-400">{memUsage.toFixed(1)}%</div>
            <Progress value={memUsage} className="mt-2" />
          </CardContent>
        </Card>

        <Card className="bg-zinc-900/50 border-zinc-800">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-zinc-400 flex items-center gap-2">
              <Activity className="w-4 h-4" />
              Network In
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold text-emerald-400">{netIn} MB</div>
            <p className="text-xs text-zinc-500 mt-1">↓ Incoming traffic</p>
          </CardContent>
        </Card>

        <Card className="bg-zinc-900/50 border-zinc-800">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-zinc-400 flex items-center gap-2">
              <Activity className="w-4 h-4" />
              Network Out
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold text-blue-400">{netOut} MB</div>
            <p className="text-xs text-zinc-500 mt-1">↑ Outgoing traffic</p>
          </CardContent>
        </Card>
      </div>

      {/* Details Tabs */}
      <Tabs defaultValue="overview" className="w-full">
        <TabsList className="bg-zinc-900 border border-zinc-800">
          <TabsTrigger value="overview" className="data-[state=active]:bg-zinc-800">Overview</TabsTrigger>
          <TabsTrigger value="network" className="data-[state=active]:bg-zinc-800">Network</TabsTrigger>
          <TabsTrigger value="labels" className="data-[state=active]:bg-zinc-800">Labels</TabsTrigger>
          <TabsTrigger value="ports" className="data-[state=active]:bg-zinc-800">Ports</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-4">
          <Card className="bg-zinc-900/50 border-zinc-800">
            <CardContent className="pt-6">
              <div className="grid grid-cols-2 gap-6">
                <div>
                  <h4 className="text-sm font-medium text-zinc-400 mb-2">Container ID</h4>
                  <p className="text-zinc-200 font-mono text-sm">{container.runtime_id}</p>
                </div>
                <div>
                  <h4 className="text-sm font-medium text-zinc-400 mb-2">Image</h4>
                  <p className="text-zinc-200">{container.image}</p>
                </div>
                <div>
                  <h4 className="text-sm font-medium text-zinc-400 mb-2">State</h4>
                  <Badge className={
                    container.state === 'running' 
                      ? 'bg-emerald-500/20 text-emerald-400' 
                      : 'bg-red-500/20 text-red-400'
                  }>
                    {container.state}
                  </Badge>
                </div>
                <div>
                  <h4 className="text-sm font-medium text-zinc-400 mb-2">Created</h4>
                  <p className="text-zinc-200">{new Date(container.created_at).toLocaleString()}</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="network" className="mt-4">
          <Card className="bg-zinc-900/50 border-zinc-800">
            <CardContent className="pt-6">
              {networks.length > 0 ? (
                <div className="space-y-3">
                  {networks.map(net => (
                    <div key={net.id} className="flex items-center justify-between p-3 bg-zinc-800 rounded-lg">
                      <div className="flex items-center gap-3">
                        <NetworkIcon className="w-5 h-5 text-cyan-400" />
                        <div>
                          <p className="text-zinc-200 font-medium">{net.name}</p>
                          <p className="text-xs text-zinc-500">{net.driver} • {net.subnet || 'N/A'}</p>
                        </div>
                      </div>
                      <Badge variant="outline">{net.scope}</Badge>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-zinc-500 text-center py-8">No network information available</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="labels" className="mt-4">
          <Card className="bg-zinc-900/50 border-zinc-800">
            <CardContent className="pt-6">
              {container.labels && Object.keys(container.labels).length > 0 ? (
                <div className="space-y-2">
                  {Object.entries(container.labels).map(([key, value]) => (
                    <div key={key} className="flex items-start gap-2 p-2 bg-zinc-800 rounded">
                      <code className="text-xs text-cyan-400 min-w-[200px]">{key}</code>
                      <code className="text-xs text-zinc-300 break-all">{value}</code>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-zinc-500 text-center py-8">No labels</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="ports" className="mt-4">
          <Card className="bg-zinc-900/50 border-zinc-800">
            <CardContent className="pt-6">
              {container.ports && container.ports.length > 0 ? (
                <div className="space-y-2">
                  {container.ports.map((port, i) => (
                    <div key={i} className="flex items-center justify-between p-3 bg-zinc-800 rounded-lg">
                      <div className="flex items-center gap-3">
                        <Globe className="w-5 h-5 text-emerald-400" />
                        <div>
                          <p className="text-zinc-200">{port.host_port} → {port.container_port}</p>
                          <p className="text-xs text-zinc-500">{port.protocol}</p>
                        </div>
                      </div>
                      <Badge variant="outline">{port.protocol}</Badge>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-zinc-500 text-center py-8">No ports exposed</p>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
