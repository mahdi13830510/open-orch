import React from 'react';
import { motion } from 'framer-motion';
import { Trash2, Edit3, CheckCircle, AlertCircle, Zap } from 'lucide-react';
import { Integration } from '../types';

interface IntegrationCardProps {
  integration: Integration;
  onEdit: (integration: Integration) => void;
  onDelete: (id: string) => void;
}

export const IntegrationCard: React.FC<IntegrationCardProps> = ({
  integration,
  onEdit,
  onDelete,
}) => {
  return (
    <motion.div
      layout
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.8 }}
      className="p-4 border-2 border-ink bg-white flex justify-between items-center group shadow-[2px_2px_0_rgba(20,20,20,1)] hover:shadow-[4px_4px_0_rgba(20,20,20,1)] transition-all hover:-translate-y-1"
    >
      <div className="flex items-center gap-4">
        <div className="w-10 h-10 border-2 border-ink flex items-center justify-center bg-bg group-hover:bg-ink group-hover:text-bg transition-colors">
          <Zap size={18} fill="currentColor" strokeWidth={0} />
        </div>
        <div>
          <h4 className="text-[11px] font-black uppercase font-mono tracking-tight">{integration.name}</h4>
          <span className="text-[8px] font-black uppercase opacity-40 px-1 border border-ink/20 inline-block mt-1">
            {integration.kind}
          </span>
          {integration.last_verified_at && !integration.last_error && (
            <span className="ml-2 text-[8px] text-emerald-600 inline-flex items-center gap-1">
              <CheckCircle size={8} /> verified
            </span>
          )}
          {integration.last_error && (
            <span className="ml-2 text-[8px] text-red-500 inline-flex items-center gap-1">
              <AlertCircle size={8} /> error
            </span>
          )}
        </div>
      </div>
      <div className="flex gap-2 md:opacity-0 group-hover:opacity-100 transition-opacity">
        <button
          onClick={() => onEdit(integration)}
          className="p-1.5 hover:bg-ink hover:text-bg transition-colors border border-transparent hover:border-ink"
        >
          <Edit3 size={12} />
        </button>
        <button
          onClick={() => onDelete(integration.id)}
          className="p-1.5 hover:bg-red-500 hover:text-white transition-colors border border-transparent hover:border-red-600"
        >
          <Trash2 size={12} />
        </button>
      </div>
    </motion.div>
  );
};
