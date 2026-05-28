package common

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

// OutboundClientIP returns the gateway-controlled outbound client IP value
// suitable for use as an X-Forwarded-For header value sent to upstream providers.
//
// Behavior:
//   - When forwarding is disabled for the current channel, returns "" (no mutation).
//   - When enabled, derives a normalized single IP string from c.ClientIP(),
//     which is trusted-proxy-aware after SetTrustedProxies is configured (Task 4).
//   - The returned value is stripped of whitespace and validated as a valid IP.
//   - This function does NOT read or forward the raw inbound XFF chain.
//   - This function does NOT derive outbound values from untrusted client header text.
func OutboundClientIP(c *gin.Context, enabled bool) string {
	if c == nil || !enabled {
		return ""
	}

	clientIP := c.ClientIP()
	if clientIP == "" {
		return ""
	}

	// Normalize: trim whitespace
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		return ""
	}

	// Validate it's a valid IP before returning
	if net.ParseIP(clientIP) == nil {
		return ""
	}

	return clientIP
}

// OutboundXFFValue is a convenience alias for OutboundClientIP.
// It returns the normalized single client IP to be used as X-Forwarded-For value.
func OutboundXFFValue(c *gin.Context, enabled bool) string {
	return OutboundClientIP(c, enabled)
}
