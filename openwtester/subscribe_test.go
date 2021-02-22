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

package openwtester

import (
	"github.com/blocktree/openwallet/v2/common/file"
	"path/filepath"
	"testing"

	"github.com/astaxie/beego/config"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openw"
	"github.com/blocktree/openwallet/v2/openwallet"
)

////////////////////////// 测试单个扫描器 //////////////////////////

type subscriberSingle struct {
}

//BlockScanNotify 新区块扫描完成通知
func (sub *subscriberSingle) BlockScanNotify(header *openwallet.BlockHeader) error {
	log.Notice("header:", header)
	return nil
}

//BlockTxExtractDataNotify 区块提取结果通知
func (sub *subscriberSingle) BlockExtractDataNotify(sourceKey string, data *openwallet.TxExtractData) error {
	log.Notice("account:", sourceKey)

	for i, input := range data.TxInputs {
		log.Std.Notice("data.TxInputs[%d]: %+v", i, input)
	}

	for i, output := range data.TxOutputs {
		log.Std.Notice("data.TxOutputs[%d]: %+v", i, output)
	}

	log.Std.Notice("data.Transaction: %+v", data.Transaction)

	return nil
}

func (sub *subscriberSingle) BlockExtractSmartContractDataNotify(sourceKey string, data *openwallet.SmartContractReceipt) error {
	return nil
}

func TestSubscribeAddress(t *testing.T) {

	var (
		endRunning = make(chan bool, 1)
		symbol     = "ATOM"
		addrs      = map[string]string{
			//"cosmos1rc0ya7shas5wkq8ua0g3zhg6dpzu8l9hj3x72n": "sender",
			"cosmos1kag2d0wkk34qdqm5h3tk03xqewt63xv5z6m56c": "receiver",
		}
	)

	//GetSourceKeyByAddress 获取地址对应的数据源标识
	var scanAddressFunc openwallet.BlockScanTargetFuncV2
	scanAddressFunc = func (target openwallet.ScanTargetParam) openwallet.ScanTargetResult {
		key, ok := addrs[target.ScanTarget]
		if !ok {
			return openwallet.ScanTargetResult{
				SourceKey:  key,
				Exist:      false,
				TargetInfo: nil,
			}
		}
		return openwallet.ScanTargetResult{
			SourceKey:  key,
			Exist:      true,
			TargetInfo: nil,
		}
	}

	assetsMgr, err := openw.GetAssetsAdapter(symbol)
	if err != nil {
		log.Error(symbol, "is not support")
		return
	}

	//读取配置
	absFile := filepath.Join(configFilePath, symbol+".ini")

	c, err := config.NewConfig("ini", absFile)
	if err != nil {
		return
	}
	assetsMgr.LoadAssetsConfig(c)

	assetsLogger := assetsMgr.GetAssetsLogger()
	if assetsLogger != nil {
		assetsLogger.SetLogFuncCall(true)
	}

	//log.Debug("already got scanner:", assetsMgr)
	scanner := assetsMgr.GetBlockScanner()

	if scanner.SupportBlockchainDAI() {
		file.MkdirAll(dbFilePath)
		dai, err := openwallet.NewBlockchainLocal(filepath.Join(dbFilePath, dbFileName), false)
		if err != nil {
			log.Error("NewBlockchainLocal err: %v", err)
			return
		}

		scanner.SetBlockchainDAI(dai)
	}

	scanner.SetRescanBlockHeight(5234708)

	if scanner == nil {
		log.Error(symbol, "is not support block scan")
		return
	}

	scanner.SetBlockScanTargetFuncV2(scanAddressFunc)

	sub := subscriberSingle{}
	scanner.AddObserver(&sub)

	scanner.Run()

	<-endRunning
}
