package alipay

import (
	"fmt"
	"time"
)

type SignType string

func (s SignType) String() string { return string(s) }

const (
	SignTypeMD5  SignType = "MD5"
	SignTypeRSA  SignType = "RSA"
	SignTypeRSA2 SignType = "RSA2"
)

const alipay_time_format = "2006-01-02 15:04:05"

type AlipayTime struct {
	time.Time
}

func (at AlipayTime) MarshalJSON() ([]byte, error) {
	return []byte(at.Format(alipay_time_format)), nil
}

func (at *AlipayTime) UnmarshalJSON(data []byte) error {
	t, err := time.ParseInLocation(alipay_time_format, string(data), time.Local)
	if err != nil {
		return err
	}
	at.Time = t
	return nil
}

func parseAlipayTime(v string) (AlipayTime, error) {
	t, err := time.ParseInLocation(alipay_time_format, v, time.Local)
	if err != nil {
		return AlipayTime{}, err
	}
	return AlipayTime{Time: t}, nil
}

const (
	success_code = "10000"
)

type commonReply struct {
	Code    string
	Msg     string
	SubCode string `json:"sub_code"`
	SubMsg  string `json:"sub_msg"`
}

func (r commonReply) checkErr() error {
	if r.Code != success_code {
		return fmt.Errorf("code:%s,error:%s", r.Code, r.Msg)
	}

	return nil
}
