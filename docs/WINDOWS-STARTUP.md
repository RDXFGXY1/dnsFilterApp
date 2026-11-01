# Windows startup and DNS availability

In **Settings → System & Security → Windows Startup**, choose:

- **Before Windows sign-in**: run in the background as SYSTEM at boot, with a sign-in trigger as an additional retry. No interactive session, Windows account password, or open terminal is required.
- **After Windows sign-in**: run as SYSTEM when any Windows user signs in. This does not start the main server at boot.
- **Manual start only**: remove main-server automatic startup. Saving does not stop a currently running server.

The second setting, **Keep Internet available if DNS Filter stops**, installs the independent `DNSFilterRecovery` Windows task. It runs at boot and once a minute, including while the main application is stopped. It only restores automatic DNS when the active adapter uses exclusively loopback DNS and a local resolver fails two probes. It leaves a resolver that answers (including refusal or SERVFAIL) alone. Adapter/DNS changes made during probing are respected.

Recovery is not instant: there can be about a minute plus probe/startup time without working DNS. During fallback, traffic is not DNS-filtered. DNS Route Watch restores the local route when the server returns, provided **local DNS routing** is enabled. This is DNS recovery, not a firewall reset; separately configured application/network firewall restrictions are left intact.

A selected automatic mode retries unexpected process failures every minute, ignores duplicate task instances, continues on battery, and has no task execution time limit. Startup diagnostics show the configured executable, working folder, and last Windows task result. Saving settings repairs the scheduled-task definition.

The previous development layout search stopped at `build/`. A server under `build/dev/` could therefore start from the wrong folder and fail to find configuration. The search now finds the repository root. A manual `start -NoBuild` command does not install a startup task and does not persist by itself across reboot.

From an Administrator terminal, after building the updated executable:

```powershell
# Run from the repository root. Registers both boot startup and DNS recovery.
.\build\dev\dns-filter-dev.exe install-autostart boot
Start-ScheduledTask -TaskName DNSFilter

# Alternatively, start only after Windows sign-in.
.\build\dev\dns-filter-dev.exe install-autostart login

# Remove both automatic startup and independent recovery.
.\build\dev\dns-filter-dev.exe uninstall-autostart
```

For installed releases the executable is `dns-filter.exe` in the installation folder. Installation requires Administrator permission; the app's configuration-vault protection also applies to dashboard changes. No changes are made to Secure Boot, TPM, Windows Hello, or Windows sign-in credentials.

The separate recovery command, `dns-filter.exe recover-dns`, is designed for the scheduled recovery task. It does not change the saved `manage_system_dns` preference. Emergency recovery and uninstallation also disable/remove the new recovery task.

Validation includes nested development paths, separate login/boot triggers, recovery permissions, manual mode, DNS fallback/healthy-resolver decisions, concurrent route changes, error propagation, API input rejection, and template parsing. A live scheduled-task start can be verified without rebooting, but this does not by itself prove the timing of a future real Windows boot.

## Windows ICS port conflict

The live startup test identified Internet Connection Sharing (`SharedAccess`) occupying `0.0.0.0:53`. DNS Filter now reserves its sockets before starting either listener and, on Windows only, falls back from an unavailable wildcard listener to an explicit loopback listener. The SYSTEM task was verified serving on both `127.0.0.1:53` and `[::1]:53` while ICS remained running. In this fallback mode the filter serves only this PC, not LAN DNS clients. Explicit non-wildcard configurations are not silently changed, and non-Windows binding behavior is preserved.

The boot task and independent recovery task were installed and the server was successfully started through Task Scheduler as SYSTEM with the correct repository working folder. The running executable's hash matched the newly built executable. A real reboot/sign-in cycle has not been performed.

The development runner now recognizes scheduled-task PIDs after reboot and uses the installed SYSTEM task for start/restart rather than launching an interactive duplicate that can conflict with ICS. Emergency recovery and uninstall scripts remove or disable both managed tasks.
