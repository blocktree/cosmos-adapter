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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/blocktree/go-owcdrivers/cosmosTransaction"
	owcrypt "github.com/blocktree/go-owcrypt"
	ow "github.com/blocktree/openwallet/v2/common"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/tidwall/gjson"
)

type TransactionDecoder struct {
	openwallet.TransactionDecoderBase
	wm *WalletManager //钱包管理者
}

//NewTransactionDecoder 交易单解析器
func NewTransactionDecoder(wm *WalletManager) *TransactionDecoder {
	decoder := TransactionDecoder{}
	decoder.wm = wm
	return &decoder
}

//CreateRawTransaction 创建交易单
func (decoder *TransactionDecoder) CreateRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {
	return decoder.CreateATOMRawTransaction(wrapper, rawTx)
}

//SignRawTransaction 签名交易单
func (decoder *TransactionDecoder) SignRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {
	return decoder.SignATOMRawTransaction(wrapper, rawTx)
}

//VerifyRawTransaction 验证交易单，验证交易单并返回加入签名后的交易单
func (decoder *TransactionDecoder) VerifyRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {
	return decoder.VerifyATOMRawTransaction(wrapper, rawTx)
}

func (decoder *TransactionDecoder) SubmitRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) (*openwallet.Transaction, error) {
	if len(rawTx.RawHex) == 0 {
		return nil, fmt.Errorf("transaction hex is empty")
	}

	if !rawTx.IsCompleted {
		return nil, fmt.Errorf("transaction is not completed validation")
	}

	txid, err := decoder.wm.SendRawTransaction(rawTx.RawHex)
	if err != nil {
		fmt.Println("Tx to send: ", rawTx.RawHex)
		return nil, err
	} else {
		sequence := gjson.Get(rawTx.RawHex, "tx").Get("signatures").Array()[0].Get("sequence").Uint() + 1
		wrapper.SetAddressExtParam(gjson.Get(rawTx.RawHex, "tx").Get("msg").Array()[0].Get("value").Get("from_address").String(), decoder.wm.FullName(), sequence)
	}

	rawTx.TxID = txid
	rawTx.IsSubmit = true

	decimals := int32(8)

	tx := openwallet.Transaction{
		From:       rawTx.TxFrom,
		To:         rawTx.TxTo,
		Amount:     rawTx.TxAmount,
		Coin:       rawTx.Coin,
		TxID:       rawTx.TxID,
		Decimal:    decimals,
		AccountID:  rawTx.Account.AccountID,
		Fees:       rawTx.Fees,
		SubmitTime: time.Now().Unix(),
	}

	tx.WxID = openwallet.GenTransactionWxID(&tx)

	return &tx, nil
}

func (decoder *TransactionDecoder) CreateATOMRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	addresses, err := wrapper.GetAddressList(0, -1, "AccountID", rawTx.Account.AccountID)

	if err != nil {
		return err
	}

	if len(addresses) == 0 {
		return openwallet.Errorf(openwallet.ErrAccountNotAddress, "[%s] have not addresses", rawTx.Account.AccountID)
	}

	addressesBalanceList := make([]AddrBalance, 0, len(addresses))

	for i, addr := range addresses {
		balance, err := decoder.wm.RestClient.getBalance(addr.Address, decoder.wm.Config.Denom)

		if err != nil {
			return err
		}

		balance.index = i
		addressesBalanceList = append(addressesBalanceList, *balance)
	}

	sort.Slice(addressesBalanceList, func(i int, j int) bool {
		return addressesBalanceList[i].Balance.Cmp(addressesBalanceList[j].Balance) >= 0
	})

	fee := uint64(0)
	gas := decoder.wm.Config.StdGas
	if len(rawTx.FeeRate) > 0 {
		fee = convertFromAmount(rawTx.FeeRate)
	} else {
		if decoder.wm.Config.PayFee {
			fee = decoder.wm.Config.MinFee
		}
	}
	// fee := big.NewInt(int64(decoder.wm.Config.FeeCharge))

	var amountStr, to string
	for k, v := range rawTx.To {
		to = k
		amountStr = v
		break
	}
	// keySignList := make([]*openwallet.KeySignature, 1, 1)

	amount := big.NewInt(int64(convertFromAmount(amountStr)))
	amount = amount.Add(amount, big.NewInt(int64(fee)))
	from := ""
	count := big.NewInt(0)
	countList := []uint64{}
	for _, a := range addressesBalanceList {
		if a.Balance.Cmp(amount) < 0 {
			count.Add(count, a.Balance)
			if count.Cmp(amount) >= 0 {
				countList = append(countList, a.Balance.Sub(a.Balance, count.Sub(count, amount)).Uint64())
				log.Error("The ATOM of the account is enough,"+
					" but cannot be sent in just one transaction!\n"+
					"the amount can be sent in "+fmt.Sprint(len(countList))+
					"times with amounts :\n"+strings.Replace(strings.Trim(fmt.Sprint(countList), "[]"), " ", ",", -1), err)
				return err
			} else {
				countList = append(countList, a.Balance.Uint64())
			}
			continue
		}
		from = a.Address
		break
	}

	if from == "" {
		return openwallet.Errorf(openwallet.ErrInsufficientBalanceOfAccount, "the balance: %s is not enough", amountStr)
	}

	rawTx.TxFrom = []string{from}
	rawTx.TxTo = []string{to}
	rawTx.TxAmount = amountStr
	rawTx.Fees = convertToAmount(fee)
	rawTx.FeeRate = convertToAmount(fee)

	denom := decoder.wm.Config.Denom
	chainID := decoder.wm.Config.ChainID
	var sequence uint64
	sequence_db, err := wrapper.GetAddressExtParam(from, decoder.wm.FullName())
	if err != nil {
		return err
	}
	if sequence_db == nil {
		sequence = 0
	} else {
		sequence = ow.NewString(sequence_db).UInt64()
	}
	accountNumber, sequenceChain, err := decoder.wm.RestClient.getAccountNumberAndSequence(from)
	if err != nil {
		return err
	}
	if sequenceChain > int(sequence) {
		sequence = uint64(sequenceChain)
	}
	memo := rawTx.GetExtParam().Get("memo").String()

	messageType := decoder.wm.Config.MsgType

	txFee := cosmosTransaction.NewStdFee(int64(gas), cosmosTransaction.Coins{cosmosTransaction.NewCoin(denom, int64(fee))})
	message := []cosmosTransaction.Message{cosmosTransaction.NewMessage(messageType, cosmosTransaction.NewMsgSend(from, to, cosmosTransaction.Coins{cosmosTransaction.NewCoin(denom, int64(convertFromAmount(amountStr)))}))}
	txStruct := cosmosTransaction.NewTxStruct(chainID, memo, accountNumber, int(sequence), &txFee, message)

	emptyTrans, hash, err := txStruct.CreateEmptyTransactionAndHash()
	if err != nil {
		return err
	}
	rawTx.RawHex = emptyTrans

	if rawTx.Signatures == nil {
		rawTx.Signatures = make(map[string][]*openwallet.KeySignature)
	}

	keySigs := make([]*openwallet.KeySignature, 0)

	addr, err := wrapper.GetAddress(from)
	if err != nil {
		return err
	}
	signature := openwallet.KeySignature{
		EccType: decoder.wm.Config.CurveType,
		Nonce:   "",
		Address: addr,
		Message: hash,
	}

	keySigs = append(keySigs, &signature)

	rawTx.Signatures[rawTx.Account.AccountID] = keySigs

	rawTx.FeeRate = big.NewInt(int64(fee)).String()

	rawTx.IsBuilt = true

	return nil
}

func (decoder *TransactionDecoder) SignATOMRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {
	key, err := wrapper.HDKey()
	if err != nil {
		return nil
	}

	keySignatures := rawTx.Signatures[rawTx.Account.AccountID]

	if keySignatures != nil {
		for _, keySignature := range keySignatures {

			childKey, err := key.DerivedKeyWithPath(keySignature.Address.HDPath, keySignature.EccType)
			keyBytes, err := childKey.GetPrivateKeyBytes()
			if err != nil {
				return err
			}

			//签名交易
			/////////交易单哈希签名
			sig, err := cosmosTransaction.SignTransactionHash(keySignature.Message, keyBytes)
			if err != nil {
				return fmt.Errorf("transaction hash sign failed, unexpected error: %v", err)
			} else {

				//for i, s := range sigPub {
				//	log.Info("第", i+1, "个签名结果")
				//	log.Info()
				//	log.Info("对应的公钥为")
				//	log.Info(hex.EncodeToString(s.Pubkey))
				//}

				// txHash.Normal.SigPub = *sigPub
			}

			keySignature.Signature = sig
		}
	}

	log.Info("transaction hash sign success")

	rawTx.Signatures[rawTx.Account.AccountID] = keySignatures

	return nil
}

func (decoder *TransactionDecoder) VerifyATOMRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	var (
		emptyTrans = rawTx.RawHex

		signature = ""
		pubkey    = []byte{}
	)

	for accountID, keySignatures := range rawTx.Signatures {
		log.Debug("accountID Signatures:", accountID)
		for _, keySignature := range keySignatures {

			signature = keySignature.Signature
			pubkey, _ = hex.DecodeString(keySignature.Address.PublicKey)

			log.Debug("Signature:", signature)
			log.Debug("PublicKey:", hex.EncodeToString(pubkey))
		}
	}
	point := owcrypt.PointDecompress(pubkey, decoder.wm.CurveType())[1:]
	pass := cosmosTransaction.VerifyTransactionSig(emptyTrans, signature, point)
	var txStruct cosmosTransaction.TxStruct
	json.Unmarshal([]byte(emptyTrans), &txStruct)
	keyType := "tendermint/PubKeySecp256k1"
	snedmode := "sync" //"block"
	signedTrans, err := txStruct.CreateJsonForSend(signature, pubkey, keyType, snedmode)

	if err != nil {
		return fmt.Errorf("transaction compose signatures failed")
	}

	if pass {
		log.Debug("transaction verify passed")
		rawTx.IsCompleted = true
		rawTx.RawHex = signedTrans.Raw
	} else {
		log.Debug("transaction verify failed")
		rawTx.IsCompleted = false
	}

	return nil
}

func (decoder *TransactionDecoder) GetRawTransactionFeeRate() (feeRate string, unit string, err error) {
	if decoder.wm.Config.PayFee {
		return convertToAmount(decoder.wm.Config.MinFee), "TX", nil
	} else {
		return convertToAmount(0), "TX", nil
	}
}

//CreateSummaryRawTransaction 创建汇总交易，返回原始交易单数组
func (decoder *TransactionDecoder) CreateSummaryRawTransaction(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransaction, error) {
	if sumRawTx.Coin.IsContract {
		return nil, nil
	} else {
		return decoder.CreateSimpleSummaryRawTransaction(wrapper, sumRawTx)
	}
}

func (decoder *TransactionDecoder) CreateSimpleSummaryRawTransaction(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransaction, error) {

	var (
		rawTxArray      = make([]*openwallet.RawTransaction, 0)
		accountID       = sumRawTx.Account.AccountID
		minTransfer     = big.NewInt(int64(convertFromAmount(sumRawTx.MinTransfer)))
		retainedBalance = big.NewInt(int64(convertFromAmount(sumRawTx.RetainedBalance)))
	)

	if minTransfer.Cmp(retainedBalance) < 0 {
		return nil, fmt.Errorf("mini transfer amount must be greater than address retained balance")
	}

	//获取wallet
	addresses, err := wrapper.GetAddressList(sumRawTx.AddressStartIndex, sumRawTx.AddressLimit,
		"AccountID", sumRawTx.Account.AccountID)
	if err != nil {
		return nil, err
	}

	if len(addresses) == 0 {
		return nil, fmt.Errorf("[%s] have not addresses", accountID)
	}

	searchAddrs := make([]string, 0)
	for _, address := range addresses {
		searchAddrs = append(searchAddrs, address.Address)
	}

	addrBalanceArray, err := decoder.wm.Blockscanner.GetBalanceByAddress(searchAddrs...)
	if err != nil {
		return nil, err
	}

	for _, addrBalance := range addrBalanceArray {

		//检查余额是否超过最低转账
		addrBalance_BI := big.NewInt(int64(convertFromAmount(addrBalance.Balance)))

		if addrBalance_BI.Cmp(minTransfer) < 0 {
			continue
		}
		//计算汇总数量 = 余额 - 保留余额
		sumAmount_BI := new(big.Int)
		sumAmount_BI.Sub(addrBalance_BI, retainedBalance)

		//this.wm.Log.Debug("sumAmount:", sumAmount)
		//计算手续费
		fee := big.NewInt(0) //(int64(decoder.wm.Config.FeeCharge))
		if len(sumRawTx.FeeRate) > 0 {
			fee = big.NewInt(int64(convertFromAmount(sumRawTx.FeeRate)))
		} else {
			if decoder.wm.Config.PayFee {
				fee = big.NewInt((int64(decoder.wm.Config.MinFee)))
			}
		}

		//减去手续费
		sumAmount_BI.Sub(sumAmount_BI, fee)
		if sumAmount_BI.Cmp(big.NewInt(0)) <= 0 {
			continue
		}

		sumAmount := convertToAmount(sumAmount_BI.Uint64())
		fees := convertToAmount(fee.Uint64())

		log.Debugf("balance: %v", addrBalance.Balance)
		log.Debugf("fees: %v", fees)
		log.Debugf("sumAmount: %v", sumAmount)

		//创建一笔交易单
		rawTx := &openwallet.RawTransaction{
			Coin:    sumRawTx.Coin,
			Account: sumRawTx.Account,
			To: map[string]string{
				sumRawTx.SummaryAddress: sumAmount,
			},
			Required: 1,
		}

		createErr := decoder.createRawTransaction(
			wrapper,
			rawTx,
			addrBalance)
		if createErr != nil {
			return nil, createErr
		}

		//创建成功，添加到队列
		rawTxArray = append(rawTxArray, rawTx)

	}
	return rawTxArray, nil
}

func (decoder *TransactionDecoder) createRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction, addrBalance *openwallet.Balance) error {

	gas := decoder.wm.Config.StdGas
	fee := uint64(0) //decoder.wm.Config.FeeCharge
	if decoder.wm.Config.PayFee {
		fee = decoder.wm.Config.MinFee
	}

	var amountStr, to string
	for k, v := range rawTx.To {
		to = k
		amountStr = v
		break
	}

	amount := big.NewInt(int64(convertFromAmount(amountStr)))
	amount = amount.Add(amount, big.NewInt(int64(fee)))
	from := addrBalance.Address
	fromAddr, err := wrapper.GetAddress(from)
	if err != nil {
		return err
	}
	//fromPubkey := fromAddr.PublicKey

	rawTx.TxFrom = []string{from}
	rawTx.TxTo = []string{to}
	rawTx.TxAmount = amountStr
	rawTx.Fees = convertToAmount(fee)
	rawTx.FeeRate = convertToAmount(fee)

	denom := decoder.wm.Config.Denom
	chainID := decoder.wm.Config.ChainID
	var sequence uint64
	sequence_db, err := wrapper.GetAddressExtParam(from, decoder.wm.FullName())
	if err != nil {
		return err
	}
	if sequence_db == nil {
		sequence = 0
	} else {
		sequence = ow.NewString(sequence_db).UInt64()
	}
	accountNumber, sequenceChain, err := decoder.wm.RestClient.getAccountNumberAndSequence(from)
	if err != nil {
		return err
	}
	if sequenceChain > int(sequence) {
		sequence = uint64(sequenceChain)
	}
	memo := ""

	messageType := decoder.wm.Config.MsgType

	txFee := cosmosTransaction.NewStdFee(int64(gas), cosmosTransaction.Coins{cosmosTransaction.NewCoin(denom, int64(fee))})
	message := []cosmosTransaction.Message{cosmosTransaction.NewMessage(messageType, cosmosTransaction.NewMsgSend(from, to, cosmosTransaction.Coins{cosmosTransaction.NewCoin(denom, int64(convertFromAmount(amountStr)))}))}
	txStruct := cosmosTransaction.NewTxStruct(chainID, memo, accountNumber, int(sequence), &txFee, message)
	emptyTrans, hash, err := txStruct.CreateEmptyTransactionAndHash()
	if err != nil {
		return err
	}
	rawTx.RawHex = emptyTrans

	if rawTx.Signatures == nil {
		rawTx.Signatures = make(map[string][]*openwallet.KeySignature)
	}

	keySigs := make([]*openwallet.KeySignature, 0)

	signature := openwallet.KeySignature{
		EccType: decoder.wm.Config.CurveType,
		Nonce:   "",
		Address: fromAddr,
		Message: hash,
	}

	keySigs = append(keySigs, &signature)

	rawTx.Signatures[rawTx.Account.AccountID] = keySigs

	rawTx.FeeRate = big.NewInt(int64(fee)).String()

	rawTx.IsBuilt = true

	return nil
}

//CreateSummaryRawTransactionWithError 创建汇总交易，返回能原始交易单数组（包含带错误的原始交易单）
func (decoder *TransactionDecoder) CreateSummaryRawTransactionWithError(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransactionWithError, error) {
	raTxWithErr := make([]*openwallet.RawTransactionWithError, 0)
	rawTxs, err := decoder.CreateSummaryRawTransaction(wrapper, sumRawTx)
	if err != nil {
		return nil, err
	}
	for _, tx := range rawTxs {
		raTxWithErr = append(raTxWithErr, &openwallet.RawTransactionWithError{
			RawTx: tx,
			Error: nil,
		})
	}
	return raTxWithErr, nil
}
