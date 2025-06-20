# BSC 批量交易脚本

这个Go脚本可以从JSON配置文件中读取多个私钥，并使用协程同时发送多笔内容相同的交易到BSC（币安智能链）网络。

## 功能特点

- 从JSON文件读取多个私钥
- 使用协程并发发送交易，提高效率
- 支持自定义十六进制数据
- 支持BSC主网和测试网
- 详细的交易结果统计
- 错误处理和重试机制

## 安装依赖

```bash
go mod init bsc-batch-tx
go mod tidy
```

## 配置文件

编辑 `config.json` 文件：

```json
{
  "private_keys": [
    "your_private_key_1_without_0x_prefix",
    "your_private_key_2_without_0x_prefix",
    "your_private_key_3_without_0x_prefix"
  ],
  "rpc_url": "https://bsc-dataseed1.binance.org/",
  "chain_id": 56,
  "to_address": "0x742d35Cc6634C0532925a3b8D5c9E5E5C3D9B9B9",
  "value": "1000000000000000000",
  "gas_limit": 21000,
  "gas_price": "5000000000",
  "data": ""
}
```

### 配置参数说明

- `private_keys`: 私钥数组，不需要0x前缀
- `rpc_url`: BSC RPC节点地址
  - 主网: `https://bsc-dataseed1.binance.org/`
  - 测试网: `https://data-seed-prebsc-1-s1.binance.org:8545/`
- `chain_id`: 链ID
  - 主网: 56
  - 测试网: 97
- `to_address`: 目标地址
- `value`: 转账金额（以wei为单位，1 BNB = 10^18 wei）
- `gas_limit`: Gas限制（普通转账通常为21000）
- `gas_price`: Gas价格（以wei为单位，可留空自动获取）
- `data`: 十六进制数据（可选，用于合约调用）

### 常用金额转换

- 0.001 BNB = `1000000000000000`
- 0.01 BNB = `10000000000000000`
- 0.1 BNB = `100000000000000000`
- 1 BNB = `1000000000000000000`

## 运行脚本

```bash
go run main.go
```

## 示例输出

```
连接到BSC网络，链ID: 56
准备发送 3 笔交易...
协程 1: 地址 0x1234... 交易已发送，哈希: 0xabcd...
协程 2: 地址 0x5678... 交易已发送，哈希: 0xefgh...
协程 3: 地址 0x9abc... 交易已发送，哈希: 0xijkl...

=== 交易结果 ===
✅ 地址 0x1234...: 成功 - 交易哈希: 0xabcd...
✅ 地址 0x5678...: 成功 - 交易哈希: 0xefgh...
✅ 地址 0x9abc...: 成功 - 交易哈希: 0xijkl...

=== 总结 ===
成功: 3 笔
失败: 0 笔
总计: 3 笔
```

## 安全注意事项

1. **私钥安全**: 确保私钥文件的安全，不要提交到代码仓库
2. **测试先行**: 先在测试网测试，确认无误后再在主网使用
3. **余额检查**: 确保每个地址都有足够的BNB支付gas费用
4. **Gas设置**: 根据网络拥堵情况调整gas价格
5. **备份重要**: 使用前请备份重要数据

## 故障排除

### 常见错误

1. **insufficient funds**: 余额不足，需要充值BNB
2. **nonce too low**: nonce值过低，等待前一笔交易确认
3. **gas price too low**: gas价格过低，提高gas_price值
4. **invalid private key**: 私钥格式错误，检查私钥是否正确

### 建议

- 如果遇到nonce冲突，可以增加协程间的延迟时间
- 网络拥堵时适当提高gas价格
- 大批量交易建议分批次执行

## 许可证

MIT License 