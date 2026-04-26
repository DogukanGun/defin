'use client';

import React from 'react';
import { Settings2, Save, Power } from 'lucide-react';
import { useBotStore } from '@/store/useBotStore';

const STRATEGY_LABELS: Record<string, string> = {
  crossDex:    'Cross-DEX',
  triangular:  'Triangular',
  curveStable: 'Curve Stable',
};

const SettingsSidebar = () => {
  const {
    strategies,
    minProfitThreshold,
    toggleStrategy,
    setMinProfitThreshold,
    saveConfiguration,
    status,
    setStatus,
  } = useBotStore();

  return (
    <aside className="w-72 flex flex-col border-l border-terminal bg-black/40 overflow-y-auto p-4 space-y-6 shrink-0">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <Settings2 size={14} className="text-white/50" />
          <h2 className="text-[10px] font-bold font-mono text-white/80 tracking-widest">CONFIGURATION</h2>
        </div>
        <button
          onClick={() => setStatus(status === 'Active' ? 'Simulating' : 'Active')}
          className={`flex items-center space-x-1.5 px-2.5 py-1 rounded text-[10px] font-mono border transition-all ${
            status === 'Active'
              ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400 hover:bg-emerald-500/20'
              : 'bg-amber-500/10 border-amber-500/30 text-amber-400 hover:bg-amber-500/20'
          }`}
        >
          <Power size={11} />
          <span>{status.toUpperCase()}</span>
        </button>
      </div>

      {/* Strategies */}
      <div className="space-y-3">
        <h3 className="text-[9px] font-bold font-mono text-white/30 uppercase tracking-widest">Strategies</h3>
        <div className="space-y-2">
          {(Object.entries(strategies) as [keyof typeof strategies, boolean][]).map(([key, value]) => (
            <div
              key={key}
              className="flex items-center justify-between p-2.5 rounded bg-white/[0.03] border border-terminal"
            >
              <label htmlFor={key} className="text-[11px] font-mono text-white/70 cursor-pointer select-none">
                {STRATEGY_LABELS[key] ?? key}
              </label>
              <button
                id={key}
                role="switch"
                aria-checked={value}
                onClick={() => toggleStrategy(key)}
                className={`w-8 h-4 rounded-full transition-colors relative shrink-0 ${value ? 'bg-emerald-500' : 'bg-white/10'}`}
              >
                <div className={`absolute top-0.5 w-3 h-3 rounded-full bg-white transition-all duration-200 shadow-sm ${value ? 'translate-x-4' : 'translate-x-0.5'}`} />
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* Profit threshold */}
      <div className="space-y-3">
        <h3 className="text-[9px] font-bold font-mono text-white/30 uppercase tracking-widest">Profit Parameters</h3>
        <div className="p-3 space-y-2 rounded bg-white/[0.03] border border-terminal">
          <label className="text-[9px] font-mono text-white/40 block tracking-wider">MIN THRESHOLD (USD)</label>
          <input
            type="number"
            value={minProfitThreshold}
            onChange={(e) => setMinProfitThreshold(parseFloat(e.target.value) || 0)}
            className="w-full bg-black/50 border border-terminal rounded px-2 py-1.5 text-xs font-mono text-white/90 focus:outline-none focus:border-emerald-500/50 transition-colors"
            step="0.1"
            min="0"
          />
        </div>
      </div>

      <div className="mt-auto pt-4">
        <button
          onClick={saveConfiguration}
          className="w-full bg-emerald-600 hover:bg-emerald-500 active:bg-emerald-700 text-white py-2.5 rounded font-mono text-[11px] font-bold transition-all flex items-center justify-center space-x-2 shadow-[0_0_20px_rgba(16,185,129,0.15)] hover:shadow-[0_0_20px_rgba(16,185,129,0.3)]"
        >
          <Save size={13} />
          <span>SAVE CONFIGURATION</span>
        </button>
      </div>
    </aside>
  );
};

export default SettingsSidebar;
