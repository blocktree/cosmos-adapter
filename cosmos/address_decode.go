/*
 * Copyright 2018 The openwallet Authors
 * This file is part of the openwallet library.
 *
 * The openwallet library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The openwallet library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 */

package cosmos

import (
	"fmt"
	"github.com/blocktree/go-owcdrivers/addressEncoder"
	owcrypt "github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/v2/openwallet"
)


type AddressDecoderV2 struct {

	openwallet.AddressDecoderV2Base
	//ScriptPubKeyToBech32Address(scriptPubKey []byte) (string, error)
}
type addressDecoder struct {
	wm *WalletManager //钱包管理者
}

//NewAddressDecoder 地址解析器
func NewAddressDecoderV2(wm *WalletManager) *AddressDecoderV2 {
	decoder := AddressDecoderV2{}
	return &decoder
}

//AddressEncode 地址编码
func (dec *AddressDecoderV2) AddressEncode(hash []byte, opts ...interface{}) (string, error) {

	cfg := addressEncoder.ATOM_mainnetAddress

	pkHash := owcrypt.Hash(hash, 32, owcrypt.HASH_ALG_HASH160)

	address := addressEncoder.AddressEncode(pkHash, cfg)
	return address, nil

}


//AddressDecode 地址解析
func (dec *AddressDecoderV2) AddressDecode(addr string, opts ...interface{}) ([]byte, error) {
	cfg := addressEncoder.ATOM_mainnetAddress

	decodeHash, err := addressEncoder.AddressDecode(addr,cfg)
	if err != nil {
		return nil, err
	}
	return decodeHash, nil
}

// AddressVerify 地址校验
func (dec *AddressDecoderV2) AddressVerify(address string, opts ...interface{}) bool {
	cfg := addressEncoder.ATOM_mainnetAddress

	_, err := addressEncoder.AddressDecode(address,cfg)
	if err != nil {
		return false
	}
	return true
}


//PrivateKeyToWIF 私钥转WIF
func (dec *AddressDecoderV2) PrivateKeyToWIF(priv []byte, isTestnet bool) (string, error) {
	return "", fmt.Errorf("PrivateKeyToWIF not implement")
}

//PublicKeyToAddress 公钥转地址
func (dec *AddressDecoderV2) PublicKeyToAddress(pub []byte, isTestnet bool) (string, error) {

	cfg := addressEncoder.ATOM_mainnetAddress

	pkHash := owcrypt.Hash(pub, 32, owcrypt.HASH_ALG_HASH160)

	address := addressEncoder.AddressEncode(pkHash, cfg)
	return address, nil
}

//WIFToPrivateKey WIF转私钥
func (dec *AddressDecoderV2) WIFToPrivateKey(wif string, isTestnet bool) ([]byte, error) {
	return nil, fmt.Errorf("WIFToPrivateKey not implement")
}

//RedeemScriptToAddress 多重签名赎回脚本转地址
func (dec *AddressDecoderV2) RedeemScriptToAddress(pubs [][]byte, required uint64, isTestnet bool) (string, error) {
	return "", fmt.Errorf("RedeemScriptToAddress not implement")
}

// CustomCreateAddress 创建账户地址
func (dec *AddressDecoderV2) CustomCreateAddress(account *openwallet.AssetsAccount, newIndex uint64) (*openwallet.Address, error) {
	return nil, fmt.Errorf("CreateAddressByAccount not implement")
}

// SupportCustomCreateAddressFunction 支持创建地址实现
func (dec *AddressDecoderV2) SupportCustomCreateAddressFunction() bool {
	return false
}
