# Apache Reverse Proxy Configuration

## Prerequisites

```bash
# Enable required Apache modules
sudo a2enmod proxy
sudo a2enmod proxy_http
sudo a2enmod proxy_wstunnel
sudo a2enmod rewrite
sudo a2enmod ssl
sudo a2enmod headers
```

## Virtual Host Configuration

Create `/etc/apache2/sites-available/containerscope.conf`:

```apache
<VirtualHost *:80>
    ServerName containerscope.yourdomain.com
    
    # Redirect to HTTPS
    RewriteEngine On
    RewriteCond %{HTTPS} off
    RewriteRule ^(.*)$ https://%{HTTP_HOST}%{REQUEST_URI} [L,R=301]
</VirtualHost>

<VirtualHost *:443>
    ServerName containerscope.yourdomain.com
    
    # SSL Configuration
    SSLEngine on
    SSLCertificateFile /path/to/certificate.crt
    SSLCertificateKeyFile /path/to/private.key
    SSLCertificateChainFile /path/to/chain.crt
    
    # Security Headers
    Header always set X-Content-Type-Options "nosniff"
    Header always set X-Frame-Options "DENY"
    Header always set X-XSS-Protection "1; mode=block"
    Header always set Referrer-Policy "strict-origin-when-cross-origin"
    Header always set Strict-Transport-Security "max-age=31536000; includeSubDomains"
    
    # Frontend (Port 3000)
    ProxyPass / http://localhost:3000/
    ProxyPassReverse / http://localhost:3000/
    
    # Backend API (Port 8080)
    ProxyPass /api/ http://localhost:8080/api/
    ProxyPassReverse /api/ http://localhost:8080/api/
    
    # WebSocket (Port 8080)
    RewriteEngine On
    RewriteCond %{HTTP:Upgrade} websocket [NC]
    RewriteCond %{HTTP:Connection} upgrade [NC]
    RewriteRule ^/ws/(.*)$ ws://localhost:8080/ws/$1 [P,L]
    
    ProxyPass /ws/ ws://localhost:8080/ws/
    ProxyPassReverse /ws/ ws://localhost:8080/ws/
    
    # Health Check
    ProxyPass /healthz http://localhost:8080/healthz
    ProxyPassReverse /healthz http://localhost:8080/healthz
    
    # Logging
    ErrorLog ${APACHE_LOG_DIR}/containerscope-error.log
    CustomLog ${APACHE_LOG_DIR}/containerscope-access.log combined
</VirtualHost>
```

## Enable the Site

```bash
sudo a2ensite containerscope.conf
sudo systemctl restart apache2
```

## Test Configuration

```bash
# Test Apache config
sudo apache2ctl configtest

# Test the proxy
curl -I http://localhost:3000
curl -I http://localhost:8080/healthz
```

## Port Reference

| Service | Port | Description |
|---------|------|-------------|
| Frontend | 3000 | React UI |
| Backend API | 8080 | REST API |
| gRPC | 8081 | Agent communication |
| Postgres | 5432 | Database |
| VictoriaMetrics | 8428 | Metrics |
| ClickHouse | 8123/9000 | Flow storage |
| MinIO | 9100/9001 | Object storage |

## Troubleshooting

### WebSocket not working
Make sure `mod_proxy_wstunnel` is enabled:
```bash
sudo a2enmod proxy_wstunnel
sudo systemctl restart apache2
### 502 Bad Gateway
Check if backend is running:
```bash
curl http://localhost:8080/healthz
```

### SSL Issues
Verify certificates:
```bash
sudo apache2ctl -t
```
