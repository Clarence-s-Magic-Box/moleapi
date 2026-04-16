package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func insertSubscriptionPlan(t *testing.T, title string) *SubscriptionPlan {
	t.Helper()

	plan := &SubscriptionPlan{
		Title:         title,
		PriceAmount:   19.9,
		Currency:      "USD",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   1000,
	}
	require.NoError(t, DB.Create(plan).Error)
	return plan
}

func insertSubscriptionOrder(t *testing.T, userID int, planID int, tradeNo string, paymentMethod string) *SubscriptionOrder {
	t.Helper()

	order := &SubscriptionOrder{
		UserId:        userID,
		PlanId:        planID,
		Money:         19.9,
		TradeNo:       tradeNo,
		PaymentMethod: paymentMethod,
		CreateTime:    1,
		Status:        common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(order).Error)
	return order
}

func reloadSubscriptionOrder(t *testing.T, orderID int) *SubscriptionOrder {
	t.Helper()

	var order SubscriptionOrder
	require.NoError(t, DB.First(&order, orderID).Error)
	return &order
}

func TestCompleteSubscriptionOrderRejectsMismatchedGateway(t *testing.T) {
	truncateTables(t)

	user := insertTopUpTestUser(t, strings.ReplaceAll(t.Name(), "/", "_"), 100, "")
	plan := insertSubscriptionPlan(t, "Mismatch Plan")
	order := insertSubscriptionOrder(t, user.Id, plan.Id, "sub_gateway_mismatch", common.PaymentGatewayCreem)

	err := CompleteSubscriptionOrder(order.TradeNo, `{"provider":"stripe"}`, common.PaymentGatewayStripe)
	require.ErrorIs(t, err, ErrPaymentMethodMismatch)

	reloadedOrder := reloadSubscriptionOrder(t, order.Id)
	require.Equal(t, common.TopUpStatusPending, reloadedOrder.Status)
	require.EqualValues(t, 0, reloadedOrder.CompleteTime)

	var subCount int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ?", user.Id).Count(&subCount).Error)
	require.Zero(t, subCount)

	var topUpCount int64
	require.NoError(t, DB.Model(&TopUp{}).Where("trade_no = ?", order.TradeNo).Count(&topUpCount).Error)
	require.Zero(t, topUpCount)
}

func TestCompleteSubscriptionOrderAcceptsEpayGatewayForLegacyMethod(t *testing.T) {
	truncateTables(t)

	user := insertTopUpTestUser(t, strings.ReplaceAll(t.Name(), "/", "_"), 100, "")
	plan := insertSubscriptionPlan(t, "Epay Plan")
	order := insertSubscriptionOrder(t, user.Id, plan.Id, "sub_epay_success", "alipay")

	err := CompleteSubscriptionOrder(order.TradeNo, `{"provider":"epay"}`, common.PaymentGatewayEpay)
	require.NoError(t, err)

	reloadedOrder := reloadSubscriptionOrder(t, order.Id)
	require.Equal(t, common.TopUpStatusSuccess, reloadedOrder.Status)
	require.NotZero(t, reloadedOrder.CompleteTime)

	var subCount int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ?", user.Id).Count(&subCount).Error)
	require.EqualValues(t, 1, subCount)

	topUp := GetTopUpByTradeNo(order.TradeNo)
	require.NotNil(t, topUp)
	require.Equal(t, "alipay", topUp.PaymentMethod)
	require.Equal(t, common.TopUpStatusSuccess, topUp.Status)
}
