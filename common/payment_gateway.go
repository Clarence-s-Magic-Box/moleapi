package common

import "strings"

const (
	PaymentGatewayEpay   = "epay"
	PaymentGatewayStripe = "stripe"
	PaymentGatewayCreem  = "creem"
	PaymentGatewayWaffo  = "waffo"
	PaymentGatewayLanTu  = "lantu"
)

// NormalizePaymentGateway maps stored payment methods to the callback gateway that is
// allowed to complete them. Dedicated gateways keep their own name, while legacy/custom
// Epay methods (alipay, wxpay, custom1, etc.) are treated as Epay.
func NormalizePaymentGateway(paymentMethod string) string {
	switch strings.ToLower(strings.TrimSpace(paymentMethod)) {
	case PaymentGatewayStripe:
		return PaymentGatewayStripe
	case PaymentGatewayCreem:
		return PaymentGatewayCreem
	case PaymentGatewayWaffo:
		return PaymentGatewayWaffo
	case PaymentGatewayLanTu:
		return PaymentGatewayLanTu
	case "":
		return ""
	default:
		return PaymentGatewayEpay
	}
}

func PaymentGatewayMatches(paymentMethod string, expectedGateway string) bool {
	return NormalizePaymentGateway(paymentMethod) == strings.ToLower(strings.TrimSpace(expectedGateway))
}
