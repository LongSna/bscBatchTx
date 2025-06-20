package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Config 配置结构体
type Config struct {
	PrivateKeys []string `json:"private_keys"`
	RpcUrl      string   `json:"rpc_url"`
	ChainID     int64    `json:"chain_id"`
	ToAddress   string   `json:"to_address"`
	Value       string   `json:"value"`        // 转账金额 (wei)
	GasLimit    uint64   `json:"gas_limit"`
	GasPrice    string   `json:"gas_price"`    // gas价格 (wei)
	Data        string   `json:"data"`         // 十六进制数据 (可选)
}

// TransactionResult 交易结果
type TransactionResult struct {
	Address string
	TxHash  string
	Error   error
}

//func dataProvider() string{
	
	//return "0x01"
	
//}

func main() {
	// 读取配置文件
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}
	//config.Data = dataProvider()

	// 连接到BSC节点
	client, err := ethclient.Dial(config.RpcUrl)
	if err != nil {
		log.Fatalf("连接到RPC节点失败: %v", err)
	}
	defer client.Close()

	// 验证链ID
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatalf("获取链ID失败: %v", err)
	}
	
	if chainID.Int64() != config.ChainID {
		log.Printf("警告: 配置的链ID (%d) 与实际链ID (%d) 不匹配", config.ChainID, chainID.Int64())
	}

	fmt.Printf("连接到BSC网络，链ID: %d\n", chainID.Int64())
	fmt.Printf("准备发送 %d 笔交易...\n", len(config.PrivateKeys))

	// 创建协程同时发送交易
	var wg sync.WaitGroup
	results := make(chan TransactionResult, len(config.PrivateKeys))

	for i, privateKeyHex := range config.PrivateKeys {
		wg.Add(1)
		go func(index int, pkHex string) {
			defer wg.Done()
			result := sendTransaction(client, config, pkHex, index)
			results <- result
		}(i, privateKeyHex)
		
		// 稍微延迟一下，避免nonce冲突
		time.Sleep(100 * time.Millisecond)
	}

	// 等待所有协程完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	var successCount, failCount int
	fmt.Println("\n=== 交易结果 ===")
	for result := range results {
		if result.Error != nil {
			fmt.Printf("❌ 地址 %s: 失败 - %v\n", result.Address, result.Error)
			failCount++
		} else {
			fmt.Printf("✅ 地址 %s: 成功 - 交易哈希: %s\n", result.Address, result.TxHash)
			successCount++
		}
	}

	fmt.Printf("\n=== 总结 ===\n")
	fmt.Printf("成功: %d 笔\n", successCount)
	fmt.Printf("失败: %d 笔\n", failCount)
	fmt.Printf("总计: %d 笔\n", successCount+failCount)
}

// loadConfig 加载配置文件
func loadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 验证必要字段
	if len(config.PrivateKeys) == 0 {
		return nil, fmt.Errorf("私钥列表不能为空")
	}
	if config.RpcUrl == "" {
		return nil, fmt.Errorf("RPC URL不能为空")
	}
	if config.ToAddress == "" {
		return nil, fmt.Errorf("目标地址不能为空")
	}

	return &config, nil
}

// sendTransaction 发送交易
func sendTransaction(client *ethclient.Client, config *Config, privateKeyHex string, index int) TransactionResult {
	// 移除0x前缀
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}

	// 解析私钥
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return TransactionResult{Error: fmt.Errorf("解析私钥失败: %v", err)}
	}

	// 获取公钥和地址
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return TransactionResult{Error: fmt.Errorf("获取公钥失败")}
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	
	// 获取nonce
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return TransactionResult{
			Address: fromAddress.Hex(),
			Error:   fmt.Errorf("获取nonce失败: %v", err),
		}
	}

	// 解析转账金额
	value := new(big.Int)
	if config.Value != "" {
		value, ok = value.SetString(config.Value, 10)
		if !ok {
			return TransactionResult{
				Address: fromAddress.Hex(),
				Error:   fmt.Errorf("解析转账金额失败"),
			}
		}
	}

	// 解析gas价格
	gasPrice := new(big.Int)
	if config.GasPrice != "" {
		gasPrice, ok = gasPrice.SetString(config.GasPrice, 10)
		if !ok {
			return TransactionResult{
				Address: fromAddress.Hex(),
				Error:   fmt.Errorf("解析gas价格失败"),
			}
		}
	} else {
		// 如果没有指定gas价格，则获取建议的gas价格
		gasPrice, err = client.SuggestGasPrice(context.Background())
		if err != nil {
			return TransactionResult{
				Address: fromAddress.Hex(),
				Error:   fmt.Errorf("获取建议gas价格失败: %v", err),
			}
		}
	}

	// 解析目标地址
	toAddress := common.HexToAddress(config.ToAddress)

	// 解析十六进制数据
	var data []byte
	if config.Data != "" {
		dataStr := config.Data
		if len(dataStr) > 2 && dataStr[:2] == "0x" {
			dataStr = dataStr[2:]
		}
		data, err = hex.DecodeString(dataStr)
		if err != nil {
			return TransactionResult{
				Address: fromAddress.Hex(),
				Error:   fmt.Errorf("解析十六进制数据失败: %v", err),
			}
		}
	}

	// 创建交易
	tx := types.NewTransaction(nonce, toAddress, value, config.GasLimit, gasPrice, data)

	// 获取链ID
	chainID := big.NewInt(config.ChainID)

	// 签名交易
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return TransactionResult{
			Address: fromAddress.Hex(),
			Error:   fmt.Errorf("签名交易失败: %v", err),
		}
	}

	// 发送交易
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return TransactionResult{
			Address: fromAddress.Hex(),
			Error:   fmt.Errorf("发送交易失败: %v", err),
		}
	}

	fmt.Printf("协程 %d: 地址 %s 交易已发送，哈希: %s\n", index+1, fromAddress.Hex(), signedTx.Hash().Hex())

	return TransactionResult{
		Address: fromAddress.Hex(),
		TxHash:  signedTx.Hash().Hex(),
	}
} 