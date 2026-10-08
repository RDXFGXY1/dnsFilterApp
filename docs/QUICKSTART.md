# Quick start

The official targets are Windows 11 x64 and Arch/Arch-based Linux. There is no
default account: the first launch asks you to create an administrator and Secure
Vault and displays two separate one-time recovery credentials.

## Windows

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build-portable.ps1
```

Run `dist\dns-filter-portable\Install or Update DNS Filter.bat` as Administrator,
then open the Avalonia desktop client or `http://127.0.0.1:8080`.

Development mode:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\dev-server.ps1
```

## Arch Linux

```bash
chmod +x scripts/build-arch.sh
./scripts/build-arch.sh x64
cd dist
tar -xzf dns-filter-arch-x64.tar.gz
cd dns-filter-linux-x64
sudo ./install-arch.sh
sudo /opt/dns-filter/arch-diagnostics.sh
```

Open DNS Filter from the application menu or visit `http://127.0.0.1:8080`.
Use [ARCH-TEST-CHECKLIST.md](ARCH-TEST-CHECKLIST.md) for the first native-network
test.

Full instructions: [../INSTALL-WINDOWS-LINUX.md](../INSTALL-WINDOWS-LINUX.md).
