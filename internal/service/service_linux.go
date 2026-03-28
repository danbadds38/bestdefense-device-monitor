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
	"github.com/bestdefense/bestdefense-device-monitor/internal/logging"
	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
	"github.com/bestdefense/bestdefense-device-monitor/internal/taskresult"
)

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

	h.runCheck(cfg, kp)

	for {
		select {
		case <-ticker.C:
			h.runCheck(cfg, kp)
		case sig := <-sigs:
			h.log.Info(fmt.Sprintf("Received signal %s, stopping", sig))
			return
		}
	}
}

func (h *Handler) runCheck(cfg *config.Config, kp *identity.KeyPair) {
	h.log.Info("Starting device check")
	start := time.Now()

	report := collector.Collect(cfg)

	for _, ce := range report.CheckErrors {
		h.log.Warning(fmt.Sprintf("Check %q error: %s", ce.Check, ce.Error))
	}

	r := reporter.New(cfg).WithKeyPair(kp)
	if err := r.Send(report); err != nil {
		h.log.Error(fmt.Sprintf("Failed to send report: %v", err))
		return
	}

	cmdr := commander.New(cfg).WithKeyPair(kp)
	tasks, err := cmdr.Poll()
	if err != nil {
		h.log.Warning(fmt.Sprintf("Failed to poll commands: %v", err))
	} else {
		h.log.Info(fmt.Sprintf("Polled %d pending command(s)", len(tasks)))
		if len(tasks) > 0 {
			results := executor.Run(tasks)
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
}
