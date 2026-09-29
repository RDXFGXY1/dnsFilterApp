<# Verifies a portable ZIP against its published checksum and manifest. #>

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Artifact,

    [string]$ChecksumFile,
    [string]$ManifestFile
)

$ErrorActionPreference = "Stop"
$artifactPath = (Resolve-Path -LiteralPath $Artifact).Path
$artifactItem = Get-Item -LiteralPath $artifactPath
if ([string]::IsNullOrWhiteSpace($ChecksumFile)) { $ChecksumFile = "$artifactPath.sha256" }
if ([string]::IsNullOrWhiteSpace($ManifestFile)) { $ManifestFile = "$artifactPath.manifest.json" }
$checksumPath = (Resolve-Path -LiteralPath $ChecksumFile).Path
$manifestPath = (Resolve-Path -LiteralPath $ManifestFile).Path

$checksumLine = (Get-Content -LiteralPath $checksumPath -Raw).Trim()
if ($checksumLine -notmatch '^([a-fA-F0-9]{64})\s{2}([^\\/]+)$') {
    throw "Checksum file has an invalid format."
}
$expectedHash = $Matches[1].ToLowerInvariant()
$expectedName = $Matches[2]
if ($expectedName -cne $artifactItem.Name) {
    throw "Checksum filename does not match the selected artifact."
}

$actualHash = (Get-FileHash -LiteralPath $artifactPath -Algorithm SHA256).Hash.ToLowerInvariant()
if ($actualHash -cne $expectedHash) {
    throw "SHA-256 verification failed. The artifact is damaged or has been modified."
}

if ($artifactItem.Name -notmatch '^dns-filter-portable-(.+)\.zip$') {
    throw "Artifact filename does not follow the versioned DNS Filter release format."
}
$expectedVersion = $Matches[1]

$manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
if ($manifest.schema_version -ne 1 -or
    $manifest.product -cne "DNS Filter" -or
    $manifest.artifact -cne $artifactItem.Name -or
    [int64]$manifest.size_bytes -ne $artifactItem.Length -or
    $manifest.sha256.ToLowerInvariant() -cne $actualHash -or
    $manifest.version -cne $expectedVersion -or
    $manifest.release_audit_passed -ne $true) {
    throw "Release manifest verification failed."
}

Write-Host "Release verified: $($artifactItem.Name)" -ForegroundColor Green
Write-Host "  Version: $($manifest.version)"
Write-Host "  SHA-256: $actualHash"
Write-Host "  Authenticode signed: $($manifest.authenticode_signed)"

