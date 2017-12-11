package wechat

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/shengzhi/payment"
)

var ErrRefundRetry = errors.New("WX server error, please retry")

const wx_pay_refund_url = "https://api.mch.weixin.qq.com/secapi/pay/refund"

type RefundRequest struct {
	XMLName       xml.Name `xml:"xml"`
	APPID         string   `xml:"appid" sign:"appid"`
	MerchantID    string   `xml:"mch_id" sign:"mch_id"`
	Noncestr      string   `xml:"nonce_str" sign:"nonce_str"`
	Sign          string   `xml:"sign"`
	SignType      string   `xml:"sign_type" sign:"sign_type"`
	TransactionID string   `xml:"transaction_id" sign:"transaction_id"`
	OutTradeNo    string   `xml:"out_trade_no" sign:"out_trade_no"`
	OutRefundNo   string   `xml:"out_refund_no" sign:"out_refund_no"`
	OrderFee      int32    `xml:"total_fee" sign:"total_fee"`
	RefundFee     int32    `xml:"refund_fee" sign:"refund_fee"`
	Currency      string   `xml:"refund_fee_type" sign:"refund_fee_type"`
	Reason        string   `xml:"refund_desc" sign:"refund_desc"`
}

func (r *RefundRequest) setSign(sign string) { r.Sign = sign }

type RefundResponse struct {
	XMLName             xml.Name `xml:"xml"`
	ReturnCode          string   `xml:"return_code" sign:"return_code"`
	ReturnMsg           string   `xml:"return_msg" sign:"return_msg"`
	AppID               string   `xml:"appid" sign:"appid"`
	MerchantID          string   `xml:"mch_id" sign:"mch_id"`
	NonceStr            string   `xml:"nonce_str" sign:"nonce_str"`
	Sign                string   `xml:"sign"`
	ResultCode          string   `xml:"result_code" sign:"result_code"`
	ErrCode             string   `xml:"err_code" sign:"err_code"`
	ErrDesc             string   `xml:"err_code_des" sign:"err_code_des"`
	TransactionID       string   `xml:"transaction_id" sign:"transaction_id"`
	OutTradeNo          string   `xml:"out_trade_no" sign:"out_trade_no"`
	OutRefundNo         string   `xml:"out_refund_no" sign:"out_refund_no"`
	RefundID            string   `xml:"refund_id" sign:"refund_id"`
	RefundFee           int32    `xml:"refund_fee" sign:"refund_fee"`
	OrderFee            int32    `xml:"total_fee" sign:"total_fee"`
	SettlementRefundFee int32    `xml:"settlement_refund_fee" sign:"settlement_refund_fee"`
	SettlementOrderFee  int32    `xml:"settlement_total_fee" sign:"settlement_total_fee"`
	Currency            string   `xml:"fee_type" sign:"fee_type"`
	CashFee             int32    `xml:"cash_fee" sign:"cash_fee"`
	CashFeeCurrency     string   `xml:"cash_fee_type" sign:"cash_fee_type"`
	CashRefundFee       int32    `xml:"cash_refund_fee" sign:"cash_refund_fee"`
}

func (r RefundResponse) getSign() string { return r.Sign }

func (c *Client) Refund(req payment.RefundRequest) (payment.RefundResponse, error) {
	refundReq := &RefundRequest{
		APPID: c.appid, MerchantID: c.payOption.MerchantID,
		Noncestr: c.genNonceStr(24), SignType: "MD5",
		OutTradeNo: req.MerchantOrderNo, OutRefundNo: req.MerchantRefundNo,
		OrderFee: req.TotalFee, RefundFee: req.RefundFee,
		Currency: "CNY", Reason: req.Reason,
	}
	c.makePaySign(refundReq)
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		enc := xml.NewEncoder(pw)
		if err := enc.Encode(refundReq); err != nil {
			log.Println(fmt.Errorf("Payment: marshal struct to xml error:%v", err))
		}
	}()
	var result payment.RefundResponse
	var refundResp RefundResponse
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: c.tlsCfg},
	}
	res, err := client.Post(wx_pay_refund_url, "application/xml", pr)
	if err != nil {
		return result, err
	}
	defer res.Body.Close()
	err = xml.NewDecoder(res.Body).Decode(&refundResp)
	if err != nil {
		return result, fmt.Errorf("Payment: decode xml to struct error:%v", err)
	}
	if refundResp.ReturnCode != "SUCCESS" {
		return result, fmt.Errorf("Payment: %s-%s", refundResp.ReturnCode, refundResp.ReturnMsg)
	}
	if refundResp.ResultCode == "FAIL" {
		switch refundResp.ErrCode {
		case "SYSTEMERROR", "BIZERR_NEED_RETRY", "FREQUENCY_LIMITED":
			return result, ErrRefundRetry
		default:
			return result, fmt.Errorf("Payment:%s-%s", refundResp.ErrCode, refundResp.ErrDesc)
		}
	}
	result.MerchantOrderNo = refundResp.OutTradeNo
	result.MerchantRefundNo = refundResp.OutRefundNo
	result.RefundFee = refundResp.RefundFee
	result.PlatRefundID = refundResp.RefundID
	return result, nil
}

// func (c *Client) mustLoadCertificates() (tls.Certificate, *x509.CertPool) {
// 	cafile := "/home/www/elect/cert/rootca.pem"
// 	certfile := "/home/www/elect/cert/apiclient_cert.pem"
// 	keyfile := "/home/www/elect/cert/apiclient_key.pem"
// 	mycert, err := tls.LoadX509KeyPair(certfile, keyfile)
// 	if err != nil {
// 		panic(err)
// 	}

// 	pem, err := ioutil.ReadFile(cafile)
// 	if err != nil {
// 		panic(err)
// 	}

// 	certPool := x509.NewCertPool()
// 	if !certPool.AppendCertsFromPEM(pem) {
// 		panic("Failed appending certs")
// 	}

// 	return mycert, certPool

// }

// func (c *Client) mustGetTlsConfiguration() *tls.Config {
// 	config := &tls.Config{}
// 	mycert, certPool := c.mustLoadCertificates()
// 	config.Certificates = []tls.Certificate{mycert}
// 	config.RootCAs = certPool
// 	// config.ClientCAs = certPool

// 	config.ClientAuth = tls.RequireAndVerifyClientCert

// 	// //Optional stuff

// 	// //Use only modern ciphers
// 	// config.CipherSuites = []uint16{tls.TLS_RSA_WITH_AES_128_CBC_SHA,
// 	// 	tls.TLS_RSA_WITH_AES_256_CBC_SHA,
// 	// 	tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
// 	// 	tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
// 	// 	tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
// 	// 	tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
// 	// 	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
// 	// 	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256}

// 	// //Use only TLS v1.2
// 	// config.MinVersion = tls.VersionTLS12

// 	// //Don't allow session resumption
// 	// config.SessionTicketsDisabled = true
// 	config.BuildNameToCertificate()
// 	fmt.Println("Get Certificate OK")
// 	return config
// }
