# Windows Code Signing

Official Windows packages must be Authenticode-signed with a trusted code-signing certificate. A self-signed development certificate does not provide public publisher trust and must not be used to claim that a public release is verified.

## Certificate handling

Install the certificate and its private key in the Windows `CurrentUser\\My` or `LocalMachine\\My` certificate store. The build accepts only its thumbprint; it never reads, copies, or exports the private key.

The certificate must:

- include the Code Signing enhanced key usage;
- have an accessible private key;
- be unexpired;
- chain to a certificate authority trusted by the target Windows machines.

## Official build

Run from PowerShell:

```powershell
.\scripts\build-portable.ps1 -RequireSigning -SigningThumbprint "CERTIFICATE_THUMBPRINT"
```

The thumbprint can instead be supplied through `DNS_FILTER_SIGNING_THUMBPRINT`. The build signs the Go service, the DNS Filter desktop executable and assembly, and the first-party update/uninstall PowerShell scripts. Vendor framework DLLs are not re-signed.

Each file uses SHA-256 and a timestamp server. After signing, every signature is read back and must be valid and match the requested certificate. The ZIP is created only after this verification succeeds. Missing, expired, inaccessible, mismatched, or invalid certificates stop the official build.

Without `-RequireSigning`, the script may create an explicitly unsigned development package. That mode is for local testing only.
