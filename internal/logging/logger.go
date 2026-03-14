//go:build windows

package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/windows/svc/eventlog"
)

const sourceName = "BestDefenseMonitor"

// Logger writes to both the Windows Event Log and a rotating file.
type Logger struct {
	mu       sync.Mutex
	evLog    *eventlog.Log
	fileLog  *log.Logger
	logFile  *os.File
	logPath  string
	maxBytes int64
}

// NewEventLogger creates a Logger using the Windows Event Log source.
// Falls back to stderr if the event log source is not registered.
func NewEventLogger() *Logger {
	el, _ := eventlog.Open(sourceName) // may be nil if not registered yet
	return &Logger{
		evLog: el,
	}
}

// NewFileLogger creates a Logger that writes to a file (and Event Log if available).
func NewFileLogger(logPath string, maxSizeMB int) *Logger {
	l := NewEventLogger()
	l.logPath = logPath
	l.maxBytes = int64(maxSizeMB) * 1024 * 1024

	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err == nil {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			l.logFile = f
			l.fileLog = log.New(f, "", 0)
		}
	}

	return l
}

// Close releases the logger resources.
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.evLog != nil {
		l.evLog.Close()
	}
	if l.logFile != nil {
		l.logFile.Close()
	}
}

// Info logs an informational message.
func (l *Logger) Info(msg string) {
	l.write("INFO", msg)
	if l.evLog != nil {
		l.evLog.Info(1, msg)
	}
}

// Warning logs a warning message.
func (l *Logger) Warning(msg string) {
	l.write("WARN", msg)
	if l.evLog != nil {
		l.evLog.Warning(2, msg)
	}
}

// Error logs an error message.
func (l *Logger) Error(msg string) {
	l.write("ERROR", msg)
	if l.evLog != nil {
		l.evLog.Error(3, msg)
	}
}

func (l *Logger) write(level, msg string) {
	line := fmt.Sprintf("%s [%s] %s", time.Now().UTC().Format(time.RFC3339), level, msg)

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.fileLog != nil {
		l.fileLog.Println(line)
		l.rotatIfNeeded()
	} else {
		fmt.Fprintln(io.Discard, line) // service context: discard if no file
	}
}

func (l *Logger) rotatIfNeeded() {
	if l.logFile == nil || l.maxBytes == 0 {
		return
	}
	info, err := l.logFile.Stat()
	if err != nil || info.Size() < l.maxBytes {
		return
	}

	// Rotate: close current file, rename to .1, open new file
	l.logFile.Close()
	rotPath := l.logPath + ".1"
	os.Rename(l.logPath, rotPath)

	f, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	l.logFile = f
	l.fileLog = log.New(f, "", 0)
}
