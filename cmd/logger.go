package cmd

import (
	"fmt"
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

// logMessage outputs a message with color and level prefix
func logMessage(level LogLevel, color, prefix, message string) {
	if level > currentLogLevel {
		return
	}
	fmt.Printf("%s[%s]%s %s\n", color, prefix, Reset, message)
}

// LogError logs an error message
func LogError(message string, args ...interface{}) {
	logMessage(LevelError, Red, "ERROR", fmt.Sprintf(message, args...))
}

// LogWarn logs a warning message
func LogWarn(message string, args ...interface{}) {
	logMessage(LevelWarn, Yellow, "WARN", fmt.Sprintf(message, args...))
}

// LogInfo logs an info message
func LogInfo(message string, args ...interface{}) {
	logMessage(LevelInfo, Green, "INFO", fmt.Sprintf(message, args...))
}

// LogDebug logs a debug message
func LogDebug(message string, args ...interface{}) {
	logMessage(LevelDebug, Gray, "DEBUG", fmt.Sprintf(message, args...))
}

// FatalError logs an error and exits
func FatalError(message string, args ...interface{}) {
	LogError(message, args...)
	os.Exit(1)
}
