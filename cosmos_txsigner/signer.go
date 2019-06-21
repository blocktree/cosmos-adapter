package cosmos_txsigner

import (
	"fmt"

	"github.com/blocktree/go-owcrypt"
)

var Default = &TransactionSigner{}

type TransactionSigner struct {
}

// SignTransactionHash 交易哈希签名算法
// required
func (singer *TransactionSigner) SignTransactionHash(msg []byte, privateKey []byte, eccType uint32) ([]byte, error) {
	signature, retCode := owcrypt.Signature(privateKey, nil, 0, msg, 32, owcrypt.ECC_CURVE_SECP256K1)
	if retCode != owcrypt.SUCCESS {
		return nil, fmt.Errorf("ECC sign hash failed")
	}

	return signature, nil
}
