# Encrypted DNS

DNS Filter supports **DNS-over-HTTPS (DoH)** and **DNS-over-TLS (DoT)**. Encrypted DNS prevents the local network or internet provider from easily reading domain names while DNS requests travel to the upstream resolver.

Encrypted DNS is not a VPN. It does not hide your IP address, application traffic, or the websites you connect to.

## Configuration

Configure secure upstream resolvers in `configs/config.yaml`:

```yaml
server:
  upstream_dns:
    - https://1.1.1.1/dns-query
    - tls://1.1.1.1:853

advanced:
  secure_dns_only: true
  upstream_timeout_seconds: 5
```

- An address beginning with `https://` is treated as DoH.
- An address beginning with `tls://` is treated as DoT.
- When `secure_dns_only` is enabled, DNS Filter never silently falls back to unencrypted DNS.
- If the first resolver fails, the application tries the next configured secure resolver.

## Security behavior

Connections require TLS 1.2 or newer. DoH redirects are disabled to prevent a trusted endpoint from silently redirecting requests elsewhere. DNS Filter also validates the response size, message ID, and original DNS question.

If every secure resolver fails, the request fails closed instead of being leaked through ordinary DNS.
