package common

func CalculateExtraQuotaRatio(amount float64) float64 {
	if amount >= 280 {
		return 0.40 // 加赠40%
	} else if amount >= 140 {
		return 0.34 // 加赠34%
	} else if amount >= 70 {
		return 0.26 // 加赠26%
	} else if amount >= 35 {
		return 0.22 // 加赠22%
	} else if amount >= 15 {
		return 0.18 // 加赠18%
	} else if amount >= 7 {
		return 0.12 // 加赠12%
	} else if amount >= 3 {
		return 0.08 // 加赠8%
	} else if amount >= 1 {
		return 0.05 // 加赠5%
	}
	return 0
}
