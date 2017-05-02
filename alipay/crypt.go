package alipay

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

func packageData(originalData []byte, packageSize int) (r [][]byte) {
	var src = make([]byte, len(originalData))
	copy(src, originalData)

	r = make([][]byte, 0)
	if len(src) <= packageSize {
		return append(r, src)
	}
	for len(src) > 0 {
		var p = src[:packageSize]
		r = append(r, p)
		src = src[packageSize:]
		if len(src) <= packageSize {
			r = append(r, src)
			break
		}
	}
	return r
}

func (c *Client) rsaEncrypt(plaintext []byte) ([]byte, error) {
	pub := &c.privateKey_.PublicKey

	data := packageData(plaintext, pub.N.BitLen()/8-11)
	cipherData := make([]byte, 0, 0)

	for _, d := range data {
		var c, e = rsa.EncryptPKCS1v15(rand.Reader, pub, d)
		if e != nil {
			return nil, e
		}
		cipherData = append(cipherData, c...)
	}

	return cipherData, nil
}

func (c *Client) rsaDecrypt(ciphertext []byte) ([]byte, error) {
	pri := c.privateKey_
	data := packageData(ciphertext, pri.PublicKey.N.BitLen()/8)
	plainData := make([]byte, 0, 0)

	for _, d := range data {
		var p, e = rsa.DecryptPKCS1v15(rand.Reader, pri, d)
		if e != nil {
			return nil, e
		}
		plainData = append(plainData, p...)
	}
	return plainData, nil
}

func (c *Client) rsa2Encrypt(src []byte, hash crypto.Hash) ([]byte, error) {
	var h = hash.New()
	h.Write(src)
	var hashed = h.Sum(nil)

	return rsa.SignPKCS1v15(rand.Reader, c.privateKey_, hash, hashed)
}

func (c *Client) rsa2Verify(src, sig []byte, hash crypto.Hash) error {
	var h = hash.New()
	h.Write(src)
	var hashed = h.Sum(nil)
	return rsa.VerifyPKCS1v15(&c.privateKey_.PublicKey, hash, hashed, sig)
}

func initRSAPublicKey(key []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, errors.New("publick key error")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pubInterface.(*rsa.PublicKey), nil
}

func initRSAPrivateKey(public, private []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(private)
	if block == nil {
		return nil, errors.New("private key error")
	}

	pri, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	publicKey, err := initRSAPublicKey(public)
	pri.PublicKey = *publicKey
	return pri, err
}
