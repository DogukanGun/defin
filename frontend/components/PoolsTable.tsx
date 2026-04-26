'use client';

import React from 'react';
import { Activity } from 'lucide-react';
import { useBotStore, PoolInfo } from '@/store/useBotStore';

const CHAIN_NAMES: Record<number, string> = {
  42161: 'Arbitrum',
  8453:  'Base',
  137:   'Polygon',
  1:     'Ethereum',
};

const TYPE_LABELS: Record<string, string> = {
  uniswap_v2: 'V2',
  uniswap_v3: 'V3',
  curve:      'CRV',
};

function shortAddr(addr: string) {
  return addr.slice(0, 6) + '…' + addr.slice(-4);
}

function formatReserve(val?: string) {
  if (!val || val === '0') return '0';
  const n = BigInt(val);
  if (n === BigInt(0)) return '0';
  const eth = Number(n) / 1e18;
  if (eth >= 1000) return eth.toFixed(0);
  if (eth >= 1) return eth.toFixed(3);
  const usdc = Number(n) / 1e6;
  if (usdc >= 1) return usdc.toFixed(2);
  return val.slice(0, 8) + '…';
}

const PoolRow = ({ pool }: { pool: PoolInfo }) => (
  <tr className="border-b border-terminal hover:bg-white/[0.02] transition-colors group">
    <td className="px-4 py-2 text-[10px] font-mono">
      <span className={`px-1.5 py-0.5 rounded text-[9px] font-bold ${
        pool.chain_id === 42161 ? 'bg-blue-500/10 text-blue-400'
        : pool.chain_id === 8453 ? 'bg-purple-500/10 text-purple-400'
        : 'bg-pink-500/10 text-pink-400'
      }`}>
        {CHAIN_NAMES[pool.chain_id] ?? pool.chain_id}
      </span>
    </td>
    <td className="px-4 py-2 text-[10px] font-mono">
      <span className="px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-400 text-[9px] font-bold">
        {TYPE_LABELS[pool.type] ?? pool.type}
      </span>
    </td>
    <td className="px-4 py-2 text-[10px] font-mono text-white/70">
      <span title={pool.token0}>{shortAddr(pool.token0)}</span>
      <span className="text-white/30 mx-1">/</span>
      <span title={pool.token1}>{shortAddr(pool.token1)}</span>
    </td>
    <td className="px-4 py-2 text-[10px] font-mono text-white/40" title={pool.address}>
      {shortAddr(pool.address)}
    </td>
    <td className="px-4 py-2 text-[10px] font-mono text-right text-white/60">
      {formatReserve(pool.reserve0)}
    </td>
    <td className="px-4 py-2 text-[10px] font-mono text-right text-white/60">
      {formatReserve(pool.reserve1)}
    </td>
    <td className="px-4 py-2 text-[10px] font-mono text-right text-white/30">
      {pool.block ? `#${pool.block.toLocaleString()}` : '—'}
    </td>
  </tr>
);

const PoolsTable = () => {
  const { pools, connectionStatus } = useBotStore();

  return (
    <div className="flex-grow flex flex-col overflow-hidden border-b border-terminal bg-black/20">
      <div className="flex items-center px-4 py-2 border-b border-terminal bg-black/40 shrink-0">
        <Activity size={14} className="text-white/40 mr-2" />
        <span className="text-[10px] font-mono font-bold text-white/60 uppercase tracking-widest">
          MONITORED POOLS
        </span>
        {pools.length > 0 && (
          <span className="ml-auto text-[10px] font-mono text-white/30">
            {pools.length} pool{pools.length !== 1 ? 's' : ''}
          </span>
        )}
      </div>

      <div className="flex-grow overflow-y-auto">
        {pools.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full py-12 text-center">
            <div className={`w-2 h-2 rounded-full mb-4 ${
              connectionStatus === 'connecting' ? 'bg-amber-500 animate-pulse' : 'bg-white/10'
            }`} />
            <p className="text-xs font-mono text-white/30">
              {connectionStatus === 'connected'
                ? 'No pools registered in config.yaml'
                : connectionStatus === 'connecting'
                ? 'Connecting to backend…'
                : 'Backend offline — start the Go bot'}
            </p>
          </div>
        ) : (
          <table className="w-full">
            <thead className="sticky top-0 bg-black/80 backdrop-blur-sm">
              <tr className="border-b border-terminal">
                {['Chain', 'Type', 'Pair', 'Address', 'Reserve 0', 'Reserve 1', 'Block'].map((h, i) => (
                  <th key={h} className={`px-4 py-2 text-[9px] font-mono font-bold text-white/30 uppercase tracking-wider ${i >= 4 ? 'text-right' : 'text-left'}`}>
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {pools.map((pool) => (
                <PoolRow key={pool.address} pool={pool} />
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
};

export default PoolsTable;
