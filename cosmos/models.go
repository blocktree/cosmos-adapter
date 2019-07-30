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
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	owcrypt "github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/crypto"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tidwall/gjson"
)

type Block struct {
	Hash          string
	VersionBlock  byte
	VersionApp    byte
	ChainID       string
	Height        uint64
	Timestamp     uint64
	PrevBlockHash string
	Transactions  []string
}

type TxValue struct {
	//MsgType string
	From   string
	To     string
	Amount uint64
	Status string
	Reason string
	// Denom  string
}

type FeeValue struct {
	Amount uint64
	//	Denom  string
}

type Transaction struct {
	TxType      string
	TxID        string
	Fee         []FeeValue
	Gas         uint64
	TimeStamp   uint64
	TxValue     []TxValue
	BlockHeight uint64
	Memo        string
}

func NewTransaction(json *gjson.Result, txType, msgType, denom string) *Transaction {

	obj := &Transaction{}
	obj.TxType = json.Get("tx").Get("type").String()
	if obj.TxType != txType {
		return &Transaction{}
	}

	msgList := json.Get("tx").Get("value").Get("msg").Array()
	feeList := json.Get("tx").Get("value").Get("fee").Get("amount").Array()
	logList := json.Get("logs").Array()
	reason := ""
	var status string

	if strings.Contains(json.Get("raw_log").String(), "true") {
		status = "true"
	} else {
		status = "false"
	}
	if logList == nil || len(logList) == 0 {
		reason = gjson.Get(json.Get("raw_log").String(), "message").String()
		status = "false"
	}
	for _, msg := range msgList {
		if msg.Get("type").String() == msgType {
			for _, coin := range msg.Get("value").Get("amount").Array() {
				if coin.Get("denom").String() == denom {
					obj.TxValue = append(obj.TxValue, TxValue{
						From:   msg.Get("value").Get("from_address").String(),
						To:     msg.Get("value").Get("to_address").String(),
						Amount: coin.Get("amount").Uint(),
						Status: status,
						Reason: reason,
					})

					if feeList != nil && len(feeList) > 0 {
						obj.Fee = append(obj.Fee, FeeValue{feeList[0].Get("amount").Uint()})
					} else {
						obj.Fee = nil
					}
				}
			}

		}
		if msg.Get("type").String() == "cosmos-sdk/MsgMultiSend" {
			for _, input := range msg.Get("value").Get("inputs").Array() {
				for _, coin := range input.Get("coins").Array() {
					if coin.Get("denom").String() == denom {
						obj.TxValue = append(obj.TxValue, TxValue{
							From: input.Get("address").String(),
							To:   "multiaddress",
							Amount: coin.Get("amount").Uint(),
							Status: status,
							Reason:reason,
						})
					}
				}
			}

			for _, output := range msg.Get("value").Get("outputs").Array() {
				for _, coin := range output.Get("coins").Array() {
					if coin.Get("denom").String() == denom {
						obj.TxValue = append(obj.TxValue, TxValue{
							From: "multiaddress",
							To: output.Get("address").String(),
							Amount: coin.Get("amount").Uint(),
							Status: status,
							Reason:reason,
						})
					}
				}
			}
			if feeList != nil && len(feeList) > 0 {
				obj.Fee = append(obj.Fee, FeeValue{feeList[0].Get("amount").Uint()})
			} else {
				obj.Fee = nil
			}
		}
	}
	obj.Gas = json.Get("tx").Get("value").Get("fee").Get("gas").Uint()
	obj.TxID = json.Get("txhash").String()
	timestamp, _ := time.Parse(time.RFC3339Nano, json.Get("timestamp").String())
	obj.TimeStamp = uint64(timestamp.Unix())
	obj.BlockHeight = json.Get("height").Uint()
	obj.Memo = json.Get("tx").Get("value").Get("memo").String()
	return obj
}

func NewBlock(json *gjson.Result) *Block {
	obj := &Block{}

	// 解析
	obj.Hash = gjson.Get(json.Raw, "block_meta").Get("block_id").Get("hash").String()
	obj.VersionBlock = byte(gjson.Get(json.Raw, "block_meta").Get("header").Get("version").Get("block").Uint())
	obj.VersionApp = byte(gjson.Get(json.Raw, "block_meta").Get("header").Get("version").Get("app").Uint())
	obj.ChainID = gjson.Get(json.Raw, "block_meta").Get("header").Get("chain_id").String()
	obj.Height = gjson.Get(json.Raw, "block_meta").Get("header").Get("height").Uint()
	timestamp, _ := time.Parse(time.RFC3339Nano, gjson.Get(json.Raw, "block_meta").Get("header").Get("time").String())
	obj.Timestamp = uint64(timestamp.Unix())
	obj.PrevBlockHash = gjson.Get(json.Raw, "block_meta").Get("header").Get("last_block_id").Get("hash").String()

	if gjson.Get(json.Raw, "block_meta").Get("header").Get("num_txs").Uint() != 0 {
		txs := gjson.Get(json.Raw, "block").Get("data").Get("txs").Array()

		for _, tx := range txs {
			txid, _ := base64.StdEncoding.DecodeString(tx.String())
			obj.Transactions = append(obj.Transactions, hex.EncodeToString(owcrypt.Hash(txid, 0, owcrypt.HASH_ALG_SHA256)))
		}
	}

	return obj
}

//BlockHeader 区块链头
func (b *Block) BlockHeader() *openwallet.BlockHeader {

	obj := openwallet.BlockHeader{}
	//解析json
	obj.Hash = b.Hash
	//obj.Confirmations = b.Confirmations
	//	obj.Merkleroot = b.TransactionMerkleRoot
	obj.Previousblockhash = b.PrevBlockHash
	obj.Height = b.Height
	//obj.Version = uint64(b.Version)
	obj.Time = b.Timestamp
	obj.Symbol = Symbol

	return &obj
}

//UnscanRecords 扫描失败的区块及交易
type UnscanRecord struct {
	ID          string `storm:"id"` // primary key
	BlockHeight uint64
	TxID        string
	Reason      string
}

func NewUnscanRecord(height uint64, txID, reason string) *UnscanRecord {
	obj := UnscanRecord{}
	obj.BlockHeight = height
	obj.TxID = txID
	obj.Reason = reason
	obj.ID = common.Bytes2Hex(crypto.SHA256([]byte(fmt.Sprintf("%d_%s", height, txID))))
	return &obj
}
