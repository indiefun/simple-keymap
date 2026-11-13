param([switch]$TestOnly, [string]$Version = '')
$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath $PSScriptRoot
if (!$Version) { $Version = (Get-Content -LiteralPath 'VERSION' -Raw).Trim() }
$Version = $Version.Trim() -replace '^v', ''
if ($Version -notmatch '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$') {
    throw 'Version must use MAJOR.MINOR.PATCH, for example 1.0.0 or v1.0.0.'
}
$taskGo = Get-Command go -ErrorAction SilentlyContinue
if ($taskGo) {
    $taskGoExe = $taskGo.Source
} else {
    $taskGoExe = Join-Path $PSScriptRoot '.tools\go\bin\go.exe'
    if (!(Test-Path -LiteralPath $taskGoExe)) { throw 'Please install Go 1.23+ from https://go.dev/dl/ and run this script again.' }
}
& $taskGoExe test -timeout 30s ./...
if ($LASTEXITCODE -ne 0) { throw 'Tests failed.' }
& $taskGoExe vet ./...
if ($LASTEXITCODE -ne 0) { throw 'Go vet failed.' }
if (!$TestOnly) {
    # Only stop this project's packaged processes, before replacing either exe.
    $taskRootPrefix = [System.IO.Path]::GetFullPath($PSScriptRoot).TrimEnd('\') + '\'
    $taskRunning = Get-CimInstance Win32_Process -Filter "Name='SimpleKeymap.exe' OR Name='SimpleKeymap.Input.exe'" | Where-Object {
        $_.ExecutablePath -and $_.ExecutablePath.StartsWith($taskRootPrefix, [System.StringComparison]::OrdinalIgnoreCase)
    }
    foreach ($taskProcessInfo in $taskRunning) {
        $taskProcess = Get-Process -Id $taskProcessInfo.ProcessId -ErrorAction SilentlyContinue
        if ($taskProcess -and !$taskProcess.HasExited) {
            Write-Host ('Stopping running package process: ' + $taskProcess.Id)
            Stop-Process -Id $taskProcess.Id -Force
            if (!$taskProcess.WaitForExit(2000)) { throw 'Windows could not stop the input process. Restart Windows before rebuilding.' }
        }
    }
    New-Item -ItemType Directory -Path 'dist' -Force | Out-Null
    $taskOutput = Join-Path $PSScriptRoot 'dist\SimpleKeymap.exe'
    $taskDestination = (Resolve-Path -LiteralPath 'dist').Path
    $taskStaging = Join-Path $taskDestination ('.build-' + [guid]::NewGuid().ToString('N') + '.exe')
    try {
        & $taskGoExe build -trimpath -ldflags "-H=windowsgui -X=main.appVersion=$Version" -o $taskStaging .
        if ($LASTEXITCODE -ne 0) { throw 'Build failed.' }
        Copy-Item -LiteralPath $taskStaging -Destination (Join-Path $taskDestination 'SimpleKeymap.Input.exe') -Force
        Copy-Item -LiteralPath $taskStaging -Destination $taskOutput -Force
    } finally {
        if (Test-Path -LiteralPath $taskStaging) { Remove-Item -LiteralPath $taskStaging }
    }
    $taskReadme = (Get-Content -LiteralPath 'assets\release-readme.txt' -Raw -Encoding UTF8).Replace('{{VERSION}}', $Version)
    [System.IO.File]::WriteAllText((Join-Path $taskDestination 'README.txt'), $taskReadme, [System.Text.UTF8Encoding]::new($true))
    $taskArchive = Join-Path $taskDestination ("SimpleKeymap-v$Version-windows-x64.zip")
    # Compress-Archive requests exclusive access and can race with Windows'
    # executable scanner. Read with sharing allowed and publish only a complete ZIP.
    Add-Type -AssemblyName System.IO.Compression, System.IO.Compression.FileSystem
    $taskArchiveStaging = Join-Path $taskDestination ('.build-' + [guid]::NewGuid().ToString('N') + '.zip')
    try {
        $taskZip = [System.IO.Compression.ZipFile]::Open($taskArchiveStaging, [System.IO.Compression.ZipArchiveMode]::Create)
        try {
            foreach ($taskFile in @($taskOutput, (Join-Path $taskDestination 'SimpleKeymap.Input.exe'), (Join-Path $taskDestination 'README.txt'))) {
                $taskEntry = $taskZip.CreateEntry([System.IO.Path]::GetFileName($taskFile), [System.IO.Compression.CompressionLevel]::Optimal)
                $taskInput = [System.IO.File]::Open($taskFile, [System.IO.FileMode]::Open, [System.IO.FileAccess]::Read, ([System.IO.FileShare]::ReadWrite -bor [System.IO.FileShare]::Delete))
                try {
                    $taskEntryStream = $taskEntry.Open()
                    try { $taskInput.CopyTo($taskEntryStream) } finally { $taskEntryStream.Dispose() }
                } finally { $taskInput.Dispose() }
            }
        } finally { $taskZip.Dispose() }
        Move-Item -LiteralPath $taskArchiveStaging -Destination $taskArchive -Force
    } finally {
        if (Test-Path -LiteralPath $taskArchiveStaging) { Remove-Item -LiteralPath $taskArchiveStaging }
    }
    $taskChecksum = (Get-FileHash -Algorithm SHA256 -LiteralPath $taskArchive).Hash.ToLowerInvariant()
    [System.IO.File]::WriteAllText(($taskArchive + '.sha256'), ($taskChecksum + '  ' + [System.IO.Path]::GetFileName($taskArchive) + "`n"), [System.Text.Encoding]::ASCII)
    Write-Host "Ready: $taskOutput"
    Write-Host "Package: $taskArchive"
}
