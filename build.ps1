param([string]$Version = "dev")
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

go run ./tools/mkicon

Push-Location cmd/deepmole
go run github.com/tc-hib/go-winres@v0.3.3 make --in winres/winres.json --out rsrc
Pop-Location

$ld = "-s -w -X main.version=$Version"
New-Item -ItemType Directory -Force dist | Out-Null
go build -trimpath -ldflags "$ld -H windowsgui" -o dist/deepmole.exe ./cmd/deepmole
go build -trimpath -ldflags "$ld" -o dist/deepmole-cli.exe ./cmd/deepmole
Copy-Item deepmole.example.json dist/deepmole.json -Force
Write-Host "built: dist/deepmole.exe, dist/deepmole-cli.exe ($Version)"
