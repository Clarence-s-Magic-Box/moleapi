package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// GetDBTimestamp returns a UNIX timestamp from database time.
// Falls back to application time on error.
func GetDBTimestamp() int64 {
	return getDBTimestampWithHandle(nil)
}

func getDBTimestampWithHandle(handle *gorm.DB) int64 {
	var ts int64
	var err error
	query := DB
	if handle != nil {
		query = handle
	}
	switch {
	case common.UsingPostgreSQL:
		err = query.Raw("SELECT EXTRACT(EPOCH FROM NOW())::bigint").Scan(&ts).Error
	case common.UsingSQLite:
		err = query.Raw("SELECT strftime('%s','now')").Scan(&ts).Error
	default:
		err = query.Raw("SELECT UNIX_TIMESTAMP()").Scan(&ts).Error
	}
	if err != nil || ts <= 0 {
		return common.GetTimestamp()
	}
	return ts
}
