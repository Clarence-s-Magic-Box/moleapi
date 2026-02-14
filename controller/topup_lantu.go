package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

const (
	lanTuJumpH5Path    = "/api/wxpay/jump_h5"
	lanTuGetOrderPath  = "/api/wxpay/get_pay_order"
	lanTuFailCode      = 1 // 查询订单失败状态码（legacy 约定）
	lanTuUnpaidCode    = 0 // 未支付状态码（legacy 约定）
	lanTuOrderTTL      = 300
	lanTuDefaultBody   = "充值"
	lanTuPaymentMethod = "lantu"
)

type LanTuConfig struct {
	MchId     string
	SecretKey string
	ApiBase   string
}

func GetLanTuPayConfig() *LanTuConfig {
	if common.LantuApiUrl == "" || common.LantuMchId == "" || common.LantuSecretKey == "" {
		return nil
	}
	return &LanTuConfig{
		MchId:     common.LantuMchId,
		SecretKey: common.LantuSecretKey,
		ApiBase:   common.LantuApiUrl,
	}
}

// RequestLanTuPay creates a LanTu WeChat H5 payment and returns the pay link.
// It is designed to be consumed by the current frontend flow (open link in new tab).
func RequestLanTuPay(c *gin.Context) {
	var req EpayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if req.Amount < getMinTopup() {
		c.JSON(200, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", getMinTopup())})
		return
	}

	cfg := GetLanTuPayConfig()
	if cfg == nil {
		c.JSON(200, gin.H{"message": "error", "data": "当前管理员未配置蓝兔支付信息"})
		return
	}

	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "获取用户分组失败"})
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	if payMoney < 0.01 {
		c.JSON(200, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}

	// trade no aligns with Epay formatting, so logs/search feel consistent.
	tradeNo := fmt.Sprintf("%s%d", common.GetRandomString(6), time.Now().Unix())
	tradeNo = fmt.Sprintf("USR%dNO%s", id, tradeNo)

	// notify url should be reachable by LanTu platform
	callBackAddress := service.GetCallbackAddress()
	notifyUrl := common.BuildURL(callBackAddress, "/api/user/lantu/notify")

	referer := c.GetHeader("Referer")
	if referer == "" {
		referer = system_setting.ServerAddress
	}

	body := fmt.Sprintf("%s%s %s", common.SystemName, lanTuDefaultBody, strconv.FormatFloat(float64(req.Amount), 'f', 0, 64))

	// Sign only required fields (legacy behavior).
	paramsToSign := map[string]string{
		"mch_id":       cfg.MchId,
		"out_trade_no": tradeNo,
		"total_fee":    strconv.FormatFloat(payMoney, 'f', 2, 64),
		"body":         body,
		"timestamp":    strconv.FormatInt(time.Now().Unix(), 10),
		"notify_url":   notifyUrl,
	}
	sign := common.GenerateSignature(paramsToSign, cfg.SecretKey)

	params := map[string]string{
		"mch_id":       paramsToSign["mch_id"],
		"out_trade_no": paramsToSign["out_trade_no"],
		"total_fee":    paramsToSign["total_fee"],
		"body":         paramsToSign["body"],
		"timestamp":    paramsToSign["timestamp"],
		"notify_url":   paramsToSign["notify_url"],
		"quit_url":     referer,
		"return_url":   referer,
		"sign":         sign,
	}

	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount := decimal.NewFromInt(int64(amount))
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		amount = dAmount.Div(dQuotaPerUnit).IntPart()
	}
	topUp := &model.TopUp{
		UserId:        id,
		Amount:        amount,
		Money:         payMoney,
		TradeNo:       tradeNo,
		PaymentMethod: lanTuPaymentMethod,
		CreateTime:    time.Now().Unix(),
		Status:        common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	payLink, err := lanTuDoPay(params, common.BuildURL(cfg.ApiBase, lanTuJumpH5Path))
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "success",
		"data": gin.H{
			"pay_link": payLink,
			"trade_no": tradeNo,
		},
	})
}

func LanTuPayNotify(c *gin.Context) {
	// Accept GET for simple health check
	outTradeNo := c.PostForm("out_trade_no")
	mchId := c.PostForm("mch_id")
	if outTradeNo == "" && mchId == "" && c.Request.Method == "GET" {
		c.String(http.StatusOK, "LANTU_CALLBACK_OK")
		return
	}

	if outTradeNo == "" {
		_ = c.Request.ParseForm()
		outTradeNo = c.PostForm("out_trade_no")
		mchId = c.PostForm("mch_id")
	}

	cfg := GetLanTuPayConfig()
	if cfg == nil {
		c.String(http.StatusOK, "FAIL")
		return
	}
	// Some callbacks may omit mch_id; fall back to configured value.
	if mchId == "" {
		mchId = cfg.MchId
	}
	if outTradeNo == "" {
		c.String(http.StatusOK, "FAIL")
		return
	}

	// Prevent double-credit on repeated callbacks.
	LockOrder(outTradeNo)
	defer UnlockOrder(outTradeNo)

	topUp := model.GetTopUpByTradeNo(outTradeNo)
	if topUp == nil {
		c.String(http.StatusOK, "FAIL")
		return
	}

	// Already processed.
	if topUp.Status != common.TopUpStatusPending {
		c.String(http.StatusOK, "SUCCESS")
		return
	}

	// Expire old orders.
	if time.Now().Unix()-topUp.CreateTime > lanTuOrderTTL {
		topUp.Status = common.TopUpStatusExpired
		_ = topUp.Update()
		c.String(http.StatusOK, "SUCCESS")
		return
	}

	paid, err := lanTuCheckPaid(cfg, mchId, outTradeNo)
	if err != nil {
		c.String(http.StatusOK, "FAIL")
		return
	}
	if !paid {
		c.String(http.StatusOK, "FAIL")
		return
	}

	topUp.Status = common.TopUpStatusSuccess
	topUp.CompleteTime = common.GetTimestamp()
	if err := topUp.Update(); err != nil {
		c.String(http.StatusOK, "FAIL")
		return
	}

	dAmount := decimal.NewFromInt(int64(topUp.Amount))
	dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
	quotaToAdd := int(dAmount.Mul(dQuotaPerUnit).IntPart())
	if err := model.IncreaseUserQuota(topUp.UserId, quotaToAdd, true); err != nil {
		c.String(http.StatusOK, "FAIL")
		return
	}

	model.RecordLog(topUp.UserId, model.LogTypeTopup, fmt.Sprintf("使用蓝兔支付充值成功，充值金额: %v，支付金额：%f", logger.LogQuota(quotaToAdd), topUp.Money))
	c.String(http.StatusOK, "SUCCESS")
}

func lanTuDoPay(params map[string]string, endpoint string) (string, error) {
	formData := make([]string, 0, len(params))
	for k, v := range params {
		formData = append(formData, k+"="+url.QueryEscape(v))
	}
	payload := strings.NewReader(strings.Join(formData, "&"))

	req, _ := http.NewRequest("POST", endpoint, payload)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.New("调起支付失败")
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", errors.New("支付响应解析失败")
	}

	// legacy behavior: code == 1 means error
	code := fmt.Sprintf("%v", result["code"])
	if code == "1" {
		msg := fmt.Sprintf("%v", result["msg"])
		if msg == "" {
			msg = "创建支付失败"
		}
		return "", errors.New(msg)
	}

	dataVal, ok := result["data"]
	if !ok || dataVal == nil {
		return "", errors.New("支付响应缺少 data")
	}
	link := fmt.Sprintf("%v", dataVal)
	if link == "" {
		return "", errors.New("支付链接为空")
	}
	return link, nil
}

func lanTuCheckPaid(cfg *LanTuConfig, mchId, outTradeNo string) (bool, error) {
	params := map[string]string{
		"mch_id":       mchId,
		"out_trade_no": outTradeNo,
		"timestamp":    strconv.FormatInt(time.Now().Unix(), 10),
	}
	params["sign"] = common.GenerateSignature(params, cfg.SecretKey)

	formData := fmt.Sprintf("mch_id=%s&out_trade_no=%s&timestamp=%s&sign=%s",
		url.QueryEscape(params["mch_id"]),
		url.QueryEscape(params["out_trade_no"]),
		url.QueryEscape(params["timestamp"]),
		url.QueryEscape(params["sign"]),
	)
	payload := strings.NewReader(formData)

	endpointUrl := common.BuildURL(cfg.ApiBase, lanTuGetOrderPath)
	req, _ := http.NewRequest("POST", endpointUrl, payload)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}

	if result["code"] == nil {
		return false, errors.New("订单查询返回数据异常")
	}
	if code, ok := result["code"].(float64); ok && int(code) == lanTuFailCode {
		return false, errors.New("订单查询失败")
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok || data == nil {
		return false, errors.New("订单查询响应格式错误")
	}

	payStatusFloat, ok := data["pay_status"].(float64)
	if !ok {
		return false, errors.New("支付状态字段格式错误")
	}
	payStatus := int(payStatusFloat)
	return payStatus != lanTuUnpaidCode, nil
}
