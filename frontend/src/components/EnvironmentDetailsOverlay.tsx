import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  X, 
  Terminal, 
  Cpu, 
  Settings, 
  Activity, 
  Layers, 
  RefreshCw, 
  Globe, 
  Server,
  Zap,
  ArrowRightLeft
} from 'lucide-react';
import { EnvironmentDetail, Repository } from '../types';
import { NetworkTopology } from './NetworkTopology';
import { StatusBadge } from './StatusBadge';

interface EnvironmentDetailsOverlayProps {
  envDetails: EnvironmentDetail | null;
  repos: Repository[];
  repoDeps: Record<string, { target: string; required: boolean }[]>;
  onClose: () => void;
  overlayTab: 'topology' | 'logs' | 'config';
  setOverlayTab: (tab: 'topology' | 'logs' | 'config') => void;
  logs: string[];
}

export const EnvironmentDetailsOverlay: React.FC<EnvironmentDetailsOverlayProps> = ({
  envDetails,
  repos,
  repoDeps,
  onClose,
  overlayTab,
  setOverlayTab,
  logs
}) => {
  if (!envDetails) return null;

  return (
    <motion.div 
      initial={{ opacity: 0, x: '100%' }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: '100%' }}
      className="fixed inset-0 z-50 flex"
    >
      <div className="flex-1 bg-ink/20 backdrop-blur-sm" onClick={onClose} />
      <div className="w-[85vw] lg:w-[75vw] bg-bg border-l-8 border-ink flex flex-col shadow-2xl">
        {/* Header */}
        <div className="p-8 border-b-4 border-ink bg-white flex justify-between items-center">
          <div className="flex items-center gap-6">
            <div className="p-4 bg-ink text-bg border-4 border-bg shadow-[4px_4px_0_rgba(20,20,20,1)]">
              {envDetails.environment.short_id.includes('prod') ? <Globe size={32} strokeWidth={3} /> : <Server size={32} strokeWidth={3} />}
            </div>
            <div>
              <div className="flex items-center gap-3 mb-1">
                <StatusBadge status={envDetails.environment.short_id.includes('prod') ? 'healthy' : 'warning'} size="md" />
                <span className="text-[10px] font-black uppercase tracking-[0.2em] opacity-40">Environment Profile</span>
              </div>
              <h2 className="text-4xl font-black uppercase tracking-tight text-ink">{envDetails.environment.name}</h2>
              <div className="flex items-center gap-8 mt-2">
                <div className="flex items-center gap-2">
                  <Terminal size={14} className="opacity-40" />
                  <span className="text-[10px] font-mono font-bold uppercase">{envDetails.environment.short_id}</span>
                </div>
                <div className="flex items-center gap-2 text-emerald-600">
                  <Activity size={14} />
                  <span className="text-[10px] font-mono font-black">99.98% AVAILABLE</span>
                </div>
              </div>
            </div>
          </div>
          <button 
            onClick={onClose}
            className="w-16 h-16 border-4 border-ink flex items-center justify-center hover:bg-ink hover:text-white transition-all bg-bg shadow-[4px_4px_0_rgba(20,20,20,1)] active:translate-x-0.5 active:translate-y-0.5"
          >
            <X size={32} strokeWidth={3} />
          </button>
        </div>

        <div className="flex-1 flex overflow-hidden">
          {/* Sidebar Tabs */}
          <div className="w-24 border-r-4 border-ink flex flex-col bg-white">
            <TabButton active={overlayTab === 'topology'} onClick={() => setOverlayTab('topology')} icon={<Layers />} label="GRID" />
            <TabButton active={overlayTab === 'logs'} onClick={() => setOverlayTab('logs')} icon={<Terminal />} label="LOGS" />
            <TabButton active={overlayTab === 'config'} onClick={() => setOverlayTab('config')} icon={<Settings />} label="CONF" />
          </div>

          {/* Tab Content */}
          <div className="flex-1 overflow-hidden flex flex-col">
            <AnimatePresence mode="wait">
              {overlayTab === 'topology' && (
                <motion.div 
                  key="topology"
                  initial={{ opacity: 0, scale: 0.98 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 1.02 }}
                  className="flex-1 p-8 flex flex-col gap-6"
                >
                  <div className="flex justify-between items-center bg-white p-4 border-2 border-ink">
                    <div className="flex items-center gap-4">
                      <div className="w-8 h-8 bg-ink text-bg flex items-center justify-center">
                        <ArrowRightLeft size={16} />
                      </div>
                      <h3 className="text-xs font-black uppercase tracking-widest">Runtime Lattice</h3>
                    </div>
                    <button className="flex items-center gap-2 text-[10px] font-black uppercase hover:opacity-60 transition-opacity">
                      <RefreshCw size={12} /> RE-LAYOUT
                    </button>
                  </div>
                  
                  <div className="flex-1 min-h-0 border-4 border-ink relative group">
                    <NetworkTopology 
                      deployments={envDetails.deployments} 
                      repositories={repos} 
                      dependencies={repoDeps}
                    />
                  </div>
                </motion.div>
              )}

              {overlayTab === 'logs' && (
                <motion.div 
                  key="logs"
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -10 }}
                  className="flex-1 p-8"
                >
                  <div className="h-full bg-ink text-emerald-500 overflow-y-auto p-6 flex flex-col gap-2 font-mono text-[11px] border-4 border-ink shadow-[inner_0_4px_12px_rgba(0,0,0,0.5)]">
                    {logs.map((log, i) => (
                      <div key={i} className="flex gap-4 border-b border-white/5 pb-1">
                        <span className="opacity-30 flex-shrink-0">[{new Date().toLocaleTimeString()}]</span>
                        <span className="opacity-50">#SYS:</span>
                        <span className="break-all">{log}</span>
                      </div>
                    ))}
                    <div className="flex gap-4 animate-pulse">
                      <span className="opacity-30">[{new Date().toLocaleTimeString()}]</span>
                      <span className="opacity-50">#SYS:</span>
                      <span>Waiting for stream ingress...</span>
                    </div>
                  </div>
                </motion.div>
              )}

              {overlayTab === 'config' && (
                <motion.div 
                  key="config"
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -10 }}
                  className="flex-1 p-8 grid grid-cols-2 gap-8 overflow-y-auto"
                >
                  <ConfigCard title="System Variables" icon={<Cpu />}>
                    <ConfigRow label="CPU_LIMIT" value="2.0" />
                    <ConfigRow label="MEM_RESERVE" value="512MB" />
                    <ConfigRow label="HMR_ENABLED" value="FALSE" />
                  </ConfigCard>
                  <ConfigCard title="Environment Secrets" icon={<Zap />}>
                    <ConfigRow label="STRIPE_KEY" value="sk_test_•••••" secret />
                    <ConfigRow label="DB_PASSWORD" value="••••••••••••" secret />
                    <ConfigRow label="AWS_REGION" value="us-west-2" />
                  </ConfigCard>
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        </div>
      </div>
    </motion.div>
  );
};

const TabButton = ({ active, onClick, icon, label }: any) => (
  <button 
    onClick={onClick}
    className={`flex-1 flex flex-col items-center justify-center gap-2 border-b-4 last:border-b-0 border-ink transition-all ${
      active ? 'bg-ink text-bg' : 'hover:bg-bg'
    }`}
  >
    <div className={active ? 'scale-110' : 'opacity-40'}>
      {React.cloneElement(icon, { size: 24, strokeWidth: 3 })}
    </div>
    <span className="text-[9px] font-black uppercase tracking-[0.2em]">{label}</span>
  </button>
);

const ConfigCard = ({ title, icon, children }: any) => (
  <div className="border-4 border-ink bg-white flex flex-col shadow-[4px_4px_0_rgba(20,20,20,0.1)]">
    <div className="p-4 border-b-2 border-ink bg-bg flex items-center gap-3">
      {React.cloneElement(icon, { size: 16, strokeWidth: 3 })}
      <h4 className="text-[10px] font-black uppercase tracking-widest">{title}</h4>
    </div>
    <div className="p-4 flex flex-col gap-3">
      {children}
    </div>
  </div>
);

const ConfigRow = ({ label, value, secret }: any) => (
  <div className="flex justify-between items-center border-b border-ink/5 pb-2 last:border-0 last:pb-0">
    <span className="text-[9px] font-black uppercase opacity-40">{label}</span>
    <span className={`text-[10px] font-black font-mono ${secret ? 'text-emerald-600' : 'text-ink'}`}>{value}</span>
  </div>
);
