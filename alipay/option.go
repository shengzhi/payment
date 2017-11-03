package alipay

import "log"

// OptionHandlerFunc 配置设置
type OptionHandlerFunc func(c *AlipayClient)

// WithRSAKey 配置RSA签名Key
func WithRSAKey(publicKey, privateKey []byte) OptionHandlerFunc {
	return func(c *AlipayClient) {
		c.cfg.rsaPubKey, c.cfg.rsaPriKey = publicKey, privateKey
	}
}

// EnableSandBox 启用沙箱环境
func EnableSandBox() OptionHandlerFunc {
	return func(c *AlipayClient) { c.cfg.apiDomain = "https://openapi.alipaydev.com/gateway.do" }
}

// WithNotifyURL 支付异步通知回调地址
func WithNotifyURL(url string) OptionHandlerFunc {
	return func(c *AlipayClient) { c.cfg.notifyURL = url }
}

// WithTracer 设置日志跟踪
func WithTracer(tracer *log.Logger) OptionHandlerFunc {
	return func(c *AlipayClient) { c.tracer = tracer }
}
