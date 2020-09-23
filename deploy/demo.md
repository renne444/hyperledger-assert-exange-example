## 工作路径设置

```bash
export FABRIC_CFG_PATH=/home/yuki/go/src/github.com/hyperledger/renne/deploy
```

## 生成证书文件

```bash
cryptogen generate --config=./crypto-config.yaml
```
org0.imocc.com
org1.imocc.com

## 生成创始区块

```bash
configtxgen -profile OneOrgOrdererGenesis -outputBlock ./config/genesis.block
```

## 生成通道的创始交易

```bash
configtxgen -profile TwoOrgsChannel -outputCreateChannelTx ./config/mychannel.tx -channelID mychannel
```

## 生成组织主节点的交易
```bash
configtxgen -profile TwoOrgsChannel -outputAnchorPeersUpdate ./config/Org0MSPanchors.tx -channelID mychannel -asOrg Org0MSP
configtxgen -profile TwoOrgsChannel -outputAnchorPeersUpdate ./config/Org1MSPanchors.tx -channelID mychannel -asOrg Org1MSP
```

## 创建通道
```bash
peer channel create -o orderer.imocc.com:7050 -c mychannel -f /etc/hyperledger/config/mychannel.tx
```

## 加入通道
-b 指创世区块
```bash
peer channel join -b mychannel.block
```

## 主节点设置
```bash
peer channel update -o orderer.imocc.com:7050 -c mychannel -f /etc/hyperledger/config/Org1MSPanchors.tx
```

## 安装链码
```bash
peer chaincode install -n badexample -v 1.0.0 -l golang -p github.com/chaincode/badexample
```
## 链码实例化

```bash
peer chaincode instantiate -o orderer.imocc.com:7050 -C mychannel -n badexample -l golang -v 1.0.0 -c '{"Args":["init"]}'
```

## 链码查询

```bash
peer chaincode query -C mychannel -n badexample -c '{"Args":[]}'
```

