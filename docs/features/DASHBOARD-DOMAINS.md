# Dashboard Domains and LAN access

Open **System → Domains** in the dashboard to configure a local hostname that
you own, such as `guard.example.org`. The application adds a local hosts entry
and serves the dashboard at that name over HTTPS. Changing the name signs out
the current dashboard session; sign in again at the new address.

The same page can enable access from specific devices on the private LAN. Set
the dashboard hostname and add each device's exact private IP address to the
allowlist first. Public IP addresses and IP ranges are not accepted. Keep LAN
access off when it is unnecessary, and protect the administrator account and
Vault credentials.

The public-Internet access option is a placeholder. The draft release does not
provide remote Internet access. For local-only use, keep the default loopback
dashboard address `http://127.0.0.1:8080`.
