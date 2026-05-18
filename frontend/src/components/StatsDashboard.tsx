import React from 'react';
import { 
  AreaChart, 
  Area, 
  XAxis, 
  YAxis, 
  CartesianGrid, 
  Tooltip, 
  ResponsiveContainer 
} from 'recharts';

const data = [
  { time: '08:00', load: 12, events: 45 },
  { time: '09:00', load: 34, events: 120 },
  { time: '10:00', load: 45, events: 89 },
  { time: '11:00', load: 30, events: 67 },
  { time: '12:00', load: 55, events: 156 },
  { time: '13:00', load: 89, events: 210 },
  { time: '14:00', load: 60, events: 134 },
];

export const StatsDashboard: React.FC = () => {
  return (
    <div className="w-full h-56 border-4 border-ink bg-white p-6 relative overflow-hidden group shadow-[12px_12px_0_rgba(20,20,20,1)]">
      <div className="absolute inset-0 opacity-[0.03] pointer-events-none" style={{ backgroundImage: 'radial-gradient(#141414 1px, transparent 0)', backgroundSize: '24px 24px' }}></div>
      
      <div className="flex justify-between items-start mb-8 relative z-10">
        <div>
          <h3 className="text-[12px] font-black uppercase tracking-[0.3em] italic flex items-center gap-3 mb-2">
             <div className="w-2 h-2 bg-ink animate-ping" />
             Lattice_Pulse::Production_Stream
          </h3>
          <p className="text-[9px] font-bold text-ink/40 uppercase tracking-widest">Real-time telemetry from all active entity nodes</p>
        </div>
        <div className="flex gap-6">
          <div className="flex flex-col items-end">
            <div className="flex items-center gap-2 mb-1">
              <div className="w-3 h-3 bg-red-500 border-2 border-ink" />
              <span className="text-[9px] font-black uppercase opacity-60">System_Payload</span>
            </div>
            <span className="text-xl font-black italic font-mono leading-none tracking-tighter">89.2%</span>
          </div>
          <div className="flex flex-col items-end">
            <div className="flex items-center gap-2 mb-1">
              <div className="w-3 h-3 bg-emerald-500 border-2 border-ink" />
              <span className="text-[9px] font-black uppercase opacity-60">Event_Velocity</span>
            </div>
            <span className="text-xl font-black italic font-mono leading-none tracking-tighter">1.2k/s</span>
          </div>
        </div>
      </div>
      <div className="w-full h-24">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data}>
            <defs>
              <pattern id="diagonalHatch" patternUnits="userSpaceOnUse" width="4" height="4">
                <path d="M-1,1 l2,-2 M0,4 l4,-4 M3,5 l2,-2" 
                      style={{ stroke: '#141414', strokeWidth: 1, opacity: 0.1 }} />
              </pattern>
            </defs>
            <CartesianGrid strokeDasharray="0" vertical={false} stroke="#eee" />
            <XAxis 
              dataKey="time" 
              hide
            />
            <YAxis hide domain={[0, 100]} />
            <Tooltip 
              cursor={{ stroke: '#141414', strokeWidth: 2 }}
              contentStyle={{ 
                backgroundColor: '#141414', 
                border: 'none', 
                color: '#fff',
                fontSize: '10px',
                fontWeight: 'bold',
                fontFamily: 'monospace',
                padding: '12px'
              }}
              itemStyle={{ color: '#fff' }}
            />
            <Area 
              type="stepAfter" 
              dataKey="load" 
              stroke="#EF4444" 
              strokeWidth={4}
              fillOpacity={1} 
              fill="url(#diagonalHatch)" 
              animationDuration={2000}
            />
            <Area 
              type="stepAfter" 
              dataKey="events" 
              stroke="#10B981" 
              strokeWidth={4}
              fillOpacity={0} 
              animationDuration={2000}
              animationBegin={500}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
      <div className="absolute bottom-4 left-6 right-6 flex justify-between items-center opacity-30">
        <div className="flex gap-4">
          {['T-minus 60s', 'T-minus 45s', 'T-minus 30s', 'T-minus 15s', 'NOW'].map((t, i) => (
            <span key={i} className="text-[8px] font-black uppercase tracking-widest">{t}</span>
          ))}
        </div>
        <span className="text-[8px] font-black uppercase tracking-widest">Buffer_Synced</span>
      </div>
    </div>
  );
};
