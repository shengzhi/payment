// 企业打款给个人

package wechat

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/shengzhi/payment"
)

// TransferRequest 打款请求
type TransferRequest struct {
	XMLName    xml.Name `xml:"xml" json:"-"`
	APPID      string   `xml:"mch_appid" sign:"mch_appid"`
	MerchantID string   `xml:"mchid" sign:"mchid"`
	Noncestr   string   `xml:"nonce_str" sign:"nonce_str"`
	Sign       string   `xml:"sign"`
	SignType   string   `xml:"sign_type" sign:"sign_type"`
	OrderNo    string   `xml:"partner_trade_no" sign:"partner_trade_no"`
	OpenID     string   `xml:"openid" sign:"openid"`         // 接受红包的用户 用户在wxappid下的openid
	CheckName  string   `xml:"check_name" sign:"check_name"` // NO_CHECK：不校验真实姓名 FORCE_CHECK：强校验真实姓名
	UserName   string   `xml:"re_user_name,omitempty" sign:"re_user_name"`
	Amount     int32    `xml:"amount" sign:"amount"`                     // 付款金额，单位分
	Desc       string   `xml:"desc" sign:"desc"`                         //
	ClientIP   string   `xml:"spbill_create_ip" sign:"spbill_create_ip"` // 调用接口的机器Ip地址
	DeviceInfo string   `xml:"device_info,omitempty" sign:"device_info"`
}

func (r *TransferRequest) setSign(sign string) { r.Sign = sign }

// TransferReply 打款响应
type TransferReply struct {
	XMLName    xml.Name `xml:"xml"`
	ReturnCode string   `xml:"return_code" sign:"return_code"`
	ReturnMsg  string   `xml:"return_msg" sign:"return_msg"`
	AppID      string   `xml:"mch_appid" sign:"mch_appid"`
	MerchantID string   `xml:"mchid" sign:"mchid"`
	NonceStr   string   `xml:"nonce_str" sign:"nonce_str"`
	Sign       string   `xml:"sign"`
	ResultCode string   `xml:"result_code" sign:"result_code"`
	ErrCode    string   `xml:"err_code" sign:"err_code"`
	ErrDesc    string   `xml:"err_code_des" sign:"err_code_des"`
	OrderNo    string   `xml:"partner_trade_no" sign:"partner_trade_no"`
	WXOrderNo  string   `xml:"payment_no" sign:"payment_no"`
	DeviceInfo string   `xml:"device_info,omitempty" sign:"device_info"`
	PayTime    string   `xml:"payment_time" sign:"payment_time"`
}

// Transfer 打款
func (c *Client) Transfer(r payment.TransferRequest) (payment.TransferResponse, error) {
	const uri = "https://api.mch.weixin.qq.com/mmpaymkttransfers/promotion/transfers"
	req := TransferRequest{
		APPID: r.WXAppID, OpenID: r.WXOpenID,
		Noncestr: c.genNonceStr(24), SignType: "MD5",
		MerchantID: c.payOption.MerchantID,
		OrderNo:    r.OrderNo,
		Amount:     r.Amount,
		ClientIP:   r.ClientIP,
		UserName:   r.UserName, Desc: r.Desc,
	}
	if r.IsCheckName {
		req.CheckName = "FORCE_CHECK"
	} else {
		req.CheckName = "NO_CHECK"
	}
	c.makePaySign(&req)

	var buf bytes.Buffer
	coder := xml.NewEncoder(&buf)
	coder.Encode(req)

	var result payment.TransferResponse
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: c.tlsCfg},
	}
	res, err := client.Post(uri, "application/xml", &buf)
	if err != nil {
		return result, err
	}
	defer res.Body.Close()
	var reply TransferReply
	err = xml.NewDecoder(res.Body).Decode(&reply)
	if err != nil {
		return result, fmt.Errorf("Payment: decode xml to struct error:%v", err)
	}
	if reply.ReturnCode != "SUCCESS" {
		return result, fmt.Errorf("Payment: %s-%s", reply.ReturnCode, reply.ReturnMsg)
	}
	if reply.ResultCode == "FAIL" {
		return result, fmt.Errorf("Payment:%s-%s", reply.ErrCode, reply.ErrDesc)
	}
	result.OrderNo = reply.OrderNo
	result.PlatOrderNo = reply.WXOrderNo
	result.WXAppID = reply.AppID
	result.PayTime, _ = time.ParseInLocation("2006-01-02 15:04:05", reply.PayTime, time.Local)
	return result, nil
}
