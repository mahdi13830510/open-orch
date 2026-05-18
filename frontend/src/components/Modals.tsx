import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X, Database, Zap, Plus, Terminal, Clock } from 'lucide-react';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  subtitle: string;
  children: React.ReactNode;
  icon?: React.ReactNode;
}

const Modal: React.FC<ModalProps> = ({ isOpen, onClose, title, subtitle, children, icon }) => {
  return (
    <AnimatePresence>
      {isOpen && (
        <>
          <motion.div 
            initial={{ opacity: 0 }}
            animate={{ opacity: 0.8 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
            className="fixed inset-0 bg-[#141414] z-[100]"
          />
          <motion.div 
            initial={{ scale: 0.9, opacity: 0, y: 20 }}
            animate={{ scale: 1, opacity: 1, y: 0 }}
            exit={{ scale: 0.9, opacity: 0, y: 20 }}
            className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[95vw] lg:w-[480px] bg-bg border-4 border-ink p-8 lg:p-12 z-[101] shadow-[24px_24px_0_rgba(20,20,20,1)] overflow-hidden flex flex-col max-h-[90vh]"
          >
            <div className="mb-8 relative z-10">
              {icon && (
                <div className="w-12 h-12 bg-ink text-bg flex items-center justify-center mb-6 shadow-[4px_4px_0_rgba(20,20,20,0.2)]">
                  {icon}
                </div>
              )}
              <h3 className="text-3xl font-black uppercase italic tracking-tighter mb-2 leading-none" dangerouslySetInnerHTML={{ __html: title }}></h3>
              <p className="text-[11px] font-bold text-ink/50 uppercase tracking-tight leading-relaxed">{subtitle}</p>
            </div>
            
            <div className="flex-1 overflow-y-auto pr-2 custom-scrollbar pb-2">
              {children}
            </div>

            <button 
              onClick={onClose}
              className="absolute top-8 right-8 p-2 hover:rotate-90 transition-transform"
            >
              <X size={24} strokeWidth={3} />
            </button>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
};

export const EnvModal = ({ isOpen, onClose, name, setName, ttl, setTTL, onConfirm }: any) => (
  <Modal 
    isOpen={isOpen} 
    onClose={onClose} 
    icon={<Terminal />}
    title="Provision<br/>Workload" 
    subtitle="Supply branch identity or feature slug to trigger the reconciler loop."
  >
    <div className="space-y-6">
      <div className="space-y-2">
        <label className="text-[10px] font-black uppercase tracking-widest text-ink/40">Target Identity</label>
        <input 
          type="text" 
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="E.G. FEAT/CORE-API"
          className="w-full h-14 bg-white border-2 border-ink px-4 text-xs font-black font-mono uppercase focus:bg-ink focus:text-bg transition-colors"
        />
      </div>
      <div className="space-y-2">
        <label className="text-[10px] font-black uppercase tracking-widest text-ink/40">Lifespan Protocol (TTL)</label>
        <input 
          type="text" 
          value={ttl}
          onChange={(e) => setTTL(e.target.value)}
          placeholder="72h"
          className="w-full h-14 bg-white border-2 border-ink px-4 text-xs font-black font-mono uppercase focus:bg-ink focus:text-bg transition-colors"
        />
      </div>
      <button 
        onClick={onConfirm}
        disabled={!name}
        className="w-full py-4 mt-4 bg-ink text-bg font-black text-xs uppercase hover:invert transition-all disabled:opacity-20"
      >
        INJECT MANIFEST
      </button>
    </div>
  </Modal>
);

export const RepoModal = ({ isOpen, onClose, data, setData, onConfirm, isEditing }: any) => (
  <Modal 
    isOpen={isOpen} 
    onClose={onClose} 
    icon={<Database />}
    title={isEditing ? "Edit<br/>Entity" : "Register<br/>Source"} 
    subtitle={isEditing ? "Modify existing codebase configuration." : "Connect a codebase to the orchestration engine logic."}
  >
    <div className="space-y-6">
      <div className="space-y-2">
        <label className="text-[10px] font-black uppercase tracking-widest text-ink/40">Service ID</label>
        <input 
          type="text" 
          value={data.name}
          onChange={(e) => setData({ ...data, name: e.target.value })}
          placeholder="E.G. AUTH-SERVICE"
          className="w-full h-14 bg-white border-2 border-ink px-4 text-xs font-black font-mono uppercase"
        />
      </div>
      <div className="space-y-2">
        <label className="text-[10px] font-black uppercase tracking-widest text-ink/40">Context (Org/Repo)</label>
        <input 
          type="text" 
          value={data.full_name}
          onChange={(e) => setData({ ...data, full_name: e.target.value })}
          placeholder="ORG/REPO"
          className="w-full h-14 bg-white border-2 border-ink px-4 text-xs font-black font-mono uppercase"
        />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <label className="text-[10px] font-black uppercase tracking-widest text-ink/40">Branch</label>
          <input 
            type="text" 
            value={data.default_branch}
            onChange={(e) => setData({ ...data, default_branch: e.target.value })}
            className="w-full h-14 bg-white border-2 border-ink px-4 text-xs font-black font-mono uppercase"
          />
        </div>
        <div className="space-y-2">
          <label className="text-[10px] font-black uppercase tracking-widest text-ink/40">Port</label>
          <input 
            type="number" 
            value={data.expose_port}
            onChange={(e) => setData({ ...data, expose_port: parseInt(e.target.value) })}
            className="w-full h-14 bg-white border-2 border-ink px-4 text-xs font-black font-mono uppercase"
          />
        </div>
      </div>
      <div className="space-y-2">
        <label className="text-[10px] font-black uppercase tracking-widest text-ink/40">Arch</label>
        <div className="grid grid-cols-3 gap-2">
            {['http', 'worker', 'static'].map(k => (
              <button 
                key={k}
                onClick={() => setData({ ...data, service_kind: k })}
                className={`h-12 border-2 border-ink text-[10px] font-black uppercase ${data.service_kind === k ? 'bg-ink text-bg' : 'bg-white hover:bg-bg'}`}
              >
                {k}
              </button>
            ))}
        </div>
      </div>
      <button 
        onClick={onConfirm}
        className="w-full py-4 bg-ink text-bg font-black text-xs uppercase hover:invert transition-all"
      >
        {isEditing ? 'COMMIT_CHANGES' : 'REGISTER_CORE'}
      </button>
    </div>
  </Modal>
);

export const IntegrationModal = ({ isOpen, onClose, data, setData, onConfirm, isEditing }: any) => (
  <Modal
    isOpen={isOpen}
    onClose={onClose}
    icon={<Zap />}
    title={isEditing ? "Edit<br/>Link" : "New<br/>Link"}
    subtitle="Configure external service authentication."
  >
    <div className="space-y-6">
      <div className="space-y-2">
        <label className="text-[10px] font-black uppercase tracking-widest text-ink/40">Name</label>
        <input
          type="text"
          value={data.name}
          onChange={(e) => setData({ ...data, name: e.target.value })}
          placeholder="MY_CREDENTIAL"
          disabled={isEditing}
          className="w-full h-12 bg-white border-2 border-ink px-4 text-xs font-black disabled:opacity-50"
        />
      </div>
      {!isEditing && (
        <div className="space-y-2">
          <label className="text-[10px] font-black uppercase tracking-widest text-ink/40">Kind</label>
          <select
            value={data.kind}
            onChange={(e) => setData({ ...data, kind: e.target.value })}
            className="w-full h-12 bg-white border-2 border-ink px-4 text-[10px] font-black uppercase appearance-none"
          >
            <option value="github_app">GitHub App</option>
            <option value="buddy">Buddy CI</option>
            <option value="cloudflare">Cloudflare</option>
            <option value="registry">Container Registry</option>
            <option value="webhook">Webhook</option>
            <option value="custom">Custom</option>
          </select>
        </div>
      )}
      <div className="space-y-2">
        <label className="text-[10px] font-black uppercase tracking-widest text-ink/40">
          Secret {isEditing && <span className="opacity-50">(leave blank to keep existing)</span>}
        </label>
        <input
          type="password"
          value={data.secret || ''}
          onChange={(e) => setData({ ...data, secret: e.target.value })}
          placeholder={isEditing ? '••••••••' : 'TOKEN_OR_KEY'}
          className="w-full h-12 bg-white border-2 border-ink px-4 text-xs font-black font-mono"
        />
      </div>
      <button
        onClick={onConfirm}
        disabled={!isEditing && (!data.name || !data.kind)}
        className="w-full py-4 bg-ink text-bg font-black text-xs uppercase hover:invert transition-all disabled:opacity-20"
      >
        {isEditing ? 'UPDATE_STATE' : 'CONNECT_ENTITY'}
      </button>
    </div>
  </Modal>
);
