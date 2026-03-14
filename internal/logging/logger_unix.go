//go:build darwin || linux

package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger writes to a rotating file on macOS and Linux.
// (Windows logger.go writes to both Windows Event Log and a file.)
type Logger struct {
	mu       sync.Mutex
	fileLog  *log.Logger
	logFile  *os.File
	logPath  string
	maxBytes int64
}

// NewEventLogger creates a Logger that writes to stderr until a log file is configured.
// On Unix there is no OS event log, so this is equivalent to a stderr logger.
func NewEventLogger() *Logger {
	return &Logger{}
}

// NewFileLogger creates a Logger that writes to a rotating file.
func NewFileLogger(logPath string, maxSizeMB int) *Logger {
	l := &Logger{
		logPath:  logPath,
		maxBytes: int64(maxSizeMB) * 1024 * 1024,
	}

	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err == nil {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			l.logFile = f
			l.fileLog = log.New(f, "", 0)
		}
	}

	return l
}

// Close releases logger resources.
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.logFile != nil {
		l.logFile.Close()
	}
}

// Info logs an informational message.
func (l *Logger) Info(msg string) {
	l.write("INFO", msg)
}

// Warning logs a warning message.
func (l *Logger) Warning(msg string) {
	l.write("WARN", msg)
}

// Error logs an error message.
func (l *Logger) Error(msg string) {
	l.write("ERROR", msg)
}

func (l *Logger) write(level, msg string) {
	line := fmt.Sprintf("%s [%s] %s", time.Now().UTC().Format(time.RFC3339), level, msg)

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.fileLog != nil {
		l.fileLog.Println(line)
		l.rotateIfNeeded()
	} else {
		fmt.Fprintln(io.Discard, line)
	}
}

func (l *Logger) rotateIfNeeded() {
	if l.logFile == nil || l.maxBytes == 0 {
		return
	}
	info, err := l.logFile.Stat()
	if err != nil || info.Size() < l.maxBytes {
		return
	}

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
