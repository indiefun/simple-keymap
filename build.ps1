param([switch]$TestOnly)
$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath $PSScriptRoot
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
        & $taskGoExe build -trimpath -ldflags '-H=windowsgui' -o $taskStaging .
        if ($LASTEXITCODE -ne 0) { throw 'Build failed.' }
        Copy-Item -LiteralPath $taskStaging -Destination (Join-Path $taskDestination 'SimpleKeymap.Input.exe') -Force
        Copy-Item -LiteralPath $taskStaging -Destination $taskOutput -Force
    } finally {
        if (Test-Path -LiteralPath $taskStaging) { Remove-Item -LiteralPath $taskStaging }
    }
    Write-Host "Ready: $taskOutput"
}
