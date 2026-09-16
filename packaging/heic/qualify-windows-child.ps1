param([Parameter(Mandatory = $true)][string]$Configuration, [switch]$RequireInstalledMSIX)
$ErrorActionPreference = 'Stop'
$config = Get-Content -LiteralPath $Configuration -Raw | ConvertFrom-Json
if ($RequireInstalledMSIX -and $config.Scenario -ne 'msix') { throw 'Installed-MSIX configuration is required for the complete Windows native gate.' }
Set-Location -LiteralPath $config.Repository
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
    Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
[ComImport, Guid("2E941141-7F97-4756-BA1D-9DECDE894A3D"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IHEICApplicationActivationManager {
 [PreserveSig] int ActivateApplication([MarshalAs(UnmanagedType.LPWStr)] string app, [MarshalAs(UnmanagedType.LPWStr)] string args, uint options, out uint processId);
}
public static class HEICInstalledActivation {
 public static uint Start(string app) {
  var type = Type.GetTypeFromCLSID(new Guid("45BA127D-10A8-46EA-8AB7-56EA9078943C"));
  var manager = (IHEICApplicationActivationManager)Activator.CreateInstance(type);
  uint processId;
  int result = manager.ActivateApplication(app, "-test.run=^TestNativeInstalledHEICActivation$ -test.v -test.timeout=4m", 0, out processId);
  Marshal.ThrowExceptionForHR(result);
  return processId;
 }
}
'@
    $activationPID = [HEICInstalledActivation]::Start($package.PackageFamilyName + '!HEICQualification')
    $process = [System.Diagnostics.Process]::GetProcessById($activationPID)
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
