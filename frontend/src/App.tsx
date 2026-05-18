import React, { useState, useEffect } from 'react';
import {
  Server,
  GitBranch,
  Activity,
  RefreshCw,
  Loader2,
  Plus,
  Box,
  Layers,
  X,
  ChevronRight,
  Menu,
  Search,
  Settings,
  Zap,
} from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { orchestratorApi } from './services/api';
import { NetworkTopology } from './components/NetworkTopology';
import { StatsDashboard } from './components/StatsDashboard';
import { StatusBadge } from './components/StatusBadge';
import { RepositoryCard } from './components/RepositoryCard';
import { EnvironmentCard } from './components/EnvironmentCard';
import { IntegrationCard } from './components/IntegrationCard';
import { EnvironmentDetailsOverlay } from './components/EnvironmentDetailsOverlay';
import { EnvModal, RepoModal, IntegrationModal } from './components/Modals';
import { AuditLogsView } from './components/AuditLogsView';
import { SettingsView } from './components/SettingsView';
import {
  Repository,
  Environment,
  Event,
  Deployment,
  Domain,
  RuntimeResource,
  Integration,
  RepositoryDependency,
} from './types';

export default function App() {
  const [activeTab, setActiveTab] = useState<'envs' | 'repos' | 'events' | 'integrations' | 'settings'>('envs');
  const [overlayTab, setOverlayTab] = useState<'topology' | 'logs' | 'config'>('topology');
  const [repos, setRepos] = useState<Repository[]>([]);
  const [envs, setEnvs] = useState<Environment[]>([]);
  const [events, setEvents] = useState<Event[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedEnv, setSelectedEnv] = useState<string | null>(null);
  const [selectedRepo, setSelectedRepo] = useState<Repository | null>(null);
  const [envDetails, setEnvDetails] = useState<any>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const [systemNotification, setSystemNotification] = useState<{ message: string; type: 'info' | 'error' | 'success' } | null>(null);
  
  // Modals
  const [isEnvModalOpen, setIsEnvModalOpen] = useState(false);
  const [isRepoModalOpen, setIsRepoModalOpen] = useState(false);
  const [isIntegrationModalOpen, setIsIntegrationModalOpen] = useState(false);
  
  // Form State
  const [newEnvFeature, setNewEnvFeature] = useState('');
  const [newEnvTTL, setNewEnvTTL] = useState('72h');
  const [repoSearch, setRepoSearch] = useState('');
  const [newRepoData, setNewRepoData] = useState<any>({ 
    name: '', full_name: '', service_kind: 'http', expose_port: 80, default_branch: 'main'
  });
  const [newIntegration, setNewIntegration] = useState<any>({ name: '', kind: 'github_app', secret: '' });
  const [editingIntegrationId, setEditingIntegrationId] = useState<string | null>(null);
  const [editingRepoId, setEditingRepoId] = useState<string | null>(null);

  // Filters
  const [auditSearch, setAuditSearch] = useState('');
  const [auditFilter, setAuditFilter] = useState('ALL');

  // Local Storage (preferences only)
  const [preferences, setPreferences] = useState(() => {
    const saved = localStorage.getItem('open_orch_prefs');
    return saved ? JSON.parse(saved) : { refreshInterval: 30, theme: 'light', notifications: true };
  });

  const [integrations, setIntegrations] = useState<Integration[]>([]);
  const [repoDeps, setRepoDeps] = useState<Record<string, RepositoryDependency[]>>({});

  useEffect(() => {
    localStorage.setItem('open_orch_prefs', JSON.stringify(preferences));
  }, [preferences]);

  const fetchIntegrations = async () => {
    try {
      const data = await orchestratorApi.getIntegrations();
      setIntegrations(data);
    } catch (err) {
      console.error('integrations fetch failed:', err);
    }
  };

  const fetchData = async () => {
    setIsRefreshing(true);
    try {
      const [r, e, ev] = await Promise.all([
        orchestratorApi.getRepositories(),
        orchestratorApi.getEnvironments(),
        orchestratorApi.getEvents(),
      ]);
      setRepos(r);
      setEnvs(e);
      setEvents(ev);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
      setIsRefreshing(false);
    }
  };

  useEffect(() => {
    fetchData();
    fetchIntegrations();
    const interval = setInterval(fetchData, preferences.refreshInterval * 1000);
    return () => clearInterval(interval);
  }, [preferences.refreshInterval]);

  useEffect(() => {
    if (!selectedRepo || repoDeps[selectedRepo.id] !== undefined) return;
    orchestratorApi.getDependencies(selectedRepo.name)
      .then(deps => setRepoDeps(prev => ({ ...prev, [selectedRepo.id]: deps })))
      .catch(() => setRepoDeps(prev => ({ ...prev, [selectedRepo.id]: [] })));
  }, [selectedRepo]);

  const viewEnv = async (id: string, tab: any = 'topology') => {
    setSelectedEnv(id);
    setOverlayTab(tab);
    setEnvDetails(null);
    try {
      const details = await orchestratorApi.getEnvironmentDetails(id);
      setEnvDetails(details);
      
      // Batch fetch missing deps (keyed by repo ID, looked up by name)
      details.deployments.forEach(async (dep: any) => {
        if (!repoDeps[dep.repository_id]) {
          const repo = repos.find(r => r.id === dep.repository_id);
          if (!repo) return;
          const deps = await orchestratorApi.getDependencies(repo.name);
          setRepoDeps(prev => ({ ...prev, [dep.repository_id]: deps }));
        }
      });
    } catch (err) {
      console.error(err);
    }
  };

  if (loading) {
    return (
      <div className="h-screen bg-bg flex flex-col items-center justify-center gap-4">
        <div className="relative">
          <Loader2 className="animate-spin text-ink" size={64} strokeWidth={3} />
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="w-2 h-2 bg-ink rounded-full animate-ping"></div>
          </div>
        </div>
        <span className="text-[10px] font-black tracking-[0.4em] uppercase opacity-40 ml-1">Lattice Initialization</span>
      </div>
    );
  }

  return (
    <div className="h-screen bg-bg text-ink flex flex-col md:flex-row overflow-hidden font-sans selection:bg-ink selection:text-bg">
      <div className="scanline" />

      {/* System Notification */}
      <AnimatePresence>
        {systemNotification && (
          <motion.div 
            initial={{ y: -100, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            exit={{ y: -100, opacity: 0 }}
            className="fixed top-8 left-1/2 -translate-x-1/2 z-[200] max-w-md w-full"
          >
            <div className={`brutalist-card p-4 flex items-center justify-between gap-4 border-l-[12px] ${
              systemNotification.type === 'error' ? 'border-red-500' : 
              systemNotification.type === 'success' ? 'border-emerald-500' : 'border-accent'
            }`}>
              <div className="flex items-center gap-3">
                <Activity size={20} className="text-ink/60" />
                <span className="text-[10px] font-black uppercase tracking-widest">{systemNotification.message}</span>
              </div>
              <button onClick={() => setSystemNotification(null)} className="p-1 hover:bg-ink hover:text-bg transition-colors">
                <X size={14} />
              </button>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
      
      {/* Sidebar */}
      <aside className={`fixed md:sticky top-0 left-0 h-full w-72 border-r-4 border-ink bg-bg z-50 transition-all duration-500 ease-[cubic-bezier(0.23,1,0.32,1)] ${isSidebarOpen ? 'translate-x-0 shadow-[20px_0_60px_rgba(0,0,0,0.2)]' : '-translate-x-full md:translate-x-0'}`}>
        <div className="p-10 border-b-4 border-ink bg-white relative overflow-hidden group">
          <div className="absolute top-0 right-0 w-24 h-24 bg-ink/5 -rotate-45 translate-x-12 -translate-y-12"></div>
          <motion.div initial={{ y: -10 }} animate={{ y: 0 }} className="flex items-center gap-2 mb-8">
             <Box size={20} className="text-accent" strokeWidth={3} />
             <span className="text-[10px] font-black uppercase tracking-[0.5em] opacity-40">Core::Lattice</span>
          </motion.div>
          <h1 className="text-4xl font-black uppercase italic tracking-tighter leading-[0.8] mb-12">Open<br/>Orchestra</h1>
          <div className="flex items-center gap-3">
            <div className="flex gap-1">
              <div className="w-1 h-1 bg-accent"></div>
              <div className="w-1 h-1 bg-accent/60"></div>
              <div className="w-1 h-1 bg-accent/30"></div>
            </div>
            <span className="text-[8px] font-black uppercase tracking-[0.25em] opacity-30">Secure_Node_0x{Math.floor(Math.random()*1000)}</span>
          </div>
        </div>

        <nav className="p-4 flex flex-col gap-1">
          <NavLink active={activeTab === 'envs'} onClick={() => { setActiveTab('envs'); setIsSidebarOpen(false); }} icon={<Server />} label="Environments" />
          <NavLink active={activeTab === 'repos'} onClick={() => { setActiveTab('repos'); setIsSidebarOpen(false); }} icon={<GitBranch />} label="Repositories" />
          <NavLink active={activeTab === 'events'} onClick={() => { setActiveTab('events'); setIsSidebarOpen(false); }} icon={<Activity />} label="Audit Logs" />
          <NavLink active={activeTab === 'integrations'} onClick={() => { setActiveTab('integrations'); setIsSidebarOpen(false); }} icon={<Zap />} label="Integrations" />
          <NavLink active={activeTab === 'settings'} onClick={() => { setActiveTab('settings'); setIsSidebarOpen(false); }} icon={<Settings />} label="Settings" />
        </nav>

        <div className="mt-auto border-t-4 border-ink bg-white flex flex-col">
          <div className="p-6 border-b-2 border-ink text-[8px] font-black uppercase tracking-widest opacity-30 flex justify-between">
            <span>v1.2.9_STABLE</span>
            <span className="text-accent">ENCRYPTED_FLOW</span>
          </div>
          <div className="p-4">
            <button 
              onClick={() => {
                fetchData();
                const btn = document.getElementById('sync-matrix-btn');
                if (btn) {
                  btn.classList.add('animate-shake');
                  setTimeout(() => btn.classList.remove('animate-shake'), 500);
                }
              }}
              id="sync-matrix-btn"
              disabled={isRefreshing}
              className="w-full py-4 bg-ink text-bg text-[10px] font-black uppercase flex items-center justify-center gap-4 transition-all hover:bg-accent active:translate-y-1 disabled:opacity-50 relative group overflow-hidden"
            >
              <div className={`transition-transform duration-700 ${isRefreshing ? 'rotate-180' : ''}`}>
                <RefreshCw className={isRefreshing ? "animate-spin" : ""} size={14} strokeWidth={3} />
              </div>
              <span className="relative z-10">{isRefreshing ? 'SYNCHRONIZING...' : 'SYNC_MATRIX'}</span>
              <div className="absolute inset-0 bg-accent translate-y-full group-hover:translate-y-0 transition-transform duration-300"></div>
            </button>
          </div>
        </div>

        <button className="md:hidden absolute top-8 right-8 w-12 h-12 border-4 border-ink bg-bg flex items-center justify-center" onClick={() => setIsSidebarOpen(false)}>
           <X size={24} strokeWidth={4} />
        </button>
      </aside>

      {/* Main Content */}
      <main className="flex-1 flex flex-col min-w-0 overflow-hidden relative">
        <header className="h-32 border-b-4 border-ink flex items-center justify-between px-8 lg:px-12 bg-white z-20">
          <div className="flex gap-12 lg:gap-20">
            <StatHeader label="Environments" value={envs.length} sub="Active Nodes" />
            <StatHeader label="Repositories" value={repos.length} sub="Source Links" />
            <div className="hidden 2xl:flex flex-col justify-center">
              <span className="text-[9px] font-black opacity-20 uppercase tracking-[0.3em] mb-3 leading-none">Traffic_Envelope</span>
              <div className="flex gap-2 h-8 items-end">
                {[0.4, 0.7, 0.2, 0.9, 0.5, 0.8, 0.3, 0.6, 0.4, 0.7].map((h, i) => (
                  <motion.div 
                    key={i} 
                    initial={{ height: 0 }}
                    animate={{ height: `${h * 100}%` }}
                    transition={{ delay: i * 0.05, repeat: Infinity, repeatType: 'reverse', duration: 1 + Math.random() }}
                    className="w-1.5 bg-ink/10 group-hover:bg-accent transition-colors" 
                  />
                ))}
              </div>
            </div>
          </div>
          
          <div className="flex items-center gap-6">
            <AnimatePresence mode="wait">
              {activeTab === 'envs' && (
                <motion.div key="btn-envs" initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -10 }}>
                  <PrimaryButton onClick={() => setIsEnvModalOpen(true)} label="INJECT WORKLOAD" />
                </motion.div>
              )}
              {activeTab === 'repos' && (
                <motion.div key="btn-repos" initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -10 }}>
                  <PrimaryButton 
                    onClick={() => {
                      setNewRepoData({ name: '', full_name: '', service_kind: 'http', expose_port: 80, default_branch: 'main' });
                      setEditingRepoId(null);
                      setIsRepoModalOpen(true);
                    }} 
                    label="REGISTER SOURCE" 
                  />
                </motion.div>
              )}
            </AnimatePresence>
            <button className="md:hidden w-16 h-16 border-4 border-ink border-l-0 -mr-8 flex items-center justify-center bg-white" onClick={() => setIsSidebarOpen(true)}>
              <Menu size={28} strokeWidth={4} />
            </button>
          </div>
        </header>

        <div className="flex-1 overflow-y-auto p-8 lg:p-16 custom-scrollbar bg-bg relative">
           <div className="absolute inset-0 opacity-[0.03] pointer-events-none" style={{ backgroundImage: 'radial-gradient(#000 1px, transparent 0)', backgroundSize: '40px 40px' }}></div>
          
          <AnimatePresence mode="wait">
            {activeTab === 'envs' && (
              <motion.div key="envs" initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: 20 }} className="space-y-16 relative z-10">
                <StatsDashboard />
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-10">
                  {envs.map(env => (
                    <EnvironmentCard key={env.id} env={env} onClick={() => viewEnv(env.short_id)} />
                  ))}
                  {envs.length === 0 && (
                    <div className="col-span-full border-4 border-dashed border-ink/10 p-40 text-center bg-white/30">
                      <Layers className="mx-auto mb-8 opacity-5" size={80} />
                      <p className="text-xs font-black uppercase opacity-20 tracking-[0.5em]">Awaiting manifest stream...</p>
                    </div>
                  )}
                </div>
              </motion.div>
            )}

            {activeTab === 'repos' && (
              <motion.div key="repos" initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: 20 }} className="space-y-16 relative z-10">
                 <div className="flex gap-6 max-w-3xl">
                    <div className="flex-1 relative group">
                      <Search className="absolute left-6 top-1/2 -translate-y-1/2 opacity-30 group-focus-within:opacity-100 transition-opacity" size={24} />
                      <input 
                        type="text" 
                        value={repoSearch} 
                        onChange={(e) => setRepoSearch(e.target.value)}
                        placeholder="SEARCH REPOSITORY MATRIX..." 
                        className="brutalist-input pl-16 shadow-[8px_8px_0_rgba(0,0,0,1)]"
                      />
                    </div>
                 </div>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-10">
                    {repos.filter(r => r.name.toLowerCase().includes(repoSearch.toLowerCase())).map(repo => (
                      <RepositoryCard 
                        key={repo.id} 
                        repo={repo} 
                        onClick={() => setSelectedRepo(repo)} 
                        onEdit={(r) => {
                          setNewRepoData(r);
                          setEditingRepoId(r.id);
                          setIsRepoModalOpen(true);
                        }}
                        onViewSpecs={(r) => setSelectedRepo(r)}
                      />
                    ))}
                  </div>
              </motion.div>
            )}

            {activeTab === 'events' && (
              <motion.div key="events" initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -20 }}>
                 <AuditLogsView 
                    events={events} 
                    search={auditSearch} 
                    setSearch={setAuditSearch} 
                    filter={auditFilter} 
                    setFilter={setAuditFilter} 
                 />
              </motion.div>
            )}

            {activeTab === 'integrations' && (
              <motion.div key="integrations" initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -20 }} className="space-y-12">
                <header className="border-l-8 border-ink pl-8 mb-12">
                  <h2 className="text-4xl font-black uppercase italic tracking-tighter leading-none mb-2">Connect_Lattice</h2>
                  <p className="text-[10px] font-black uppercase opacity-40">Manage identity links to external infrastructure providers.</p>
                </header>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
                  {integrations.map((integration) => (
                    <IntegrationCard
                      key={integration.id}
                      integration={integration}
                      onEdit={(i) => { setEditingIntegrationId(i.id); setNewIntegration({ name: i.name, kind: i.kind, secret: '' }); setIsIntegrationModalOpen(true); }}
                      onDelete={async (id) => {
                        try {
                          await orchestratorApi.deleteIntegration(id);
                          setIntegrations(prev => prev.filter(i => i.id !== id));
                        } catch (e) {
                          setSystemNotification({ message: 'Failed to delete integration', type: 'error' });
                          setTimeout(() => setSystemNotification(null), 3000);
                        }
                      }}
                    />
                  ))}
                  <button
                    onClick={() => { setEditingIntegrationId(null); setNewIntegration({ name: '', kind: 'github_app', secret: '' }); setIsIntegrationModalOpen(true); }}
                    className="border-4 border-dashed border-ink/20 p-12 flex flex-col items-center justify-center gap-6 hover:bg-white hover:border-ink transition-all group bg-white/50"
                  >
                    <div className="w-16 h-16 border-4 border-ink flex items-center justify-center bg-bg group-hover:bg-ink group-hover:text-bg transition-colors">
                      <Plus size={32} strokeWidth={3} className="group-hover:rotate-90 transition-transform duration-500" />
                    </div>
                    <span className="text-[10px] font-black uppercase opacity-40 tracking-[0.2em]">Add Protocol Link</span>
                  </button>
                </div>
              </motion.div>
            )}

            {activeTab === 'settings' && (
              <motion.div key="settings" initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -20 }}>
                 <SettingsView preferences={preferences} setPreferences={setPreferences} />
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </main>

      {/* Dependency Modal Overlay */}
      <AnimatePresence>
        {selectedRepo && (
           <motion.div 
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-[100] flex items-center justify-center p-6 md:p-12 bg-ink/60 backdrop-blur-md"
          >
             <motion.div 
               initial={{ scale: 0.95, y: 20 }} 
               animate={{ scale: 1, y: 0 }} 
               exit={{ scale: 0.98, opacity: 0 }}
               className="w-full max-w-7xl h-full bg-bg border-8 border-ink flex flex-col shadow-[32px_32px_0_rgba(20,20,20,1)] relative"
             >
                <div className="p-8 lg:p-10 border-b-8 border-ink bg-white flex justify-between items-center relative overflow-hidden">
                  <div className="absolute top-0 left-0 w-2 h-full bg-emerald-500 animate-pulse"></div>
                  <div>
                    <div className="flex items-center gap-4 mb-2">
                       <StatusBadge status="healthy" size="sm" />
                       <span className="text-[10px] font-black uppercase tracking-[0.3em] opacity-40">System_Trace</span>
                    </div>
                    <h2 className="text-5xl font-black uppercase italic tracking-tighter leading-none">{selectedRepo.name}</h2>
                    <p className="text-[11px] font-bold opacity-30 mt-3 uppercase tracking-tight">Source Context: {selectedRepo.full_name}</p>
                  </div>
                  <button onClick={() => setSelectedRepo(null)} className="w-20 h-20 border-4 border-ink flex items-center justify-center hover:bg-ink hover:text-bg transition-all shadow-[6px_6px_0_rgba(20,20,20,1)] active:translate-x-1 active:translate-y-1 active:shadow-none bg-bg">
                    <X size={36} strokeWidth={4} />
                  </button>
                </div>
                <div className="flex-1 bg-bg relative">
                  <NetworkTopology 
                    deployments={[{ repository_id: selectedRepo.id, state: 'healthy', container_name: selectedRepo.name, kind: selectedRepo.service_kind } as any]} 
                    repositories={repos} 
                    dependencies={repoDeps} 
                  />
                  <div className="absolute bottom-10 left-10 p-8 bg-white border-4 border-ink shadow-[12px_12px_0_rgba(20,20,20,1)] max-w-md">
                    <div className="flex items-center gap-3 mb-6">
                      <div className="w-10 h-10 bg-ink text-bg flex items-center justify-center">
                         <Layers size={20} />
                      </div>
                      <h4 className="text-xs font-black uppercase tracking-[0.2em]">Downstream Lattice</h4>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {repoDeps[selectedRepo.id]?.map(d => (
                        <div key={d.depends_on_id} className="px-3 py-2 border-2 border-ink bg-bg text-[10px] font-black uppercase flex items-center gap-2 group hover:bg-ink hover:text-bg transition-all cursor-default">
                           <div className={`w-2 h-2 border border-current ${d.required ? 'bg-red-500' : 'bg-zinc-400'}`}></div>
                           {d.depends_on_name}
                        </div>
                      ))}
                      {!repoDeps[selectedRepo.id]?.length && (
                        <div className="flex flex-col gap-2">
                           <span className="text-[10px] opacity-30 italic font-bold uppercase tracking-widest">Isolated_System_Node</span>
                           <div className="h-1 bg-zinc-200 w-full overflow-hidden">
                              <motion.div className="h-full bg-ink w-1/3" animate={{ x: ['100%', '-100%'] }} transition={{ repeat: Infinity, duration: 2, ease: "linear" }} />
                           </div>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
             </motion.div>
           </motion.div>
        )}
      </AnimatePresence>

      <EnvironmentDetailsOverlay 
        envDetails={envDetails} 
        repos={repos} 
        repoDeps={repoDeps} 
        onClose={() => setSelectedEnv(null)} 
        overlayTab={overlayTab} 
        setOverlayTab={setOverlayTab}
        logs={['Manifest converging...', 'Relating entity nodes...', 'Lattice healthy.', 'Synchronizing telemetry...', 'Ingress verified.']}
      />

      <EnvModal 
        isOpen={isEnvModalOpen} 
        onClose={() => setIsEnvModalOpen(false)} 
        name={newEnvFeature} setName={setNewEnvFeature}
        ttl={newEnvTTL} setTTL={setNewEnvTTL}
        onConfirm={async () => { 
          try {
            await orchestratorApi.createEnvironment({ feature: newEnvFeature, ttl: newEnvTTL }); 
            setSystemNotification({ message: 'Environment Manifest Injected Successfully', type: 'success' });
            setTimeout(() => setSystemNotification(null), 3000);
          } catch (e) {
            setSystemNotification({ message: 'Lattice Protocol Exception', type: 'error' });
          }
          setIsEnvModalOpen(false); 
          fetchData(); 
        }}
      />

      <RepoModal
        isOpen={isRepoModalOpen}
        onClose={() => { setIsRepoModalOpen(false); setEditingRepoId(null); }}
        data={newRepoData} setData={setNewRepoData}
        isEditing={!!editingRepoId}
        onConfirm={async () => { 
          try {
            if (editingRepoId) {
              await orchestratorApi.upsertRepository({ ...newRepoData, id: editingRepoId });
              setSystemNotification({ message: 'Source Matrix Updated', type: 'success' });
            } else {
              await orchestratorApi.upsertRepository(newRepoData); 
              setSystemNotification({ message: 'Source Registered to Lattice', type: 'success' });
            }
            setTimeout(() => setSystemNotification(null), 3000);
          } catch (e) {
             setSystemNotification({ message: 'Lattice Protocol Error', type: 'error' });
          }
          setIsRepoModalOpen(false); 
          setEditingRepoId(null);
          fetchData(); 
        }}
      />

      <IntegrationModal
        isOpen={isIntegrationModalOpen}
        onClose={() => { setIsIntegrationModalOpen(false); setEditingIntegrationId(null); }}
        data={newIntegration} setData={setNewIntegration}
        isEditing={!!editingIntegrationId}
        onConfirm={async () => {
          try {
            if (editingIntegrationId) {
              const patch: any = {};
              if (newIntegration.secret !== undefined) patch.secret = newIntegration.secret;
              await orchestratorApi.patchIntegration(editingIntegrationId, patch);
              setSystemNotification({ message: 'Identity Link Updated', type: 'success' });
            } else {
              await orchestratorApi.createIntegration({
                kind: newIntegration.kind,
                name: newIntegration.name,
                secret: newIntegration.secret || undefined,
              });
              setSystemNotification({ message: 'New Protocol Link Established', type: 'success' });
            }
            await fetchIntegrations();
            setTimeout(() => setSystemNotification(null), 3000);
          } catch (e) {
            setSystemNotification({ message: 'Authentication Refused', type: 'error' });
            setTimeout(() => setSystemNotification(null), 3000);
          }
          setIsIntegrationModalOpen(false);
          setEditingIntegrationId(null);
        }}
      />
    </div>
  );
}

const NavLink = ({ active, onClick, icon, label }: any) => (
  <button 
    onClick={onClick}
    className={`p-5 flex items-center justify-between group transition-all duration-300 border-l-8 ${
      active ? 'bg-ink text-bg border-ink translate-x-1' : 'hover:bg-white border-transparent hover:border-ink/10'
    }`}
  >
    <div className="flex items-center gap-5">
      <div className={`transition-transform duration-500 ${active ? 'scale-125' : 'group-hover:scale-110'}`}>
        {React.cloneElement(icon, { size: 18, strokeWidth: active ? 3 : 2.5 })}
      </div>
      <span className={`text-[11px] font-black uppercase tracking-[0.2em] ${active ? 'opacity-100' : 'opacity-40 group-hover:opacity-100'}`}>{label}</span>
    </div>
    {active && <ChevronRight size={14} strokeWidth={4} className="animate-pulse text-accent" />}
  </button>
);

const StatHeader = ({ label, value, sub }: any) => (
  <div className="flex flex-col group cursor-default">
    <span className="text-[9px] font-black opacity-30 uppercase tracking-[0.25em] leading-none mb-3 group-hover:opacity-100 transition-opacity">{label}</span>
    <div className="flex items-baseline gap-3">
      <span className="text-5xl font-black italic font-mono leading-none text-ink drop-shadow-sm">{value < 10 ? `0${value}` : value}</span>
      <div className="flex flex-col">
        <span className="text-[8px] font-black uppercase tracking-widest text-accent mb-0.5">{sub}</span>
        <div className="w-8 h-1 bg-ink/10 group-hover:bg-ink transition-colors"></div>
      </div>
    </div>
  </div>
);

const PrimaryButton = ({ onClick, label }: any) => (
  <button 
    onClick={onClick}
    className="h-16 px-10 bg-ink text-bg text-[11px] font-black uppercase tracking-[0.2em] hover:bg-accent transition-all active:translate-y-1.5 shadow-[8px_8px_0_rgba(0,0,0,1)] active:shadow-none relative overflow-hidden group"
  >
    <span className="relative z-10">{label}</span>
    <div className="absolute inset-0 bg-white translate-y-full group-hover:translate-y-0 transition-transform duration-500 mix-blend-difference"></div>
  </button>
);
