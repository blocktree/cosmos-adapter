# cosmos-adapter

本项目适配了openwallet.AssetsAdapter接口，给应用提供了底层的区块链协议支持。

## 如何测试

openwtester包下的测试用例已经集成了openwallet钱包体系，创建conf文件，新建ATOM.ini文件，编辑如下内容：

```ini

# transaction type
txType = "auth/StdTx"
# message type
msgSend = "cosmos-sdk/MsgSend"
msgVote = "cosmos-sdk/MsgVote"
msgDelegate = "cosmos-sdk/MsgDelegate"
# message choose 1-send  2-vote  3-delegate
msgType = 1


# mainnet rest api url
mainnetRestAPI = "http://127.0.0.1:1317"
# mainnet node api url
mainnetNodeAPI = "http://127.0.0.1:26657"
# chain id
mainnetChainID = "cosmoshub-2"
# mainnet denom
mainnetDenom = "uatom"

# testnet rest api url
testnetRestAPI = "http://192.168.27.124:20042"
# testnet node api url
testnetNodeAPI = "http://192.168.27.124:20041"
# chain id
testnetChainID = "gaia-13003"
# testnet denom
testnetDenom = "muon"

# Is network test?
isTestNet = true

# scan mempool or not
isScanMemPool = false

# pay fee or not
payFee = false
# minimum fee to pay in muon/uatom(1 mon = 1000000muon , 1 atom = 1000000uatom)
minFee = 1000
# standed gas
stdGas = 200000

# Cache data file directory, default = "", current directory: ./data
dataDir = ""
```
