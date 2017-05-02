package alipay

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shengzhi/payment"
)

var _ payment.Provider = &Client{}

// const api_gateway = "https://openapi.alipay.com/gateway.do"

func init() {
	http.DefaultClient.Timeout = time.Second * 30
	http.DefaultClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
}

type Config struct {
	NotifyURL string
}
type Client struct {
	appId      string
	apiDomain  string
	partnerId  string
	client     *http.Client
	privateKey *rsa.PrivateKey
	bufPool    *sync.Pool
	conf       Config
}

// NewClient 创建支付宝客户端
func NewClient(apigateway, appID, partnerID string, publicKey, privateKey []byte) (client *Client, err error) {
	client = &Client{}
	client.appId = appID
	client.partnerId = partnerID
	client.client = http.DefaultClient
	client.apiDomain = apigateway
	client.bufPool = &sync.Pool{
		New: func() interface{} { return new(bytes.Buffer) },
	}
	client.privateKey, err = initRSAPrivateKey(publicKey, privateKey)
	return
}

// SetConfig 配置客户端
func (c *Client) SetConfig(conf Config) {
	c.conf = conf
}
func (c *Client) getBuf() *bytes.Buffer {
	buf := c.bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

type actReq struct {
	method   string
	data     interface{}
	signType SignType
	params   url.Values
}

func (c *Client) makeParams(req actReq) url.Values {
	var p = url.Values{}
	p.Add("app_id", c.appId)
	p.Add("method", req.method)
	p.Add("format", "JSON")
	p.Add("charset", "utf-8")
	p.Add("sign_type", req.signType.String())
	p.Add("timestamp", time.Now().Format("2006-01-02 15:04:05"))
	p.Add("version", "1.0")
	if len(req.params) > 0 {
		for k, v := range req.params {
			p.Add(k, v[0])
		}
	}
	content, _ := json.Marshal(&req.data)
	p.Add("biz_content", string(content))
	p.Add("sign", c.makeSign(req.signType, c.makePlainTxt(p)))
	return p
}

func (c *Client) makePlainTxt(params url.Values) []byte {
	var keys = make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	buf := c.getBuf()
	defer c.bufPool.Put(buf)

	for i := range keys {
		fmt.Fprintf(buf, "%s=%s&", keys[i], strings.TrimSpace(params.Get(keys[i])))
	}
	buf.Truncate(buf.Len() - 1)
	return buf.Bytes()
}

func (c *Client) makeSign(signType SignType, src []byte) string {
	switch signType {
	case SignTypeRSA2:
		cipher, err := c.rsa2Encrypt(src, crypto.SHA256)
		if err != nil {
			return ""
		}
		return base64.StdEncoding.EncodeToString(cipher)
	case SignTypeRSA:
		cipher, err := c.rsaEncrypt(src)
		if err != nil {
			return ""
		}
		return base64.StdEncoding.EncodeToString(cipher)
	default:
		return ""
	}
}

func (c *Client) verifySign(signType SignType, src []byte, sign string) error {
	switch signType {
	case SignTypeRSA2:
		cipherTxt, err := base64.StdEncoding.DecodeString(sign)
		if err != nil {
			return err
		}
		return c.rsa2Verify(src, cipherTxt, crypto.SHA256)
	default:
		return errors.New("verify failed")
	}
}

func (c *Client) do(params url.Values, reply interface{}) error {
	rep, err := c.client.PostForm(c.apiDomain, params)
	if err != nil {
		return err
	}
	defer rep.Body.Close()
	return json.NewDecoder(rep.Body).Decode(reply)
}
