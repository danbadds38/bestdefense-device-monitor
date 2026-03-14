# BestDefense Device Monitor

A lightweight Windows security compliance agent — open source for full transparency.

Collects security configuration data from employee machines and reports to your BestDefense backend. Similar in scope to Vanta's device agent.

## What it checks

| Check | What it verifies |
|-------|-----------------|
| **BitLocker** | Drive encryption status per volume |
| **Antivirus** | Windows Defender status + definition freshness |
| **Firewall** | All three Windows Firewall profiles (domain/private/public) |
| **Screen lock** | Screensaver timeout and password-on-resume |
| **Windows Update** | Auto-update configuration and last successful update |
| **Installed apps** | Application inventory from registry |
| **Local users** | Account list, admin membership, disabled accounts |
| **Password policy** | Complexity, length, age, lockout settings |
| **Disk encryption** | BitLocker status per drive |
| **Hardware/OS** | Device identity, OS version, hardware specs |
| **Network** | Interface info (IP + MAC only — no traffic) |

See [docs/DATA_COLLECTED.md](docs/DATA_COLLECTED.md) for a complete field-by-field breakdown.

## Audit before you deploy

Run this on any Windows machine to see exactly what would be sent — no installation required:

```
bestdefense-device-monitor.exe check
```

This prints the full JSON payload to stdout. Review it, inspect the source code, then decide whether to deploy.

## Installation

Requires **Windows 10 or 11**, **Administrator privileges**.

```powershell
# Install and start the monitoring service
.\bestdefense-device-monitor.exe install --key YOUR_REGISTRATION_KEY
```

The registration key is your BestDefense customer ID, available in your BestDefense dashboard.

After installation:
- The service runs as **LocalSystem** and starts automatically on boot
- It checks in every **4 hours** (configurable)
- Config stored at: `C:\ProgramData\BestDefense\config.json`
- Logs at: `C:\ProgramData\BestDefense\logs\agent.log`
- Windows Event Log source: `BestDefenseMonitor`

## Uninstall

```powershell
.\bestdefense-device-monitor.exe uninstall
```

This removes the service. Config and logs are preserved at `C:\ProgramData\BestDefense\` — delete that folder manually if desired.

## CLI Reference

```
install --key <key>   Install and start the service (requires elevation)
uninstall             Stop and remove the service (requires elevation)
check [--send]        One-shot check — prints JSON to stdout; add --send to also transmit
status                Show service status and config
version               Show version and build info
```

## Configuration

Edit `C:\ProgramData\BestDefense\config.json` to change defaults:

```json
{
  "registration_key": "cust_abc123xyz",
  "api_endpoint": "https://app.bestdefense.io/monitoring/employee/update",
  "check_interval_hours": 4,
  "log_level": "info"
}
```

Restart the service after editing: `net stop BestDefenseMonitor && net start BestDefenseMonitor`

## Enterprise deployment (Intune / SCCM / GPO)

1. Copy `bestdefense-device-monitor.exe` to a shared location or deploy via your MDM
2. Run the install command with your registration key via a deployment script:
   ```cmd
   bestdefense-device-monitor.exe install --key cust_abc123xyz
   ```
3. The service persists across reboots automatically

## Building from source

**Requirements:** Go 1.22+, Windows (or cross-compile from Linux/macOS)

```powershell
# Windows
.\scripts\build.ps1 -Version "1.0.0"

# Linux/macOS (cross-compile)
./scripts/build.sh 1.0.0
```

Output: `dist\bestdefense-device-monitor.exe`

## What we do NOT collect

- Browser history, passwords, or cookies
- File contents
- Keystrokes or clipboard
- Email content
- Screenshots
- Network traffic or packet data
- Location data

## Security considerations

- The binary makes **outbound HTTPS only** to `app.bestdefense.io`
- No inbound ports are opened
- Source code is fully auditable — this repo is the complete implementation
- The installer adds a Windows Firewall outbound rule for the binary
- Without a code signing certificate, Windows SmartScreen may warn on first run — this is expected for unsigned binaries and can be bypassed by IT administrators

## License

MIT — see [LICENSE](LICENSE)
