# Usage Guide

## Getting Started

Once DNS Filter is installed and running, you can manage it through the web dashboard or command-line interface.

## Web Dashboard

### Accessing the Dashboard

Open your browser and navigate to:
```
http://localhost:8080
```

Or if accessing from another device on your network:
```
http://[server-ip]:8080
```

Default credentials:
- **Username**: admin
- **Password**: changeme

⚠️ **Important**: Change the default password immediately after first login!

### Dashboard Overview

The dashboard provides several sections:

#### 1. Statistics Overview

Real-time statistics showing:
- **Total Queries**: Number of DNS queries processed
- **Blocked Queries**: Number of blocked requests
- **Block Rate**: Percentage of queries blocked
- **Blocked Domains**: Total domains in blocklist

#### 2. Top Blocked Domains

Shows the most frequently blocked domains in the last 24 hours. This helps you:
- Identify problematic websites
- Understand blocking patterns
- Add legitimate sites to whitelist if needed

#### 3. Recent Activity

Live feed of recently blocked DNS queries showing:
- Timestamp of the query
- Domain that was blocked
- Client IP address that made the request

Refreshes automatically every 5 seconds.

#### 4. Whitelist Management

Add trusted domains that should never be blocked:

```
Example: trusted-site.com
```

Use wildcards for subdomains:
```
*.example.com  (allows all subdomains of example.com)
```

#### 5. System Actions

- **Update Blocklists**: Downloads latest blocklists from configured sources
- **Clear DNS Cache**: Clears cached DNS responses
- **Export Logs**: Download query logs (coming soon)

## Managing Blocklists

### Viewing Current Blocklists

Blocklists are configured in `configs/config.yaml`:

```yaml
blocklists:
  auto_update_interval: 24  # Hours
  sources:
    - name: "StevenBlack Unified"
      url: "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts"
      category: "general"
      enabled: true
```

### Adding Custom Blocklists

1. Edit the configuration file
2. Add a new source:

```yaml
- name: "My Custom List"
  url: "https://example.com/blocklist.txt"
  category: "custom"
  enabled: true
```

3. Restart the service:
```bash
sudo systemctl restart dns-filter
```

4. Update blocklists from the dashboard

### Supported Blocklist Formats

- **Hosts file format**: `0.0.0.0 domain.com`
- **AdBlock format**: `||domain.com^`
- **Plain list**: One domain per line

## Whitelist Management

### Adding Domains to Whitelist

#### Via Web Dashboard:
1. Navigate to "Whitelist Management"
2. Enter domain name
3. Click "Add to Whitelist"

#### Via Configuration File:
Edit `configs/config.yaml`:
```yaml
whitelist:
  domains:
    - "example-trusted-site.com"
    - "*.safedomain.org"
```

### Wildcard Patterns

Use wildcards to whitelist entire domains:

```yaml
whitelist:
  domains:
    - "*.google.com"      # Allows all Google subdomains
    - "*.microsoft.com"   # Allows all Microsoft subdomains
```

## Schedule-Based Filtering

Control when filtering is active using schedules.

### Configuration

Edit `configs/config.yaml`:

```yaml
filtering:
  schedule:
    enabled: true
    rules:
      - name: "School Hours"
        days: ["monday", "tuesday", "wednesday", "thursday", "friday"]
        start_time: "08:00"
        end_time: "15:00"
        strict_mode: true
      
      - name: "Evening Restriction"
        days: ["monday", "tuesday", "wednesday", "thursday", "sunday"]
        start_time: "20:00"
        end_time: "22:00"
        strict_mode: false
```

### Schedule Parameters

- **days**: Days of the week (lowercase)
- **start_time**: When restriction begins (24-hour format)
- **end_time**: When restriction ends (24-hour format)
- **strict_mode**: 
  - `true`: Block everything except whitelist
  - `false`: Use normal blocklist

### Example Schedules

**Block everything during homework time:**
```yaml
- name: "Homework Time"
  days: ["monday", "tuesday", "wednesday", "thursday", "friday"]
  start_time: "16:00"
  end_time: "18:00"
  strict_mode: true
```

**Relaxed filtering on weekends:**
```yaml
- name: "Weekend Mode"
  days: ["saturday", "sunday"]
  start_time: "00:00"
  end_time: "23:59"
  strict_mode: false
```

## Monitoring and Logs

### Viewing Logs

#### Web Dashboard:
Real-time monitoring in the "Recent Activity" section

#### Log Files:
```bash
# View live logs
tail -f /opt/dns-filter/data/logs/dns-filter.log

# View systemd logs (Linux)
sudo journalctl -u dns-filter -f
```

### Log Configuration

Configure logging in `configs/config.yaml`:

```yaml
logging:
  level: "info"  # debug, info, warn, error
  file: "./data/logs/dns-filter.log"
  log_queries: true        # Log all DNS queries
  log_blocked_only: false  # Log only blocked queries
```

### Log Levels

- **debug**: Detailed information for troubleshooting
- **info**: General information about operations
- **warn**: Warning messages
- **error**: Error messages only

## Statistics and Analytics

### Database Query

The SQLite database stores all blocked queries:

```bash
sqlite3 /opt/dns-filter/data/dns-filter.db

# View recent blocks
SELECT * FROM blocked_queries ORDER BY timestamp DESC LIMIT 10;

# Count by domain
SELECT domain, COUNT(*) as count 
FROM blocked_queries 
GROUP BY domain 
ORDER BY count DESC 
LIMIT 20;

# Queries by client
SELECT client_ip, COUNT(*) as count 
FROM blocked_queries 
GROUP BY client_ip 
ORDER BY count DESC;
```

### API Access

Access statistics via REST API:

```bash
# Get overall stats
curl http://localhost:8080/api/stats/blocked

# Get top blocked domains
curl http://localhost:8080/api/stats/top-blocked

# Get recent queries
curl http://localhost:8080/api/recent
```

## Network-Wide Protection

### Router Setup

1. **Find Router IP**: Usually `192.168.1.1` or `192.168.0.1`
2. **Login** to router admin panel
3. **Navigate** to DHCP/DNS settings
4. **Set Primary DNS** to your DNS Filter server IP
5. **Save** and reboot router

### DHCP Configuration

To force all devices to use DNS Filter:

```
Primary DNS: 192.168.1.100  (your server)
Secondary DNS: (leave empty or use 1.1.1.1)
```

### Device-Specific Configuration

Some devices cache DNS settings. After changing router DNS:

1. **Restart** the device
2. **Renew DHCP lease**:
   ```bash
   # Linux
   sudo dhclient -r && sudo dhclient
   
   # Windows
   ipconfig /release
   ipconfig /renew
   
   # macOS
   sudo dscacheutil -flushcache
   ```

## Performance Tuning

### Cache Settings

Adjust cache for performance:

```yaml
server:
  cache_size: 10000    # Number of entries (increase for busy networks)
  cache_ttl: 3600      # Time to live in seconds
```

### Worker Threads

For high-traffic networks:

```yaml
server:
  workers: 8  # Increase for multi-core systems
```

### Database Maintenance

Clean old logs periodically:

```yaml
database:
  log_retention_days: 30  # Auto-delete logs older than 30 days
  max_log_entries: 100000 # Maximum log entries
```

## Troubleshooting Common Issues

### Sites Not Loading

1. **Check if domain is blocked**:
   ```bash
   nslookup problematic-site.com 127.0.0.1
   ```

2. **Add to whitelist** if legitimate
3. **Check upstream DNS** is responding:
   ```bash
   nslookup google.com 8.8.8.8
   ```

### Slow DNS Resolution

1. **Increase cache size**
2. **Add more upstream DNS servers**
3. **Check network latency** to upstream DNS

### Service Not Starting

1. **Check port 53** is not in use:
   ```bash
   sudo netstat -tulpn | grep :53
   ```

2. **Verify configuration** syntax:
   ```bash
   ./dns-filter --config configs/config.yaml --validate
   ```

3. **Review logs** for errors

### High Memory Usage

1. **Reduce cache size**
2. **Limit log entries**
3. **Clean old blocklist entries**

## Best Practices

### Security

1. ✅ Change default admin password
2. ✅ Enable HTTPS for web dashboard
3. ✅ Restrict dashboard access to local network
4. ✅ Regularly update blocklists
5. ✅ Review logs for suspicious activity

### Performance

1. ✅ Use SSD for database
2. ✅ Enable DNS caching
3. ✅ Configure appropriate cache size
4. ✅ Use multiple upstream DNS servers
5. ✅ Clean old logs regularly

### Maintenance

1. ✅ Backup configuration weekly
2. ✅ Monitor disk space
3. ✅ Review blocked domains monthly
4. ✅ Update software regularly
5. ✅ Test failover scenarios

## Advanced Usage

For advanced topics, see:
- [Configuration Guide](CONFIGURATION.md)
- [API Documentation](API.md)
- [Development Guide](DEVELOPMENT.md)
