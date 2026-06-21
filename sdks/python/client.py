"""ContainerScope Python SDK"""

import requests
from typing import Optional, List, Dict, Any
from dataclasses import dataclass
from datetime import datetime


@dataclass
class Container:
    id: str
    connection_id: str
    runtime_id: str
    name: str
    image: str
    state: str
    labels: Dict[str, str]
    created_at: datetime


@dataclass
class Network:
    id: str
    connection_id: str
    name: str
    driver: str
    scope: str
    subnet: str


@dataclass
class Edge:
    id: str
    connection_id: str
    src_container_id: str
    dst_container_id: Optional[str]
    dst_ip: str
    dst_port: int
    protocol: str


@dataclass
class Topology:
    containers: List[Container]
    networks: List[Network]
    edges: List[Edge]


@dataclass
class Alert:
    id: str
    rule_id: str
    rule_name: str
    severity: str
    status: str
    starts_at: datetime
    fired_count: int


class ContainerScopeClient:
    """Client for ContainerScope API"""
    
    def __init__(self, base_url: str, api_key: Optional[str] = None):
        self.base_url = base_url.rstrip('/')
        self.session = requests.Session()
        if api_key:
            self.session.headers['X-API-Key'] = api_key
    
    def get_topology(self, org_id: str, connection_id: str) -> Topology:
        """Get topology for a connection"""
        response = self._get(f'/api/v1/orgs/{org_id}/connections/{connection_id}/topology')
        return Topology(
            containers=[Container(**c) for c in response.get('containers', [])],
            networks=[Network(**n) for n in response.get('networks', [])],
            edges=[Edge(**e) for e in response.get('edges', [])],
        )
    
    def list_containers(self, org_id: str, connection_id: str) -> List[Container]:
        """List all containers for a connection"""
        topology = self.get_topology(org_id, connection_id)
        return topology.containers
    
    def list_connections(self, org_id: str) -> List[Dict[str, Any]]:
        """List all connections for an organization"""
        return self._get(f'/api/v1/orgs/{org_id}/connections')
    
    def get_metrics(
        self,
        org_id: str,
        connection_id: str,
        runtime_id: str,
        metric: str = 'cpu',
    ) -> List[Dict[str, Any]]:
        """Get metrics for a container"""
        response = self._get(
            f'/api/v1/orgs/{org_id}/connections/{connection_id}/metrics',
            params={'runtime_id': runtime_id, 'metric': metric}
        )
        return response.get('results', [])
    
    def list_alerts(self, org_id: str, connection_id: str) -> List[Alert]:
        """List all alerts for a connection"""
        response = self._get(f'/api/v1/orgs/{org_id}/connections/{connection_id}/alerts')
        return [Alert(**a) for a in response]
    
    def list_flows(self, org_id: str, connection_id: str) -> List[Dict[str, Any]]:
        """List network flows for a connection"""
        response = self._get(f'/api/v1/orgs/{org_id}/connections/{connection_id}/flows')
        return response.get('flows', [])
    
    def health(self) -> Dict[str, str]:
        """Check backend health"""
        return self._get('/healthz')
    
    def version(self) -> Dict[str, str]:
        """Get backend version"""
        return self._get('/version')
    
    def _get(self, path: str, params: Optional[Dict] = None) -> Any:
        response = self.session.get(f'{self.base_url}{path}', params=params)
        response.raise_for_status()
        return response.json()
    
    def _post(self, path: str, data: Optional[Dict] = None) -> Any:
        response = self.session.post(f'{self.base_url}{path}', json=data)
        response.raise_for_status()
        return response.json()
