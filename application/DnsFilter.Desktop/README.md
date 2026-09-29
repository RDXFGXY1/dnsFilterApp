# DNS Filter Windows desktop app

This WPF/.NET 8 client keeps the existing Go service as the security engine. The first release provides local login, health and DNS status, filtering controls, cache clearing, list updates, manual Connection Freeze, system-DNS setup/reset, and a link to advanced web controls.

The client connects internally to `http://127.0.0.1:8080` and opens the browser dashboard at `http://dnsfilter.localhost:8080`. Login uses the server's existing HttpOnly session cookie held only in this process. The password is not saved.

When the service is unavailable, the app searches nearby parent/build directories for `dns-filter-gui-server.exe` and starts it hidden. Keep the published desktop folder inside the repository's `build` directory so discovery works.

Build:

```powershell
dotnet build -c Release
```

Publish a self-contained Windows x64 build:

```powershell
dotnet publish -c Release -r win-x64 --self-contained true -p:PublishSingleFile=true
```
