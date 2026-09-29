# Emergency Lockdown

Emergency Lockdown temporarily blocks every DNS request except domains in its dedicated allowlist. It is useful when you suspect malware activity or need to quickly restrict network access without stopping DNS protection.

## Using Lockdown

Open the **Decision Center**, then:

1. Select a duration between one minute and 24 hours.
2. Add only the essential domains that must remain available.
3. Review the warning and start Lockdown.

Allowing `example.com` also allows its subdomains. Lockdown ends automatically when its timer expires and can be stopped manually at any time.

DNS Filter clears its DNS cache when Lockdown starts or stops. Blocked requests are logged with the reason `emergency_lockdown`.

## Priority

Emergency Lockdown has a higher priority than the filtering master switch and temporary filtering pauses. Add a domain to the Lockdown allowlist only when you trust it and genuinely need it during the emergency period.
