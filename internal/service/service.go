//go:build windows

package service

import (
	"fmt"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/collector"
	"github.com/bestdefense/bestdefense-device-monitor/internal/commander"
	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
	"github.com/bestdefense/bestdefense-device-monitor/internal/executor"
	"github.com/bestdefense/bestdefense-device-monitor/internal/identity"
	"github.com/bestdefense/bestdefense-device-monitor/internal/keyrotation"
	"github.com/bestdefense/bestdefense-device-monitor/internal/logging"
	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
	"github.com/bestdefense/bestdefense-device-monitor/internal/taskresult"
	"golang.org/x/sys/windows/svc"
)

// shortPollInterval is the re-poll delay used after an execute_script task.
// It lets the server pick up dry-run results quickly before the next full check.
const shortPollInterval = 2 * time.Minute

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
func (h *Handler) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue

	s <- svc.Status{State: svc.StartPending}

	cfg, err := config.Load()
	if err != nil {
		h.log.Error(fmt.Sprintf("Failed to load config: %v", err))
		s <- svc.Status{State: svc.Stopped}
		return false, 1
	}

	fileLog := logging.NewFileLogger(cfg.LogFile, cfg.MaxLogSizeMB)
	defer fileLog.Close()
	h.log = fileLog

	h.log.Info("BestDefense Device Monitor service starting")

	kp, err := identity.LoadOrGenerate()
	if err != nil {
		h.log.Error(fmt.Sprintf("Failed to load identity key: %v", err))
		s <- svc.Status{State: svc.Stopped}
		return false, 1
	}
	cfg.PublicKeyBase64 = kp.PublicKeyBase64()
	h.log.Info("Identity key loaded")

	s <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	executor.SetRotateKeysFunc(keyrotation.New(cfg).Rotate)

	interval := time.Duration(cfg.CheckIntervalHours) * time.Hour
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	paused := false

	if sp := h.runCheck(cfg, kp); sp {
		ticker.Reset(shortPollInterval)
	}

	for {
		select {
		case <-ticker.C:
			if !paused {
				if sp := h.runCheck(cfg, kp); sp {
					h.log.Info(fmt.Sprintf("execute_script task processed; re-polling in %.0fs", shortPollInterval.Seconds()))
					ticker.Reset(shortPollInterval)
				} else {
					ticker.Reset(interval)
				}
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
				if sp := h.runCheck(cfg, kp); sp {
					ticker.Reset(shortPollInterval)
				} else {
					ticker.Reset(interval)
				}
			}
		}
	}
}

// runCheck performs one collection + report + command poll cycle.
// Returns true if at least one execute_script task was processed (shortPoll).
func (h *Handler) runCheck(cfg *config.Config, kp *identity.KeyPair) bool {
	h.log.Info("Starting device check")
	start := time.Now()

	report := collector.Collect(cfg)

	if len(report.CheckErrors) > 0 {
		for _, ce := range report.CheckErrors {
			h.log.Warning(fmt.Sprintf("Check %q error: %s", ce.Check, ce.Error))
		}
	}

	r := reporter.New(cfg).WithKeyPair(kp)
	if err := r.Send(report); err != nil {
		h.log.Error(fmt.Sprintf("Failed to send report: %v", err))
		return false
	}

	shortPoll := false
	cmdr := commander.New(cfg).WithKeyPair(kp)
	tasks, err := cmdr.Poll()
	if err != nil {
		h.log.Warning(fmt.Sprintf("Failed to poll commands: %v", err))
	} else {
		h.log.Info(fmt.Sprintf("Polled %d pending command(s)", len(tasks)))
		if len(tasks) > 0 {
			results, sp := executor.Run(tasks, kp)
			shortPoll = sp
			poster := taskresult.New(cfg).WithKeyPair(kp)
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
	return shortPoll
}
