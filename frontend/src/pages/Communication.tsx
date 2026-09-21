import { useEffect, useState } from 'react';
import { api, toList, errMsg } from '../api/client';
import { useAppStore } from '../store/app';
import toast from 'react-hot-toast';
import type { Announcement, Complaint, FeedbackItem } from '../types';
import { useElevated } from '../hooks/useRole';
import { Card, Badge, statusTone, Empty, Field, inputCls, timeAgo } from '../components/ui';

const TABS = ['Announcements', 'Complaints', 'Feedback'] as const;
const PRIORITIES = ['NORMAL', 'IMPORTANT', 'URGENT'];

export default function Communication() {
  const { currentBatchID } = useAppStore();
  const [tab, setTab] = useState<(typeof TABS)[number]>('Announcements');

  if (!currentBatchID) {
    return (
      <Card className="p-8">
        <Empty message="Select a cohort from the dropdown above." />
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <Card className="px-4 pt-2 flex gap-1 overflow-x-auto">
        {TABS.map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-4 py-3 text-[15px] whitespace-nowrap border-b-2 -mb-px ${
              tab === t ? 'border-primary text-primary font-medium' : 'border-transparent text-muted hover:text-ink'
            }`}
          >
            {t}
          </button>
        ))}
      </Card>
      {tab === 'Announcements' && <AnnouncementsTab batchID={currentBatchID} />}
      {tab === 'Complaints' && <ComplaintsTab batchID={currentBatchID} />}
      {tab === 'Feedback' && <FeedbackTab batchID={currentBatchID} />}
    </div>
  );
}

function AnnouncementsTab({ batchID }: { batchID: string }) {
  const [items, setItems] = useState<Announcement[]>([]);
  const [form, setForm] = useState({ title: '', body: '', priority: 'NORMAL' });
  const elevated = useElevated();
  const [show, setShow] = useState(false);

  const load = () => api.get(`/v1/batches/${batchID}/announcements`, { limit: 100 }).then((res) => setItems(toList<Announcement>(res.data))).catch(() => {});
  useEffect(() => {
    load();
  }, [batchID]);

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.post(`/v1/batches/${batchID}/announcements`, { title: form.title, body: form.body, priority: form.priority });
      toast.success('Announcement created');
      setShow(false);
      setForm({ title: '', body: '', priority: 'NORMAL' });
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Create failed'));
    }
  };

  return (
    <div className="space-y-5">
      <div className="flex justify-end">
        {elevated && (<button onClick={() => setShow(!show)} className="px-5 py-2.5 rounded-xl bg-primary text-white text-[15px] font-medium hover:bg-primary-dark">
          + New announcement
        </button>)}
      </div>
      {show && elevated && (
        <Card className="p-7">
          <form onSubmit={create} className="space-y-4">
            <div className="grid grid-cols-3 gap-4">
              <div className="col-span-2">
                <Field label="Title"><input required className={inputCls} value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} /></Field>
              </div>
              <Field label="Priority">
                <select className={inputCls} value={form.priority} onChange={(e) => setForm({ ...form, priority: e.target.value })}>
                  {PRIORITIES.map((p) => <option key={p} value={p}>{p}</option>)}
                </select>
              </Field>
            </div>
            <Field label="Body"><textarea required rows={4} className={inputCls} value={form.body} onChange={(e) => setForm({ ...form, body: e.target.value })} /></Field>
            <button className="px-5 py-2.5 rounded-xl bg-emerald-600 text-white text-sm font-medium">Create</button>
          </form>
        </Card>
      )}
      {items.length === 0 && <Card className="p-8"><Empty message="No announcements yet." /></Card>}
      {items.map((a) => (
        <Card key={a.id} className="p-7">
          <div className="flex items-start justify-between gap-4">
            <div className="flex items-start gap-3">
              {a.pinned && <span className="mt-1 text-primary text-lg leading-none">📌</span>}
              <div>
                <h3 className="font-semibold text-ink text-[17px]">{a.title}</h3>
                <p className="mt-1.5 text-[15px] text-ink/80 whitespace-pre-wrap">{a.body}</p>
                <p className="mt-2 text-xs text-muted">{timeAgo(a.created_at)}{a.priority !== 'NORMAL' ? ` · ${a.priority}` : ''}</p>
              </div>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Badge tone={statusTone(a.status)}>{a.status}</Badge>
              {elevated && a.status !== 'PUBLISHED' && (
                <button onClick={() => api.post(`/v1/batches/${batchID}/announcements/${a.id}/publish`).then(() => { toast.success('Published'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-xs text-emerald-600 font-medium hover:underline">Publish</button>
              )}
              {elevated && (<button onClick={() => api.delete(`/v1/batches/${batchID}/announcements/${a.id}`).then(() => { toast.success('Archived'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-xs text-red-600 hover:underline">Archive</button>)}
            </div>
          </div>
        </Card>
      ))}
    </div>
  );
}

function ComplaintsTab({ batchID }: { batchID: string }) {
  const [items, setItems] = useState<Complaint[]>([]);
  const [form, setForm] = useState({ category: '', subject: '', message: '', is_anonymous: false });
  const [show, setShow] = useState(false);
  const [thread, setThread] = useState<{ id: string; messages: { message: string; created_at: string }[] } | null>(null);
  const [reply, setReply] = useState('');

  const load = () => api.get(`/v1/batches/${batchID}/complaints/mine`, { limit: 100 }).then((res) => setItems(toList<Complaint>(res.data))).catch(() => {});
  useEffect(() => {
    load();
  }, [batchID]);

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.post(`/v1/batches/${batchID}/complaints`, form);
      toast.success('Complaint submitted');
      setShow(false);
      setForm({ category: '', subject: '', message: '', is_anonymous: false });
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Submit failed'));
    }
  };

  const openThread = async (id: string) => {
    try {
      const res = await api.get(`/v1/batches/${batchID}/complaints/${id}/messages`);
      setThread({ id, messages: toList(res.data) });
    } catch (err) {
      toast.error(errMsg(err, 'Could not load conversation'));
    }
  };

  const sendReply = async () => {
    if (!thread || !reply.trim()) return;
    try {
      await api.post(`/v1/batches/${batchID}/complaints/${thread.id}/messages`, { message: reply });
      setReply('');
      openThread(thread.id);
    } catch (err) {
      toast.error(errMsg(err, 'Reply failed'));
    }
  };

  return (
    <div className="space-y-5">
      <div className="flex justify-end">
        <button onClick={() => setShow(!show)} className="px-5 py-2.5 rounded-xl bg-primary text-white text-[15px] font-medium hover:bg-primary-dark">
          + New complaint
        </button>
      </div>
      {show && (
        <Card className="p-7">
          <form onSubmit={create} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <Field label="Category"><input required placeholder="e.g. facilities" className={inputCls} value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })} /></Field>
              <Field label="Subject"><input required className={inputCls} value={form.subject} onChange={(e) => setForm({ ...form, subject: e.target.value })} /></Field>
            </div>
            <Field label="Message"><textarea required rows={4} className={inputCls} value={form.message} onChange={(e) => setForm({ ...form, message: e.target.value })} /></Field>
            <label className="flex items-center gap-2 text-sm text-ink">
              <input type="checkbox" checked={form.is_anonymous} onChange={(e) => setForm({ ...form, is_anonymous: e.target.checked })} className="w-4 h-4 accent-primary" />
              Submit anonymously
            </label>
            <button className="px-5 py-2.5 rounded-xl bg-emerald-600 text-white text-sm font-medium">Submit</button>
          </form>
        </Card>
      )}
      {items.length === 0 && <Card className="p-8"><Empty message="No complaints submitted." /></Card>}
      {items.map((c) => (
        <Card key={c.id} className="p-6">
          <div className="flex items-start justify-between gap-4">
            <div>
              <p className="font-medium text-ink">{c.subject} <span className="text-xs text-muted font-normal">· {c.category}</span></p>
              <p className="text-sm text-muted mt-1 line-clamp-2">{c.message}</p>
              <p className="text-xs text-muted mt-1">{timeAgo(c.created_at)}{c.is_anonymous ? ' · anonymous' : ''}</p>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Badge tone={statusTone(c.status)}>{c.status}</Badge>
              <button onClick={() => openThread(c.id)} className="text-xs text-primary font-medium hover:underline">Conversation</button>
            </div>
          </div>
          {thread?.id === c.id && (
            <div className="mt-4 rounded-xl bg-surface p-4 space-y-2">
              {thread.messages.map((m, i) => (
                <div key={i} className="text-sm"><span className="text-ink">{m.message}</span> <span className="text-xs text-muted">· {timeAgo(m.created_at)}</span></div>
              ))}
              {thread.messages.length === 0 && <p className="text-sm text-muted">No messages yet.</p>}
              <div className="flex gap-2 pt-1">
                <input className={inputCls} placeholder="Write a reply…" value={reply} onChange={(e) => setReply(e.target.value)} />
                <button onClick={sendReply} className="px-4 rounded-xl bg-primary text-white text-sm font-medium whitespace-nowrap">Send</button>
              </div>
            </div>
          )}
        </Card>
      ))}
    </div>
  );
}

function FeedbackTab({ batchID }: { batchID: string }) {
  const [items, setItems] = useState<FeedbackItem[]>([]);
  const [form, setForm] = useState({ category: '', message: '', rating: '', is_anonymous: false });
  const elevated = useElevated();
  const [show, setShow] = useState(false);

  const load = () => api.get(`/v1/batches/${batchID}/feedback`, { limit: 100 }).then((res) => setItems(toList<FeedbackItem>(res.data))).catch(() => {});
  useEffect(() => {
    load();
  }, [batchID]);

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    const body: Record<string, unknown> = { category: form.category, message: form.message, is_anonymous: form.is_anonymous };
    if (form.rating) body.rating = Number(form.rating);
    try {
      await api.post(`/v1/batches/${batchID}/feedback`, body);
      toast.success('Feedback submitted');
      setShow(false);
      setForm({ category: '', message: '', rating: '', is_anonymous: false });
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Submit failed'));
    }
  };

  const setStatus = (id: string, status: string) =>
    api.patch(`/v1/batches/${batchID}/feedback/${id}/status`, { status }).then(() => { toast.success(`Marked ${status}`); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')));

  return (
    <div className="space-y-5">
      <div className="flex justify-end">
        <button onClick={() => setShow(!show)} className="px-5 py-2.5 rounded-xl bg-primary text-white text-[15px] font-medium hover:bg-primary-dark">
          + New feedback
        </button>
      </div>
      {show && (
        <Card className="p-7">
          <form onSubmit={create} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <Field label="Category"><input placeholder="e.g. teaching" className={inputCls} value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })} /></Field>
              <Field label="Rating (1–5, optional)">
                <select className={inputCls} value={form.rating} onChange={(e) => setForm({ ...form, rating: e.target.value })}>
                  <option value="">None</option>{[1, 2, 3, 4, 5].map((n) => <option key={n} value={n}>{n}</option>)}
                </select>
              </Field>
            </div>
            <Field label="Message"><textarea required rows={4} className={inputCls} value={form.message} onChange={(e) => setForm({ ...form, message: e.target.value })} /></Field>
            <label className="flex items-center gap-2 text-sm text-ink">
              <input type="checkbox" checked={form.is_anonymous} onChange={(e) => setForm({ ...form, is_anonymous: e.target.checked })} className="w-4 h-4 accent-primary" />
              Submit anonymously
            </label>
            <button className="px-5 py-2.5 rounded-xl bg-emerald-600 text-white text-sm font-medium">Submit</button>
          </form>
        </Card>
      )}
      {items.length === 0 && <Card className="p-8"><Empty message="No feedback yet." /></Card>}
      {items.map((f) => (
        <Card key={f.id} className="p-6">
          <div className="flex items-start justify-between gap-4">
            <div>
              <p className="text-[15px] text-ink">{f.message}</p>
              <p className="text-xs text-muted mt-1">{f.category && `${f.category} · `}{f.rating ? `★ ${f.rating} · ` : ''}{timeAgo(f.created_at)}</p>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Badge tone={statusTone(f.status)}>{f.status}</Badge>
              {elevated && f.status === 'NEW' && <button onClick={() => setStatus(f.id, 'REVIEWED')} className="text-xs text-primary font-medium hover:underline">Review</button>}
              {elevated && f.status === 'REVIEWED' && <button onClick={() => setStatus(f.id, 'ARCHIVED')} className="text-xs text-muted hover:underline">Archive</button>}
            </div>
          </div>
        </Card>
      ))}
    </div>
  );
}
