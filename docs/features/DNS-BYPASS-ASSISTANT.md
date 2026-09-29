# DNS bypass assistant

Open **Decision Center → DNS bypass assistant → Run DNS check**.

This read-only check uses the existing authenticated protection-health endpoint. It shows the selected adapter's configured DNS servers, filtering state, and historical query count. It changes no DNS, browser, VPN, or firewall settings.

External DNS configuration is a potential bypass risk, not proof that a particular browser used it. A local resolver address does not identify the process answering there. Browser DoH, alternate adapters, VPN DNS and application-specific resolvers remain explicitly unverified. Query counts are historical, not a browser-specific probe. Cached pages, existing connections and direct IP traffic are outside DNS filtering.

Unknown or failed checks must not be interpreted as protected. Follow the displayed manual checks and rerun after any network change. This version does not send test queries to a third-party leak-testing service.
