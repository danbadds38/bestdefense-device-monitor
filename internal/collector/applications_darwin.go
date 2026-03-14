//go:build darwin

package collector

import (
	"encoding/json"
	"os/exec"

	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
)

// spApplicationsOutput is the JSON structure returned by
// system_profiler SPApplicationsDataType -json
type spApplicationsOutput struct {
	SPApplicationsDataType []spAppItem `json:"SPApplicationsDataType"`
}

type spAppItem struct {
	Name        string `json:"_name"`
	Version     string `json:"version"`
	ObtainedFrom string `json:"obtained_from"`
	Path        string `json:"path"`
}

func collectApplications() (result reporter.InstalledAppsInfo) {
	err := safeCollect("applications", func() error {
		// system_profiler SPApplicationsDataType -json can be slow (~5-10s).
		// It scans /Applications and user ~/Applications directories.
		out, err := exec.Command("system_profiler", "SPApplicationsDataType", "-json").Output()
		if err != nil {
			return err
		}

		var sp spApplicationsOutput
		if err := json.Unmarshal(out, &sp); err != nil {
			return err
		}

		for _, item := range sp.SPApplicationsDataType {
			if item.Name == "" {
				continue
			}
			app := reporter.InstalledApp{
				Name:            item.Name,
				Version:         item.Version,
				InstallLocation: item.Path,
				Source:          item.ObtainedFrom,
			}
			if app.Source == "" {
				app.Source = "/Applications"
			}
			result.Applications = append(result.Applications, app)
		}

		result.TotalCount = len(result.Applications)
		return nil
	})
	result.CollectionError = errPtr(err)
	return result
}
