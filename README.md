# DNS Content Filter

A high-performance DNS-based content filtering application built in Go. Blocks inappropriate websites at the DNS level, running as a background service with a web-based management dashboard.

## Features

- 🚀 **High-Performance DNS Server** - Built with Go for speed and efficiency
- 🛡️ **Content Filtering** - Block adult/inappropriate websites at DNS level
- 🎯 **Category-Based Blocking** - Predefined and custom blocklists
- ⚙️ **Web Dashboard** - Easy-to-use management interface
- 📊 **Analytics & Logging** - Track blocked requests and statistics
- ⏰ **Schedule-Based Filtering** - Time-based blocking rules
- 🔐 **Whitelist Management** - Allow specific trusted sites
- 🌐 **Multi-Device Support** - Works for entire network
- 💾 **SQLite Database** - Lightweight, no external dependencies

## Project Structure

```
dns-filter-app/
├── cmd/
│   ├── server/          # Main DNS server application
│   └── cli/             # Command-line management tool
├── internal/
│   ├── dns/             # DNS server logic
│   ├── filter/          # Filtering engine
│   ├── database/        # Database operations
│   ├── api/             # REST API handlers
│   └── config/          # Configuration management
├── web/
│   ├── static/          # CSS, JS, images
│   └── templates/       # HTML templates
├── pkg/
│   ├── blocklist/       # Blocklist management
│   └── logger/          # Logging utilities
├── configs/             # Configuration files
├── scripts/             # Build and deployment scripts
├── data/                # Database and blocklists
└── docs/                # Documentation
```

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Admin/root privileges (for DNS port 53)

### Installation

```bash
curl -fsSL https://raw.githubusercontent.com/RDXFGXY1/dnsFilterApp/main/setup.sh | sudo bash

# Or Clone the repository and Install it manually
git clone https://github.com/RDXFGXY1/dnsFilterApp.git
cd dns-filter-app

# Build the application
go build -o dns-filter ./cmd/server
```

### Configuration

Edit `configs/config.yaml`:

```yaml
server:
  dns_port: 53
  api_port: 8080
  upstream_dns: "8.8.8.8:53"

filtering:
  enabled: true
  block_categories:
    - adult
    - malware
    - gambling

database:
  path: "./data/dns-filter.db"
```

### Web Dashboard

Access the dashboard at: `http://localhost:8080`

Default credentials:
- Username: `admin`
- Password: `changeme` (please change on first login)

## Usage

### Running as Background Service

#### Linux (systemd)
```bash
sudo cp scripts/dns-filter.service /etc/systemd/system/
sudo systemctl enable dns-filter
sudo systemctl start dns-filter
```

#### Windows
```bash
# Run as Windows Service
./dns-filter install
./dns-filter start
```

#### macOS (launchd)
```bash
sudo cp scripts/com.dnsfilter.plist /Library/LaunchDaemons/
sudo launchctl load /Library/LaunchDaemons/com.dnsfilter.plist
```

### Setting as System DNS

**Windows:**
1. Control Panel → Network Connections
2. Right-click your connection → Properties
3. Select IPv4 → Properties
4. Set DNS to: `127.0.0.1`

**Linux:**
```bash
# Edit /etc/resolv.conf
nameserver 127.0.0.1
```

**macOS:**
1. System Preferences → Network
2. Select connection → Advanced → DNS
3. Add: `127.0.0.1`

## Development

### Building from Source

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Build for current platform
make build

# Build for all platforms
make build-all
```

### Running in Development Mode

```bash
# Run with auto-reload
go run cmd/server/main.go --dev
```

### Adding Custom Blocklists

```bash
# Via CLI
./dns-filter blocklist add --url "https://example.com/blocklist.txt"

# Via API
curl -X POST http://localhost:8080/api/blocklists \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/blocklist.txt", "enabled": true}'
```

## API Documentation

### Authentication
All API requests require authentication using Bearer token.

### Endpoints

- `GET /api/stats` - Get filtering statistics
- `GET /api/blocked` - List recently blocked domains
- `POST /api/whitelist` - Add domain to whitelist
- `DELETE /api/whitelist/:domain` - Remove from whitelist
- `GET /api/blocklists` - List all blocklists
- `POST /api/blocklists` - Add new blocklist

Full API documentation: [docs/API.md](docs/API.md)

## Security Considerations

- Run with minimal privileges
- Change default admin password
- Enable HTTPS for web dashboard in production
- Regularly update blocklists
- Monitor logs for unusual activity

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) first.

## License

MIT License - see [LICENSE](LICENSE) file for details

## Support

- Documentation: [docs/](docs/)
- Issues: GitHub Issues
- Discussions: GitHub Discussions

## Roadmap

- [ ] DNS-over-HTTPS (DoH) support
- [ ] DNS-over-TLS (DoT) support
- [ ] Machine learning-based filtering
- [ ] Mobile app for remote management
- [ ] Multi-user support with profiles
- [ ] Cloud sync for settings
- [ ] Advanced analytics dashboard
