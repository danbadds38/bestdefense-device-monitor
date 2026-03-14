# Data Collected by BestDefense Device Monitor

This document is a complete, field-by-field description of every data point collected
and transmitted by the agent. You can verify this independently by running:

```
bestdefense-device-monitor.exe check
```

This prints the exact JSON payload that would be sent to the API, before you deploy.

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

| Field | Description | PII Risk | Win32 Source |
|-------|-------------|----------|--------------|
| `hostname` | Machine network name | Low | `os.Hostname()` |
| `mac_addresses` | Hardware MAC addresses of active interfaces | Low | `net.Interfaces()` |
| `serial_number` | BIOS serial number | Low | `Win32_BIOS.SerialNumber` (WMI) |
| `hardware_uuid` | SMBIOS hardware UUID | Low | `Win32_ComputerSystemProduct.UUID` (WMI) |
| `computer_name` | Same as hostname | Low | `os.Hostname()` |
| `domain` | Workgroup or Active Directory domain name | Low | `Win32_ComputerSystem.Domain` (WMI) |

---

## os

| Field | Description | PII Risk | Win32 Source |
|-------|-------------|----------|--------------|
| `name` | e.g. `"Windows 11 Pro"` | None | Registry: `SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProductName` |
| `version` | e.g. `"10.0.22621"` | None | `Win32_OperatingSystem.Version` (WMI) |
| `build_number` | e.g. `"22621"` | None | Registry: `CurrentBuildNumber` |
| `display_version` | e.g. `"22H2"` | None | Registry: `DisplayVersion` |
| `architecture` | `"x86_64"` or `"arm64"` | None | Registry: `PROCESSOR_ARCHITECTURE` |
| `install_date` | When Windows was installed | None | Registry: `InstallDate` (DWORD unix timestamp) |
| `registered_owner` | Name set at Windows install | **Medium** | Registry: `RegisteredOwner` |
| `registered_organization` | Org set at Windows install | Low | Registry: `RegisteredOrganization` |

> **Note on `registered_owner`**: This may contain an employee's name if set during OS install. It is included because it helps identify whose machine is being monitored and is the same data surface as the Windows "About" screen.

---

## hardware

| Field | Description | PII Risk | Win32 Source |
|-------|-------------|----------|--------------|
| `cpu_name` | e.g. `"Intel Core i7-1185G7"` | None | `Win32_Processor.Name` (WMI) |
| `cpu_cores` | Physical CPU cores | None | `Win32_Processor.NumberOfCores` |
| `cpu_logical_processors` | Logical processors (inc. hyperthreading) | None | `Win32_Processor.NumberOfLogicalProcessors` |
| `ram_total_bytes` | Total visible RAM in bytes | None | `Win32_OperatingSystem.TotalVisibleMemorySize` |
| `disks[].device_id` | e.g. `"\\.\PHYSICALDRIVE0"` | None | `Win32_DiskDrive.DeviceID` |
| `disks[].model` | Drive model name | None | `Win32_DiskDrive.Model` |
| `disks[].size_bytes` | Drive capacity | None | `Win32_DiskDrive.Size` |
| `disks[].media_type` | `"SSD"` or `"HDD"` | None | `Win32_DiskDrive.MediaType` |
| `disks[].interface_type` | e.g. `"NVMe"`, `"SATA"` | None | `Win32_DiskDrive.InterfaceType` |

---

## bitlocker

Requires local administrator or SYSTEM privileges. If the agent lacks access, this section will contain `collection_error` instead of drive data.

| Field | Description | PII Risk | Win32 Source |
|-------|-------------|----------|--------------|
| `drives[].drive_letter` | e.g. `"C:"` | None | `Win32_EncryptableVolume.DriveLetter` (WMI) |
| `drives[].protection_status` | `"protected"` / `"unprotected"` | None | `Win32_EncryptableVolume.ProtectionStatus` |
| `drives[].encryption_method` | e.g. `"XtsAes256"` | None | `Win32_EncryptableVolume.EncryptionMethod` |
| `drives[].lock_status` | `"locked"` / `"unlocked"` | None | `Win32_EncryptableVolume.LockStatus` |
| `drives[].conversion_status` | `"fully_encrypted"` etc. | None | `Win32_EncryptableVolume.ConversionStatus` |
| `drives[].percentage_encrypted` | 0–100 | None | `Win32_EncryptableVolume.EncryptionPercentage` |

---

## antivirus

Requires local administrator or SYSTEM privileges to access the Defender WMI namespace.

| Field | Description | PII Risk | Win32 Source |
|-------|-------------|----------|--------------|
| `windows_defender_enabled` | Is Windows Defender service running | None | `MSFT_MpComputerStatus.AMServiceEnabled` |
| `realtime_protection_enabled` | Is real-time scanning on | None | `MSFT_MpComputerStatus.RealTimeProtectionEnabled` |
| `antispyware_enabled` | Antispyware component status | None | `MSFT_MpComputerStatus.AntispywareEnabled` |
| `behavior_monitor_enabled` | Behavioral monitoring status | None | `MSFT_MpComputerStatus.BehaviorMonitorEnabled` |
| `on_access_protection_enabled` | On-access file scanning | None | `MSFT_MpComputerStatus.OnAccessProtectionEnabled` |
| `definition_version` | Signature database version | None | `MSFT_MpComputerStatus.AntivirusSignatureVersion` |
| `definition_date` | When signatures were last updated | None | `MSFT_MpComputerStatus.AntivirusSignatureLastUpdated` |
| `am_service_enabled` | AM service state | None | `MSFT_MpComputerStatus.AMServiceEnabled` |
| `product_status` | `"up_to_date"` / `"disabled"` / `"definitions_outdated"` | None | Derived |

---

## firewall

| Field | Description | PII Risk | Win32 Source |
|-------|-------------|----------|--------------|
| `profiles.domain.enabled` | Domain profile firewall on/off | None | `HNetCfg.FwPolicy2.FirewallEnabled(1)` (COM) |
| `profiles.domain.default_inbound_action` | `"block"` / `"allow"` | None | `HNetCfg.FwPolicy2.DefaultInboundAction(1)` |
| `profiles.domain.default_outbound_action` | `"block"` / `"allow"` | None | `HNetCfg.FwPolicy2.DefaultOutboundAction(1)` |
| `profiles.private.*` | Same for private profile | None | COM profile 2 |
| `profiles.public.*` | Same for public profile | None | COM profile 4 |

---

## screen_lock

| Field | Description | PII Risk | Win32 Source |
|-------|-------------|----------|--------------|
| `screensaver_enabled` | Is screensaver active | None | Registry: `HKCU\Control Panel\Desktop\ScreenSaveActive` |
| `screensaver_timeout_seconds` | Inactivity timeout | None | Registry: `HKCU\Control Panel\Desktop\ScreenSaveTimeOut` |
| `screensaver_requires_password` | Password on resume | None | Registry: `HKCU\Control Panel\Desktop\ScreenSaverIsSecure` |
| `lock_on_sleep` | Whether machine locks on display off | None | Registry: `HKLM\Software\Policies\Microsoft\Power\PowerSettings\...` |

---

## windows_update

| Field | Description | PII Risk | Win32 Source |
|-------|-------------|----------|--------------|
| `automatic_updates_enabled` | Auto-update behavior enabled | None | Registry: `HKLM\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU\AUOptions` |
| `au_option` | `"auto_install"`, `"notify"`, `"disabled"` etc. | None | Registry: `AUOptions` |
| `wsus_server` | Corporate WSUS server URL if configured | Low | Registry: `HKLM\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\WUServer` |
| `last_successful_update_time` | Last Windows Update success timestamp | None | Registry: `...WindowsUpdate\Auto Update\Results\Install\LastSuccessTime` |
| `pending_reboot` | Whether a reboot is pending for updates | None | Registry: `...WindowsUpdate\Auto Update\RebootRequired` key existence |

---

## installed_applications

Only application names, versions, and publishers from the registry — **no file contents, no usage data**.

| Field | Description | PII Risk | Win32 Source |
|-------|-------------|----------|--------------|
| `applications[].name` | Application display name | Low | Registry: `Uninstall\*\DisplayName` |
| `applications[].version` | Version string | None | Registry: `Uninstall\*\DisplayVersion` |
| `applications[].publisher` | Publisher name | None | Registry: `Uninstall\*\Publisher` |
| `applications[].install_date` | Date installed (YYYYMMDD) | None | Registry: `Uninstall\*\InstallDate` |
| `applications[].install_location` | Install directory path | Low | Registry: `Uninstall\*\InstallLocation` |
| `total_count` | Total number of apps found | None | Derived |

> **Note**: We read from 4 registry hives (`HKLM` x64, `HKLM` WOW6432Node, `HKCU` x64, `HKCU` WOW6432Node). We deliberately **do not** use `Win32_Product` (WMI) as that triggers unintended MSI repair sequences.

---

## local_users

| Field | Description | PII Risk | Win32 Source |
|-------|-------------|----------|--------------|
| `accounts[].username` | Local account name | **Medium** | `NetUserEnum` (Win32 API, level 1) |
| `accounts[].full_name` | Display name | **Medium** | `NetUserEnum.Usri1_full_name` |
| `accounts[].is_admin` | Member of Administrators group | Low | `NetLocalGroupGetMembers("Administrators")` |
| `accounts[].is_enabled` | Account enabled/disabled | None | `Usri1_flags & UF_ACCOUNTDISABLE` |
| `accounts[].is_local` | Always `true` (we only enumerate local accounts) | None | Derived |
| `accounts[].password_required` | Whether a password is required | None | `Usri1_flags & UF_PASSWD_NOTREQD` |
| `accounts[].password_never_expires` | Password expiry setting | None | `Usri1_flags & UF_DONT_EXPIRE_PASSWD` |

> **Note on `username` and `full_name`**: These may contain employee names. They are collected to identify privileged accounts and disabled/stale accounts — standard compliance checks.

---

## password_policy

Machine-level password policy only (not per-user passwords).

| Field | Description | PII Risk | Win32 Source |
|-------|-------------|----------|--------------|
| `min_password_length` | Minimum required password length | None | `NetUserModalsGet` level 0 |
| `max_password_age_days` | Max days before password must change (0 = never) | None | `NetUserModalsGet` level 0 |
| `min_password_age_days` | Minimum days before password can change | None | `NetUserModalsGet` level 0 |
| `password_history_count` | Number of previous passwords remembered | None | `NetUserModalsGet` level 0 |
| `complexity_enabled` | Requires mixed case/numbers/symbols | None | Registry: `HKLM\SYSTEM\...\Lsa\PasswordComplexity` |
| `lockout_threshold` | Failed attempts before lockout (0 = never) | None | `NetUserModalsGet` level 3 |
| `lockout_duration_minutes` | How long account stays locked | None | `NetUserModalsGet` level 3 |
| `lockout_observation_window_minutes` | Window for counting failed attempts | None | `NetUserModalsGet` level 3 |

---

## network_interfaces

**No traffic data, no packet capture, no DNS queries, no connection state.**

| Field | Description | PII Risk | Win32 Source |
|-------|-------------|----------|--------------|
| `interfaces[].name` | Adapter name e.g. `"Ethernet"` | None | `net.Interfaces()` |
| `interfaces[].mac_address` | Hardware MAC address | Low | `net.Interface.HardwareAddr` |
| `interfaces[].ip_addresses` | Assigned IP addresses (IPv4 and IPv6) | Low | `net.Interface.Addrs()` |
| `interfaces[].is_up` | Whether adapter is active | None | `net.FlagUp` |
| `interfaces[].type` | `"ethernet"`, `"wifi"`, or `"loopback"` | None | Derived from name |

---

## system_health

| Field | Description | PII Risk | Win32 Source |
|-------|-------------|----------|--------------|
| `last_reboot_time` | When the machine was last rebooted | None | `Win32_OperatingSystem.LastBootUpTime` (WMI) |
| `uptime_hours` | Hours since last reboot | None | Derived |

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
