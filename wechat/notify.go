package wechat

import (
	"encoding/xml"
	"io"
	"time"

	"github.com/shengzhi/payment"
)

// WXNotifyResult 异步通知结果
type WXNotifyResult struct {
	XMLName       xml.Name `xml:"xml"`
	ReturnCode    string   `xml:"return_code" sign:"return_code"`
	ReturnMsg     string   `xml:"return_msg" sign:"return_msg"`
	AppID         string   `xml:"appid" sign:"appid"`
	MerchantID    string   `xml:"mch_id" sign:"mch_id"`
	DeviceInfo    string   `xml:"device_info" sign:"device_info"`
	NonceStr      string   `xml:"nonce_str" sign:"nonce_str"`
	Sign          string   `xml:"sign"`
	ResultCode    string   `xml:"result_code" sign:"result_code"`
	ErrCode       string   `xml:"err_code" sign:"err_code"`
	ErrDesc       string   `xml:"err_code_des" sign:"err_code_des"`
	TradeType     string   `xml:"trade_type" sign:"trade_type"`
	OpenID        string   `xml:"openid" sign:"openid"`
	IsSubscribe   string   `xml:"is_subscribe" sign:"is_subscribe"`
	BankType      string   `xml:"bank_type" sign:"bank_type"`
	TotalAmount   int64    `xml:"total_fee" sign:"total_fee"`
	AccountAmount int64    `xml:"settlement_total_fee" sign:"settlement_total_fee"` //结算金额
	Currency      string   `xml:"fee_type" sign:"fee_type"`
	CashAmount    int64    `xml:"cash_fee" sign:"cash_fee"`
	CashCurrency  string   `xml:"cash_fee_type" sign:"cash_fee_type"`
	CouponAmount  int64    `xml:"coupon_fee" sign:"coupon_fee"`
	CouponNum     int      `xml:"coupon_count" sign:"coupon_count"`
	// CouponType    string   `xml:"coupon_type_$n"`
	TransactionID   string `xml:"transaction_id" sign:"transaction_id"`
	MerchantOrderNo string `xml:"out_trade_no" sign:"out_trade_no"`
	Attach          string `xml:"attach" sign:"attach"`
	CompletedTime   string `xml:"time_end" sign:"time_end"`
}

func (n WXNotifyResult) getSign() string { return n.Sign }

func (n WXNotifyResult) toNotifyResult() *payment.NotifyResult {
	rslt := &payment.NotifyResult{
		MerchantOrderNo: n.MerchantOrderNo,
		Plat:            payment.PayPlatWechat,
		TransactionID:   n.TransactionID,
		TotalAmount:     n.TotalAmount,
		Currency:        n.Currency,
		Attach:          n.Attach,
	}
	rslt.CompletedTime, _ = time.ParseInLocation("20060102150405", n.CompletedTime, time.Local)
	rslt.Wechat.OpenID = n.OpenID
	return rslt
}

type WXNotifyReply struct {
	XMLName xml.Name `xml:"xml"`
	Code    string   `xml:"return_code"`
	Message string   `xml:"return_msg"`
}

// NotifyCallback 异步通知处理
func (c *Client) NotifyCallback(body io.Reader, f payment.NotifyHandleFunc) interface{} {
	var result WXNotifyResult
	d := xml.NewDecoder(body)
	err := d.Decode(&result)
	if err != nil {
		return WXNotifyReply{Code: "FAIL", Message: "序列化失败"}
	}
	if !c.validatePayRes(result) {
		return WXNotifyReply{Code: "FAIL", Message: "签名失败"}
	}
	if err = f(result.toNotifyResult()); err != nil {
		return WXNotifyReply{Code: "FAIL", Message: err.Error()}
	}
	return WXNotifyReply{Code: "SUCCESS", Message: "OK"}
}
