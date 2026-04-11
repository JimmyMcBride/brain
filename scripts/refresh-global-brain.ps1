param(
    [string]$InstallDir = $(if ($env:BRAIN_INSTALL_DIR) { $env:BRAIN_INSTALL_DIR } elseif ($env:LOCALAPPDATA) { Join-Path $env:LOCALAPPDATA "Programs\brain" } else { Join-Path $HOME "AppData\Local\Programs\brain" })
)

$ErrorActionPreference = "Stop"

function Fail([string]$Message) {
    throw "brain refresh: $Message"
}

function Test-DirectoryEqual([string]$Left, [string]$Right) {
    $leftFiles = Get-ChildItem -Path $Left -Recurse -File | ForEach-Object {
        [PSCustomObject]@{
            Relative = $_.FullName.Substring($Left.Length).TrimStart('\','/')
            Hash = (Get-FileHash -Algorithm SHA256 -Path $_.FullName).Hash
        }
    } | Sort-Object Relative
    $rightFiles = Get-ChildItem -Path $Right -Recurse -File | ForEach-Object {
        [PSCustomObject]@{
            Relative = $_.FullName.Substring($Right.Length).TrimStart('\','/')
            Hash = (Get-FileHash -Algorithm SHA256 -Path $_.FullName).Hash
        }
    } | Sort-Object Relative

    if ($leftFiles.Count -ne $rightFiles.Count) {
        return $false
    }
    for ($i = 0; $i -lt $leftFiles.Count; $i++) {
        if ($leftFiles[$i].Relative -ne $rightFiles[$i].Relative -or $leftFiles[$i].Hash -ne $rightFiles[$i].Hash) {
            return $false
        }
    }
    return $true
}

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent $scriptDir
$binPath = Join-Path $InstallDir "brain.exe"
$globalSkillPath = Join-Path $HOME ".codex\skills\brain"
$repoSkillPath = Join-Path $repoRoot "skills\brain"

if (-not (Test-Path (Join-Path $repoRoot ".git"))) {
    Fail "repo root does not look like a git checkout: $repoRoot"
}
if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Fail "need git"
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Fail "need go"
}
if (-not (Test-Path $repoSkillPath)) {
    Fail "missing repo skill source: $repoSkillPath"
}

$commit = (& git -C $repoRoot rev-parse HEAD).Trim()
$date = (& git -C $repoRoot show -s --format=%cI HEAD).Trim()

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
& go build -C $repoRoot -ldflags "-X brain/internal/buildinfo.Commit=$commit -X brain/internal/buildinfo.Date=$date" -o $binPath .

if (-not (Test-Path $binPath)) {
    Fail "build did not produce $binPath"
}

$versionOutput = & $binPath version
if ($versionOutput -notmatch [regex]::Escape("commit:  $commit")) {
    Fail "installed binary commit does not match $commit"
}

& $binPath skills install --scope global --agent codex --skill brain --mode copy --project $repoRoot | Out-Null

if (-not (Test-Path $globalSkillPath)) {
    Fail "global Codex brain skill was not installed"
}
if (-not (Test-DirectoryEqual -Left $repoSkillPath -Right $globalSkillPath)) {
    Fail "global Codex brain skill does not match repo copy"
}

Write-Host "Refreshed global brain"
Write-Host "  binary: $binPath"
Write-Host "  commit: $commit"
Write-Host "  skill:  $globalSkillPath"
