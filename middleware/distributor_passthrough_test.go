package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	operation_setting "github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const distributorPassthroughModel = "distributor-passthrough-model"

type distributorPassthroughResult struct {
	status                int
	selectedID            int
	handlerCalled         bool
	affinityUsageRecorded bool
	body                  []byte
}

func newDistributorPassthroughFixture(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:distributor-passthrough?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}))

	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL
	originalRedisEnabled := common.RedisEnabled
	originalGlobalSettings := *model_setting.GetGlobalSettings()
	originalAffinitySetting := *operation_setting.GetChannelAffinitySetting()

	model.DB = db
	model.LOG_DB = db
	common.MemoryCacheEnabled = true
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	model_setting.GetGlobalSettings().PassThroughRequestEnabled = false
	*operation_setting.GetChannelAffinitySetting() = operation_setting.ChannelAffinitySetting{}
	service.ClearChannelAffinityCacheAll()
	model.InitChannelCache()
	require.NoError(t, i18n.Init())

	t.Cleanup(func() {
		service.ClearChannelAffinityCacheAll()
		*model_setting.GetGlobalSettings() = originalGlobalSettings
		*operation_setting.GetChannelAffinitySetting() = originalAffinitySetting
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		common.UsingSQLite = originalUsingSQLite
		common.UsingMySQL = originalUsingMySQL
		common.UsingPostgreSQL = originalUsingPostgreSQL
		common.RedisEnabled = originalRedisEnabled
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
}

func insertDistributorPassthroughChannel(t *testing.T, id int, channelType int, passthroughEnabled bool) {
	t.Helper()

	settingBytes, err := common.Marshal(dto.ChannelSettings{PassThroughBodyEnabled: passthroughEnabled})
	require.NoError(t, err)
	setting := string(settingBytes)
	priority := int64(0)
	weight := uint(0)
	channel := &model.Channel{
		Id:       id,
		Type:     channelType,
		Key:      "secret-channel-key",
		Status:   common.ChannelStatusEnabled,
		Name:     "distributor-test-channel",
		Weight:   &weight,
		Priority: &priority,
		Models:   distributorPassthroughModel,
		Group:    "default",
		Setting:  &setting,
	}
	require.NoError(t, model.DB.Create(channel).Error)
	require.NoError(t, model.DB.Create(&model.Ability{
		Group:     "default",
		Model:     distributorPassthroughModel,
		ChannelId: id,
		Enabled:   true,
		Priority:  &priority,
		Weight:    weight,
	}).Error)
	model.InitChannelCache()
}

func configureDistributorPassthroughAffinity(t *testing.T) {
	t.Helper()
	*operation_setting.GetChannelAffinitySetting() = operation_setting.ChannelAffinitySetting{
		Enabled:           true,
		MaxEntries:        100,
		DefaultTTLSeconds: 3600,
		Rules: []operation_setting.ChannelAffinityRule{{
			Name:       "distributor-passthrough-test",
			ModelRegex: []string{"^" + distributorPassthroughModel + "$"},
			PathRegex:  []string{"^/v1/responses$"},
			KeySources: []operation_setting.ChannelAffinityKeySource{{Type: "request_header", Key: "X-Affinity"}},
			TTLSeconds: 3600,
		}},
	}
}

func seedDistributorPassthroughAffinity(t *testing.T, affinityValue string, channelID int) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"distributor-passthrough-model"}`))
	request.Header.Set("X-Affinity", affinityValue)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	common.SetContextKey(context, constant.ContextKeyUsingGroup, "default")

	preferredID, found := service.GetPreferredChannelByAffinity(context, distributorPassthroughModel, "default")
	require.False(t, found)
	require.Zero(t, preferredID)
	service.RecordChannelAffinity(context, channelID)

	preferredID, found = service.GetPreferredChannelByAffinity(context, distributorPassthroughModel, "default")
	require.True(t, found)
	require.Equal(t, channelID, preferredID)
}

func runDistributorPassthroughRequest(t *testing.T, path string, tokenChannelID string, affinityValue string) distributorPassthroughResult {
	t.Helper()

	result := distributorPassthroughResult{}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		common.SetContextKey(c, constant.ContextKeyUsingGroup, "default")
		if tokenChannelID != "" {
			common.SetContextKey(c, constant.ContextKeyTokenSpecificChannelId, tokenChannelID)
		}
		c.Next()
	})
	router.Use(Distribute())
	router.POST(path, func(c *gin.Context) {
		result.handlerCalled = true
		result.selectedID = common.GetContextKeyInt(c, constant.ContextKeyChannelId)
		_, result.affinityUsageRecorded = c.Get("channel_affinity_log_info")
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{"model":"distributor-passthrough-model"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept-Language", "en")
	if affinityValue != "" {
		request.Header.Set("X-Affinity", affinityValue)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	result.status = recorder.Code
	result.body = append([]byte(nil), recorder.Body.Bytes()...)
	return result
}

func TestDistributeAffinityIncompatiblePassthroughFallsThrough(t *testing.T) {
	newDistributorPassthroughFixture(t)
	configureDistributorPassthroughAffinity(t)
	insertDistributorPassthroughChannel(t, 1, constant.ChannelTypeOpenAI, true)
	insertDistributorPassthroughChannel(t, 2, constant.ChannelTypeOpenAIResponses, true)
	affinityValue := "affinity-incompatible"
	seedDistributorPassthroughAffinity(t, affinityValue, 1)

	result := runDistributorPassthroughRequest(t, "/v1/responses", "", affinityValue)

	require.Equal(t, http.StatusNoContent, result.status)
	require.True(t, result.handlerCalled)
	require.False(t, result.affinityUsageRecorded)
	require.Equal(t, 2, result.selectedID)
}

func TestDistributeCompatibleAffinityPassthroughRemainsSticky(t *testing.T) {
	newDistributorPassthroughFixture(t)
	configureDistributorPassthroughAffinity(t)
	insertDistributorPassthroughChannel(t, 1, constant.ChannelTypeOpenAIResponses, true)
	insertDistributorPassthroughChannel(t, 2, constant.ChannelTypeOpenAI, true)
	affinityValue := "affinity-compatible"
	seedDistributorPassthroughAffinity(t, affinityValue, 1)

	result := runDistributorPassthroughRequest(t, "/v1/responses", "", affinityValue)

	require.Equal(t, http.StatusNoContent, result.status)
	require.True(t, result.handlerCalled)
	require.True(t, result.affinityUsageRecorded)
	require.Equal(t, 1, result.selectedID)
}

func TestDistributeTokenSpecificIncompatiblePassthroughReturnsLocalized400(t *testing.T) {
	newDistributorPassthroughFixture(t)
	insertDistributorPassthroughChannel(t, 1, constant.ChannelTypeOpenAI, true)

	result := runDistributorPassthroughRequest(t, "/v1/responses", "1", "")

	require.Equal(t, http.StatusBadRequest, result.status)
	require.False(t, result.handlerCalled)
	require.Zero(t, result.selectedID)
	var response struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, common.Unmarshal(result.body, &response))
	require.Equal(t, string(types.ErrorCodeInvalidRequest), response.Error.Code)
	require.Contains(t, response.Error.Message, "body passthrough")
	require.NotContains(t, response.Error.Message, "secret-channel-key")
}

func TestDistributePassthroughChineseTranslationLoads(t *testing.T) {
	require.NoError(t, i18n.Init())
	require.Equal(t, "启用请求体透传时，该令牌选择的渠道无法接收此 API 格式", i18n.Translate("zh-CN", i18n.MsgDistributorTokenChannelFormatError))
}

func TestDistributeNonPassthroughDirectSelectionRemainsCompatible(t *testing.T) {
	newDistributorPassthroughFixture(t)
	insertDistributorPassthroughChannel(t, 1, constant.ChannelTypeOpenAI, false)

	result := runDistributorPassthroughRequest(t, "/v1/responses", "1", "")

	require.Equal(t, http.StatusNoContent, result.status)
	require.True(t, result.handlerCalled)
	require.Equal(t, 1, result.selectedID)
}
