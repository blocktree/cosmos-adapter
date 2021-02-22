package cosmos

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blocktree/go-owcrypt"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/types"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

func NewPublicKey(key []byte) cryptotypes.PubKey {
	return &secp256k1.PubKey{Key: key}
}

//func NewPrivateKey(key []byte) cryptotypes.PrivKey {
//	return &secp256k1.PrivKey{Key: key}
//}

type CosmosTx struct {
	From string `json:"from"`
	To string `json:"to"`
	Denom string `json:"denom"`
	FeeDenom string `json:"fee_denom"`
	Memo string `json:"memo"`
	ChainID string `json:"chain_id"`
	PublicKey string `json:"public_key"`
	Amount int64 `json:"amount"`
	Fee int64 `json:"fee"`
	AccNum uint64 `json:"acc_num"`
	AccSeq uint64 `json:"acc_seq"`
	GasLimit uint64 `json:"gas_limit"`
	Timeout uint64 `json:"timeout"`
}


func (t CosmosTx) getUnsignedTxAndHash() (string, string, error) {
	encCfg := simapp.MakeTestEncodingConfig()
	txBuilder := encCfg.TxConfig.NewTxBuilder()

	from, err := types.AccAddressFromBech32(t.From)
	if err != nil {
		return "", "", err
	}
	to, err := types.AccAddressFromBech32(t.To)
	if err != nil {
		return "", "", err
	}

	msg := banktypes.NewMsgSend(from, to, types.NewCoins(types.NewInt64Coin(t.Denom, t.Amount)))

	err = txBuilder.SetMsgs(msg)
	if err != nil {
		return "", "", err
	}

	txBuilder.SetGasLimit(t.GasLimit)
	txBuilder.SetFeeAmount(types.NewCoins(types.NewCoin(t.FeeDenom, types.NewInt(t.Fee))))
	txBuilder.SetMemo(t.Memo)
	txBuilder.SetTimeoutHeight(t.Timeout)

	var sigsV2 []signing.SignatureV2
	publicKey, err := hex.DecodeString(t.PublicKey)
	if err != nil {
		return "", "", err
	}

	sigV2 := signing.SignatureV2{
		PubKey:   NewPublicKey(publicKey),
		Data:     &signing.SingleSignatureData{
			SignMode:  encCfg.TxConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: t.AccSeq,
	}

	sigsV2 = append(sigsV2, sigV2)

	err = txBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return "", "", err
	}

	signerData := xauthsigning.SignerData{
		ChainID:       t.ChainID,
		AccountNumber: t.AccNum,
		Sequence:      t.AccSeq,
	}
	sigbytes, err := encCfg.TxConfig.SignModeHandler().GetSignBytes(encCfg.TxConfig.SignModeHandler().DefaultMode(), signerData, txBuilder.GetTx())
	if err != nil {
		return "", "", err
	}

	sigHash := owcrypt.Hash(sigbytes, 0, owcrypt.HASH_ALG_SHA256)

	txBytes, err := json.Marshal(t)
	if err != nil {
		return "", "", err
	}
	
	return hex.EncodeToString(txBytes), hex.EncodeToString(sigHash), nil
}

func signTransactionHash(txHash string, prikey []byte) (string, error) {
	hash, err := hex.DecodeString(txHash)
	if err != nil {
		return "", errors.New("Invalid transaction hash!")
	}
	if len(hash) != 32 || prikey == nil || len(prikey) != 32 {
		return "", errors.New("Invalid transaction hash!")
	}

	sig,_, ret := owcrypt.Signature(prikey, nil, hash, owcrypt.ECC_CURVE_SECP256K1)

	if ret != owcrypt.SUCCESS {
		return "", errors.New("Signature failed!")
	}

	return hex.EncodeToString(sig), nil
}

func getBroadcastBytes(unsignedTrans,  signature string) (string, error) {
	txBytes, err := hex.DecodeString(unsignedTrans)
	if err != nil || len(txBytes) == 0 {
		return "", err
	}
	t := CosmosTx{}
	err = json.Unmarshal(txBytes, &t)
	if err != nil {
		return "", err
	}
	sig, err := hex.DecodeString(signature)
	if err != nil || len(sig) != 64 {
		return "", err
	}




	encCfg := simapp.MakeTestEncodingConfig()
	txBuilder := encCfg.TxConfig.NewTxBuilder()

	from, err := types.AccAddressFromBech32(t.From)
	if err != nil {
		return "", err
	}
	to, err := types.AccAddressFromBech32(t.To)
	if err != nil {
		return "", err
	}

	msg := banktypes.NewMsgSend(from, to, types.NewCoins(types.NewInt64Coin(t.Denom, t.Amount)))

	err = txBuilder.SetMsgs(msg)
	if err != nil {
		return "", err
	}

	txBuilder.SetGasLimit(t.GasLimit)
	txBuilder.SetFeeAmount(types.NewCoins(types.NewCoin(t.FeeDenom, types.NewInt(t.Fee))))
	txBuilder.SetMemo(t.Memo)
	txBuilder.SetTimeoutHeight(t.Timeout)

	var sigsV2 []signing.SignatureV2
	publicKey, err := hex.DecodeString(t.PublicKey)
	if err != nil {
		return "", err
	}

	sigV2 := signing.SignatureV2{
		PubKey:   NewPublicKey(publicKey),
		Data:     &signing.SingleSignatureData{
			SignMode:  encCfg.TxConfig.SignModeHandler().DefaultMode(),
			Signature: sig,
		},
		Sequence: t.AccSeq,
	}

	sigsV2 = append(sigsV2, sigV2)

	err = txBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return "", err
	}


	txBytes, err = encCfg.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(txBytes) + ":" + t.From + "@" + fmt.Sprint(t.AccSeq), nil
}