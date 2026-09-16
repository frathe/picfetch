# Provisions disposable CI-only account/package state. Application tests run
# under that standard account; none of this provisioning is in PicFetch.
param(
    [ValidateSet('standalone', 'msix')][string]$Scenario = 'standalone',
    [Parameter(Mandatory = $true)][string]$EvidenceDirectory,
    [string]$Executable,
    [string]$RuntimeArchive
)
$ErrorActionPreference = 'Stop'
if ($env:GITHUB_ACTIONS -ne 'true' -or $env:RUNNER_ENVIRONMENT -ne 'github-hosted') {
    throw 'Disposable-account provisioning is restricted to GitHub-hosted CI. Run nativeguards directly as a standard user for local standalone qualification.'
}
$repository = (Get-Location).Path
$go = (Get-Command go).Source
$architecture = (& $go env GOARCH).Trim()
if ($architecture -notin @('amd64', 'arm64')) { throw 'A native amd64 or arm64 host is required.' }
$identity = [Guid]::NewGuid().ToString('N')
$userName = 'pfheic' + $identity.Substring(0, 10)
$workRoot = Join-Path $env:RUNNER_TEMP ('heic-owned-' + $identity)
$copy = Join-Path $workRoot 'repository'
$evidence = Join-Path $workRoot 'evidence'
$certificate = $null
$trusted = $null
$account = $null
New-Item -ItemType Directory -Path $workRoot, $evidence -Force | Out-Null
New-Item -ItemType Directory -Path $EvidenceDirectory -Force | Out-Null
try {
    $secret = ConvertTo-SecureString ('Pf!' + [Guid]::NewGuid().ToString('N') + '9a') -AsPlainText -Force
    $account = New-LocalUser -Name $userName -Password $secret -Description 'Disposable PicFetch HEIC qualification account'
    Add-LocalGroupMember -SID 'S-1-5-32-545' -Member $account
    $credential = [PSCredential]::new("$env:COMPUTERNAME\$userName", $secret)
    & icacls $workRoot /grant "*$($account.SID.Value):(OI)(CI)M" | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Could not prepare the owned qualification workspace.' }
    & robocopy $repository $copy /E /XJ /XD .git .scratch bin fyne-cross /NFL /NDL /NJH /NJS | Out-Null
    if ($LASTEXITCODE -ge 8) { throw 'Could not copy the source into the owned qualification workspace.' }
    $config = @{
        Scenario = $Scenario; Repository = $copy; Go = $go; Work = $workRoot
        Evidence = $evidence; Arch = $architecture; Commit = $env:GITHUB_SHA
    }
    if ($Scenario -eq 'msix') {
        if (-not $Executable -or -not $RuntimeArchive) { throw 'MSIX qualification requires the built Store executable and pinned runtime archive.' }
        $stage = Join-Path $workRoot 'stage'
        & $go run -tags no_emoji,nodynamic ./scripts/msixstage -arch $architecture -exe $Executable -runtime-archive $RuntimeArchive -out $stage
        if ($LASTEXITCODE -ne 0) { throw 'MSIX staging failed.' }
        & $go run ./scripts/heicpackage -os windows -arch $architecture -out $stage
        if ($LASTEXITCODE -ne 0) { throw 'HEIC helper staging failed.' }
        & $go test -tags no_emoji,nodynamic,heicnative,microsoftstore -c -o (Join-Path $stage 'heic-activation.test.exe') ./internal/ui
        if ($LASTEXITCODE -ne 0) { throw 'Application activation probe build failed.' }
        Copy-Item -LiteralPath (Join-Path $repository 'scripts/heicbuild/testdata/tenbit.heic') -Destination (Join-Path $stage 'heic-activation-fixture.heic')
        @{ Evidence = (Join-Path $evidence 'installed-msix'); Commit = $env:GITHUB_SHA; UserSID = $account.SID.Value } |
            ConvertTo-Json | Set-Content -LiteralPath (Join-Path $stage 'heic-activation.json') -Encoding utf8NoBOM
        $manifestPath = Join-Path $stage 'AppxManifest.xml'
        [xml]$manifest = Get-Content -LiteralPath $manifestPath -Raw
        $publisher = 'CN=PicFetch HEIC Qualification ' + $identity
        $packageName = 'PicFetch.HEIC.' + $identity
        $manifest.Package.Identity.SetAttribute('Name', $packageName)
        $manifest.Package.Identity.SetAttribute('Publisher', $publisher)
        $probe = $manifest.Package.Applications.Application.CloneNode($true)
        $probe.SetAttribute('Id', 'HEICQualification')
        $probe.SetAttribute('Executable', 'heic-activation.test.exe')
        $probe.RemoveChild($probe.Extensions) | Out-Null
        $manifest.Package.Applications.AppendChild($probe) | Out-Null
        $manifest.Save($manifestPath)
        $sdk = 'C:\Program Files (x86)\Windows Kits\10\bin'
        $makeappx = Get-ChildItem -LiteralPath $sdk -Filter MakeAppx.exe -Recurse | Where-Object FullName -Match '\\x64\\MakeAppx\.exe$' | Sort-Object FullName -Descending | Select-Object -First 1
        $signtool = Get-ChildItem -LiteralPath $sdk -Filter SignTool.exe -Recurse | Where-Object FullName -Match '\\x64\\SignTool\.exe$' | Sort-Object FullName -Descending | Select-Object -First 1
        if (-not $makeappx -or -not $signtool) { throw 'The Windows SDK packaging/signing tools are required.' }
        $package = Join-Path $workRoot 'qualification.msix'
        & $makeappx.FullName pack /o /h SHA256 /d $stage /p $package
        if ($LASTEXITCODE -ne 0) { throw 'Test-MSIX packing failed.' }
        $certificate = New-SelfSignedCertificate -Type Custom -Subject $publisher -KeyUsage DigitalSignature -CertStoreLocation 'Cert:\CurrentUser\My' -TextExtension @('2.5.29.37={text}1.3.6.1.5.5.7.3.3')
        $trusted = [System.Security.Cryptography.X509Certificates.X509Store]::new('TrustedPeople', 'LocalMachine')
        $trusted.Open('ReadWrite')
        $trusted.Add($certificate)
        & $signtool.FullName sign /fd SHA256 /sha1 $certificate.Thumbprint $package
        if ($LASTEXITCODE -ne 0) { throw 'Disposable package signing failed.' }
        & $signtool.FullName verify /pa /all /v $package
        if ($LASTEXITCODE -ne 0) { throw 'Disposable package verification failed.' }
        $frameworkArch = if ($architecture -eq 'amd64') { 'x64' } else { 'arm64' }
        $frameworkRoot = 'C:\Program Files (x86)\Microsoft SDKs\Windows Kits\10\ExtensionSDKs\Microsoft.VCLibs.Desktop\14.0'
        $framework = Get-ChildItem -LiteralPath $frameworkRoot -Filter "*$frameworkArch*.appx" -Recurse | Where-Object FullName -Match '\\Retail\\' | Sort-Object FullName -Descending | Select-Object -First 1
        if (-not $framework) { throw 'The architecture-matched Desktop C++ framework is required.' }
        $dependency = Join-Path $workRoot $framework.Name
        Copy-Item -LiteralPath $framework.FullName -Destination $dependency
        $config.Package = $package; $config.PackageName = $packageName; $config.Dependency = $dependency
        @{
            Commit = $env:GITHUB_SHA; Arch = $architecture; StandardUserSID = $account.SID.Value
            Package = $packageName; SHA256 = (Get-FileHash -LiteralPath $package -Algorithm SHA256).Hash
            TestCertificate = $certificate.Thumbprint
            Helper = (Get-Content -LiteralPath (Join-Path $stage 'heic/manifest.json') -Raw | ConvertFrom-Json)
        } | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $evidence 'package-setup.json') -Encoding utf8NoBOM
    }
    $configPath = Join-Path $workRoot 'configuration.json'
    $config | ConvertTo-Json | Set-Content -LiteralPath $configPath -Encoding utf8NoBOM
    $child = Join-Path $copy 'packaging/heic/qualify-windows-child.ps1'
    $powershell = (Get-Command pwsh).Source
    $process = Start-Process -FilePath $powershell -Credential $credential -LoadUserProfile -WorkingDirectory $copy -ArgumentList @('-NoProfile', '-File', "`"$child`"", '-Configuration', "`"$configPath`"") -PassThru -Wait -RedirectStandardOutput (Join-Path $evidence 'standard-user.stdout.log') -RedirectStandardError (Join-Path $evidence 'standard-user.stderr.log')
    Get-Content -LiteralPath (Join-Path $evidence 'standard-user.stdout.log')
    Get-Content -LiteralPath (Join-Path $evidence 'standard-user.stderr.log')
    if ($process.ExitCode -ne 0) { throw "Standard-user qualification failed with exit $($process.ExitCode)." }
} finally {
    try {
        Copy-Item -Path (Join-Path $evidence '*') -Destination $EvidenceDirectory -Recurse -Force
    } finally {
        try {
            if ($trusted) {
                try { if ($certificate) { $trusted.Remove($certificate) } } finally { $trusted.Close() }
            }
            if ($certificate) { Remove-Item -LiteralPath ("Cert:\CurrentUser\My\" + $certificate.Thumbprint) }
        } finally {
            try {
                if ($account) {
                    try {
                        Get-CimInstance -ClassName Win32_UserProfile -Filter "SID='$($account.SID.Value)'" | Remove-CimInstance
                    } finally { Remove-LocalUser -SID $account.SID }
                }
            } finally { Remove-Item -LiteralPath $workRoot -Recurse -Force }
        }
    }
}
