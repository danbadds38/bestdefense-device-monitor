package reporter

import "time"

// DeviceReport is the top-level JSON payload sent to the API.
// Every subsection carries its own CollectionError so a single failed check
// does not prevent the rest of the data from being reported.
type DeviceReport struct {
	SchemaVersion   string    `json:"schema_version"`
	RegistrationKey string    `json:"registration_key"`
	AgentVersion    string    `json:"agent_version"`
	CollectedAt     time.Time `json:"collected_at"`
	Platform        string    `json:"platform"` // "windows", "darwin", "linux"

	DeviceIdentity    DeviceIdentity    `json:"device_identity"`
	OS                OSInfo            `json:"os"`
	Hardware          HardwareInfo      `json:"hardware"`
	DiskEncryption    DiskEncryptionInfo `json:"disk_encryption"`
	Antivirus         AntivirusInfo     `json:"antivirus"`
	Firewall          FirewallInfo      `json:"firewall"`
	ScreenLock        ScreenLockInfo    `json:"screen_lock"`
	SoftwareUpdate    SoftwareUpdateInfo `json:"software_update"`
	InstalledApps     InstalledAppsInfo `json:"installed_applications"`
	LocalUsers        LocalUsersInfo    `json:"local_users"`
	PasswordPolicy    PasswordPolicyInfo `json:"password_policy"`
	NetworkInterfaces NetworkInfo       `json:"network_interfaces"`
	SystemHealth      SystemHealthInfo  `json:"system_health"`

	CheckErrors []CheckError `json:"check_errors"`
}

// CheckError captures a fatal failure for a named check.
type CheckError struct {
	Check string `json:"check"`
	Error string `json:"error"`
}

// DeviceIdentity identifies the physical machine.
type DeviceIdentity struct {
	Hostname        string   `json:"hostname"`
	MACAddresses    []string `json:"mac_addresses"`
	SerialNumber    string   `json:"serial_number"`
	HardwareUUID    string   `json:"hardware_uuid"`
	ComputerName    string   `json:"computer_name"`
	Domain          string   `json:"domain"`
	CollectionError *string  `json:"collection_error,omitempty"`
}

// OSInfo describes the operating system.
type OSInfo struct {
	Name                   string     `json:"name"`
	Version                string     `json:"version"`
	BuildNumber            string     `json:"build_number"`
	DisplayVersion         string     `json:"display_version"`
	Architecture           string     `json:"architecture"`
	InstallDate            *time.Time `json:"install_date,omitempty"`
	RegisteredOwner        string     `json:"registered_owner,omitempty"`
	RegisteredOrganization string     `json:"registered_organization,omitempty"`
	CollectionError        *string    `json:"collection_error,omitempty"`
}

// HardwareInfo describes the physical hardware.
type HardwareInfo struct {
	CPUName         string     `json:"cpu_name"`
	CPUCores        int        `json:"cpu_cores"`
	CPULogicalProcs int        `json:"cpu_logical_processors"`
	RAMTotalBytes   int64      `json:"ram_total_bytes"`
	Disks           []DiskInfo `json:"disks"`
	CollectionError *string    `json:"collection_error,omitempty"`
}

// DiskInfo describes a single physical disk.
type DiskInfo struct {
	DeviceID      string `json:"device_id"`
	Model         string `json:"model"`
	SizeBytes     int64  `json:"size_bytes"`
	MediaType     string `json:"media_type"`
	InterfaceType string `json:"interface_type"`
}

// DiskEncryptionInfo reports encryption status for all drives.
// On Windows this reflects BitLocker; on macOS, FileVault; on Linux, LUKS.
type DiskEncryptionInfo struct {
	Drives          []EncryptedDriveInfo `json:"drives"`
	CollectionError *string              `json:"collection_error,omitempty"`
}

// EncryptedDriveInfo is the encryption status of one drive or volume.
type EncryptedDriveInfo struct {
	DriveLetter         string `json:"drive_letter"`
	ProtectionStatus    string `json:"protection_status"`    // "protected", "unprotected", "unknown"
	EncryptionMethod    string `json:"encryption_method"`    // "XtsAes256", "FileVault", "LUKS2", etc.
	LockStatus          string `json:"lock_status"`          // "unlocked", "locked"
	ConversionStatus    string `json:"conversion_status"`    // "fully_encrypted", "encrypting", etc.
	PercentageEncrypted int    `json:"percentage_encrypted"`
}

// AntivirusInfo reports antivirus/EDR status.
type AntivirusInfo struct {
	WindowsDefenderEnabled    bool       `json:"windows_defender_enabled"`
	RealtimeProtectionEnabled bool       `json:"realtime_protection_enabled"`
	AntispywareEnabled        bool       `json:"antispyware_enabled"`
	BehaviorMonitorEnabled    bool       `json:"behavior_monitor_enabled"`
	OnAccessProtectionEnabled bool       `json:"on_access_protection_enabled"`
	DefinitionVersion         string     `json:"definition_version"`
	DefinitionDate            *time.Time `json:"definition_date,omitempty"`
	AMServiceEnabled          bool       `json:"am_service_enabled"`
	ProductStatus             string     `json:"product_status"`
	CollectionError           *string    `json:"collection_error,omitempty"`
}

// FirewallInfo reports firewall status.
type FirewallInfo struct {
	Profiles        FirewallProfiles `json:"profiles"`
	CollectionError *string          `json:"collection_error,omitempty"`
}

// FirewallProfiles holds firewall profile states.
// On Windows: domain/private/public profiles. On macOS/Linux: only "public" is populated.
type FirewallProfiles struct {
	Domain  FirewallProfile `json:"domain"`
	Private FirewallProfile `json:"private"`
	Public  FirewallProfile `json:"public"`
}

// FirewallProfile is the state of a single firewall profile.
type FirewallProfile struct {
	Enabled               bool   `json:"enabled"`
	DefaultInboundAction  string `json:"default_inbound_action"`  // "block", "allow"
	DefaultOutboundAction string `json:"default_outbound_action"` // "block", "allow"
}

// ScreenLockInfo reports screen lock and screensaver settings.
type ScreenLockInfo struct {
	ScreensaverEnabled          bool    `json:"screensaver_enabled"`
	ScreensaverTimeoutSeconds   int     `json:"screensaver_timeout_seconds"`
	ScreensaverRequiresPassword bool    `json:"screensaver_requires_password"`
	LockOnSleep                 bool    `json:"lock_on_sleep"`
	CollectionError             *string `json:"collection_error,omitempty"`
}

// SoftwareUpdateInfo reports software update configuration.
// On Windows: Windows Update settings. On macOS: SoftwareUpdate preferences.
// On Linux: unattended-upgrades / dnf-automatic configuration.
type SoftwareUpdateInfo struct {
	AutomaticUpdatesEnabled  bool       `json:"automatic_updates_enabled"`
	AUOption                 string     `json:"au_option"` // "disabled", "notify", "auto_download", "auto_install"
	WSUSServer               *string    `json:"wsus_server,omitempty"`
	LastSuccessfulUpdateTime *time.Time `json:"last_successful_update_time,omitempty"`
	PendingReboot            bool       `json:"pending_reboot"`
	CollectionError          *string    `json:"collection_error,omitempty"`
}

// InstalledAppsInfo lists installed applications.
type InstalledAppsInfo struct {
	Applications    []InstalledApp `json:"applications"`
	TotalCount      int            `json:"total_count"`
	CollectionError *string        `json:"collection_error,omitempty"`
}

// InstalledApp is a single installed application.
type InstalledApp struct {
	Name            string `json:"name"`
	Version         string `json:"version,omitempty"`
	Publisher       string `json:"publisher,omitempty"`
	InstallDate     string `json:"install_date,omitempty"`
	InstallLocation string `json:"install_location,omitempty"`
	Source          string `json:"source"` // registry hive, /Applications, dpkg, rpm, etc.
}

// LocalUsersInfo lists local user accounts.
type LocalUsersInfo struct {
	Accounts        []LocalUser `json:"accounts"`
	CollectionError *string     `json:"collection_error,omitempty"`
}

// LocalUser describes a local user account.
type LocalUser struct {
	Username            string     `json:"username"`
	FullName            string     `json:"full_name,omitempty"`
	IsAdmin             bool       `json:"is_admin"`
	IsEnabled           bool       `json:"is_enabled"`
	IsLocal             bool       `json:"is_local"`
	PasswordRequired    bool       `json:"password_required"`
	PasswordNeverExpires bool      `json:"password_never_expires"`
	LastLogon           *time.Time `json:"last_logon,omitempty"`
}

// PasswordPolicyInfo reports the local password policy.
type PasswordPolicyInfo struct {
	MinPasswordLength               int     `json:"min_password_length"`
	MaxPasswordAgeDays              int     `json:"max_password_age_days"`
	MinPasswordAgeDays              int     `json:"min_password_age_days"`
	PasswordHistoryCount            int     `json:"password_history_count"`
	ComplexityEnabled               bool    `json:"complexity_enabled"`
	LockoutThreshold                int     `json:"lockout_threshold"`
	LockoutDurationMinutes          int     `json:"lockout_duration_minutes"`
	LockoutObservationWindowMinutes int     `json:"lockout_observation_window_minutes"`
	CollectionError                 *string `json:"collection_error,omitempty"`
}

// NetworkInfo lists network interfaces (IP and MAC only — no traffic data).
type NetworkInfo struct {
	Interfaces      []NetworkInterface `json:"interfaces"`
	CollectionError *string            `json:"collection_error,omitempty"`
}

// NetworkInterface describes a single network adapter.
type NetworkInterface struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	MACAddress  string   `json:"mac_address"`
	IPAddresses []string `json:"ip_addresses"`
	DNSServers  []string `json:"dns_servers,omitempty"`
	IsUp        bool     `json:"is_up"`
	Type        string   `json:"type,omitempty"` // "ethernet", "wifi", "loopback"
}

// SystemHealthInfo reports uptime and last reboot.
type SystemHealthInfo struct {
	LastRebootTime  *time.Time `json:"last_reboot_time,omitempty"`
	UptimeHours     float64    `json:"uptime_hours"`
	CollectionError *string    `json:"collection_error,omitempty"`
}
