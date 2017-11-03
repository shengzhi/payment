// 异步通知验证

package alipay

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/shengzhi/payment"
)

func (c *AlipayClient) NotifyCallback(r io.Reader, f payment.NotifyHandleFunc) interface{} {
	buf := c.getBuf()
	defer c.bufPool.Put(buf)
	io.Copy(buf, r)
	val, err := url.ParseQuery(buf.String())
	if err != nil {
		return err.Error()
	}
	var reply appPayReply
	err = c.Verify(val, &reply)
	if err != nil {
		return err.Error()
	}
	result := &payment.NotifyResult{
		Plat:            payment.PayPlatAlipay,
		MerchantOrderNo: reply.OutTrandeNo,
		TransactionID:   reply.TradeNo,
		CompletedTime:   reply.GmtPayment.Time,
		TotalAmount:     int64(reply.TotalAmount * 100),
		Currency:        "CNY",
		Attach:          reply.PassbackParams,
	}
	result.Alipay.BuyerID = reply.BuyerID
	result.Alipay.BuyerLoginID = reply.BuyerLoginID
	result.Alipay.NotifyID = reply.NotifyID
	if err = f(result); err != nil {
		return err.Error()
	}
	return "success"
}

// Verify 异步回到通知验证及解析
func (c *AlipayClient) Verify(params url.Values, v interface{}) error {
	signType := SignType(params.Get("sign_type"))
	sign := params.Get("sign")
	params.Del("sign_type")
	params.Del("sign")

	plainTxt := c.makePlainTxt(params)
	err := c.verifySign(signType, plainTxt, sign)
	if err != nil {
		return fmt.Errorf("Verify sign failed,error:%v", err)
	}
	return mapToStruct(params, v)
}

func mapToStruct(params url.Values, v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("Paramter v must be pointer to struct")
	}
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("Paramter v must be pointer to struct")
	}

	tp := val.Type()
	for i := 0; i < tp.NumField(); i++ {
		if !val.Field(i).CanSet() {
			continue
		}
		name := tp.Field(i).Name
		if tag, has := tp.Field(i).Tag.Lookup("json"); has {
			name = strings.SplitN(tag, ",", 2)[0]
		}
		value := params.Get(name)
		if value == "" {
			continue
		}
		switch tp.Field(i).Type.Kind() {
		case reflect.String:
			val.Field(i).SetString(value)
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8:
			uintv, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return fmt.Errorf("Cant convert value %s to field %s", value, name)
			}
			val.Field(i).SetUint(uintv)
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8:
			intv, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("Cant convert value %s to field %s", value, name)
			}
			val.Field(i).SetInt(intv)
		case reflect.Float32, reflect.Float64:
			floatv, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("Cant convert value %s to field %s", value, name)
			}
			val.Field(i).SetFloat(floatv)
		case reflect.Bool:
			boolv, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("Cant convert value %s to field %s", value, name)
			}
			val.Field(i).SetBool(boolv)
		case reflect.Struct:
			if tp.Field(i).Name == "AlipayTime" {
				dv, err := parseAlipayTime(value)
				if err != nil {
					return fmt.Errorf("Cant convert value %s to field %s", value, name)
				}
				val.Field(i).Set(reflect.ValueOf(dv))
			} else {
				json.Unmarshal([]byte(value), val.Field(i).Addr().Interface())
			}
		default:
			return fmt.Errorf("Cant convert value %s to field %s", value, name)
		}
	}
	return nil
}
