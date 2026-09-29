<#
  Authenticode-sign and verify DNS Filter-owned Windows release files.
  The private key stays in the Windows certificate store and is never exported.
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Path,

    [Parameter(Mandatory = $true)]
    [string]$Thumbprint,

    [string]$TimestampServer = "http://timestamp.digicert.com"
)

$ErrorActionPreference = "Stop"
$releaseRoot = (Resolve-Path -LiteralPath $Path).Path
$normalizedThumbprint = ($Thumbprint -replace '\s', '').ToUpperInvariant()

$certificate = @(
    Get-ChildItem Cert:\CurrentUser\My, Cert:\LocalMachine\My -CodeSigningCert -ErrorAction SilentlyContinue |
        Where-Object { $_.Thumbprint.ToUpperInvariant() -eq $normalizedThumbprint }
)

if ($certificate.Count -ne 1) {
    throw "Exactly one code-signing certificate with the requested thumbprint must exist in CurrentUser or LocalMachine certificate storage."
}

$certificate = $certificate[0]
if (-not $certificate.HasPrivateKey) {
    throw "The selected code-signing certificate does not have an accessible private key."
}
if ($certificate.NotAfter -le (Get-Date)) {
    throw "The selected code-signing certificate has expired."
}

$ownedFiles = @(
    (Join-Path $releaseRoot "dns-filter.exe"),
    (Join-Path $releaseRoot "desktop\DNSFilter.exe"),
    (Join-Path $releaseRoot "desktop\DNSFilter.dll"),
    (Join-Path $releaseRoot "update-windows.ps1"),
    (Join-Path $releaseRoot "uninstall-windows.ps1")
) | Where-Object { Test-Path -LiteralPath $_ -PathType Leaf }

if ($ownedFiles.Count -lt 3) {
    throw "Expected DNS Filter release binaries were not found under $releaseRoot."
}

foreach ($file in $ownedFiles) {
    Write-Host "Signing $([IO.Path]::GetFileName($file))..." -ForegroundColor Cyan
    $signature = Set-AuthenticodeSignature -LiteralPath $file -Certificate $certificate -HashAlgorithm SHA256 -TimestampServer $TimestampServer
    if ($signature.Status -ne [Management.Automation.SignatureStatus]::Valid) {
        throw "Authenticode signing failed for $([IO.Path]::GetFileName($file)): $($signature.Status)"
    }
}

foreach ($file in $ownedFiles) {
    $signature = Get-AuthenticodeSignature -LiteralPath $file
    if ($signature.Status -ne [Management.Automation.SignatureStatus]::Valid -or
        $signature.SignerCertificate.Thumbprint.ToUpperInvariant() -ne $normalizedThumbprint) {
        throw "Authenticode verification failed for $([IO.Path]::GetFileName($file))."
    }
}

Write-Host "Authenticode signing verified for $($ownedFiles.Count) DNS Filter release file(s)." -ForegroundColor Green
