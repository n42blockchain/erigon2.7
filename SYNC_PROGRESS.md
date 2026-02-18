# Erigon 2.61.3 → 3.3.7 功能同步进度报告

## 实施状态概览

| 阶段 | 任务 | 状态 | 完成日期 |
|------|------|------|----------|
| 1.1 | CommitteeCount 计算修正 | ✅ 完成 | 2026-02-01 |
| 1.2 | 重试逻辑改进 (指数退避) | ✅ 完成 | 2026-02-01 |
| 1.3 | Discovery v5 默认启用 | ✅ 完成 | 2026-02-01 |
| 2.1 | EIP-7928 BlockAccessList | ⏸️ 暂缓 (需要修改区块头) | - |
| 2.2 | EIP-7825 MaxTxnGasLimit | ✅ 完成 | 2026-02-01 |
| 3.1 | eth_simulateV1 RPC | ✅ 完成 | 2026-02-01 |
| 3.2 | eth_call 区块覆盖 | ✅ 完成 | 2026-02-01 |
| 3.3 | trace_filter 区块标签 | ✅ 完成 | 2026-02-01 |
| 4.1 | P2P 连接死锁修复 | ✅ 完成 | 2026-02-01 |
| 4.2 | 订阅处理死锁修复 | ✅ 完成 | 2026-02-01 |
| 5.1 | DirtySpace 剪枝优化 | ✅ 评估完成 (保持现状) | 2026-02-01 |

---

## 已修改的文件详细列表

### 阶段 1: CL 共识层修复

#### 1.1 CommitteeCount 计算修正
**文件**: `cl/phase1/core/state/cache_accessors.go`
**修改内容**:
- 简化 CommitteeCount 函数，使用单行 max/min 表达式
- 确保结果至少为 1

```go
// 修改前
func (b *CachingBeaconState) CommitteeCount(epoch uint64) uint64 {
    committeCount := min(...)
    if committeCount < 1 {
        committeCount = 1
    }
    return committeCount
}

// 修改后
func (b *CachingBeaconState) CommitteeCount(epoch uint64) uint64 {
    return max(min(b.BeaconConfig().MaxCommitteesPerSlot, uint64(
        len(b.GetActiveValidatorsIndices(epoch)),
    )/b.BeaconConfig().SlotsPerEpoch/b.BeaconConfig().TargetCommitteeSize), 1)
}
```

#### 1.2 重试逻辑改进
**文件**: `cl/phase1/network/gossip_manager.go`
**修改内容**:
- 添加 `getSyncLagSlots()` 辅助函数
- 实现指数退避重连逻辑
- 参数: initialBackoff=1s, maxBackoff=30s, backoffFactor=2

#### 1.3 Discovery v5 默认启用
**文件**: `cmd/utils/flags.go`
**修改内容**:
- DiscoveryV5Flag 默认值改为 true
- 配置加载逻辑改为始终应用 flag 值

```go
DiscoveryV5Flag = cli.BoolFlag{
    Name:  "v5disc",
    Usage: "Enables the RLPx V5 (Topic Discovery) mechanism (enabled by default)",
    Value: true,
}
```

---

### 阶段 2: VM/EVM 改进

#### 2.2 EIP-7825 Osaka MaxTxnGasLimit
**文件**: `turbo/jsonrpc/eth_call.go`
**修改内容**:
- 在 EstimateGas 函数中添加 Osaka 硬分叉 MaxTxnGasLimit 检查

```go
// EIP-7825 (Osaka/Fusaka): Apply per-transaction gas limit cap
if chainConfig.IsOsaka(block.Time()) && hi > params.MaxTxnGasLimit {
    log.Debug("Gas estimation capped by MaxTxnGasLimit (EIP-7825)", "original", hi, "cap", params.MaxTxnGasLimit)
    hi = params.MaxTxnGasLimit
}
```

---

### 阶段 3: RPC 增强

#### 3.1 eth_simulateV1 实现
**新增文件**: `turbo/jsonrpc/eth_simulate.go`
**修改文件**: `turbo/jsonrpc/eth_api.go`

**新增类型**:
- `SimulateBlockOverrides` - 区块参数覆盖
- `SimulateCall` - 单个调用定义
- `SimulateBlockStateCall` - 区块状态调用
- `SimulateCallResult` - 调用结果
- `SimulateError` - 错误信息
- `SimulateBlockResult` - 区块模拟结果

**新增函数**:
- `SimulateV1()` - 主 RPC 方法，支持最多 16 个区块模拟
- `applySimulateBlockOverrides()` - 应用区块覆盖
- `executeSimulateCall()` - 执行单个调用
- `simulateCallToMessage()` - 转换调用为 Message

#### 3.2 eth_call 区块覆盖支持
**修改文件**:
- `turbo/adapter/ethapi/state_overrides.go` - 添加 BlockOverrides 类型
- `turbo/transactions/call.go` - DoCall 函数添加 blockOverrides 参数
- `turbo/jsonrpc/eth_call.go` - Call 函数添加 blockOverrides 参数
- `turbo/jsonrpc/eth_api.go` - 更新接口定义
- `turbo/jsonrpc/eth_api_test.go` - 更新测试
- `turbo/jsonrpc/eth_call_test.go` - 更新测试

**新增类型** (`state_overrides.go`):
```go
type BlockOverrides struct {
    Number        *hexutil.Uint64
    Time          *hexutil.Uint64
    GasLimit      *hexutil.Uint64
    FeeRecipient  *libcommon.Address
    PrevRandao    *libcommon.Hash
    BaseFeePerGas *hexutil.Big
    BlobBaseFee   *hexutil.Big
}
```

---

### 阶段 4: P2P 网络修复

#### 4.1 连接死锁修复
**文件**: `p2p/server.go`
**修改内容**:
- 添加 `checkpointTimeout = 30 * time.Second`
- 添加 `peerOpTimeout = 10 * time.Second`
- `checkpoint()` 函数添加超时保护
- `doPeerOp()` 函数添加超时保护

#### 4.2 订阅处理死锁修复
**文件**: `event/subscription.go`
**修改内容**:
- unsub channel 从无缓冲改为缓冲: `unsub: make(chan struct{}, 1)`
- 添加 `unsubscribeTimeout = 5 * time.Second`
- `Unsubscribe()` 函数使用 select + timeout

---

## 暂缓实施的功能

### EIP-7928 BlockAccessList (阶段 2.1)
**原因**: 需要修改区块头结构，影响范围过大
**建议**: 等待后续版本单独评估

#### 3.3 trace_filter 区块标签支持
**修改文件**:
- `turbo/jsonrpc/trace_filtering.go`
- `turbo/jsonrpc/call_traces_test.go`

**修改内容**:
- 将 `TraceFilterRequest.FromBlock` 和 `ToBlock` 从 `*hexutil.Uint64` 改为 `*rpc.BlockNumber`
- 支持区块标签: "latest", "earliest", "pending", "safe", "finalized"
- Filter 函数使用 `rpchelper.GetBlockNumber` 解析区块标签

```go
// 修改后的 TraceFilterRequest
type TraceFilterRequest struct {
    FromBlock   *rpc.BlockNumber  `json:"fromBlock"`
    ToBlock     *rpc.BlockNumber  `json:"toBlock"`
    // ...
}
```

---

## 验证命令

```bash
# 编译验证
go build ./...

# 单元测试
go test ./cl/... -v
go test ./turbo/jsonrpc/... -v
go test ./p2p/... -v
go test ./event/... -v

# 特定功能测试
go test ./turbo/jsonrpc/... -v -run "EstimateGas"
go test ./turbo/jsonrpc/... -v -run "Call"
```

---

## 注意事项

1. **数据库兼容性**: 所有修改不涉及存储结构变更，DBSchemaVersion 保持 6.1.0
2. **向后兼容**: 新增的 RPC 参数都是可选的 (nil 时使用默认行为)
3. **测试覆盖**: 已更新相关测试文件以适配新的 API 签名
