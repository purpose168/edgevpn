// Copyright © 2021-2022 Ettore Di Giacinto <mudler@mocaccino.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// AESEncrypt 使用AES-GCM加密明文
// 参数 plaintext 为要加密的明文，key 为32字节的加密密钥
// 返回十六进制编码的密文字符串和可能的错误
func AESEncrypt(plaintext string, key *[32]byte) (ciphertext string, err error) {
	// 创建AES密码块
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return "", err
	}

	// 加密数据
	cypher := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// 转换为十六进制字符串
	cyp := fmt.Sprintf("%x", cypher)
	return cyp, nil
}

// AESDecrypt 使用AES-GCM解密密文
// 参数 text 为十六进制编码的密文，key 为32字节的解密密钥
// 返回解密后的明文字符串和可能的错误
func AESDecrypt(text string, key *[32]byte) (plaintext string, err error) {
	// 解码十六进制密文
	ciphertext, _ := hex.DecodeString(text)

	// 创建AES密码块
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 检查密文长度
	if len(ciphertext) < gcm.NonceSize() {
		return "", errors.New("密文格式错误")
	}

	// 解密数据
	decodedtext, err := gcm.Open(nil,
		ciphertext[:gcm.NonceSize()],
		ciphertext[gcm.NonceSize():],
		nil,
	)
	if err != nil {
		return "", err
	}

	return string(decodedtext), err
}
