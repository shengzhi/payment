// APP支付

package alipay

import "github.com/shengzhi/payment"
import "fmt"

type appPayRequest struct {
	Body               string       `json:"body,omitempty"`
	Subject            string       `json:"subject,omitempty"`
	OutTradeNo         string       `json:"out_trade_no,omitempty"`
	Timeout            string       `json:"timeout_express,omitempty"`
	TotalAmount        string       `json:"total_amount,omitempty"` //单位：元
	SellerID           string       `json:"seller_id,omitempty"`    //收款支付宝用户ID。 如果该值为空，则默认为商户签约账号对应的支付宝用户ID
	ProductCode        string       `json:"product_code,omitempty"` //商家和支付宝签约的产品码，为固定值QUICK_MSECURITY_PAY
	GoodsType          string       `json:"goods_type,omitempty"`   //商品主类型：0—虚拟类商品，1—实物类商品
	PassbackParams     string       `json:"passback_params,omitempty"`
	PromoParams        string       `json:"promo_params,omitempty"`
	ExtendParams       *extendParam `json:"extend_params,omitempty"`
	EnablePayChannels  string       `json:"enable_pay_channels,omitempty"`
	DisablePayChannels string       `json:"disable_pay_channels,omitempty"`
	StoreID            string       `json:"store_id,omitempty"`
}

type extendParam struct {
	ServiceProviderID string `json:"sys_service_provider_id,omitempty"`
	//是否发起实名校验
	//T：发起
	//F：不发起
	NeedRealName string `json:"needBuyerRealnamed,omitempty"`
	Remark       string `json:"TRANS_MEMO,omitempty"` //账务备注
}

type appPayReply struct {
	NotifyTime        AlipayTime `json:"notify_time"`
	NotifyType        string     `json:"notify_type"`
	NotifyID          string     `json:"notify_id"`
	APPID             string     `json:"app_id"`
	Charset           string     `json:"charset"`
	Version           string     `json:"version"`
	SingType          SignType   `json:"sign_type"`
	Sign              string     `json:"sign"`
	TradeNo           string     `json:"trade_no"`
	OutTrandeNo       string     `json:"out_trade_no"`
	OutBizNo          string     `json:"out_biz_no"`
	BuyerID           string     `json:"buyer_id"`
	BuyerLoginID      string     `json:"buyer_logon_id"`
	SellerID          string     `json:"seller_id"`
	SellerEmail       string     `json:"seller_email"`
	TradeStatus       string     `json:"trade_status"`
	TotalAmount       float32    `json:"total_amount"`
	ReceiptAmount     float32    `json:"receipt_amount"`
	InvoiceAmount     float32    `json:"invoice_amount"`
	BuyerPayAmount    float32    `json:"buyer_pay_amount"`
	PointAmount       float32    `json:"point_amount"`
	RefundFee         float32    `json:"refund_fee"`
	Subject           string     `json:"subject"`
	Body              string     `json:"body"`
	GmtCreate         AlipayTime `json:"gmt_create"`
	GmtPayment        AlipayTime `json:"gmt_payment"`
	GmtRefund         AlipayTime `json:"gmt_refund"`
	GmtClose          AlipayTime `json:"gmt_close"`
	FundBillList      string     `json:"fund_bill_list"`
	PassbackParams    string     `json:"passback_params"`
	VoucherDetailList string     `json:"voucher_detail_list"`
}

// Order 统一下单
func (c *AlipayClient) Order(order *payment.OrderRequest) (*payment.OrderResponse, error) {
	orderReq := appPayRequest{
		Body:        order.Desc,
		Subject:     order.Subject,
		OutTradeNo:  order.MerchanOrderNo,
		Timeout:     "1d", // TODO
		TotalAmount: fmt.Sprintf("%.2f", float64(order.Amount)/100),
		GoodsType:   "0",
	}
	if order.Source == payment.PaySourceApp {
		orderReq.ProductCode = "QUICK_MSECURITY_PAY"
		res := payment.OrderResponse{}
		var err error
		res.Alipay.PayForm, err = c.TradeAppPay(orderReq)
		return &res, err
	}
	if order.Source == payment.PaySourceWap {
		orderReq.ProductCode = "QUICK_WAP_WAY"
		res := payment.OrderResponse{}
		var err error
		res.Alipay.PayForm, err = c.TradeWapPay(orderReq, order.ReturnURL)
		return &res, err
	}
	return nil, fmt.Errorf("Not Support")
}
