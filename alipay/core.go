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
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shengzhi/payment"
)

var _ payment.Provider = &AlipayClient{}

const api_gateway = "https://openapi.alipay.com/gateway.do"

type aliPayConfig struct {
	appId                string
	apiDomain            string
	partnerId            string
	notifyURL            string
	rsaPubKey, rsaPriKey []byte
}

// AlipayClient alipay client
type AlipayClient struct {
	client     *http.Client
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	bufPool    *sync.Pool
	cfg        aliPayConfig
	tracer     *log.Logger
}

// NewClient 创建支付宝客户端
func NewClient(appID, partnerID string, options ...OptionHandlerFunc) *AlipayClient {
	client := &AlipayClient{
		cfg:    aliPayConfig{appId: appID, partnerId: partnerID, apiDomain: api_gateway},
		client: &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}},
	}
	client.bufPool = &sync.Pool{
		New: func() interface{} { return new(bytes.Buffer) },
	}
	for _, fn := range options {
		fn(client)
	}
	var err error
	client.publicKey, err = initRSAPublicKey(client.cfg.rsaPubKey)
	if err != nil {
		log.Fatalln(err)
	}
	client.privateKey, err = initRSAPrivateKey(client.cfg.rsaPriKey)
	if err != nil {
		log.Fatalln(err)
	}
	return client
}

func (c *AlipayClient) getBuf() *bytes.Buffer {
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

func (c *AlipayClient) makeParams(req actReq) url.Values {
	var p = url.Values{}
	p.Add("app_id", c.cfg.appId)
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

func (c *AlipayClient) makePlainTxt(params url.Values) []byte {
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

func (c *AlipayClient) makeSign(signType SignType, src []byte) string {
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

func (c *AlipayClient) verifySign(signType SignType, src []byte, sign string) error {
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

func (c *AlipayClient) do(params url.Values, reply interface{}) error {
	buf := c.getBuf()
	defer c.bufPool.Put(buf)
	buf.WriteString(params.Encode())
	req, err := http.NewRequest("POST", c.cfg.apiDomain, buf)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	c.dumpRequest(req)
	rep, err := c.client.Do(req)
	if rep != nil {
		defer rep.Body.Close()
	}
	if err != nil {
		return err
	}
	c.dumpResponse(rep)
	return json.NewDecoder(rep.Body).Decode(reply)
}

func (c *AlipayClient) dumpRequest(req *http.Request) {
	if c.tracer != nil {
		data, _ := httputil.DumpRequest(req, true)
		c.tracer.Println(string(data))
	}
}

func (c *AlipayClient) dumpResponse(resp *http.Response) {
	if c.tracer != nil {
		data, _ := httputil.DumpResponse(resp, true)
		c.tracer.Println(string(data))
	}
}

func (c *AlipayClient) buildHTML(method string, params url.Values) string {
	buf := c.getBuf()
	defer c.bufPool.Put(buf)
	fmt.Fprint(buf, "<html><body>")
	fmt.Fprintf(buf, "<form id='alipaysubmit' name='alipaysubmit' action='%s?charset=utf-8' method='%s' style='display:none;'>", c.cfg.apiDomain, method)
	for k, v := range params {
		fmt.Fprintf(buf, `<input name='%s' value='%s' />`, k, v[0])
	}
	fmt.Fprintf(buf, "<input type='submit' value='%s' style='display:none;'></form></body>", method)
	fmt.Fprint(buf, "<script>document.forms['alipaysubmit'].submit();</script></html>")
	return buf.String()
}
