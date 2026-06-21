import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import { api, Org } from '../lib/api';
import { useAuth } from './AuthContext';

interface OrgContextType {
  orgs: Org[];
  selectedOrg: Org | null;
  isLoading: boolean;
  setSelectedOrg: (org: Org | null) => void;
  createOrg: (name: string, slug: string) => Promise<Org>;
  refreshOrgs: () => Promise<void>;
}

const OrgContext = createContext<OrgContextType | null>(null);

export function OrgProvider({ children }: { children: ReactNode }) {
  const { token } = useAuth();
  const [orgs, setOrgs] = useState<Org[]>([]);
  const [selectedOrg, setSelectedOrg] = useState<Org | null>(() => {
    const stored = localStorage.getItem('selectedOrg');
    return stored ? JSON.parse(stored) : null;
  });
  const [isLoading, setIsLoading] = useState(true);

  const refreshOrgs = useCallback(async () => {
    if (!token) return;
    try {
      const data = await api.orgs.list(token);
      setOrgs(data);
      
      // If no org selected, select the first one
      if (!selectedOrg && data.length > 0) {
        setSelectedOrg(data[0]);
      }
      
      // If selected org no longer exists, select first
      if (selectedOrg && !data.find(o => o.id === selectedOrg.id)) {
        setSelectedOrg(data.length > 0 ? data[0] : null);
      }
    } catch (err) {
      console.error('Failed to fetch orgs:', err);
    } finally {
      setIsLoading(false);
    }
  }, [token, selectedOrg]);

  useEffect(() => {
    refreshOrgs();
  }, [token]);

  const handleSetSelectedOrg = useCallback((org: Org | null) => {
    setSelectedOrg(org);
    if (org) {
      localStorage.setItem('selectedOrg', JSON.stringify(org));
    } else {
      localStorage.removeItem('selectedOrg');
    }
  }, []);

  const createOrg = useCallback(async (name: string, slug: string): Promise<Org> => {
    if (!token) throw new Error('Not authenticated');
    const result = await api.orgs.create(token, name, slug);
    await refreshOrgs();
    handleSetSelectedOrg(result.org);
    return result.org;
  }, [token, refreshOrgs, handleSetSelectedOrg]);

  return (
    <OrgContext.Provider value={{
      orgs,
      selectedOrg,
      isLoading,
      setSelectedOrg: handleSetSelectedOrg,
      createOrg,
      refreshOrgs,
    }}>
      {children}
    </OrgContext.Provider>
  );
}

// eslint-disable-next-line react-refresh/only-export-components
export function useOrg() {
  const context = useContext(OrgContext);
  if (!context) {
    throw new Error('useOrg must be used within an OrgProvider');
  }
  return context;
}
