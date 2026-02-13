// Copyright (C) 2015 The Syncthing Authors.
// Copyright (C) 2022 Ettore Di Giacinto
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

// Package signature provides simple methods to create and verify signatures
// in PEM format.
// Extracted https://github.com/syncthing/syncthing/blob/main/lib/signature/signature.go and adapted to encode directly into base64
// signature包提供了创建和验证PEM格式签名的简单方法。
// 提取自 https://github.com/syncthing/syncthing/blob/main/lib/signature/signature.go 并适配为直接编码为base64

package ecdsa

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
)

// GenerateKeys 返回新的密钥对，私钥和公钥都以PEM格式编码。
func GenerateKeys() (privKey []byte, pubKey []byte, err error) {
	// 生成新的密钥对
	key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	// 序列化私钥
	bs, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, err
	}

	// 以PEM格式编码
	privKey = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: bs,
	})

	// 序列化公钥
	bs, err = x509.MarshalPKIXPublicKey(key.Public())
	if err != nil {
		return nil, nil, err
	}

	// 以PEM格式编码
	pubKey = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PUBLIC KEY",
		Bytes: bs,
	})

	// 编码为base64
	privKey = []byte(base64.URLEncoding.EncodeToString(privKey))
	pubKey = []byte(base64.URLEncoding.EncodeToString(pubKey))

	return
}

// sign 计算数据的哈希值并使用私钥签名，返回PEM格式的签名。
// 参数 privKeyPEM 为PEM格式的私钥，data 为要签名的数据
func sign(privKeyPEM []byte, data io.Reader) ([]byte, error) {
	// 解析私钥
	key, err := loadPrivateKey(privKeyPEM)
	if err != nil {
		return nil, err
	}

	// 计算读取器数据的哈希值
	hash, err := hashReader(data)
	if err != nil {
		return nil, err
	}

	// 对哈希值签名
	r, s, err := ecdsa.Sign(rand.Reader, key, hash)
	if err != nil {
		return nil, err
	}

	// 使用ASN.1序列化签名
	sig, err := marshalSignature(r, s)
	if err != nil {
		return nil, err
	}

	// 编码为PEM块
	bs := pem.EncodeToMemory(&pem.Block{
		Type:  "SIGNATURE",
		Bytes: sig,
	})

	return []byte(base64.URLEncoding.EncodeToString(bs)), nil
}

// verify 计算数据的哈希值并使用给定的公钥与签名进行比较。
// 如果签名正确则返回nil。
// 参数 pubKeyPEM 为PEM格式的公钥，signature 为签名，data 为原始数据
func verify(pubKeyPEM []byte, signature []byte, data io.Reader) error {
	// 解析公钥
	key, err := loadPublicKey(pubKeyPEM)
	if err != nil {
		return err
	}

	// 解码base64签名
	bsDec, err := base64.URLEncoding.DecodeString(string(signature))
	if err != nil {
		return err
	}
	// 解析签名
	block, _ := pem.Decode(bsDec)
	r, s, err := unmarshalSignature(block.Bytes)
	if err != nil {
		return err
	}

	// 计算数据的哈希值
	hash, err := hashReader(data)
	if err != nil {
		return err
	}

	// 验证签名
	if !ecdsa.Verify(key, hash, r, s) {
		return errors.New("签名不正确")
	}

	return nil
}

// hashReader 返回读取器内容的SHA256哈希值
func hashReader(r io.Reader) ([]byte, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return nil, err
	}
	hash := []byte(fmt.Sprintf("%x", h.Sum(nil)))
	return hash, nil
}

// loadPrivateKey 从给定的PEM数据返回ECDSA私钥结构。
func loadPrivateKey(bs []byte) (*ecdsa.PrivateKey, error) {
	bDecoded, err := base64.URLEncoding.DecodeString(string(bs))
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode([]byte(bDecoded))
	return x509.ParseECPrivateKey(block.Bytes)
}

// loadPublicKey 从给定的PEM数据返回ECDSA公钥结构。
func loadPublicKey(bs []byte) (*ecdsa.PublicKey, error) {
	bDecoded := []byte{}
	bDecoded, err := base64.URLEncoding.DecodeString(string(bs))
	if err != nil {
		return nil, err
	}

	// 解码并解析公钥PEM块
	block, _ := pem.Decode(bDecoded)
	intf, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	// 应该是ECDSA公钥
	pk, ok := intf.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("不支持的公钥格式")
	}

	return pk, nil
}

// signature 签名整数包装器，用于序列化和反序列化
type signature struct {
	R, S *big.Int
}

// marshalSignature 返回给定整数的ASN.1编码字节，适合PEM编码。
func marshalSignature(r, s *big.Int) ([]byte, error) {
	sig := signature{
		R: r,
		S: s,
	}

	bs, err := asn1.Marshal(sig)
	if err != nil {
		return nil, err
	}

	return bs, nil
}

// unmarshalSignature 从给定的ASN.1编码签名返回R和S整数。
func unmarshalSignature(sig []byte) (r *big.Int, s *big.Int, err error) {
	var ts signature
	_, err = asn1.Unmarshal(sig, &ts)
	if err != nil {
		return nil, nil, err
	}

	return ts.R, ts.S, nil
}
