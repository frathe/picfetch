# Microsoft Store release

PicFetch uses the MSIX submission reserved in Partner Center under Store ID
`9P0DM0KTH01K`. Store packages are separate from the signed ZIP downloads:
Microsoft signs the MSIX bundle after certification, installs it under the
protected package location, and owns its updates.

## Build the bundle

The `Microsoft Store package` workflow runs automatically for every `v*` tag
and can also be started manually from GitHub Actions. It runs the full CI gate,
cross-builds x64 and ARM64 binaries with the `microsoftstore` build tag, stages
both packages, validates them with MakeAppx, creates and test-signs one bundle,
and runs the Windows App Certification Kit.

Download the `picfetch-microsoft-store` workflow artifact. It contains:

- `picfetch-microsoft-store.msixbundle`, the file to upload to Partner Center;
- `wack-report.xml`, the certification-kit result to inspect before upload.

Do not publish the bundle as a GitHub Release or direct download. Its temporary
CI signature is trusted only on the runner; Partner Center replaces it with a
Microsoft signature after certification.

## Versioning

MSIX requires a four-part numeric version with a non-zero first component, and
the Store reserves the fourth component. Starting with PicFetch 1.0.0, the
three-part public `Version` in `FyneApp.toml` is therefore also the Store
version, with a trailing zero added for MSIX: for example, PicFetch `1.0.0`
becomes `1.0.0.0` and PicFetch `1.0.1` becomes `1.0.1.0`. The separate Fyne
`Build` counter remains internal packaging metadata and does not affect the
public version in either release channel.

Store packaging deliberately rejects versions below 1.0.0, prerelease labels,
and components outside MSIX's 0..65535 range. The first Store submission must
therefore be built from the `v1.0.0` release tag, not from an older `v0.*` tag.

## Partner Center

1. Under Properties, select that PicFetch accesses or transmits personal
   information and set the privacy-policy URL to
   `https://github.com/frathe/picfetch/blob/main/PRIVACY.md`. The declaration
   covers the user-initiated OpenStreetMap requests documented in that policy.
2. Upload the bundle under Submission -> Packages and leave only Windows 10/11
   Desktop enabled.
3. Explain the detected `runFullTrust` restricted capability in Submission
   options: PicFetch is a native Fyne/Win32 desktop image viewer. It needs normal
   desktop process access to open user-selected files and folders, save/export
   images, copy files and pixels, move selected files to the Recycle Bin, and set
   wallpaper. It does not elevate, install services, or run background tasks.
4. Add English and German listings using
   `packaging/microsoft-store/listing.md` and its referenced assets.
5. Review the complete submission and its visibility/release timing before
   selecting Submit for certification.

The package identity values in `scripts/msixstage` are case-sensitive and come
from Partner Center. If the reserved product changes, update them together with
this document and the plan before building another bundle.
