# Windows desktop control

Run `build/DNS-Filter-Go-Freeze.exe` from this project. Keep the executable inside
the build folder: it uses the existing configs, web assets, and data in the parent
folder. This is a native Windows control window; the full dashboard opens in your
browser and retains its normal sign-in.

1. Click **Start server** and accept the Windows administrator prompt.
2. Wait for the dashboard port to become reachable.
3. Click **Enable local DNS** and accept the Windows prompt. The application checks
   that both local DNS resolvers answer successfully before changing IPv4 and IPv6 DNS.
4. Open the dashboard to manage filtering, settings, and activity.
5. Use **Restore automatic DNS (DHCP)** before manually stopping the server if you
   want to return to your network's automatic DNS configuration.

Closing the control window leaves the server running. Restore remains available
when the server is offline. The operation log reports unsuccessful or cancelled
actions; the logs button opens the server log directory.

## Limits

This version configures IPv4 and IPv6 DNS on the currently selected default-route adapter.
It does not enforce traffic through DNS or block browser secure-DNS
bypasses. Connection Freeze is a separate optional control described in
[its feature guide](connection-freeze.md). Automatic restoration means DHCP, not a
backup of previously configured static DNS addresses. The desktop controller does
not install a Windows service or provide crash recovery. Port reachability alone
does not prove that the process is DNS Filter; verify the dashboard after starting.

The controller is written in Go and uses native Win32 controls. It does not launch
PowerShell or require .NET or a separate GUI runtime. Status checks run in the
background, and Windows UAC handles permission requests for DNS setup. Child process
exit codes are checked before reporting success. The older Desktop and Desktop-fixed
executables are superseded by DNS-Filter-Go-Freeze.exe. Its Connection Freeze button
opens the authenticated dashboard controls and connection log.

## Build

```powershell
go build -trimpath -ldflags="-s -w" -o build/dns-filter-gui-server.exe ./cmd/server
go build -trimpath -ldflags="-H windowsgui -s -w" -o build/DNS-Filter-Go-Freeze.exe ./cmd/desktop
```
