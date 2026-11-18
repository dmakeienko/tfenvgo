package cmd

import (
	"fmt"
	"io"
	"os"
)

// LogLevel represents different log levels
type LogLevel int

const (
	LevelError LogLevel = iota
	LevelWarn
	LevelInfo
	LevelDebug
)

var currentLogLevel = LevelInfo

// SetLogLevel sets the current logging level
func SetLogLevel(level LogLevel) {
	currentLogLevel = level
}

// logMessage outputs a message with color and level prefix to w
func logMessage(w io.Writer, level LogLevel, color, prefix, message string) {
	if level > currentLogLevel {
		return
	}
	_, _ = fmt.Fprintf(w, "%s[%s]%s %s\n", color, prefix, Reset, message)
}

// LogError logs an error message to stderr
func LogError(message string, args ...interface{}) {
	logMessage(os.Stderr, LevelError, Red, "ERROR", fmt.Sprintf(message, args...))
}

// LogWarn logs a warning message to stderr
func LogWarn(message string, args ...interface{}) {
	logMessage(os.Stderr, LevelWarn, Yellow, "WARN", fmt.Sprintf(message, args...))
}

// LogInfo logs an info message
func LogInfo(message string, args ...interface{}) {
	logMessage(os.Stdout, LevelInfo, Green, "INFO", fmt.Sprintf(message, args...))
}

// LogDebug logs a debug message
func LogDebug(message string, args ...interface{}) {
	logMessage(os.Stdout, LevelDebug, Gray, "DEBUG", fmt.Sprintf(message, args...))
}

// FatalError logs an error and exits
func FatalError(message string, args ...interface{}) {
	LogError(message, args...)
	os.Exit(1)
}
