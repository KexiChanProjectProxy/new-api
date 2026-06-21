package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// confirmSubscriptionPaymentComplianceForTest enables payment compliance for test duration.
func confirmSubscriptionPaymentComplianceForTest(t *testing.T) {
	t.Helper()
	ps := operation_setting.GetPaymentSetting()
	origConfirmed := ps.ComplianceConfirmed
	origTerms := ps.ComplianceTermsVersion
	t.Cleanup(func() {
		ps.ComplianceConfirmed = origConfirmed
		ps.ComplianceTermsVersion = origTerms
	})
	ps.ComplianceConfirmed = true
	ps.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
}

// setupSubscriptionControllerTestDB creates an in-memory SQLite DB with required tables.
func setupSubscriptionControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "open in-memory SQLite")

	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(&model.SubscriptionPlan{}, &model.UserSubscription{}, &model.User{}), "AutoMigrate")

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

// newSubscriptionAdminContext creates a gin context for testing admin subscription handlers.
func newSubscriptionAdminContext(t *testing.T, method string, target string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	var requestBody *bytes.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		require.NoError(t, err, "marshal request body")
		requestBody = bytes.NewReader(payload)
	} else {
		requestBody = bytes.NewReader(nil)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, requestBody)
	if body != nil {
		ctx.Request.Header.Set("Content-Type", "application/json")
	}
	return ctx, recorder
}

// subscriptionAPIResponse is a helper type for decoding API responses.
type subscriptionAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// TestAdminCreateSubscriptionPlanQuotaWindows tests validation+normalization on plan creation.
func TestAdminCreateSubscriptionPlanQuotaWindows(t *testing.T) {
	setupSubscriptionControllerTestDB(t)
	confirmSubscriptionPaymentComplianceForTest(t)

	t.Run("valid windowed plan is created successfully", func(t *testing.T) {
		windows := []model.QuotaWindow{
			{Name: "5H", DurationSeconds: 5 * 3600, Quota: 1000},
			{Name: "7D", DurationSeconds: 7 * 86400, Quota: 5000},
		}
		reqBody := AdminUpsertSubscriptionPlanRequest{
			Plan: model.SubscriptionPlan{
				Title:         "Windowed Plan",
				PriceAmount:   9.99,
				DurationUnit:  model.SubscriptionDurationMonth,
				DurationValue: 1,
				TotalAmount:   10000,
				QuotaWindows:  model.NewQuotaWindowList(windows),
			},
		}

		ctx, recorder := newSubscriptionAdminContext(t, http.MethodPost, "/subscription/admin/plans", reqBody)
		AdminCreateSubscriptionPlan(ctx)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var resp subscriptionAPIResponse
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
		assert.True(t, resp.Success, "expected success, got message: %s", resp.Message)

		// Verify the plan was saved with quota_windows
		var plan model.SubscriptionPlan
		require.NoError(t, model.DB.First(&plan).Error)
		savedWindows := plan.QuotaWindows.Slice()
		assert.Len(t, savedWindows, 2)
		assert.Equal(t, "5H", savedWindows[0].Name)
		assert.Equal(t, "7D", savedWindows[1].Name)
	})

	t.Run("invalid quota windows returns error", func(t *testing.T) {
		// Duplicate names
		windows := []model.QuotaWindow{
			{Name: "5H", DurationSeconds: 5 * 3600, Quota: 1000},
			{Name: "5H", DurationSeconds: 7 * 86400, Quota: 5000},
		}
		reqBody := AdminUpsertSubscriptionPlanRequest{
			Plan: model.SubscriptionPlan{
				Title:         "Bad Plan",
				PriceAmount:   9.99,
				DurationUnit:  model.SubscriptionDurationMonth,
				DurationValue: 1,
				TotalAmount:   10000,
				QuotaWindows:  model.NewQuotaWindowList(windows),
			},
		}

		ctx, recorder := newSubscriptionAdminContext(t, http.MethodPost, "/subscription/admin/plans", reqBody)
		AdminCreateSubscriptionPlan(ctx)

		var resp subscriptionAPIResponse
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
		assert.False(t, resp.Success, "duplicate names should fail validation")
	})

	t.Run("legacy plan without quota windows is created successfully", func(t *testing.T) {
		reqBody := AdminUpsertSubscriptionPlanRequest{
			Plan: model.SubscriptionPlan{
				Title:         "Legacy Plan",
				PriceAmount:   4.99,
				DurationUnit:  model.SubscriptionDurationMonth,
				DurationValue: 1,
				TotalAmount:   5000,
			},
		}

		ctx, recorder := newSubscriptionAdminContext(t, http.MethodPost, "/subscription/admin/plans", reqBody)
		AdminCreateSubscriptionPlan(ctx)

		var resp subscriptionAPIResponse
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
		assert.True(t, resp.Success, "legacy plan should succeed, got message: %s", resp.Message)
	})
}

// TestAdminUpdateSubscriptionPlanQuotaWindows tests validation+normalization on plan update.
func TestAdminUpdateSubscriptionPlanQuotaWindows(t *testing.T) {
	db := setupSubscriptionControllerTestDB(t)
	confirmSubscriptionPaymentComplianceForTest(t)

	// Seed a plan first
	plan := &model.SubscriptionPlan{
		Title:         "Existing Plan",
		PriceAmount:   9.99,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		TotalAmount:   10000,
	}
	require.NoError(t, db.Create(plan).Error, "seed plan")

	t.Run("valid windowed plan is updated successfully", func(t *testing.T) {
		windows := []model.QuotaWindow{
			{Name: "1H", DurationSeconds: 3600, Quota: 500},
			{Name: "1D", DurationSeconds: 86400, Quota: 3000},
		}
		reqBody := AdminUpsertSubscriptionPlanRequest{
			Plan: model.SubscriptionPlan{
				Title:         "Updated Windowed Plan",
				PriceAmount:   19.99,
				DurationUnit:  model.SubscriptionDurationMonth,
				DurationValue: 1,
				TotalAmount:   20000,
				QuotaWindows:  model.NewQuotaWindowList(windows),
			},
		}

		ctx, recorder := newSubscriptionAdminContext(t, http.MethodPut, "/subscription/admin/plans/"+strconv.Itoa(plan.Id), reqBody)
		ctx.AddParam("id", strconv.Itoa(plan.Id))

		AdminUpdateSubscriptionPlan(ctx)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var resp subscriptionAPIResponse
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
		assert.True(t, resp.Success, "expected success, got message: %s", resp.Message)

		// Verify the plan was updated with quota_windows
		var updated model.SubscriptionPlan
		require.NoError(t, db.First(&updated, plan.Id).Error)
		savedWindows := updated.QuotaWindows.Slice()
		assert.Len(t, savedWindows, 2)
		assert.Equal(t, "1H", savedWindows[0].Name)
		assert.Equal(t, "1D", savedWindows[1].Name)
	})

	t.Run("invalid quota windows on update returns error", func(t *testing.T) {
		// Negative duration
		windows := []model.QuotaWindow{
			{Name: "Bad", DurationSeconds: -1, Quota: 100},
		}
		reqBody := AdminUpsertSubscriptionPlanRequest{
			Plan: model.SubscriptionPlan{
				Title:         "Bad Update",
				PriceAmount:   9.99,
				DurationUnit:  model.SubscriptionDurationMonth,
				DurationValue: 1,
				TotalAmount:   10000,
				QuotaWindows:  model.NewQuotaWindowList(windows),
			},
		}

		ctx, recorder := newSubscriptionAdminContext(t, http.MethodPut, "/subscription/admin/plans/"+strconv.Itoa(plan.Id), reqBody)
		ctx.AddParam("id", strconv.Itoa(plan.Id))

		AdminUpdateSubscriptionPlan(ctx)

		var resp subscriptionAPIResponse
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
		assert.False(t, resp.Success, "invalid duration should fail validation")
	})
}

// TestSubscriptionAPIReturnsQuotaWindows verifies that subscription endpoints
// return quota_windows as a proper JSON array (not an escaped string).
func TestSubscriptionAPIReturnsQuotaWindows(t *testing.T) {
	setupSubscriptionControllerTestDB(t)
	confirmSubscriptionPaymentComplianceForTest(t)

	windows := []model.QuotaWindow{
		{Name: "5H", DurationSeconds: 5 * 3600, Quota: 1000},
	}

	// Create a plan with quota windows
	plan := &model.SubscriptionPlan{
		Title:         "Windowed Plan",
		PriceAmount:   9.99,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		TotalAmount:   5000,
		QuotaWindows:  model.NewQuotaWindowList(windows),
		Enabled:       true,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	// --- Test GetSubscriptionPlans ---
	t.Run("GetSubscriptionPlans returns quota_windows as array", func(t *testing.T) {
		ctx, recorder := newSubscriptionAdminContext(t, http.MethodGet, "/api/subscription/plans", nil)
		GetSubscriptionPlans(ctx)

		assert.Equal(t, http.StatusOK, recorder.Code)
		body := recorder.Body.String()
		// quota_windows must appear as a JSON array, e.g. "quota_windows":[{"name":"5H",...}]
		assert.Contains(t, body, `"quota_windows":[{`, "quota_windows should be a JSON array, not a string")
		assert.NotContains(t, body, `"quota_windows":"[`, "quota_windows must NOT be an escaped string")
	})

	// --- Test AdminListSubscriptionPlans ---
	t.Run("AdminListSubscriptionPlans returns quota_windows as array", func(t *testing.T) {
		ctx, recorder := newSubscriptionAdminContext(t, http.MethodGet, "/api/subscription/admin/plans", nil)
		AdminListSubscriptionPlans(ctx)

		assert.Equal(t, http.StatusOK, recorder.Code)
		body := recorder.Body.String()
		assert.Contains(t, body, `"quota_windows":[{`, "quota_windows should be a JSON array")
		assert.NotContains(t, body, `"quota_windows":"[`, "quota_windows must NOT be an escaped string")
	})

	// Create a user subscription with quota windows for user/self/admin tests
	user := &model.User{Id: 100, Username: "testuser", Status: common.UserStatusEnabled}
	require.NoError(t, model.DB.Create(user).Error)

	now := common.GetTimestamp()
	sub := &model.UserSubscription{
		UserId:       100,
		PlanId:       plan.Id,
		AmountTotal:  5000,
		AmountUsed:   0,
		StartTime:    now,
		EndTime:      now + 86400*30,
		Status:       "active",
		QuotaWindows: model.NewQuotaWindowList(windows),
		QuotaWindowStates: model.NewQuotaWindowStateList([]model.QuotaWindowState{
			{Name: "5H", DurationSeconds: 5 * 3600, Quota: 1000, Used: 200, WindowStart: now},
		}),
	}
	require.NoError(t, model.DB.Create(sub).Error)

	// --- Test GetSubscriptionSelf ---
	t.Run("GetSubscriptionSelf returns quota_windows as array", func(t *testing.T) {
		ctx, recorder := newSubscriptionAdminContext(t, http.MethodGet, "/api/user/subscription", nil)
		ctx.Set("id", 100)

		GetSubscriptionSelf(ctx)

		assert.Equal(t, http.StatusOK, recorder.Code)
		body := recorder.Body.String()
		// Both quota_windows and quota_window_states should be arrays
		assert.Contains(t, body, `"quota_windows":[{`, "quota_windows should be a JSON array")
		assert.Contains(t, body, `"quota_window_states":[{`, "quota_window_states should be a JSON array")
		assert.NotContains(t, body, `"quota_windows":"[`, "quota_windows must NOT be an escaped string")
		assert.NotContains(t, body, `"quota_window_states":"[`, "quota_window_states must NOT be an escaped string")
	})

	// --- Test AdminListUserSubscriptions ---
	t.Run("AdminListUserSubscriptions returns quota_windows as array", func(t *testing.T) {
		ctx, recorder := newSubscriptionAdminContext(t, http.MethodGet, "/api/subscription/admin/user/100", nil)
		ctx.AddParam("id", "100")

		AdminListUserSubscriptions(ctx)

		assert.Equal(t, http.StatusOK, recorder.Code)
		body := recorder.Body.String()
		assert.Contains(t, body, `"quota_windows":[{`, "quota_windows should be a JSON array")
		assert.Contains(t, body, `"quota_window_states":[{`, "quota_window_states should be a JSON array")
		assert.NotContains(t, body, `"quota_windows":"[`, "quota_windows must NOT be an escaped string")
		assert.NotContains(t, body, `"quota_window_states":"[`, "quota_window_states must NOT be an escaped string")
	})
}

// TestSubscriptionAPINormalizesMalformedWindowState verifies that subscriptions
// with malformed DB JSON in quota_windows/quota_window_states do not crash
// the API and return empty arrays instead.
func TestSubscriptionAPINormalizesMalformedWindowState(t *testing.T) {
	setupSubscriptionControllerTestDB(t)

	user := &model.User{Id: 200, Username: "malformeduser", Status: common.UserStatusEnabled}
	require.NoError(t, model.DB.Create(user).Error)

	// Insert a subscription with malformed quota_windows directly into DB
	now := common.GetTimestamp()
	// Use raw SQL to insert malformed JSON that GORM Scan will flag as invalid
	require.NoError(t, model.DB.Exec(
		`INSERT INTO user_subscriptions (user_id, plan_id, amount_total, amount_used, start_time, end_time, status, quota_windows, quota_window_states, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		200, 1, 5000, 0, now, now+86400*30, "active",
		`{not valid json`,
		`also not valid`,
		now, now,
	).Error)

	t.Run("GetSubscriptionSelf handles malformed data", func(t *testing.T) {
		ctx, recorder := newSubscriptionAdminContext(t, http.MethodGet, "/api/user/subscription", nil)
		ctx.Set("id", 200)

		GetSubscriptionSelf(ctx)

		assert.Equal(t, http.StatusOK, recorder.Code)
		body := recorder.Body.String()
		// Should not crash; should return empty arrays for the malformed fields
		assert.Contains(t, body, `"quota_windows":[]`, "malformed quota_windows should be normalized to empty array")
		assert.Contains(t, body, `"quota_window_states":[]`, "malformed quota_window_states should be normalized to empty array")
	})

	t.Run("AdminListUserSubscriptions handles malformed data", func(t *testing.T) {
		ctx, recorder := newSubscriptionAdminContext(t, http.MethodGet, "/api/subscription/admin/user/200", nil)
		ctx.AddParam("id", "200")

		AdminListUserSubscriptions(ctx)

		assert.Equal(t, http.StatusOK, recorder.Code)
		body := recorder.Body.String()
		assert.Contains(t, body, `"quota_windows":[]`, "malformed quota_windows should be normalized to empty array")
		assert.Contains(t, body, `"quota_window_states":[]`, "malformed quota_window_states should be normalized to empty array")
	})
}

// TestSubscriptionAPIDoesNotReturnWindowStrings verifies the raw JSON output
// does not contain the "string-of-array" anti-pattern (e.g. "[" instead of [[).
func TestSubscriptionAPIDoesNotReturnWindowStrings(t *testing.T) {
	setupSubscriptionControllerTestDB(t)
	confirmSubscriptionPaymentComplianceForTest(t)

	windows := []model.QuotaWindow{
		{Name: "5H", DurationSeconds: 5 * 3600, Quota: 1000},
	}

	plan := &model.SubscriptionPlan{
		Title:         "Windowed Plan",
		PriceAmount:   9.99,
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		TotalAmount:   5000,
		QuotaWindows:  model.NewQuotaWindowList(windows),
		Enabled:       true,
	}
	require.NoError(t, model.DB.Create(plan).Error)

	t.Run("plan endpoint raw JSON has array not string", func(t *testing.T) {
		ctx, recorder := newSubscriptionAdminContext(t, http.MethodGet, "/api/subscription/plans", nil)
		GetSubscriptionPlans(ctx)

		body := recorder.Body.String()
		// quota_windows must be a JSON array like [{"name":"5H",...}], not a string like "[{\"name\":\"5H\",...}]"
		assert.Contains(t, body, `"quota_windows":[{"name":"5H"`, "quota_windows should be a JSON array with objects")
		assert.NotContains(t, body, `"quota_windows":"[`, "quota_windows must NOT be an escaped JSON string")
	})
}
