package common

import (
	"crypto/md5"
	"encoding/hex"
	"sort"
	"strings"
)

// GenerateSignature generates signature for LanTu payment.
// It matches the implementation used in the legacy MoleAPI code:
// 1) sort keys by ASCII
// 2) build k=v&k2=v2...
// 3) append &key=SECRET
// 4) MD5 + uppercase hex
func GenerateSignature(params map[string]string, key string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for i, k := range keys {
		if i > 0 {
			builder.WriteString("&")
		}
		builder.WriteString(k)
		builder.WriteString("=")
		builder.WriteString(params[k])
	}
	builder.WriteString("&key=")
	builder.WriteString(key)

	sum := md5.Sum([]byte(builder.String()))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}
