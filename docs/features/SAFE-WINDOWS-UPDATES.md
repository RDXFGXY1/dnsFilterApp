# Safe Windows Installation and Updates

The portable Windows package includes `Install or Update DNS Filter.bat` and `update-windows.ps1`. Run the batch file as Administrator from an extracted release folder.

## Preserved state

Updates never replace `configs` or `data`. This preserves the administrator account, recovery hash, filtering configuration, SQLite database, Secure Vault, audit chain, logs, certificates, quarantine records, and local security state.

## Update transaction

1. Validate that the package contains the server and complete web assets.
2. Copy immutable files into a temporary staging directory.
3. Acquire an exclusive update lock and stop only the process running from the installed executable path.
4. Move the previous executable, web assets, and desktop files into a timestamped `.rollback` directory.
5. Move the staged release into the installation directory.
6. Start the new service and poll its public `/healthz` identity endpoint. The reported product and exact version must match the package `VERSION` file, preventing another process on the same port from producing a false success.
7. If the health check fails, stop the failed version, restore the previous files, and restart the previous service automatically.

The updater writes operational messages to `data/logs/update.log`. Rollback copies are intentionally retained until a later cleanup policy is applied.

## Package validation

Validation does not require Administrator rights and changes nothing:

```powershell
.\update-windows.ps1 -PackageDir . -ValidateOnly
```

Use `-InstallDir` or `-HealthUrl` when the installation uses a non-default location or dashboard port.
