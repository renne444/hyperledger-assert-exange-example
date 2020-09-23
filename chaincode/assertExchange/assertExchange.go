package main

import (
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type AssertExchangeCC struct {}

const (
	originOwner = "originOwnerPlaceHolder"
)

// 用户
type User struct {
	Name string 				`json:"name"` //其他序列化格式：messagepack || protobuf
	Id 	 string 				`json:"id"`
	//Assets map[string]string	`json:"asserts"`// 资产id->资产name 这里可能会引入bug，因为map的顺序可能不一致
	Assets []string `json:"asserts"`
}

type Asset struct{
	Name 		string 				`json:"name"`
	Id	 		string 				`json:"id"`
	//Metadata	map[string]string 	`json:"metadata"`  // 特殊
	Metadata string `json:"metadata"`
}

type AssetHistory struct {
	AssetId			string		`json:"asset_id"`
	OriginOwnerId	string 		`json:"origin_owner_id"`
	CurrentOwnerId	string		`json:"current_owner_id"`
}

// 写接口的套路
// 套路1 ： 检验参数个数
// 套路2 ： 检验参数正确性
// 套路3 ： 验证数据是否存在 应该存在/不应该存在 比如开户不应该存在
// 套路4 ： 状态写入

func constructUserKey(userId string) string {
	return fmt.Sprintf("user_%s",userId)
}

func constructAssetKey(assetId string) string{
	return fmt.Sprintf("asset_%s",assetId)
}

func userRegister(stub shim.ChaincodeStubInterface ,args []string) pb.Response {

	if len(args) != 2 {
		return shim.Error("not enough args")
	}

	name := args[0]
	id := args[1]
	if name == "" || id == ""{
		return shim.Error("invalid args")
	}

	if userBytes, err := stub.GetState(constructUserKey(id)); err == nil && len(userBytes) != 0{
		return shim.Error("user already exist")
	}

	user := &User{
		Name:   name,
		Id:     id,
		Assets: make([]string,0),
	}
	userBytes ,err := json.Marshal(user)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error %s",err))
	}
	if err := stub.PutState(constructUserKey(id),userBytes); err != nil {
		return shim.Error(fmt.Sprintf("put user error %s",err))
	}

	fmt.Println("user registered")
	return shim.Success(userBytes)
}

func userDestroy(stub shim.ChaincodeStubInterface, args []string)pb.Response{
	if len(args) != 1 {
		return shim.Error("not enouth args")
	}

	id := args[0]
	if  id == ""{
		return shim.Error("invalid args")
	}

	userBytes, err := stub.GetState(constructUserKey(id))
	if err != nil || len(userBytes) == 0{
		return shim.Error("user not found")
	}

	if err := stub.DelState(constructUserKey(id)); err != nil{
		return shim.Error(fmt.Sprintf("delete user error: %s",err))
	}
	user := new(User)
	if err := json.Unmarshal(userBytes,user); err != nil {
		return shim.Error(fmt.Sprintf("unmarshal user error: %s",err))
	}
	for _, assetId := range user.Assets{
		if err := stub.DelState(constructAssetKey(assetId)); err != nil {
			return shim.Error(fmt.Sprintf("delete asset error: %s",err))
		}
	}

	return shim.Success(nil)
}

func assetEnroll(stub shim.ChaincodeStubInterface, args []string)pb.Response{
	if len(args) != 4 {
		return shim.Error("not enouth args")
	}


	assetName := args[0]
	assetId := args[1]
	metadata := args[2]
	ownerId := args[3]
	if assetName == "" || assetId == "" || ownerId == ""{
		return shim.Error("invalid args")
	}

	userBytes, err := stub.GetState(constructUserKey(ownerId))
	if err != nil || len(userBytes) == 0{
		return shim.Error("user not found")
	}
	assetIdBytes, err := stub.GetState(constructAssetKey(assetId))
	if err == nil && len(assetIdBytes) != 0{
		return shim.Error("asset already exist")
	}

	//写入资产对象 更新用户对象 写入变更记录
	asset := &Asset{
		Name: assetName,
		Id : assetId,
		Metadata:metadata,
	}
	assetBytes ,err := json.Marshal(asset)
	if err != nil{
		return shim.Error(fmt.Sprintf("marshal asset error: %s",err))
	}
	if err := stub.PutState(constructAssetKey(assetId),assetBytes); err != nil{
		return shim.Error(fmt.Sprintf("save asset error: %s",err))
	}

	user := new(User)
	if err := json.Unmarshal(userBytes,user);err != nil{
		return shim.Error(fmt.Sprintf("unmarshal user error: %s",err))
	}
	user.Assets = append(user.Assets,assetId)
	userBytes, err = json.Marshal(user)
	if err != nil {
		shim.Error(fmt.Sprintf("marshal user error: %s",err))
	}
	if err := stub.PutState(constructUserKey(user.Id),userBytes); err != nil{
		return shim.Error(fmt.Sprintf("update user error: %s",err))
	}

	history := AssetHistory{
		AssetId:        assetId,
		OriginOwnerId:  originOwner,
		CurrentOwnerId: ownerId,
	}
	historyBytes, err := json.Marshal(history)
	if err != nil{
		return shim.Error(fmt.Sprintf("marshal assert history error: %s",err))
	}

	historyKey,err := stub.CreateCompositeKey("history",[]string{
		assetId,originOwner,ownerId,
	})
	if err != nil{
		shim.Error(fmt.Sprintf("create key error: %s",err))
	}

	if err := stub.PutState(historyKey,historyBytes);err != nil{
		return shim.Error(fmt.Sprintf("save asset history error: %s",err))
	}

	return shim.Success(nil)
}

func assetExchange(stub shim.ChaincodeStubInterface, args []string)pb.Response{
	if len(args) != 3 {
		return shim.Error("not enouth args")
	}

	ownerId := args[0]
	assetId := args[1]
	currentOwnerId := args[2]

	if ownerId == "" || assetId == "" || currentOwnerId == "" {
		return shim.Error("invalid args")
	}


	originOwnerBytes, err := stub.GetState(constructUserKey(ownerId))
	if err != nil || len(originOwnerBytes) == 0{
		return shim.Error("ownerId not found")
	}
	currentOwnerBytes, err := stub.GetState(constructUserKey(currentOwnerId))
	if  err != nil || len(currentOwnerBytes) == 0{
		return shim.Error("currentOwnerId not found")
	}
	if assetIdBytes, err := stub.GetState(constructAssetKey(assetId)); err != nil || len(assetIdBytes) == 0{
		return shim.Error("asset not found")
	}

	//校验原始拥有者确实拥有资产
	originOwner := new(User)
	if err := json.Unmarshal(originOwnerBytes,originOwner);err != nil{
		return shim.Error(fmt.Sprintf("unmarshal user error: %s",err))
	}
	aidexist := false
	for _, aid := range originOwner.Assets{
		if aid == assetId {
			aidexist = true
			break
		}
	}
	if aidexist == false {
		return shim.Error("asset owner not match")
	}

	assetIds := make([]string,0)
	for _,aid := range originOwner.Assets{
		if aid == assetId {
			continue
		}
		assetIds = append(assetIds,aid)
	}
	originOwner.Assets = assetIds

	originOwnerBytes, err = json.Marshal(originOwner)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s",err))
	}
	if err := stub.PutState(constructUserKey(ownerId), originOwnerBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s",err))
	}

	// 当前拥有者插入资产ID
	currentOwner := &User{}
	if err := json.Unmarshal(currentOwnerBytes,currentOwner);err != nil{
		return shim.Error(fmt.Sprintf("unmarshal user error: %s",err))
	}
	currentOwner.Assets = append(currentOwner.Assets,assetId)

	currentOwnerBytes, err = json.Marshal(currentOwner)
	if err != nil {
		return shim.Error(fmt.Sprintf("marshal user error: %s",err))
	}
	if err := stub.PutState(constructUserKey(currentOwnerId), currentOwnerBytes); err != nil {
		return shim.Error(fmt.Sprintf("update user error: %s",err))
	}

	// 插入资产变更记录
	historyKey,err := stub.CreateCompositeKey("history",[]string{
		assetId,ownerId,currentOwnerId,
	})
	history := AssetHistory{
		AssetId:        assetId,
		OriginOwnerId:  ownerId,
		CurrentOwnerId: currentOwnerId,
	}
	historyBytes, err := json.Marshal(history)
	if err != nil{
		return shim.Error(fmt.Sprintf("marshal assert history error: %s",err))
	}
	if err := stub.PutState(historyKey,historyBytes);err != nil{
		return shim.Error(fmt.Sprintf("save asset history error: %s",err))
	}


	return shim.Success(nil)
}

func queryUser(stub shim.ChaincodeStubInterface, args [] string) pb.Response{

	if len(args) != 1 {
		return shim.Error("not enouth args")
	}

	id := args[0]
	if id == "" {
		return shim.Error("invalid args")
	}


	userBytes, err := stub.GetState(constructUserKey(id))
	if err != nil || len(userBytes) == 0{
		return shim.Error("user not found")
	}

	return shim.Success(userBytes)
}

func queryAsset(stub shim.ChaincodeStubInterface, args []string) pb.Response{

	if len(args) != 1 {
		return shim.Error("not enough args")
	}

	assetId := args[0]
	if assetId == ""{
		return shim.Error("invalid args")
	}


	assetBytes, err := stub.GetState(constructAssetKey(assetId))
	if  err != nil || len(assetBytes) == 0{
		return shim.Error("asset not found")
	}
	return shim.Success(assetBytes)
}

func queryAssetHistory(stub shim.ChaincodeStubInterface, args []string) pb.Response{

	if len(args) != 2 && len(args) != 1 {
		return shim.Error("not enough args")
	}

	assetId := args[0]
	if assetId == "" {
		shim.Error("invalid args")
	}

	queryType   := "all"
	if len(args) == 2 {
		queryType = args[1]
	}
	if queryType != "all" && queryType != "enroll" && queryType != "exchange" {
		return shim.Error(fmt.Sprintf("queryType unknown %s",queryType))
	}

	assetBytes, err := stub.GetState(constructAssetKey(assetId))
	if  err != nil || len(assetBytes) == 0{
		return shim.Error("asset not found")
	}

	//查询组合键历史数据
	keys := make([]string,0)
	keys = append(keys,assetId)
	switch queryType{
	case "enroll":
		keys = append(keys, originOwner)
	case "exchange", "all": //不添加任何附件
	default:
		return shim.Error(fmt.Sprintf("unsupport query type : %s",queryType))
	}
	result,err := stub.GetStateByPartialCompositeKey("history",keys)
	if err != nil {
		return shim.Error(fmt.Sprintf("query history error: %s",err))
	}

	histories := make([]*AssetHistory,0)
	for result.HasNext() {
		historyVal, err := result.Next()
		if err != nil {
			return shim.Error(fmt.Sprintf("query error :%s",err))
		}
		history := new(AssetHistory)
		if err := json.Unmarshal(historyVal.GetValue(),history); err != nil{
			shim.Error("unmarshal error: %s")
		}

		//过滤掉不是资产转让到记录
		if queryType == "exchange" && history.OriginOwnerId == originOwner{
			continue
		}
		histories = append(histories,history)
	}
	historiesBytes, err := json.Marshal(histories)
	if err != nil{
		return shim.Error(fmt.Sprintf("marshal history error: %s",err))
	}

	return shim.Success(historiesBytes)
}

// Init is called during Instantiate transaction after the chaincode container
// has been established for the first time, allowing the chaincode to
// initialize its internal data
func (c *AssertExchangeCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke is called to update or query the ledger in a proposal transaction.
// Updated state variables are not committed to the ledger until the
// transaction is committed.
func (c *AssertExchangeCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {

	funcName, args := stub.GetFunctionAndParameters()

	switch funcName {
	case "userRegister" : return userRegister(stub,args)
	case "userDestroy"  : return userDestroy(stub,args)
	case "assetEnroll"  : return assetEnroll(stub,args)
	case "assetExchange": return assetExchange(stub,args)
	case "queryUser"	: return queryUser(stub,args)
	case "queryAsset"	: return queryAsset(stub,args)
	case "queryAssetHistory" : return queryAssetHistory(stub,args)
	default:
		return shim.Error(fmt.Sprintf("unsupport function: %s",funcName))
	}
}

func main() {
	err := shim.Start(new(AssertExchangeCC))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

