package alipay

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/shengzhi/payment"
)

type RefundRequest struct {
	OutTradeNo   string  `json:"out_trade_no"`
	AliTradeNo   string  `json:"trade_no"`
	RefundAmount float32 `json:"refund_amount"`
	Reason       string  `json:"refund_reason"`
	OutRefundID  string  `json:"out_request_no"`
	OperatorID   string  `json:"operator_id"`
	StoreID      string  `json:"store_id"`
	TerminalID   string  `json:"terminal_id"`
}

type TradeRefundResponse struct {
	Sign     string
	Response json.RawMessage `json:"alipay_trade_refund_response"`
}

type TradeRefundReply struct {
	commonReply
	AliRefundID   string     `json:"trade_no"`
	OutTradeNo    string     `json:"out_trade_no"`
	BuyerLoginID  string     `json:"buyer_logon_id"`
	IsFundChanged string     `json:"fund_change"`
	RefundFee     float32    `json:"refund_fee"`
	CompletedTime AlipayTime `json:"gmt_refund_pay"`
	ItemList      []struct {
		Channel    string  `json:"fund_channel"`
		Amount     float32 `json:"amount"`
		RealAmount float32 `json:"real_amount"`
		FundType   string  `json:"fund_type"`
	} `json:"refund_detail_item_list"`
	StoreName   string `json:"store_name"`
	BuyerUserID string `json:"buyer_user_id"`
}

// Refund 退款
func (c *AlipayClient) Refund(req payment.RefundRequest) (payment.RefundResponse, error) {
	var resp payment.RefundResponse
	if req.MerchantOrderNo == "" {
		return resp, fmt.Errorf("缺少商户订单号")
	}
	if req.RefundFee <= 0 {
		return resp, fmt.Errorf("退款金额必须大于0")
	}
	if req.MerchantRefundNo == "" {
		return resp, fmt.Errorf("缺少退款单号")
	}
	bizdata := RefundRequest{
		OutTradeNo:   req.MerchantOrderNo,
		RefundAmount: float32(req.RefundFee / 100),
		Reason:       req.Reason,
		OutRefundID:  req.MerchantRefundNo,
	}
	reply, err := c.tradeRefund(bizdata)
	if err != nil {
		return payment.RefundResponse{}, err
	}

	return payment.RefundResponse{
		MerchantOrderNo:  reply.OutTradeNo,
		MerchantRefundNo: req.MerchantRefundNo,
		PlatRefundID:     reply.AliRefundID,
		RefundFee:        int32(reply.RefundFee * 100),
		IsInstant:        true,
		CompletedTime:    time.Now(),
	}, nil
}

func (c *AlipayClient) tradeRefund(bizData RefundRequest) (TradeRefundReply, error) {
	req := actReq{
		method:   "alipay.trade.refund",
		data:     bizData,
		signType: SignTypeRSA2,
		params:   url.Values{},
	}
	params := c.makeParams(req)
	var resp TradeRefundResponse
	if err := c.do(params, &resp); err != nil {
		return TradeRefundReply{}, err
	}
	if err := c.verifySign(SignTypeRSA2, resp.Response, resp.Sign); err != nil {
		return TradeRefundReply{}, fmt.Errorf("Verify signature failed")
	}
	var reply TradeRefundReply
	err := json.Unmarshal(resp.Response, &reply)
	if err != nil {
		return reply, err
	}
	if err = reply.checkErr(); err != nil {
		return reply, err
	}
	if reply.SubCode != "" {
		return reply, fmt.Errorf("code:%s, message:%s", reply.SubCode, reply.SubMsg)
	}
	return reply, nil
}
