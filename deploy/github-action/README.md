# ContainerScope GitHub Actions Integration

## Usage

```yaml
name: Security Scan
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  security-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: ContainerScope Security Check
        uses: containerscope/security-action@v1
        with:
          api-key: ${{ secrets.CONTAINERSCOPE_API_KEY }}
          org-id: ${{ secrets.CONTAINERSCOPE_ORG_ID }}
          connection-id: ${{ secrets.CONTAINERSCOPE_CONNECTION_ID }}
          fail-on-critical: true
          fail-on-high: false
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `api-key` | ContainerScope API key | Yes | - |
| `org-id` | Organization ID | Yes | - |
| `connection-id` | Connection ID | Yes | - |
| `image` | Image to scan | No | - |
| `fail-on-critical` | Fail if critical vulns found | No | `true` |
| `fail-on-high` | Fail if high vulns found | No | `false` |
| `fail-on-misconfigs` | Fail if misconfigurations found | No | `false` |

## Example Workflow

```yaml
name: Deploy
on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Build Docker Image
        run: docker build -t myapp:${{ github.sha }} .
      
      - name: Scan Image
        uses: containerscope/security-action@v1
        with:
          api-key: ${{ secrets.CONTAINERSCOPE_API_KEY }}
          org-id: ${{ secrets.CONTAINERSCOPE_ORG_ID }}
          connection-id: ${{ secrets.CONTAINERSCOPE_CONNECTION_ID }}
          image: myapp:${{ github.sha }}
          fail-on-critical: true
      
      - name: Push Image
        if: success()
        run: docker push myapp:${{ github.sha }}
      
      - name: Deploy
        if: success()
        run: kubectl apply -f k8s/
```

## Deployment Tracking

```yaml
      - name: Track Deployment
        uses: containerscope/deploy-action@v1
        with:
          api-key: ${{ secrets.CONTAINERSCOPE_API_KEY }}
          org-id: ${{ secrets.CONTAINERSCOPE_ORG_ID }}
          connection-id: ${{ secrets.CONTAINERSCOPE_CONNECTION_ID }}
          environment: production
          version: ${{ github.sha }}
          commit: ${{ github.sha }}
```
