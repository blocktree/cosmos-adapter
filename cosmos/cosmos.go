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
	"errors"
	"fmt"
	"path/filepath"

	"github.com/astaxie/beego/config"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openwallet"
)

//初始化配置流程
func (wm *WalletManager) InitConfigFlow() error {

	wm.Config.InitConfig()
	file := filepath.Join(wm.Config.configFilePath, wm.Config.configFileName)
	fmt.Printf("You can run 'vim %s' to edit wallet's Config.\n", file)
	return nil
}

//查看配置信息
func (wm *WalletManager) ShowConfig() error {
	return wm.Config.PrintConfig()
}

//创建钱包流程
func (wm *WalletManager) CreateWalletFlow() error {

	// var (
	// 	password string
	// 	name     string
	// 	err      error
	// 	keyFile  string
	// )

	// //先加载是否有配置文件
	// err = wm.LoadConfig()
	// if err != nil {
	// 	return err
	// }

	// // 等待用户输入钱包名字
	// name, err = console.InputText("Enter wallet's name: ", true)

	// // 等待用户输入密码
	// password, err = console.InputPassword(false, 3)

	// _, keyFile, err = wm.CreateNewWallet(name, password)
	// if err != nil {
	// 	return err
	// }

	// fmt.Printf("\n")
	// fmt.Printf("Wallet create successfully, key path: %s\n", keyFile)

	return nil

}

//创建地址流程
func (wm *WalletManager) CreateAddressFlow() error {

	// //先加载是否有配置文件
	// err := wm.LoadConfig()
	// if err != nil {
	// 	return err
	// }

	// //查询所有钱包信息
	// wallets, err := wm.GetWallets()
	// if err != nil {
	// 	fmt.Printf("The node did not create any wallet!\n")
	// 	return err
	// }

	// //打印钱包
	// wm.printWalletList(wallets)

	// fmt.Printf("[Please select a wallet account to create address] \n")

	// //选择钱包
	// num, err := console.InputNumber("Enter wallet number: ", true)
	// if err != nil {
	// 	return err
	// }

	// if int(num) >= len(wallets) {
	// 	return errors.New("Input number is out of index! ")
	// }

	// account := wallets[num]

	// // 输入地址数量
	// count, err := console.InputNumber("Enter the number of addresses you want: ", false)
	// if err != nil {
	// 	return err
	// }

	// if count > maxAddresNum {
	// 	return errors.New(fmt.Sprintf("The number of addresses can not exceed %d", maxAddresNum))
	// }

	// //输入密码
	// password, err := console.InputPassword(false, 3)

	// log.Std.Info("Start batch creation")
	// log.Std.Info("-------------------------------------------------")

	// filePath, _, err := wm.CreateBatchAddress(account.WalletID, password, count)
	// if err != nil {
	// 	return err
	// }

	// log.Std.Info("-------------------------------------------------")
	// log.Std.Info("All addresses have created, file path:%s", filePath)

	return nil
}

//汇总钱包流程

/*

  汇总执行流程：
  1. 执行启动汇总某个币种命令。
  2. 列出该币种的全部可用钱包信息。
  3. 输入需要汇总的钱包序号数组（以,号分隔）。
  4. 输入每个汇总钱包的密码，完成汇总登记。
  5. 工具启动定时器监听钱包，并输出日志到log文件夹。
  6. 待已登记的汇总钱包达到阀值，发起账户汇总到配置下的地址。

*/

// SummaryFollow 汇总流程
func (wm *WalletManager) SummaryFollow() error {

	// var (
	// 	endRunning = make(chan bool, 1)
	// )

	// //先加载是否有配置文件
	// err := wm.LoadConfig()
	// if err != nil {
	// 	return err
	// }

	// //判断汇总地址是否存在
	// if len(wm.Config.SumAddress) == 0 {

	// 	return errors.New(fmt.Sprintf("Summary address is not set. Please set it in './conf/%s.ini' \n", Symbol))
	// }

	// if wm.Config.CycleSeconds == 0 {
	// 	wm.Config.CycleSeconds = 30 * 1000
	// }

	// //查询所有钱包信息
	// wallets, err := wm.GetWallets()
	// if err != nil {
	// 	fmt.Printf("The node did not create any wallet!\n")
	// 	return err
	// }

	// //打印钱包
	// wm.printWalletList(wallets)

	// fmt.Printf("[Please select the wallet to summary, and enter the numbers split by ','." +
	// 	" For example: 0,1,2,3] \n")

	// // 等待用户输入钱包名字
	// nums, err := console.InputText("Enter the No. group: ", true)
	// if err != nil {
	// 	return err
	// }

	// //分隔数组
	// array := strings.Split(nums, ",")

	// for _, numIput := range array {
	// 	if common.IsNumberString(numIput) {
	// 		numInt := common.NewString(numIput).Int()
	// 		if numInt < len(wallets) {
	// 			w := wallets[numInt]

	// 			fmt.Printf("Register summary wallet [%s]-[%s]\n", w.Alias, w.WalletID)
	// 			//输入钱包密码完成登记
	// 			password, err := console.InputPassword(false, 3)
	// 			if err != nil {
	// 				return err
	// 			}

	// 			//解锁钱包验证密码
	// 			_, err = w.HDKey(password)
	// 			if err != nil {
	// 				openwLogger.Log.Errorf("The password to unlock wallet is incorrect! ")
	// 				continue
	// 			}

	// 			//解锁钱包
	// 			err = wm.UnlockWallet(password, 1)
	// 			if err != nil {
	// 				openwLogger.Log.Errorf("The password to unlock wallet is incorrect! ")
	// 				continue
	// 			}

	// 			w.Password = password

	// 			wm.AddWalletInSummary(w.WalletID, w)
	// 		} else {
	// 			return errors.New("The input No. out of index! ")
	// 		}
	// 	} else {
	// 		return errors.New("The input No. is not numeric! ")
	// 	}
	// }

	// if len(wm.WalletsInSum) == 0 {
	// 	return errors.New("Not summary wallets to register! ")
	// }

	// fmt.Printf("The timer for summary has started. Execute by every %v seconds.\n", wm.Config.CycleSeconds.Seconds())

	// //启动钱包汇总程序
	// sumTimer := timer.NewTask(wm.Config.CycleSeconds, wm.SummaryWallets)
	// sumTimer.Start()

	// <-endRunning

	return nil
}

//备份钱包流程
func (wm *WalletManager) BackupWalletFlow() error {

	// var (
	// 	err        error
	// 	backupPath string
	// )

	// //先加载是否有配置文件
	// err = wm.LoadConfig()
	// if err != nil {
	// 	return err
	// }

	// list, err := wm.GetWallets()
	// if err != nil {
	// 	return err
	// }

	// //打印钱包列表
	// wm.printWalletList(list)

	// fmt.Printf("[Please select a wallet to backup] \n")

	// //选择钱包
	// num, err := console.InputNumber("Enter wallet No. : ", true)
	// if err != nil {
	// 	return err
	// }

	// if int(num) >= len(list) {
	// 	return errors.New("Input number is out of index! ")
	// }

	// wallet := list[num]

	// backupPath, err = wm.BackupWallet(wallet.WalletID)
	// if err != nil {
	// 	return err
	// }

	// //输出备份导出目录
	// fmt.Printf("Wallet backup file path: %s\n", backupPath)

	return nil

}

//SendTXFlow 发送交易
func (wm *WalletManager) TransferFlow() error {

	// //先加载是否有配置文件
	// err := wm.LoadConfig()
	// if err != nil {
	// 	return err
	// }

	// list, err := wm.GetWallets()
	// if err != nil {
	// 	return err
	// }

	// //打印钱包列表
	// wm.printWalletList(list)

	// fmt.Printf("[Please select a wallet to send transaction] \n")

	// //选择钱包
	// num, err := console.InputNumber("Enter wallet No. : ", true)
	// if err != nil {
	// 	return err
	// }

	// if int(num) >= len(list) {
	// 	return errors.New("Input number is out of index! ")
	// }

	// wallet := list[num]

	// // 等待用户输入发送数量
	// amount, err := console.InputRealNumber("Enter amount to send: ", true)
	// if err != nil {
	// 	return err
	// }

	// atculAmount, _ := decimal.NewFromString(amount)
	// balance, _ := decimal.NewFromString(wm.GetWalletBalance(wallet.WalletID))

	// if atculAmount.GreaterThan(balance) {
	// 	return errors.New("Input amount is greater than balance! ")
	// }

	// // 等待用户输入发送数量
	// receiver, err := console.InputText("Enter receiver address: ", true)
	// if err != nil {
	// 	return err
	// }

	// //输入密码解锁钱包
	// password, err := console.InputPassword(false, 3)
	// if err != nil {
	// 	return err
	// }

	// //重新加载utxo
	// wm.RebuildWalletUnspent(wallet.WalletID)

	// //建立交易单
	// txID, err := wm.SendTransaction(wallet.WalletID,
	// 	receiver, atculAmount, password, true)
	// if err != nil {
	// 	return err
	// }

	// fmt.Printf("Send transaction successfully, TXID：%s\n", txID)

	return nil
}

//GetWalletList 获取钱包列表
func (wm *WalletManager) GetWalletList() error {

	// //先加载是否有配置文件
	// err := wm.LoadConfig()
	// if err != nil {
	// 	return err
	// }

	// list, err := wm.GetWallets()

	// if len(list) == 0 {
	// 	log.Std.Info("No any wallets have created.")
	// 	return nil
	// }

	// //打印钱包列表
	// wm.printWalletList(list)

	return nil
}

//RestoreWalletFlow 恢复钱包流程
func (wm *WalletManager) RestoreWalletFlow() error {

	// var (
	// 	err      error
	// 	keyFile  string
	// 	dbFile   string
	// 	datFile  string
	// 	password string
	// )

	// //先加载是否有配置文件
	// err = wm.LoadConfig()
	// if err != nil {
	// 	return err
	// }

	// //输入恢复文件路径
	// keyFile, err = console.InputText("Enter backup key file path: ", true)
	// if err != nil {
	// 	return err
	// }

	// dbFile, err = console.InputText("Enter backup db file path: ", true)
	// if err != nil {
	// 	return err
	// }

	// datFile, err = console.InputText("Enter backup wallet.db file path: ", true)
	// if err != nil {
	// 	return err
	// }

	// password, err = console.InputPassword(false, 3)
	// if err != nil {
	// 	return err
	// }

	// fmt.Printf("Wallet restoring, please wait a moment...\n")
	// err = wm.RestoreWallet(keyFile, dbFile, datFile, password)
	// if err != nil {
	// 	return err
	// }

	// //输出备份导出目录
	// fmt.Printf("Restore wallet successfully.\n")

	return nil

}

//InstallNode 安装节点
func (wm *WalletManager) InstallNodeFlow() error {
	return errors.New("Install node is unsupport now. ")
}

//InitNodeConfig 初始化节点配置文件
func (wm *WalletManager) InitNodeConfigFlow() error {
	return errors.New("Install node is unsupport now. ")
}

//StartNodeFlow 开启节点
func (wm *WalletManager) StartNodeFlow() error {

	//return wm.startNode()
	return nil
}

//StopNodeFlow 关闭节点
func (wm *WalletManager) StopNodeFlow() error {

	//return wm.stopNode()
	return nil
}

//RestartNodeFlow 重启节点
func (wm *WalletManager) RestartNodeFlow() error {
	return errors.New("Install node is unsupport now. ")
}

//ShowNodeInfo 显示节点信息
func (wm *WalletManager) ShowNodeInfo() error {
	return errors.New("Install node is unsupport now. ")
}

//SetConfigFlow 初始化配置流程
func (wm *WalletManager) SetConfigFlow(subCmd string) error {
	file := wm.Config.configFilePath + wm.Config.configFileName
	fmt.Printf("You can run 'vim %s' to edit %s Config.\n", file, subCmd)
	return nil
}

//ShowConfigInfo 查看配置信息
func (wm *WalletManager) ShowConfigInfo(subCmd string) error {
	wm.Config.PrintConfig()
	return nil
}

//CurveType 曲线类型
func (wm *WalletManager) CurveType() uint32 {
	return wm.Config.CurveType
}

//FullName 币种全名
func (wm *WalletManager) FullName() string {
	return "cosmos"
}

//Symbol 币种标识
func (wm *WalletManager) Symbol() string {
	return wm.Config.Symbol
}

//小数位精度
func (wm *WalletManager) Decimal() int32 {
	return 6
}

func (wm *WalletManager) BalanceModelType() openwallet.BalanceModelType {
	return openwallet.BalanceModelTypeAddress
}

//AddressDecode 地址解析器
func (wm *WalletManager) GetAddressDecoderV2() openwallet.AddressDecoderV2 {
	return wm.Decoder
}

//TransactionDecoder 交易单解析器
func (wm *WalletManager) GetTransactionDecoder() openwallet.TransactionDecoder {
	return wm.TxDecoder
}

//GetBlockScanner 获取区块链
func (wm *WalletManager) GetBlockScanner() openwallet.BlockScanner {

	//先加载是否有配置文件
	//err := wm.LoadConfig()
	//if err != nil {
	//	return nil
	//}

	return wm.Blockscanner
}

//ImportWatchOnlyAddress 导入观测地址
func (wm *WalletManager) ImportWatchOnlyAddress(address ...*openwallet.Address) error {

	// //先加载是否有配置文件
	// //err := wm.LoadConfig()
	// //if err != nil {
	// //	return nil
	// //}

	// var (
	// 	failedIndex = make([]int, 0)
	// )

	// //for i, a := range address {
	// //	log.Debug("start ImportAddress")
	// //	err = wm.ImportAddress(a)
	// //	if err != nil {
	// //		log.Error("import address failed, unexpected error:", err)
	// //		failedIndex = append(failedIndex, i)
	// //	}
	// //	log.Debug("end ImportAddress")
	// //}

	// if len(wm.Config.WalletPassword) > 0 {
	// 	wm.UnlockWallet(wm.Config.WalletPassword, 600)
	// }

	// failedIndex, err := wm.ImportMulti(address, nil, true)
	// if err != nil {
	// 	return err
	// }

	// if len(failedIndex) > 0 {
	// 	failedReason := ""

	// 	for _, index := range failedIndex {
	// 		failedReason = failedReason + address[index].Address + ", "
	// 	}

	// 	failedReason = failedReason + "import failed"

	// 	return fmt.Errorf(failedReason)
	// }

	return nil
}

//GetAddressWithBalance
func (wm *WalletManager) GetAddressWithBalance(address ...*openwallet.Address) error {

	// var (
	// 	addressMap  = make(map[string]*openwallet.Address)
	// 	searchAddrs = make([]string, 0)
	// )

	// //先加载是否有配置文件
	// //err := wm.LoadConfig()
	// //if err != nil {
	// //	return err
	// //}

	// for _, address := range address {
	// 	searchAddrs = append(searchAddrs, address.Address)
	// 	addressMap[address.Address] = address
	// }

	// //查找核心钱包确认数大于0的
	// utxos, err := wm.ListUnspent(0, searchAddrs...)
	// if err != nil {
	// 	return err
	// }
	// log.Debug(utxos)
	// //balanceDel := decimal.New(0, 0)

	// //批量插入到本地数据库
	// //设置utxo的钱包账户
	// for _, utxo := range utxos {
	// 	a := addressMap[utxo.Address]
	// 	a.Balance = utxo.Amount

	// }

	return nil

}

//LoadAssetsConfig 加载外部配置
func (wm *WalletManager) LoadAssetsConfig(c config.Configer) error {
	wm.Config.IsTestNet, _ = c.Bool("isTestNet")

	if wm.Config.IsTestNet {
		wm.Config.RestAPI = c.String("testnetRestAPI")
		wm.Config.ChainID = c.String("testnetChainID")
		wm.Config.Denom = c.String("testnetDenom")
		wm.Config.NodeAPI = c.String("testnetNodeAPI")

	} else {
		wm.Config.RestAPI = c.String("mainnetRestAPI")
		wm.Config.ChainID = c.String("mainnetChainID")
		wm.Config.Denom = c.String("mainnetDenom")
		wm.Config.NodeAPI = c.String("mainnetNodeAPI")
	}

	wm.RestClient = NewClient(wm.Config.RestAPI, false)
	wm.NodeClient = NewClient(wm.Config.NodeAPI, false)

	wm.Config.TxType = c.String("txType")
	msgType, _ := c.Int("msgType")
	switch msgType {
	case 1:
		wm.Config.MsgType = c.String("msgSend")
	case 2:
		wm.Config.MsgType = c.String("msgVote")
	case 3:
		wm.Config.MsgType = c.String("msgDelegate")
	}

	wm.Config.PayFee, _ = c.Bool("payFee")
	minFee, _ := c.Int("minFee")
	wm.Config.MinFee = uint64(minFee)
	stdGas, _ := c.Int("stdGas")
	wm.Config.StdGas = uint64(stdGas)
	wm.Config.IsScanMemPool, _ = c.Bool("isScanMemPool")
	wm.Config.DataDir = c.String("dataDir")

	//数据文件夹
	wm.Config.makeDataDir()
	return nil
}

//InitAssetsConfig 初始化默认配置
func (wm *WalletManager) InitAssetsConfig() (config.Configer, error) {
	return config.NewConfigData("ini", []byte(wm.Config.DefaultConfig))
}

//GetAssetsLogger 获取资产账户日志工具
func (wm *WalletManager) GetAssetsLogger() *log.OWLogger {
	//return nil
	return wm.Log
}

// GetSmartContractDecoder 获取智能合约解析器
func (wm *WalletManager) GetSmartContractDecoder() openwallet.SmartContractDecoder {
	return wm.ContractDecoder
}
