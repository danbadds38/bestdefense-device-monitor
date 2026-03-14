# Data Collected by BestDefense Device Monitor

This document is a complete, field-by-field description of every data point collected
and transmitted by the agent. You can verify this independently by running:

```
# Windows
bestdefense-device-monitor-windows-amd64.exe check

# macOS
./bestdefense-device-monitor-darwin-arm64 check

# Linux
./bestdefense-device-monitor-linux-amd64 check
```

This prints the exact JSON payload that would be sent to the API, before you deploy.

The top-level `platform` field in the payload identifies the operating system (`"windows"`, `"darwin"`, or `"linux"`). Sections marked **Windows only**, **macOS only**, or **Linux only** below are populated on their respective platforms and omitted (or contain a `collection_error`) on others.

---

## Top-Level Fields

| Field | Value | PII Risk | Source |
|-------|-------|----------|--------|
| `schema_version` | Always `"1"` | None | Hard-coded |
| `registration_key` | Your customer ID | None | Config file |
| `agent_version` | e.g. `"1.0.0"` | None | Build-time constant |
| `collected_at` | UTC timestamp of collection | None | `time.Now()` |

---

## device_identity

| Field | Description | PII Risk | Windows Source | macOS Source | Linux Source |
|-------|-------------|----------|----------------|--------------|--------------|
| `hostname` | Machine network name | Low | `os.Hostname()` | `hostname` cmd | `hostname` cmd |
| `mac_addresses` | MAC addresses of all interfaces | Low | `net.Interfaces()` | `net.Interfaces()` | `net.Interfaces()` |
| `serial_number` | Hardware serial number | Low | `Win32_BIOS.SerialNumber` (WMI) | `system_profiler SPHardwareDataType` | `/sys/class/dmi/id/product_serial` |
| `hardware_uuid` | SMBIOS/IOREG hardware UUID | Low | `Win32_ComputerSystemProduct.UUID` (WMI) | `system_profiler SPHardwareDataType` | `/sys/class/dmi/id/product_uuid` |
| `computer_name` | Same as hostname | Low | `os.Hostname()` | `hostname` cmd | `hostname` cmd |
| `domain` | AD domain name (**Windows only**) | Low | `Win32_ComputerSystem.Domain` (WMI) | — | — |

---

## os

| Field | Description | PII Risk | Windows Source | macOS Source | Linux Source |
|-------|-------------|----------|----------------|--------------|--------------|
| `name` | e.g. `"Windows 11 Pro"`, `"macOS"`, `"Ubuntu 22.04 LTS"` | None | Registry: `ProductName` | `sw_vers -productName` | `/etc/os-release PRETTY_NAME` |
| `version` | e.g. `"10.0.22621"`, `"14.4.1"`, `"22.04"` | None | `Win32_OperatingSystem.Version` (WMI) | `sw_vers -productVersion` | `/etc/os-release VERSION_ID` |
| `build_number` | e.g. `"22621"` (Windows), kernel version (Linux) | None | Registry: `CurrentBuildNumber` | `sw_vers -buildVersion` | `uname -r` |
| `display_version` | e.g. `"22H2"` (**Windows only**) | None | Registry: `DisplayVersion` | — | — |
| `architecture` | `"x86_64"` or `"arm64"` | None | Registry: `PROCESSOR_ARCHITECTURE` | `uname -m` | `uname -m` |
| `install_date` | When OS was installed (**Windows only**) | None | Registry: `InstallDate` | — | — |
| `registered_owner` | Name set at install (**Windows only**) | **Medium** | Registry: `RegisteredOwner` | — | — |
| `registered_organization` | Org set at install (**Windows only**) | Low | Registry: `RegisteredOrganization` | — | — |

> **Note on `registered_owner`**: Windows only. May contain an employee's name if set during OS install. Collected to help identify whose machine is being monitored.

---

## hardware

| Field | Description | PII Risk | Windows Source | macOS Source | Linux Source |
|-------|-------------|----------|----------------|--------------|--------------|
| `cpu_name` | e.g. `"Intel Core i7-1185G7"` | None | `Win32_Processor.Name` (WMI) | `system_profiler SPHardwareDataType` | `/proc/cpuinfo model name` |
| `cpu_cores` | Physical CPU cores | None | `Win32_Processor.NumberOfCores` | `system_profiler` | `lscpu Core(s) per socket` |
| `cpu_logical_processors` | Logical processors (incl. hyperthreading) | None | `Win32_Processor.NumberOfLogicalProcessors` | `system_profiler` | `/proc/cpuinfo processor count` |
| `ram_total_bytes` | Total visible RAM in bytes | None | `Win32_OperatingSystem.TotalVisibleMemorySize` | `sysctl hw.memsize` | `/proc/meminfo MemTotal` |
| `disks[].device_id` | e.g. `"\\.\PHYSICALDRIVE0"`, `"/dev/sda"` | None | `Win32_DiskDrive.DeviceID` | `diskutil list` | `lsblk -d` |
| `disks[].model` | Drive model name | None | `Win32_DiskDrive.Model` | `system_profiler SPStorageDataType` | `lsblk model` |
| `disks[].size_bytes` | Drive capacity | None | `Win32_DiskDrive.Size` | `diskutil info` | `lsblk -b size` |
| `disks[].media_type` | `"SSD"` or `"HDD"` | None | `Win32_DiskDrive.MediaType` | `system_profiler` | `lsblk rota` field |
| `disks[].interface_type` | e.g. `"NVMe"`, `"SATA"`, `"USB"` | None | `Win32_DiskDrive.InterfaceType` | `system_profiler` | `lsblk tran` field |

---

## disk_encryption

Requires administrator/root privileges. On Windows this reflects BitLocker; on macOS, FileVault; on Linux, LUKS dm-crypt volumes. If the agent lacks access, `collection_error` is set and drive data is omitted.

| Field | Description | PII Risk | Windows Source | macOS Source | Linux Source |
|-------|-------------|----------|----------------|--------------|--------------|
| `drives[].drive_letter` | e.g. `"C:"`, `"/"`, `"/dev/sda2"` | None | `Win32_EncryptableVolume.DriveLetter` (WMI) | Volume mount point | `lsblk` device path |
| `drives[].protection_status` | `"protected"` / `"unprotected"` | None | `Win32_EncryptableVolume.ProtectionStatus` | `fdesetup status` | `lsblk type=crypt` detection |
| `drives[].encryption_method` | e.g. `"XtsAes256"`, `"FileVault"`, `"LUKS2"` | None | `Win32_EncryptableVolume.EncryptionMethod` | `fdesetup` | `dmsetup status` |
| `drives[].lock_status` | `"locked"` / `"unlocked"` | None | `Win32_EncryptableVolume.LockStatus` | Derived from mount state | Derived from dm-crypt presence |
| `drives[].conversion_status` | `"fully_encrypted"` / `"encrypting"` etc. | None | `Win32_EncryptableVolume.ConversionStatus` | `fdesetup` | `dmsetup` |
| `drives[].percentage_encrypted` | 0–100 | None | `Win32_EncryptableVolume.EncryptionPercentage` | 100 when encrypted | 100 when encrypted |

---

## antivirus

**Windows**: requires local administrator or SYSTEM privileges to access the Defender WMI namespace.
**macOS**: scans for known AV app bundles in `/Applications` and checks XProtect version — no special privileges needed.
**Linux**: checks known AV service names via `systemctl is-active`.

| Field | Description | PII Risk | Windows Source | macOS Source | Linux Source |
|-------|-------------|----------|----------------|--------------|--------------|
| `windows_defender_enabled` | Windows Defender running (**Windows only**) | None | `MSFT_MpComputerStatus.AMServiceEnabled` | always `false` | always `false` |
| `realtime_protection_enabled` | Real-time scanning on | None | `MSFT_MpComputerStatus` | AV service running | Known AV systemd service active |
| `antispyware_enabled` | Antispyware component (**Windows only**) | None | `MSFT_MpComputerStatus` | — | — |
| `behavior_monitor_enabled` | Behavioral monitoring (**Windows only**) | None | `MSFT_MpComputerStatus` | — | — |
| `on_access_protection_enabled` | On-access file scanning | None | `MSFT_MpComputerStatus` | Derived from AV presence | Derived from AV presence |
| `definition_version` | Signature database version | None | `MSFT_MpComputerStatus` | XProtect plist version | `clamscan --version` |
| `definition_date` | When signatures were last updated | None | `MSFT_MpComputerStatus` | — | — |
| `am_service_enabled` | AM service state | None | `MSFT_MpComputerStatus` | Derived | Derived |
| `product_status` | `"active"` / `"not_detected"` / `"disabled"` | None | Derived | Derived | Derived |

---

## firewall

Windows reports three named profiles (domain/private/public). macOS and Linux only populate `profiles.public` — the other two will have `enabled: false`.

| Field | Description | PII Risk | Windows Source | macOS Source | Linux Source |
|-------|-------------|----------|----------------|--------------|--------------|
| `profiles.domain.*` | Domain profile (**Windows only**) | None | `HNetCfg.FwPolicy2` (COM) | — | — |
| `profiles.private.*` | Private profile (**Windows only**) | None | `HNetCfg.FwPolicy2` (COM) | — | — |
| `profiles.public.enabled` | Firewall on/off | None | `HNetCfg.FwPolicy2` (COM) | `socketfilterfw --getglobalstate` | `ufw status` / `firewall-cmd --state` / `iptables -L` |
| `profiles.public.default_inbound_action` | `"block"` / `"allow"` | None | `HNetCfg.FwPolicy2` | `"block"` when enabled | `"block"` when enabled |
| `profiles.public.default_outbound_action` | `"block"` / `"allow"` | None | `HNetCfg.FwPolicy2` | `"allow"` | `"allow"` |

---

## screen_lock

| Field | Description | PII Risk | Windows Source | macOS Source | Linux Source |
|-------|-------------|----------|----------------|--------------|--------------|
| `screensaver_enabled` | Is screensaver/idle lock active | None | Registry: `HKCU\Control Panel\Desktop\ScreenSaveActive` | `defaults read com.apple.screensaver idleTime` | `gsettings get org.gnome.desktop.screensaver idle-delay` / `xset q` |
| `screensaver_timeout_seconds` | Inactivity timeout | None | Registry: `ScreenSaveTimeOut` | `com.apple.screensaver idleTime` | `gsettings` idle-delay / KDE Daemon Timeout |
| `screensaver_requires_password` | Password required on resume | None | Registry: `ScreenSaverIsSecure` | `com.apple.screensaver askForPassword` | `gsettings lock-enabled` |
| `lock_on_sleep` | Whether machine locks on sleep | None | Registry: Power policy | `com.apple.screensaver lockDelay == 0` | Derived from lock-delay setting |

---

## software_update

| Field | Description | PII Risk | Windows Source | macOS Source | Linux Source |
|-------|-------------|----------|----------------|--------------|--------------|
| `automatic_updates_enabled` | Auto-updates enabled | None | Registry: `WindowsUpdate\AU\AUOptions` | `defaults read com.apple.SoftwareUpdate AutomaticCheckEnabled` | `/etc/apt/apt.conf.d/20auto-upgrades` or `/etc/dnf/automatic.conf` |
| `au_option` | `"auto_install"`, `"auto_download"`, `"notify"`, `"disabled"` | None | Registry: `AUOptions` value | Derived from SoftwareUpdate prefs | Derived from apt/dnf config |
| `wsus_server` | Corporate WSUS server URL (**Windows only**) | Low | Registry: `WindowsUpdate\WUServer` | — | — |
| `last_successful_update_time` | Last successful update timestamp (**Windows only**) | None | Registry: `Auto Update\Results\Install\LastSuccessTime` | — | — |
| `pending_reboot` | Reboot pending for updates (**Windows only**) | None | Registry: `Auto Update\RebootRequired` key existence | — | — |

---

## installed_applications

Only application names, versions, and publishers — **no file contents, no usage data**.

| Field | Description | PII Risk | Windows Source | macOS Source | Linux Source |
|-------|-------------|----------|----------------|--------------|--------------|
| `applications[].name` | Application name | Low | Registry: `Uninstall\*\DisplayName` | `system_profiler SPApplicationsDataType` `_name` | `dpkg-query` Package / `rpm -qa` NAME |
| `applications[].version` | Version string | None | Registry: `Uninstall\*\DisplayVersion` | `system_profiler` version | dpkg/rpm version |
| `applications[].publisher` | Publisher/maintainer | None | Registry: `Uninstall\*\Publisher` | — | dpkg Maintainer / rpm Vendor |
| `applications[].install_date` | Date installed | None | Registry: `Uninstall\*\InstallDate` | — | `rpm --queryformat %{INSTALLTIME:date}` |
| `applications[].install_location` | Install path | Low | Registry: `Uninstall\*\InstallLocation` | `system_profiler` path | — |
| `applications[].source` | Where the app came from | None | Registry hive name | `obtained_from` field | `dpkg` or `rpm` |
| `total_count` | Total number of apps | None | Derived | Derived | Derived |

> **Windows note**: We read from 4 registry hives (`HKLM` x64, `HKLM` WOW6432Node, `HKCU` x64, `HKCU` WOW6432Node). We deliberately do **not** use `Win32_Product` (WMI) as that triggers unintended MSI repair sequences.
> **Linux note**: Auto-detects whether to use `dpkg-query` (Debian/Ubuntu) or `rpm` (RHEL/Fedora) based on which tool is available.

---

## local_users

| Field | Description | PII Risk | Windows Source | macOS Source | Linux Source |
|-------|-------------|----------|----------------|--------------|--------------|
| `accounts[].username` | Local account name | **Medium** | `NetUserEnum` (Win32 API, level 1) | `dscl . list /Users` | `/etc/passwd` username field |
| `accounts[].full_name` | Display name (**Windows only**) | **Medium** | `NetUserEnum.Usri1_full_name` | — | — |
| `accounts[].is_admin` | Member of admin group | Low | `NetLocalGroupGetMembers("Administrators")` | `dseditgroup -o checkmember -m <user> admin` | Member of `sudo`/`wheel`/`admin` in `/etc/group` |
| `accounts[].is_enabled` | Account enabled/disabled | None | `Usri1_flags & UF_ACCOUNTDISABLE` | `dscl AuthenticationAuthority` | Non-nologin shell in `/etc/passwd` |
| `accounts[].is_local` | Always `true` | None | Derived | Derived | Derived |
| `accounts[].password_required` | Password required | None | `Usri1_flags & UF_PASSWD_NOTREQD` | Derived | Assumed `true` for all active accounts |
| `accounts[].password_never_expires` | Password expiry (**Windows only**) | None | `Usri1_flags & UF_DONT_EXPIRE_PASSWD` | — | — |

> **Note on `username`**: May contain employee names. Collected to identify privileged accounts and disabled/stale accounts — standard compliance checks. System accounts with no-login shells (e.g. `daemon`, `nobody`) are excluded on macOS and Linux.

---

## password_policy

Machine-level password policy only — not per-user passwords or credentials.

| Field | Description | PII Risk | Windows Source | macOS Source | Linux Source |
|-------|-------------|----------|----------------|--------------|--------------|
| `min_password_length` | Minimum required password length | None | `NetUserModalsGet` level 0 | `pwpolicy -n . -getglobalhashtable` minChars | `/etc/login.defs PASS_MIN_LEN` or PAM pwquality `minlen` |
| `max_password_age_days` | Max days before password must change | None | `NetUserModalsGet` level 0 | `pwpolicy` maxMinutesUntilChangePassword | `/etc/login.defs PASS_MAX_DAYS` |
| `min_password_age_days` | Minimum days before password can change | None | `NetUserModalsGet` level 0 | — | `/etc/login.defs PASS_MIN_DAYS` |
| `password_history_count` | Previous passwords remembered | None | `NetUserModalsGet` level 0 | `pwpolicy` usingHistory | — |
| `complexity_enabled` | Requires mixed case/numbers/symbols | None | Registry: `Lsa\PasswordComplexity` | `pwpolicy` requiresAlpha + requiresNumeric | PAM `pam_pwquality` dcredit/ucredit/lcredit |
| `lockout_threshold` | Failed attempts before lockout (0 = never) | None | `NetUserModalsGet` level 3 | `pwpolicy` maxFailedLoginAttempts | PAM `pam_faillock deny` or `pam_tally2 deny` |
| `lockout_duration_minutes` | How long account stays locked | None | `NetUserModalsGet` level 3 | — | PAM `pam_faillock unlock_time / 60` |
| `lockout_observation_window_minutes` | Window for counting failed attempts | None | `NetUserModalsGet` level 3 | — | PAM `pam_faillock fail_interval / 60` |

---

## network_interfaces

**No traffic data, no packet capture, no DNS queries, no connection state.** Same implementation on all platforms using Go's standard library `net` package.

| Field | Description | PII Risk | Source (all platforms) |
|-------|-------------|----------|------------------------|
| `interfaces[].name` | Adapter name e.g. `"en0"`, `"eth0"` | None | `net.Interfaces()` |
| `interfaces[].description` | Adapter description (Windows only) | None | WMI `Win32_NetworkAdapter.Description` |
| `interfaces[].mac_address` | Hardware MAC address | Low | `net.Interface.HardwareAddr` |
| `interfaces[].ip_addresses` | Assigned IP addresses (IPv4 and IPv6) | Low | `net.Interface.Addrs()` |
| `interfaces[].dns_servers` | Configured DNS servers (Windows only) | Low | WMI `Win32_NetworkAdapterConfiguration` |
| `interfaces[].is_up` | Whether adapter is active | None | `net.FlagUp` |
| `interfaces[].type` | `"ethernet"`, `"wifi"`, or `"loopback"` | None | Derived from interface name/flags |

---

## system_health

| Field | Description | PII Risk | Windows Source | macOS Source | Linux Source |
|-------|-------------|----------|----------------|--------------|--------------|
| `last_reboot_time` | When the machine was last rebooted | None | `Win32_OperatingSystem.LastBootUpTime` (WMI) | `sysctl kern.boottime` | `/proc/uptime` (current time minus uptime) |
| `uptime_hours` | Hours since last reboot | None | Derived | Derived | `/proc/uptime` first field ÷ 3600 |

---

## What We Do NOT Collect

- Browser history, cookies, or saved passwords
- File contents of any kind
- Keystrokes or clipboard contents
- Email content
- Screenshots or screen recordings
- Running process details (command lines, arguments)
- Network traffic or packet data
- User activity logs
- Location data
- Camera or microphone data
- Documents or personal files
