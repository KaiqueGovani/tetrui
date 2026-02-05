Param(
  [string]$Repo = "KaiqueGovani/tetrui",
  [string]$Version = "nightly",
  [string]$Dest = "$env:LOCALAPPDATA\Programs\tetrui\tetrui.exe"
)

$ErrorActionPreference = "Stop"

$dir = Split-Path -Parent $Dest
New-Item -ItemType Directory -Force $dir | Out-Null

$url = "https://github.com/$Repo/releases/download/$Version/tetrui-windows-amd64.exe"
Invoke-WebRequest $url -OutFile $Dest

Write-Host "Installed tetrui to $Dest"
if ($env:Path -notmatch [regex]::Escape($dir)) {
  [Environment]::SetEnvironmentVariable("Path", $env:Path + ";" + $dir, "User")
  $env:Path = [Environment]::GetEnvironmentVariable("Path", "User")
}
Write-Host "Added $dir to PATH (user scope). Open a new terminal to use 'tetrui'."
