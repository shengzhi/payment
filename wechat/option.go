package wechat

import (
	"time"
)

type OptionFunc func(c *Client)

// WithTimeOut 设置超时时长
func WithTimeOut(d time.Duration) OptionFunc {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// WithCurrency 设置货币类型
func WithCurrency(currency string) OptionFunc {
	return func(c *Client) { c.payOption.FeeType = currency }
}

// WithNotifyURL 设置通知地址
func WithNotifyURL(notifyURL string) OptionFunc {
	return func(c *Client) { c.payOption.NotifyURL = notifyURL }
}

// WithCertFile 设置证书路径
func WithCertFile(caroot, clientCrt, clientKey string) OptionFunc {
	return func(c *Client) {
		c.caroot, c.clientcrt, c.clientkey = caroot, clientCrt, clientKey
	}
}
