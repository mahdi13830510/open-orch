import React from 'react';
import { motion } from 'framer-motion';
import { Globe, Server, Activity, ChevronRight } from 'lucide-react';
import { Environment } from '../types';
import { StatusBadge } from './StatusBadge';

interface EnvironmentCardProps {
  env: Environment;
  onClick: () => void;
}

export const EnvironmentCard: React.FC<EnvironmentCardProps> = ({ env, onClick }) => {
  const healthStatus = env.short_id.includes('prod') ? 'healthy' : 'warning';

  return (
    <motion.div 
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      whileHover={{ y: -5 }}
      onClick={onClick}
      className="brutalist-card p-6 flex flex-col gap-6 cursor-pointer group"
    >
      <div className="flex justify-between items-start">
        <div className="flex items-center gap-4">
          <div className="p-3 bg-ink text-bg">
            {env.short_id.includes('prod') ? <Globe size={24} strokeWidth={3} /> : <Server size={24} strokeWidth={3} />}
          </div>
          <div>
            <h3 className="text-lg font-black uppercase italic tracking-tighter leading-none mb-1">{env.name}</h3>
            <span className="text-[10px] font-black font-mono opacity-30 uppercase tracking-widest">{env.short_id}</span>
          </div>
        </div>
        <StatusBadge status={healthStatus} size="sm" />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="bg-zinc-50 border-2 border-ink/10 p-3 flex flex-col justify-center">
          <span className="text-[9px] font-black uppercase opacity-40 mb-1">Stability_Core</span>
          <span className="text-xs font-black font-mono text-emerald-600 tracking-tighter">99.98%</span>
        </div>
        <div className="bg-zinc-50 border-2 border-ink/10 p-3 flex flex-col justify-center">
          <span className="text-[9px] font-black uppercase opacity-40 mb-1">Lattice_Load</span>
          <span className="text-xs font-black font-mono text-ink tracking-tighter">0.42_AVG</span>
        </div>
      </div>
      
      <div className="flex items-center justify-between mt-2 pt-4 border-t-2 border-ink/5">
        <div className="flex items-center gap-2">
           <div className="w-1.5 h-1.5 bg-emerald-500 animate-pulse rounded-full" />
           <span className="text-[9px] font-black uppercase tracking-widest opacity-40">Stream_Active</span>
        </div>
        <div className="flex items-center gap-2 group-hover:gap-4 transition-all text-[10px] font-black uppercase italic tracking-tighter">
          <span>Inspect</span>
          <ChevronRight size={16} strokeWidth={4} />
        </div>
      </div>
    </motion.div>
  );
};
