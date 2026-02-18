package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type PaymentSetting struct {
	AmountOptions  []int           `json:"amount_options"`
	// AmountDiscount: legacy "pay discount multiplier" (e.g. 0.95 means pay 95% of original).
	// NOTE: Newer versions may use amount_bonus for "extra quota bonus". Keep this field for compatibility.
	AmountDiscount map[int]float64 `json:"amount_discount"`
	// AmountBonus: "extra quota bonus rate" (e.g. 0.05 means +5% extra quota).
	AmountBonus map[int]float64 `json:"amount_bonus"`
}

// 默认配置
var paymentSetting = PaymentSetting{
	AmountOptions:  []int{10, 20, 50, 100, 200, 500},
	AmountDiscount: map[int]float64{},
	AmountBonus:    map[int]float64{},
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("payment_setting", &paymentSetting)
}

func GetPaymentSetting() *PaymentSetting {
	return &paymentSetting
}

// GetTopupDiscountMultiplier returns a multiplier applied to the pay money.
// Only values in (0.5, 1.0] are treated as discounts to avoid accidentally
// interpreting bonus-like configs (e.g. 0.05) as a 95% off discount.
func GetTopupDiscountMultiplier(amount int64) float64 {
	m := paymentSetting.AmountDiscount
	if len(m) == 0 {
		return 1.0
	}
	amt := int(amount)

	// Tier matching: take the highest key <= amount.
	bestKey := -1
	bestVal := 1.0
	for k, v := range m {
		if k <= amt && k > bestKey {
			bestKey = k
			bestVal = v
		}
	}

	if bestKey < 0 {
		return 1.0
	}
	// Treat (0.5, 1] as a discount multiplier; otherwise ignore.
	if bestVal > 0.5 && bestVal <= 1.0 {
		return bestVal
	}
	return 1.0
}

// GetTopupBonusRate returns the extra quota bonus rate for the given amount.
// Prefer payment_setting.amount_bonus. If empty, fall back to legacy amount_discount
// entries that look like bonus rates (0, 0.5].
func GetTopupBonusRate(amount int64) float64 {
	amt := int(amount)

	// Prefer explicit bonus map.
	if len(paymentSetting.AmountBonus) > 0 {
		bestKey := -1
		bestVal := 0.0
		for k, v := range paymentSetting.AmountBonus {
			if k <= amt && k > bestKey {
				bestKey = k
				bestVal = v
			}
		}
		if bestKey >= 0 && bestVal > 0 {
			return bestVal
		}
	}

	// Legacy fallback: treat amount_discount values <= 0.5 as bonus rates.
	if len(paymentSetting.AmountDiscount) > 0 {
		bestKey := -1
		bestVal := 0.0
		for k, v := range paymentSetting.AmountDiscount {
			if k <= amt && k > bestKey && v > 0 && v <= 0.5 {
				bestKey = k
				bestVal = v
			}
		}
		if bestKey >= 0 && bestVal > 0 {
			return bestVal
		}
	}

	return 0.0
}

// GetTopupBonusMapForAPI returns the bonus map to be exposed to the frontend.
// If amount_bonus is configured, return it; otherwise derive from amount_discount
// entries that look like bonus rates (0, 0.5].
func GetTopupBonusMapForAPI() map[int]float64 {
	if len(paymentSetting.AmountBonus) > 0 {
		return paymentSetting.AmountBonus
	}
	derived := map[int]float64{}
	for k, v := range paymentSetting.AmountDiscount {
		if v > 0 && v <= 0.5 {
			derived[k] = v
		}
	}
	return derived
}
