# Quick Start Guide

Get DNS Filter up and running in 5 minutes!

## Prerequisites

- Go 1.21+ installed
- Admin/root privileges
- 512MB RAM minimum

## Installation (Linux/macOS)

```bash
# 1. Clone or download the project
cd dns-filter-app

# 2. Install dependencies
make deps

# 3. Create required directories
make setup

# 4. Build the application
make build

# 5. Run (requires sudo for port 53)
sudo ./build/dns-filter
```

## Installation (Windows)

```bash
# 1. Download and extract the project

# 2. Install Go dependencies
go mod download

# 3. Build for Windows
go build -o dns-filter.exe ./cmd/server

# 4. Run as Administrator
.\dns-filter.exe
```

## Set Your System DNS

### Linux
```bash
# Edit /etc/resolv.conf
echo "nameserver 127.0.0.1" | sudo tee /etc/resolv.conf
```

### Windows
1. Control Panel → Network Connections
2. Right-click adapter → Properties → IPv4
3. Set DNS to: `127.0.0.1`

### macOS
1. System Preferences → Network
2. Advanced → DNS → Add `127.0.0.1`

## Access Web Dashboard

Open browser:
```
http://localhost:8080
```

Login:
- Username: `admin`
- Password: `changeme`

## First Steps

1. **Change Password**: Go to Settings → Update password
2. **Update Blocklists**: Click "Update Blocklists" button
3. **Test Blocking**: Try accessing a known adult site
4. **Add Whitelist**: Add trusted domains

## Verify It's Working

```bash
# Should resolve normally
nslookup google.com 127.0.0.1

# Should be blocked (NXDOMAIN)
nslookup pornhub.com 127.0.0.1
```

## Configuration

Edit `configs/config.yaml` to customize:

```yaml
filtering:
  enabled: true
  block_categories:
    - adult
    - gambling
    - malware
  
server:
  dns_port: 53
  api_port: 8080
```

## Run as Background Service

### Linux (systemd)
```bash
# Install
sudo make install-service

# Start
sudo systemctl start dns-filter

# Enable auto-start
sudo systemctl enable dns-filter
```

### Windows
```bash
# Use NSSM or Task Scheduler
# See docs/INSTALLATION.md
```

## Network-Wide Filtering

### Configure Router

1. Login to router (usually 192.168.1.1)
2. Find DNS settings
3. Set Primary DNS to your server IP
4. Save and reboot router

All devices will now be protected!

## Troubleshooting

### Port 53 already in use
```bash
# Check what's using port 53
sudo netstat -tulpn | grep :53

# Stop systemd-resolved (Ubuntu)
sudo systemctl stop systemd-resolved
```

### Can't access dashboard
```bash
# Check if running
sudo systemctl status dns-filter

# Check logs
tail -f data/logs/dns-filter.log
```

### DNS not working
```bash
# Test local DNS
nslookup google.com 127.0.0.1

# Check upstream DNS
nslookup google.com 8.8.8.8
```

## What's Being Blocked?

Default blocklists include:
- 🔞 Adult content
- 🎰 Gambling sites
- 🦠 Malware domains
- 📧 Phishing sites
- 📢 Advertising trackers

## Customize Filtering

### Add Custom Blocklist

Edit `configs/config.yaml`:

```yaml
blocklists:
  sources:
    - name: "My Custom List"
      url: "https://example.com/blocklist.txt"
      enabled: true
```

### Whitelist a Site

Via dashboard:
1. Go to Whitelist Management
2. Enter domain
3. Click Add

Via config:
```yaml
whitelist:
  domains:
    - "trusted-site.com"
```

## Next Steps

📚 **Read the full guides:**
- [Installation Guide](docs/INSTALLATION.md)
- [Usage Guide](docs/USAGE.md)
- [Configuration Guide](docs/CONFIGURATION.md)

🔧 **Advanced Features:**
- Schedule-based filtering
- Custom blocklists
- API integration
- Multi-user support

🛡️ **Security:**
- Enable HTTPS
- Configure firewall
- Regular updates
- Monitor logs

## Support

Having issues? Check:
1. Log files: `data/logs/dns-filter.log`
2. Configuration: `configs/config.yaml`
3. Service status: `systemctl status dns-filter`

## Important Notes

⚠️ **Change Default Password**: The default password is `changeme` - change it immediately!

⚠️ **Backup Configuration**: Before updates, backup your config and database

⚠️ **Root Required**: DNS filtering requires running as root/administrator

## Success!

If you see blocked domains in the dashboard, you're all set! 🎉

Your network is now protected from inappropriate content.
