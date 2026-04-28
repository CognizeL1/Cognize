# Cognize Agent Daemon

A sidecar daemon for Cognize nodes that automatically sends heartbeat transactions to keep the Agent's online status active.

## Features

- Automatic heartbeat transmission
- Configurable interval
- Automatic reconnection
- Error handling and logging

## Configuration

Set the environment variables:

```bash
export COGNIZE_RPC="http://localhost:8545"
export AGENT_PRIVATE_KEY="0x..."
export HEARTBEAT_INTERVAL=100
```

## Usage

```bash
# Build
cargo build --release

# Run
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

For full documentation, see [docs](./docs)