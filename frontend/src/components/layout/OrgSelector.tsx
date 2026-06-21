import { useState } from 'react';
import { ChevronDown, Plus, Building2 } from 'lucide-react';
import { useOrg } from '../../contexts/OrgContext';
import { Org } from '../../lib/api';

interface OrgSelectorProps {
  onCreateOrg?: () => void;
}

export default function OrgSelector({ onCreateOrg }: OrgSelectorProps) {
  const { orgs, selectedOrg, setSelectedOrg } = useOrg();
  const [isOpen, setIsOpen] = useState(false);

  const handleSelect = (org: Org) => {
    setSelectedOrg(org);
    setIsOpen(false);
  };

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 px-3 py-2 rounded-lg border border-border hover:bg-muted/50 transition-colors"
      >
        <Building2 className="w-4 h-4 text-muted-foreground" />
        <span className="text-sm font-medium">{selectedOrg?.name || 'Select Organization'}</span>
        <ChevronDown className={`w-4 h-4 text-muted-foreground transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </button>

      {isOpen && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setIsOpen(false)} />
          <div className="absolute left-0 top-full mt-2 w-64 bg-popover border border-border rounded-lg shadow-lg z-50 overflow-hidden">
            <div className="p-2 border-b border-border">
              <p className="text-xs font-medium text-muted-foreground px-2 py-1">Organizations</p>
            </div>
            <div className="max-h-64 overflow-y-auto">
              {orgs.map((org) => (
                <button
                  key={org.id}
                  onClick={() => handleSelect(org)}
                  className={`w-full flex items-center gap-3 px-3 py-2 text-sm hover:bg-muted/50 transition-colors ${
                    selectedOrg?.id === org.id ? 'bg-primary/10 text-primary' : ''
                  }`}
                >
                  <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center">
                    <Building2 className="w-4 h-4 text-primary" />
                  </div>
                  <div className="flex-1 text-left">
                    <p className="font-medium">{org.name}</p>
                    <p className="text-xs text-muted-foreground">{org.slug}</p>
                  </div>
                  {selectedOrg?.id === org.id && (
                    <div className="w-2 h-2 rounded-full bg-primary" />
                  )}
                </button>
              ))}
            </div>
            {onCreateOrg && (
              <div className="p-2 border-t border-border">
                <button
                  onClick={() => {
                    setIsOpen(false);
                    onCreateOrg();
                  }}
                  className="w-full flex items-center gap-2 px-3 py-2 text-sm text-primary hover:bg-primary/10 rounded transition-colors"
                >
                  <Plus className="w-4 h-4" />
                  <span>Create Organization</span>
                </button>
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
