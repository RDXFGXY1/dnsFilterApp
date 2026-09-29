# Local dashboard address

Use `http://dnsfilter.localhost:8080/` (or your configured API port). The native Go desktop launcher opens this address and displays it alongside the IP fallback during startup. The console setup also displays both addresses.

In the dashboard, open **Settings → System & Security → Dashboard address** for both links. The fallback remains `http://127.0.0.1:8080/`.

This change does not alter network bindings, DNS filtering, firewall settings, the hosts file, or the server port. It does not publish the dashboard online or add HTTPS. Supported browsers resolve `.localhost` locally; if your browser cannot, use the IP fallback. Internal console API requests retain IP addressing.

The new hostname has a separate browser origin: sign in again and select your theme there if necessary. Keep the port in the URL; port-free access is not configured.
