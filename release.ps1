param(
    [string]$Version = "dev",
    [string]$ZapretDir = ""
)
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

.\build.ps1 -Version $Version

$stage = Join-Path $env:TEMP ("deepmole-release-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force $stage | Out-Null
Copy-Item dist/deepmole.exe, dist/deepmole-cli.exe, README.md, LICENSE $stage
Copy-Item deepmole.example.json (Join-Path $stage "deepmole.json")

$lite = "dist/DeepMole-$Version-windows-x64.zip"
Remove-Item $lite -ErrorAction SilentlyContinue
Compress-Archive -Path "$stage\*" -DestinationPath $lite
Write-Host "packed: $lite"

if ($ZapretDir -and (Test-Path (Join-Path $ZapretDir "bin\winws.exe"))) {
    $z = Join-Path $PSScriptRoot "dist\zapret"
    Remove-Item $z -Recurse -Force -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Force (Join-Path $z "bin"), (Join-Path $z "lists") | Out-Null
    Copy-Item (Join-Path $ZapretDir "bin\*") (Join-Path $z "bin")
    Get-ChildItem (Join-Path $ZapretDir "lists") -File |
        Where-Object { $_.Name -notlike "*backup*" -and $_.Name -notlike "*-user.txt" } |
        Copy-Item -Destination (Join-Path $z "lists")
    $lic = Join-Path $ZapretDir "LICENSE.txt"
    if (Test-Path $lic) { Copy-Item $lic (Join-Path $z "LICENSE.txt") }

    Copy-Item $z (Join-Path $stage "zapret") -Recurse
    $full = "dist/DeepMole-$Version-full-windows-x64.zip"
    Remove-Item $full -ErrorAction SilentlyContinue
    Compress-Archive -Path "$stage\*" -DestinationPath $full
    Write-Host "packed: $full"

    $iscc = @(
        "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe",
        "$env:ProgramFiles\Inno Setup 6\ISCC.exe",
        "$env:LOCALAPPDATA\Programs\Inno Setup 6\ISCC.exe"
    ) | Where-Object { Test-Path $_ } | Select-Object -First 1
    if ($iscc) {
        & $iscc /Q "/DAppVersion=$Version" "/DZapretDir=$z" installer\deepmole.iss
        Write-Host "packed: dist/DeepMole-$Version-setup.exe"
    } else {
        Write-Host "Inno Setup not found, installer skipped"
    }
}

Remove-Item $stage -Recurse -Force
