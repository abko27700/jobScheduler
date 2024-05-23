package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var logMutex sync.Mutex

func clearLogFile(filename string) error {
	// Open the file with O_TRUNC flag to truncate the file to zero length
	file, err := os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to clear log file: %w", err)
	}
	defer file.Close()
	return nil
}

func log(callerMethod string, msg string) {
	// Get the current time
	now := time.Now()

	// Format the time as [hours:minutes:seconds:milliseconds]
	timeFormat := now.Format("2006-01-02 15:04:05.000")

	// Acquire the lock
	logMutex.Lock()
	defer logMutex.Unlock()

	// Print the log message
	file, err := os.OpenFile("logfile.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening or creating log file: %v\n", err)
		return
	}
	defer file.Close()

	// Create the log message
	logMessage := fmt.Sprintf("[%s]: %s: %s\n", timeFormat, callerMethod, msg)

	// Write the log message to the file
	if _, err := file.WriteString(logMessage); err != nil {
		fmt.Printf("Error writing to log file: %v\n", err)
	}
}

func endLog(callerMethod string, startTime time.Time) {
	elapsedTime := time.Since(startTime)

	// Convert to seconds
	seconds := elapsedTime.Seconds()

	// Convert to milliseconds
	milliseconds := elapsedTime.Milliseconds()

	// Get elapsed time in microseconds
	microseconds := elapsedTime.Microseconds()

	// Log the elapsed time in the desired format
	log(callerMethod, fmt.Sprintf("Exiting Method. Time Taken: %.0f.%03d.%06d", seconds, milliseconds, microseconds))
}
