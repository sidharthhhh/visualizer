import { create } from 'zustand';
import { Org, Connection, Container, Network, Edge } from '../lib/api';

interface AppState {
  orgs: Org[];
  selectedOrg: Org | null;
  connections: Connection[];
  selectedConnection: Connection | null;
  containers: Container[];
  networks: Network[];
  edges: Edge[];
  selectedContainer: Container | null;
  searchQuery: string;

  setOrgs: (orgs: Org[]) => void;
  setSelectedOrg: (org: Org | null) => void;
  setConnections: (connections: Connection[]) => void;
  setSelectedConnection: (connection: Connection | null) => void;
  setContainers: (containers: Container[]) => void;
  setNetworks: (networks: Network[]) => void;
  setEdges: (edges: Edge[]) => void;
  setSelectedContainer: (container: Container | null) => void;
  setSearchQuery: (query: string) => void;
}

export const useAppStore = create<AppState>((set) => ({
  orgs: [],
  selectedOrg: null,
  connections: [],
  selectedConnection: null,
  containers: [],
  networks: [],
  edges: [],
  selectedContainer: null,
  searchQuery: '',

  setOrgs: (orgs) => set({ orgs }),
  setSelectedOrg: (org) => set({ selectedOrg: org }),
  setConnections: (connections) => set({ connections }),
  setSelectedConnection: (connection) => set({ selectedConnection: connection }),
  setContainers: (containers) => set({ containers }),
  setNetworks: (networks) => set({ networks }),
  setEdges: (edges) => set({ edges }),
  setSelectedContainer: (container) => set({ selectedContainer: container }),
  setSearchQuery: (query) => set({ searchQuery: query }),
}));
