package cosmos

import (
	"encoding/hex"
	"fmt"
	"testing"
)

func Test_transaction(t *testing.T) {
	cosmosTx := CosmosTx{
		From:      "cosmos1djhe9ury7c05gu5ptjefv0uj9gp48a90vxq3u9",
		To:        "cosmos1rc0ya7shas5wkq8ua0g3zhg6dpzu8l9hj3x72n",
		Denom:     "uatom",
		FeeDenom:  "uatom",
		Memo:      "123",
		ChainID:   "cosmoshub-4",
		PublicKey: "025b8ed615288ce216206af060838d5df5c2d14af2651dd231c199ab2567dbb0a3",
		Amount:    500000,
		Fee:       2500,
		AccNum:    173110,
		AccSeq:    5,
		GasLimit:  200000,
		Timeout:   0,
	}

	unsignedTrans, hash, err := cosmosTx.getUnsignedTxAndHash()

	if err != nil {
		t.Error("create failed")
		return
	}

	fmt.Println("tx : ", unsignedTrans)
	fmt.Println("hash : ", hash)


	private_key, _ := hex.DecodeString("1234567812345678123456781234567812345678123456781234567812345678")
	signature, err := signTransactionHash(hash, private_key)
	if err != nil {
		t.Error("sign failed")
		return
	}

	// signature = hex.EncodeToString([]byte{169,213,22,69,126,153,158,86,46,52,137,108,112,198,224,171,82,18,230,38,133,30,179,81,31,245,74,123,106,248,57,172,95,199,41,201,188,125,51,35,152,52,112,59,149,92,235,23,217,162,165,83,76,118,72,22,31,24,222,231,10,100,65,227})
	fmt.Println("signature : ", signature)

	broadcastBytes, err :=getBroadcastBytes(unsignedTrans, signature)
	if err != nil {
		t.Error("combine failed")
		return
	}

	fmt.Println("broadcast : ", broadcastBytes)
}