package alipay

import "net/url"

// TradeAppPay App 支付
func (c *AlipayClient) TradeAppPay(bizData interface{}) (string, error) {
	req := actReq{
		method:   "alipay.trade.app.pay",
		data:     bizData,
		signType: SignTypeRSA2,
		params:   url.Values{},
	}
	req.params.Add("notify_url", c.cfg.notifyURL)
	params := c.makeParams(req)
	return params.Encode(), nil
}

// TradeWapPay 手机网站支付
func (c *AlipayClient) TradeWapPay(bizData interface{}, returnURL string) (string, error) {
	req := actReq{
		method:   "alipay.trade.wap.pay",
		data:     bizData,
		signType: SignTypeRSA2,
		params:   url.Values{},
	}
	req.params.Add("notify_url", c.cfg.notifyURL)
	if returnURL != "" {
		req.params.Add("return_url", returnURL)
	}
	params := c.makeParams(req)
	return c.buildHTML("post", params), nil
}
