package mcp

import (
	"github.com/QuantumNous/new-api/common"
)

var ChannelName = "MCP"

// MCP method constants
const (
	// Non-billable methods (handshake/lifecycle)
	MethodInitialize             = "initialize"
	MethodPing                   = "ping"
	MethodNotificationsInitialized = "notifications/initialized"
	MethodNotificationsCancelled = "notifications/cancelled"

	// Billable methods (business calls)
	MethodToolsCall              = "tools/call"
	MethodResourcesRead          = "resources/read"
	MethodResourcesSubscribe     = "resources/subscribe"
	MethodResourcesUnsubscribe   = "resources/unsubscribe"
	MethodPromptsGet             = "prompts/get"
	MethodCompletionComplete     = "completion/complete"
	MethodSamplingCreateMessage  = "sampling/createMessage"
	MethodRootsList              = "roots/list"
)

// nonBillableMethods is a map for O(1) lookup of non-billable methods
var nonBillableMethods = map[string]bool{
	MethodInitialize:             true,
	MethodPing:                   true,
	MethodNotificationsInitialized: true,
	MethodNotificationsCancelled: true,
}

// billableMethods is a map for O(1) lookup of known billable methods
var billableMethods = map[string]bool{
	MethodToolsCall:             true,
	MethodResourcesRead:         true,
	MethodResourcesSubscribe:    true,
	MethodResourcesUnsubscribe:  true,
	MethodPromptsGet:            true,
	MethodCompletionComplete:    true,
	MethodSamplingCreateMessage: true,
	MethodRootsList:             true,
}

// IsBillableMethod returns true if the MCP method is billable.
// Non-billable methods are handshake/lifecycle methods (initialize, ping, notifications).
// Unknown methods default to billable (safer default - charge rather than give free rides).
func IsBillableMethod(method string) bool {
	if method == "" {
		return false
	}
	if nonBillableMethods[method] {
		return false
	}
	if billableMethods[method] {
		return true
	}
	// Unknown methods default to billable
	return true
}

// jsonRpcRequest is a minimal struct for extracting the method field from JSON-RPC requests
type jsonRpcRequest struct {
	Method string `json:"method"`
}

// ExtractMCPMethod extracts the JSON-RPC "method" field from a request body.
// Returns empty string if the method field cannot be extracted.
func ExtractMCPMethod(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var req jsonRpcRequest
	if err := common.Unmarshal(body, &req); err != nil {
		return ""
	}
	return req.Method
}