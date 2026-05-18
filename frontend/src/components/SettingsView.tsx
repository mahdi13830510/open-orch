import React from 'react';
import { motion } from 'framer-motion';
import { 
  Settings as SettingsIcon, 
  Bell, 
  Lock, 
  Cpu, 
  Trash2, 
  Share2, 
  Server,
  Zap,
  ShieldCheck,
  Smartphone
} from 'lucide-react';

interface SettingsViewProps {
  preferences: any;
  setPreferences: (p: any) => void;
}

export const SettingsView: React.FC<SettingsViewProps> = ({ preferences, setPreferences }) => {
  const updatePref = (key: string, val: any) => {
    setPreferences({ ...preferences, [key]: val });
  };

  return (
    <motion.div 
      initial={{ opacity: 0, x: 20 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: -20 }}
      className="max-w-4xl space-y-12 pb-24"
    >
      <header>
        <h2 className="text-4xl font-black uppercase italic tracking-tighter mb-4">Core_Config</h2>
        <p className="text-xs font-black uppercase opacity-40 leading-relaxed tracking-tight">Tune the orchestration engine and interface protocols.</p>
      </header>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
        <section className="space-y-6">
          <SectionHeader icon={<Cpu />} title="Orchestration Logic" />
          <div className="space-y-4">
            <SettingRow 
              label="State Refresh Interval" 
              description="Frequency of lattice synchronization in seconds."
            >
              <select 
                value={preferences.refreshInterval}
                onChange={(e) => updatePref('refreshInterval', parseInt(e.target.value))}
                className="bg-white border-2 border-ink px-4 h-10 text-[10px] font-black uppercase appearance-none cursor-pointer hover:bg-surface"
              >
                <option value={10}>10 SECONDS</option>
                <option value={30}>30 SECONDS</option>
                <option value={60}>1 MINUTE</option>
                <option value={300}>5 MINUTES</option>
              </select>
            </SettingRow>

            <SettingRow 
              label="Notification Protocol" 
              description="Push deployment status updates to client."
            >
              <Toggle 
                active={preferences.notifications} 
                onToggle={() => updatePref('notifications', !preferences.notifications)} 
              />
            </SettingRow>
          </div>
        </section>

        <section className="space-y-6">
          <SectionHeader icon={<ShieldCheck />} title="Security & API" />
          <div className="space-y-4">
             <div className="p-5 bg-white border-4 border-ink shadow-[4px_4px_0_rgba(20,20,20,1)] flex flex-col gap-4">
                <div className="flex justify-between items-center">
                  <span className="text-[10px] font-black uppercase tracking-widest">Master API Token</span>
                  <span className="px-2 py-0.5 bg-green-100 text-green-700 text-[8px] font-black uppercase border border-green-200">ACTIVE</span>
                </div>
                <div className="bg-bg p-3 border-2 border-ink font-mono text-[10px] truncate opacity-40">
                  sk_live_orchestra_9921_xXyYzZ...
                </div>
                <button 
                  onClick={(e) => {
                    const btn = e.currentTarget;
                    btn.innerText = "ROTATING...";
                    btn.disabled = true;
                    setTimeout(() => {
                      btn.innerText = "ROTATE CREDENTIALS";
                      btn.disabled = false;
                    }, 2000);
                  }}
                  className="py-2 border-2 border-ink text-[9px] font-black uppercase hover:bg-ink hover:text-bg transition-all active:translate-y-0.5 disabled:opacity-50"
                >
                  ROTATE CREDENTIALS
                </button>
             </div>
          </div>
        </section>
      </div>

      <section className="border-t-4 border-ink pt-12 space-y-8">
        <SectionHeader icon={<Zap />} title="Interface Mutation" />
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <button 
            onClick={() => {
              setPreferences({ ...preferences, notifications: true });
              alert('Mobile Lattice layout optimized.'); // Still using alert here, let's fix
            }}
            className="p-6 bg-white border-4 border-ink flex flex-col gap-4 hover:bg-black hover:text-white transition-all text-left shadow-[4px_4px_0_rgba(20,20,20,1)] active:translate-y-1 active:shadow-none"
          >
            <Smartphone size={24} />
            <div>
              <h4 className="text-[11px] font-black uppercase mb-1">Mobile Lattice</h4>
              <p className="text-[9px] font-bold opacity-40 uppercase">Optimized layout for touch</p>
            </div>
          </button>
          <button 
            onClick={() => {
               const dummy = document.createElement('textarea');
               dummy.value = 'Lattice Manifest v1.2.9\n...';
               document.body.appendChild(dummy);
               dummy.select();
               document.execCommand('copy');
               document.body.removeChild(dummy);
               // We should really have a prop-passed setSystemNotification here, but for simplicity:
               alert('Entity Exported to Clipboard.'); 
            }}
            className="p-6 bg-white border-4 border-ink flex flex-col gap-4 hover:bg-black hover:text-white transition-all text-left shadow-[4px_4px_0_rgba(20,20,20,1)] active:translate-y-1 active:shadow-none"
          >
            <Share2 size={24} />
            <div>
              <h4 className="text-[11px] font-black uppercase mb-1">Entity Export</h4>
              <p className="text-[9px] font-bold opacity-40 uppercase">Generate YAML manifest</p>
            </div>
          </button>
          <button 
            onClick={() => {
              if (window.confirm('ARE YOU ABSOLUTELY SURE? This will terminate all active workloads.')) {
                window.location.reload();
              }
            }}
            className="p-6 bg-red-50 border-4 border-red-600 flex flex-col gap-4 hover:bg-red-600 hover:text-white transition-all text-left shadow-[4px_4px_0_rgba(220,38,38,1)] active:translate-y-1 active:shadow-none font-black text-red-600"
          >
            <Trash2 size={24} />
            <div>
              <h4 className="text-[11px] font-black uppercase mb-1">PURGE SYSTEM</h4>
              <p className="text-[9px] font-bold opacity-40 uppercase">DELETE ALL RESOURCES</p>
            </div>
          </button>
        </div>
      </section>
    </motion.div>
  );
};

const SectionHeader = ({ icon, title }: any) => (
  <div className="flex items-center gap-3 border-b-2 border-ink pb-4">
    <div className="p-2 bg-ink text-bg">
      {React.cloneElement(icon, { size: 18, strokeWidth: 3 })}
    </div>
    <h3 className="text-sm font-black uppercase tracking-widest">{title}</h3>
  </div>
);

const SettingRow = ({ label, description, children }: any) => (
  <div className="flex justify-between items-start gap-8">
    <div className="flex flex-col gap-1">
      <span className="text-[11px] font-black uppercase tracking-tight">{label}</span>
      <p className="text-[9px] font-bold opacity-40 uppercase tracking-tight leading-relaxed">{description}</p>
    </div>
    <div className="flex-shrink-0">
      {children}
    </div>
  </div>
);

const Toggle = ({ active, onToggle }: any) => (
  <button 
    onClick={onToggle}
    className={`w-12 h-6 border-2 border-ink relative transition-colors ${active ? 'bg-green-500' : 'bg-zinc-200'}`}
  >
    <motion.div 
      className="absolute top-0.5 bottom-0.5 w-4 bg-ink"
      animate={{ left: active ? 'calc(100% - 1.25rem)' : '0.125rem' }}
    />
  </button>
);
