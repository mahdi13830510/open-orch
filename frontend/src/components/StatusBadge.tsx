import React from 'react';
import { motion } from 'framer-motion';
import { 
  CheckCircle2, 
  AlertCircle, 
  Loader2, 
  XCircle, 
  CircleDot,
  ShieldCheck,
  ShieldAlert
} from 'lucide-react';

export type StatusType = 'healthy' | 'running' | 'failed' | 'warning' | 'pending' | 'stopped';

interface StatusBadgeProps {
  status: StatusType | string;
  size?: 'sm' | 'md' | 'lg';
  showLabel?: boolean;
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ 
  status, 
  size = 'md', 
  showLabel = true 
}) => {
  const config = {
    healthy: {
      color: 'text-green-600',
      bg: 'bg-green-50',
      border: 'border-green-200',
      icon: CheckCircle2,
      label: 'Healthy',
      animate: false
    },
    running: {
      color: 'text-blue-600',
      bg: 'bg-blue-50',
      border: 'border-blue-200',
      icon: Loader2,
      label: 'Running',
      animate: true
    },
    failed: {
      color: 'text-red-600',
      bg: 'bg-red-50',
      border: 'border-red-200',
      icon: XCircle,
      label: 'Failed',
      animate: false
    },
    warning: {
      color: 'text-amber-600',
      bg: 'bg-amber-50',
      border: 'border-amber-200',
      icon: AlertCircle,
      label: 'Degraded',
      animate: false
    },
    pending: {
      color: 'text-zinc-400',
      bg: 'bg-zinc-50',
      border: 'border-zinc-200',
      icon: CircleDot,
      label: 'Provisioning',
      animate: true
    },
    stopped: {
      color: 'text-zinc-600',
      bg: 'bg-zinc-100',
      border: 'border-zinc-300',
      icon: ShieldAlert,
      label: 'Halted',
      animate: false
    }
  };

  const s = (config[status as StatusType] || config.pending);
  const Icon = s.icon;

  const sizeClasses = {
    sm: 'px-1.5 py-0.5 text-[9px] gap-1',
    md: 'px-2 py-1 text-[10px] gap-1.5',
    lg: 'px-3 py-1.5 text-xs gap-2'
  };

  return (
    <div className={`flex items-center ${sizeClasses[size]} font-black font-mono uppercase border-2 ${s.bg} ${s.color} ${s.border} shadow-[2px_2px_0_rgba(20,20,20,0.1)]`}>
      <motion.div
        animate={s.animate ? { rotate: 360 } : {}}
        transition={s.animate ? { repeat: Infinity, duration: 2, ease: "linear" } : {}}
      >
        <Icon size={size === 'sm' ? 10 : size === 'md' ? 12 : 16} strokeWidth={3} />
      </motion.div>
      {showLabel && <span>{s.label}</span>}
      {s.animate && (
        <motion.div 
          className="w-1.5 h-1.5 rounded-full bg-current"
          animate={{ opacity: [1, 0.4, 1] }}
          transition={{ repeat: Infinity, duration: 1.5 }}
        />
      )}
    </div>
  );
};
