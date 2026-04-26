import Header from '@/components/Header';
import GraphWrapper from '@/components/GraphWrapper';
import PoolsTable from '@/components/PoolsTable';
import SettingsSidebar from '@/components/SettingsSidebar';
import LiveLogs from '@/components/LiveLogs';

export default function Home() {
  return (
    <main className="flex flex-col h-screen w-full text-foreground selection:bg-emerald-500/20">
      <Header />

      <div className="flex flex-1 overflow-hidden" style={{ minHeight: 0 }}>
        {/* Left column: graph (top) + logs (bottom) */}
        <div className="flex flex-col flex-1 overflow-hidden" style={{ minHeight: 0 }}>
          {/* Graph: takes 50% of the column height */}
          <div style={{ flex: '0 0 50%', minHeight: 0, overflow: 'hidden' }}>
            <GraphWrapper />
          </div>

          {/* Pools table: 25% */}
          <div style={{ flex: '0 0 25%', minHeight: 0, overflow: 'hidden' }}>
            <PoolsTable />
          </div>

          {/* Logs: remaining 25% */}
          <div style={{ flex: '0 0 25%', minHeight: 0, overflow: 'hidden' }}>
            <LiveLogs />
          </div>
        </div>

        {/* Right: settings sidebar */}
        <SettingsSidebar />
      </div>
    </main>
  );
}
