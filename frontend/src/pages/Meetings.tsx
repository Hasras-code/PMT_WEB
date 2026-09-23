import { useEffect, useState } from 'react';
import { PlayCircleIcon, ClockIcon } from '@heroicons/react/24/outline';
import { api, toList, errMsg } from '../api/client';
import { useAppStore } from '../store/app';
import toast from 'react-hot-toast';
import type { Kuppi, Module } from '../types';
import { useAccess, useCan } from '../hooks/useRole';
import { usePatchEditor } from '../hooks/usePatchEditor';
import { Card, Badge, statusTone, Empty, Field, inputCls, fmtDateTime } from '../components/ui';
import KuppiPlayer from '../components/KuppiPlayer';

export default function Meetings() {
  const { currentBatchID } = useAppStore();
  const access = useAccess();
  const [items, setItems] = useState<Kuppi[]>([]);
  const [modules, setModules] = useState<Module[]>([]);
  const [show, setShow] = useState(false);
  const membershipRoles = access?.memberships.find((membership) => membership.batch_id === currentBatchID)?.roles || [];
  const elevated = useCan('kuppi.create', currentBatchID) || membershipRoles.some((role) => role === 'BATCH_REP' || role === 'ACADEMIC_REP') || access?.platform_roles.includes('PLATFORM_ADMIN') === true;
  const [form, setForm] = useState({ module_id: '', title: '', description: '', youtube_url: '', recorded_at: '', duration_seconds: '' });
  const [creating, setCreating] = useState(false);

  const load = () => {
    if (!currentBatchID) return;
    api.get(`/v1/batches/${currentBatchID}/kuppis`, { limit: 100 }).then((res) => setItems(toList<Kuppi>(res.data))).catch(() => {});
    api.get(`/v1/batches/${currentBatchID}/modules`, { limit: 100 }).then((res) => setModules(toList<Module>(res.data))).catch(() => {});
  };
  useEffect(load, [currentBatchID]);
  const editor = usePatchEditor(load, 'Kuppi updated');

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    if (creating) return;
    const body: Record<string, unknown> = { module_id: form.module_id, title: form.title, youtube_url: form.youtube_url };
    if (form.description) body.description = form.description;
    if (form.recorded_at) body.recorded_at = new Date(form.recorded_at).toISOString();
    if (form.duration_seconds) body.duration_seconds = Number(form.duration_seconds);
    setCreating(true);
    try {
      await api.post(`/v1/batches/${currentBatchID}/kuppis`, body);
      toast.success('Kuppi created as a draft');
      setShow(false);
      setForm({ module_id: '', title: '', description: '', youtube_url: '', recorded_at: '', duration_seconds: '' });
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Create failed'));
    } finally {
      setCreating(false);
    }
  };

  const editKuppi = (item: Kuppi) => {
    editor.open('Edit Kuppi', `/v1/batches/${currentBatchID}/kuppis/${item.id}`, [
      { key: 'title', label: 'Kuppi title', value: item.title, required: true },
      { key: 'description', label: 'Description', value: item.description || '', multiline: true },
      { key: 'youtube_url', label: 'YouTube URL', value: item.watch_url, required: true },
    ]);
  };

  if (!currentBatchID) {
    return (
      <Card className="p-8">
        <Empty message="Select a cohort from the dropdown above to view its Kuppis." />
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <p className="text-muted text-[15px]">{items.length} Kuppi{items.length === 1 ? '' : 's'}</p>
        {elevated && (<button onClick={() => setShow(!show)} className="px-5 py-2.5 rounded-xl bg-primary text-white text-[15px] font-medium hover:bg-primary-dark">
          + Add Kuppi
        </button>)}
      </div>

      {show && elevated && (
        <Card className="p-7">
          <form onSubmit={create} className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Field label="Module"><select required className={inputCls} value={form.module_id} onChange={(e) => setForm({ ...form, module_id: e.target.value })}><option value="">Select…</option>{modules.map((m) => <option key={m.id} value={m.id}>{m.module_code} — {m.name}</option>)}</select></Field>
            <Field label="Title"><input required className={inputCls} value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} /></Field>
            <div className="col-span-2">
              <Field label="YouTube URL"><input required type="url" placeholder="https://youtu.be/..." className={inputCls} value={form.youtube_url} onChange={(e) => setForm({ ...form, youtube_url: e.target.value })} /></Field>
            </div>
            <Field label="Recorded at"><input type="datetime-local" className={inputCls} value={form.recorded_at} onChange={(e) => setForm({ ...form, recorded_at: e.target.value })} /></Field>
            <Field label="Duration (seconds)"><input type="number" min={1} className={inputCls} value={form.duration_seconds} onChange={(e) => setForm({ ...form, duration_seconds: e.target.value })} /></Field>
            <div className="col-span-2"><Field label="Description"><textarea rows={2} className={inputCls} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></Field></div>
            <div className="col-span-2">
              <button disabled={creating} className="px-5 py-2.5 rounded-xl bg-emerald-600 text-white text-sm font-medium disabled:opacity-50">{creating ? 'Creating…' : 'Create draft'}</button>
            </div>
          </form>
        </Card>
      )}

      {items.length === 0 && <Card className="p-8"><Empty message="No Kuppis published yet." /></Card>}
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
        {items.map((ev) => (
          <Card key={ev.id} className="p-6 space-y-5">
            <KuppiPlayer embedUrl={ev.embed_url} title={ev.title} />
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-xl bg-primary-light text-primary flex items-center justify-center shrink-0">
                <PlayCircleIcon className="w-6 h-6" strokeWidth={1.8} />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-start justify-between gap-3">
                  <h3 className="font-semibold text-ink text-[17px]">{ev.title}</h3>
                  <Badge tone={statusTone(ev.status)}>{ev.status}</Badge>
                </div>
                <p className="text-sm text-muted mt-1">{ev.module.code} · {ev.module.name}</p>
                <p className="text-sm text-muted mt-1 line-clamp-2">{ev.description || 'No description.'}</p>
                <div className="mt-3 space-y-1 text-sm text-muted">
                  {ev.recorded_at && <p className="flex items-center gap-2"><ClockIcon className="w-4 h-4" />{fmtDateTime(ev.recorded_at)}{ev.duration_seconds ? ` · ${Math.round(ev.duration_seconds / 60)} min` : ''}</p>}
                </div>
                <div className="mt-4 flex flex-wrap gap-x-3 gap-y-2 text-xs font-medium">
                  <a className="text-primary hover:underline" href={ev.watch_url} target="_blank" rel="noreferrer">Watch recording</a>
                  {elevated && ev.status === 'DRAFT' && (
                    <button onClick={() => api.post(`/v1/batches/${currentBatchID}/kuppis/${ev.id}/publish`).then(() => { toast.success('Published'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-emerald-600 hover:underline">Publish</button>
                  )}
                  {elevated && ev.status !== 'ARCHIVED' && <button onClick={() => editKuppi(ev)} className="text-primary hover:underline">Edit</button>}
                  {elevated && ev.status !== 'ARCHIVED' && (<button onClick={() => api.post(`/v1/batches/${currentBatchID}/kuppis/${ev.id}/archive`).then(() => { toast.success('Archived'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-red-600 hover:underline">Archive</button>)}
                </div>
              </div>
            </div>
          </Card>
        ))}
      </div>
      {editor.dialog}
    </div>
  );
}
