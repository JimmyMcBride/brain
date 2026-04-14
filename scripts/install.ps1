param(
    [string]$Version = $env:BRAIN_VERSION,
    [string]$InstallDir = $(if ($env:BRAIN_INSTALL_DIR) { $env:BRAIN_INSTALL_DIR } elseif ($env:LOCALAPPDATA) { Join-Path $env:LOCALAPPDATA "Programs\brain" } else { Join-Path $HOME "AppData\Local\Programs\brain" })
)

$ErrorActionPreference = "Stop"

$Owner = "JimmyMcBride"
$Repo = "brain"
$ApiBase = "https://api.github.com"
$ReleaseBase = "https://github.com/$Owner/$Repo/releases/download"
$SourceBase = "https://codeload.github.com/$Owner/$Repo/zip/refs/heads"

function Fail([string]$Message) {
    throw "brain install: $Message"
}

function Get-Arch {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
    switch ($arch) {
        "x64" { return "amd64" }
        "arm64" { return "arm64" }
        default { Fail "unsupported architecture: $arch" }
    }
}

function Get-LatestVersion {
    try {
        $response = Invoke-RestMethod -Uri "$ApiBase/repos/$Owner/$Repo/releases/latest" -Headers @{ "User-Agent" = "brain-installer" }
        return $response.tag_name
    } catch {
        return ""
    }
}

function Get-Checksum([string]$ChecksumsPath, [string]$AssetName) {
    foreach ($line in Get-Content -Path $ChecksumsPath) {
        if ([string]::IsNullOrWhiteSpace($line)) {
            continue
        }
        $parts = $line -split '\s+'
        if ($parts.Length -ge 2 -and $parts[-1] -eq $AssetName) {
            return $parts[0]
        }
    }
    Fail "checksum entry missing for $AssetName"
}

function Get-GlobalSkillPath([string]$Agent) {
    switch ($Agent) {
        "codex" { return Join-Path $HOME ".codex\skills\brain" }
        "claude" { return Join-Path $HOME ".claude\skills\brain" }
        "copilot" { return Join-Path $HOME ".copilot\skills\brain" }
        "openclaw" { return Join-Path $HOME ".openclaw\skills\brain" }
        "pi" { return Join-Path $HOME ".pi\agent\skills\brain" }
        "ai" { return Join-Path $HOME ".ai\skills\brain" }
        default { return $null }
    }
}

function Refresh-GlobalSkills([string]$BinaryPath) {
    $agents = @()
    foreach ($candidate in @("codex", "claude", "copilot", "openclaw", "pi", "ai")) {
        $path = Get-GlobalSkillPath -Agent $candidate
        if ($path -and (Test-Path $path)) {
            $agents += $candidate
        }
    }
    if ($agents.Count -eq 0) {
        return
    }

    $arguments = @("skills", "install", "--scope", "global")
    foreach ($agent in $agents) {
        $arguments += @("--agent", $agent)
    }
    & $BinaryPath @arguments | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Fail "refresh existing global skills failed"
    }
}

function Install-FromSourceMain([string]$TempDir, [string]$InstallDir) {
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Fail "no published release found and Go is not installed"
    }
    $sourceArchive = Join-Path $TempDir "$Repo-main.zip"
    Invoke-WebRequest -Uri "$SourceBase/main" -OutFile $sourceArchive
    Expand-Archive -LiteralPath $sourceArchive -DestinationPath $TempDir -Force

    $sourceDir = Get-ChildItem -Path $TempDir -Directory | Where-Object { $_.Name -like "$Repo-*" } | Select-Object -First 1
    if (-not $sourceDir) {
        Fail "could not unpack source archive"
    }

    $binaryPath = Join-Path $TempDir "brain.exe"
    Push-Location $sourceDir.FullName
    try {
        & go build -o $binaryPath .
    } finally {
        Pop-Location
    }
    if (-not (Test-Path $binaryPath)) {
        Fail "source build did not produce brain.exe"
    }

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item -Path $binaryPath -Destination (Join-Path $InstallDir "brain.exe") -Force

    Write-Host "Installed to $InstallDir\brain.exe by building the current main branch source"
    Refresh-GlobalSkills -BinaryPath (Join-Path $InstallDir "brain.exe")
}

$arch = Get-Arch
if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = Get-LatestVersion
}

$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("brain-install-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $tempDir | Out-Null

try {
    if ([string]::IsNullOrWhiteSpace($Version)) {
        Install-FromSourceMain -TempDir $tempDir -InstallDir $InstallDir
        exit 0
    }

    $archive = "brain_${Version}_windows_${arch}.zip"
    $checksums = "brain_${Version}_checksums.txt"
    $archivePath = Join-Path $tempDir $archive
    $checksumsPath = Join-Path $tempDir $checksums

    Write-Host "Installing brain $Version for windows/$arch"
    Invoke-WebRequest -Uri "$ReleaseBase/$Version/$archive" -OutFile $archivePath
    Invoke-WebRequest -Uri "$ReleaseBase/$Version/$checksums" -OutFile $checksumsPath

    $expected = Get-Checksum -ChecksumsPath $checksumsPath -AssetName $archive
    $actual = (Get-FileHash -Algorithm SHA256 -Path $archivePath).Hash.ToLowerInvariant()
    if ($expected.ToLowerInvariant() -ne $actual) {
        Fail "checksum mismatch for $archive"
    }

    Expand-Archive -LiteralPath $archivePath -DestinationPath $tempDir -Force
    $binaryPath = Join-Path $tempDir "brain.exe"
    if (-not (Test-Path $binaryPath)) {
        Fail "archive did not contain brain.exe"
    }

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item -Path $binaryPath -Destination (Join-Path $InstallDir "brain.exe") -Force

    Write-Host "Installed to $InstallDir\brain.exe"
    Refresh-GlobalSkills -BinaryPath (Join-Path $InstallDir "brain.exe")
    $pathEntries = ($env:Path -split ';') | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
    if ($pathEntries -notcontains $InstallDir) {
        Write-Host "PATH note: ensure $InstallDir is on PATH"
    }
} finally {
    if (Test-Path $tempDir) {
        Remove-Item -Recurse -Force $tempDir
    }
}
