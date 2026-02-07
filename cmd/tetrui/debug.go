package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	debugEnabled bool
	debugMu      sync.Mutex
	debugFile    *os.File
)

func EnableDebugLogging(enabled bool) {
	debugEnabled = enabled
}

func DebugLogf(format string, args ...any) {
	if !debugEnabled {
		return
	}
	debugMu.Lock()
	defer debugMu.Unlock()
	if debugFile == nil {
		path := filepath.Join(os.TempDir(), "tetrui-debug.log")
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return
		}
		debugFile = file
	}
	timestamp := time.Now().Format(time.RFC3339)
	message := fmt.Sprintf(format, args...)
	message = strings.ReplaceAll(message, "\n", " ")
	_, _ = fmt.Fprintf(debugFile, "%s %s\n", timestamp, message)
}
