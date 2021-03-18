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
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/asdine/storm"
	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/v2/common"
	"github.com/blocktree/openwallet/v2/openwallet"
	gosocketio "github.com/graarh/golang-socketio"
	"github.com/graarh/golang-socketio/transport"
	"github.com/shopspring/decimal"
)

const (
	maxExtractingSize = 20 //并发的扫描线程数
)

//ATOMBlockScanner atom的区块链扫描器
type ATOMBlockScanner struct {
	*openwallet.BlockScannerBase

	CurrentBlockHeight   uint64             //当前区块高度
	extractingCH         chan struct{}      //扫描工作令牌
	wm                   *WalletManager     //钱包管理者
	IsScanMemPool        bool               //是否扫描交易池
	RescanLastBlockCount uint64             //重扫上N个区块数量
	socketIO             *gosocketio.Client //socketIO客户端
	RPCServer            int
}

//ExtractResult 扫描完成的提取结果
type ExtractResult struct {
	extractData map[string]*openwallet.TxExtractData
	TxID        string
	BlockHeight uint64
	Success     bool
}

//SaveResult 保存结果
type SaveResult struct {
	TxID        string
	BlockHeight uint64
	Success     bool
}

//NewATOMBlockScanner 创建区块链扫描器
func NewATOMBlockScanner(wm *WalletManager) *ATOMBlockScanner {
	bs := ATOMBlockScanner{
		BlockScannerBase: openwallet.NewBlockScannerBase(),
	}

	bs.extractingCH = make(chan struct{}, maxExtractingSize)
	bs.wm = wm
	bs.IsScanMemPool = bs.wm.Config.IsScanMemPool
	bs.RescanLastBlockCount = 1

	//设置扫描任务
	bs.SetTask(bs.ScanBlockTask)

	return &bs
}

//SetRescanBlockHeight 重置区块链扫描高度
func (bs *ATOMBlockScanner) SetRescanBlockHeight(height uint64) error {
	height = height - 1
	if height < 0 {
		return errors.New("block height to rescan must greater than 0.")
	}

	hash, err := bs.wm.GetBlockHash(height)
	if err != nil {
		return err
	}

	bs.wm.Blockscanner.SaveLocalNewBlock(height, hash)

	return nil
}

//ScanBlockTask 扫描任务
func (bs *ATOMBlockScanner) ScanBlockTask() {

	//获取本地区块高度
	blockHeader, err := bs.GetScannedBlockHeader()
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get new block height; unexpected error: %v", err)
		return
	}

	currentHeight := blockHeader.Height
	currentHash := blockHeader.Hash
	var previousHeight uint64 = 0

	for {

		if !bs.Scanning {
			//区块扫描器已暂停，马上结束本次任务
			return
		}

		//获取最大高度
		maxHeight, err := bs.wm.GetBlockHeight()
		if err != nil {
			//下一个高度找不到会报异常
			bs.wm.Log.Std.Info("block scanner can not get rpc-server block height; unexpected error: %v", err)
			break
		}

		//是否已到最新高度
		if currentHeight >= maxHeight {
			bs.wm.Log.Std.Info("block scanner has scanned full chain data. Current height: %d", maxHeight)
			break
		}

		//继续扫描下一个区块
		currentHeight = currentHeight + 1
		bs.wm.Log.Std.Info("block scanner scanning height: %d ...", currentHeight)

		localBlock, err := bs.wm.RestClient.getBlockByHeight(currentHeight)
		if err != nil {
			bs.wm.Log.Std.Info("getBlockByHeight failed; unexpected error: %v", err)
			break
		}

		isFork := false

		//判断hash是否上一区块的hash
		if currentHash != localBlock.PrevBlockHash {
			previousHeight = currentHeight - 1
			bs.wm.Log.Std.Info("block has been fork on height: %d.", currentHeight)
			bs.wm.Log.Std.Info("block height: %d local hash = %s ", previousHeight, currentHash)
			bs.wm.Log.Std.Info("block height: %d mainnet hash = %s ", previousHeight, localBlock.PrevBlockHash)

			bs.wm.Log.Std.Info("delete recharge records on block height: %d.", previousHeight)

			//删除上一区块链的所有充值记录
			//bs.DeleteRechargesByHeight(currentHeight - 1)
			forkBlock, _ := bs.GetLocalBlock(uint32(previousHeight))
			//删除上一区块链的未扫记录
			bs.wm.Blockscanner.DeleteUnscanRecord(uint32(previousHeight))
			currentHeight = previousHeight - 1 //倒退2个区块重新扫描
			if currentHeight <= 0 {
				currentHeight = 1
			}

			localBlock, err = bs.GetLocalBlock(uint32(currentHeight))
			if err != nil && err != storm.ErrNotFound {
				bs.wm.Log.Std.Error("block scanner can not get local block; unexpected error: %v", err)
				break
			} else if err == storm.ErrNotFound {
				//查找core钱包的RPC
				bs.wm.Log.Info("block scanner prev block height:", currentHeight)

				localBlock, err = bs.wm.RestClient.getBlockByHeight(currentHeight)
				if err != nil {
					bs.wm.Log.Std.Error("block scanner can not get prev block; unexpected error: %v", err)
					break
				}

			}

			//重置当前区块的hash
			currentHash = localBlock.Hash

			bs.wm.Log.Std.Info("rescan block on height: %d, hash: %s .", currentHeight, currentHash)

			//重新记录一个新扫描起点
			bs.wm.Blockscanner.SaveLocalNewBlock(localBlock.Height, localBlock.Hash)

			isFork = true

			if forkBlock != nil {
				//通知分叉区块给观测者，异步处理
				bs.newBlockNotify(forkBlock, isFork)
			}

		} else {

			err = bs.BatchExtractTransaction(localBlock.Height, localBlock.Hash, localBlock.Transactions, false)
			if err != nil {
				bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", err)
			}

			//重置当前区块的hash
			currentHash = localBlock.Hash

			//保存本地新高度
			bs.wm.Blockscanner.SaveLocalNewBlock(currentHeight, currentHash)
			bs.SaveLocalBlock(localBlock)

			isFork = false

			//通知新区块给观测者，异步处理
			bs.newBlockNotify(localBlock, isFork)
		}

	}

	//重扫前N个块，为保证记录找到
	for i := currentHeight - bs.RescanLastBlockCount; i < currentHeight; i++ {
		bs.scanBlock(i)
	}

	if bs.IsScanMemPool {
		//扫描交易内存池
		bs.ScanTxMemPool()
	}

	//重扫失败区块
	bs.RescanFailedRecord()

}

//ScanBlock 扫描指定高度区块
func (bs *ATOMBlockScanner) ScanBlock(height uint64) error {

	block, err := bs.scanBlock(height)
	if err != nil {
		return err
	}
	bs.newBlockNotify(block, false)
	return nil
}

func (bs *ATOMBlockScanner) scanBlock(height uint64) (*Block, error) {

	block, err := bs.wm.RestClient.getBlockByHeight(height)

	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", err)

		//记录未扫区块
		unscanRecord := openwallet.NewUnscanRecord(height, "", err.Error(), bs.wm.Symbol())
		bs.SaveUnscanRecord(unscanRecord)
		bs.wm.Log.Std.Info("block height: %d extract failed.", height)
		return nil, err
	}

	bs.wm.Log.Std.Info("block scanner scanning height: %d ...", block.Height)

	err = bs.BatchExtractTransaction(block.Height, block.Hash, block.Transactions, false)
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", err)
	}

	return block, nil
}

//ScanTxMemPool 扫描交易内存池
func (bs *ATOMBlockScanner) ScanTxMemPool() {

	bs.wm.Log.Std.Info("block scanner scanning mempool ...")

	//提取未确认的交易单
	txIDsInMemPool, err := bs.wm.GetTxIDsInMemPool()
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get mempool data; unexpected error: %v", err)
		return
	}

	if len(txIDsInMemPool) == 0 {
		bs.wm.Log.Std.Info("no transactions in mempool ...")
		return
	}

	err = bs.BatchExtractTransaction(0, "", txIDsInMemPool, true)
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", err)
	}

}

//rescanFailedRecord 重扫失败记录
func (bs *ATOMBlockScanner) RescanFailedRecord() {

	var (
		blockMap = make(map[uint64][]string)
	)

	list, err := bs.GetUnscanRecords()
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get rescan data; unexpected error: %v", err)
	}

	//组合成批处理
	for _, r := range list {

		if _, exist := blockMap[r.BlockHeight]; !exist {
			blockMap[r.BlockHeight] = make([]string, 0)
		}

		if len(r.TxID) > 0 {
			arr := blockMap[r.BlockHeight]
			arr = append(arr, r.TxID)

			blockMap[r.BlockHeight] = arr
		}
	}

	for height, txs := range blockMap {

		var hash string

		if height != 0 {
			bs.wm.Log.Std.Info("block scanner rescanning height: %d ...", height)

			if len(txs) == 0 {

				block, err := bs.wm.RestClient.getBlockByHeight(height)
				if err != nil {
					bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", err)
					continue
				}

				txs = block.Transactions
			}

			err = bs.BatchExtractTransaction(height, hash, txs, false)
			if err != nil {
				bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", err)
				continue
			}
		}
		//删除未扫记录
		bs.wm.Blockscanner.DeleteUnscanRecord(uint32(height))
	}

	//删除未没有找到交易记录的重扫记录
	bs.wm.Blockscanner.DeleteUnscanRecordNotFindTX()
}

//newBlockNotify 获得新区块后，通知给观测者
func (bs *ATOMBlockScanner) newBlockNotify(block *Block, isFork bool) {
	header := block.BlockHeader()
	header.Fork = isFork
	bs.NewBlockNotify(header)
}

//BatchExtractTransaction 批量提取交易单
//bitcoin 1M的区块链可以容纳3000笔交易，批量多线程处理，速度更快
func (bs *ATOMBlockScanner) BatchExtractTransaction(blockHeight uint64, blockHash string, txs []string, memPool bool) error {

	var (
		quit       = make(chan struct{})
		done       = 0 //完成标记
		failed     = 0
		shouldDone = len(txs) //需要完成的总数
	)

	if len(txs) == 0 {
		return nil
		// return errors.New("BatchExtractTransaction block is nil.")
	}

	//生产通道
	producer := make(chan ExtractResult)
	defer close(producer)

	//消费通道
	worker := make(chan ExtractResult)
	defer close(worker)

	//保存工作
	saveWork := func(height uint64, result chan ExtractResult) {
		//回收创建的地址
		for gets := range result {

			if gets.Success {

				notifyErr := bs.newExtractDataNotify(height, gets.extractData)
				//saveErr := bs.SaveRechargeToWalletDB(height, gets.Recharges)
				if notifyErr != nil {
					failed++ //标记保存失败数
					bs.wm.Log.Std.Info("newExtractDataNotify unexpected error: %v", notifyErr)
				}
			} else {
				//记录未扫区块
				unscanRecord := openwallet.NewUnscanRecord(height, "", "", bs.wm.Symbol())
				bs.SaveUnscanRecord(unscanRecord)
				bs.wm.Log.Std.Info("block height: %d extract failed.", height)
				failed++ //标记保存失败数
			}
			//累计完成的线程数
			done++
			if done == shouldDone {
				//bs.wm.Log.Std.Info("done = %d, shouldDone = %d ", done, len(txs))
				close(quit) //关闭通道，等于给通道传入nil
			}
		}
	}

	//提取工作
	extractWork := func(eblockHeight uint64, eBlockHash string, mTxs []string, eProducer chan ExtractResult) {
		for _, txid := range mTxs {
			bs.extractingCH <- struct{}{}
			//shouldDone++
			go func(mBlockHeight uint64, mTxid string, end chan struct{}, mProducer chan<- ExtractResult) {

				//导出提出的交易
				mProducer <- bs.ExtractTransaction(mBlockHeight, eBlockHash, mTxid, bs.ScanTargetFunc, memPool)
				//释放
				<-end

			}(eblockHeight, txid, bs.extractingCH, eProducer)
		}
	}

	/*	开启导出的线程	*/

	//独立线程运行消费
	go saveWork(blockHeight, worker)

	//独立线程运行生产
	go extractWork(blockHeight, blockHash, txs, producer)

	//以下使用生产消费模式
	bs.extractRuntime(producer, worker, quit)

	if failed > 0 {
		return fmt.Errorf("block scanner saveWork failed")
	} else {
		return nil
	}

	//return nil
}

//extractRuntime 提取运行时
func (bs *ATOMBlockScanner) extractRuntime(producer chan ExtractResult, worker chan ExtractResult, quit chan struct{}) {

	var (
		values = make([]ExtractResult, 0)
	)

	for {
		var activeWorker chan<- ExtractResult
		var activeValue ExtractResult
		//当数据队列有数据时，释放顶部，传输给消费者
		if len(values) > 0 {
			activeWorker = worker
			activeValue = values[0]
		}
		select {
		//生成者不断生成数据，插入到数据队列尾部
		case pa := <-producer:
			values = append(values, pa)
		case <-quit:
			//退出
			//bs.wm.Log.Std.Info("block scanner have been scanned!")
			return
		case activeWorker <- activeValue:
			values = values[1:]
		}
	}
	//return

}

//ExtractTransaction 提取交易单
func (bs *ATOMBlockScanner) ExtractTransaction(blockHeight uint64, blockHash string, txid string, scanAddressFunc openwallet.BlockScanTargetFunc, memPool bool) ExtractResult {

	var (
		result = ExtractResult{
			BlockHeight: blockHeight,
			TxID:        txid,
			extractData: make(map[string]*openwallet.TxExtractData),
			Success:     true,
		}
	)

	//bs.wm.Log.Std.Debug("block scanner scanning tx: %s ...", txid)
	var trx *Transaction
	var err error
	if memPool {
		trx, err = bs.wm.GetTransactionInMemPool(txid)
		if err != nil {
			trx, err = bs.wm.GetTransaction(txid)
			if err != nil {
				bs.wm.Log.Std.Info("block scanner can not extract transaction data in mempool and block chain; unexpected error: %v", err)
				result.Success = false
				return result
			}
		}
	} else {
		trx, err = bs.wm.GetTransaction(txid)

		if err != nil {
			bs.wm.Log.Std.Info("block scanner can not extract transaction data; unexpected error: %v", err)
			result.Success = false
			return result
		}
	}

	//优先使用传入的高度
	if blockHeight > 0 && trx.BlockHeight == 0 {
		trx.BlockHeight = blockHeight
		///		trx.BlockHash = blockHash
	}

	bs.extractTransaction(trx, &result, bs.ScanTargetFuncV2)

	return result

}

// 从最小单位的 amount 转为带小数点的表示
func convertToAmount(amount uint64) string {
	amountStr := fmt.Sprintf("%d", amount)
	d, _ := decimal.NewFromString(amountStr)
	w, _ := decimal.NewFromString("1000000")
	d = d.Div(w)
	return d.String()
}

// amount 字符串转为最小单位的表示
func convertFromAmount(amountStr string) uint64 {
	d, _ := decimal.NewFromString(amountStr)
	w, _ := decimal.NewFromString("1000000")
	d = d.Mul(w)
	r, _ := strconv.ParseInt(d.String(), 10, 64)
	return uint64(r)
}

//ExtractTransactionData 提取交易单
func (bs *ATOMBlockScanner) extractTransaction(trx *Transaction, result *ExtractResult, scanAddressFunc openwallet.BlockScanTargetFuncV2) {
	var (
		success    = true
		multiindex = uint64(0)
	)
	createAt := time.Now().Unix()
	if trx == nil {
		//记录哪个区块哪个交易单没有完成扫描
		success = false
	} else {

		if success && trx.TxValue != nil {
			isReceived := false
			inputindex := 0
			blockhash, _ := bs.wm.RestClient.getBlockHash(trx.BlockHeight)
			from := ""
			to := ""
			fromArray := []string{}
			toArray := []string{}
			amountCount := uint64(0)
			fee := "0"

			status := "1"
			reason := ""

			for i, tx := range trx.TxValue {
				//	if tx.Status == "true" {

				if tx.Status != "true" {
					status = "0"
					reason = tx.Reason
				}

				from = tx.From
				targetResult := scanAddressFunc(openwallet.ScanTargetParam {
					ScanTarget:     from,
					Symbol:         bs.wm.Symbol(),
					ScanTargetType: openwallet.ScanTargetTypeAccountAddress,
				})

				if targetResult.Exist {
					input := openwallet.TxInput{}
					input.TxID = trx.TxID
					input.Address = from
					input.Amount = convertToAmount(tx.Amount)
					amountCount += tx.Amount
					fromArray = append(fromArray, from+":"+input.Amount)
					toArray = append(toArray, tx.To+":"+input.Amount)
					input.Coin = openwallet.Coin{
						Symbol:     bs.wm.Symbol(),
						IsContract: false,
					}
					input.Index = uint64(inputindex)
					inputindex++
					input.Sid = openwallet.GenTxInputSID(trx.TxID, bs.wm.Symbol(), "", input.Index)
					input.CreateAt = createAt
					input.BlockHeight = trx.BlockHeight
					input.BlockHash = blockhash
					input.IsMemo = true
					input.Memo = trx.Memo
					ed := result.extractData[targetResult.SourceKey]
					if ed == nil {
						ed = openwallet.NewBlockExtractData()
						result.extractData[targetResult.SourceKey] = ed
					}
					ed.TxInputs = append(ed.TxInputs, &input)

					if trx.Fee != nil && trx.Fee[0].Amount != 0 {
						tmp := *&input
						feeCharge := &tmp
						feeCharge.Amount = convertToAmount(trx.Fee[i].Amount)
						fee = feeCharge.Amount
						fee = feeCharge.Amount
						feeCharge.Index = uint64(inputindex)
						inputindex++
						feeCharge.Sid = openwallet.GenTxInputSID(trx.TxID, bs.wm.Symbol(), "", feeCharge.Index)
						ed.TxInputs = append(ed.TxInputs, feeCharge)
					}
				}

				to = tx.To
				targetResult = scanAddressFunc(openwallet.ScanTargetParam {
					ScanTarget:     to,
					Symbol:         bs.wm.Symbol(),
					ScanTargetType: openwallet.ScanTargetTypeAccountAddress,
				})


				if targetResult.Exist {
					isReceived = true
					output := openwallet.TxOutPut{}
					output.Received = true
					output.TxID = trx.TxID
					output.Address = to
					output.Amount = convertToAmount(tx.Amount)
					output.IsMemo = true
					output.Memo = trx.Memo

					notified := false

					for _, v := range fromArray {
						if v == tx.From+":"+output.Amount {
							notified = true
						}
					}
					if !notified {
						fromArray = append(fromArray, tx.From+":"+output.Amount)
					}
					notified = false
					for _, v := range toArray {
						if v == to+":"+output.Amount {
							notified = true
						}
					}
					if !notified {
						toArray = append(toArray, to+":"+output.Amount)
					}

					amountCount += tx.Amount
					output.Coin = openwallet.Coin{
						Symbol:     bs.wm.Symbol(),
						IsContract: false,
					}
					if tx.From == "multiaddress" {
						output.Index = multiindex
						multiindex++
					} else {
						output.Index = uint64(i)
					}

					output.Sid = openwallet.GenTxOutPutSID(trx.TxID, bs.wm.Symbol(), "", output.Index)
					output.CreateAt = createAt
					output.BlockHeight = trx.BlockHeight
					output.BlockHash = blockhash
					ed := result.extractData[targetResult.SourceKey]
					if ed == nil {
						ed = openwallet.NewBlockExtractData()
						result.extractData[targetResult.SourceKey] = ed
					}

					ed.TxOutputs = append(ed.TxOutputs, &output)
				}
				//	}
			}

			for _, extractData := range result.extractData {
				// status := "1"
				// reason := ""

				tx := &openwallet.Transaction{
					From:   fromArray,
					To:     toArray,
					Amount: convertToAmount(amountCount),
					Fees:   fee,
					Coin: openwallet.Coin{
						Symbol:     bs.wm.Symbol(),
						IsContract: false,
					},
					BlockHash:   blockhash,
					BlockHeight: trx.BlockHeight,
					TxID:        trx.TxID,
					Decimal:     6,
					Status:      status,
					Reason:      reason,
					SubmitTime:  int64(trx.TimeStamp),
					ConfirmTime: int64(trx.TimeStamp),
					IsMemo:      true,
					Memo:        trx.Memo,
					Received:    isReceived,
					TxType:      0,
				}
				if trx.Memo != "" {
					tx.SetExtParam("memo", trx.Memo)
				}
				wxID := openwallet.GenTransactionWxID(tx)
				tx.WxID = wxID
				extractData.Transaction = tx
			}

		}

		success = true

	}
	result.Success = success
}

//newExtractDataNotify 发送通知
func (bs *ATOMBlockScanner) newExtractDataNotify(height uint64, extractData map[string]*openwallet.TxExtractData) error {

	for o, _ := range bs.Observers {
		for key, data := range extractData {
			err := o.BlockExtractDataNotify(key, data)
			if err != nil {
				bs.wm.Log.Error("BlockExtractDataNotify unexpected error:", err)
				//记录未扫区块
				unscanRecord := openwallet.NewUnscanRecord(height, "", "ExtractData Notify failed.", bs.wm.Symbol())
				err = bs.SaveUnscanRecord(unscanRecord)
				if err != nil {
					bs.wm.Log.Std.Error("block height: %d, save unscan record failed. unexpected error: %v", height, err.Error())
				}

			}
		}
	}

	return nil
}

//DeleteUnscanRecordNotFindTX 删除未没有找到交易记录的重扫记录
func (bs *ATOMBlockScanner) DeleteUnscanRecordNotFindTX() error {

	//删除找不到交易单
	reason := "[-5]No information available about transaction"

	if bs.BlockchainDAI == nil {
		return fmt.Errorf("Blockchain DAI is not setup ")
	}

	list, err := bs.BlockchainDAI.GetUnscanRecords(bs.wm.Symbol())
	if err != nil {
		return err
	}

	for _, r := range list {
		if strings.HasPrefix(r.Reason, reason) {
			bs.BlockchainDAI.DeleteUnscanRecordByID(r.ID, bs.wm.Symbol())
		}
	}
	return nil
}

//GetCurrentBlockHeader 获取全网最新高度区块头
func (bs *ATOMBlockScanner) GetCurrentBlockHeader() (*openwallet.BlockHeader, error) {
	var (
		blockHeight uint64 = 0
		err         error
	)

	blockHeight, err = bs.wm.GetBlockHeight()
	if err != nil {
		return nil, err
	}

	block, err := bs.wm.RestClient.getBlockByHeight(blockHeight)
	if err != nil {
		bs.wm.Log.Errorf("get block spec by block number failed, err=%v", err)
		return nil, err
	}

	return &openwallet.BlockHeader{Height: blockHeight, Hash: block.Hash}, nil
}

//GetScannedBlockHeader 获取已扫高度区块头
func (bs *ATOMBlockScanner) GetScannedBlockHeader() (*openwallet.BlockHeader, error) {

	var (
		blockHeight uint64 = 0
		block       *Block
		err         error
	)

	blockHeight, _, err = bs.wm.Blockscanner.GetLocalNewBlock()
	if err != nil {
		bs.wm.Log.Errorf("get local new block failed, err=%v", err)
		return nil, err
	}

	//如果本地没有记录，查询接口的高度
	if blockHeight == 0 {
		blockHeight, err = bs.wm.GetBlockHeight()
		if err != nil {
			bs.wm.Log.Errorf("ATOM GetBlockHeight failed,err = %v", err)
			return nil, err
		}

		//就上一个区块链为当前区块
		blockHeight = blockHeight - 1

	}
	block, err = bs.wm.RestClient.getBlockByHeight(blockHeight)
	if err != nil {
		bs.wm.Log.Errorf("get block spec by block number failed, err=%v", err)
		return nil, err
	}

	return block.BlockHeader(), nil
}

//GetScannedBlockHeight 获取已扫区块高度
func (bs *ATOMBlockScanner) GetScannedBlockHeight() uint64 {
	localHeight, _, _ := bs.wm.Blockscanner.GetLocalNewBlock()
	return localHeight
}

func (bs *ATOMBlockScanner) ExtractTransactionData(txid string, scanTargetFunc openwallet.BlockScanTargetFunc) (map[string][]*openwallet.TxExtractData, error) {

	scanAddressFunc := func(t openwallet.ScanTarget) (string, bool) {
		sourceKey, ok := scanTargetFunc(openwallet.ScanTarget{
			Address:          t.Address,
			Symbol:           bs.wm.Symbol(),
			BalanceModelType: bs.wm.BalanceModelType(),
		})
		return sourceKey, ok
	}
	result := bs.ExtractTransaction(0, "", txid, scanAddressFunc, false)
	if !result.Success {
		return nil, fmt.Errorf("extract transaction failed")
	}
	extData := make(map[string][]*openwallet.TxExtractData)
	for key, data := range result.extractData {
		txs := extData[key]
		if txs == nil {
			txs = make([]*openwallet.TxExtractData, 0)
		}
		txs = append(txs, data)
		extData[key] = txs
	}
	return extData, nil
}

//GetSourceKeyByAddress 获取地址对应的数据源标识
func (bs *ATOMBlockScanner) GetSourceKeyByAddress(address string) (string, bool) {
	bs.Mu.RLock()
	defer bs.Mu.RUnlock()

	sourceKey, ok := bs.AddressInScanning[address]
	return sourceKey, ok
}

//GetBlockHeight 获取区块链高度
func (wm *WalletManager) GetBlockHeight() (uint64, error) {
	return wm.RestClient.getBlockHeight()
}

//GetLocalNewBlock 获取本地记录的区块高度和hash
func (bs *ATOMBlockScanner) GetLocalNewBlock() (uint64, string, error) {

	if bs.BlockchainDAI == nil {
		return 0, "", fmt.Errorf("Blockchain DAI is not setup ")
	}

	header, err := bs.BlockchainDAI.GetCurrentBlockHead(bs.wm.Symbol())
	if err != nil {
		return 0, "", err
	}

	return header.Height, header.Hash, nil
}

//SaveLocalNewBlock 记录区块高度和hash到本地
func (bs *ATOMBlockScanner) SaveLocalNewBlock(blockHeight uint64, blockHash string) error {

	if bs.BlockchainDAI == nil {
		return fmt.Errorf("Blockchain DAI is not setup ")
	}

	header := &openwallet.BlockHeader{
		Hash:   blockHash,
		Height: blockHeight,
		Fork:   false,
		Symbol: bs.wm.Symbol(),
	}

	return bs.BlockchainDAI.SaveCurrentBlockHead(header)
}

//GetBlockHash 根据区块高度获得区块hash
func (wm *WalletManager) GetBlockHash(height uint64) (string, error) {
	return wm.RestClient.getBlockHash(height)
}

//GetBlock 获取区块数据
func (wm *WalletManager) GetBlock(hash string) (*Block, error) {
	return nil, errors.New("get block by hash is not supported right now!")
}

//GetTxIDsInMemPool 获取待处理的交易池中的交易单IDs
func (wm *WalletManager) GetTxIDsInMemPool() ([]string, error) {
	txids := make([]string, 0)
	path := "/unconfirmed_txs?limit=1000"
	trans, err := wm.NodeClient.Call(path, nil, "GET")
	if err != nil {
		return nil, err
	}
	for _, tran := range trans.Get("result").Get("txs").Array() {
		transBytes, _ := base64.StdEncoding.DecodeString(tran.String())
		txid := owcrypt.Hash(transBytes, 0, owcrypt.HASH_ALG_SHA256)
		txids = append(txids, hex.EncodeToString(txid))
	}
	return txids, nil

}

func (wm *WalletManager) GetTransactionInMemPool(txid string) (*Transaction, error) {
	path := "/txs/" + txid
	trans, err := wm.RestClient.Call(path, nil, "GET")

	if err != nil {
		return nil, err
	}
	return NewTransaction(trans, wm.Config.TxType, wm.Config.MsgType, wm.Config.Denom), nil
}

//GetTransaction 获取交易单
func (wm *WalletManager) GetTransaction(txid string) (*Transaction, error) {

	path := "/cosmos/tx/v1beta1/txs/" + txid
	trans, err := wm.RestClient.Call(path, nil, "GET")

	if err != nil {
		return nil, err
	}

	ret := NewTransaction(trans, wm.Config.TxType, wm.Config.MsgType, wm.Config.Denom)
	return ret, nil
}

//GetAssetsAccountBalanceByAddress 查询账户相关地址的交易记录
func (bs *ATOMBlockScanner) GetBalanceByAddress(address ...string) ([]*openwallet.Balance, error) {

	addrsBalance := make([]*openwallet.Balance, 0)

	for _, addr := range address {
		balance, err := bs.wm.RestClient.getBalance(addr, bs.wm.Config.Denom)
		if err != nil {
			return nil, err
		}

		addrsBalance = append(addrsBalance, &openwallet.Balance{
			Symbol:  bs.wm.Symbol(),
			Address: addr,
			Balance: convertToAmount(uint64(balance.Balance.Int64())),
		})
	}

	return addrsBalance, nil
}

func (c *Client) getMultiAddrTransactions(txType, msgType, denom string, offset, limit int, addresses ...string) ([]*Transaction, error) {
	var (
		trxs = make([]*Transaction, 0)
	)

	for _, addr := range addresses {
		path := "/txs/" + addr

		resp, err := c.Call(path, nil, "GET")
		if err != nil {
			return nil, err
		}
		txArray := resp.Array()[0].Array()

		for _, txDetail := range txArray {
			trxs = append(trxs, NewTransaction(&txDetail, txType, msgType, denom))
		}
	}

	return trxs, nil
}

//GetAssetsAccountTransactionsByAddress 查询账户相关地址的交易记录
func (bs *ATOMBlockScanner) GetTransactionsByAddress(offset, limit int, coin openwallet.Coin, address ...string) ([]*openwallet.TxExtractData, error) {

	//var (
	//	array = make([]*openwallet.TxExtractData, 0)
	//)
	//
	//trxs, err := bs.wm.RestClient.getMultiAddrTransactions(bs.wm.Config.TxType, bs.wm.Config.MsgType, bs.wm.Config.Denom, offset, limit, address...)
	//if err != nil {
	//	return nil, err
	//}
	//
	//key := "account"
	//
	////提取账户相关的交易单
	//var scanAddressFunc openwallet.BlockScanAddressFunc = func(findAddr string) (string, bool) {
	//	for _, a := range address {
	//		if findAddr == a {
	//			return key, true
	//		}
	//	}
	//	return "", false
	//}
	//
	////要检查一下tx.BlockHeight是否有值
	//
	//for _, tx := range trxs {
	//
	//	result := ExtractResult{
	//		BlockHeight: tx.BlockHeight,
	//		TxID:        tx.TxID,
	//		extractData: make(map[string]*openwallet.TxExtractData),
	//		Success:     true,
	//	}
	//
	//	bs.extractTransaction(tx, &result, bs.ScanTargetFuncV2)
	//	data := result.extractData
	//	txExtract := data[key]
	//	if txExtract != nil {
	//		array = append(array, txExtract)
	//	}
	//
	//}
	//
	//return array, nil
	return nil, nil
}

//Run 运行
func (bs *ATOMBlockScanner) Run() error {

	bs.BlockScannerBase.Run()

	return nil
}

////Stop 停止扫描
func (bs *ATOMBlockScanner) Stop() error {

	bs.BlockScannerBase.Stop()

	return nil
}

//Pause 暂停扫描
func (bs *ATOMBlockScanner) Pause() error {

	bs.BlockScannerBase.Pause()

	return nil
}

//Restart 继续扫描
func (bs *ATOMBlockScanner) Restart() error {

	bs.BlockScannerBase.Restart()

	return nil
}

/******************* 使用insight socket.io 监听区块 *******************/

//setupSocketIO 配置socketIO监听新区块
func (bs *ATOMBlockScanner) setupSocketIO() error {

	bs.wm.Log.Info("block scanner use socketIO to listen new data")

	var (
		room = "inv"
	)

	if bs.socketIO == nil {

		apiUrl, err := url.Parse(bs.wm.Config.ServerAPI)
		if err != nil {
			return err
		}
		domain := apiUrl.Hostname()
		port := common.NewString(apiUrl.Port()).Int()
		c, err := gosocketio.Dial(
			gosocketio.GetUrl(domain, port, false),
			transport.GetDefaultWebsocketTransport())
		if err != nil {
			return err
		}

		bs.socketIO = c

	}

	err := bs.socketIO.On("tx", func(h *gosocketio.Channel, args interface{}) {
		//bs.wm.Log.Info("block scanner socketIO get new transaction received: ", args)
		txMap, ok := args.(map[string]interface{})
		if ok {
			txid := txMap["txid"].(string)
			errInner := bs.BatchExtractTransaction(0, "", []string{txid}, false)
			if errInner != nil {
				bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", errInner)
			}
		}

	})
	if err != nil {
		return err
	}

	/*
		err = bs.socketIO.On("block", func(h *gosocketio.Channel, args interface{}) {
			bs.wm.Log.Info("block scanner socketIO get new block received: ", args)
			hash, ok := args.(string)
			if ok {

				block, errInner := bs.wm.GetBlock(hash)
				if errInner != nil {
					bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", errInner)
				}

				errInner = bs.scanBlock(block)
				if errInner != nil {
					bs.wm.Log.Std.Info("block scanner can not block: %d; unexpected error: %v", block.Height, errInner)
				}
			}

		})
		if err != nil {
			return err
		}
	*/

	err = bs.socketIO.On(gosocketio.OnDisconnection, func(h *gosocketio.Channel) {
		bs.wm.Log.Info("block scanner socketIO disconnected")
	})
	if err != nil {
		return err
	}

	err = bs.socketIO.On(gosocketio.OnConnection, func(h *gosocketio.Channel) {
		bs.wm.Log.Info("block scanner socketIO connected")
		h.Emit("subscribe", room)
	})
	if err != nil {
		return err
	}

	return nil
}

//SupportBlockchainDAI 支持外部设置区块链数据访问接口
//@optional
func (bs *ATOMBlockScanner) SupportBlockchainDAI() bool {
	return true
}
