//go:build linux

package service

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
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
)

// shortPollInterval is the re-poll delay used after an execute_script task.
// It lets the server pick up dry-run results quickly before the next full check.
const shortPollInterval = 2 * time.Minute

// ServiceName is the systemd unit name used for this agent.
const ServiceName = "bestdefense-monitor"

// Handler implements the Linux daemon run loop.
type Handler struct {
	log *logging.Logger
}

// New creates a Handler.
func New(log *logging.Logger) *Handler {
	return &Handler{log: log}
}

// Run starts the check scheduler and blocks until SIGTERM or SIGINT is received.
func (h *Handler) Run() {
	cfg, err := config.Load()
	if err != nil {
		h.log.Error(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}

	fileLog := logging.NewFileLogger(cfg.LogFile, cfg.MaxLogSizeMB)
	defer fileLog.Close()
	h.log = fileLog

	h.log.Info("BestDefense Device Monitor daemon starting")

	kp, err := identity.LoadOrGenerate()
	if err != nil {
		h.log.Error(fmt.Sprintf("Failed to load identity key: %v", err))
		os.Exit(1)
	}
	cfg.PublicKeyBase64 = kp.PublicKeyBase64()
	h.log.Info("Identity key loaded")

	interval := time.Duration(cfg.CheckIntervalHours) * time.Hour
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)

	executor.SetRotateKeysFunc(keyrotation.New(cfg).Rotate)

	if sp := h.runCheck(cfg, kp); sp {
		ticker.Reset(shortPollInterval)
	}

	for {
		select {
		case <-ticker.C:
			if sp := h.runCheck(cfg, kp); sp {
				h.log.Info(fmt.Sprintf("execute_script task processed; re-polling in %.0fs", shortPollInterval.Seconds()))
				ticker.Reset(shortPollInterval)
			} else {
				ticker.Reset(interval)
			}
		case sig := <-sigs:
			h.log.Info(fmt.Sprintf("Received signal %s, stopping", sig))
			return
		}
	}
}

// runCheck performs one collection + report + command poll cycle.
// Returns true if at least one execute_script task was processed (shortPoll).
func (h *Handler) runCheck(cfg *config.Config, kp *identity.KeyPair) bool {
	h.log.Info("Starting device check")
	start := time.Now()

	report := collector.Collect(cfg)

	for _, ce := range report.CheckErrors {
		h.log.Warning(fmt.Sprintf("Check %q error: %s", ce.Check, ce.Error))
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
