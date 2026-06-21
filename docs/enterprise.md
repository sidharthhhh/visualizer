# ContainerScope Enterprise Features

## SSO/SAML/OIDC Integration

### Configuration

```yaml
# config.yaml
auth:
  sso:
    enabled: true
    provider: oidc  # or saml
    oidc:
      issuer: https://accounts.google.com
      client_id: your-client-id
      client_secret: your-client-secret
      redirect_url: https://containerscope.example.com/callback
    saml:
      metadata_url: https://idp.example.com/metadata
      entity_id: containerscope
      acs_url: https://containerscope.example.com/saml/acs
```

### Supported Providers

- Google Workspace
- Microsoft Azure AD
- Okta
- Auth0
- Keycloak
- Any OIDC/SAML provider

## Audit Logging

All user actions are logged with:
- User ID
- Action type
- Resource affected
- Timestamp
- IP address
- User agent

### Query Audit Logs

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "https://api.containerscope.example.com/api/v1/orgs/{orgId}/audit-logs?start=2024-01-01&end=2024-01-31"
```

## Role-Based API Access

| Role | Topology | Metrics | Security | Alerts | Settings |
|------|----------|---------|----------|--------|----------|
| Viewer | Read | Read | Read | Read | - |
| Member | Read | Read | Read | Read | - |
| Admin | Read/Write | Read/Write | Read/Write | Read/Write | Read |
| Owner | Full | Full | Full | Full | Full |

## Data Retention Policies

```yaml
retention:
  metrics: 90d
  flows: 30d
  alerts: 365d
  audit_logs: 730d
  vulnerability_scans: 90d
```

## Multi-Region Support

- Data residency per organization
- Region-specific endpoints
- Cross-region replication (optional)

## Compliance

- SOC 2 Type II
- GDPR compliant
- HIPAA compliant (with BAA)
- ISO 27001
