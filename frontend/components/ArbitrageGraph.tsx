'use client';

// Cytoscape is imported synchronously here — safe because this component
// is only ever loaded via dynamic({ ssr: false }) in GraphWrapper.
import cytoscape from 'cytoscape';
import { useEffect, useRef } from 'react';
import { useBotStore, TradeFlow, TradeStep } from '@/store/useBotStore';

// ─── Token metadata ───────────────────────────────────────────────────────────

const TOKEN_META: Record<string, { name: string; bg: string; border: string }> = {
  '0x82af49447d8a07e3bd95bd0d56f35241523fbab1': { name: 'WETH',   bg: '#1d4ed8', border: '#93c5fd' },
  '0xff970a61a04b1ca14834a43f5de4533ebddb5cc8': { name: 'USDC.e', bg: '#15803d', border: '#86efac' },
  '0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9': { name: 'USDT',   bg: '#166534', border: '#4ade80' },
  '0x912ce59144191c1204e64559fe8253a0e49e6548': { name: 'ARB',    bg: '#6d28d9', border: '#d8b4fe' },
  '0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f': { name: 'WBTC',  bg: '#c2410c', border: '#fdba74' },
  '0xaf88d065e77c8cc2239327c5edb3a432268e5831': { name: 'USDC',   bg: '#15803d', border: '#86efac' },
};

function tok(addr: string) {
  return TOKEN_META[addr.toLowerCase()]?.name ?? addr.slice(0, 6) + '…';
}

// ─── Pool → DEX label ─────────────────────────────────────────────────────────

const POOL_DEX: Record<string, string> = {
  '0x905dfcd5649217c42684f23958568e533c711aa3': 'SushiSwap',
  '0x84652bb2539513baf36e225c930fdd8eaa63ce27': 'Camelot',
  '0xcb0e5bfa72bbb4d16ab5aa0c60601c438f04b4ad': 'SushiSwap',
  '0x1c31fb3359357f6436565ccbb2db23b1a6e8a9bd': 'SushiSwap',
  '0xc31e54c7a869b9fcbecc14363cf510d1c41fa443': 'Uniswap V3',
  '0xc6f780497a95e246eb9449f5e4770916dcd6396a': 'Uniswap V3',
  '0x2f5e87c9312fa29aed5c179e456625d79015299c': 'Uniswap V3',
  '0xe56e04ded3a3bc5fe2b9f6abc5c00a0a2deea57': 'SushiSwap',
};

// ─── Static pool topology (mirrors config.yaml) ───────────────────────────────

const STATIC_POOLS = [
  { address: '0x905dfcd5649217c42684f23958568e533c711aa3', type: 'uniswap_v2',
    token0: '0x82af49447d8a07e3bd95bd0d56f35241523fbab1', token1: '0xff970a61a04b1ca14834a43f5de4533ebddb5cc8' },
  { address: '0x84652bb2539513baf36e225c930fdd8eaa63ce27', type: 'uniswap_v2',
    token0: '0x82af49447d8a07e3bd95bd0d56f35241523fbab1', token1: '0xff970a61a04b1ca14834a43f5de4533ebddb5cc8' },
  { address: '0xcb0e5bfa72bbb4d16ab5aa0c60601c438f04b4ad', type: 'uniswap_v2',
    token0: '0x82af49447d8a07e3bd95bd0d56f35241523fbab1', token1: '0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9' },
  { address: '0x1c31fb3359357f6436565ccbb2db23b1a6e8a9bd', type: 'uniswap_v2',
    token0: '0xff970a61a04b1ca14834a43f5de4533ebddb5cc8', token1: '0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9' },
  { address: '0xc31e54c7a869b9fcbecc14363cf510d1c41fa443', type: 'uniswap_v3',
    token0: '0x82af49447d8a07e3bd95bd0d56f35241523fbab1', token1: '0xff970a61a04b1ca14834a43f5de4533ebddb5cc8' },
  { address: '0xc6f780497a95e246eb9449f5e4770916dcd6396a', type: 'uniswap_v3',
    token0: '0x912ce59144191c1204e64559fe8253a0e49e6548', token1: '0x82af49447d8a07e3bd95bd0d56f35241523fbab1' },
  { address: '0x2f5e87c9312fa29aed5c179e456625d79015299c', type: 'uniswap_v3',
    token0: '0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f', token1: '0x82af49447d8a07e3bd95bd0d56f35241523fbab1' },
  { address: '0xe56e04ded3a3bc5fe2b9f6abc5c00a0a2deea57', type: 'uniswap_v2',
    token0: '0x912ce59144191c1204e64559fe8253a0e49e6548', token1: '0xff970a61a04b1ca14834a43f5de4533ebddb5cc8' },
];

// ─── Demo flows (USDC.e as origin) ───────────────────────────────────────────

const DEMO_FLOWS: { strategy: string; netProfit: string; steps: TradeStep[] }[] = [
  {
    strategy: 'cross_dex',
    netProfit: '2100000000000000',
    steps: [
      { tokenIn: '0xff970a61a04b1ca14834a43f5de4533ebddb5cc8', tokenOut: '0x82af49447d8a07e3bd95bd0d56f35241523fbab1',
        pool: '0x905dfcd5649217c42684f23958568e533c711aa3', poolType: 'uniswap_v2' },
      { tokenIn: '0x82af49447d8a07e3bd95bd0d56f35241523fbab1', tokenOut: '0xff970a61a04b1ca14834a43f5de4533ebddb5cc8',
        pool: '0x84652bb2539513baf36e225c930fdd8eaa63ce27', poolType: 'uniswap_v2' },
    ],
  },
  {
    strategy: 'triangular',
    netProfit: '3400000000000000',
    steps: [
      { tokenIn: '0xff970a61a04b1ca14834a43f5de4533ebddb5cc8', tokenOut: '0x82af49447d8a07e3bd95bd0d56f35241523fbab1',
        pool: '0x905dfcd5649217c42684f23958568e533c711aa3', poolType: 'uniswap_v2' },
      { tokenIn: '0x82af49447d8a07e3bd95bd0d56f35241523fbab1', tokenOut: '0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9',
        pool: '0xcb0e5bfa72bbb4d16ab5aa0c60601c438f04b4ad', poolType: 'uniswap_v2' },
      { tokenIn: '0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9', tokenOut: '0xff970a61a04b1ca14834a43f5de4533ebddb5cc8',
        pool: '0x1c31fb3359357f6436565ccbb2db23b1a6e8a9bd', poolType: 'uniswap_v2' },
    ],
  },
  {
    strategy: 'triangular (V3)',
    netProfit: '1800000000000000',
    steps: [
      { tokenIn: '0xff970a61a04b1ca14834a43f5de4533ebddb5cc8', tokenOut: '0x82af49447d8a07e3bd95bd0d56f35241523fbab1',
        pool: '0xc31e54c7a869b9fcbecc14363cf510d1c41fa443', poolType: 'uniswap_v3' },
      { tokenIn: '0x82af49447d8a07e3bd95bd0d56f35241523fbab1', tokenOut: '0x912ce59144191c1204e64559fe8253a0e49e6548',
        pool: '0xc6f780497a95e246eb9449f5e4770916dcd6396a', poolType: 'uniswap_v3' },
      { tokenIn: '0x912ce59144191c1204e64559fe8253a0e49e6548', tokenOut: '0xff970a61a04b1ca14834a43f5de4533ebddb5cc8',
        pool: '0xe56e04ded3a3bc5fe2b9f6abc5c00a0a2deea57', poolType: 'uniswap_v2' },
    ],
  },
];

// ─── Build graph elements ─────────────────────────────────────────────────────

function buildElements(): cytoscape.ElementDefinition[] {
  const tokenSet = new Set<string>();
  STATIC_POOLS.forEach((p) => { tokenSet.add(p.token0); tokenSet.add(p.token1); });

  const nodes: cytoscape.ElementDefinition[] = Array.from(tokenSet).map((addr) => {
    const m = TOKEN_META[addr.toLowerCase()] ?? { name: tok(addr), bg: '#334155', border: '#64748b' };
    return {
      data: { id: addr, label: m.name },
      style: { 'background-color': m.bg, 'border-color': m.border },
    };
  });

  const edges: cytoscape.ElementDefinition[] = STATIC_POOLS.map((p) => ({
    data: {
      id: `edge-${p.address}`,
      source: p.token0,
      target: p.token1,
      label: POOL_DEX[p.address] ?? (p.type === 'uniswap_v3' ? 'Uniswap V3' : 'Uniswap V2'),
    },
  }));

  return [...nodes, ...edges];
}

// ─── Cytoscape stylesheet ─────────────────────────────────────────────────────

const CY_STYLE = [
  {
    selector: 'node',
    style: {
      'border-width': 2.5,
      label: 'data(label)',
      color: '#ffffff',
      'font-family': '"JetBrains Mono", "Fira Code", monospace',
      'font-size': 12,
      'font-weight': 700,
      'text-valign': 'center',
      'text-halign': 'center',
      width: 70,
      height: 70,
    },
  },
  {
    selector: 'edge',
    style: {
      'curve-style': 'bezier',
      'line-color': '#334155',
      label: 'data(label)',
      color: '#94a3b8',
      'font-size': 10,
      'font-family': '"JetBrains Mono", monospace',
      'text-rotation': 'autorotate',
      'text-background-color': '#0f172a',
      'text-background-opacity': 1,
      'text-background-padding': '3px',
      width: 1.5,
    },
  },
  {
    selector: '.flash-edge',
    style: {
      'line-color': '#10b981',
      'target-arrow-color': '#10b981',
      'target-arrow-shape': 'triangle',
      width: 4,
    },
  },
  {
    selector: '.flash-node',
    style: {
      'border-color': '#10b981',
      'border-width': 4,
    },
  },
] as any;

// ─── Component ────────────────────────────────────────────────────────────────

export default function ArbitrageGraph() {
  const containerRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<cytoscape.Core | null>(null);
  const { recentFlows, addTradeFlow, connectionStatus } = useBotStore();
  const lastFlowId = useRef('');
  const demoIndex = useRef(0);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;

    let cy: cytoscape.Core | null = null;
    let destroyed = false;

    const init = () => {
      if (destroyed || cy) return;
      if (el.offsetWidth === 0 || el.offsetHeight === 0) return;

      cy = cytoscape({
        container: el,
        elements: buildElements(),
        style: CY_STYLE as any,
        layout: { name: 'circle', animate: false, padding: 40 } as any,
        userZoomingEnabled: true,
        userPanningEnabled: true,
        boxSelectionEnabled: false,
        minZoom: 0.2,
        maxZoom: 4,
      });
      cyRef.current = cy;
      cy.resize();
      cy.fit(undefined, 40);
    };

    // Defer init to rAF so the browser has finished computing flex layout
    const raf = requestAnimationFrame(init);

    const ro = new ResizeObserver(() => {
      init();
      cy?.resize();
      cy?.fit(undefined, 40);
    });
    ro.observe(el);

    return () => {
      destroyed = true;
      cancelAnimationFrame(raf);
      ro.disconnect();
      cy?.destroy();
      cy = null;
      cyRef.current = null;
    };
  }, []);

  // ── Highlight trade path ────────────────────────────────────────────────────
  useEffect(() => {
    const cy = cyRef.current;
    if (!cy || recentFlows.length === 0) return;

    const latest = recentFlows[recentFlows.length - 1];
    if (latest.id === lastFlowId.current) return;
    lastFlowId.current = latest.id;

    cy.elements('.flash-edge, .flash-node').removeClass('flash-edge flash-node');

    latest.steps.forEach((step) => {
      cy.getElementById(step.tokenIn.toLowerCase()).addClass('flash-node');
      cy.getElementById(step.tokenOut.toLowerCase()).addClass('flash-node');
      cy.getElementById(`edge-${step.pool.toLowerCase()}`).addClass('flash-edge');
    });

    const t = setTimeout(() => {
      cy?.elements('.flash-edge, .flash-node').removeClass('flash-edge flash-node');
    }, 4000);
    return () => clearTimeout(t);
  }, [recentFlows]);

  // ── Demo flows every 8 s ────────────────────────────────────────────────────
  useEffect(() => {
    const id = setInterval(() => {
      const tpl = DEMO_FLOWS[demoIndex.current % DEMO_FLOWS.length];
      demoIndex.current += 1;
      addTradeFlow({
        id: Math.random().toString(36).substring(7),
        strategy: tpl.strategy + ' (sim)',
        chain: 'arbitrum',
        netProfit: tpl.netProfit,
        gasEstimate: 350000,
        steps: tpl.steps,
        timestamp: new Date().toLocaleTimeString(),
      });
    }, 8000);
    return () => clearInterval(id);
  }, [addTradeFlow]);

  const latestFlow: TradeFlow | null = recentFlows.length > 0 ? recentFlows[recentFlows.length - 1] : null;

  return (
    <div className="flex flex-col h-full w-full overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-zinc-800 shrink-0 bg-black/40">
        <div className="flex items-center gap-3">
          <span className="text-xs font-bold text-zinc-200 font-mono tracking-widest uppercase">
            DeFi Arbitrage Multigraph
          </span>
          <span className="text-[10px] text-zinc-600 font-mono">
            {STATIC_POOLS.length} pools · drag to explore
          </span>
        </div>
        {latestFlow && (
          <div className="flex items-center gap-2">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse" />
            <span className="text-[10px] font-mono text-emerald-400">
              {latestFlow.strategy} · +{(parseFloat(latestFlow.netProfit) / 1e18).toFixed(5)} ETH
            </span>
            <span className="text-[10px] font-mono text-zinc-600">{latestFlow.timestamp}</span>
          </div>
        )}
      </div>

      {/* Canvas + sidebar */}
      <div className="relative flex-1 overflow-hidden">
        <div
          ref={containerRef}
          className="absolute inset-0"
          style={{ backgroundColor: '#0d1117' }}
        />

        {recentFlows.length > 0 && (
          <div className="absolute right-0 top-0 bottom-0 w-52 border-l border-zinc-800/80 bg-black/75 backdrop-blur-sm overflow-y-auto flex flex-col z-10">
            <div className="text-[10px] font-mono text-zinc-500 uppercase tracking-widest px-3 pt-3 pb-1 shrink-0">
              Recent Trades
            </div>
            <div className="flex flex-col gap-1.5 px-2 pb-2">
              {[...recentFlows].reverse().slice(0, 12).map((flow) => {
                const hops = flow.steps.map((s) => tok(s.tokenIn));
                if (flow.steps.length > 0) hops.push(tok(flow.steps[flow.steps.length - 1].tokenOut));
                const profit = (parseFloat(flow.netProfit) / 1e18).toFixed(5);
                return (
                  <div key={flow.id} className="bg-zinc-900 border border-zinc-800 rounded p-2 text-[10px] font-mono">
                    <div className="flex items-center justify-between mb-0.5">
                      <span className="text-emerald-400 font-bold truncate">{flow.strategy}</span>
                      <span className="text-emerald-300 ml-1 shrink-0">+{profit}</span>
                    </div>
                    <div className="text-zinc-400 truncate">{hops.join(' → ')}</div>
                    <div className="text-zinc-600 mt-0.5">{flow.timestamp}</div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {connectionStatus === 'disconnected' && (
          <div className="absolute bottom-3 left-3 z-10">
            <div className="bg-zinc-900/90 border border-zinc-700 rounded px-3 py-1.5 text-[10px] font-mono text-amber-400">
              ⚠ Backend offline · showing demo simulation
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
