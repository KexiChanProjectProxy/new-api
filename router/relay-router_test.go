package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelaySubpathRegistration(t *testing.T) {
	// Test that subpath routes are registered alongside root routes
	tests := []struct {
		name           string
		subpaths       []string
		requestPath    string
		expectNotFound bool
	}{
		// Root routes always exist
		{"root /v1/models", nil, "/v1/models", false},
		{"root /v1/chat/completions", nil, "/v1/chat/completions", false},
		{"root /v1/messages", nil, "/v1/messages", false},

		// Subpath routes when configured
		{"subpath /a/b/v1/models", []string{"/a/b"}, "/a/b/v1/models", false},
		{"subpath /a/b/v1/chat/completions", []string{"/a/b"}, "/a/b/v1/chat/completions", false},
		{"subpath /a/b/v1/messages", []string{"/a/b"}, "/a/b/v1/messages", false},

		// UUID-style subpath
		{"uuid subpath models", []string{"/123e4567-e89b-12d3-a456-426614174000/123e4567-e89b-12d3-a456-426614174111"},
			"/123e4567-e89b-12d3-a456-426614174000/123e4567-e89b-12d3-a456-426614174111/v1/models", false},

		// Non-existent subpath route
		{"non-existent subpath", []string{"/a/b"}, "/x/y/v1/models", true},

		// Root still works when subpath is configured
		{"root exists alongside subpath", []string{"/a/b"}, "/v1/models", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			common.RelaySubpaths = tt.subpaths
			SetRelayRouter(r)

			// Use appropriate HTTP method based on path
			method := http.MethodGet
			pathLen := len(tt.requestPath)
			isPostPath := false
			// Check for POST paths: /messages or /chat/completions at the end
			if pathLen >= 9 && tt.requestPath[pathLen-9:] == "/messages" {
				isPostPath = true
			}
			if pathLen >= 17 && tt.requestPath[pathLen-17:] == "/chat/completions" {
				isPostPath = true
			}
			if isPostPath {
				method = http.MethodPost
			}

			req := httptest.NewRequest(method, tt.requestPath, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if tt.expectNotFound {
				assert.Equal(t, http.StatusNotFound, w.Code, "expected 404 for %s", tt.requestPath)
			} else {
				assert.NotEqual(t, http.StatusNotFound, w.Code, "route %s should be registered (got %d, not 404)", tt.requestPath, w.Code)
			}
		})
	}
}

func TestNoSubpathNoAlias(t *testing.T) {
	// When no subpaths configured, only root routes exist
	gin.SetMode(gin.TestMode)
	r := gin.New()
	common.RelaySubpaths = nil
	SetRelayRouter(r)

	// Root routes should exist
	for _, path := range []string{"/v1/models", "/v1/chat/completions", "/v1/messages"} {
		method := http.MethodGet
		if path != "/v1/models" {
			method = http.MethodPost
		}
		req := httptest.NewRequest(method, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusNotFound, w.Code, "root route %s should exist", path)
	}

	// Fake subpath should 404
	req := httptest.NewRequest(http.MethodGet, "/a/b/v1/models", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code, "subpath route should not exist when not configured")
}

func TestRelayV1RouteFamilyCoverage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	common.RelaySubpaths = nil
	SetRelayRouter(r)

	routes := r.Routes()

	hasRoute := func(method string, path string) bool {
		for _, route := range routes {
			if route.Method == method && route.Path == path {
				return true
			}
		}
		return false
	}

	require.True(t, hasRoute(http.MethodPost, "/v1/messages"), "Anthropic entrypoint must be registered")
	require.True(t, hasRoute(http.MethodPost, "/v1/chat/completions"), "OpenAI chat entrypoint must be registered")
	require.True(t, hasRoute(http.MethodPost, "/v1/completions"), "OpenAI completions entrypoint must be registered")

	assert.Equal(t, common.RoutingFamilyAnthropic, common.PathToRoutingFamily("/v1/messages"), "messages path should map to Anthropic family")
	assert.Equal(t, common.RoutingFamilyOpenAI, common.PathToRoutingFamily("/v1/chat/completions"), "chat completions should map to OpenAI family")
	assert.Equal(t, common.RoutingFamilyOpenAI, common.PathToRoutingFamily("/v1/completions"), "completions should map to OpenAI family")
}
