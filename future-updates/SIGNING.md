# Code Signing — Future Implementation Plan

> **Status:** Deferred. Implement after cross-platform compilation is stable and binaries are being distributed to customers.

---

## Overview

Without code signing, each platform will warn on first run:
- **Windows** — SmartScreen: "Windows protected your PC" (bypassable by IT admins)
- **macOS** — Gatekeeper: blocks execution entirely on macOS 10.15+ (requires manual override or MDM)
- **Linux** — no OS-level signing enforcement, but package managers can verify GPG signatures on `.deb`/`.rpm`

---

## Windows

### Certificate options

| Type | Cost | SmartScreen | Delivery |
|------|------|-------------|----------|
| OV (standard) | ~$100–200/yr | Warns until reputation builds (~days/weeks) | `.pfx` file, immediate |
| EV (extended validation) | ~$300–500/yr | Instant trust, no warning | Hardware USB token, 1–5 days |

Recommended CAs: Sectigo (cheapest), DigiCert (best support).

For enterprise B2B where IT deploys via Intune/SCCM/GPO, **OV is sufficient** — SmartScreen doesn't trigger in managed deployments.

### Signing command
```bash
signtool sign \
  /tr http://timestamp.digicert.com \
  /td sha256 \
  /fd sha256 \
  /f signing.pfx \
  /p "$SIGNING_PASSWORD" \
  dist/bestdefense-device-monitor.exe
```

### CI integration (GitHub Actions)
Store the `.pfx` as a base64-encoded GitHub Actions secret:
```yaml
- name: Sign Windows binary
  if: startsWith(github.ref, 'refs/tags/')
  env:
    SIGNING_CERT_BASE64: ${{ secrets.WINDOWS_SIGNING_CERT }}
    SIGNING_PASSWORD: ${{ secrets.WINDOWS_SIGNING_PASSWORD }}
  run: |
    echo "$SIGNING_CERT_BASE64" | base64 -d > signing.pfx
    signtool sign /tr http://timestamp.digicert.com /td sha256 /fd sha256 \
      /f signing.pfx /p "$SIGNING_PASSWORD" \
      dist/bestdefense-device-monitor.exe
    rm signing.pfx
```

Run on `windows-latest` runner (signtool ships with Windows SDK, pre-installed on GitHub-hosted Windows runners).

---

## macOS

### Requirements
- **Apple Developer ID Application certificate** — $99/yr via Apple Developer Program (developer.apple.com)
- **Notarization** — required for macOS 13+ Ventura and above; free but mandatory; submitted via `xcrun notarytool`
- Must run on a `macos-latest` GitHub Actions runner

### Process
1. Export Developer ID certificate from Keychain as `.p12`
2. Store `.p12` + password as GitHub Actions secrets
3. Sign all binary architectures:
```bash
codesign --deep --force --verify --verbose \
  --sign "Developer ID Application: BestDefense Inc (TEAMID)" \
  --options runtime \
  dist/bestdefense-device-monitor-darwin-amd64

codesign --deep --force --verify --verbose \
  --sign "Developer ID Application: BestDefense Inc (TEAMID)" \
  --options runtime \
  dist/bestdefense-device-monitor-darwin-arm64
```
4. Notarize each binary (or the DMG):
```bash
xcrun notarytool submit bestdefense-device-monitor.dmg \
  --apple-id "$APPLE_ID" \
  --password "$APPLE_APP_PASSWORD" \
  --team-id "$APPLE_TEAM_ID" \
  --wait
```
5. Staple the notarization ticket to the DMG:
```bash
xcrun stapler staple bestdefense-device-monitor.dmg
```

### CI integration
```yaml
- name: Sign and notarize macOS binaries
  if: startsWith(github.ref, 'refs/tags/')
  runs-on: macos-latest
  env:
    APPLE_CERT_BASE64: ${{ secrets.MACOS_SIGNING_CERT }}
    APPLE_CERT_PASSWORD: ${{ secrets.MACOS_SIGNING_PASSWORD }}
    APPLE_ID: ${{ secrets.APPLE_ID }}
    APPLE_APP_PASSWORD: ${{ secrets.APPLE_APP_SPECIFIC_PASSWORD }}
    APPLE_TEAM_ID: ${{ secrets.APPLE_TEAM_ID }}
  run: |
    echo "$APPLE_CERT_BASE64" | base64 -d > signing.p12
    security create-keychain -p "" build.keychain
    security import signing.p12 -k build.keychain -P "$APPLE_CERT_PASSWORD" -T /usr/bin/codesign
    security set-key-partition-list -S apple-tool:,apple: -s -k "" build.keychain
    security default-keychain -s build.keychain
    # sign, package DMG, notarize, staple...
```

---

## Linux — GPG Package Signing

Linux doesn't block unsigned binaries, but `.deb` and `.rpm` package managers verify GPG signatures. Unsigned packages generate warnings during `apt install` / `dnf install`.

### GPG key setup
1. Generate a GPG key for BestDefense (done once, stored offline):
```bash
gpg --full-generate-key  # RSA 4096, no expiry, key ID: releases@bestdefense.io
gpg --export --armor releases@bestdefense.io > bestdefense-releases.gpg
```
2. Publish public key to a key server and host at `https://app.bestdefense.io/releases.gpg`
3. Store the private key + passphrase as GitHub Actions secrets

### Signing .deb packages
```bash
dpkg-sig --sign builder \
  --gpg-options "--passphrase $GPG_PASSPHRASE" \
  dist/bestdefense-device-monitor_amd64.deb
```

### Signing .rpm packages
```bash
rpmsign --addsign \
  --define "_gpg_name releases@bestdefense.io" \
  dist/bestdefense-device-monitor.x86_64.rpm
```

### Customer setup (one-time)
```bash
# Debian/Ubuntu
curl -fsSL https://app.bestdefense.io/releases.gpg | sudo gpg --dearmor -o /usr/share/keyrings/bestdefense.gpg

# RHEL/Fedora
sudo rpm --import https://app.bestdefense.io/releases.gpg
```

---

## GitHub Actions Secrets Required (all platforms)

| Secret | Platform | Purpose |
|--------|----------|---------|
| `WINDOWS_SIGNING_CERT` | Windows | Base64-encoded `.pfx` file |
| `WINDOWS_SIGNING_PASSWORD` | Windows | PFX password |
| `MACOS_SIGNING_CERT` | macOS | Base64-encoded `.p12` file |
| `MACOS_SIGNING_PASSWORD` | macOS | P12 password |
| `APPLE_ID` | macOS | Apple Developer account email |
| `APPLE_APP_SPECIFIC_PASSWORD` | macOS | App-specific password for notarytool |
| `APPLE_TEAM_ID` | macOS | 10-character Apple Team ID |
| `GPG_PRIVATE_KEY` | Linux | Armored GPG private key |
| `GPG_PASSPHRASE` | Linux | GPG key passphrase |

---

## Recommended rollout order

1. **Windows OV cert** — cheapest, lowest friction, covers the largest current install base
2. **Linux GPG signing** — free once GPG key is generated; important for enterprise package managers
3. **macOS Developer ID + notarization** — required before any macOS distribution; $99/yr
