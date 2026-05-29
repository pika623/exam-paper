$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$goCandidates = @(
  "D:\g\versions\1.25.8\bin\go.exe",
  "D:\g\versions\1.24.1\bin\go.exe",
  "go"
)

$goExe = $null
foreach ($candidate in $goCandidates) {
  if ($candidate -eq "go") {
    $cmd = Get-Command go -ErrorAction SilentlyContinue
    if ($cmd) {
      $goExe = $cmd.Source
      break
    }
  } elseif (Test-Path $candidate) {
    $goExe = $candidate
    break
  }
}

if (-not $goExe) {
  throw "Go executable was not found."
}

if ($goExe -like "D:\g\versions\*\bin\go.exe") {
  $env:GOROOT = Split-Path -Parent (Split-Path -Parent $goExe)
}

$env:GOCACHE = Join-Path $root ".gocache"
Set-Location $root
& $goExe build -o exam-paper.exe ./cmd
& (Join-Path $root "exam-paper.exe") --host 0.0.0.0 --port 16666

