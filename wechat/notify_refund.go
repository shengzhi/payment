package wechat

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/shengzhi/payment"
)

// WXRefundNotifyResult 异步通知结果
type WXRefundNotifyResult struct {
	XMLName    xml.Name `xml:"xml"`
	ReturnCode string   `xml:"return_code"`
	ReturnMsg  string   `xml:"return_msg"`
	AppID      string   `xml:"appid"`
	MerchantID string   `xml:"mch_id"`
	NonceStr   string   `xml:"nonce_str"`
	ReqInfo    string   `xml:"req_info"`
}

type WXRefundNotifyInfo struct {
	XMLName           xml.Name `xml:"xml"`
	WXTransID         string   `xml:"transaction_id"`
	OutOrderNo        string   `xml:"out_trade_no"`
	WXRefundID        string   `xml:"refund_id"`
	OutRefundNo       string   `xml:"out_refund_no"`
	TotalAmount       int32    `xml:"total_fee"`
	RefundAmount      int32    `xml:"refund_fee"`
	ActualRefundAmout int32    `xml:"settlement_refund_fee"`
	Status            string   `xml:"refund_status"`
	CompletedTime     string   `xml:"success_time"`
	RecAccount        string   `xml:"refund_recv_accout"` //退款入账账户
	RefundAccount     string   `xml:"refund_account"`
	Source            string   `xml:"refund_request_source"` //退款发起来源
}

// RefundNotifyHandler 退款异步通知回调处理
func (c *Client) RefundNotifyHandler(in io.Reader, fn payment.RefundNotifyHandleFunc) WXNotifyReply {
	var notifyResult WXRefundNotifyResult
	err := xml.NewDecoder(in).Decode(&notifyResult)
	if err != nil {
		return WXNotifyReply{Code: "FAIL", Message: err.Error()}
	}
	if notifyResult.ReturnCode != "SUCCESS" {
		return WXNotifyReply{Code: "FAIL", Message: fmt.Sprintf("退款失败:%s", notifyResult.ReturnMsg)}
	}
	cipherTxt, err := base64.StdEncoding.DecodeString(notifyResult.ReqInfo)
	if err != nil {
		return WXNotifyReply{Code: "FAIL", Message: fmt.Sprintf("Base64 decode error: %v", err)}
	}
	info, err := c.decrypteRefundInfo(cipherTxt)
	if err != nil {
		return WXNotifyReply{Code: "FAIL", Message: fmt.Sprintf("AES-256-ECB decrypt error: %v", err)}
	}
	refundResult := payment.RefundNotifyResult{
		Plat:             payment.PayPlatWechat,
		MerchantOrderNo:  info.OutOrderNo,
		MerchantRefundNo: info.OutRefundNo,
		RefundID:         info.WXRefundID,
		RefundAmount:     info.ActualRefundAmout,
		TotalAmount:      info.TotalAmount,
	}
	refundResult.CompletedTime, _ = time.ParseInLocation("20060102150405", info.CompletedTime, time.Local)
	refundResult.IsSuccess = info.Status == "SUCCESS"
	if err = fn(refundResult); err != nil {
		return WXNotifyReply{Code: "FAIL", Message: err.Error()}
	}
	return WXNotifyReply{Code: "SUCCESS", Message: "OK"}
}

func (c *Client) decrypteRefundInfo(cipherTxt []byte) (WXRefundNotifyInfo, error) {
	if len(c.refundKey) <= 0 {
		c.refundKey = []byte(strings.ToLower(md5Encrypt([]byte(c.secret))))
	}

	block, err := aes.NewCipher(c.refundKey)
	if err != nil {
		return WXRefundNotifyInfo{}, err
	}
	ecbMode := NewECBDecrypter(block)
	plainTxt := make([]byte, len(cipherTxt))
	ecbMode.CryptBlocks(plainTxt, cipherTxt)
	plainTxt = pkcs5UnPadding(plainTxt)
	// fmt.Println(string(plainTxt))
	var buf bytes.Buffer
	buf.WriteString("<xml>")
	buf.Write(plainTxt)
	buf.WriteString("</xml>")
	var result WXRefundNotifyInfo
	err = xml.NewDecoder(&buf).Decode(&result)
	return result, err
}

func pkcs5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func pkcs5UnPadding(origData []byte) []byte {
	length := len(origData)
	// 去掉最后一个字节 unpadding 次
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

type ecb struct {
	b         cipher.Block
	blockSize int
}

func newECB(b cipher.Block) *ecb {
	return &ecb{
		b:         b,
		blockSize: b.BlockSize(),
	}
}

type ecbEncrypter ecb

// NewECBEncrypter returns a BlockMode which encrypts in electronic code book
// mode, using the given Block.
func NewECBEncrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbEncrypter)(newECB(b))
}
func (x *ecbEncrypter) BlockSize() int { return x.blockSize }
func (x *ecbEncrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Encrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}

type ecbDecrypter ecb

// NewECBDecrypter returns a BlockMode which decrypts in electronic code book
// mode, using the given Block.
func NewECBDecrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbDecrypter)(newECB(b))
}
func (x *ecbDecrypter) BlockSize() int { return x.blockSize }
func (x *ecbDecrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Decrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}
