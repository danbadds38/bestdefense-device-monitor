//go:build windows

package main

import (
	"fmt"
	"os"

	"github.com/bestdefense/bestdefense-device-monitor/internal/logging"
	"github.com/bestdefense/bestdefense-device-monitor/internal/service"
	"golang.org/x/sys/windows/svc"
)

// isWindowsService detects whether the process was started by the Windows
// Service Control Manager rather than a user running it in a terminal.
func isWindowsService() bool {
	ok, err := svc.IsWindowsService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to detect service context: %v\n", err)
		os.Exit(1)
	}
	return ok
}

// runAsService hands control to the Windows Service Control Manager.
func runAsService() {
	log := logging.NewEventLogger()
	if err := svc.Run(service.ServiceName, service.New(log)); err != nil {
		log.Error(fmt.Sprintf("Service failed: %v", err))
		os.Exit(1)
	}
}
