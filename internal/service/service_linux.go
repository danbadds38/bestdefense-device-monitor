//go:build linux

package service

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bestdefense/bestdefense-device-monitor/internal/collector"
	"github.com/bestdefense/bestdefense-device-monitor/internal/config"
	"github.com/bestdefense/bestdefense-device-monitor/internal/logging"
	"github.com/bestdefense/bestdefense-device-monitor/internal/reporter"
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
// This is called when the binary is invoked with the "run" argument by systemd.
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

	interval := time.Duration(cfg.CheckIntervalHours) * time.Hour
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)

	// Run one check immediately on start
	h.runCheck(cfg)

	for {
		select {
		case <-ticker.C:
			h.runCheck(cfg)
		case sig := <-sigs:
			h.log.Info(fmt.Sprintf("Received signal %s, stopping", sig))
			return
		}
	}
}

func (h *Handler) runCheck(cfg *config.Config) {
	h.log.Info("Starting device check")
	start := time.Now()

	report := collector.Collect(cfg)

	for _, ce := range report.CheckErrors {
		h.log.Warning(fmt.Sprintf("Check %q error: %s", ce.Check, ce.Error))
	}

	r := reporter.New(cfg)
	if err := r.Send(report); err != nil {
		h.log.Error(fmt.Sprintf("Failed to send report: %v", err))
		return
	}

	h.log.Info(fmt.Sprintf("Device check completed in %.1fs, %d apps collected, %d check errors",
		time.Since(start).Seconds(),
		report.InstalledApps.TotalCount,
		len(report.CheckErrors),
	))
}
