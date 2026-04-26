import { create } from 'zustand';

export type LogLevel = 'INFO' | 'SUCCESS' | 'ERROR' | 'WARN' | 'DEBUG';

export interface TradeStep {
  tokenIn: string;
  tokenOut: string;
  pool: string;
  poolType: string;
}

export interface TradeFlow {
  id: string;
  strategy: string;
  chain: string;
  netProfit: string;
  gasEstimate: number;
  steps: TradeStep[];
  timestamp: string;
}

export interface LogEntry {
  id: string;
  timestamp: string;
  level: LogLevel;
  message: string;
}

export interface TokenBalance {
  symbol: string;
  address: string;
  raw_wei: string;
  decimals: number;
}

export interface WalletBalance {
  walletAddress: string;
  balances: TokenBalance[];
}

export interface PoolInfo {
  address: string;
  type: string;
  chain_id: number;
  token0: string;
  token1: string;
  fee_bps: number;
  reserve0?: string;
  reserve1?: string;
  block?: number;
}

type ConnectionStatus = 'connected' | 'connecting' | 'disconnected';

interface BotState {
  status: 'Active' | 'Simulating';
  gasPrice: number;
  totalProfit: number;
  pools: PoolInfo[];
  connectionStatus: ConnectionStatus;
  recentFlows: TradeFlow[];
  wallet: WalletBalance | null;
  strategies: {
    crossDex: boolean;
    triangular: boolean;
    curveStable: boolean;
  };
  minProfitThreshold: number;
  logs: LogEntry[];

  // Actions
  setStatus: (status: 'Active' | 'Simulating') => void;
  setGasPrice: (price: number) => void;
  setTotalProfit: (profit: number) => void;
  setPools: (pools: PoolInfo[]) => void;
  setConnectionStatus: (s: ConnectionStatus) => void;
  addTradeFlow: (flow: TradeFlow) => void;
  setWallet: (w: WalletBalance) => void;
  toggleStrategy: (strategy: keyof BotState['strategies']) => void;
  setMinProfitThreshold: (threshold: number) => void;
  addLog: (level: LogLevel, message: string) => void;
  saveConfiguration: () => void;
}

export const useBotStore = create<BotState>((set, get) => ({
  status: 'Simulating',
  gasPrice: 0,
  totalProfit: 0,
  pools: [],
  connectionStatus: 'disconnected',
  recentFlows: [],
  wallet: null,
  strategies: {
    crossDex: true,
    triangular: true,
    curveStable: false,
  },
  minProfitThreshold: 1.0,
  logs: [],

  setStatus: (status) => set({ status }),
  setGasPrice: (gasPrice) => set({ gasPrice }),
  setTotalProfit: (totalProfit) => set({ totalProfit }),
  setPools: (pools) => set({ pools }),
  setConnectionStatus: (connectionStatus) => set({ connectionStatus }),
  addTradeFlow: (flow) => set((state) => ({
    recentFlows: [...state.recentFlows, flow].slice(-20),
  })),
  setWallet: (wallet) => set({ wallet }),
  toggleStrategy: (strategy) => set((state) => ({
    strategies: {
      ...state.strategies,
      [strategy]: !state.strategies[strategy],
    }
  })),
  setMinProfitThreshold: (minProfitThreshold) => set({ minProfitThreshold }),
  addLog: (level, message) => set((state) => ({
    logs: [
      ...state.logs,
      {
        id: Math.random().toString(36).substring(7),
        timestamp: new Date().toLocaleTimeString(),
        level,
        message,
      }
    ].slice(-200)
  })),
  saveConfiguration: async () => {
    const { strategies, minProfitThreshold } = get();
    try {
      await fetch('/api/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ strategies, min_profit_threshold: minProfitThreshold }),
      });
      get().addLog('INFO', 'Configuration saved to backend.');
    } catch {
      get().addLog('ERROR', 'Failed to save configuration — backend unreachable.');
    }
  },
}));

// Backend connection — only runs in browser
if (typeof window !== 'undefined') {
  // HTTP API calls use relative URLs — Next.js rewrites them to the backend container.
  // WebSocket connects directly to the backend port using the same hostname the
  // browser used, so it works for both localhost and remote deployments.
  const wsBase = `ws://${window.location.hostname}:8080`;

  // Initial data fetch
  const fetchInitial = async () => {
    const store = useBotStore.getState();
    try {
      const [statusRes, poolsRes] = await Promise.all([
        fetch('/api/status'),
        fetch('/api/pools'),
      ]);
      if (statusRes.ok) {
        const data = await statusRes.json();
        store.setStatus(data.mode === 'execute' ? 'Active' : 'Simulating');
      }
      if (poolsRes.ok) {
        const pools = await poolsRes.json();
        store.setPools(pools ?? []);
      }
    } catch {
      // Backend not yet available — that's fine
    }
  };

  // WebSocket with exponential backoff reconnect
  let retryDelay = 1000;
  const connect = () => {
    const store = useBotStore.getState();
    store.setConnectionStatus('connecting');
    store.addLog('INFO', `Connecting to backend…`);

    const ws = new WebSocket(`${wsBase}/ws`);

    ws.onopen = () => {
      retryDelay = 1000;
      useBotStore.getState().setConnectionStatus('connected');
      useBotStore.getState().addLog('INFO', 'Connected to trading bot backend.');
      fetchInitial();
    };

    ws.onmessage = (evt) => {
      try {
        const ev = JSON.parse(evt.data as string);
        const s = useBotStore.getState();
        switch (ev.type) {
          case 'log': {
            const level = (ev.level as LogLevel) ?? 'INFO';
            s.addLog(level, ev.message ?? '');
            break;
          }
          case 'opportunity': {
            const profitEth = parseFloat(ev.net_profit ?? '0') / 1e18;
            s.addLog('SUCCESS', `[${ev.chain ?? ''}] ${ev.strategy} → +${profitEth.toFixed(6)} ETH`);
            s.setTotalProfit(s.totalProfit + profitEth);
            s.addTradeFlow({
              id: Math.random().toString(36).substring(7),
              strategy: ev.strategy ?? '',
              chain: ev.chain ?? '',
              netProfit: ev.net_profit ?? '0',
              gasEstimate: ev.gas_estimate ?? 0,
              steps: ev.steps ?? [],
              timestamp: new Date().toLocaleTimeString(),
            });
            break;
          }
          case 'status':
            if (ev.gas_price_gwei) s.setGasPrice(ev.gas_price_gwei);
            if (ev.mode) s.setStatus(ev.mode === 'execute' ? 'Active' : 'Simulating');
            break;
          case 'balance':
            if (ev.wallet_address) s.setWallet({ walletAddress: ev.wallet_address, balances: ev.balances ?? [] });
            break;
        }
      } catch {
        // Ignore malformed messages
      }
    };

    ws.onclose = () => {
      useBotStore.getState().setConnectionStatus('disconnected');
      useBotStore.getState().addLog('WARN', `Backend disconnected. Reconnecting in ${retryDelay / 1000}s…`);
      setTimeout(() => {
        retryDelay = Math.min(retryDelay * 2, 30000);
        connect();
      }, retryDelay);
    };

    ws.onerror = () => {
      ws.close();
    };
  };

  connect();
}
