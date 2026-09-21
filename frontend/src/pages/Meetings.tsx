import { useEffect, useState } from 'react';
import { VideoCameraIcon, MapPinIcon, ClockIcon } from '@heroicons/react/24/outline';
import { api, toList, errMsg } from '../api/client';
import { useAppStore } from '../store/app';
import toast from 'react-hot-toast';
import type { LmsEvent } from '../types';
import { useElevated } from '../hooks/useRole';
import { Card, Badge, statusTone, Empty, Field, inputCls, fmtDateTime } from '../components/ui';

export default function Meetings() {
  const { currentBatchID } = useAppStore();
  const [items, setItems] = useState<LmsEvent[]>([]);
  const [show, setShow] = useState(false);
  const elevated = useElevated();
  const [form, setForm] = useState({ title: '', description: '', location: '', starts_at: '', ends_at: '', visibility: 'MEMBERS_ONLY' });

  const load = () => {
    if (!currentBatchID) return;
    api.get(`/v1/batches/${currentBatchID}/events`, { limit: 100 }).then((res) => setItems(toList<LmsEvent>(res.data))).catch(() => {});
  };
  useEffect(load, [currentBatchID]);

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    const body: Record<string, unknown> = {
      title: form.title,
      starts_at: new Date(form.starts_at).toISOString(),
      visibility: form.visibility,
    };
    if (form.description) body.description = form.description;
    if (form.location) body.location = form.location;
    if (form.ends_at) body.ends_at = new Date(form.ends_at).toISOString();
    try {
      await api.post(`/v1/batches/${currentBatchID}/events`, body);
      toast.success('Meeting scheduled');
      setShow(false);
      setForm({ title: '', description: '', location: '', starts_at: '', ends_at: '', visibility: 'MEMBERS_ONLY' });
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Create failed'));
    }
  };

  if (!currentBatchID) {
    return (
      <Card className="p-8">
        <Empty message="Select a cohort from the dropdown above to view its meetings." />
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <p className="text-muted text-[15px]">{items.length} meeting{items.length === 1 ? '' : 's'} scheduled</p>
        {elevated && (<button onClick={() => setShow(!show)} className="px-5 py-2.5 rounded-xl bg-primary text-white text-[15px] font-medium hover:bg-primary-dark">
          + Schedule meeting
        </button>)}
      </div>

      {show && elevated && (
        <Card className="p-7">
          <form onSubmit={create} className="grid grid-cols-2 gap-4">
            <div className="col-span-2">
              <Field label="Title"><input required className={inputCls} value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} /></Field>
            </div>
            <div className="col-span-2">
              <Field label="Description"><textarea rows={2} className={inputCls} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></Field>
            </div>
            <Field label="Location / meeting link"><input placeholder="Room 4 or https://…" className={inputCls} value={form.location} onChange={(e) => setForm({ ...form, location: e.target.value })} /></Field>
            <Field label="Visibility">
              <select className={inputCls} value={form.visibility} onChange={(e) => setForm({ ...form, visibility: e.target.value })}>
                <option value="MEMBERS_ONLY">Members only</option>
                <option value="PUBLIC">Public</option>
              </select>
            </Field>
            <Field label="Starts at"><input required type="datetime-local" className={inputCls} value={form.starts_at} onChange={(e) => setForm({ ...form, starts_at: e.target.value })} /></Field>
            <Field label="Ends at"><input type="datetime-local" className={inputCls} value={form.ends_at} onChange={(e) => setForm({ ...form, ends_at: e.target.value })} /></Field>
            <div className="col-span-2">
              <button className="px-5 py-2.5 rounded-xl bg-emerald-600 text-white text-sm font-medium">Schedule</button>
            </div>
          </form>
        </Card>
      )}

      {items.length === 0 && <Card className="p-8"><Empty message="No meetings scheduled." /></Card>}
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
        {items.map((ev) => (
          <Card key={ev.id} className="p-6">
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-xl bg-primary-light text-primary flex items-center justify-center shrink-0">
                <VideoCameraIcon className="w-6 h-6" strokeWidth={1.8} />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-start justify-between gap-3">
                  <h3 className="font-semibold text-ink text-[17px]">{ev.title}</h3>
                  <Badge tone={statusTone(ev.status)}>{ev.status}</Badge>
                </div>
                <p className="text-sm text-muted mt-1 line-clamp-2">{ev.description || 'No description.'}</p>
                <div className="mt-3 space-y-1 text-sm text-muted">
                  <p className="flex items-center gap-2"><ClockIcon className="w-4 h-4" />{fmtDateTime(ev.starts_at)}{ev.ends_at ? ` → ${fmtDateTime(ev.ends_at)}` : ''}</p>
                  {ev.location && <p className="flex items-center gap-2"><MapPinIcon className="w-4 h-4" />{ev.location}</p>}
                </div>
                <div className="mt-3 flex gap-3 text-xs font-medium">
                  <span className="text-muted">{ev.visibility.replaceAll('_', ' ')}</span>
                  {elevated && ev.status !== 'PUBLISHED' && (
                    <button onClick={() => api.post(`/v1/batches/${currentBatchID}/events/${ev.id}/publish`).then(() => { toast.success('Published'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-emerald-600 hover:underline">Publish</button>
                  )}
                  {elevated && (<button onClick={() => api.delete(`/v1/batches/${currentBatchID}/events/${ev.id}`).then(() => { toast.success('Cancelled'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-red-600 hover:underline">Cancel</button>)}
                </div>
              </div>
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}
