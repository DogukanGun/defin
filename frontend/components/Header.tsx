'use client';

import React from 'react';
import { Activity, Fuel, DollarSign, Wifi, WifiOff, Wallet } from 'lucide-react';
import { useBotStore, TokenBalance } from '@/store/useBotStore';

function formatBalance(tb: TokenBalance): string {
  const raw = BigInt(tb.raw_wei || '0');
  const divisor = BigInt(10 ** tb.decimals);
  const whole = raw / divisor;
  const frac = raw % divisor;
  const fracStr = frac.toString().padStart(tb.decimals, '0').slice(0, 4);
  return `${whole}.${fracStr}`;
}

// Only show tokens with non-zero balance
function nonZero(b: TokenBalance) {
  return BigInt(b.raw_wei || '0') > 0n;
}

const Header = () => {
  const { status, gasPrice, totalProfit, connectionStatus, wallet } = useBotStore();
  const shownBalances = wallet?.balances.filter(nonZero) ?? [];

  return (
    <header className="border-b border-terminal bg-black/60 backdrop-blur-sm z-10 shrink-0">
      {/* ── Main row ── */}
      <div className="h-14 flex items-center justify-between px-6">
        <div className="flex items-center space-x-4">
          <h1 className="text-sm font-bold tracking-widest text-white/90 uppercase">
            TRADER<span className="text-emerald-500">_BOT</span>
          </h1>
          <div className="h-4 w-px bg-white/20" />
          <div className="flex items-center space-x-2">
            <div className={`h-2 w-2 rounded-full animate-pulse ${
              status === 'Active'
                ? 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.6)]'
                : 'bg-amber-500 shadow-[0_0_8px_rgba(245,158,11,0.5)]'
            }`} />
            <span className="text-xs font-mono text-white/60 tracking-tighter">
              {status.toUpperCase()}
            </span>
          </div>
        </div>

        <div className="flex items-center space-x-8">
          {gasPrice > 0 && (
            <div className="flex items-center space-x-2">
              <Fuel size={14} className="text-white/40" />
              <span className="text-xs font-mono text-white/80">
                {gasPrice.toFixed(1)} <span className="text-white/40 text-[10px]">GWEI</span>
              </span>
            </div>
          )}

          <div className="flex items-center space-x-2">
            <DollarSign size={14} className="text-emerald-500/60" />
            <span className="text-xs font-mono font-bold text-emerald-400 transition-all duration-300">
              {totalProfit.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 4 })}
              <span className="ml-1 text-[10px] text-emerald-500/40">ETH profit</span>
            </span>
          </div>

          <div className="flex items-center space-x-1.5">
            {connectionStatus === 'connected' ? (
              <Wifi size={13} className="text-emerald-400" />
            ) : connectionStatus === 'connecting' ? (
              <Wifi size={13} className="text-amber-400 animate-pulse" />
            ) : (
              <WifiOff size={13} className="text-rose-400" />
            )}
            <span className={`text-[10px] font-mono ${
              connectionStatus === 'connected' ? 'text-emerald-400/80'
              : connectionStatus === 'connecting' ? 'text-amber-400/80'
              : 'text-rose-400/80'
            }`}>
              {connectionStatus.toUpperCase()}
            </span>
          </div>
        </div>
      </div>

      {/* ── Balance row (only shown when wallet is known) ── */}
      {wallet && (
        <div className="flex items-center gap-4 px-6 py-1.5 border-t border-white/5 bg-black/30 overflow-x-auto">
          <div className="flex items-center gap-1.5 shrink-0">
            <Wallet size={11} className="text-white/30" />
            <span className="text-[10px] font-mono text-white/30">
              {wallet.walletAddress.slice(0, 6)}…{wallet.walletAddress.slice(-4)}
            </span>
          </div>
          <div className="h-3 w-px bg-white/10 shrink-0" />
          {shownBalances.length === 0 ? (
            <span className="text-[10px] font-mono text-white/20">No token balances</span>
          ) : (
            shownBalances.map((tb) => (
              <div key={tb.symbol} className="flex items-center gap-1.5 shrink-0">
                <span className={`text-[10px] font-mono font-bold ${
                  tb.symbol === 'ETH' || tb.symbol === 'WETH' ? 'text-blue-400'
                  : tb.symbol.startsWith('USDC') || tb.symbol === 'USDT' ? 'text-green-400'
                  : tb.symbol === 'ARB' ? 'text-violet-400'
                  : tb.symbol === 'WBTC' ? 'text-orange-400'
                  : 'text-white/60'
                }`}>
                  {formatBalance(tb)}
                </span>
                <span className="text-[10px] font-mono text-white/30">{tb.symbol}</span>
              </div>
            ))
          )}
        </div>
      )}
    </header>
  );
};

export default Header;
