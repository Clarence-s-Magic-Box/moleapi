package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func insertTopUpTestUser(t *testing.T, username string, quota int, email string) *User {
	t.Helper()

	user := &User{
		Username:    username,
		Password:    "password123",
		DisplayName: username,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Email:       email,
		Group:       "default",
		Quota:       quota,
		Setting:     "{}",
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func insertTopUpRecord(t *testing.T, userID int, tradeNo string, paymentMethod string) *TopUp {
	t.Helper()

	topUp := &TopUp{
		UserId:        userID,
		Amount:        10,
		Money:         12.34,
		TradeNo:       tradeNo,
		PaymentMethod: paymentMethod,
		CreateTime:    1,
		Status:        common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(topUp).Error)
	return topUp
}

func reloadUser(t *testing.T, userID int) *User {
	t.Helper()

	var user User
	require.NoError(t, DB.First(&user, userID).Error)
	return &user
}

func reloadTopUp(t *testing.T, topUpID int) *TopUp {
	t.Helper()

	var topUp TopUp
	require.NoError(t, DB.First(&topUp, topUpID).Error)
	return &topUp
}

func TestRechargeRejectsMismatchedPaymentMethod(t *testing.T) {
	truncateTables(t)

	user := insertTopUpTestUser(t, strings.ReplaceAll(t.Name(), "/", "_"), 321, "")
	topUp := insertTopUpRecord(t, user.Id, "trade_recharge_mismatch", "creem")

	err := Recharge(topUp.TradeNo, "cus_test")
	require.Error(t, err)

	reloadedUser := reloadUser(t, user.Id)
	require.Equal(t, 321, reloadedUser.Quota)
	require.Empty(t, reloadedUser.StripeCustomer)

	reloadedTopUp := reloadTopUp(t, topUp.Id)
	require.Equal(t, common.TopUpStatusPending, reloadedTopUp.Status)
	require.EqualValues(t, 0, reloadedTopUp.CompleteTime)
}

func TestRechargeCreemRejectsMismatchedPaymentMethod(t *testing.T) {
	truncateTables(t)

	user := insertTopUpTestUser(t, strings.ReplaceAll(t.Name(), "/", "_"), 654, "original@example.com")
	topUp := insertTopUpRecord(t, user.Id, "trade_creem_mismatch", "stripe")

	err := RechargeCreem(topUp.TradeNo, "new@example.com", "New Name")
	require.Error(t, err)

	reloadedUser := reloadUser(t, user.Id)
	require.Equal(t, 654, reloadedUser.Quota)
	require.Equal(t, "original@example.com", reloadedUser.Email)

	reloadedTopUp := reloadTopUp(t, topUp.Id)
	require.Equal(t, common.TopUpStatusPending, reloadedTopUp.Status)
	require.EqualValues(t, 0, reloadedTopUp.CompleteTime)
}

func TestRechargeWaffoRejectsMismatchedPaymentMethod(t *testing.T) {
	truncateTables(t)

	user := insertTopUpTestUser(t, strings.ReplaceAll(t.Name(), "/", "_"), 987, "")
	topUp := insertTopUpRecord(t, user.Id, "trade_waffo_mismatch", "stripe")

	err := RechargeWaffo(topUp.TradeNo)
	require.Error(t, err)

	reloadedUser := reloadUser(t, user.Id)
	require.Equal(t, 987, reloadedUser.Quota)

	reloadedTopUp := reloadTopUp(t, topUp.Id)
	require.Equal(t, common.TopUpStatusPending, reloadedTopUp.Status)
	require.EqualValues(t, 0, reloadedTopUp.CompleteTime)
}
