# Cognize Agent 守护进程

Cognize 节点的 Agent 心跳守护进程，自动向链上注册表预编译合约发送心跳交易，保持 Agent 在线状态。

## 功能

- 自动心跳发送
- 可配置间隔
- 自动重连
- 错误处理和日志

## 配置

设置环境变量：

```bash
export COGNIZE_RPC="http://localhost:8545"
export AGENT_PRIVATE_KEY="0x..."
export HEARTBEAT_INTERVAL=100
```

## 使用

```bash
# 构建
cargo build --release

# 运行
./target/release/cognize-agent-daemon
```

## Docker

```bash
docker run -d \
  -e COGNIZE_RPC=http://host.docker.internal:8545 \
  -e AGENT_PRIVATE_KEY=0x... \
  -e HEARTBEAT_INTERVAL=100 \
  cognize/agent-daemon:latest
```

---

完整文档见 [docs](./docs)