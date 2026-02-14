package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"one-api/common"
	"one-api/model"
	"one-api/setting"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	LanTuWxPayUrl    = "https://api.ltzf.cn/api/wxpay/native"
	LanTuGetWxOrder  = "https://api.ltzf.cn/api/wxpay/get_pay_order"
	LanTuFailCode    = 1 // 查询订单失败状态码
	LanTuPayFailCode = 0 // 未支付状态码
)

type LanTuConfig struct {
	MchId     string
	SecretKey string
	ApiUrl    string
}

func GetLanTuPayConfig() *LanTuConfig {
	if common.LantuApiUrl == "" || common.LantuMchId == "" || common.LantuSecretKey == "" {
		return nil
	}
	lanTuConfig := &LanTuConfig{
		MchId:     common.LantuMchId,
		SecretKey: common.LantuSecretKey,
		ApiUrl:    common.LantuApiUrl,
	}
	return lanTuConfig
}

func RequestLanTuPay(c *gin.Context) {
	var req EpayRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "data": 10})
		return
	}
	if req.Amount < 1 {
		c.JSON(200, gin.H{"message": "充值金额不能小于1", "data": 10})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(200, gin.H{"message": "获取用户分组失败"})
		return
	}
	amount := getPayMoney(req.Amount, group)
	extraRatio := common.CalculateExtraQuotaRatio(float64(req.Amount))
	extraQuota := int(math.Ceil(float64(req.Amount) * extraRatio * common.QuotaPerUnit))

	log.Printf("充值计算 - 用户ID: %d, 原始金额: %v, 支付金额: %f, 加赠比例: %f, 额外赠送: %v",
		id, req.Amount, amount, extraRatio, extraQuota)

	config := GetLanTuPayConfig()
	if config == nil {
		c.JSON(200, gin.H{"message": "error", "data": "当前管理员未配置支付信息"})
		return
	}
	config.ApiUrl = LanTuWxPayUrl // 微信支付
	notifyUrl := setting.ServerAddress + "/api/user/lantu/notify"
	tradeNo := GenerateTradeNo(id)
	payMoney := amount
	body := fmt.Sprintf("Mole API 充值 $%.2f", float64(req.Amount))
	// 生成支付链接和参数
	params := map[string]string{
		"mch_id":       config.MchId,
		"out_trade_no": tradeNo,
		"total_fee":    strconv.FormatFloat(payMoney, 'f', 2, 64),
		"body":         body,
		"timestamp":    strconv.FormatInt(time.Now().Unix(), 10),
		"notify_url":   notifyUrl,
	}
	lanTuSign := common.GenerateSignature(params, config.SecretKey)
	params["sign"] = lanTuSign

	topUp := &model.TopUp{
		UserId:     id,
		Amount:     int64(req.Amount),
		ExtraQuota: extraQuota,
		Money:      payMoney,
		TradeNo:    tradeNo,
		CreateTime: time.Now().Unix(),
		Status:     common.TopUpStatusPending,
	}
	err = topUp.Insert()
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}
	response, err := LanTuDoPay(params, config.ApiUrl)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "创建支付失败"})
		return
	}
	if fmt.Sprintf("%v", response["code"]) == "1" {
		c.JSON(200, gin.H{"message": "error", "data": response["msg"]})
		return
	}
	result := response["data"].(map[string]interface{})
	result["trade_no"] = tradeNo
	c.JSON(200, gin.H{"message": "success", "data": result, "url": ""})
}

func LanTuPayNotify(c *gin.Context) {
	// 解析表单数据并记录所有请求参数进行调试
	c.Request.ParseForm()
	allParams := make(map[string]string)
	for key, values := range c.Request.Form {
		if len(values) > 0 {
			allParams[key] = values[0]
		}
	}
	log.Printf("蓝兔支付回调收到所有参数: %+v", allParams)

	outTradeNo := c.PostForm("out_trade_no")
	mchId := c.PostForm("mch_id")

	// 添加详细的调试日志
	log.Printf("蓝兔支付回调收到请求 - 订单号: %s, 商户ID: %s", outTradeNo, mchId)

	// 如果订单号为空，可能是GET请求的测试
	if outTradeNo == "" && mchId == "" {
		log.Printf("蓝兔支付回调接口测试访问")
		c.String(http.StatusOK, "LANTU_CALLBACK_OK")
		return
	}

	config := GetLanTuPayConfig()
	if config == nil {
		log.Println("蓝兔支付回调失败 未找到配置信息")
		c.String(http.StatusOK, "FAIL")
		return
	}

	// 使用数据库事务
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		// 获取到监听的订单，并加锁
		topUp := &model.TopUp{}
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("trade_no = ?", outTradeNo).First(topUp).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("订单号不存在")
			}
			return err
		}

		// 检查订单状态，如果已经处理过，直接返回成功
		if topUp.Status != "pending" {
			return nil
		}

		// 检查订单是否超时
		if time.Now().Unix()-topUp.CreateTime > 300 { // 5分钟 = 300秒
			topUp.Status = "expired"
			if err := tx.Save(topUp).Error; err != nil {
				log.Printf("更新过期订单状态失败: %v", err)
				return err
			}
			return nil
		}

		// 构造查询订单请求体
		respBody := map[string]string{
			"mch_id":       mchId,
			"out_trade_no": outTradeNo,
			"timestamp":    strconv.FormatInt(time.Now().Unix(), 10),
		}
		// 生成签名
		sign := common.GenerateSignature(respBody, config.SecretKey)
		respBody["sign"] = sign

		// 查询订单状态
		orderStatus, err := GetPayOrder(respBody)
		if err != nil {
			log.Printf("蓝兔支付查询订单失败: %v", err)
			return fmt.Errorf("订单查询出错: %v", err)
		}

		// 调试：记录完整的API响应
		log.Printf("蓝兔支付API响应 - 订单号: %s, 完整响应: %+v", outTradeNo, orderStatus)

		if orderStatus["code"] == nil {
			log.Printf("蓝兔支付返回数据异常: %+v", orderStatus)
			return fmt.Errorf("订单查询返回数据异常")
		}
		if orderStatus["code"] == LanTuFailCode {
			log.Printf("蓝兔支付查询失败，错误码: %v", orderStatus["code"])
			return fmt.Errorf("订单查询失败")
		}

		// 安全获取支付状态
		var payStatus int
		if data, ok := orderStatus["data"].(map[string]interface{}); ok && data != nil {
			if status, ok := data["pay_status"].(float64); ok {
				payStatus = int(status)
			} else {
				log.Printf("蓝兔支付状态检查错误 - 订单号: %s, pay_status字段类型错误: %+v", outTradeNo, data["pay_status"])
				return fmt.Errorf("支付状态字段格式错误")
			}
		} else {
			log.Printf("蓝兔支付状态检查错误 - 订单号: %s, data字段为空或格式错误: %+v", outTradeNo, orderStatus["data"])
			return fmt.Errorf("支付状态响应格式错误")
		}
		log.Printf("蓝兔支付状态检查 - 订单号: %s, 支付状态: %d", outTradeNo, payStatus)

		// 检查订单是否已经处理过（防止重复回调）
		if topUp.Status == common.TopUpStatusSuccess {
			log.Printf("蓝兔支付订单已处理过 - 订单号: %s, 当前状态: %s", outTradeNo, topUp.Status)
			return nil // 订单已处理成功，直接返回
		}

		// 检查订单是否支付成功
		if payStatus != LanTuPayFailCode {
			// 计算总额度（基础额度 + 加赠额度）
			totalQuota := int(topUp.Amount)*int(common.QuotaPerUnit) + topUp.ExtraQuota

			// 更新用户额度
			if err := model.IncreaseUserQuota(topUp.UserId, totalQuota, true); err != nil {
				return fmt.Errorf("更新用户额度失败: %v", err)
			}

			// 更新订单状态为 success
			if err := topUp.UpdateStatus(tx, common.TopUpStatusSuccess); err != nil {
				return fmt.Errorf("更新订单状态失败: %v", err)
			}

			// 记录充值日志
			logContent := fmt.Sprintf("微信支付充值成功，充值金额: %v，支付金额：%f，赠送：%v",
				common.LogQuota(int(topUp.Amount)*int(common.QuotaPerUnit)),
				topUp.Money,
				common.LogQuota(topUp.ExtraQuota))
			model.RecordLog(topUp.UserId, model.LogTypeTopup, logContent)

			log.Printf("蓝兔支付处理成功 - 用户ID: %d, 订单号: %s, 充值金额: %v, 支付金额: %f, 赠送: %v",
				topUp.UserId, topUp.TradeNo, common.LogQuota(int(topUp.Amount)*int(common.QuotaPerUnit)), topUp.Money, common.LogQuota(topUp.ExtraQuota))
		} else {
			return fmt.Errorf("订单支付失败")
		}

		return nil
	})

	if err != nil {
		log.Printf("蓝兔支付处理错误: %v", err)
		c.String(http.StatusOK, "FAIL")
		return
	}

	c.String(http.StatusOK, "SUCCESS")
}

func LanTuDoPay(params map[string]string, url string) (map[string]interface{}, error) {
	formData := ""
	for k, v := range params {
		formData += k + "=" + v + "&"
	}
	payload := strings.NewReader(formData[:len(formData)-len("&")])

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, errors.New("调起支付失败")
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	result := make(map[string]interface{})
	err = json.Unmarshal(body, &result)
	return result, err
}

// GetPayOrder 查询订单支付状态
func GetPayOrder(respBody map[string]string) (map[string]interface{}, error) {
	params := fmt.Sprintf("mch_id=%v&out_trade_no=%v&timestamp=%v&sign=%v", respBody["mch_id"], respBody["out_trade_no"], respBody["timestamp"], respBody["sign"])
	payload := strings.NewReader(params)
	var url string
	url = LanTuGetWxOrder
	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}
	body, _ := io.ReadAll(res.Body)
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func GetLanTuOrderStatus(c *gin.Context) {
	tradeNo := c.Query("trade_no")
	if tradeNo == "" {
		c.JSON(http.StatusOK, gin.H{
			"message": "error",
			"error":   "trade_no is required",
		})
		return
	}

	var topUp model.TopUp
	if err := model.DB.Where("trade_no = ?", tradeNo).First(&topUp).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "error",
			"error":   "order not found",
		})
		return
	}

	// 检查订单是否已过期（5分钟）
	if topUp.Status == common.TopUpStatusPending && time.Now().Unix()-topUp.CreateTime > 300 {
		topUp.Status = common.TopUpStatusExpired
		if err := model.DB.Save(&topUp).Error; err != nil {
			log.Printf("更新过期订单状态失败: %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"status":  topUp.Status,
	})
}

func RequestLanTuH5Pay(c *gin.Context) {
	var req EpayRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(200, gin.H{"message": err.Error(), "data": 10})
		return
	}
	if req.Amount < 1 {
		c.JSON(200, gin.H{"message": "充值金额不能小于1", "data": 10})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(200, gin.H{"message": "获取用户分组失败"})
		return
	}
	amount := getPayMoney(req.Amount, group)
	extraRatio := common.CalculateExtraQuotaRatio(float64(req.Amount))
	extraQuota := int(math.Ceil(float64(req.Amount) * extraRatio * common.QuotaPerUnit))

	log.Printf("充值计算 - 用户ID: %d, 原始金额: %v, 支付金额: %f, 加赠比例: %f, 额外赠送: %v",
		id, req.Amount, amount, extraRatio, extraQuota)

	config := GetLanTuPayConfig()
	if config == nil {
		c.JSON(200, gin.H{"message": "error", "data": "当前管理员未配置支付信息"})
		return
	}
	config.ApiUrl = "https://api.ltzf.cn/api/wxpay/jump_h5" // H5支付
	notifyUrl := setting.ServerAddress + "/api/user/lantu/notify"
	quitUrl := c.GetHeader("Referer") // 获取发起请求的页面URL
	returnUrl := quitUrl              // 设置返回URL为发起请求的页面
	tradeNo := GenerateTradeNo(id)
	payMoney := amount
	body := fmt.Sprintf("Mole API 充值 $%.2f", float64(req.Amount))
	// 生成支付链接和参数
	paramsToSign := map[string]string{
		"mch_id":       config.MchId,
		"out_trade_no": tradeNo,
		"total_fee":    strconv.FormatFloat(payMoney, 'f', 2, 64),
		"body":         body,
		"timestamp":    strconv.FormatInt(time.Now().Unix(), 10),
		"notify_url":   notifyUrl,
	}
	lanTuSign := common.GenerateSignature(paramsToSign, config.SecretKey)

	// 创建完整的参数map，包括不需要签名的参数
	params := map[string]string{
		"mch_id":       config.MchId,
		"out_trade_no": tradeNo,
		"total_fee":    strconv.FormatFloat(payMoney, 'f', 2, 64),
		"body":         body,
		"timestamp":    strconv.FormatInt(time.Now().Unix(), 10),
		"notify_url":   notifyUrl,
		"quit_url":     quitUrl,
		"return_url":   returnUrl,
		"sign":         lanTuSign,
	}

	topUp := &model.TopUp{
		UserId:     id,
		Amount:     int64(req.Amount),
		ExtraQuota: extraQuota,
		Money:      payMoney,
		TradeNo:    tradeNo,
		CreateTime: time.Now().Unix(),
		Status:     common.TopUpStatusPending,
	}
	err = topUp.Insert()
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}
	response, err := LanTuDoPay(params, config.ApiUrl)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "创建支付失败"})
		return
	}
	if fmt.Sprintf("%v", response["code"]) == "1" {
		c.JSON(200, gin.H{"message": "error", "data": response["msg"]})
		return
	}
	result := response["data"].(string)
	c.JSON(200, gin.H{"message": "success", "data": result, "trade_no": tradeNo})
}
