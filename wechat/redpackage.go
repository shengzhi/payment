// 红包

package wechat

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/shengzhi/payment"
)

// SendRedPackRequest 红包发送请求
type SendRedPackRequest struct {
	XMLName      xml.Name `xml:"xml" json:"-"`
	APPID        string   `xml:"wxappid" sign:"wxappid"`
	MerchantID   string   `xml:"mch_id" sign:"mch_id"`
	Noncestr     string   `xml:"nonce_str" sign:"nonce_str"`
	Sign         string   `xml:"sign"`
	SignType     string   `xml:"sign_type" sign:"sign_type"`
	OrderNo      string   `xml:"mch_billno" sign:"mch_billno"`
	SendName     string   `xml:"send_name" sign:"send_name"`                     // 红包发送者名称
	OpenID       string   `xml:"re_openid" sign:"re_openid"`                     // 接受红包的用户 用户在wxappid下的openid
	Amount       int32    `xml:"total_amount" sign:"total_amount"`               // 付款金额，单位分
	Num          int      `xml:"total_num" sign:"total_num"`                     // 红包发放总人数 total_num=1
	Wishing      string   `xml:"wishing" sign:"wishing"`                         // 红包祝福语
	ClientIP     string   `xml:"client_ip" sign:"client_ip"`                     // 调用接口的机器Ip地址
	ActName      string   `xml:"act_name" sign:"act_name"`                       // 活动名称
	Remark       string   `xml:"remark" sign:"remark"`                           // 备注信息
	SceneID      string   `xml:"scene_id,omitempty" sign:"scene_id"`             // 发放红包使用场景，红包金额大于200时必传 PRODUCT_1:商品促销 PRODUCT_2:抽奖 PRODUCT_3:虚拟物品兑奖 PRODUCT_4:企业内部福利 PRODUCT_5:渠道分润 PRODUCT_6:保险回馈 PRODUCT_7:彩票派奖 PRODUCT_8:税务刮奖
	RiskInfo     RiskInfo `xml:"risk_info,omitempty" sign:"risk_info"`           // 活动信息
	ConsumeMchID string   `xml:"consume_mch_id,omitempty" sign:"consume_mch_id"` // 资金授权商户号
}

func (r *SendRedPackRequest) setSign(sign string) { r.Sign = sign }

// RiskInfo 红包活动信息
type RiskInfo struct {
	PostTime      time.Time
	Mobile        string
	DeviceID      string
	ClientVersion string
}

// MarshalXML xml encoding
func (ri RiskInfo) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	var data string
	var buf bytes.Buffer
	if !ri.PostTime.IsZero() {
		fmt.Fprintf(&buf, "posttime=%d&", ri.PostTime.Unix())
	}
	if ri.Mobile != "" {
		fmt.Fprintf(&buf, "mobile=%s&", ri.Mobile)
	}
	if ri.DeviceID != "" {
		fmt.Fprintf(&buf, "deviceid=%s&", ri.DeviceID)
	}
	if ri.ClientVersion != "" {
		fmt.Fprintf(&buf, "clientversion=%s&", ri.ClientVersion)
	}
	if buf.Len() > 0 {
		buf.Truncate(buf.Len() - 1)
		data = url.QueryEscape(buf.String())
	}
	return e.EncodeElement(data, start)
}

// SendRedPackageReply 红包发送响应
type SendRedPackageReply struct {
	XMLName    xml.Name `xml:"xml"`
	ReturnCode string   `xml:"return_code" sign:"return_code"`
	ReturnMsg  string   `xml:"return_msg" sign:"return_msg"`
	AppID      string   `xml:"wxappid" sign:"wxappid"`
	OpenID     string   `xml:"re_openid" sign:"re_openid"`
	MerchantID string   `xml:"mch_id" sign:"mch_id"`
	NonceStr   string   `xml:"nonce_str" sign:"nonce_str"`
	Sign       string   `xml:"sign"`
	ResultCode string   `xml:"result_code" sign:"result_code"`
	ErrCode    string   `xml:"err_code" sign:"err_code"`
	ErrDesc    string   `xml:"err_code_des" sign:"err_code_des"`
	OrderNo    string   `xml:"mch_billno" sign:"mch_billno"`
	Amount     int32    `xml:"total_amount" sign:"total_amount"`
	SendListID string   `xml:"send_listid" sign:"send_listid"`
}

// SendRedPack 发放红包
func (c *Client) SendRedPack(r payment.RedPackageRequest) (payment.RedPackageResponse, error) {
	const uri = "https://api.mch.weixin.qq.com/mmpaymkttransfers/sendredpack"
	req := SendRedPackRequest{
		APPID: r.WXAppID, OpenID: r.WXOpenID,
		Noncestr: c.genNonceStr(24), SignType: "MD5",
		MerchantID: c.payOption.MerchantID,
		OrderNo:    r.OrderNo,
		SendName:   r.MerchantName, Amount: r.TotalAmount, Num: r.TotalNum,
		Wishing: r.Wishing, ClientIP: r.ClientIP,
		ActName: r.ActiveName,
		SceneID: string(r.Scene),
	}
	req.RiskInfo.PostTime = time.Now()
	req.RiskInfo.DeviceID = r.DeviceID
	req.RiskInfo.Mobile = r.Mobile
	req.RiskInfo.ClientVersion = r.ClientVersion
	c.makePaySign(&req)

	var buf bytes.Buffer
	coder := xml.NewEncoder(&buf)
	coder.Encode(req)

	var result payment.RedPackageResponse
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: c.tlsCfg},
	}
	res, err := client.Post(uri, "application/xml", &buf)
	if err != nil {
		return result, err
	}
	defer res.Body.Close()
	var reply SendRedPackageReply
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
	result.PlatOrderNo = reply.SendListID
	result.WXAppID = reply.AppID
	result.TotalAmount = reply.Amount
	result.WXOpenID = reply.OpenID
	return result, nil
}
