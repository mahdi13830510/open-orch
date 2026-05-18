import React from 'react';
import { motion } from 'framer-motion';
import { GitBranch, History, ChevronRight } from 'lucide-react';
import { Repository } from '../types';
import { StatusBadge } from './StatusBadge';

interface RepositoryCardProps {
  repo: Repository;
  isSelected?: boolean;
  onClick?: () => void;
  onEdit?: (repo: Repository) => void;
  onViewSpecs?: (repo: Repository) => void;
}

export const RepositoryCard: React.FC<RepositoryCardProps> = ({ 
  repo, 
  isSelected, 
  onClick, 
  onEdit, 
  onViewSpecs 
}) => {
  return (
    <motion.div 
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      whileHover={{ y: -4 }}
      onClick={onClick}
      className={`group cursor-pointer border-4 transition-all ${
        isSelected ? 'border-ink bg-surface shadow-[8px_8px_0_rgba(20,20,20,1)]' : 'border-ink/10 bg-white hover:border-ink/40'
      }`}
    >
      <div className="p-5 space-y-4">
        <div className="flex justify-between items-start">
          <div className="flex items-center gap-3">
            <div className={`p-2 border-2 border-ink ${isSelected ? 'bg-ink text-bg' : 'bg-bg text-ink'}`}>
              {repo.service_kind === 'http' ? '🌐' : '⚙️'}
            </div>
            <div>
              <h3 className="text-sm font-black uppercase tracking-tight">{repo.name}</h3>
              <p className="text-[9px] font-bold opacity-40 uppercase">{repo.full_name}</p>
            </div>
          </div>
          <StatusBadge status="healthy" size="sm" />
        </div>

        <div className="flex gap-4 border-t-2 border-ink/5 pt-4">
          <div className="flex items-center gap-1.5 opacity-60">
            <GitBranch size={12} strokeWidth={3} />
            <span className="text-[9px] font-black uppercase tracking-widest">main</span>
          </div>
          <div className="flex items-center gap-1.5 opacity-60">
            <History size={12} strokeWidth={3} />
            <span className="text-[9px] font-black uppercase tracking-widest">v1.2.4</span>
          </div>
        </div>

        <div className="flex gap-2 pt-2 md:opacity-0 group-hover:opacity-100 transition-opacity">
          <button 
            onClick={(e) => { e.stopPropagation(); onEdit?.(repo); }}
            className="flex-1 py-1.5 bg-ink text-bg text-[9px] font-black uppercase hover:invert transition-all"
          >
            EDIT
          </button>
          <button 
            onClick={(e) => { e.stopPropagation(); onViewSpecs?.(repo); }}
            className="flex-1 py-1.5 border-2 border-ink text-ink text-[9px] font-black uppercase hover:bg-ink hover:text-bg transition-all"
          >
            SPECS
          </button>
        </div>
      </div>
    </motion.div>
  );
};
