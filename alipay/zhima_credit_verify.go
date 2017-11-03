package alipay

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ZhimaCreditVerifyRequest 芝麻欺骗认证请求参数
type ZhimaCreditVerifyRequest struct {
	// TransactionID 商户请求的唯一标志，长度64位以内字符串，仅限字母数字下划线组合。
	// 该标识作为业务调用的唯一标识，商户要保证其业务唯一性，
	// 使用相同transaction_id的查询，芝麻在一段时间内（一般为1天）返回首次查询结果，
	// 超过有效期的查询即为无效并返回异常，有效期内的重复查询不重新计费
	TransactionID string `json:"transaction_id"`
	// CertNo 身份证号码
	CertNo string `json:"cert_no"`
	// 证件类型。IDENTITY_CARD标识为身份证，目前仅支持身份证类型
	CertType    string `json:"cert_type"`
	Name        string `json:"name"`
	Mobile      string `json:"mobile,omitempty"`
	Email       string `json:"email,omitempty"`
	ProductCode string `json:"product_code"`
}
type zhimaCreditVerifyResponse struct {
	Sign     string
	Response json.RawMessage `json:"zhima_credit_antifraud_verify_response"`
}
type zhimaCreditVerifyReply struct {
	commonReply
	BizNo      string   `json:"biz_no"`
	VerifyCode []string `json:"verify_code"`
}

// ZhimaCreditVerify 芝麻欺诈认证
func (c *AlipayClient) ZhimaCreditVerify(r ZhimaCreditVerifyRequest) (err error) {
	r.ProductCode = "w1010100000000002859"
	r.CertType = "IDENTITY_CARD"

	actReq := actReq{
		method:   "zhima.credit.antifraud.verify",
		signType: SignTypeRSA2, data: r,
	}
	params := c.makeParams(actReq)
	var res zhimaCreditVerifyResponse
	err = c.do(params, &res)
	if err != nil {
		return
	}
	if err = c.verifySign(SignTypeRSA2, res.Response, res.Sign); err != nil {
		return fmt.Errorf("verify sign error %v \r\n", err)
	}
	var reply zhimaCreditVerifyReply
	json.Unmarshal(res.Response, &reply)
	if err = reply.checkErr(); err != nil {
		return
	}
	ismatch, err := verifyCode(reply.VerifyCode)
	if err != nil {
		return
	}
	if !ismatch {
		err = errors.New("信息无法验证")
	}
	return
}

func verifyCode(codes []string) (bool, error) {
	ismatch := false
	errs := make([]string, 0)
	for _, code := range codes {
		var err string
		switch code {
		case "V_CN_NM_MA", "V_PH_CN_MA_UL30D", "V_PH_CN_MA_UL90D",
			"V_PH_CN_MA_UL180D", "V_PH_CN_MA_UM180D", "V_PH_NM_MA_UL30D", "V_PH_NM_MA_UL90D",
			"V_PH_NM_MA_UL180D", "V_PH_NM_MA_UM180D", "V_EM_CN_MA", "V_EM_PH_MA":
			ismatch = true
		case "V_CN_NA":
			err = "查询不到身份证信息"
		case "V_CN_NM_UM":
			err = "姓名与身份证号不匹配"
		case "V_PH_NA":
			err = "查询不到电话号码信息"
		case "V_PH_CN_UM":
			err = "电话号码与本人不匹配"
		case "V_PH_NM_UM":
			err = "电话号码与姓名不匹配"
		case "V_EM_CN_UM":
			err = "EMAIL与本人不匹配"
		case "V_EM_PH_UM":
			err = "EMAIL与手机号码不匹配"
		}
		if err != "" {
			errs = append(errs, err)
		}
	}
	if len(errs) <= 0 {
		return ismatch, nil
	}
	return ismatch, errors.New(strings.Join(errs, ";"))
}
