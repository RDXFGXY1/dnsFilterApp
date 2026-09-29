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
└── docs/                # Guides, feature documentation, and update notes
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
  upstream_dns:
    - "https://1.1.1.1/dns-query" # DNS-over-HTTPS
    - "tls://1.1.1.1:853"         # DNS-over-TLS

advanced:
  secure_dns_only: true
  upstream_timeout_seconds: 5

filtering:
  enabled: true
  block_categories:
    - adult
    - malware
    - gambling

database:
  path: "./data/dns-filter.db"
```

### Signed security packs

Security packs are verified with Ed25519 before installation and rechecked at
startup. Generate a local publisher key and sign a newline-separated domain
file with:

```bash
go run ./cmd/security-pack keygen -id local -out-dir ./data/security-pack-keys
go run ./cmd/security-pack sign -key-id local -private ./data/security-pack-keys/local.private -id local-threats -name "Local Threats" -category malware -domains domains.txt -out local-threats.json
```

Add the contents of `local.public` as a trusted publisher key in the Blocklists
screen, then verify and install `local-threats.json`. Never share the private key.

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

#### Automatic (recommended, cross-platform)

The binary can point your system's DNS at the filter and undo it. Run from an
elevated shell (Administrator on Windows, `sudo` on Linux/macOS):

```bash
# Point system DNS at the local filter (127.0.0.1)
./dns-filter setup-dns

# Restore automatic (DHCP) DNS
./dns-filter reset-dns
```

> On Windows PowerShell, call the binary by its path, e.g.
> `.\build\dns-filter.exe setup-dns` (the name `dns-filter` only works if the
> binary is on your PATH). You can also toggle this from the web dashboard's
> **Middleman** button once the server is running as Administrator.

It auto-detects the active network interface and uses the right mechanism per OS
(`netsh` on Windows, `resolvectl`/`resolv.conf` on Linux, `networksetup` on macOS).

#### Manual

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
The first launch opens `/setup` and requires creation of a local administrator
and Secure Vault. The dashboard authenticates with a cryptographically random,
HttpOnly, SameSite=Strict local session cookie. There is no shared default
password and this release does not claim to use JWT or Bearer-token API access.

Protected API calls must come from an authenticated local dashboard or desktop
session. First-run setup is accepted only through the computer's loopback
interface and closes after completion.

### Endpoints

- `GET /api/stats` - Get filtering statistics
- `GET /api/recent` - List recently blocked domains
- `POST /api/whitelist` - Add domain to whitelist
- `DELETE /api/whitelist/:domain` - Remove from whitelist
- `POST /api/blocklist/update` - Refresh configured blocklists
- `GET /api/system/info` - Read local service and DNS information

Feature and usage documentation: [docs/README.md](docs/README.md)

## Security Considerations

- Use the first-run wizard to create unique administrator and Vault passwords
- Save the two one-time recovery credentials separately and offline
- Keep the dashboard bound to loopback unless HTTPS and access controls are configured
- Regularly update blocklists
- Monitor logs for unusual activity

## Documentation

- [Documentation index](docs/README.md)
- [Feature documentation](docs/features/README.md)
- [Updates and release notes](docs/updates/README.md)

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) first.

## License

MIT License - see [LICENSE](LICENSE) file for details

## Support

- Documentation: [docs/README.md](docs/README.md)
- Issues: GitHub Issues
- Discussions: GitHub Discussions

If you find a bug, encounter unexpected behavior, or have a feature request, please open an issue on GitHub so we can track and address it:

- **Report an issue:** https://github.com/RDXFGXY1/dnsFilterApp/issues

When opening an issue, include as much of the following as possible:
- **Description:** what you expected vs what happened
- **Steps to reproduce:** exact commands or actions
- **Environment:** OS, Go version, installation method (setup.sh or manual)
- **Logs / error output:** relevant log lines or stack traces

This helps maintainers triage and fix problems faster. Thank you for contributing!

## Roadmap

- [x] DNS-over-HTTPS (DoH) support
- [x] DNS-over-TLS (DoT) support
- [ ] Machine learning-based filtering
- [ ] Mobile app for remote management
- [ ] Multi-user support with profiles
- [ ] Cloud sync for settings
- [ ] Advanced analytics dashboard
