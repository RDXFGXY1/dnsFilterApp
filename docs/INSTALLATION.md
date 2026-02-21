# Installation Guide

## Overview

DNS Filter is a content filtering application that blocks inappropriate websites at the DNS level. This guide will help you install and configure it on your system.

## System Requirements

- **Operating System**: Linux, Windows, or macOS
- **RAM**: Minimum 512MB
- **Disk Space**: 100MB
- **Administrator/Root privileges**: Required for DNS port 53

## Installation Methods

### Linux (Debian/Ubuntu)

#### Method 1: Using Installation Script

```bash
# Build the application
make build

# Run installation script
sudo chmod +x scripts/install.sh
sudo ./scripts/install.sh

# Start the service
sudo systemctl start dns-filter

# Enable auto-start on boot
sudo systemctl enable dns-filter
```

#### Method 2: Manual Installation

```bash
# Build
make build

# Create directories
sudo mkdir -p /opt/dns-filter/{configs,data/logs,web}

# Copy files
sudo cp build/dns-filter /opt/dns-filter/
sudo cp -r configs/* /opt/dns-filter/configs/
sudo cp -r web/* /opt/dns-filter/web/

# Install service
sudo cp scripts/dns-filter.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable dns-filter
sudo systemctl start dns-filter
```

### Windows

#### Using the Installation Script

1. Download the Windows build
2. Run Command Prompt as Administrator
3. Navigate to the application directory
4. Run: `scripts\install-windows.bat`

#### Manual Installation

1. Create directory: `C:\Program Files\DNS-Filter`
2. Copy `dns-filter-windows-amd64.exe` and rename to `dns-filter.exe`
3. Copy `configs` and `web` folders
4. Install as Windows Service using NSSM or run directly

### macOS

```bash
# Build for macOS
make build

# Create directories
sudo mkdir -p /usr/local/dns-filter/{configs,data/logs,web}

# Copy files
sudo cp build/dns-filter /usr/local/dns-filter/
sudo cp -r configs/* /usr/local/dns-filter/configs/
sudo cp -r web/* /usr/local/dns-filter/web/

# Create launchd plist (see scripts/com.dnsfilter.plist)
sudo cp scripts/com.dnsfilter.plist /Library/LaunchDaemons/
sudo launchctl load /Library/LaunchDaemons/com.dnsfilter.plist
```

## Post-Installation Configuration

### 1. Set System DNS

#### Linux

Edit `/etc/resolv.conf`:
```
nameserver 127.0.0.1
```

Or use NetworkManager:
```bash
nmcli connection modify <connection-name> ipv4.dns "127.0.0.1"
nmcli connection up <connection-name>
```

#### Windows

1. Open Network Connections
2. Right-click your network adapter → Properties
3. Select "Internet Protocol Version 4 (TCP/IPv4)" → Properties
4. Select "Use the following DNS server addresses"
5. Preferred DNS: `127.0.0.1`
6. Alternate DNS: `8.8.8.8` (optional)

#### macOS

1. System Preferences → Network
2. Select your connection → Advanced
3. DNS tab → Click + and add `127.0.0.1`
4. Click OK → Apply

### 2. Change Default Password

1. Access dashboard: `http://localhost:8080`
2. Login with default credentials:
   - Username: `admin`
   - Password: `changeme`
3. Go to Settings → Security
4. Update password

### 3. Configure Filtering

Edit `/opt/dns-filter/configs/config.yaml` (or equivalent path):

```yaml
filtering:
  enabled: true
  block_categories:
    - adult
    - gambling
    - malware
  block_action: "nxdomain"
```

Restart the service after changes:
```bash
sudo systemctl restart dns-filter  # Linux
```

## Network-Wide Filtering

### Router Configuration

To filter all devices on your network:

1. Access your router's admin panel (usually `192.168.1.1` or `192.168.0.1`)
2. Find DNS settings (often under DHCP or WAN settings)
3. Set Primary DNS to your DNS Filter server IP
4. Save and reboot router

### Static IP for Server

Ensure your DNS Filter server has a static IP:

#### Linux
```bash
# Edit /etc/netplan/01-netcfg.yaml
network:
  version: 2
  ethernets:
    eth0:
      dhcp4: no
      addresses: [192.168.1.100/24]
      gateway4: 192.168.1.1
      nameservers:
        addresses: [127.0.0.1, 8.8.8.8]
```

## Verification

### Test DNS Resolution

```bash
# Linux/macOS
nslookup google.com 127.0.0.1
dig google.com @127.0.0.1

# Windows
nslookup google.com 127.0.0.1
```

### Test Blocking

Try accessing a blocked domain:
```bash
nslookup blocked-test-domain.com 127.0.0.1
```

Should return NXDOMAIN or redirect.

### Check Service Status

```bash
# Linux
sudo systemctl status dns-filter
sudo journalctl -u dns-filter -f

# Check logs
tail -f /opt/dns-filter/data/logs/dns-filter.log
```

## Troubleshooting

### DNS Not Working

1. Check if service is running:
   ```bash
   sudo systemctl status dns-filter
   ```

2. Check if port 53 is bound:
   ```bash
   sudo netstat -tulpn | grep :53
   ```

3. Check logs:
   ```bash
   sudo journalctl -u dns-filter -n 100
   ```

### Permission Denied

Port 53 requires root privileges:
```bash
sudo ./dns-filter
```

### Web Dashboard Not Accessible

1. Check if API is running on port 8080:
   ```bash
   netstat -tulpn | grep :8080
   ```

2. Check firewall:
   ```bash
   sudo ufw allow 8080/tcp  # Linux
   ```

### High Memory Usage

Reduce cache size in config:
```yaml
server:
  cache_size: 5000  # Reduce from 10000
```

## Updating

### Linux

```bash
# Stop service
sudo systemctl stop dns-filter

# Build new version
make build

# Replace binary
sudo cp build/dns-filter /opt/dns-filter/

# Start service
sudo systemctl start dns-filter
```

### Backup

Before updating, backup your configuration and database:
```bash
sudo tar -czf dns-filter-backup.tar.gz \
  /opt/dns-filter/configs \
  /opt/dns-filter/data/dns-filter.db
```

## Uninstallation

### Linux

```bash
# Stop and disable service
sudo systemctl stop dns-filter
sudo systemctl disable dns-filter

# Remove service file
sudo rm /etc/systemd/system/dns-filter.service
sudo systemctl daemon-reload

# Remove installation
sudo rm -rf /opt/dns-filter

# Restore DNS settings
# Edit /etc/resolv.conf and remove 127.0.0.1
```

### Windows

1. Stop the service
2. Delete `C:\Program Files\DNS-Filter`
3. Remove firewall rules
4. Restore original DNS settings

## Advanced Configuration

See [docs/CONFIGURATION.md](CONFIGURATION.md) for advanced options including:
- Schedule-based filtering
- Custom blocklists
- DNS-over-HTTPS
- Rate limiting
- Multiple upstream DNS servers

## Support

For issues and questions:
- Check logs: `/opt/dns-filter/data/logs/dns-filter.log`
- Review configuration: `/opt/dns-filter/configs/config.yaml`
- Visit documentation: `docs/`
