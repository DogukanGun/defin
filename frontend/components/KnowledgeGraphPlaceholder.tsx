'use client';

import React from 'react';
import { Share2 } from 'lucide-react';

const KnowledgeGraphPlaceholder = () => {
  return (
    <div className="flex-grow flex flex-col items-center justify-center border-r border-terminal bg-black/20">
      <div className="p-8 rounded-full border border-dashed border-white/10 bg-white/[0.02] flex flex-col items-center animate-pulse">
        <Share2 size={48} className="text-white/20 mb-4" />
        <h2 className="text-sm font-mono text-white/40 tracking-wider">
          KNOWLEDGE GRAPH ENGINE
        </h2>
        <p className="text-[10px] font-mono text-white/20 mt-2">
          (CYTOSCAPE / REACT-FLOW INTEGRATION PENDING)
        </p>
      </div>
    </div>
  );
};

export default KnowledgeGraphPlaceholder;
