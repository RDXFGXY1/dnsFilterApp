<#
  Fails a release when it contains local runtime state, secrets, private keys,
  debug symbols, or developer-specific paths. Findings report rule names and
  file paths only; secret values are never printed.
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Path
)

$ErrorActionPreference = "Stop"
$target = (Resolve-Path -LiteralPath $Path).Path
$temporaryRoot = $null
$findings = [System.Collections.Generic.List[string]]::new()

function Add-Finding([string]$Rule, [string]$RelativePath) {
    $findings.Add("[$Rule] $RelativePath")
}

function Get-RelativeReleasePath([string]$Root, [string]$FilePath) {
    $rootPath = [IO.Path]::GetFullPath($Root).TrimEnd('\', '/') + [IO.Path]::DirectorySeparatorChar
    $fullPath = [IO.Path]::GetFullPath($FilePath)
    if (-not $fullPath.StartsWith($rootPath, [StringComparison]::OrdinalIgnoreCase)) {
        throw "File is outside the release root."
    }
    return $fullPath.Substring($rootPath.Length).Replace('\', '/')
}

try {
    if ([IO.Path]::GetExtension($target) -ieq ".zip") {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        $temporaryRoot = Join-Path ([IO.Path]::GetTempPath()) ("dns-filter-release-audit-" + [guid]::NewGuid().ToString("N"))
        New-Item -ItemType Directory -Path $temporaryRoot | Out-Null

        $archive = [IO.Compression.ZipFile]::OpenRead($target)
        try {
            foreach ($entry in $archive.Entries) {
                $entryPath = $entry.FullName.Replace('\', '/')
                if ($entryPath.StartsWith('/') -or $entryPath -match '^[A-Za-z]:' -or $entryPath.Split('/') -contains '..') {
                    Add-Finding "unsafe-archive-path" $entryPath
                }
            }
        }
        finally {
            $archive.Dispose()
        }
        if ($findings.Count -eq 0) {
            [IO.Compression.ZipFile]::ExtractToDirectory($target, $temporaryRoot)
        }
        $scanRoot = $temporaryRoot
    }
    elseif (Test-Path -LiteralPath $target -PathType Container) {
        $scanRoot = $target
    }
    else {
        throw "Release path must be a directory or .zip archive: $target"
    }

    if ($findings.Count -eq 0) {
        $files = @(Get-ChildItem -LiteralPath $scanRoot -Recurse -Force -File)
        foreach ($file in $files) {
            $relative = Get-RelativeReleasePath $scanRoot $file.FullName
            $lower = $relative.ToLowerInvariant()

            if ($lower -match '(^|/)(data/(logs|vault|vault-archive|quarantine|security-packs)/|backups/)' -or
                $lower -match '\.(db|db-journal|db-shm|db-wal|log|key|pem|pfx|p12|pdb)$' -or
                $lower -match '(^|/)(\.env($|\.)|\.git/)' -or
                $lower -match '\.(ya?ml)\.bak$') {
                Add-Finding "runtime-or-private-file" $relative
                continue
            }

            $bytes = [IO.File]::ReadAllBytes($file.FullName)
            $content = [Text.Encoding]::ASCII.GetString($bytes)
            $rules = @(
                @{ Name = "private-key-material"; Pattern = '-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----' },
                @{ Name = "developer-path"; Pattern = '(?i)C:\\Users\\ayoub(?:\\|/)' }
            )
            foreach ($rule in $rules) {
                if ($content -match $rule.Pattern) {
                    Add-Finding $rule.Name $relative
                }
            }

            # Token-shaped byte sequences can occur by chance in compressed or
            # compiled files. Scan only formats that can intentionally carry a
            # copied credential; binaries are still checked for key markers and
            # developer paths above.
            if ($file.Extension -match '(?i)^\.(txt|md|json|ya?ml|xml|config|ini|env|js|css|html?|ps1|bat|cmd)$' -and
                $content -match '(?i)(github_pat_[A-Za-z0-9_]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|sk-(proj-)?[A-Za-z0-9_-]{20,})') {
                Add-Finding "credential-token" $relative
            }

            if ($lower -eq "configs/config.yaml") {
                if ($content -notmatch '(?m)^\s*setup_complete:\s*false\s*$' -or
                    $content -notmatch '(?m)^\s*jwt_secret:\s*(""|''\s*'')\s*$' -or
                    $content -notmatch '(?m)^\s*recovery_key_hash:\s*(""|''\s*'')\s*$') {
                    Add-Finding "unsafe-default-config" $relative
                }
            }
        }
    }

    if ($findings.Count -gt 0) {
        Write-Host "Release audit FAILED ($($findings.Count) finding(s)):" -ForegroundColor Red
        $findings | Sort-Object -Unique | ForEach-Object { Write-Host "  $_" }
        exit 1
    }

    Write-Host "Release audit passed: no runtime state, private material, known tokens, debug symbols, or developer paths found." -ForegroundColor Green
}
finally {
    if ($temporaryRoot -and (Test-Path -LiteralPath $temporaryRoot)) {
        $resolvedTemp = (Resolve-Path -LiteralPath $temporaryRoot).Path
        $systemTemp = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
        if ($resolvedTemp.StartsWith($systemTemp, [StringComparison]::OrdinalIgnoreCase) -and
            [IO.Path]::GetFileName($resolvedTemp).StartsWith("dns-filter-release-audit-")) {
            Remove-Item -LiteralPath $resolvedTemp -Recurse -Force
        }
    }
}
