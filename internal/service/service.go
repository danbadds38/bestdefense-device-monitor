//go:build windows

package service

import (
	"fmt"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/collector"
	"github.com/bestdefense/bestdefense-device-monitor/internal/commander"
	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
	"github.com/bestdefense/bestdefense-device-monitor/internal/executor"
	"github.com/bestdefense/bestdefense-device-monitor/internal/logging"
	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
	"github.com/bestdefense/bestdefense-device-monitor/internal/taskresult"
	"golang.org/x/sys/windows/svc"
)

// ServiceName is the Windows Service name used for registration.
const ServiceName = "BestDefenseMonitor"

// Handler implements svc.Handler for the Windows Service lifecycle.
type Handler struct {
	log *logging.Logger
}

// New creates a Handler.
func New(log *logging.Logger) *Handler {
	return &Handler{log: log}
}

// Execute is called by the Windows Service Control Manager.
// It runs the scheduler loop and handles service control requests.
func (h *Handler) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue

	s <- svc.Status{State: svc.StartPending}

	cfg, err := config.Load()
	if err != nil {
		h.log.Error(fmt.Sprintf("Failed to load config: %v", err))
		s <- svc.Status{State: svc.Stopped}
		return false, 1
	}

	// Upgrade logger to file logger now that we have config
	fileLog := logging.NewFileLogger(cfg.LogFile, cfg.MaxLogSizeMB)
	defer fileLog.Close()
	h.log = fileLog

	h.log.Info("BestDefense Device Monitor service starting")
	s <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	interval := time.Duration(cfg.CheckIntervalHours) * time.Hour
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	paused := false

	// Run one check immediately on start
	h.runCheck(cfg)

	for {
		select {
		case <-ticker.C:
			if !paused {
				h.runCheck(cfg)
			}

		case req := <-r:
			switch req.Cmd {
			case svc.Interrogate:
				s <- req.CurrentStatus

			case svc.Stop, svc.Shutdown:
				h.log.Info("Service stopping")
				s <- svc.Status{State: svc.StopPending}
				return false, 0

			case svc.Pause:
				paused = true
				s <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				h.log.Info("Service paused")

			case svc.Continue:
				paused = false
				s <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				h.log.Info("Service resumed")
				// Run a check immediately after resume
				h.runCheck(cfg)
			}
		}
	}
}

func (h *Handler) runCheck(cfg *config.Config) {
	h.log.Info("Starting device check")
	start := time.Now()

	report := collector.Collect(cfg)

	if len(report.CheckErrors) > 0 {
		for _, ce := range report.CheckErrors {
			h.log.Warning(fmt.Sprintf("Check %q error: %s", ce.Check, ce.Error))
		}
	}

	r := reporter.New(cfg)
	if err := r.Send(report); err != nil {
		h.log.Error(fmt.Sprintf("Failed to send report: %v", err))
		return
	}

	cmdr := commander.New(cfg)
	tasks, err := cmdr.Poll()
	if err != nil {
		h.log.Warning(fmt.Sprintf("Failed to poll commands: %v", err))
	} else {
		h.log.Info(fmt.Sprintf("Polled %d pending command(s)", len(tasks)))
		if len(tasks) > 0 {
			results := executor.Run(tasks)
			poster := taskresult.New(cfg)
			if err := poster.Post(results); err != nil {
				h.log.Warning(fmt.Sprintf("Failed to post task results: %v", err))
			}
		}
	}

	h.log.Info(fmt.Sprintf("Device check completed in %.1fs, %d apps collected, %d check errors",
		time.Since(start).Seconds(),
		report.InstalledApps.TotalCount,
		len(report.CheckErrors),
	))
}
