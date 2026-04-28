# Cognize TypeScript SDK

TypeScript SDK for interacting with the Cognize AI Agent blockchain. Built on [ethers.js v6](https://docs.ethers.org/v6/).

Use your own node's EVM JSON-RPC endpoint, or a public one when available.

## Chain Parameters

| Item | Value |
|------|-------|
| Cosmos Chain ID | `cognize_8210-1` |
| EVM Chain ID | `8210` |
| Native Token | `ACognize` |

## Installation

```bash
cd sdk/typescript
npm install
```

## Quick Start

```typescript
import { AgentClient } from "@cognize/sdk";

const client = new AgentClient("http://localhost:8545");
await client.connect("your-private-key");

// Register as agent
const tx = await client.registerAgent("nlp,reasoning", "10");
await tx.wait();

// Add stake
const addStakeTx = await client.addStake("100");
await addStakeTx.wait();
```

## Examples

### Query Agent Info

```typescript
const agent = await client.getAgent("0x...");
console.log(agent.address, agent.reputation, agent.status);
```

### Send Heartbeat

```typescript
const tx = await client.heartbeat();
await tx.wait();
```

---

For full documentation, see [SDK Reference](./docs/reference.md)