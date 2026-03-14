//go:build darwin || linux

package main

import (
	"github.com/bestdefense/bestdefense-device-monitor/internal/logging"
	"github.com/bestdefense/bestdefense-device-monitor/internal/service"
)

// isWindowsService always returns false on Unix — launchd/systemd invoke
// the binary directly with the "run" argument instead of using a service API.
func isWindowsService() bool {
	return false
}

// runAsService starts the Unix daemon run loop.
// On macOS this is called by launchd; on Linux by systemd.
func runAsService() {
	log := logging.NewEventLogger()
	service.New(log).Run()
}
