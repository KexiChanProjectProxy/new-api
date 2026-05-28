package common

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// LogWriterMu protects concurrent access to gin.DefaultWriter/gin.DefaultErrorWriter
// during log file rotation. Acquire RLock when reading/writing through the writers,
// acquire Lock when swapping writers and closing old files.
var LogWriterMu sync.RWMutex

func SysLog(s string) {
	t := time.Now()
	LogWriterMu.RLock()
	_, _ = fmt.Fprintf(gin.DefaultWriter, "[SYS] %v | %s \n", t.Format("2006/01/02 - 15:04:05"), s)
	LogWriterMu.RUnlock()
}

func SysError(s string) {
	t := time.Now()
	LogWriterMu.RLock()
	_, _ = fmt.Fprintf(gin.DefaultErrorWriter, "[SYS] %v | %s \n", t.Format("2006/01/02 - 15:04:05"), s)
	LogWriterMu.RUnlock()
}

func FatalLog(v ...any) {
	t := time.Now()
	LogWriterMu.RLock()
	_, _ = fmt.Fprintf(gin.DefaultErrorWriter, "[FATAL] %v | %v \n", t.Format("2006/01/02 - 15:04:05"), v)
	LogWriterMu.RUnlock()
	os.Exit(1)
}

func LogStartupSuccess(startTime time.Time, bindAddress string) {
	duration := time.Since(startTime)
	durationMs := duration.Milliseconds()

	// Get network IPs
	networkIps := GetNetworkIps()

	// Parse bind address for display
	displayHost, displayPort := FormatBindAddressForDisplay(bindAddress)

	LogWriterMu.RLock()
	defer LogWriterMu.RUnlock()

	fmt.Fprintf(gin.DefaultWriter, "\n")
	fmt.Fprintf(gin.DefaultWriter, "  \033[32m%s %s\033[0m  ready in %d ms\n", SystemName, Version, durationMs)
	fmt.Fprintf(gin.DefaultWriter, "\n")

	if !IsRunningInContainer() {
		// For local display, use localhost if bind is wildcard, otherwise use the actual host
		localHost := displayHost
		if IsAnyAddress(localHost) {
			localHost = "localhost"
		}
		// Format URL correctly for IPv6 hosts (wrap in brackets)
		var localURL string
		if IsIPv6Host(localHost) {
			localURL = fmt.Sprintf("http://[%s]:%s/", localHost, displayPort)
		} else {
			localURL = fmt.Sprintf("http://%s:%s/", localHost, displayPort)
		}
		fmt.Fprintf(gin.DefaultWriter, "  ➜  \033[1mLocal:\033[0m   %s\n", localURL)
	}

	for _, ip := range networkIps {
		fmt.Fprintf(gin.DefaultWriter, "  ➜  \033[1mNetwork:\033[0m http://%s:%s/\n", ip, displayPort)
	}

	fmt.Fprintf(gin.DefaultWriter, "\n")
}
