package relay

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestDetailTokensExceedTotalDetectsSplitUsageTotals(t *testing.T) {
	require.True(t, detailTokensExceedTotal(
		decimal.NewFromInt(5),
		decimal.NewFromInt(5),
		decimal.NewFromInt(1120),
	))
}

func TestDetailTokensExceedTotalAllowsInclusiveUsageTotals(t *testing.T) {
	require.False(t, detailTokensExceedTotal(
		decimal.NewFromInt(1125),
		decimal.NewFromInt(5),
		decimal.NewFromInt(1120),
	))
}

func TestDetailTokensExceedTotalIgnoresEmptyDetails(t *testing.T) {
	require.False(t, detailTokensExceedTotal(
		decimal.NewFromInt(42),
		decimal.Zero,
		decimal.Zero,
	))
}
