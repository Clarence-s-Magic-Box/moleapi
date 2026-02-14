package model

import (
	"errors"
	"fmt"
	"log"
	"time"

	"one-api/common"

	"gorm.io/gorm"
)

type TopUp struct {
	Id           int     `json:"id"`
	UserId       int     `json:"user_id" gorm:"index"`
	Amount       int64   `json:"amount"`
	ExtraQuota   int     `json:"extra_quota"` // 微信支付额外赠送额度
	Money        float64 `json:"money"`
	TradeNo      string  `json:"trade_no" gorm:"unique"`
	CreateTime   int64   `json:"create_time"`
	CompleteTime int64   `json:"complete_time"` // stripe支付完成时间
	Status       string  `json:"status"`
}

func (topUp *TopUp) Insert() error {
	var err error
	err = DB.Create(topUp).Error
	return err
}

func (topUp *TopUp) Update() error {
	var err error
	err = DB.Save(topUp).Error
	return err
}

func GetTopUpById(id int) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("id = ?", id).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

func GetTopUpByTradeNo(tradeNo string) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("trade_no = ?", tradeNo).First(&topUp).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("未找到订单号为 %s 的记录", tradeNo)
			return nil
		}
		log.Printf("查询订单号 %s 时发生错误: %v", tradeNo, err)
		return nil
	}
	return topUp
}

// ProcessTopUp 处理微信支付充值（兼容0.7.0版本）
func ProcessTopUp(userId int, amount int, extraQuota int, money float64, tradeNo string) error {
	topUp := &TopUp{
		UserId:     userId,
		Amount:     int64(amount),
		ExtraQuota: extraQuota,
		Money:      money,
		TradeNo:    tradeNo,
		CreateTime: time.Now().Unix(),
		Status:     common.TopUpStatusSuccess,
	}

	totalQuota := amount*int(common.QuotaPerUnit) + extraQuota

	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(topUp).Error; err != nil {
			return fmt.Errorf("创建订单失败: %v", err)
		}

		if err := IncreaseUserQuota(userId, totalQuota, true); err != nil {
			return fmt.Errorf("更新用户额度失败: %v", err)
		}

		logContent := fmt.Sprintf("微信支付充值成功，充值金额: %v，支付金额：%f，赠送：%v",
			common.LogQuota(amount*int(common.QuotaPerUnit)),
			money,
			common.LogQuota(extraQuota))
		RecordLog(userId, LogTypeTopup, logContent)

		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("微信支付充值成功 - 用户ID: %d, 充值金额: %v, 支付金额: %f, 赠送: %v",
		userId, common.LogQuota(amount*int(common.QuotaPerUnit)), money, common.LogQuota(extraQuota))

	return nil
}

// Recharge 处理stripe支付充值（0.8.0版本）
func Recharge(referenceId string, customerId string) (err error) {
	if referenceId == "" {
		return errors.New("未提供支付单号")
	}

	var quota float64
	topUp := &TopUp{}

	refCol := "`trade_no`"
	if common.UsingPostgreSQL {
		refCol = `"trade_no"`
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(refCol+" = ?", referenceId).First(topUp).Error
		if err != nil {
			return errors.New("充值订单不存在")
		}

		if topUp.Status != common.TopUpStatusPending {
			return errors.New("充值订单状态错误")
		}

		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		err = tx.Save(topUp).Error
		if err != nil {
			return err
		}

		quota = topUp.Money * common.QuotaPerUnit
		err = tx.Model(&User{}).Where("id = ?", topUp.UserId).Updates(map[string]interface{}{"stripe_customer": customerId, "quota": gorm.Expr("quota + ?", quota)}).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return errors.New("充值失败，" + err.Error())
	}

	RecordLog(topUp.UserId, LogTypeTopup, fmt.Sprintf("使用Stripe在线充值成功，充值金额: %v，支付金额：%d", common.FormatQuota(int(quota)), int(topUp.Amount)))

	return nil
}

// UpdateStatus 更新订单状态
func (topUp *TopUp) UpdateStatus(tx *gorm.DB, status string) error {
	topUp.Status = status
	return tx.Save(topUp).Error
}
