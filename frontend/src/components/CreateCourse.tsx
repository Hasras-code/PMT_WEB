import { useEffect, useId, useRef, useState } from 'react';
import { api, errMsg } from '../api/client';
import { useAppStore } from '../store/app';
import toast from 'react-hot-toast';
import type { Batch } from '../types';
import { Field, inputCls, PrimaryButton } from './ui';

export default function CreateCourse({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { setBatches } = useAppStore();
  const [form, setForm] = useState({ name: '', slug: '', entry_year: new Date().getFullYear(), graduation_year: '', description: '' });
  const [saving, setSaving] = useState(false);
  const titleId = useId();
  const firstInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!open) return;
    firstInputRef.current?.focus();
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.stopPropagation();
        onClose();
      }
    };
    document.addEventListener('keydown', onKey, true);
    return () => document.removeEventListener('keydown', onKey, true);
  }, [open, onClose]);

  if (!open) return null;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      const body: Record<string, unknown> = {
        name: form.name,
        slug: form.slug,
        entry_year: Number(form.entry_year),
        description: form.description,
      };
      if (form.graduation_year !== '') body.graduation_year = Number(form.graduation_year);
      await api.post('/v1/admin/batches', body);
      const res = await api.get('/v1/batches');
      setBatches(res.data as Batch[]);
      toast.success('Course created');
      onClose();
    } catch (err) {
      toast.error(errMsg(err, 'Could not create course (platform admin only)'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-ink/40 p-4"
      onClick={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div role="dialog" aria-modal="true" aria-labelledby={titleId} className="w-full max-w-lg bg-charcoal-card border border-line rounded-2xl p-7 shadow-2xl" onClick={(e) => e.stopPropagation()}>
        <h2 id={titleId} className="text-xl font-semibold text-ink">Create Course</h2>
        <p className="text-sm text-muted mt-1">Requires the platform batch.create permission.</p>
        <form onSubmit={submit} className="mt-5 space-y-4">
          <Field label="Name">
            <input ref={firstInputRef} required className={inputCls} value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </Field>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Field label="Slug (lowercase, dashes)">
              <input required pattern="[a-z0-9]+(?:-[a-z0-9]+)*" placeholder="cohort-2026" className={inputCls} value={form.slug} onChange={(e) => setForm({ ...form, slug: e.target.value })} />
            </Field>
            <Field label="Entry year">
              <input required type="number" min={1900} max={2200} className={inputCls} value={form.entry_year} onChange={(e) => setForm({ ...form, entry_year: Number(e.target.value) })} />
            </Field>
          </div>
          <Field label="Graduation year (optional)">
            <input type="number" min={1900} max={2200} className={inputCls} value={form.graduation_year} onChange={(e) => setForm({ ...form, graduation_year: e.target.value })} />
          </Field>
          <Field label="Description">
            <textarea rows={3} className={inputCls} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
          </Field>
          <div className="flex gap-3">
            <PrimaryButton type="submit" disabled={saving}>
              {saving ? 'Creating…' : 'Create course'}
            </PrimaryButton>
            <button type="button" onClick={onClose} className="px-5 py-3 rounded-xl border border-line text-ink text-[15px] font-medium">
              Cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
