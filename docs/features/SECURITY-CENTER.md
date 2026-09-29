# Security Center

The Security Center provides one dashboard for the application's privacy and security features. Open **Security Center** from the main sidebar.

## What it shows

- Whether strict encrypted DNS is active and whether every upstream resolver uses DoH or DoT.
- The current protection-health score and access to domain decision explanations.
- Whether Emergency Lockdown is active and how many exceptions it has.
- The number of installed signed security packs and trusted publisher keys.
- Whether the configuration vault is configured, locked, or temporarily unlocked.
- A security-readiness score and a checklist of recommended actions.

The score is a local readiness indicator. It measures whether recommended application features are configured; it is not a guarantee that the device or network is free from threats.

## Actions

Each card links to the existing detailed controls. The Security Center does not duplicate sensitive actions such as enabling Lockdown, trusting a signing key, or configuring the vault. This keeps confirmation and recovery controls in their dedicated workspaces.

Use **Run security check** to refresh every status. You can also enter a domain and select **Explain** to open its full result in the Decision Center.
