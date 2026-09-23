import type { ReactNode } from 'react';
import { useEffect, useId, useRef, useState } from 'react';
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

/** Shimmering placeholder shown while content loads (theme-safe). */
export function Skeleton({ className = '' }: { className?: string }) {
  return <div aria-hidden="true" className={`animate-pulse rounded-lg bg-surface ${className}`} />;
}

/** Persistent inline error with a retry action (for failed loads). */
export function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="py-8 text-center" role="alert">
      <p className="text-sm text-muted">{message}</p>
      <button
        type="button"
        onClick={onRetry}
        className="mt-3 px-5 py-2 rounded-xl border border-line text-sm font-medium text-ink hover:bg-surface"
      >
        Try again
      </button>
    </div>
  );
}

/** Accessible tab bar: tablist pattern with arrow-key navigation. */
export function Tabs<T extends string>({
  tabs,
  active,
  onChange,
  labels,
}: {
  tabs: readonly T[];
  active: T;
  onChange: (tab: T) => void;
  labels?: Partial<Record<T, string>>;
}) {
  const refs = useRef<Array<HTMLButtonElement | null>>([]);

  const move = (event: React.KeyboardEvent, index: number) => {
    let next: number | null = null;
    if (event.key === 'ArrowRight') next = (index + 1) % tabs.length;
    else if (event.key === 'ArrowLeft') next = (index - 1 + tabs.length) % tabs.length;
    else if (event.key === 'Home') next = 0;
    else if (event.key === 'End') next = tabs.length - 1;
    if (next === null) return;
    event.preventDefault();
    const target = tabs[next];
    if (target === undefined) return;
    refs.current[next]?.focus();
    onChange(target);
  };

  return (
    <div role="tablist" aria-label="Sections" className="flex gap-1 overflow-x-auto">
      {tabs.map((t, index) => {
        const selected = t === active;
        return (
          <button
            key={t}
            ref={(el) => {
              refs.current[index] = el;
            }}
            role="tab"
            aria-selected={selected}
            tabIndex={selected ? 0 : -1}
            onClick={() => onChange(t)}
            onKeyDown={(event) => move(event, index)}
            className={`px-4 py-3 text-[15px] whitespace-nowrap border-b-2 -mb-px ${
              selected ? 'border-primary text-primary font-medium' : 'border-transparent text-muted hover:text-ink'
            }`}
          >
            {labels?.[t] ?? t}
          </button>
        );
      })}
    </div>
  );
}

/** Accessible replacement for window.prompt(): labelled fields in a modal
 * dialog with Escape-to-close, initial focus, focus trap and inline errors.
 * Parents must pass a `key` that changes per edited item so values reset. */
export interface EditDialogField {
  key: string;
  label: string;
  value: string;
  multiline?: boolean;
  required?: boolean;
}

export function EditDialog({
  title,
  fields,
  busy,
  submitLabel = 'Save',
  onClose,
  onSubmit,
}: {
  title: string;
  fields: EditDialogField[];
  busy: boolean;
  submitLabel?: string;
  onClose: () => void;
  onSubmit: (values: Record<string, string>) => void;
}) {
  const titleId = useId();
  const descId = useId();
  const dialogRef = useRef<HTMLDivElement>(null);
  const firstInputRef = useRef<HTMLInputElement | HTMLTextAreaElement | null>(null);
  const [values, setValues] = useState<Record<string, string>>(() =>
    Object.fromEntries(fields.map((f) => [f.key, f.value])),
  );
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    firstInputRef.current?.focus();
  }, []);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.stopPropagation();
        onClose();
        return;
      }
      // Minimal focus trap: cycle Tab inside the dialog.
      if (event.key !== 'Tab' || !dialogRef.current) return;
      const focusables = Array.from(
        dialogRef.current.querySelectorAll<HTMLElement>(
          'button:not([disabled]), input, textarea, select, [tabindex]:not([tabindex="-1"])',
        ),
      );
      if (focusables.length === 0) return;
      const first = focusables[0];
      const last = focusables[focusables.length - 1];
      if (!first || !last) return;
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };
    document.addEventListener('keydown', onKey, true);
    return () => document.removeEventListener('keydown', onKey, true);
  }, [onClose]);

  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    const nextErrors: Record<string, string> = {};
    for (const field of fields) {
      if (field.required && !(values[field.key] ?? '').trim()) {
        nextErrors[field.key] = `${field.label} is required.`;
      }
    }
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) return;
    onSubmit(values);
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-ink/50 p-4"
      onClick={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={descId}
        className="w-full max-w-lg rounded-2xl border border-line bg-charcoal-card p-7 shadow-2xl"
      >
        <h2 id={titleId} className="text-lg font-semibold text-ink">
          {title}
        </h2>
        <p id={descId} className="mt-1 text-sm text-muted">
          Press Escape or click outside to cancel.
        </p>
        <form onSubmit={submit} className="mt-5 space-y-4" noValidate>
          {fields.map((field, index) => {
            const errorId = `${descId}-${field.key}`;
            const invalid = !!errors[field.key];
            const control = field.multiline ? (
              <textarea
                ref={index === 0 ? (el) => { firstInputRef.current = el; } : undefined}
                value={values[field.key] ?? ''}
                rows={3}
                aria-invalid={invalid}
                aria-describedby={invalid ? errorId : undefined}
                onChange={(event) => setValues((v) => ({ ...v, [field.key]: event.target.value }))}
                className={inputCls}
              />
            ) : (
              <input
                ref={index === 0 ? (el) => { firstInputRef.current = el; } : undefined}
                value={values[field.key] ?? ''}
                aria-invalid={invalid}
                aria-describedby={invalid ? errorId : undefined}
                onChange={(event) => setValues((v) => ({ ...v, [field.key]: event.target.value }))}
                className={inputCls}
              />
            );
            return (
              <div key={field.key}>
                <Field label={field.required ? `${field.label} *` : field.label}>{control}</Field>
                {invalid && (
                  <p id={errorId} role="alert" className="mt-1.5 text-xs text-red-600">
                    {errors[field.key]}
                  </p>
                )}
              </div>
            );
          })}
          <div className="flex justify-end gap-3 pt-1">
            <button
              type="button"
              onClick={onClose}
              className="rounded-xl border border-line px-5 py-2.5 text-sm font-medium text-ink hover:bg-surface"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={busy}
              className="rounded-xl bg-primary px-5 py-2.5 text-sm font-bold text-slate-950 hover:opacity-90 disabled:opacity-50"
            >
              {busy ? 'Saving…' : submitLabel}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
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

const COVER_BG = ['#D4AF37', '#7C3AED', '#059669', '#EA580C', '#DB2777', '#F59E0B'];

export function coverColor(seed: string): string {
  let h = 0;
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) % 997;
  return COVER_BG[h % COVER_BG.length] ?? '#D4AF37';
}

export function initials(name: string): string {
  return name
    .split(/\s+/)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() || '')
    .join('');
}
