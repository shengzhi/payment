// 微信支付实现
// 微信统一下单接口

package wechat

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/shengzhi/payment"
)

var DoubleSubmitError = errors.New("重复提交")

type CDATAString struct {
	Bytes []byte `xml:",cdata"`
}

func (cdata CDATAString) String() string {
	return string(cdata.Bytes)
}

// WXOrderRequest 微信支付统一下单请求
type WXOrderRequest struct {
	XMLName        xml.Name    `xml:"xml"`
	AppID          string      `xml:"appid" sign:"appid"`
	MerchantID     string      `xml:"mch_id" sign:"mch_id"`
	DeviceInfo     string      `xml:"device_info" sign:"device_info"`
	NonceStr       string      `xml:"nonce_str" sign:"nonce_str"`
	Sign           string      `xml:"sign"`
	Body           string      `xml:"body" sign:"body"`
	Detail         CDATAString `xml:"detail," sign:"detail"`
	Attach         string      `xml:"attach,omitempty" sign:"attach"`
	MerchatOrderNo string      `xml:"out_trade_no" sign:"out_trade_no"`
	Currency       string      `xml:"fee_type" sign:"fee_type"`
	TotalAmount    int64       `xml:"total_fee" sign:"total_fee"`
	ClientIP       string      `xml:"spbill_create_ip" sign:"spbill_create_ip"`
	Start          string      `xml:"time_start" sign:"time_start"`
	End            string      `xml:"time_expire" sign:"time_expire"`
	Tag            string      `xml:"goods_tag,omitempty" sign:"goods_tag"`
	NotifyURL      string      `xml:"notify_url" sign:"notify_url"`
	TradeType      string      `xml:"trade_type" sign:"trade_type"`
	ProductID      string      `xml:"product_id" sign:"product_id"` //trade_type=NATIVE，此参数必传。此id为二维码中包含的商品ID，商户自行定义。
	LimitPay       string      `xml:"limit_pay,omitempty" sign:"limit_pay"`
	OpenID         string      `xml:"openid,omitempty" sign:"openid"`
}

func (o *WXOrderRequest) setSign(sign string) { o.Sign = sign }

func (o WXOrderRequest) toSignMap() signMap {
	m := make(signMap, 0)
	m["appid"] = o.AppID
	m["mch_id"] = o.MerchantID
	m["device_info"] = o.DeviceInfo
	m["nonce_str"] = o.NonceStr
	m["body"] = o.Body
	m["detail"] = string(o.Detail.Bytes)
	m["attach"] = o.Attach
	m["out_trade_no"] = o.MerchatOrderNo
	m["fee_type"] = o.Currency
	m["total_fee"] = strconv.FormatInt(o.TotalAmount, 10)
	m["spbill_create_ip"] = o.ClientIP
	m["time_start"] = o.Start
	m["time_expire"] = o.End
	m["goods_tag"] = o.Tag
	m["notify_url"] = o.NotifyURL
	m["trade_type"] = o.TradeType
	m["product_id"] = o.ProductID
	m["limit_pay"] = o.LimitPay
	m["openid"] = o.OpenID
	return m
}

// WXOrderResponse 微信支付返回结果
type WXOrderResponse struct {
	XMLName    xml.Name `xml:"xml"`
	ReturnCode string   `xml:"return_code" sign:"return_code"`
	ReturnMsg  string   `xml:"return_msg" sign:"return_msg"`
	AppID      string   `xml:"appid" sign:"appid"`
	MerchantID string   `xml:"mch_id" sign:"mch_id"`
	DeviceInfo string   `xml:"device_info" sign:"device_info"`
	NonceStr   string   `xml:"nonce_str" sign:"nonce_str"`
	Sign       string   `xml:"sign"`
	ResultCode string   `xml:"result_code" sign:"result_code"`
	ErrCode    string   `xml:"err_code" sign:"err_code"`
	ErrDesc    string   `xml:"err_code_des" sign:"err_code_des"`
	TradeType  string   `xml:"trade_type" sign:"trade_type"`
	PrepayID   string   `xml:"prepay_id" sign:"prepay_id"`
	CodeURL    string   `xml:"code_url" sign:"code_url"`
}

func (r WXOrderResponse) getSign() string { return r.Sign }

func (r WXOrderResponse) toSignMap(secret string) signMap {
	m := make(signMap, 0)
	m["return_code"] = r.ReturnCode
	m["return_msg"] = r.ReturnMsg
	m["appid"] = r.AppID
	m["mch_id"] = r.MerchantID
	m["device_info"] = r.DeviceInfo
	m["nonce_str"] = r.NonceStr
	m["result_code"] = r.ResultCode
	m["err_code"] = r.ErrCode
	m["err_code_des"] = r.ErrDesc
	m["trade_type"] = r.TradeType
	m["prepay_id"] = r.PrepayID
	m["code_url"] = r.CodeURL
	return m
}

// ProductDetails 商品详情集合
type WXProductDetails struct {
	Details []WXProductDetail `json:"goods_detail"`
}

// ProductDetail 商品详情
type WXProductDetail struct {
	GoodsID   string `json:"goods_id"`                 //必填 32 商品编号
	WXGoodsID string `json:"wxpay_goods_id,omitempty"` //可选 32 微信支付定义的统一商品编号
	GoodsName string `json:"goods_name"`               //必填 256 商品名称
	Num       int    `json:"goods_num"`                //必填 商品数量
	Price     int64  `json:"price"`                    //必填 商品单价，单位为分
	Category  string `json:"goods_category"`           // 可选 32 商品类目ID
	Body      string `json:"body,omitempty"`           // 可选 1000 商品描述信息
}

// 统一下单接口
const wx_pay_order_url = "https://api.mch.weixin.qq.com/pay/unifiedorder"

// Order 下单
func (c *Client) Order(order *payment.OrderRequest) (*payment.OrderResponse, error) {
	details := WXProductDetails{Details: make([]WXProductDetail, 0, len(order.Details))}
	for _, d := range order.Details {
		details.Details = append(details.Details, WXProductDetail{
			GoodsID:   d.GoodsID,
			WXGoodsID: d.WXGoodsID,
			GoodsName: d.GoodsName,
			Category:  d.Category,
			Body:      d.Body,
			Num:       d.Num,
			Price:     d.Price,
		})
	}
	wxOrderReq := &WXOrderRequest{
		AppID:          c.appid,
		MerchantID:     c.payOption.MerchantID,
		NonceStr:       c.genNonceStr(32),
		Body:           order.Desc,
		Detail:         CDATAString{toJSON(details)},
		Attach:         order.Attach,
		MerchatOrderNo: order.MerchanOrderNo,
		Currency:       c.payOption.FeeType,
		TotalAmount:    order.Amount,
		ClientIP:       order.ClientIP,
		Start:          time.Now().Format("20060102150405"),
		End:            time.Now().Add(c.payOption.Timeout).Format("20060102150405"),
		Tag:            order.Tag,
		NotifyURL:      c.payOption.NotifyURL,
		TradeType:      order.TradeType,
		ProductID:      order.ProduceID,
		LimitPay:       c.payOption.LimitPay,
		OpenID:         order.OpenID,
	}
	if order.Source == payment.PaySourceApp {
		wxOrderReq.TradeType = "APP"
	} else {
		wxOrderReq.DeviceInfo = "WEB"
		wxOrderReq.TradeType = "JSAPI"
	}

	c.makePaySign(wxOrderReq)
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		enc := xml.NewEncoder(pw)
		if err := enc.Encode(wxOrderReq); err != nil {
			log.Println(fmt.Errorf("Payment: marshal struct to xml error:%v", err))
		}
	}()
	res, err := c.httpClient.Post(wx_pay_order_url, "application/xml", pr)

	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var wxres WXOrderResponse
	err = xml.NewDecoder(res.Body).Decode(&wxres)
	if err != nil {
		return nil, err
	}
	if wxres.ReturnCode != "SUCCESS" {
		return nil, fmt.Errorf("%s-%s", wxres.ReturnCode, wxres.ReturnMsg)
	}
	if wxres.ResultCode != "SUCCESS" {
		if wxres.ErrDesc == "201 商户订单号重复" {
			return nil, DoubleSubmitError
		}
		return nil, fmt.Errorf("%s-%s", wxres.ErrCode, wxres.ErrDesc)
	}
	or := &payment.OrderResponse{}
	or.Wechat.PrepayID = wxres.PrepayID
	if order.Source == payment.PaySourceApp {
		or.Wechat.PayForm = c.genAppPayArgs(wxres.PrepayID)
	} else {
		or.Wechat.PayForm = c.genWebPayArgs(wxres.PrepayID)
	}
	or.Wechat.CodeURL = wxres.CodeURL
	return or, nil
}

func (c *Client) genAppPayArgs(prepayid string) payment.WXPayObject {
	object := payment.WXPayObject{
		APPID:     c.appid,
		Noncestr:  c.genNonceStr(24),
		Package:   "Sign=WXPay",
		PartnerID: c.payOption.MerchantID,
		PrepayID:  prepayid,
		Timestamp: time.Now().Unix(),
	}
	buf := c.getBuf()
	defer c.bufpool.Put(buf)
	fmt.Fprintf(buf, "appid=%s&", object.APPID)
	fmt.Fprintf(buf, "noncestr=%s&", object.Noncestr)
	fmt.Fprintf(buf, "package=%s&", object.Package)
	fmt.Fprintf(buf, "partnerid=%s&", object.PartnerID)
	fmt.Fprintf(buf, "prepayid=%s&", object.PrepayID)
	fmt.Fprintf(buf, "timestamp=%d&", object.Timestamp)
	fmt.Fprintf(buf, "key=%s", c.secret)
	object.Sign = strings.ToUpper(md5Encrypt(buf.Bytes()))
	return object
}

func (c *Client) genWebPayArgs(prepayid string) payment.WXPayObject {
	object := payment.WXPayObject{
		APPID:     c.appid,
		Noncestr:  c.genNonceStr(24),
		Package:   fmt.Sprintf("prepay_id=%s", prepayid),
		Timestamp: time.Now().Unix(),
		SignType:  "MD5",
	}
	buf := c.getBuf()
	defer c.bufpool.Put(buf)

	buf.WriteString(fmt.Sprintf("appId=%s", object.APPID))
	buf.WriteString(fmt.Sprintf("&nonceStr=%s", object.Noncestr))
	buf.WriteString(fmt.Sprintf("&package=%s", object.Package))
	buf.WriteString(fmt.Sprintf("&signType=%s", object.SignType))
	buf.WriteString(fmt.Sprintf("&timeStamp=%d", object.Timestamp))
	buf.WriteString(fmt.Sprintf("&key=%s", c.secret))
	object.Sign = strings.ToUpper(md5Encrypt(buf.Bytes()))
	return object
}
