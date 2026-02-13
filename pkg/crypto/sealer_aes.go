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

// AESSealer AES密封器结构体
type AESSealer struct{}

// Seal 使用AES加密消息
// 参数 message 为要加密的消息，key 为加密密钥
// 返回加密后的字符串和可能的错误
func (*AESSealer) Seal(message, key string) (encoded string, err error) {
	enckey := [32]byte{}
	copy(enckey[:], key)
	return AESEncrypt(message, &enckey)
}

// Unseal 使用AES解密消息
// 参数 message 为要解密的消息，key 为解密密钥
// 返回解密后的字符串和可能的错误
func (*AESSealer) Unseal(message, key string) (decoded string, err error) {
	enckey := [32]byte{}
	copy(enckey[:], key)
	return AESDecrypt(message, &enckey)
}
