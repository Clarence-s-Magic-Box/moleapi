package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v81"
	"gorm.io/gorm"
)

func setupPaymentWebhookTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := "file:" + url.QueryEscape(t.Name()) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.TopUp{}))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func insertWebhookTopUp(t *testing.T, db *gorm.DB, tradeNo string, paymentMethod string, status string) *model.TopUp {
	t.Helper()

	paymentProvider := model.PaymentProviderEpay
	switch paymentMethod {
	case model.PaymentMethodStripe:
		paymentProvider = model.PaymentProviderStripe
	case model.PaymentMethodCreem:
		paymentProvider = model.PaymentProviderCreem
	case model.PaymentMethodWaffo:
		paymentProvider = model.PaymentProviderWaffo
	case model.PaymentMethodWaffoPancake:
		paymentProvider = model.PaymentProviderWaffoPancake
	case "lantu":
		paymentProvider = model.PaymentProviderLanTu
	}

	topUp := &model.TopUp{
		UserId:          1,
		Amount:          10,
		Money:           1.23,
		TradeNo:         tradeNo,
		PaymentMethod:   paymentMethod,
		PaymentProvider: paymentProvider,
		CreateTime:      1,
		Status:          status,
	}
	require.NoError(t, db.Create(topUp).Error)
	return topUp
}

func loadWebhookTopUp(t *testing.T, db *gorm.DB, topUpID int) *model.TopUp {
	t.Helper()

	var topUp model.TopUp
	require.NoError(t, db.First(&topUp, topUpID).Error)
	return &topUp
}

func TestSessionAsyncPaymentFailedMarksStripeOrderFailed(t *testing.T) {
	db := setupPaymentWebhookTestDB(t)
	topUp := insertWebhookTopUp(t, db, "stripe_async_failed", model.PaymentMethodStripe, common.TopUpStatusPending)

	event := stripe.Event{
		Data: &stripe.EventData{
			Object: map[string]interface{}{
				"client_reference_id": topUp.TradeNo,
			},
		},
	}

	sessionAsyncPaymentFailed(context.Background(), event, "127.0.0.1")

	reloadedTopUp := loadWebhookTopUp(t, db, topUp.Id)
	require.Equal(t, common.TopUpStatusFailed, reloadedTopUp.Status)
}

func TestSessionAsyncPaymentFailedIgnoresNonStripeOrder(t *testing.T) {
	db := setupPaymentWebhookTestDB(t)
	topUp := insertWebhookTopUp(t, db, "creem_async_event", model.PaymentMethodCreem, common.TopUpStatusPending)

	event := stripe.Event{
		Data: &stripe.EventData{
			Object: map[string]interface{}{
				"client_reference_id": topUp.TradeNo,
			},
		},
	}

	sessionAsyncPaymentFailed(context.Background(), event, "127.0.0.1")

	reloadedTopUp := loadWebhookTopUp(t, db, topUp.Id)
	require.Equal(t, common.TopUpStatusPending, reloadedTopUp.Status)
}

func TestEpayNotifyIgnoresNonEpayOrder(t *testing.T) {
	db := setupPaymentWebhookTestDB(t)
	topUp := insertWebhookTopUp(t, db, "epay_to_stripe_order", model.PaymentMethodStripe, common.TopUpStatusPending)

	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	t.Cleanup(func() {
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
	})

	operation_setting.PayAddress = "https://payment.example.com"
	operation_setting.EpayId = "partner-test"
	operation_setting.EpayKey = "secret-test"

	params := epay.GenerateParams(map[string]string{
		"pid":          operation_setting.EpayId,
		"type":         "alipay",
		"trade_no":     "epay-123",
		"out_trade_no": topUp.TradeNo,
		"name":         "TUC10",
		"money":        "1.23",
		"trade_status": epay.StatusTradeSuccess,
	}, operation_setting.EpayKey)

	query := url.Values{}
	for key, value := range params {
		query.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/epay/notify?"+query.Encode(), nil)

	EpayNotify(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "success", recorder.Body.String())

	reloadedTopUp := loadWebhookTopUp(t, db, topUp.Id)
	require.Equal(t, common.TopUpStatusPending, reloadedTopUp.Status)
}

func TestEpayNotifyIgnoresLanTuOrder(t *testing.T) {
	db := setupPaymentWebhookTestDB(t)
	topUp := insertWebhookTopUp(t, db, "epay_to_lantu_order", "lantu", common.TopUpStatusPending)

	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	t.Cleanup(func() {
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
	})

	operation_setting.PayAddress = "https://payment.example.com"
	operation_setting.EpayId = "partner-test"
	operation_setting.EpayKey = "secret-test"

	params := epay.GenerateParams(map[string]string{
		"pid":          operation_setting.EpayId,
		"type":         "alipay",
		"trade_no":     "epay-456",
		"out_trade_no": topUp.TradeNo,
		"name":         "TUC10",
		"money":        "1.23",
		"trade_status": epay.StatusTradeSuccess,
	}, operation_setting.EpayKey)

	query := url.Values{}
	for key, value := range params {
		query.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/epay/notify?"+query.Encode(), nil)

	EpayNotify(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "success", recorder.Body.String())

	reloadedTopUp := loadWebhookTopUp(t, db, topUp.Id)
	require.Equal(t, common.TopUpStatusPending, reloadedTopUp.Status)
}

func TestLanTuNotifyIgnoresNonLanTuOrder(t *testing.T) {
	db := setupPaymentWebhookTestDB(t)
	topUp := insertWebhookTopUp(t, db, "lantu_to_epay_order", "alipay", common.TopUpStatusPending)

	originalMchID := common.LantuMchId
	originalSecretKey := common.LantuSecretKey
	t.Cleanup(func() {
		common.LantuMchId = originalMchID
		common.LantuSecretKey = originalSecretKey
	})

	common.LantuMchId = "mch_test"
	common.LantuSecretKey = "secret_test"

	form := url.Values{}
	form.Set("out_trade_no", topUp.TradeNo)
	form.Set("mch_id", common.LantuMchId)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/user/lantu/notify", strings.NewReader(form.Encode()))
	ctx.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	LanTuPayNotify(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "SUCCESS", recorder.Body.String())

	reloadedTopUp := loadWebhookTopUp(t, db, topUp.Id)
	require.Equal(t, common.TopUpStatusPending, reloadedTopUp.Status)
}
