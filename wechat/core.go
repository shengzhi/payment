// Package wechat 微信支付实现SDK
package wechat

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"time"

	"sync"

	"github.com/shengzhi/payment"
)

var _ payment.Provider = &Client{}

// WechatPayClient 微信支付客服端
type Client struct {
	appid, secret string
	payOption     Config
	bufpool       *sync.Pool
}

// NewClient 创建微信支付客服端
func NewClient(appid, secret string) *Client {
	rand.Seed(time.Now().UnixNano())
	c := &Client{appid: appid, secret: secret}
	c.bufpool = &sync.Pool{
		New: func() interface{} { return new(bytes.Buffer) },
	}
	return c
}

func (c *Client) getBuf() *bytes.Buffer {
	buf := c.bufpool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}
func (c *Client) makePaySign(req signRequest) {
	b := structToSignMap(req).signString(c.secret)
	req.setSign(strings.ToUpper(md5Encrypt(b)))
}

func (c *Client) validatePayRes(res signResponse) bool {
	p := structToSignMap(res).signString(c.secret)
	return res.getSign() == strings.ToUpper(md5Encrypt(p))
}

// SetPayOption 配置微信支付
func (c *Client) SetPayOption(option Config) { c.payOption = option }

var charMatrix = []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func (c *Client) genNonceStr(length int) string {

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = charMatrix[rand.Intn(len(charMatrix))]
	}
	return string(result)
}

type signRequest interface {
	setSign(sign string)
}

type signResponse interface {
	getSign() string
}

type signMap map[string]string

func (m signMap) signString(secret string) []byte {
	var buf bytes.Buffer
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if v := m[k]; len(v) > 0 {
			fmt.Fprintf(&buf, "%s=%s&", k, v)
		}
	}
	buf.WriteString(fmt.Sprintf("key=%s", secret))
	return buf.Bytes()
}

// Config 微信支付设置
type Config struct {
	MerchantID, DeviceInfo, FeeType string
	Timeout                         time.Duration //交易超时，不低于5分钟
	NotifyURL                       string        //后台通知地址
	LimitPay                        string        //指定支付方式 no_credit--指定不能使用信用卡支付
}

// structToSignMap 结构体转换为
func structToSignMap(v interface{}) signMap {
	val := reflect.ValueOf(v)
	if val.Type().Kind() == reflect.Ptr {
		val = val.Elem()
	}
	sm := make(signMap, val.NumField())
	st := val.Type()
	for i := 0; i < st.NumField(); i++ {
		tag := st.Field(i).Tag.Get("sign")
		if len(tag) <= 0 {
			continue
		}
		kind := st.Field(i).Type.Kind()
		switch kind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			strVal := fmt.Sprintf("%v", val.Field(i).Interface())
			if strVal != "0" {
				sm[tag] = fmt.Sprintf("%v", val.Field(i).Interface())
			}
		default:
			sm[tag] = fmt.Sprintf("%v", val.Field(i).Interface())
		}

	}
	return sm
}

func md5Encrypt(plainText []byte) string {
	m := md5.New()
	m.Write(plainText)
	return hex.EncodeToString(m.Sum(nil))
}

func toJSON(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}
