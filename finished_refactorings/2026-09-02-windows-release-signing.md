# Windows release signing

## Problem

The tag-release workflow currently publishes Windows ZIP files built by
build-cross without a code signature. A Certum SimplySign cloud certificate is
available and must sign every published Windows executable without storing its
private key in the repository.

## Decisions

| Decision | Choice |
|---|---|
| Signing point | A dedicated Windows job after build-cross, before release. |
| Credential boundary | Protected GitHub environment release-signing. |
| Build artifact boundary | Separate unsigned Windows, signed Windows, and Linux artifacts. |
| Signing policy | SHA-256 file digest, Certum RFC-3161 timestamp, then SignTool verification. |
| Publication policy | Release downloads signed Windows artifacts only. |
| SimplySign card | Must be pinless because CI cannot answer the interactive card-PIN prompt. |
| Installer trust | Explicit Certum 9.4.4.92 URL plus Authenticode verification before installation. |

## Acceptance criteria

1. Tag releases cannot publish Windows archives until the protected signing job
   succeeds.
2. Both Windows architectures are signed and verified before being repackaged.
3. Certum credentials are referenced only as GitHub Environment secrets.
4. Linux and macOS artifacts retain their current release behavior.
5. Repository documentation describes the required secrets and approval setup.

## Non-goals

- Configuring the real GitHub Environment or entering the user’s secrets.
- Signing development builds, pull requests, macOS applications, or Linux
  binaries.
- Replacing Certum SimplySign with another certificate provider.

## Verification

- Parse the release workflow as YAML.
- Confirm the final release job downloads picfetch-windows-signed and never
  picfetch-windows-unsigned.
- Confirm both ZIP names are enforced in the signing job before publication.
