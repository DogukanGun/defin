# DeFi Arbitrage Trader

A high-performance MEV arbitrage bot that scans multiple blockchains in real-time, detects profitable trading opportunities across decentralized exchanges, and executes trades atomically — with optional flash loan support and Flashbots MEV protection.

---

## What It Does

The bot connects to Arbitrum, Base, and Polygon, subscribes to new blocks, and on each block:

1. Fetches the state of all configured pools via **Multicall3** (batched RPC calls)
2. Runs all enabled arbitrage strategies concurrently
3. Ranks opportunities by net profit (gross profit minus gas cost)
4. Simulates the top opportunity via `eth_call`
5. Optionally executes on-chain — either via a standard transaction or a **Flashbots bundle**

All activity streams in real-time to a web dashboard over WebSocket.

---

## Arbitrage Strategies

| Strategy | Description |
|---|---|
| **Cross-DEX** | Detects price differences for the same token pair across two pools (e.g. V2 vs V3). Uses ternary search to find the optimal input amount. |
| **Triangular** | Multi-hop cycle across 3+ pools (e.g. USDC → WETH → USDT → USDC). Graph-based path finding with configurable max hops. |
| **Curve Stable** | Exploits price imbalances in Curve stablecoin pools. |
| **Liquidation** | Monitors Aave V3 health factors and identifies liquidatable accounts for profit. |

---

## Execution Modes

- **Simulate** — Runs the full pipeline but only dry-runs transactions via `eth_call`. Safe for testing.
- **Execute** — Signs and submits real transactions.
  - Standard sender via the configured wallet
  - **Flashbots** mode — bundles the transaction and submits to a Flashbots relay for MEV protection and front-run prevention

Flash loan execution wraps any opportunity in an **Aave V3 flash loan**, enabling capital-efficient arbitrage without holding a large position.

---

## Architecture

```
cmd/trader/main.go          Entry point: loads config, starts all services
internal/
  chain/                    RPC clients, block subscription, Multicall3 batching
  pool/                     Unified pool abstraction (Uniswap V2, V3, Curve)
  strategy/                 Arbitrage strategies + evaluator
  executor/                 Transaction builder, simulator, Flashbots sender
  engine/                   Main per-chain block processing loop
  api/                      HTTP + WebSocket server, real-time event hub
  math/                     AMM formulas (V2, V3, Curve), profit calculation
  monitor/                  Aave health factor monitor (liquidation strategy)
  telemetry/                Prometheus metrics, structured JSON logging
  config/                   YAML config loader with env var interpolation
contracts/FlashArb.sol      On-chain atomic flash loan + swap executor
frontend/                   Next.js 16 monitoring dashboard
```

**Block processing loop (per chain):**

```
New block → Multicall3 state fetch (rate-limited) → Evaluate strategies (concurrent)
→ Rank by net profit → Simulate top opportunity → Execute or broadcast
```

---

## Smart Contract

`FlashArb.sol` is deployed on-chain and called by the executor. It:

1. Borrows tokens from Aave V3 via `flashLoan()`
2. Executes the encoded swap path across multiple pools
3. Repays the flash loan + 0.05% premium
4. Sends profit to the owner wallet

The bot encodes the full swap path off-chain and passes it as calldata — the contract executes atomically.

---

## Dashboard

The frontend is a **Next.js 16** app that connects to the bot over WebSocket and shows:

- Live opportunity feed with profit, gas cost, swap path, and chain
- Pool topology graph (Cytoscape.js) with animated trade flows
- Sortable pool table (address, type, chain, token pair, reserves)
- Wallet balance tracker per token
- Settings sidebar to adjust execution mode, profit threshold, and gas limits

---

## Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.22+ |
| Smart Contract | Solidity 0.8.20 (Aave V3 flash loans) |
| Frontend | Next.js 16, TypeScript, Tailwind CSS 4, Zustand, Cytoscape.js |
| Observability | Prometheus metrics (`:9090`), structured slog JSON |
| Deployment | Docker + docker-compose |

---

## Configuration

All settings live in `config.yaml`. Environment variables are interpolated at load time.

**Chains:**
```yaml
chains:
  - name: arbitrum
    chain_id: 42161
    rpc_http: ${ARBITRUM_RPC_HTTP}
    rpc_ws: ${ARBITRUM_RPC_WS}
    block_time_ms: 250
    max_gas_price_gwei: 0.05
```

**Pools:**
```yaml
pools:
  - address: 0x905dfCD5649217c42684f23958568e533c711Aa3
    type: uniswap_v2
    chain_id: 42161
    token0: 0x82aF49447D8a07e3bd95BD0d56f35241523fBab1  # WETH
    token1: 0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8  # USDC.e
    fee_bps: 30
```

**Execution:**
```yaml
execution:
  mode: simulate          # simulate | execute
  use_flashbots: false
  use_flash_loan: true
  max_position_usd: 10000
  slippage_bps: 50
```

---

## Getting Started

**Interactive setup (recommended):**

```bash
./start.sh
```

This will prompt for your RPC endpoints and private key, write a `.env` file, and start the bot.

**Manual:**

```bash
cp .env.example .env
# fill in your RPC URLs and PRIVATE_KEY

make build
make run
```

**Docker:**

```bash
docker-compose up --build
```

| Service | Port |
|---|---|
| Backend API + WebSocket | `8080` |
| Prometheus metrics | `9090` |
| Frontend dashboard | `3001` |

---

## Development

```bash
make test          # Full test suite with race detector
make test-short    # Fast tests
make bench         # Benchmark AMM math
make lint          # golangci-lint
make abigen        # Regenerate Go bindings from ABIs
```

---

## Environment Variables

| Variable | Description |
|---|---|
| `ARBITRUM_RPC_HTTP` | Arbitrum HTTP RPC URL |
| `ARBITRUM_RPC_WS` | Arbitrum WebSocket RPC URL |
| `BASE_RPC_HTTP` | Base HTTP RPC URL |
| `BASE_RPC_WS` | Base WebSocket RPC URL |
| `POLYGON_RPC_HTTP` | Polygon HTTP RPC URL |
| `POLYGON_RPC_WS` | Polygon WebSocket RPC URL |
| `PRIVATE_KEY` | Wallet private key (without 0x prefix) |
| `EXECUTION_MODE` | `simulate` or `execute` |

---

## Disclaimer

This software is for educational and research purposes. Running an arbitrage bot with real funds carries risk. Always test in simulate mode first and understand the code before executing live transactions.
