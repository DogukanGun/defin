'use client';

import React, { useEffect, useRef } from 'react';
import { Terminal } from 'lucide-react';
import { useBotStore, LogLevel } from '@/store/useBotStore';

const getLevelColor = (level: LogLevel) => {
  switch (level) {
    case 'SUCCESS': return 'text-emerald-400 font-bold';
    case 'ERROR':   return 'text-rose-400 font-bold';
    case 'WARN':    return 'text-amber-400';
    case 'DEBUG':   return 'text-white/30';
    default:        return 'text-sky-400';
  }
};

const LiveLogs = () => {
  const { logs } = useBotStore();
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const isNearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80;
    if (isNearBottom) el.scrollTop = el.scrollHeight;
  }, [logs]);

  return (
    <div className="flex flex-col h-full overflow-hidden">
      <div className="flex items-center px-4 py-2 border-t border-terminal bg-black/40 shrink-0">
        <Terminal size={14} className="text-white/40 mr-2" />
        <span className="text-[10px] font-mono font-bold text-white/60 uppercase tracking-widest">
          LIVE EXECUTION LOGS
        </span>
        <span className="ml-auto text-[10px] font-mono text-white/20">{logs.length} entries</span>
      </div>

      <div
        ref={scrollRef}
        className="flex-grow overflow-y-auto p-4 font-mono text-[11px] space-y-1 bg-black/60"
      >
        {logs.length === 0 ? (
          <span className="text-white/20 italic">Awaiting connection…</span>
        ) : (
          logs.map((log) => (
            <div
              key={log.id}
              className="flex space-x-3 hover:bg-white/[0.02] -mx-4 px-4 py-px transition-colors"
            >
              <span className="text-white/20 whitespace-nowrap shrink-0">[{log.timestamp}]</span>
              <span className={`w-14 shrink-0 whitespace-nowrap ${getLevelColor(log.level)}`}>
                {log.level}
              </span>
              <span className="text-white/75 leading-relaxed break-all">{log.message}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
};

export default LiveLogs;
