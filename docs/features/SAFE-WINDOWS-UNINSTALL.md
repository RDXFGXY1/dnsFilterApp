# Safe Windows Uninstall

Run `Uninstall DNS Filter.bat` from the installed directory as Administrator. The default operation removes the application while preserving `configs` and `data` for recovery or a later reinstall.

The uninstaller stops only executables launched from the selected installation directory, restores DNS on active default-route adapters to automatic configuration, flushes the DNS cache, removes the exact `DNSFilter` scheduled task and known service names, and deletes only firewall rules owned by DNS Filter. The local root certificate is removed only when its thumbprint matches the installation's own `data/ca/ca.crt` file.

Connection Freeze uses dynamic Windows Filtering Platform objects and disappears when the service exits. Persistent application, schedule, quarantine, and DNS-bypass firewall rules are removed by their owned name prefixes.

## Preview without changes

```powershell
.\uninstall-windows.ps1 -InstallDir "C:\Program Files\DNS Filter" -PlanOnly
```

## Permanently remove local data

This is intentionally separate and irreversible:

```powershell
.\uninstall-windows.ps1 `
  -InstallDir "C:\Program Files\DNS Filter" `
  -RemoveUserData `
  -DataRemovalConfirmation "DELETE LOCAL DNS FILTER DATA"
```

Without both data-removal arguments, account credentials, Vault state, databases, logs, certificates, quarantine metadata, and configuration remain on disk.
