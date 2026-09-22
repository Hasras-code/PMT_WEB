import type { ReactNode } from 'react';
import type { Status } from '../types';

export function Card({ children, className = '' }: { children: ReactNode; className?: string }) {
  return (
    <div className={`bg-charcoal-card backdrop-blur-md rounded-2xl border border-line shadow-lg ${className}`}>
      {children}
    </div>
  );
}

export function CardTitle({ children }: { children: ReactNode }) {
  return <h2 className="text-lg font-semibold text-ink">{children}</h2>;
}

export function PrimaryButton({
  children,
  onClick,
  type = 'button',
  disabled = false,
  className = '',
}: {
  children: ReactNode;
  onClick?: () => void;
  type?: 'button' | 'submit';
  disabled?: boolean;
  className?: string;
}) {
  return (
    <button
      type={type}
      disabled={disabled}
      onClick={onClick}
      className={`w-full px-5 py-3 rounded-xl bg-gradient-to-r from-amber-500 via-yellow-500 to-amber-600 text-slate-950 text-[15px] font-bold hover:shadow-gold-glow hover:scale-[1.02] transition-all duration-300 disabled:opacity-50 disabled:hover:scale-100 ${className}`}
    >
      {children}
    </button>
  );
}

export function OutlineButton({
  children,
  onClick,
  type = 'button',
  className = '',
}: {
  children: ReactNode;
  onClick?: () => void;
  type?: 'button' | 'submit';
  className?: string;
}) {
  return (
    <button
      type={type}
      onClick={onClick}
      className={`w-full px-5 py-3 rounded-xl border border-amber-500/40 bg-charcoal-card text-primary text-[15px] font-medium hover:bg-primary-light transition-colors ${className}`}
    >
      {children}
    </button>
  );
}

export function Badge({ tone = 'gray', children }: { tone?: 'gray' | 'green' | 'blue' | 'orange' | 'red' | 'purple'; children: ReactNode }) {
  const tones: Record<string, string> = {
    gray: 'bg-surface text-muted',
    green: 'bg-emerald-50 text-emerald-700',
    blue: 'bg-primary-light text-primary',
    orange: 'bg-orange-50 text-orange-700',
    red: 'bg-red-50 text-red-700',
    purple: 'bg-purple-50 text-purple-700',
  };
  return <span className={`inline-flex px-2.5 py-1 rounded-full text-xs font-medium ${tones[tone]}`}>{children}</span>;
}

export function statusTone(status: Status): 'gray' | 'green' | 'blue' | 'orange' | 'red' | 'purple' {
  const s = status.toUpperCase();
  if (s === 'PUBLISHED' || s === 'ACTIVE' || s === 'RESOLVED' || s === 'REVIEWED' || s === 'CURRENT') return 'green';
  if (s === 'DRAFT' || s === 'NEW' || s === 'OPEN') return 'blue';
  if (s === 'IN_REVIEW' || s === 'PENDING') return 'orange';
  if (s === 'SUSPENDED' || s === 'ARCHIVED' || s === 'CLOSED') return 'gray';
  if (s === 'URGENT' || s === 'LEFT' || s === 'GRADUATED') return 'purple';
  return 'gray';
}

export function Empty({ message }: { message: string }) {
  return <p className="py-8 text-center text-sm text-muted">{message}</p>;
}

export function Field({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <label className="block">
      <span className="block text-sm font-medium text-ink mb-1.5">{label}</span>
      {children}
    </label>
  );
}

export const inputCls =
  'w-full px-3.5 py-2.5 rounded-xl border border-line bg-slate-950/80 text-[15px] text-ink placeholder:text-muted/60 focus:outline-none focus:ring-2 focus:ring-amber-500/50 focus:border-amber-500';

export function timeAgo(iso: string | null | undefined): string {
  if (!iso) return '';
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins} min${mins > 1 ? 's' : ''} ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours} hour${hours > 1 ? 's' : ''} ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days} day${days > 1 ? 's' : ''} ago`;
  return new Date(iso).toLocaleDateString();
}

export function fmtDate(iso: string | null | undefined): string {
  if (!iso) return '—';
  return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
}

export function fmtDateTime(iso: string | null | undefined): string {
  if (!iso) return '—';
  return new Date(iso).toLocaleString(undefined, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
}

const COVER_BG = ['#205BFF', '#7C3AED', '#059669', '#EA580C', '#DB2777', '#0891B2'];

export function coverColor(seed: string): string {
  let h = 0;
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) % 997;
  return COVER_BG[h % COVER_BG.length] ?? '#205BFF';
}

export function initials(name: string): string {
  return name
    .split(/\s+/)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() || '')
    .join('');
}
