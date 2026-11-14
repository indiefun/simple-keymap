param([string]$Version = '', [string]$Archive = '')
$ErrorActionPreference = 'Stop'
$taskProjectRoot = Split-Path -Parent $PSScriptRoot
if (!$Version) { $Version = (Get-Content -LiteralPath (Join-Path $taskProjectRoot 'VERSION') -Raw).Trim() }
$Version = $Version -replace '^v', ''
if ($Version -notmatch '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$') { throw 'Invalid package version.' }
if (!$Archive) { $Archive = Join-Path $taskProjectRoot "dist/SimpleKeymap-v$Version-windows-x64.zip" }
$taskPackagePath = (Resolve-Path -LiteralPath $Archive).Path
$taskExpectedHash = ((Get-Content -LiteralPath ($taskPackagePath + '.sha256') -Raw) -split ' ')[0]
if ((Get-FileHash -Algorithm SHA256 -LiteralPath $taskPackagePath).Hash.ToLowerInvariant() -ne $taskExpectedHash) { throw 'Package checksum mismatch.' }
Add-Type -AssemblyName System.IO.Compression, System.IO.Compression.FileSystem
$taskZip = [System.IO.Compression.ZipFile]::OpenRead($taskPackagePath)
try {
    $taskExpectedNames = @('SimpleKeymap.exe', 'SimpleKeymap.Input.exe', 'README.txt', 'LICENSE', 'NOTICE', 'TRADEMARKS.md')
    if (@(Compare-Object @($taskZip.Entries.FullName | Sort-Object) @($taskExpectedNames | Sort-Object)).Count) { throw 'Unexpected package contents.' }
    foreach ($taskName in $taskExpectedNames) {
        $taskStream = $taskZip.GetEntry($taskName).Open()
        try {
            $taskHasher = [System.Security.Cryptography.SHA256]::Create()
            try { $taskEntryHash = [BitConverter]::ToString($taskHasher.ComputeHash($taskStream)).Replace('-', '').ToLowerInvariant() }
            finally { $taskHasher.Dispose() }
        } finally { $taskStream.Dispose() }
        $taskLocalFile = if ($taskName -in @('LICENSE', 'NOTICE', 'TRADEMARKS.md')) { Join-Path $taskProjectRoot $taskName } else { Join-Path (Split-Path -Parent $taskPackagePath) $taskName }
        if ($taskEntryHash -ne (Get-FileHash -Algorithm SHA256 -LiteralPath $taskLocalFile).Hash.ToLowerInvariant()) { throw "Packaged file mismatch: $taskName" }
    }
    $taskReader = [System.IO.StreamReader]::new($taskZip.GetEntry('README.txt').Open(), [System.Text.Encoding]::UTF8)
    try { $taskReadmeText = $taskReader.ReadToEnd() } finally { $taskReader.Dispose() }
    if ($taskReadmeText.Contains('{{VERSION}}') -or !$taskReadmeText.Contains("/tree/v$Version") -or !$taskReadmeText.Contains("/archive/refs/tags/v$Version.zip")) { throw 'Missing corresponding source links in README.' }
    if (!$taskReadmeText.Contains('GPL-3.0-only')) { throw 'Package license notice missing.' }
} finally { $taskZip.Dispose() }
$taskDist = Split-Path -Parent $taskPackagePath
$taskVersionProcess = [System.Diagnostics.Process]::new()
try {
    $taskVersionProcess.StartInfo.FileName = Join-Path $taskDist 'SimpleKeymap.exe'
    $taskVersionProcess.StartInfo.Arguments = '--version'
    $taskVersionProcess.StartInfo.UseShellExecute = $false
    $taskVersionProcess.StartInfo.CreateNoWindow = $true
    $taskVersionProcess.StartInfo.RedirectStandardOutput = $true
    $taskVersionProcess.StartInfo.RedirectStandardError = $true
    if (!$taskVersionProcess.Start()) { throw 'Could not start version check.' }
    if (!$taskVersionProcess.WaitForExit(10000)) { $taskVersionProcess.Kill(); throw 'Version check timed out.' }
    if ($taskVersionProcess.ExitCode -ne 0 -or $taskVersionProcess.StandardOutput.ReadToEnd().Trim() -ne "Simple Keymap $Version") { throw 'Embedded version mismatch.' }
} finally { $taskVersionProcess.Dispose() }
Write-Host "Verified package $Version`: contents, legal notices, source links, embedded version and SHA256."
