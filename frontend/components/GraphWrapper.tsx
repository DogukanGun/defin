'use client';

import dynamic from 'next/dynamic';

// Cytoscape needs browser APIs; must be client-only.
const ArbitrageGraph = dynamic(() => import('./ArbitrageGraph'), { ssr: false });

export default function GraphWrapper() {
  return (
    <div style={{ height: '100%', width: '100%' }}>
      <ArbitrageGraph />
    </div>
  );
}
