param([Parameter(Mandatory = $true)][string]$Configuration, [switch]$RequireInstalledMSIX)
$ErrorActionPreference = 'Stop'
$config = Get-Content -LiteralPath $Configuration -Raw | ConvertFrom-Json
if ($RequireInstalledMSIX -and $config.Scenario -ne 'msix') { throw 'Installed-MSIX configuration is required for the complete Windows native gate.' }
Set-Location -LiteralPath $config.Repository
# Start-Process with credentials still inherits the runner's environment.
# Replace profile values with the loaded standard user's native environment
# before any known-folder query. This changes only the owned CI test process.
Add-Type -TypeDefinition @'
using System;
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.Security.Principal;
public static class HEICStandardUserEnvironment {
 [DllImport("userenv.dll", SetLastError = true)]
 [return: MarshalAs(UnmanagedType.Bool)]
 static extern bool CreateEnvironmentBlock(out IntPtr block, IntPtr token, [MarshalAs(UnmanagedType.Bool)] bool inherit);
 [DllImport("userenv.dll", SetLastError = true)]
 [return: MarshalAs(UnmanagedType.Bool)]
 static extern bool DestroyEnvironmentBlock(IntPtr block);
 public static void Apply() {
  using (var identity = WindowsIdentity.GetCurrent()) {
   IntPtr block;
   if (!CreateEnvironmentBlock(out block, identity.Token, false)) throw new Win32Exception(Marshal.GetLastWin32Error());
   try {
    for (IntPtr cursor = block; Marshal.ReadInt16(cursor) != 0;) {
     string entry = Marshal.PtrToStringUni(cursor);
     cursor = IntPtr.Add(cursor, (entry.Length + 1) * 2);
     int separator = entry.IndexOf('=');
     if (separator > 0) Environment.SetEnvironmentVariable(entry.Substring(0, separator), entry.Substring(separator + 1));
    }
   } finally { if (!DestroyEnvironmentBlock(block)) throw new Win32Exception(Marshal.GetLastWin32Error()); }
  }
 }
}
'@
[HEICStandardUserEnvironment]::Apply()
$env:PATH = (Split-Path -Parent $config.Go) + ';' + $env:PATH
$profileDirectory = [Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)
if (-not $profileDirectory) { throw 'The standard user has no accessible local application-data folder.' }
Write-Output "Standard-user local application data: $profileDirectory"
$env:GOCACHE = Join-Path $config.Work 'go-cache'
$env:GOPATH = Join-Path $config.Work 'go-path'
$env:CGO_ENABLED = '0'
$env:TEMP = Join-Path $config.Work 'user-temp'
$env:TMP = $env:TEMP
New-Item -ItemType Directory -Path $env:TEMP -Force | Out-Null
& whoami /all
if ($LASTEXITCODE -ne 0) { throw 'Cannot record native execution identity.' }
if ($config.Scenario -eq 'standalone') {
    & $config.Go run ./scripts/nativeguards -suite heic-windows -capture (Join-Path $config.Evidence 'native-windows.json')
    if ($LASTEXITCODE -ne 0) { throw 'Native HEIC guards failed under the standard account.' }
    exit 0
}
$package = $null
try {
    Add-AppxPackage -Path $config.Package -DependencyPath $config.Dependency
    $package = Get-AppxPackage -Name $config.PackageName
    if (-not $package) { throw 'The test-MSIX was not installed for the standard user.' }
    $package | Format-List Name, PackageFullName, PackageFamilyName, InstallLocation, Architecture
    # Normal installed executable launch lets Windows resolve package identity.
    # The probe requires actual identity and the exact standard-user token;
    # running an unpackaged copy cannot satisfy the installed-MSIX guard.
    $start = [System.Diagnostics.ProcessStartInfo]::new()
    $start.FileName = Join-Path $package.InstallLocation 'heic-activation.test.exe'
    $start.Arguments = '-test.run=^TestNativeInstalledHEICActivation$ -test.v -test.timeout=4m'
    $start.WorkingDirectory = $config.Work
    $start.UseShellExecute = $false
    $process = [System.Diagnostics.Process]::Start($start)
    if (-not $process) { throw 'Windows did not start the installed test application.' }
    if (-not $process.WaitForExit(300000)) { $process.Kill(); throw 'Installed activation did not terminate within five minutes.' }
    $resultPath = Join-Path $config.Evidence 'installed-msix.json'
    if (-not (Test-Path -LiteralPath $resultPath)) { throw 'Installed activation did not produce its completion record.' }
    $result = Get-Content -LiteralPath $resultPath -Raw | ConvertFrom-Json
    Get-Content -LiteralPath (Join-Path $config.Evidence 'installed-msix.log')
    if ($process.ExitCode -ne 0 -or -not $result.Passed -or $result.Test -ne 'TestNativeInstalledHEICActivation' -or $result.OS -ne 'windows' -or $result.Arch -ne $config.Arch -or $result.Commit -ne $config.Commit) {
        throw 'Installed standard-user MSIX activation did not pass for this build.'
    }
} finally {
    if ($package) { Remove-AppxPackage -Package $package.PackageFullName }
}
