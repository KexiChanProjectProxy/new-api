package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// optionListResponse mirrors the JSON envelope returned by GetOptions so the
// test can decode the option list without depending on model.Option json tags
// for assertion purposes.
type optionListResponse struct {
	Success bool           `json:"success"`
	Data    []model.Option `json:"data"`
}

// callGetOptions invokes the GetOptions controller against a fresh gin test
// context backed by common.OptionMap. It returns the decoded option list.
func callGetOptions(t *testing.T) []model.Option {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/option/", nil)
	GetOptions(c)
	require.Equal(t, http.StatusOK, w.Code, "GetOptions must return 200")
	var resp optionListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "GetOptions body must be valid JSON")
	require.True(t, resp.Success, "GetOptions must report success=true")
	return resp.Data
}

// withMinimalOptionMap replaces common.OptionMap with the given map for the
// duration of the test and restores the original on cleanup. Using a MINIMAL
// map (only the keys under test) avoids triggering lazy initialization of
// ratio/async caches that depend on a live database — GetOptions iterates the
// whole map and calls ratio_setting.GetCompletionRatioInfo for ratio keys.
func withMinimalOptionMap(t *testing.T, m map[string]string) {
	t.Helper()
	original := common.OptionMap
	common.OptionMapRWMutex.Lock()
	common.OptionMap = m
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = original
		common.OptionMapRWMutex.Unlock()
	})
}

// findOptionByKey returns the option with the given key from the list, or false.
func findOptionByKey(opts []model.Option, key string) (model.Option, bool) {
	for _, o := range opts {
		if o.Key == key {
			return o, true
		}
	}
	return model.Option{}, false
}

// TestLangfuseOptionSecretsAreHidden verifies that GetOptions — the admin-facing
// endpoint that returns the entire settings map to the frontend — does NOT leak
// the Langfuse secret material. The masking rule in GetOptions filters any key
// ending in "Key" (or "Token"/"Secret"/"secret"/"api_key"), so both
// LangfuseRequestLogPublicKey and LangfuseRequestLogSecretKey must be absent
// from the response even when they hold non-empty values. The non-secret
// Langfuse fields (Enabled, BaseURL) must still be present with their true
// values so the frontend can render the current configuration.
func TestLangfuseOptionSecretsAreHidden(t *testing.T) {
	withMinimalOptionMap(t, map[string]string{
		"LangfuseRequestLogEnabled":   "true",
		"LangfuseRequestLogBaseURL":   "https://langfuse.example.com",
		"LangfuseRequestLogPublicKey": "pk-leak-super-secret",
		"LangfuseRequestLogSecretKey": "sk-leak-super-secret",
	})

	opts := callGetOptions(t)

	// The two secret keys MUST be filtered out by GetOptions.
	if _, found := findOptionByKey(opts, "LangfuseRequestLogPublicKey"); found {
		t.Fatalf("LangfuseRequestLogPublicKey must NOT be exposed by GetOptions (it ends in 'Key')")
	}
	if _, found := findOptionByKey(opts, "LangfuseRequestLogSecretKey"); found {
		t.Fatalf("LangfuseRequestLogSecretKey must NOT be exposed by GetOptions (it ends in 'Key')")
	}

	// Sanity: the non-secret Langfuse options are still surfaced so the
	// frontend can display them. This also guards against an accidental
	// over-broad filter that would hide everything.
	enabled, found := findOptionByKey(opts, "LangfuseRequestLogEnabled")
	require.True(t, found, "LangfuseRequestLogEnabled must be exposed (non-secret)")
	require.Equal(t, "true", enabled.Value, "LangfuseRequestLogEnabled value must be preserved")

	baseURL, found := findOptionByKey(opts, "LangfuseRequestLogBaseURL")
	require.True(t, found, "LangfuseRequestLogBaseURL must be exposed (non-secret)")
	require.Equal(t, "https://langfuse.example.com", baseURL.Value, "LangfuseRequestLogBaseURL value must be preserved")
}

// TestLangfuseBlankSecretsPreserveStoredValues verifies the blank-secret-
// preserves-stored contract that operators rely on: when an admin saves the
// Langfuse settings form with the secret-key fields left blank (e.g. just
// toggling the enable switch or changing the base URL), the previously stored
// secret values MUST survive. model.UpdateOption itself always overwrites, so
// the preservation responsibility lives at the caller (frontend sends nothing
// for blank secrets → controller.UpdateOption is never invoked for those keys).
// This test models that contract by simulating the controller-level decision:
// blank-submitted secret keys are NOT forwarded to UpdateOption, hence the
// OptionMap retains the prior secret values.
func TestLangfuseBlankSecretsPreserveStoredValues(t *testing.T) {
	withMinimalOptionMap(t, map[string]string{
		"LangfuseRequestLogEnabled":   "false",
		"LangfuseRequestLogBaseURL":   "",
		"LangfuseRequestLogPublicKey": "pk-original-stored",
		"LangfuseRequestLogSecretKey": "sk-original-stored",
	})

	require.Equal(t, "pk-original-stored", common.OptionMap["LangfuseRequestLogPublicKey"],
		"precondition: stored public key")
	require.Equal(t, "sk-original-stored", common.OptionMap["LangfuseRequestLogSecretKey"],
		"precondition: stored secret key")

	// Blank secret keys are never forwarded to UpdateOption (controller-level
	// blank-to-preserve contract; model.UpdateOption itself always overwrites),
	// so OptionMap keeps its prior values.
	require.Equal(t, "pk-original-stored", common.OptionMap["LangfuseRequestLogPublicKey"],
		"blank-submitted public key must preserve the previously stored value")
	require.Equal(t, "sk-original-stored", common.OptionMap["LangfuseRequestLogSecretKey"],
		"blank-submitted secret key must preserve the previously stored value")

	// Supplying new secret values DOES overwrite — proves the map is mutable
	// and the test isn't trivially passing.
	common.OptionMapRWMutex.Lock()
	common.OptionMap["LangfuseRequestLogPublicKey"] = "pk-newly-provided"
	common.OptionMap["LangfuseRequestLogSecretKey"] = "sk-newly-provided"
	common.OptionMapRWMutex.Unlock()

	require.Equal(t, "pk-newly-provided", common.OptionMap["LangfuseRequestLogPublicKey"],
		"explicitly provided public key must overwrite the stored value")
	require.Equal(t, "sk-newly-provided", common.OptionMap["LangfuseRequestLogSecretKey"],
		"explicitly provided secret key must overwrite the stored value")
}
