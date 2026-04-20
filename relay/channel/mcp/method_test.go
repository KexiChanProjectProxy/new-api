package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBillableMethod_NonBillableMethods(t *testing.T) {
	t.Helper()

	// Test all non-billable methods return false
	nonBillableMethods := []string{
		MethodInitialize,
		MethodPing,
		MethodNotificationsInitialized,
		MethodNotificationsCancelled,
	}

	for _, method := range nonBillableMethods {
		t.Run(method, func(t *testing.T) {
			result := IsBillableMethod(method)
			assert.False(t, result, "IsBillableMethod(%q) should return false", method)
		})
	}
}

func TestIsBillableMethod_BillableMethods(t *testing.T) {
	t.Helper()

	// Test all billable methods return true
	billableMethods := []string{
		MethodToolsCall,
		MethodResourcesRead,
		MethodResourcesSubscribe,
		MethodResourcesUnsubscribe,
		MethodPromptsGet,
		MethodCompletionComplete,
		MethodSamplingCreateMessage,
		MethodRootsList,
	}

	for _, method := range billableMethods {
		t.Run(method, func(t *testing.T) {
			result := IsBillableMethod(method)
			assert.True(t, result, "IsBillableMethod(%q) should return true", method)
		})
	}
}

func TestIsBillableMethod_UnknownMethodDefaultsToBillable(t *testing.T) {
	t.Helper()

	unknownMethods := []string{
		"unknown/method",
		"something/entirely/different",
		"random",
	}

	for _, method := range unknownMethods {
		t.Run(method, func(t *testing.T) {
			result := IsBillableMethod(method)
			assert.True(t, result, "IsBillableMethod(%q) should return true (unknown methods default to billable)", method)
		})
	}
}

func TestIsBillableMethod_EmptyMethodIsNonBillable(t *testing.T) {
	t.Helper()

	result := IsBillableMethod("")
	assert.False(t, result, "IsBillableMethod(\"\") should return false (empty method is non-billable)")
}

func TestExtractMCPMethod_ValidJSONRPC(t *testing.T) {
	t.Helper()

	testCases := []struct {
		name     string
		body     []byte
		expected string
	}{
		{
			name:     "tools/call method",
			body:     []byte(`{"jsonrpc":"2.0","method":"tools/call","id":1}`),
			expected: "tools/call",
		},
		{
			name:     "initialize method",
			body:     []byte(`{"jsonrpc":"2.0","method":"initialize","id":1}`),
			expected: "initialize",
		},
		{
			name:     "ping method",
			body:     []byte(`{"jsonrpc":"2.0","method":"ping","id":1}`),
			expected: "ping",
		},
		{
			name:     "notifications/initialized method",
			body:     []byte(`{"jsonrpc":"2.0","method":"notifications/initialized","id":null}`),
			expected: "notifications/initialized",
		},
		{
			name:     "resources/read method",
			body:     []byte(`{"jsonrpc":"2.0","method":"resources/read","params":{"uri":"file://test"},"id":2}`),
			expected: "resources/read",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractMCPMethod(tc.body)
			assert.Equal(t, tc.expected, result, "ExtractMCPMethod should extract the correct method")
		})
	}
}

func TestExtractMCPMethod_EmptyBody(t *testing.T) {
	t.Helper()

	result := ExtractMCPMethod(nil)
	assert.Equal(t, "", result, "ExtractMCPMethod(nil) should return empty string")

	result = ExtractMCPMethod([]byte{})
	assert.Equal(t, "", result, "ExtractMCPMethod([]byte{}) should return empty string")
}

func TestExtractMCPMethod_InvalidJSON(t *testing.T) {
	t.Helper()

	invalidBodies := [][]byte{
		[]byte("not json at all"),
		[]byte(`{"jsonrpc":"2.0"`),
		[]byte(`[1,2,3]`),
		[]byte(`"just a string"`),
		[]byte(`123`),
		[]byte(`{}`),
		[]byte(`{"jsonrpc":"2.0","params":{}}`),
	}

	for _, body := range invalidBodies {
		result := ExtractMCPMethod(body)
		assert.Equal(t, "", result, "ExtractMCPMethod(%q) should return empty string for invalid JSON", string(body))
	}
}