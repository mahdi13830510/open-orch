import React from 'react';
import { motion } from 'framer-motion';
import { Search, Activity, Clock, Terminal, ChevronRight } from 'lucide-react';
import { Event } from '../types';

interface AuditLogsViewProps {
  events: Event[];
  search: string;
  setSearch: (s: string) => void;
  filter: string;
  setFilter: (f: string) => void;
}

export const AuditLogsView: React.FC<AuditLogsViewProps> = ({
  events,
  search,
  setSearch,
  filter,
  setFilter,
}) => {
  const filteredEvents = events.filter(e => {
    const matchesSearch =
      !search ||
      e.event_type.toLowerCase().includes(search.toLowerCase()) ||
      (e.repository && e.repository.toLowerCase().includes(search.toLowerCase()));
    const matchesFilter =
      filter === 'ALL' ||
      (filter === 'SYSTEM' && e.source === 'system') ||
      (filter === 'USER' && e.source === 'user') ||
      (filter === 'CRITICAL' && (e.process_error || e.event_type.includes('fail')));
    return matchesSearch && matchesFilter;
  });

  const handleReview = (event: Event) => {
    const detail = [
      `Type: ${event.event_type}`,
      event.action ? `Action: ${event.action}` : '',
      event.repository ? `Repo: ${event.repository}` : '',
      event.process_error ? `Error: ${event.process_error}` : '',
    ].filter(Boolean).join('\n');
    alert(`Event: ${event.id}\n\n${detail}`);
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -10 }}
      className="space-y-8"
    >
      <div className="flex flex-col md:flex-row gap-4">
        <div className="flex-1 relative">
          <Search className="absolute left-4 top-1/2 -translate-y-1/2 opacity-20" size={18} />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Filter Audit Stream..."
            className="w-full h-14 pl-12 pr-4 bg-white border-4 border-ink font-black font-mono text-xs uppercase focus:bg-ink focus:text-bg transition-colors"
          />
        </div>
        <div className="flex gap-2">
          {['ALL', 'SYSTEM', 'USER', 'CRITICAL'].map((f) => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={`px-6 border-4 border-ink text-[10px] font-black uppercase transition-all ${
                filter === f ? 'bg-ink text-bg' : 'bg-white hover:bg-bg'
              }`}
            >
              {f}
            </button>
          ))}
        </div>
      </div>

      <div className="bg-white border-4 border-ink overflow-hidden shadow-[8px_8px_0_rgba(20,20,20,1)]">
        <div className="p-4 bg-ink text-bg flex items-center justify-between border-b-4 border-ink">
          <div className="flex items-center gap-3">
            <Activity size={16} />
            <span className="text-[10px] font-black uppercase tracking-[0.2em]">Real-time Event Stream</span>
          </div>
          <span className="text-[9px] font-bold opacity-50 uppercase tracking-widest">{filteredEvents.length} Entries Recorded</span>
        </div>

        <div className="divide-y-2 divide-ink/10 overflow-y-auto max-h-[60vh] custom-scrollbar">
          {filteredEvents.map((event) => (
            <div key={event.id} className="group p-6 flex flex-col md:flex-row md:items-center gap-6 hover:bg-zinc-50 transition-colors">
              <div className="flex flex-col gap-1 w-32 shrink-0">
                <div className="flex items-center gap-2 text-ink/40">
                  <Clock size={12} />
                  <span className="text-[9px] font-black font-mono uppercase">
                    {new Date(event.received_at).toLocaleTimeString([], { hour12: false })}
                  </span>
                </div>
                <span className="text-[8px] font-bold text-ink/20 uppercase tracking-tighter">
                  {new Date(event.received_at).toLocaleDateString()}
                </span>
              </div>

              <div className="flex-1 flex items-center gap-4">
                <div className={`p-2 border-2 border-ink ${event.process_error || event.event_type.includes('fail') ? 'bg-red-500 text-white' : 'bg-bg text-ink'}`}>
                  <Terminal size={14} />
                </div>
                <div>
                  <h4 className="text-xs font-black uppercase tracking-tight mb-1">{event.event_type}</h4>
                  <div className="flex items-center gap-2 md:gap-4 flex-wrap">
                    <span className="px-1.5 py-0.5 bg-zinc-100 border border-zinc-200 text-[8px] font-black font-mono text-ink/60 uppercase">
                      {event.source.toUpperCase()}
                    </span>
                    {event.repository && (
                      <>
                        <span className="w-1 h-1 rounded-full bg-ink/20 hidden md:block" />
                        <span className="text-[9px] font-medium text-ink/60 break-all">{event.repository}</span>
                      </>
                    )}
                    {event.process_error && (
                      <span className="text-[9px] font-medium text-red-500 break-all">{event.process_error}</span>
                    )}
                  </div>
                </div>
              </div>

              <button
                onClick={() => handleReview(event)}
                className="p-2 border-2 border-ink hover:bg-ink hover:text-bg transition-all self-end md:self-center md:opacity-0 group-hover:opacity-100"
              >
                <ChevronRight size={14} />
              </button>
            </div>
          ))}
          {filteredEvents.length === 0 && (
            <div className="p-24 text-center">
              <p className="text-sm font-black uppercase opacity-20 tracking-widest text-center">No events matching protocol</p>
            </div>
          )}
        </div>
      </div>
    </motion.div>
  );
};
