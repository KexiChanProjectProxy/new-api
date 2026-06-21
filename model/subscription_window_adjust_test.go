package model

import (
	"errors"
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupWindowedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	common.UsingSQLite = true
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	origDB := DB
	DB = db
	t.Cleanup(func() { DB = origDB })

	initCol()
	require.NoError(t, db.AutoMigrate(&User{}, &SubscriptionPlan{}, &UserSubscription{}, &SubscriptionPreConsumeRecord{}, &Task{}))

	return db
}

func createWindowedPlanAndSub(t *testing.T, db *gorm.DB, username string) (*SubscriptionPlan, *UserSubscription) {
	t.Helper()

	windows := []QuotaWindow{
		{Name: "5H", DurationSeconds: 5 * 3600, Quota: 10000},
		{Name: "7D", DurationSeconds: 7 * 86400, Quota: 50000},
	}
	plan := &SubscriptionPlan{
		Title:         "Windowed Plan for " + username,
		PriceAmount:   9.99,
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		QuotaWindows:  NewQuotaWindowList(windows),
	}
	require.NoError(t, db.Create(plan).Error)

	user := &User{
		Username:    username,
		Email:       username + "@example.com",
		Password:    "hash",
		AccessToken: ptrStr("token-" + username),
		Group:       "default",
	}
	require.NoError(t, db.Create(user).Error)

	sub, err := CreateUserSubscriptionFromPlanTx(db, user.Id, plan, "admin")
	require.NoError(t, err)
	require.NotNil(t, sub)

	return plan, sub
}

func createLegacyPlanAndSub(t *testing.T, db *gorm.DB, username string) (*SubscriptionPlan, *UserSubscription) {
	t.Helper()

	plan := &SubscriptionPlan{
		Title:            "Legacy Plan for " + username,
		PriceAmount:      9.99,
		DurationUnit:     SubscriptionDurationMonth,
		DurationValue:    1,
		QuotaResetPeriod: SubscriptionResetMonthly,
	}
	require.NoError(t, db.Create(plan).Error)

	user := &User{
		Username:    username,
		Email:       username + "@example.com",
		Password:    "hash",
		AccessToken: ptrStr("token-" + username),
		Group:       "default",
	}
	require.NoError(t, db.Create(user).Error)

	sub, err := CreateUserSubscriptionFromPlanTx(db, user.Id, plan, "admin")
	require.NoError(t, err)
	require.NotNil(t, sub)

	return plan, sub
}

// ============================================================================
// TestSetSubscriptionRequestConsumedAmountTx
// ============================================================================

func TestSetSubscriptionRequestConsumedAmountTx_QuotaWindows(t *testing.T) {
	t.Run("settle lower amount refunds window used", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "settle-lower")

		result, err := PreConsumeUserSubscription("req-settle-lower", sub.UserId, "gpt-4", 1, 500)
		require.NoError(t, err)
		assert.True(t, result.Windowed)
		assert.Equal(t, int64(500), result.CurrentConsumed)

		// Settle to 300 (refund 200)
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-settle-lower", sub.Id, 300, "settled_adjusted")
		require.NoError(t, err)

		var record SubscriptionPreConsumeRecord
		require.NoError(t, db.Where("request_id = ?", "req-settle-lower").First(&record).Error)
		assert.Equal(t, int64(300), record.CurrentConsumed)
		assert.Equal(t, int64(300), record.FinalConsumed)
		assert.Equal(t, "settled_adjusted", record.Status)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(300), states[0].Used)
		assert.Equal(t, int64(300), states[1].Used)
	})

	t.Run("settle equal amount is idempotent", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "settle-equal")

		result, err := PreConsumeUserSubscription("req-settle-equal", sub.UserId, "gpt-4", 1, 500)
		require.NoError(t, err)
		assert.Equal(t, int64(500), result.CurrentConsumed)

		// Settle to 500 (no change)
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-settle-equal", sub.Id, 500, "settled_adjusted")
		require.NoError(t, err)

		// Call again — should be idempotent
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-settle-equal", sub.Id, 500, "settled_adjusted")
		require.NoError(t, err)

		var record SubscriptionPreConsumeRecord
		require.NoError(t, db.Where("request_id = ?", "req-settle-equal").First(&record).Error)
		assert.Equal(t, int64(500), record.CurrentConsumed)
		assert.Equal(t, "settled_adjusted", record.Status)
	})

	t.Run("settle higher amount charges more", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "settle-higher")

		result, err := PreConsumeUserSubscription("req-settle-higher", sub.UserId, "gpt-4", 1, 300)
		require.NoError(t, err)
		assert.Equal(t, int64(300), result.CurrentConsumed)

		// Settle to 700 (charge 200 more)
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-settle-higher", sub.Id, 700, "settled_adjusted")
		require.NoError(t, err)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(700), states[0].Used)
		assert.Equal(t, int64(700), states[1].Used)
	})

	t.Run("positive delta after window reset returns error", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "expired-pos")
		_, err := PreConsumeUserSubscription("req-expired-pos", sub.UserId, "gpt-4", 1, 100)
		require.NoError(t, err)

		// Simulate window reset by modifying window_start to the past
		var subBefore UserSubscription
		require.NoError(t, db.First(&subBefore, sub.Id).Error)
		states := subBefore.QuotaWindowStates.Slice()
		for i := range states {
			states[i].WindowStart = 1 // far in the past → window will have "expired"
		}
		require.NoError(t, db.Model(&UserSubscription{}).Where("id = ?", sub.Id).Updates(map[string]interface{}{
			"quota_window_states": NewQuotaWindowStateList(states),
		}).Error)

		// Try to increase consumed amount — should fail because window expired
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-expired-pos", sub.Id, 200, "")
		assert.True(t, errors.Is(err, ErrSubscriptionWindowExpiredForAdjustment))
	})

	t.Run("negative delta after window reset is safe no-op", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "expired-neg")
		_, err := PreConsumeUserSubscription("req-expired-neg", sub.UserId, "gpt-4", 1, 500)
		require.NoError(t, err)

		// Simulate window reset
		var subBefore UserSubscription
		require.NoError(t, db.First(&subBefore, sub.Id).Error)
		states := subBefore.QuotaWindowStates.Slice()
		for i := range states {
			states[i].WindowStart = 1
		}
		require.NoError(t, db.Model(&UserSubscription{}).Where("id = ?", sub.Id).Updates(map[string]interface{}{
			"quota_window_states": NewQuotaWindowStateList(states),
		}).Error)

		// Refund (negative delta) after window reset — should succeed (safe no-op)
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-expired-neg", sub.Id, 200, "")
		require.NoError(t, err)

		var record SubscriptionPreConsumeRecord
		require.NoError(t, db.Where("request_id = ?", "req-expired-neg").First(&record).Error)
		assert.Equal(t, int64(200), record.CurrentConsumed)
	})

	t.Run("legacy subscription uses PostConsumeUserSubscriptionDelta", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createLegacyPlanAndSub(t, db, "legacy-settle")

		result, err := PreConsumeUserSubscription("req-legacy-settle", sub.UserId, "gpt-4", 1, 300)
		require.NoError(t, err)
		assert.False(t, result.Windowed)

		var subBefore UserSubscription
		require.NoError(t, db.First(&subBefore, sub.Id).Error)
		t.Logf("After PreConsume: AmountUsed=%d, AmountTotal=%d, result.AmountUsedAfter=%d", subBefore.AmountUsed, subBefore.AmountTotal, result.AmountUsedAfter)

		// Settle to 200 (refund 100)
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-legacy-settle", sub.Id, 200, "settled_adjusted")
		require.NoError(t, err)

		var record SubscriptionPreConsumeRecord
		require.NoError(t, db.Where("request_id = ?", "req-legacy-settle").First(&record).Error)
		assert.Equal(t, int64(200), record.CurrentConsumed)
		assert.Equal(t, "settled_adjusted", record.Status)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		assert.Equal(t, int64(200), subCheck.AmountUsed)
	})

	t.Run("cannot adjust refunded record", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "refused-adj")

		_, err := PreConsumeUserSubscription("req-refused-adj", sub.UserId, "gpt-4", 1, 100)
		require.NoError(t, err)

		// Refund the pre-consume
		err = RefundSubscriptionPreConsume("req-refused-adj")
		require.NoError(t, err)

		// Try to adjust — should fail
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-refused-adj", sub.Id, 50, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "refunded")
	})

	t.Run("repeated calls are idempotent", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "idempotent")

		_, err := PreConsumeUserSubscription("req-idempotent", sub.UserId, "gpt-4", 1, 500)
		require.NoError(t, err)

		// Call 3 times with same target
		for i := 0; i < 3; i++ {
			err = SetSubscriptionRequestConsumedAmountTx(db, "req-idempotent", sub.Id, 300, "settled_adjusted")
			require.NoError(t, err)
		}

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(300), states[0].Used, "should not double-apply")
		assert.Equal(t, int64(300), states[1].Used, "should not double-apply")
	})
}

// ============================================================================
// TestSubscriptionFundingSettle_QuotaWindows
// ============================================================================

func TestSubscriptionFundingSettle_QuotaWindows(t *testing.T) {
	t.Run("windowed settle uses target-based adjustment", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "funding-settle")

		result, err := PreConsumeUserSubscription("req-funding-settle", sub.UserId, "gpt-4", 1, 400)
		require.NoError(t, err)

		// Simulate what SubscriptionFunding.Settle would do for windowed
		targetAmount := result.CurrentConsumed + 100 // delta = +100
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-funding-settle", sub.Id, targetAmount, "settled_adjusted")
		require.NoError(t, err)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(500), states[0].Used)
		assert.Equal(t, int64(500), states[1].Used)
	})
}

// ============================================================================
// TestBillingSessionReserve_QuotaWindows
// ============================================================================

func TestBillingSessionReserve_QuotaWindows(t *testing.T) {
	t.Run("reserve adds delta via target-based adjustment", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "reserve")

		result, err := PreConsumeUserSubscription("req-reserve", sub.UserId, "gpt-4", 1, 300)
		require.NoError(t, err)
		assert.Equal(t, int64(300), result.CurrentConsumed)

		// Simulate reserve: adding 200 more
		targetAmount := result.CurrentConsumed + 200
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-reserve", sub.Id, targetAmount, "")
		require.NoError(t, err)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(500), states[0].Used)
		assert.Equal(t, int64(500), states[1].Used)
	})

	t.Run("reserve rollback refunds via target-based adjustment", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "reserve-rollback")

		result, err := PreConsumeUserSubscription("req-reserve-rollback", sub.UserId, "gpt-4", 1, 300)
		require.NoError(t, err)

		// Reserve 200 more
		targetAfterReserve := result.CurrentConsumed + 200
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-reserve-rollback", sub.Id, targetAfterReserve, "")
		require.NoError(t, err)

		// Rollback the reserve (go back to 300)
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-reserve-rollback", sub.Id, result.CurrentConsumed, "")
		require.NoError(t, err)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(300), states[0].Used, "rollback should restore to original")
		assert.Equal(t, int64(300), states[1].Used, "rollback should restore to original")
	})

	t.Run("failed reserve does not double-credit", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "no-double")

		result, err := PreConsumeUserSubscription("req-no-double", sub.UserId, "gpt-4", 1, 300)
		require.NoError(t, err)

		// Call with same target multiple times
		for i := 0; i < 3; i++ {
			err = SetSubscriptionRequestConsumedAmountTx(db, "req-no-double", sub.Id, result.CurrentConsumed, "")
			require.NoError(t, err)
		}

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(300), states[0].Used, "idempotent — no double-credit")
	})
}

// ============================================================================
// TestBillingSessionSettleEqualTarget_QuotaWindows
// ============================================================================

func TestBillingSessionSettleEqualTarget_QuotaWindows(t *testing.T) {
	t.Run("settle with delta=0 is idempotent", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "settle-eq")

		result, err := PreConsumeUserSubscription("req-settle-eq", sub.UserId, "gpt-4", 1, 500)
		require.NoError(t, err)

		// Settle with target = current (no delta)
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-settle-eq", sub.Id, result.CurrentConsumed, "settled_adjusted")
		require.NoError(t, err)

		// Call again
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-settle-eq", sub.Id, result.CurrentConsumed, "settled_adjusted")
		require.NoError(t, err)

		var record SubscriptionPreConsumeRecord
		require.NoError(t, db.Where("request_id = ?", "req-settle-eq").First(&record).Error)
		assert.Equal(t, int64(500), record.CurrentConsumed)
		assert.Equal(t, "settled_adjusted", record.Status)
	})
}

// ============================================================================
// TestBillingSessionExtraReserveRefund_QuotaWindows
// ============================================================================

func TestBillingSessionExtraReserveRefund_QuotaWindows(t *testing.T) {
	t.Run("extra reserve refund uses target-based", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "extra-ref")

		result, err := PreConsumeUserSubscription("req-extra-ref", sub.UserId, "gpt-4", 1, 300)
		require.NoError(t, err)

		// Extra reserve of 200
		targetAfterReserve := result.CurrentConsumed + 200
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-extra-ref", sub.Id, targetAfterReserve, "")
		require.NoError(t, err)

		// Refund the extra reserve (subtract 200 from current)
		targetAfterRefund := targetAfterReserve - 200
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-extra-ref", sub.Id, targetAfterRefund, "")
		require.NoError(t, err)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(300), states[0].Used)
		assert.Equal(t, int64(300), states[1].Used)
	})
}

// ============================================================================
// TestPostConsumeQuota_QuotaWindows
// ============================================================================

func TestPostConsumeQuota_QuotaWindows(t *testing.T) {
	t.Run("windowed post-consume uses target-based adjustment", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "post-consume")

		result, err := PreConsumeUserSubscription("req-post-consume", sub.UserId, "gpt-4", 1, 300)
		require.NoError(t, err)

		// Simulate PostConsumeQuota with delta = +100
		delta := int64(100)
		targetAmount := result.CurrentConsumed + delta
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-post-consume", sub.Id, targetAmount, "")
		require.NoError(t, err)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(400), states[0].Used)
		assert.Equal(t, int64(400), states[1].Used)
	})

	t.Run("negative delta post-consume refunds window used", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "post-refund")

		result, err := PreConsumeUserSubscription("req-post-refund", sub.UserId, "gpt-4", 1, 500)
		require.NoError(t, err)

		// Simulate PostConsumeQuota with delta = -200
		delta := int64(-200)
		targetAmount := result.CurrentConsumed + delta
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-post-refund", sub.Id, targetAmount, "")
		require.NoError(t, err)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(300), states[0].Used)
		assert.Equal(t, int64(300), states[1].Used)
	})
}

// ============================================================================
// TestViolationFeeQuotaWindows
// ============================================================================

func TestViolationFeeQuotaWindows(t *testing.T) {
	t.Run("violation fee uses separate request_id", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "violation")

		// Normal request pre-consume
		result, err := PreConsumeUserSubscription("req-violation", sub.UserId, "gpt-4", 1, 300)
		require.NoError(t, err)
		assert.Equal(t, int64(300), result.CurrentConsumed)

		// Violation fee uses separate request_id
		violationRequestId := "req-violation:violation_fee"
		vResult, err := PreConsumeUserSubscription(violationRequestId, sub.UserId, "gpt-4", 1, 100)
		require.NoError(t, err)
		assert.Equal(t, int64(100), vResult.CurrentConsumed)

		// Refund original request — should not affect violation fee
		err = RefundSubscriptionPreConsume("req-violation")
		require.NoError(t, err)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		// Original 300 refunded, violation fee 100 still present
		assert.Equal(t, int64(100), states[0].Used, "violation fee should remain")
		assert.Equal(t, int64(100), states[1].Used, "violation fee should remain")

		// Verify violation fee record still exists
		var violationRecord SubscriptionPreConsumeRecord
		require.NoError(t, db.Where("request_id = ?", violationRequestId).First(&violationRecord).Error)
		assert.Equal(t, "consumed", violationRecord.Status)
	})

	t.Run("fee before refund ordering is safe", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "fee-order")

		_, err := PreConsumeUserSubscription("req-fee-order", sub.UserId, "gpt-4", 1, 200)
		require.NoError(t, err)

		// Charge violation fee first
		_, err = PreConsumeUserSubscription("req-fee-order:violation_fee", sub.UserId, "gpt-4", 1, 50)
		require.NoError(t, err)

		// Then refund original
		err = RefundSubscriptionPreConsume("req-fee-order")
		require.NoError(t, err)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(50), states[0].Used, "only violation fee remains")
	})
}

// ============================================================================
// TestTaskAdjustFunding_QuotaWindows
// ============================================================================

func TestTaskAdjustFunding_QuotaWindows(t *testing.T) {
	t.Run("task adjustment uses SetTaskSubscriptionWindowAmountTx", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "task-adj")

		result, err := PreConsumeUserSubscription("req-task-adj", sub.UserId, "gpt-4", 1, 500)
		require.NoError(t, err)

		// Create a task with window snapshot data
		task := &Task{
			TaskID:    "task_test_adj",
			UserId:    sub.UserId,
			Group:     "default",
			ChannelId: 1,
			PrivateData: TaskPrivateData{
				BillingSource:                   "subscription",
				SubscriptionId:                  sub.Id,
				SubscriptionRequestId:           "req-task-adj",
				SubscriptionWindowSnapshots:     result.QuotaWindowSnapshots,
				SubscriptionWindowAppliedAmount: result.CurrentConsumed,
				TokenId:                         1,
			},
		}
		require.NoError(t, db.Create(task).Error)

		// Adjust to 300 (refund 200)
		err = SetTaskSubscriptionWindowAmountTx(db, task, 300)
		require.NoError(t, err)
		assert.Equal(t, int64(300), task.PrivateData.SubscriptionWindowAppliedAmount)
		assert.Equal(t, fmt.Sprintf("%d:%d", task.ID, int64(300)), task.PrivateData.SubscriptionWindowAdjustmentMarker)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(300), states[0].Used)
		assert.Equal(t, int64(300), states[1].Used)
	})

	t.Run("repeated task adjustment calls are idempotent", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "task-idem")

		result, err := PreConsumeUserSubscription("req-task-idem", sub.UserId, "gpt-4", 1, 500)
		require.NoError(t, err)

		task := &Task{
			TaskID:    "task_test_idem",
			UserId:    sub.UserId,
			Group:     "default",
			ChannelId: 1,
			PrivateData: TaskPrivateData{
				BillingSource:                   "subscription",
				SubscriptionId:                  sub.Id,
				SubscriptionRequestId:           "req-task-idem",
				SubscriptionWindowSnapshots:     result.QuotaWindowSnapshots,
				SubscriptionWindowAppliedAmount: result.CurrentConsumed,
				TokenId:                         1,
			},
		}
		require.NoError(t, db.Create(task).Error)

		// Adjust to 300 three times
		for i := 0; i < 3; i++ {
			err = SetTaskSubscriptionWindowAmountTx(db, task, 300)
			require.NoError(t, err)
		}

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(300), states[0].Used, "should not double-apply")
		assert.Equal(t, int64(300), states[1].Used, "should not double-apply")
	})

	t.Run("legacy task uses PostConsumeUserSubscriptionDelta", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createLegacyPlanAndSub(t, db, "task-legacy")

		result, err := PreConsumeUserSubscription("req-task-legacy", sub.UserId, "gpt-4", 1, 300)
		require.NoError(t, err)
		assert.False(t, result.Windowed)

		task := &Task{
			TaskID:    "task_test_legacy",
			UserId:    sub.UserId,
			Group:     "default",
			ChannelId: 1,
			PrivateData: TaskPrivateData{
				BillingSource:                   "subscription",
				SubscriptionId:                  sub.Id,
				SubscriptionWindowAppliedAmount: result.CurrentConsumed,
				TokenId:                         1,
			},
		}
		require.NoError(t, db.Create(task).Error)

		err = SetTaskSubscriptionWindowAmountTx(db, task, 200)
		require.NoError(t, err)
		assert.Equal(t, int64(200), task.PrivateData.SubscriptionWindowAppliedAmount)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		assert.Equal(t, int64(200), subCheck.AmountUsed)
	})
}

// ============================================================================
// TestSubscriptionWindowAdjustmentStateMachine
// ============================================================================

func TestSubscriptionWindowAdjustmentStateMachine(t *testing.T) {
	t.Run("consumed -> settled_adjusted -> no further adjustment", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "sm")

		_, err := PreConsumeUserSubscription("req-sm", sub.UserId, "gpt-4", 1, 500)
		require.NoError(t, err)

		// Settle to 400
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-sm", sub.Id, 400, "settled_adjusted")
		require.NoError(t, err)

		var record SubscriptionPreConsumeRecord
		require.NoError(t, db.Where("request_id = ?", "req-sm").First(&record).Error)
		assert.Equal(t, "settled_adjusted", record.Status)
		assert.Equal(t, int64(400), record.CurrentConsumed)
		assert.Equal(t, int64(400), record.FinalConsumed)

		// Try to adjust again to a different target — should still work (it's not refunded)
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-sm", sub.Id, 300, "settled_adjusted")
		require.NoError(t, err)

		require.NoError(t, db.Where("request_id = ?", "req-sm").First(&record).Error)
		assert.Equal(t, int64(300), record.CurrentConsumed)
	})

	t.Run("consumed -> refunded blocks further adjustment", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "sm-refund")

		_, err := PreConsumeUserSubscription("req-sm-refund", sub.UserId, "gpt-4", 1, 500)
		require.NoError(t, err)

		err = RefundSubscriptionPreConsume("req-sm-refund")
		require.NoError(t, err)

		// Should not be able to adjust after refund
		err = SetSubscriptionRequestConsumedAmountTx(db, "req-sm-refund", sub.Id, 300, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "refunded")
	})
}

// ============================================================================
// TestRefundTaskQuota_Subscription
// ============================================================================

func TestRefundTaskQuota_Subscription(t *testing.T) {
	t.Run("refund task via SetTaskSubscriptionWindowAmountTx", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "task-refund")

		result, err := PreConsumeUserSubscription("req-task-refund", sub.UserId, "gpt-4", 1, 500)
		require.NoError(t, err)

		task := &Task{
			TaskID:    "task_test_refund",
			UserId:    sub.UserId,
			Group:     "default",
			ChannelId: 1,
			Quota:     500,
			PrivateData: TaskPrivateData{
				BillingSource:                   "subscription",
				SubscriptionId:                  sub.Id,
				SubscriptionRequestId:           "req-task-refund",
				SubscriptionWindowSnapshots:     result.QuotaWindowSnapshots,
				SubscriptionWindowAppliedAmount: result.CurrentConsumed,
				TokenId:                         1,
			},
		}
		require.NoError(t, db.Create(task).Error)

		// Full refund: set applied amount to 0
		err = SetTaskSubscriptionWindowAmountTx(db, task, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(0), task.PrivateData.SubscriptionWindowAppliedAmount)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(0), states[0].Used)
		assert.Equal(t, int64(0), states[1].Used)
	})

	t.Run("partial refund via task snapshot", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "task-partial")

		result, err := PreConsumeUserSubscription("req-task-partial", sub.UserId, "gpt-4", 1, 500)
		require.NoError(t, err)

		task := &Task{
			TaskID:    "task_test_partial",
			UserId:    sub.UserId,
			Group:     "default",
			ChannelId: 1,
			Quota:     500,
			PrivateData: TaskPrivateData{
				BillingSource:                   "subscription",
				SubscriptionId:                  sub.Id,
				SubscriptionRequestId:           "req-task-partial",
				SubscriptionWindowSnapshots:     result.QuotaWindowSnapshots,
				SubscriptionWindowAppliedAmount: result.CurrentConsumed,
				TokenId:                         1,
			},
		}
		require.NoError(t, db.Create(task).Error)

		// Refund 200 from 500
		err = SetTaskSubscriptionWindowAmountTx(db, task, 300)
		require.NoError(t, err)
		assert.Equal(t, int64(300), task.PrivateData.SubscriptionWindowAppliedAmount)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(300), states[0].Used)
		assert.Equal(t, int64(300), states[1].Used)
	})
}

// ============================================================================
// TestRefundSubscriptionPreConsume_QuotaWindows
// ============================================================================

func TestRefundSubscriptionPreConsume_QuotaWindows(t *testing.T) {
	t.Run("windowed refund decrements all window states", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "refund-win")

		_, err := PreConsumeUserSubscription("req-refund-win", sub.UserId, "gpt-4", 1, 300)
		require.NoError(t, err)

		err = RefundSubscriptionPreConsume("req-refund-win")
		require.NoError(t, err)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(0), states[0].Used)
		assert.Equal(t, int64(0), states[1].Used)

		var record SubscriptionPreConsumeRecord
		require.NoError(t, db.Where("request_id = ?", "req-refund-win").First(&record).Error)
		assert.Equal(t, "refunded", record.Status)
	})

	t.Run("refund before fee preserves fee", func(t *testing.T) {
		db := setupWindowedTestDB(t)
		_, sub := createWindowedPlanAndSub(t, db, "refund-before-fee")

		_, err := PreConsumeUserSubscription("req-refund-before-fee", sub.UserId, "gpt-4", 1, 300)
		require.NoError(t, err)

		// Charge violation fee
		_, err = PreConsumeUserSubscription("req-refund-before-fee:violation_fee", sub.UserId, "gpt-4", 1, 100)
		require.NoError(t, err)

		// Refund original
		err = RefundSubscriptionPreConsume("req-refund-before-fee")
		require.NoError(t, err)

		var subCheck UserSubscription
		require.NoError(t, db.First(&subCheck, sub.Id).Error)
		states := subCheck.QuotaWindowStates.Slice()
		assert.Equal(t, int64(100), states[0].Used, "violation fee should remain")
		assert.Equal(t, int64(100), states[1].Used, "violation fee should remain")
	})
}
