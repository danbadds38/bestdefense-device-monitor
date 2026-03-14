<p align="center">
  <img src="https://cdn.prod.website-files.com/6851d460e044fd271d0e1790/688420a8bf4b436c842282ef_mainlogo-p-500.png" alt="BestDefense.io Logo" width="400">
</p>

<p align="center">
  <strong>Providing Tomorrow's Cyber tools. Today</strong>
</p>

<p align="center">
  <a href="https://bestdefense.io"><img src="https://img.shields.io/badge/Website-bestdefense.io-blue"></a>
  <a href="mailto:contact@bestdefense.io"><img src="https://img.shields.io/badge/Contact-Email-green"></a>
  <a href="#"><img src="https://img.shields.io/badge/Status-TechStars_2025-purple"></a>
  <a href="#"><img src="https://img.shields.io/badge/Platform-AWS-orange"></a>
</p>

# BestDefense Device Monitor

A lightweight cross-platform security compliance agent — open source for full transparency.

Collects security configuration data from employee machines (Windows, macOS, Linux) and reports to your BestDefense backend. Similar in scope to Vanta's device agent.

## What it checks

| Check | Windows | macOS | Linux |
|-------|---------|-------|-------|
| **Disk encryption** | BitLocker status per volume | FileVault status | LUKS dm-crypt detection |
| **Antivirus** | Windows Defender + definition freshness | XProtect version + known AV apps | Known AV service detection |
| **Firewall** | Domain/private/public profiles | Application firewall | ufw / firewalld / iptables |
| **Screen lock** | Screensaver + password-on-resume | Screensaver idle time | GNOME/KDE/X11 settings |
| **Software update** | Windows Update auto-install config | SoftwareUpdate preferences | apt/dnf unattended upgrades |
| **Installed apps** | Registry uninstall keys | `/Applications` via system_profiler | dpkg / rpm package list |
| **Local users** | NetUserEnum + admin group | dscl + dseditgroup | /etc/passwd + sudo/wheel |
| **Password policy** | NetUserModalsGet + LSA | pwpolicy global hash table | /etc/login.defs + PAM pwquality |
| **Hardware/OS** | WMI Win32_* | system_profiler + sw_vers | /proc/cpuinfo + /etc/os-release |
| **Network** | net.Interfaces() | net.Interfaces() | net.Interfaces() |
| **System health** | WMI LastBootUpTime | sysctl kern.boottime | /proc/uptime |

See [docs/DATA_COLLECTED.md](docs/DATA_COLLECTED.md) for a complete field-by-field breakdown.

## Audit before you deploy

Run this to see the exact JSON payload that would be sent — no installation required:

```sh
# Windows
.\bestdefense-device-monitor-windows-amd64.exe check

# macOS
./bestdefense-device-monitor-darwin-arm64 check

# Linux
./bestdefense-device-monitor-linux-amd64 check
```

This prints the full JSON payload to stdout. Review it, inspect the source code, then decide whether to deploy.

## Installation

### Windows

Requires **Windows 10 or 11**, **Administrator privileges**.

```powershell
.\bestdefense-device-monitor-windows-amd64.exe install --key YOUR_REGISTRATION_KEY
```

After installation:
- Service runs as **LocalSystem**, starts automatically on boot
- Checks in every **4 hours** (configurable)
- Config: `C:\ProgramData\BestDefense\config.json`
- Logs: `C:\ProgramData\BestDefense\logs\agent.log`
- Windows Event Log source: `BestDefenseMonitor`

### macOS

Requires **root** (installs as a launchd daemon). macOS 11 Big Sur or later.

```sh
# Apple Silicon
chmod +x bestdefense-device-monitor-darwin-arm64
sudo ./bestdefense-device-monitor-darwin-arm64 install --key YOUR_REGISTRATION_KEY

# Intel
chmod +x bestdefense-device-monitor-darwin-amd64
sudo ./bestdefense-device-monitor-darwin-amd64 install --key YOUR_REGISTRATION_KEY
```

After installation:
- Launchd daemon at `/Library/LaunchDaemons/io.bestdefense.monitor.plist`
- Runs as root, starts at boot, auto-restarts on failure
- Config: `/Library/Application Support/BestDefense/config.json`
- Logs: `/Library/Application Support/BestDefense/logs/agent.log`

> **Note on Gatekeeper:** Without a Developer ID certificate, macOS will block execution. IT admins can allow it via MDM (Intune/Jamf) or with `spctl --add`. See [future-updates/SIGNING.md](future-updates/SIGNING.md) for the signing plan.

### Linux — Debian/Ubuntu (.deb)

```sh
sudo dpkg -i bestdefense-device-monitor_amd64.deb
sudo bestdefense-device-monitor install --key YOUR_REGISTRATION_KEY
```

### Linux — RHEL/Fedora/CentOS (.rpm)

```sh
sudo rpm -i bestdefense-device-monitor.x86_64.rpm
sudo bestdefense-device-monitor install --key YOUR_REGISTRATION_KEY
```

### Linux — binary (any distro)

```sh
chmod +x bestdefense-device-monitor-linux-amd64
sudo ./bestdefense-device-monitor-linux-amd64 install --key YOUR_REGISTRATION_KEY
```

After installation:
- systemd service: `bestdefense-monitor.service`
- Runs as root, enabled and started at boot, `Restart=always`
- Config: `/var/lib/bestdefense/config.json`
- Logs: `/var/lib/bestdefense/logs/agent.log` + journald

## Uninstall

```sh
# Windows (Admin PowerShell)
.\bestdefense-device-monitor-windows-amd64.exe uninstall

# macOS / Linux (root)
sudo ./bestdefense-device-monitor-<platform> uninstall
```

Config and logs are preserved. Delete the config directory manually if desired.

## CLI Reference

```
install --key <key>   Install and start the service (requires elevation/root)
uninstall             Stop and remove the service (requires elevation/root)
check [--send]        One-shot check — prints JSON to stdout; add --send to also transmit
status                Show service status and config
version               Show version and build info
```

## Configuration

Edit the platform config file to change defaults:

```json
{
  "registration_key": "cust_abc123xyz",
  "api_endpoint": "https://app.bestdefense.io/monitoring/employee/update",
  "check_interval_hours": 4,
  "log_level": "info"
}
```

| Platform | Config path |
|----------|-------------|
| Windows | `C:\ProgramData\BestDefense\config.json` |
| macOS | `/Library/Application Support/BestDefense/config.json` |
| Linux | `/var/lib/bestdefense/config.json` |

Restart the service after editing:
- Windows: `net stop BestDefenseMonitor && net start BestDefenseMonitor`
- macOS: `sudo launchctl kickstart -k system/io.bestdefense.monitor`
- Linux: `sudo systemctl restart bestdefense-monitor`

## Enterprise deployment

### Windows — Intune / SCCM / GPO

1. Deploy `bestdefense-device-monitor-windows-amd64.exe` via your MDM
2. Run as SYSTEM: `bestdefense-device-monitor-windows-amd64.exe install --key cust_abc123xyz`

### macOS — Jamf / MDM

1. Deploy the binary + a pre-configured `config.json` via your MDM
2. Run the install command as a policy script: `sudo ./bestdefense-device-monitor-darwin-<arch> install --key cust_abc123xyz`
3. Use a configuration profile to allow the binary past Gatekeeper (recommended over manual `spctl --add`)

### Linux — Ansible / Chef / Puppet

Deploy the `.deb` or `.rpm` via your configuration management tool. The install command writes the systemd unit and enables the service in a single step.

## Building from source

**Requirements:** Docker with BuildKit (no local Go installation needed)

```sh
# Build all 5 platform binaries at once
make -f docker/Makefile build-all

# Build a single platform
make -f docker/Makefile build-darwin-arm64
make -f docker/Makefile build-linux-amd64
make -f docker/Makefile build-windows

# Run vet + tests for all platforms
make -f docker/Makefile vet-all
make -f docker/Makefile test

# Build Linux .deb and .rpm packages
make -f docker/Makefile package-linux
```

Outputs land in `dist/`:

```
dist/
  bestdefense-device-monitor-windows-amd64.exe
  bestdefense-device-monitor-darwin-amd64
  bestdefense-device-monitor-darwin-arm64
  bestdefense-device-monitor-linux-amd64
  bestdefense-device-monitor-linux-arm64
  bestdefense-device-monitor_amd64.deb
  bestdefense-device-monitor.x86_64.rpm
```

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
