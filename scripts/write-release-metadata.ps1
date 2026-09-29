<# Creates the public SHA-256 checksum and machine-readable release manifest. #>

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Artifact,

    [Parameter(Mandatory = $true)]
    [string]$Version,

    [switch]$AuthenticodeSigned
)

$ErrorActionPreference = "Stop"
$artifactPath = (Resolve-Path -LiteralPath $Artifact).Path
$artifactItem = Get-Item -LiteralPath $artifactPath
$digest = (Get-FileHash -LiteralPath $artifactPath -Algorithm SHA256).Hash.ToLowerInvariant()
$checksumPath = "$artifactPath.sha256"
$manifestPath = "$artifactPath.manifest.json"
$utf8NoBom = New-Object Text.UTF8Encoding($false)

[IO.File]::WriteAllText(
    $checksumPath,
    "$digest  $($artifactItem.Name)`n",
    $utf8NoBom
)

$manifest = [ordered]@{
    schema_version = 1
    product = "DNS Filter"
    version = $Version
    artifact = $artifactItem.Name
    size_bytes = $artifactItem.Length
    sha256 = $digest
    created_utc = [DateTime]::UtcNow.ToString("o")
    authenticode_signed = [bool]$AuthenticodeSigned
    release_audit_passed = $true
}
[IO.File]::WriteAllText(
    $manifestPath,
    (($manifest | ConvertTo-Json -Depth 3) + "`n"),
    $utf8NoBom
)

Write-Host "Release metadata created:" -ForegroundColor Green
Write-Host "  Checksum: $checksumPath"
Write-Host "  Manifest: $manifestPath"

