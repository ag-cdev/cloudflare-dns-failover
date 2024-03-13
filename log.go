package main

import (
	"fmt"
	"time"
)

type customLogWriter struct{}

func (writer customLogWriter) Write(bytes []byte) (int, error) {
	// Output the log message directly without adding a timestamp
	return fmt.Print(string(bytes))
}

type logEntry struct {
	domain    string
	msg       string
	timestamp time.Time
}

func sendLogEntry(logCh chan<- logEntry, domain, message string) {
	logCh <- logEntry{
		domain:    domain,
		msg:       message,
		timestamp: time.Now(),
	}
}
