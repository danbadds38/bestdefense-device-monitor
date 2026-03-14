//go:build linux

package collector

import (
	"bufio"
	"os/exec"
	"strings"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

func collectApplications() (result reporter.InstalledAppsInfo) {
	err := safeCollect("installed_apps", func() error {
		var apps []reporter.InstalledApp

		// Debian/Ubuntu: dpkg-query
		if dpkgApps, err := collectDpkgApps(); err == nil && len(dpkgApps) > 0 {
			apps = append(apps, dpkgApps...)
		} else if rpmApps, err := collectRpmApps(); err == nil && len(rpmApps) > 0 {
			// RHEL/Fedora/CentOS: rpm
			apps = append(apps, rpmApps...)
		}

		result.Applications = apps
		result.TotalCount = len(apps)
		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}

func collectDpkgApps() ([]reporter.InstalledApp, error) {
	out, err := exec.Command(
		"dpkg-query", "-W",
		"-f=${Package}\t${Version}\t${Maintainer}\t${db:Status-Abbrev}\n",
	).Output()
	if err != nil {
		return nil, err
	}

	var apps []reporter.InstalledApp
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) < 4 {
			continue
		}
		status := strings.TrimSpace(fields[3])
		// Only include installed packages (status starts with "ii")
		if !strings.HasPrefix(status, "ii") {
			continue
		}
		apps = append(apps, reporter.InstalledApp{
			Name:      fields[0],
			Version:   fields[1],
			Publisher: fields[2],
			Source:    "dpkg",
		})
	}
	return apps, nil
}

func collectRpmApps() ([]reporter.InstalledApp, error) {
	out, err := exec.Command(
		"rpm", "-qa",
		"--queryformat", "%{NAME}\t%{VERSION}-%{RELEASE}\t%{VENDOR}\t%{INSTALLTIME:date}\n",
	).Output()
	if err != nil {
		return nil, err
	}

	var apps []reporter.InstalledApp
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) < 3 {
			continue
		}
		app := reporter.InstalledApp{
			Name:      fields[0],
			Version:   fields[1],
			Publisher: fields[2],
			Source:    "rpm",
		}
		if len(fields) >= 4 {
			app.InstallDate = fields[3]
		}
		apps = append(apps, app)
	}
	return apps, nil
}
