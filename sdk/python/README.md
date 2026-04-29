# Cognize Python SDK

Python SDK for interacting with the Cognize AI Agent blockchain.

Use your own node's EVM JSON-RPC endpoint, or a public one when available.

## Chain Parameters

| Item | Value |
|------|-------|
| Cosmos Chain ID | `cognize_8210-1` |
| EVM Chain ID | `8210` |
| Native Token | `COGNIZE` |

## Installation

```bash
pip install cognize-sdk
```

## Quick Start

```python
from cognize import AgentClient

client = AgentClient("http://localhost:8545")
client.set_account("your-private-key")

# Register as agent
tx = client.register_agent("nlp,reasoning", stake_cognize=10)
tx.wait()

# Add stake
add_stake = client.add_stake(100)
add_stake.wait()
```

## Examples

### Query Agent Info

```python
agent = client.get_agent("0x...")
print(agent.address, agent.reputation, agent.status)
```

### Send Heartbeat

```python
tx = client.heartbeat()
tx.wait()
```

---

For full documentation, see [SDK Reference](./docs/reference.md)