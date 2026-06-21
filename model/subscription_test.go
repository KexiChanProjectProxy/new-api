package model

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func ptrStr(s string) *string { return &s }

// ============================================================================
// TestSubscriptionQuotaWindowValidate — validation helpers
// ============================================================================

func TestSubscriptionQuotaWindowValidate(t *testing.T) {
	t.Run("valid windows 5H/7D/1M", func(t *testing.T) {
		windows := []QuotaWindow{
			{Name: "5H", DurationSeconds: 5 * 3600, Quota: 1000},
			{Name: "7D", DurationSeconds: 7 * 86400, Quota: 50000},
			{Name: "1M", DurationSeconds: 30 * 86400, Quota: 200000},
		}
		assert.NoError(t, ValidateQuotaWindows(windows))
	})

	t.Run("empty list rejected", func(t *testing.T) {
		assert.Error(t, ValidateQuotaWindows(nil))
		assert.Error(t, ValidateQuotaWindows([]QuotaWindow{}))
	})

	t.Run("duplicate names rejected", func(t *testing.T) {
		windows := []QuotaWindow{
			{Name: "5H", DurationSeconds: 18000, Quota: 1000},
			{Name: "5H", DurationSeconds: 36000, Quota: 2000},
		}
		err := ValidateQuotaWindows(windows)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})

	t.Run("duration_seconds <= 0 rejected", func(t *testing.T) {
		windows := []QuotaWindow{
			{Name: "zero", DurationSeconds: 0, Quota: 1000},
		}
		err := ValidateQuotaWindows(windows)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duration_seconds must be > 0")

		windows = []QuotaWindow{
			{Name: "neg", DurationSeconds: -100, Quota: 1000},
		}
		err = ValidateQuotaWindows(windows)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duration_seconds must be > 0")
	})

	t.Run("quota <= 0 rejected", func(t *testing.T) {
		windows := []QuotaWindow{
			{Name: "zero-quota", DurationSeconds: 3600, Quota: 0},
		}
		err := ValidateQuotaWindows(windows)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "quota must be > 0")

		windows = []QuotaWindow{
			{Name: "neg-quota", DurationSeconds: 3600, Quota: -50},
		}
		err = ValidateQuotaWindows(windows)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "quota must be > 0")
	})

	t.Run("exceeds MaxSubscriptionQuotaWindows rejected", func(t *testing.T) {
		windows := make([]QuotaWindow, MaxSubscriptionQuotaWindows+1)
		for i := range windows {
			windows[i] = QuotaWindow{
				Name:            fmt.Sprintf("W%d", i),
				DurationSeconds: int64(i+1) * 3600,
				Quota:           100,
			}
		}
		err := ValidateQuotaWindows(windows)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum")
	})

	t.Run("at MaxSubscriptionQuotaWindows is ok", func(t *testing.T) {
		windows := make([]QuotaWindow, MaxSubscriptionQuotaWindows)
		for i := range windows {
			windows[i] = QuotaWindow{
				Name:            fmt.Sprintf("W%d", i),
				DurationSeconds: int64(i+1) * 3600,
				Quota:           100,
			}
		}
		assert.NoError(t, ValidateQuotaWindows(windows))
	})

	t.Run("empty name rejected", func(t *testing.T) {
		windows := []QuotaWindow{
			{Name: "", DurationSeconds: 3600, Quota: 100},
		}
		err := ValidateQuotaWindows(windows)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name must not be empty")
	})

	t.Run("whitespace-only name rejected", func(t *testing.T) {
		windows := []QuotaWindow{
			{Name: "   ", DurationSeconds: 3600, Quota: 100},
		}
		err := ValidateQuotaWindows(windows)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name must not be empty")
	})
}

func TestSubscriptionQuotaWindowUsedOverflow(t *testing.T) {
	t.Run("no overflow with small values", func(t *testing.T) {
		states := []QuotaWindowState{
			{Name: "5H", DurationSeconds: 18000, Quota: 1000, Used: 500},
		}
		assert.NoError(t, ValidateQuotaWindowUsedOverflow(states, 100))
	})

	t.Run("overflow detected", func(t *testing.T) {
		states := []QuotaWindowState{
			{Name: "5H", DurationSeconds: 18000, Quota: 1000, Used: math.MaxInt64 - 5},
		}
		err := ValidateQuotaWindowUsedOverflow(states, 100)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "overflow")
	})

	t.Run("exact boundary no overflow", func(t *testing.T) {
		states := []QuotaWindowState{
			{Name: "5H", DurationSeconds: 18000, Quota: 1000, Used: math.MaxInt64 - 100},
		}
		assert.NoError(t, ValidateQuotaWindowUsedOverflow(states, 100))
	})

	t.Run("zero used no overflow", func(t *testing.T) {
		states := []QuotaWindowState{
			{Name: "5H", DurationSeconds: 18000, Quota: 1000, Used: 0},
		}
		assert.NoError(t, ValidateQuotaWindowUsedOverflow(states, math.MaxInt64))
	})

	t.Run("negative amount no overflow check", func(t *testing.T) {
		states := []QuotaWindowState{
			{Name: "5H", DurationSeconds: 18000, Quota: 1000, Used: math.MaxInt64},
		}
		// Negative amount is a refund, not an overflow concern
		assert.NoError(t, ValidateQuotaWindowUsedOverflow(states, -1))
	})

	t.Run("empty states no error", func(t *testing.T) {
		assert.NoError(t, ValidateQuotaWindowUsedOverflow(nil, 100))
		assert.NoError(t, ValidateQuotaWindowUsedOverflow([]QuotaWindowState{}, 100))
	})
}

// ============================================================================
// TestSubscriptionQuotaWindowJSON — JSON marshal/unmarshal for custom list types
// ============================================================================

func TestSubscriptionQuotaWindowJSON(t *testing.T) {
	t.Run("QuotaWindowList Scan nil/null/empty", func(t *testing.T) {
		var l QuotaWindowList
		assert.NoError(t, l.Scan(nil))
		assert.Equal(t, []QuotaWindow{}, l.Slice())
		assert.False(t, l.IsInvalid())

		assert.NoError(t, l.Scan([]byte{}))
		assert.Equal(t, []QuotaWindow{}, l.Slice())

		assert.NoError(t, l.Scan([]byte("null")))
		assert.Equal(t, []QuotaWindow{}, l.Slice())
	})

	t.Run("QuotaWindowList Scan valid JSON", func(t *testing.T) {
		data := `[{"name":"5H","duration_seconds":18000,"quota":1000}]`
		var l QuotaWindowList
		assert.NoError(t, l.Scan([]byte(data)))
		assert.Equal(t, 1, len(l.Slice()))
		assert.Equal(t, "5H", l.Slice()[0].Name)
		assert.Equal(t, int64(18000), l.Slice()[0].DurationSeconds)
		assert.Equal(t, int64(1000), l.Slice()[0].Quota)
		assert.False(t, l.IsInvalid())
	})

	t.Run("QuotaWindowList Scan malformed JSON → empty list, no error, invalid flag", func(t *testing.T) {
		var l QuotaWindowList
		assert.NoError(t, l.Scan([]byte("{broken")))
		assert.Equal(t, []QuotaWindow{}, l.Slice())
		assert.True(t, l.IsInvalid())
	})

	t.Run("QuotaWindowList Scan string type", func(t *testing.T) {
		data := `[{"name":"7D","duration_seconds":604800,"quota":50000}]`
		var l QuotaWindowList
		assert.NoError(t, l.Scan(data))
		assert.Equal(t, 1, len(l.Slice()))
		assert.Equal(t, "7D", l.Slice()[0].Name)
	})

	t.Run("QuotaWindowList Scan unsupported type → empty list, invalid flag", func(t *testing.T) {
		var l QuotaWindowList
		assert.NoError(t, l.Scan(12345))
		assert.Equal(t, []QuotaWindow{}, l.Slice())
		assert.True(t, l.IsInvalid())
	})

	t.Run("QuotaWindowList Value round-trip", func(t *testing.T) {
		original := NewQuotaWindowList([]QuotaWindow{
			{Name: "5H", DurationSeconds: 18000, Quota: 1000},
			{Name: "7D", DurationSeconds: 604800, Quota: 50000},
		})
		val, err := original.Value()
		require.NoError(t, err)

		var restored QuotaWindowList
		assert.NoError(t, restored.Scan(val))
		assert.Equal(t, original.Slice(), restored.Slice())
	})

	t.Run("QuotaWindowList Value empty → []", func(t *testing.T) {
		l := NewQuotaWindowList(nil)
		val, err := l.Value()
		require.NoError(t, err)
		assert.Equal(t, "[]", string(val.([]byte)))
	})

	t.Run("QuotaWindowStateList Scan/Value round-trip", func(t *testing.T) {
		original := NewQuotaWindowStateList([]QuotaWindowState{
			{Name: "5H", DurationSeconds: 18000, Quota: 1000, Used: 300, WindowStart: 1700000000},
		})
		val, err := original.Value()
		require.NoError(t, err)

		var restored QuotaWindowStateList
		assert.NoError(t, restored.Scan(val))
		assert.Equal(t, original.Slice(), restored.Slice())
	})

	t.Run("QuotaWindowStateList Scan malformed → empty, invalid", func(t *testing.T) {
		var l QuotaWindowStateList
		assert.NoError(t, l.Scan([]byte("not json")))
		assert.Equal(t, []QuotaWindowState{}, l.Slice())
		assert.True(t, l.IsInvalid())
	})

	t.Run("QuotaWindowSnapshotList Scan/Value round-trip", func(t *testing.T) {
		original := NewQuotaWindowSnapshotList([]QuotaWindowSnapshot{
			{Name: "5H", DurationSeconds: 18000, WindowStart: 1700000000, Amount: 50},
		})
		val, err := original.Value()
		require.NoError(t, err)

		var restored QuotaWindowSnapshotList
		assert.NoError(t, restored.Scan(val))
		assert.Equal(t, original.Slice(), restored.Slice())
	})

	t.Run("QuotaWindowSnapshotList Scan malformed → empty, invalid", func(t *testing.T) {
		var l QuotaWindowSnapshotList
		assert.NoError(t, l.Scan([]byte("}broken{")))
		assert.Equal(t, []QuotaWindowSnapshot{}, l.Slice())
		assert.True(t, l.IsInvalid())
	})
}

// ============================================================================
// TestSubscriptionQuotaWindowHTTPJSON — HTTP JSON array marshal/unmarshal
// ============================================================================

func TestSubscriptionQuotaWindowHTTPJSON(t *testing.T) {
	t.Run("QuotaWindowList MarshalJSON produces array not string", func(t *testing.T) {
		l := NewQuotaWindowList([]QuotaWindow{
			{Name: "5H", DurationSeconds: 18000, Quota: 1000},
		})
		data, err := json.Marshal(l)
		require.NoError(t, err)
		// Must be a JSON array, not a JSON string
		assert.Equal(t, byte('['), data[0])
		assert.Contains(t, string(data), `"name":"5H"`)
		assert.Contains(t, string(data), `"duration_seconds":18000`)
		assert.Contains(t, string(data), `"quota":1000`)
	})

	t.Run("QuotaWindowList MarshalJSON empty → []", func(t *testing.T) {
		l := NewQuotaWindowList(nil)
		data, err := json.Marshal(l)
		require.NoError(t, err)
		assert.Equal(t, "[]", string(data))
	})

	t.Run("QuotaWindowList UnmarshalJSON from array", func(t *testing.T) {
		input := `[{"name":"5H","duration_seconds":18000,"quota":1000}]`
		var l QuotaWindowList
		assert.NoError(t, json.Unmarshal([]byte(input), &l))
		assert.Equal(t, 1, len(l.Slice()))
		assert.Equal(t, "5H", l.Slice()[0].Name)
	})

	t.Run("QuotaWindowList UnmarshalJSON from null", func(t *testing.T) {
		var l QuotaWindowList
		assert.NoError(t, json.Unmarshal([]byte("null"), &l))
		assert.Equal(t, []QuotaWindow{}, l.Slice())
	})

	t.Run("QuotaWindowList UnmarshalJSON from empty → error", func(t *testing.T) {
		// json.Unmarshal returns error for empty byte input; this is expected.
		// Our UnmarshalJSON only handles "null" and valid arrays gracefully.
		var l QuotaWindowList
		assert.Error(t, json.Unmarshal([]byte(""), &l))
	})

	t.Run("QuotaWindowList UnmarshalJSON malformed → error", func(t *testing.T) {
		var l QuotaWindowList
		assert.Error(t, json.Unmarshal([]byte("{bad"), &l))
	})

	t.Run("QuotaWindowStateList HTTP round-trip", func(t *testing.T) {
		original := NewQuotaWindowStateList([]QuotaWindowState{
			{Name: "5H", DurationSeconds: 18000, Quota: 1000, Used: 300, WindowStart: 1700000000},
		})
		data, err := json.Marshal(original)
		require.NoError(t, err)
		assert.Equal(t, byte('['), data[0])

		var restored QuotaWindowStateList
		assert.NoError(t, json.Unmarshal(data, &restored))
		assert.Equal(t, original.Slice(), restored.Slice())
	})

	t.Run("QuotaWindowSnapshotList HTTP round-trip", func(t *testing.T) {
		original := NewQuotaWindowSnapshotList([]QuotaWindowSnapshot{
			{Name: "5H", DurationSeconds: 18000, WindowStart: 1700000000, Amount: 50},
		})
		data, err := json.Marshal(original)
		require.NoError(t, err)
		assert.Equal(t, byte('['), data[0])

		var restored QuotaWindowSnapshotList
		assert.NoError(t, json.Unmarshal(data, &restored))
		assert.Equal(t, original.Slice(), restored.Slice())
	})
}

// ============================================================================
// TestSubscriptionQuotaWindowMalformedScan — display-tolerant Scan behavior
// ============================================================================

func TestSubscriptionQuotaWindowMalformedScan(t *testing.T) {
	t.Run("QuotaWindowList malformed JSON returns empty list no error", func(t *testing.T) {
		cases := []string{
			"{broken",
			"not json at all",
			`{"name":"5H"}`, // object instead of array
			"[",
			"null",
		}
		for _, input := range cases {
			var l QuotaWindowList
			assert.NoError(t, l.Scan([]byte(input)), "input: %q", input)
			// Should always produce empty list for malformed data
			// (null produces empty list too, but not invalid)
		}
	})

	t.Run("QuotaWindowStateList malformed JSON returns empty list no error", func(t *testing.T) {
		var l QuotaWindowStateList
		assert.NoError(t, l.Scan([]byte("{{invalid}}")))
		assert.Equal(t, []QuotaWindowState{}, l.Slice())
		assert.True(t, l.IsInvalid())
	})

	t.Run("QuotaWindowSnapshotList malformed JSON returns empty list no error", func(t *testing.T) {
		var l QuotaWindowSnapshotList
		assert.NoError(t, l.Scan([]byte("{{invalid}}")))
		assert.Equal(t, []QuotaWindowSnapshot{}, l.Slice())
		assert.True(t, l.IsInvalid())
	})

	t.Run("valid JSON after malformed Scan works", func(t *testing.T) {
		var l QuotaWindowList
		// First scan malformed
		assert.NoError(t, l.Scan([]byte("broken")))
		assert.True(t, l.IsInvalid())

		// Then scan valid — should clear invalid flag
		validData := `[{"name":"5H","duration_seconds":18000,"quota":1000}]`
		assert.NoError(t, l.Scan([]byte(validData)))
		assert.False(t, l.IsInvalid())
		assert.Equal(t, 1, len(l.Slice()))
	})
}

// ============================================================================
// NormalizeQuotaWindows
// ============================================================================

func TestNormalizeQuotaWindows(t *testing.T) {
	t.Run("nil → empty slice", func(t *testing.T) {
		result := NormalizeQuotaWindows(nil)
		assert.NotNil(t, result)
		assert.Equal(t, 0, len(result))
	})

	t.Run("empty → empty slice", func(t *testing.T) {
		result := NormalizeQuotaWindows([]QuotaWindow{})
		assert.NotNil(t, result)
		assert.Equal(t, 0, len(result))
	})

	t.Run("non-empty preserved", func(t *testing.T) {
		input := []QuotaWindow{
			{Name: "5H", DurationSeconds: 18000, Quota: 1000},
		}
		result := NormalizeQuotaWindows(input)
		assert.Equal(t, input, result)
	})
}

// ============================================================================
// PerformLazyWindowReset
// ============================================================================

func TestPerformLazyWindowReset(t *testing.T) {
	t.Run("nil state no panic", func(t *testing.T) {
		PerformLazyWindowReset(nil, 1000)
	})

	t.Run("first use: window_start == 0 sets to now", func(t *testing.T) {
		state := &QuotaWindowState{
			Name:            "5H",
			DurationSeconds: 18000,
			Quota:           1000,
			Used:            0,
			WindowStart:     0,
		}
		now := int64(1700000000)
		PerformLazyWindowReset(state, now)
		assert.Equal(t, now, state.WindowStart)
		assert.Equal(t, int64(0), state.Used) // used unchanged
	})

	t.Run("at boundary: now >= window_start + duration resets used and window_start", func(t *testing.T) {
		state := &QuotaWindowState{
			Name:            "5H",
			DurationSeconds: 18000,
			Quota:           1000,
			Used:            500,
			WindowStart:     1700000000,
		}
		// now is exactly at boundary
		now := state.WindowStart + state.DurationSeconds
		PerformLazyWindowReset(state, now)
		assert.Equal(t, now, state.WindowStart)
		assert.Equal(t, int64(0), state.Used)
	})

	t.Run("past boundary: resets used and window_start", func(t *testing.T) {
		state := &QuotaWindowState{
			Name:            "5H",
			DurationSeconds: 18000,
			Quota:           1000,
			Used:            500,
			WindowStart:     1700000000,
		}
		// now is well past boundary
		now := state.WindowStart + state.DurationSeconds + 10000
		PerformLazyWindowReset(state, now)
		assert.Equal(t, now, state.WindowStart)
		assert.Equal(t, int64(0), state.Used)
	})

	t.Run("before boundary: no change", func(t *testing.T) {
		state := &QuotaWindowState{
			Name:            "5H",
			DurationSeconds: 18000,
			Quota:           1000,
			Used:            500,
			WindowStart:     1700000000,
		}
		// now is 1 second before boundary
		now := state.WindowStart + state.DurationSeconds - 1
		PerformLazyWindowReset(state, now)
		assert.Equal(t, int64(1700000000), state.WindowStart) // unchanged
		assert.Equal(t, int64(500), state.Used)               // unchanged
	})

	t.Run("1M window boundary", func(t *testing.T) {
		oneMonth := int64(30 * 86400) // 2592000
		state := &QuotaWindowState{
			Name:            "1M",
			DurationSeconds: oneMonth,
			Quota:           200000,
			Used:            100000,
			WindowStart:     1700000000,
		}
		// Exactly at boundary
		now := state.WindowStart + oneMonth
		PerformLazyWindowReset(state, now)
		assert.Equal(t, now, state.WindowStart)
		assert.Equal(t, int64(0), state.Used)
	})
}

// ============================================================================
// TestSubscriptionQuotaWindowMigration — schema migration coverage
// ============================================================================

func TestSubscriptionQuotaWindowMigration(t *testing.T) {
	// Use an isolated in-memory SQLite DB so we don't depend on the global TestMain DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "open in-memory SQLite")
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// Save and restore global DB
	origDB := DB
	DB = db
	defer func() { DB = origDB }()

	common.UsingSQLite = true
	initCol()

	// Run AutoMigrate for the three models
	require.NoError(t, db.AutoMigrate(&SubscriptionPlan{}, &UserSubscription{}, &SubscriptionPreConsumeRecord{}), "AutoMigrate")

	// Also run the SQLite manual table logic for SubscriptionPlan
	require.NoError(t, ensureSubscriptionPlanTableSQLite(), "ensureSubscriptionPlanTableSQLite")

	// Helper: check column exists
	assertColumnExists := func(table, column string) {
		var cols []struct {
			Name string `gorm:"column:name"`
		}
		require.NoError(t, db.Raw("PRAGMA table_info(`"+table+"`)").Scan(&cols).Error, "PRAGMA table_info for %s", table)
		found := false
		for _, c := range cols {
			if c.Name == column {
				found = true
				break
			}
		}
		assert.True(t, found, "column %s should exist in table %s", column, table)
	}

	// SubscriptionPlan columns
	assertColumnExists("subscription_plans", "quota_windows")

	// UserSubscription columns
	assertColumnExists("user_subscriptions", "quota_windows")
	assertColumnExists("user_subscriptions", "quota_window_states")

	// SubscriptionPreConsumeRecord columns
	assertColumnExists("subscription_pre_consume_records", "quota_window_snapshots")
	assertColumnExists("subscription_pre_consume_records", "current_consumed")
	assertColumnExists("subscription_pre_consume_records", "final_consumed")
}

// ============================================================================
// TestSubscriptionPreConsumeAccountingBackfill — backfill current_consumed
// ============================================================================

func TestSubscriptionPreConsumeAccountingBackfill(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "open in-memory SQLite")
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	origDB := DB
	DB = db
	defer func() { DB = origDB }()

	common.UsingSQLite = true
	initCol()

	require.NoError(t, db.AutoMigrate(&SubscriptionPreConsumeRecord{}), "AutoMigrate")

	// Insert rows that simulate pre-existing data (before current_consumed column existed).
	// We use raw SQL to insert with current_consumed=0 to simulate the pre-migration state.
	now := common.GetTimestamp()

	// Consumed row: current_consumed should be backfilled to pre_consumed
	require.NoError(t, db.Exec(
		"INSERT INTO subscription_pre_consume_records (request_id, user_id, user_subscription_id, pre_consumed, status, current_consumed, final_consumed, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		"req-consumed-1", 1, 1, int64(500), "consumed", int64(0), int64(0), now, now,
	).Error, "insert consumed row")

	// Another consumed row with different pre_consumed
	require.NoError(t, db.Exec(
		"INSERT INTO subscription_pre_consume_records (request_id, user_id, user_subscription_id, pre_consumed, status, current_consumed, final_consumed, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		"req-consumed-2", 2, 2, int64(1200), "consumed", int64(0), int64(0), now, now,
	).Error, "insert consumed row 2")

	// Refunded row: current_consumed should stay 0
	require.NoError(t, db.Exec(
		"INSERT INTO subscription_pre_consume_records (request_id, user_id, user_subscription_id, pre_consumed, status, current_consumed, final_consumed, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		"req-refunded-1", 3, 3, int64(800), "refunded", int64(0), int64(0), now, now,
	).Error, "insert refunded row")

	// Consumed row already with non-zero current_consumed: should NOT be changed
	require.NoError(t, db.Exec(
		"INSERT INTO subscription_pre_consume_records (request_id, user_id, user_subscription_id, pre_consumed, status, current_consumed, final_consumed, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		"req-already-set", 4, 4, int64(999), "consumed", int64(100), int64(0), now, now,
	).Error, "insert already-set row")

	// Run backfill
	require.NoError(t, backfillSubscriptionPreConsumeRecords(), "backfill")

	// Verify consumed rows got current_consumed = pre_consumed
	var consumed1, consumed2 SubscriptionPreConsumeRecord
	require.NoError(t, db.Where("request_id = ?", "req-consumed-1").First(&consumed1).Error)
	assert.Equal(t, int64(500), consumed1.CurrentConsumed, "consumed row 1: current_consumed should equal pre_consumed")
	assert.Equal(t, int64(0), consumed1.FinalConsumed, "consumed row 1: final_consumed should stay 0")

	require.NoError(t, db.Where("request_id = ?", "req-consumed-2").First(&consumed2).Error)
	assert.Equal(t, int64(1200), consumed2.CurrentConsumed, "consumed row 2: current_consumed should equal pre_consumed")

	// Verify refunded row stays at 0
	var refunded SubscriptionPreConsumeRecord
	require.NoError(t, db.Where("request_id = ?", "req-refunded-1").First(&refunded).Error)
	assert.Equal(t, int64(0), refunded.CurrentConsumed, "refunded row: current_consumed should stay 0")

	// Verify already-set row was NOT changed
	var alreadySet SubscriptionPreConsumeRecord
	require.NoError(t, db.Where("request_id = ?", "req-already-set").First(&alreadySet).Error)
	assert.Equal(t, int64(100), alreadySet.CurrentConsumed, "already-set row: current_consumed should NOT be overwritten")

	// Idempotency: running backfill again should not change anything
	require.NoError(t, backfillSubscriptionPreConsumeRecords(), "backfill idempotent")
	require.NoError(t, db.Where("request_id = ?", "req-consumed-1").First(&consumed1).Error)
	assert.Equal(t, int64(500), consumed1.CurrentConsumed, "idempotent: consumed row 1 unchanged")
}

// ============================================================================
// TestCreateUserSubscriptionSnapshotsQuotaWindows — snapshot at creation
// ============================================================================

func TestCreateUserSubscriptionSnapshotsQuotaWindows(t *testing.T) {
	common.UsingSQLite = true
	common.RedisEnabled = false

	t.Run("snapshots quota windows and initializes states", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err, "open in-memory SQLite")
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		origDB := DB
		DB = db
		defer func() { DB = origDB }()

		initCol()

		require.NoError(t, db.AutoMigrate(&User{}, &SubscriptionPlan{}, &UserSubscription{}), "AutoMigrate")

		windows := []QuotaWindow{
			{Name: "5H", DurationSeconds: 5 * 3600, Quota: 1000},
			{Name: "7D", DurationSeconds: 7 * 86400, Quota: 5000},
			{Name: "1M", DurationSeconds: 30 * 86400, Quota: 20000},
		}

		plan := &SubscriptionPlan{
			Id:            0,
			Title:         "Test Plan",
			PriceAmount:   9.99,
			DurationUnit:  SubscriptionDurationMonth,
			DurationValue: 1,
			TotalAmount:   50000,
			QuotaWindows:  NewQuotaWindowList(windows),
		}
		require.NoError(t, db.Create(plan).Error, "create plan")
		require.NotZero(t, plan.Id, "plan should have an ID after create")

		// Create a user for the subscription
		user := &User{
			Username:    "testuser",
			Email:       "test@example.com",
			Password:    "hash",
			AccessToken: ptrStr("test-token-snap"),
			Group:       "default",
		}
		require.NoError(t, db.Create(user).Error, "create user")

		sub, err := CreateUserSubscriptionFromPlanTx(db, user.Id, plan, "admin")
		require.NoError(t, err, "CreateUserSubscriptionFromPlanTx")
		require.NotNil(t, sub)

		// Verify quota windows were snapshotted
		subWindows := sub.QuotaWindows.Slice()
		assert.Len(t, subWindows, 3, "should have 3 quota windows")
		assert.Equal(t, "5H", subWindows[0].Name)
		assert.Equal(t, int64(5*3600), subWindows[0].DurationSeconds)
		assert.Equal(t, int64(1000), subWindows[0].Quota)
		assert.Equal(t, "7D", subWindows[1].Name)
		assert.Equal(t, int64(7*86400), subWindows[1].DurationSeconds)
		assert.Equal(t, int64(5000), subWindows[1].Quota)
		assert.Equal(t, "1M", subWindows[2].Name)
		assert.Equal(t, int64(30*86400), subWindows[2].DurationSeconds)
		assert.Equal(t, int64(20000), subWindows[2].Quota)

		// Verify quota window states were initialized
		states := sub.QuotaWindowStates.Slice()
		assert.Len(t, states, 3, "should have 3 quota window states")
		for i, s := range states {
			assert.Equal(t, subWindows[i].Name, s.Name, "state[%d] name", i)
			assert.Equal(t, subWindows[i].DurationSeconds, s.DurationSeconds, "state[%d] duration", i)
			assert.Equal(t, subWindows[i].Quota, s.Quota, "state[%d] quota", i)
			assert.Equal(t, int64(0), s.Used, "state[%d] used should be 0", i)
			assert.Equal(t, int64(0), s.WindowStart, "state[%d] window_start should be 0", i)
		}

		// Verify legacy fields are zeroed
		assert.Equal(t, int64(0), sub.AmountTotal, "amount_total should be 0 for windowed subscription")
		assert.Equal(t, int64(0), sub.AmountUsed, "amount_used should be 0 for windowed subscription")
		assert.Equal(t, int64(0), sub.LastResetTime, "last_reset_time should be 0 for windowed subscription")
		assert.Equal(t, int64(0), sub.NextResetTime, "next_reset_time should be 0 for windowed subscription")
	})

	t.Run("legacy plan without quota windows keeps legacy fields", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err, "open in-memory SQLite")
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		origDB := DB
		DB = db
		defer func() { DB = origDB }()

		initCol()

		require.NoError(t, db.AutoMigrate(&User{}, &SubscriptionPlan{}, &UserSubscription{}), "AutoMigrate")

		plan := &SubscriptionPlan{
			Id:            0,
			Title:         "Legacy Plan",
			PriceAmount:   4.99,
			DurationUnit:  SubscriptionDurationMonth,
			DurationValue: 1,
			TotalAmount:   10000,
			QuotaWindows:  NewQuotaWindowList(nil), // empty = legacy
		}
		require.NoError(t, db.Create(plan).Error, "create plan")
		require.NotZero(t, plan.Id)

		user := &User{
			Username:    "legacyuser",
			Email:       "legacy@example.com",
			Password:    "hash",
			AccessToken: ptrStr("test-token-legacy"),
			Group:       "default",
		}
		require.NoError(t, db.Create(user).Error, "create user")

		sub, err := CreateUserSubscriptionFromPlanTx(db, user.Id, plan, "admin")
		require.NoError(t, err, "CreateUserSubscriptionFromPlanTx")
		require.NotNil(t, sub)

		// No quota windows snapshotted
		assert.Empty(t, sub.QuotaWindows.Slice(), "should have no quota windows")
		assert.Empty(t, sub.QuotaWindowStates.Slice(), "should have no quota window states")

		// Legacy fields should be set normally
		assert.Equal(t, int64(10000), sub.AmountTotal, "legacy amount_total should match plan")
		assert.Equal(t, int64(0), sub.AmountUsed, "legacy amount_used should be 0")
	})
}

// ============================================================================
// TestPreConsumeUserSubscription_Legacy — legacy (non-windowed) path unchanged
// ============================================================================

func TestPreConsumeUserSubscription_Legacy(t *testing.T) {
	common.UsingSQLite = true
	common.RedisEnabled = false

	t.Run("legacy subscription pre-consumes and tracks amount", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err, "open in-memory SQLite")
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		origDB := DB
		DB = db
		defer func() { DB = origDB }()

		initCol()
		require.NoError(t, db.AutoMigrate(&User{}, &SubscriptionPlan{}, &UserSubscription{}, &SubscriptionPreConsumeRecord{}), "AutoMigrate")

		plan := &SubscriptionPlan{
			Title:         "Legacy Plan",
			PriceAmount:   4.99,
			DurationUnit:  SubscriptionDurationMonth,
			DurationValue: 1,
			TotalAmount:   10000,
			QuotaWindows:  NewQuotaWindowList(nil),
		}
		require.NoError(t, db.Create(plan).Error)

		user := &User{
			Username:    "legacyuser",
			Email:       "legacy@example.com",
			Password:    "hash",
			AccessToken: ptrStr("test-token-legacy-pc"),
			Group:       "default",
		}
		require.NoError(t, db.Create(user).Error)

		sub, err := CreateUserSubscriptionFromPlanTx(db, user.Id, plan, "admin")
		require.NoError(t, err)
		require.NotNil(t, sub)

		result, err := PreConsumeUserSubscription("req-legacy-1", user.Id, "gpt-4", 1, 500)
		require.NoError(t, err)
		assert.Equal(t, sub.Id, result.UserSubscriptionId)
		assert.Equal(t, int64(500), result.PreConsumed)
		assert.Equal(t, int64(10000), result.AmountTotal)
		assert.Equal(t, int64(0), result.AmountUsedBefore)
		assert.Equal(t, int64(500), result.AmountUsedAfter)
		assert.False(t, result.Windowed, "legacy should not be windowed")
		assert.Equal(t, int64(500), result.CurrentConsumed, "legacy CurrentConsumed should match preConsumed")
	})

	t.Run("legacy subscription rejects when quota insufficient", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		origDB := DB
		DB = db
		defer func() { DB = origDB }()

		initCol()
		require.NoError(t, db.AutoMigrate(&User{}, &SubscriptionPlan{}, &UserSubscription{}, &SubscriptionPreConsumeRecord{}))

		plan := &SubscriptionPlan{
			Title:         "Small Plan",
			PriceAmount:   1.99,
			DurationUnit:  SubscriptionDurationMonth,
			DurationValue: 1,
			TotalAmount:   100,
			QuotaWindows:  NewQuotaWindowList(nil),
		}
		require.NoError(t, db.Create(plan).Error)

		user := &User{
			Username:    "smalluser",
			Email:       "small@example.com",
			Password:    "hash",
			AccessToken: ptrStr("test-token-small"),
			Group:       "default",
		}
		require.NoError(t, db.Create(user).Error)

		sub, err := CreateUserSubscriptionFromPlanTx(db, user.Id, plan, "admin")
		require.NoError(t, err)
		require.NotNil(t, sub)

		_, err = PreConsumeUserSubscription("req-legacy-big", user.Id, "gpt-4", 1, 200)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient")
	})

	t.Run("legacy duplicate request id returns same result", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		origDB := DB
		DB = db
		defer func() { DB = origDB }()

		initCol()
		require.NoError(t, db.AutoMigrate(&User{}, &SubscriptionPlan{}, &UserSubscription{}, &SubscriptionPreConsumeRecord{}))

		plan := &SubscriptionPlan{
			Title:         "Dup Plan",
			PriceAmount:   4.99,
			DurationUnit:  SubscriptionDurationMonth,
			DurationValue: 1,
			TotalAmount:   10000,
			QuotaWindows:  NewQuotaWindowList(nil),
		}
		require.NoError(t, db.Create(plan).Error)

		user := &User{
			Username:    "dupuser",
			Email:       "dup@example.com",
			Password:    "hash",
			AccessToken: ptrStr("test-token-dup"),
			Group:       "default",
		}
		require.NoError(t, db.Create(user).Error)

		sub, err := CreateUserSubscriptionFromPlanTx(db, user.Id, plan, "admin")
		require.NoError(t, err)
		require.NotNil(t, sub)

		result1, err := PreConsumeUserSubscription("req-dup-legacy", user.Id, "gpt-4", 1, 300)
		require.NoError(t, err)

		result2, err := PreConsumeUserSubscription("req-dup-legacy", user.Id, "gpt-4", 1, 300)
		require.NoError(t, err)

		assert.Equal(t, result1.PreConsumed, result2.PreConsumed)
		assert.Equal(t, result1.UserSubscriptionId, result2.UserSubscriptionId)
		assert.False(t, result2.Windowed)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		assert.Equal(t, int64(300), subCheck.AmountUsed, "should not double-consume")
	})
}

// ============================================================================
// TestPreConsumeUserSubscription_QuotaWindows — windowed subscription path
// ============================================================================

func TestPreConsumeUserSubscription_QuotaWindows(t *testing.T) {
	common.UsingSQLite = true
	common.RedisEnabled = false

	t.Run("first use starts windows", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		origDB := DB
		DB = db
		defer func() { DB = origDB }()

		initCol()
		require.NoError(t, db.AutoMigrate(&User{}, &SubscriptionPlan{}, &UserSubscription{}, &SubscriptionPreConsumeRecord{}))

		windows := []QuotaWindow{
			{Name: "5H", DurationSeconds: 5 * 3600, Quota: 1000},
			{Name: "7D", DurationSeconds: 7 * 86400, Quota: 5000},
		}
		plan := &SubscriptionPlan{
			Title:         "Windowed Plan",
			PriceAmount:   9.99,
			DurationUnit:  SubscriptionDurationMonth,
			DurationValue: 1,
			QuotaWindows:  NewQuotaWindowList(windows),
		}
		require.NoError(t, db.Create(plan).Error)

		user := &User{
			Username:    "winuser",
			Email:       "win@example.com",
			Password:    "hash",
			AccessToken: ptrStr("test-token-win"),
			Group:       "default",
		}
		require.NoError(t, db.Create(user).Error)

		sub, err := CreateUserSubscriptionFromPlanTx(db, user.Id, plan, "admin")
		require.NoError(t, err)
		require.NotNil(t, sub)

		result, err := PreConsumeUserSubscription("req-win-first", user.Id, "gpt-4", 1, 100)
		require.NoError(t, err)

		assert.Equal(t, sub.Id, result.UserSubscriptionId)
		assert.Equal(t, int64(100), result.PreConsumed)
		assert.True(t, result.Windowed, "should be windowed")
		assert.Equal(t, int64(100), result.CurrentConsumed)
		assert.Equal(t, int64(0), result.AmountTotal, "windowed AmountTotal should be 0")
		assert.Equal(t, int64(0), result.AmountUsedBefore, "windowed AmountUsedBefore should be 0")
		assert.Equal(t, int64(0), result.AmountUsedAfter, "windowed AmountUsedAfter should be 0")

		snapshots := result.QuotaWindowSnapshots.Slice()
		assert.Len(t, snapshots, 2)
		assert.Equal(t, "5H", snapshots[0].Name)
		assert.Equal(t, int64(100), snapshots[0].Amount)
		assert.Equal(t, "7D", snapshots[1].Name)
		assert.Equal(t, int64(100), snapshots[1].Amount)
		assert.True(t, snapshots[0].WindowStart > 0, "first use should set window_start")
		assert.True(t, snapshots[1].WindowStart > 0, "first use should set window_start")

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Len(t, states, 2)
		assert.Equal(t, int64(100), states[0].Used)
		assert.Equal(t, int64(100), states[1].Used)
		assert.True(t, states[0].WindowStart > 0, "window_start should be set after first use")
		assert.True(t, states[1].WindowStart > 0, "window_start should be set after first use")
	})

	t.Run("no reset before window boundary", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		origDB := DB
		DB = db
		defer func() { DB = origDB }()

		initCol()
		require.NoError(t, db.AutoMigrate(&User{}, &SubscriptionPlan{}, &UserSubscription{}, &SubscriptionPreConsumeRecord{}))

		windows := []QuotaWindow{
			{Name: "5H", DurationSeconds: 5 * 3600, Quota: 1000},
		}
		plan := &SubscriptionPlan{
			Title:         "Windowed Plan",
			PriceAmount:   9.99,
			DurationUnit:  SubscriptionDurationMonth,
			DurationValue: 1,
			QuotaWindows:  NewQuotaWindowList(windows),
		}
		require.NoError(t, db.Create(plan).Error)

		user := &User{
			Username:    "winuser2",
			Email:       "win2@example.com",
			Password:    "hash",
			AccessToken: ptrStr("test-token-win2"),
			Group:       "default",
		}
		require.NoError(t, db.Create(user).Error)

		sub, err := CreateUserSubscriptionFromPlanTx(db, user.Id, plan, "admin")
		require.NoError(t, err)
		require.NotNil(t, sub)

		// First consume to start the window
		_, err = PreConsumeUserSubscription("req-win2-first", user.Id, "gpt-4", 1, 100)
		require.NoError(t, err)

		// Read back to get the window_start
		var subAfter UserSubscription
		require.NoError(t, db.First(&subAfter, sub.Id).Error)
		states := subAfter.QuotaWindowStates.Slice()
		windowStart := states[0].WindowStart
		require.True(t, windowStart > 0, "window should have started")

		// Second consume — should NOT reset (window still active)
		result, err := PreConsumeUserSubscription("req-win2-second", user.Id, "gpt-4", 1, 200)
		require.NoError(t, err)
		assert.True(t, result.Windowed)
		assert.Equal(t, int64(200), result.CurrentConsumed)

		// Verify used accumulated (not reset)
		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states2 := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(300), states2[0].Used, "used should accumulate to 300")
		assert.Equal(t, windowStart, states2[0].WindowStart, "window_start should not change within window")
	})

	t.Run("reset at window boundary", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		origDB := DB
		DB = db
		defer func() { DB = origDB }()

		initCol()
		require.NoError(t, db.AutoMigrate(&User{}, &SubscriptionPlan{}, &UserSubscription{}, &SubscriptionPreConsumeRecord{}))

		windows := []QuotaWindow{
			{Name: "5H", DurationSeconds: 5 * 3600, Quota: 1000},
		}
		plan := &SubscriptionPlan{
			Title:         "Windowed Plan",
			PriceAmount:   9.99,
			DurationUnit:  SubscriptionDurationMonth,
			DurationValue: 1,
			QuotaWindows:  NewQuotaWindowList(windows),
		}
		require.NoError(t, db.Create(plan).Error)

		user := &User{
			Username:    "winuser3",
			Email:       "win3@example.com",
			Password:    "hash",
			AccessToken: ptrStr("test-token-win3"),
			Group:       "default",
		}
		require.NoError(t, db.Create(user).Error)

		sub, err := CreateUserSubscriptionFromPlanTx(db, user.Id, plan, "admin")
		require.NoError(t, err)
		require.NotNil(t, sub)

		// First consume to start the window
		_, err = PreConsumeUserSubscription("req-win3-first", user.Id, "gpt-4", 1, 100)
		require.NoError(t, err)

		// Manually set window_start to the past so the window has expired
		now := GetDBTimestamp()
		expiredStart := now - 5*3600 - 1 // 1 second past the 5H window
		states := []QuotaWindowState{
			{Name: "5H", DurationSeconds: 5 * 3600, Quota: 1000, Used: 100, WindowStart: expiredStart},
		}
		require.NoError(t, db.Model(&UserSubscription{}).Where("id = ?", sub.Id).
			Updates(map[string]interface{}{
				"quota_window_states": NewQuotaWindowStateList(states),
			}).Error)

		// Consume again — should trigger lazy reset (used=0, window_start=now)
		result, err := PreConsumeUserSubscription("req-win3-reset", user.Id, "gpt-4", 1, 50)
		require.NoError(t, err)
		assert.True(t, result.Windowed)
		assert.Equal(t, int64(50), result.CurrentConsumed)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states2 := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(50), states2[0].Used, "used should be reset to 50 (not 150)")
		assert.True(t, states2[0].WindowStart > expiredStart, "window_start should be updated to now")
	})

	t.Run("insufficient single window rejects", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		origDB := DB
		DB = db
		defer func() { DB = origDB }()

		initCol()
		require.NoError(t, db.AutoMigrate(&User{}, &SubscriptionPlan{}, &UserSubscription{}, &SubscriptionPreConsumeRecord{}))

		windows := []QuotaWindow{
			{Name: "5H", DurationSeconds: 5 * 3600, Quota: 100},
			{Name: "7D", DurationSeconds: 7 * 86400, Quota: 5000},
		}
		plan := &SubscriptionPlan{
			Title:         "Windowed Plan",
			PriceAmount:   9.99,
			DurationUnit:  SubscriptionDurationMonth,
			DurationValue: 1,
			QuotaWindows:  NewQuotaWindowList(windows),
		}
		require.NoError(t, db.Create(plan).Error)

		user := &User{
			Username:    "winuser4",
			Email:       "win4@example.com",
			Password:    "hash",
			AccessToken: ptrStr("test-token-win4"),
			Group:       "default",
		}
		require.NoError(t, db.Create(user).Error)

		sub, err := CreateUserSubscriptionFromPlanTx(db, user.Id, plan, "admin")
		require.NoError(t, err)
		require.NotNil(t, sub)

		// Consume 80 first
		_, err = PreConsumeUserSubscription("req-win4-first", user.Id, "gpt-4", 1, 80)
		require.NoError(t, err)

		// Try to consume 30 more — 5H window only has 20 remaining, should reject
		_, err = PreConsumeUserSubscription("req-win4-reject", user.Id, "gpt-4", 1, 30)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient")

		// Verify no state was changed for the rejected request
		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(80), states[0].Used, "5H used should still be 80")
		assert.Equal(t, int64(80), states[1].Used, "7D used should still be 80 (not incremented)")
	})

	t.Run("duplicate request IDs do not double-consume", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		origDB := DB
		DB = db
		defer func() { DB = origDB }()

		initCol()
		require.NoError(t, db.AutoMigrate(&User{}, &SubscriptionPlan{}, &UserSubscription{}, &SubscriptionPreConsumeRecord{}))

		windows := []QuotaWindow{
			{Name: "5H", DurationSeconds: 5 * 3600, Quota: 1000},
			{Name: "7D", DurationSeconds: 7 * 86400, Quota: 5000},
		}
		plan := &SubscriptionPlan{
			Title:         "Windowed Plan",
			PriceAmount:   9.99,
			DurationUnit:  SubscriptionDurationMonth,
			DurationValue: 1,
			QuotaWindows:  NewQuotaWindowList(windows),
		}
		require.NoError(t, db.Create(plan).Error)

		user := &User{
			Username:    "winuser5",
			Email:       "win5@example.com",
			Password:    "hash",
			AccessToken: ptrStr("test-token-win5"),
			Group:       "default",
		}
		require.NoError(t, db.Create(user).Error)

		sub, err := CreateUserSubscriptionFromPlanTx(db, user.Id, plan, "admin")
		require.NoError(t, err)
		require.NotNil(t, sub)

		result1, err := PreConsumeUserSubscription("req-win5-dup", user.Id, "gpt-4", 1, 100)
		require.NoError(t, err)
		assert.True(t, result1.Windowed)
		assert.Equal(t, int64(100), result1.CurrentConsumed)

		result2, err := PreConsumeUserSubscription("req-win5-dup", user.Id, "gpt-4", 1, 100)
		require.NoError(t, err)
		assert.True(t, result2.Windowed)
		assert.Equal(t, int64(100), result2.CurrentConsumed, "idempotent: same CurrentConsumed")
		assert.Equal(t, result1.QuotaWindowSnapshots.Slice()[0].WindowStart, result2.QuotaWindowSnapshots.Slice()[0].WindowStart, "idempotent: same snapshot")

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(100), states[0].Used, "should not double-consume")
		assert.Equal(t, int64(100), states[1].Used, "should not double-consume")
	})

	t.Run("refund windowed subscription decrements window used", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		origDB := DB
		DB = db
		defer func() { DB = origDB }()

		initCol()
		require.NoError(t, db.AutoMigrate(&User{}, &SubscriptionPlan{}, &UserSubscription{}, &SubscriptionPreConsumeRecord{}))

		windows := []QuotaWindow{
			{Name: "5H", DurationSeconds: 5 * 3600, Quota: 1000},
			{Name: "7D", DurationSeconds: 7 * 86400, Quota: 5000},
		}
		plan := &SubscriptionPlan{
			Title:         "Windowed Plan",
			PriceAmount:   9.99,
			DurationUnit:  SubscriptionDurationMonth,
			DurationValue: 1,
			QuotaWindows:  NewQuotaWindowList(windows),
		}
		require.NoError(t, db.Create(plan).Error)

		user := &User{
			Username:    "winuser6",
			Email:       "win6@example.com",
			Password:    "hash",
			AccessToken: ptrStr("test-token-win6"),
			Group:       "default",
		}
		require.NoError(t, db.Create(user).Error)

		sub, err := CreateUserSubscriptionFromPlanTx(db, user.Id, plan, "admin")
		require.NoError(t, err)
		require.NotNil(t, sub)

		_, err = PreConsumeUserSubscription("req-win6-refund", user.Id, "gpt-4", 1, 100)
		require.NoError(t, err)

		// Verify used is 100
		var subBefore UserSubscription
		require.NoError(t, db.First(&subBefore, sub.Id).Error)
		assert.Equal(t, int64(100), subBefore.QuotaWindowStates.Slice()[0].Used)

		// Refund
		err = RefundSubscriptionPreConsume("req-win6-refund")
		require.NoError(t, err)

		// Verify used is back to 0
		var subAfter UserSubscription
		require.NoError(t, db.First(&subAfter, sub.Id).Error)
		states := subAfter.QuotaWindowStates.Slice()
		assert.Equal(t, int64(0), states[0].Used, "5H used should be 0 after refund")
		assert.Equal(t, int64(0), states[1].Used, "7D used should be 0 after refund")

		// Verify record is refunded
		var record SubscriptionPreConsumeRecord
		require.NoError(t, db.Where("request_id = ?", "req-win6-refund").First(&record).Error)
		assert.Equal(t, "refunded", record.Status)
	})
}

// ============================================================================
// TestResetDueSubscriptions_LegacyOnly — windowed subs must not be reset
// ============================================================================

func TestResetDueSubscriptions_LegacyOnly(t *testing.T) {
	origSQLite := common.UsingSQLite
	origRedis := common.RedisEnabled
	common.UsingSQLite = true
	common.RedisEnabled = false
	defer func() {
		common.UsingSQLite = origSQLite
		common.RedisEnabled = origRedis
	}()

	t.Run("only legacy subscription is reset; windowed subscription is untouched", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err, "open in-memory SQLite")
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		origDB := DB
		DB = db
		defer func() { DB = origDB }()

		// Reset the plan cache to prevent stale entries from previous tests
		subscriptionPlanCacheOnce = sync.Once{}
		subscriptionPlanInfoCacheOnce = sync.Once{}
		initCol()
		require.NoError(t, db.AutoMigrate(&User{}, &SubscriptionPlan{}, &UserSubscription{}, &SubscriptionPreConsumeRecord{}), "AutoMigrate")

		now := time.Now().Unix()

		// --- Legacy subscription with reset due ---
		legacyPlan := &SubscriptionPlan{
			Title:            "Legacy Monthly",
			PriceAmount:      9.99,
			DurationUnit:     SubscriptionDurationMonth,
			DurationValue:    1,
			TotalAmount:      50000,
			QuotaResetPeriod: SubscriptionResetMonthly,
		}
		require.NoError(t, db.Create(legacyPlan).Error, "create legacy plan")

		legacyUser := &User{
			Username:    "legacy-user",
			Email:       "legacy@test.com",
			Password:    "hash",
			AccessToken: ptrStr("legacy-token"),
			Group:       "default",
			AffCode:     "leg1",
		}
		require.NoError(t, db.Create(legacyUser).Error, "create legacy user")

		legacySub, err := CreateUserSubscriptionFromPlanTx(db, legacyUser.Id, legacyPlan, "admin")
		if err != nil {
			t.Fatalf("CreateUserSubscriptionFromPlanTx legacy: %v", err)
		}
		if legacySub == nil {
			t.Fatal("legacySub is nil")
		}

		// Set legacy sub as due for reset: AmountUsed > 0, NextResetTime in the past
		// Must also set LastResetTime in the past so calcNextResetTime produces a due time
		require.NoError(t, db.Model(&UserSubscription{}).Where("id = ?", legacySub.Id).Updates(map[string]interface{}{
			"amount_used":     int64(10000),
			"last_reset_time": now - 3600*24*35, // 35 days ago
			"next_reset_time": now - 3600*24*5,  // 5 days ago
		}).Error, "set legacy sub due for reset")

		// --- Windowed subscription that would "look due" if next_reset_time were set ---
		windows := []QuotaWindow{
			{Name: "5H", DurationSeconds: 5 * 3600, Quota: 1000},
			{Name: "7D", DurationSeconds: 7 * 86400, Quota: 5000},
		}
		windowedPlan := &SubscriptionPlan{
			Title:            "Windowed Plan",
			PriceAmount:      19.99,
			DurationUnit:     SubscriptionDurationMonth,
			DurationValue:    1,
			TotalAmount:      50000,
			QuotaWindows:     NewQuotaWindowList(windows),
			QuotaResetPeriod: SubscriptionResetMonthly,
		}
		require.NoError(t, db.Create(windowedPlan).Error, "create windowed plan")

		windowedUser := &User{
			Username:    "windowed-user",
			Email:       "windowed@test.com",
			Password:    "hash",
			AccessToken: ptrStr("windowed-token"),
			Group:       "default",
			AffCode:     "win1",
		}
		require.NoError(t, db.Create(windowedUser).Error, "create windowed user")

		windowedSub, err := CreateUserSubscriptionFromPlanTx(db, windowedUser.Id, windowedPlan, "admin")
		if err != nil {
			t.Fatalf("CreateUserSubscriptionFromPlanTx windowed: %v", err)
		}
		if windowedSub == nil {
			t.Fatal("windowedSub is nil")
		}

		// Windowed sub should have next_reset_time=0 by creation (Task 3 guarantee)
		var windowedCheck UserSubscription
		require.NoError(t, db.First(&windowedCheck, windowedSub.Id).Error)
		assert.Equal(t, int64(0), windowedCheck.NextResetTime, "windowed sub should have next_reset_time=0")

		// --- Run ResetDueSubscriptions ---
		resetCount, err := ResetDueSubscriptions(100)
		require.NoError(t, err)
		assert.Equal(t, 1, resetCount, "only the legacy sub should be reset")

		// --- Verify legacy sub was reset ---
		var legacyAfter UserSubscription
		require.NoError(t, db.First(&legacyAfter, legacySub.Id).Error)
		assert.Equal(t, int64(0), legacyAfter.AmountUsed, "legacy sub AmountUsed should be reset to 0")

		// --- Verify windowed sub was NOT reset (next_reset_time still 0, states untouched) ---
		var windowedAfter UserSubscription
		require.NoError(t, db.First(&windowedAfter, windowedSub.Id).Error)
		assert.Equal(t, int64(0), windowedAfter.NextResetTime, "windowed sub next_reset_time should remain 0")
		assert.Equal(t, int64(0), windowedAfter.AmountUsed, "windowed sub AmountUsed should remain 0")
	})
}

// ============================================================================
// TestExpireDueSubscriptions_QuotaWindows — windowed subs expire like any other
// ============================================================================

func TestExpireDueSubscriptions_QuotaWindows(t *testing.T) {
	common.UsingSQLite = true
	common.RedisEnabled = false

	t.Run("windowed subscription is expired when end_time passes", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err, "open in-memory SQLite")
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		origDB := DB
		DB = db
		defer func() { DB = origDB }()

		initCol()
		require.NoError(t, db.AutoMigrate(&User{}, &SubscriptionPlan{}, &UserSubscription{}, &SubscriptionPreConsumeRecord{}), "AutoMigrate")

		windows := []QuotaWindow{
			{Name: "5H", DurationSeconds: 5 * 3600, Quota: 1000},
			{Name: "7D", DurationSeconds: 7 * 86400, Quota: 5000},
		}
		plan := &SubscriptionPlan{
			Title:         "Windowed Plan",
			PriceAmount:   9.99,
			DurationUnit:  SubscriptionDurationDay,
			DurationValue: 1,
			TotalAmount:   50000,
			QuotaWindows:  NewQuotaWindowList(windows),
		}
		require.NoError(t, db.Create(plan).Error, "create plan")

		user := &User{
			Username:    "expire-test-user",
			Email:       "expire@test.com",
			Password:    "hash",
			AccessToken: ptrStr("expire-token"),
			Group:       "default",
			AffCode:     "exp1",
		}
		require.NoError(t, db.Create(user).Error, "create user")

		sub, err := CreateUserSubscriptionFromPlanTx(db, user.Id, plan, "admin")
		require.NoError(t, err)
		require.NotNil(t, sub)

		// Manually set end_time to the past so it's due for expiry
		now := time.Now().Unix()
		require.NoError(t, db.Model(&UserSubscription{}).Where("id = ?", sub.Id).Updates(map[string]interface{}{
			"end_time": now - 100,
		}).Error, "set end_time to past")

		// Run ExpireDueSubscriptions
		expiredCount, err := ExpireDueSubscriptions(100)
		require.NoError(t, err)
		assert.Equal(t, 1, expiredCount, "windowed sub should be expired")

		// Verify the subscription is now expired
		var expired UserSubscription
		require.NoError(t, db.First(&expired, sub.Id).Error)
		assert.Equal(t, "expired", expired.Status, "windowed sub should have status=expired")
	})
}

// ============================================================================
// TestBuildSubscriptionSummariesMalformedData — display-safe normalization
// ============================================================================

func TestBuildSubscriptionSummariesMalformedData(t *testing.T) {
	t.Run("invalid QuotaWindows is normalized to empty", func(t *testing.T) {
		subs := []UserSubscription{
			{
				Id:                1,
				UserId:            100,
				QuotaWindows:      QuotaWindowList{windows: nil, invalid: true},
				QuotaWindowStates: NewQuotaWindowStateList(nil),
			},
		}
		result := buildSubscriptionSummaries(subs)
		require.Len(t, result, 1)
		assert.False(t, result[0].Subscription.QuotaWindows.IsInvalid(), "invalid flag should be cleared")
		assert.Empty(t, result[0].Subscription.QuotaWindows.Slice(), "windows should be empty")
	})

	t.Run("invalid QuotaWindowStates is normalized to empty", func(t *testing.T) {
		subs := []UserSubscription{
			{
				Id:                2,
				UserId:            100,
				QuotaWindows:      NewQuotaWindowList(nil),
				QuotaWindowStates: QuotaWindowStateList{states: nil, invalid: true},
			},
		}
		result := buildSubscriptionSummaries(subs)
		require.Len(t, result, 1)
		assert.False(t, result[0].Subscription.QuotaWindowStates.IsInvalid(), "invalid flag should be cleared")
		assert.Empty(t, result[0].Subscription.QuotaWindowStates.Slice(), "states should be empty")
	})

	t.Run("both invalid are normalized", func(t *testing.T) {
		subs := []UserSubscription{
			{
				Id:                3,
				UserId:            100,
				QuotaWindows:      QuotaWindowList{windows: nil, invalid: true},
				QuotaWindowStates: QuotaWindowStateList{states: nil, invalid: true},
			},
		}
		result := buildSubscriptionSummaries(subs)
		require.Len(t, result, 1)
		assert.False(t, result[0].Subscription.QuotaWindows.IsInvalid())
		assert.False(t, result[0].Subscription.QuotaWindowStates.IsInvalid())
		assert.Empty(t, result[0].Subscription.QuotaWindows.Slice())
		assert.Empty(t, result[0].Subscription.QuotaWindowStates.Slice())
	})

	t.Run("valid data is preserved", func(t *testing.T) {
		windows := []QuotaWindow{{Name: "5H", DurationSeconds: 18000, Quota: 1000}}
		states := []QuotaWindowState{{Name: "5H", DurationSeconds: 18000, Quota: 1000, Used: 200, WindowStart: 1000}}
		subs := []UserSubscription{
			{
				Id:                4,
				UserId:            100,
				QuotaWindows:      NewQuotaWindowList(windows),
				QuotaWindowStates: NewQuotaWindowStateList(states),
			},
		}
		result := buildSubscriptionSummaries(subs)
		require.Len(t, result, 1)
		assert.False(t, result[0].Subscription.QuotaWindows.IsInvalid())
		assert.Len(t, result[0].Subscription.QuotaWindows.Slice(), 1)
		assert.Equal(t, "5H", result[0].Subscription.QuotaWindows.Slice()[0].Name)
		assert.Len(t, result[0].Subscription.QuotaWindowStates.Slice(), 1)
		assert.Equal(t, int64(200), result[0].Subscription.QuotaWindowStates.Slice()[0].Used)
	})

	t.Run("normalized data marshals as JSON arrays not strings", func(t *testing.T) {
		subs := []UserSubscription{
			{
				Id:                5,
				UserId:            100,
				QuotaWindows:      QuotaWindowList{windows: nil, invalid: true},
				QuotaWindowStates: QuotaWindowStateList{states: nil, invalid: true},
			},
		}
		result := buildSubscriptionSummaries(subs)
		data, err := json.Marshal(result)
		require.NoError(t, err)
		s := string(data)
		assert.Contains(t, s, `"quota_windows":[]`, "should marshal as empty JSON array")
		assert.Contains(t, s, `"quota_window_states":[]`, "should marshal as empty JSON array")
		assert.NotContains(t, s, `"quota_windows":"`, "must NOT be a JSON string")
		assert.NotContains(t, s, `"quota_window_states":"`, "must NOT be a JSON string")
	})
}
