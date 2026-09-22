import { useCallback, useEffect, useState } from 'react';
import { api, toList, errMsg, uploadFile } from '../api/client';
import { useAppStore } from '../store/app';
import toast from 'react-hot-toast';
import type { Announcement, Complaint, FeedbackItem, Member } from '../types';
import { useCan } from '../hooks/useRole';
import { usePatchEditor } from '../hooks/usePatchEditor';
import { Card, Badge, statusTone, Empty, Field, inputCls, timeAgo, Tabs } from '../components/ui';

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
      <Card className="px-4 pt-2">
        <Tabs tabs={TABS} active={tab} onChange={setTab} />
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
  const elevated = useCan('announcement.create', batchID);
  const [show, setShow] = useState(false);
  const [attachments, setAttachments] = useState<Record<string, { id: string; file_name: string; size_bytes: number }[]>>({});

  const load = useCallback(() => api.get(`/v1/batches/${batchID}/announcements`, { limit: 100 }).then((res) => setItems(toList<Announcement>(res.data))).catch(() => {}), [batchID]);
  const editor = usePatchEditor(load, 'Announcement updated');
  useEffect(() => {
    load();
  }, [load]);

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

  const toggleAttachments = async (announcementID: string) => {
    if (attachments[announcementID]) {
      setAttachments((current) => {
        const next = { ...current };
        delete next[announcementID];
        return next;
      });
      return;
    }
    try {
      const response = await api.get(`/v1/batches/${batchID}/announcements/${announcementID}/attachments`);
      setAttachments((current) => ({ ...current, [announcementID]: toList(response.data) }));
    } catch (err) {
      toast.error(errMsg(err, 'Could not load attachments'));
    }
  };

  const addAttachment = async (announcementID: string, file?: File) => {
    if (!file) return;
    try {
      const init = await api.post(`/v1/batches/${batchID}/announcements/${announcementID}/attachments/uploads`, {
        file_name: file.name,
        mime_type: file.type || 'application/octet-stream',
        size_bytes: file.size,
      });
      await uploadFile(init.data.upload_url, file);
      await api.post(`/v1/batches/${batchID}/announcements/${announcementID}/attachments`, { upload_id: init.data.upload_id });
      const response = await api.get(`/v1/batches/${batchID}/announcements/${announcementID}/attachments`);
      setAttachments((current) => ({ ...current, [announcementID]: toList(response.data) }));
      toast.success('Attachment added');
    } catch (err) {
      toast.error(errMsg(err, 'Attachment upload failed'));
    }
  };

  const downloadAttachment = async (announcementID: string, attachmentID: string) => {
    try {
      const response = await api.get(`/v1/batches/${batchID}/announcements/${announcementID}/attachments/${attachmentID}/download`);
      window.open(response.data.url, '_blank');
    } catch (err) {
      toast.error(errMsg(err, 'Download failed'));
    }
  };

  const editAnnouncement = (item: Announcement) => {
    editor.open('Edit announcement', `/v1/batches/${batchID}/announcements/${item.id}`, [
      { key: 'title', label: 'Announcement title', value: item.title, required: true },
      { key: 'body', label: 'Announcement body', value: item.body, multiline: true, required: true },
    ]);
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
              {elevated && <button onClick={() => editAnnouncement(a)} className="text-xs text-primary hover:underline">Edit</button>}
              {elevated && (<button onClick={() => api.delete(`/v1/batches/${batchID}/announcements/${a.id}`).then(() => { toast.success('Archived'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-xs text-red-600 hover:underline">Archive</button>)}
              <button onClick={() => toggleAttachments(a.id)} className="text-xs text-primary hover:underline">Attachments</button>
              {elevated && (
                <label className="text-xs text-primary cursor-pointer hover:underline">
                  Add file
                  <input type="file" accept="application/pdf,.pdf" className="hidden" onChange={(event) => addAttachment(a.id, event.target.files?.[0])} />
                </label>
              )}
            </div>
          </div>
          {attachments[a.id] && (
            <div className="mt-4 rounded-xl bg-surface p-4 space-y-2">
              {(attachments[a.id] ?? []).length === 0 && <p className="text-sm text-muted">No attachments.</p>}
              {(attachments[a.id] ?? []).map((attachment) => (
                <div key={attachment.id} className="flex items-center justify-between gap-3 text-sm">
                  <button onClick={() => downloadAttachment(a.id, attachment.id)} className="text-primary hover:underline">{attachment.file_name}</button>
                  {elevated && <button onClick={() => api.delete(`/v1/batches/${batchID}/announcements/${a.id}/attachments/${attachment.id}`).then(async () => {
                    const response = await api.get(`/v1/batches/${batchID}/announcements/${a.id}/attachments`);
                    setAttachments((current) => ({ ...current, [a.id]: toList(response.data) }));
                    toast.success('Attachment removed');
                  }).catch((err) => toast.error(errMsg(err, 'Remove failed')))} className="text-red-600 text-xs">Remove</button>}
                </div>
              ))}
            </div>
          )}
        </Card>
      ))}
      {editor.dialog}
    </div>
  );
}

function ComplaintsTab({ batchID }: { batchID: string }) {
  const [items, setItems] = useState<Complaint[]>([]);
  const [form, setForm] = useState({ category: '', subject: '', message: '', is_anonymous: false });
  const [show, setShow] = useState(false);
  const [thread, setThread] = useState<{ id: string; messages: { message: string; created_at: string }[] } | null>(null);
  const [reply, setReply] = useState('');
  const canManage = useCan('complaint.view_all', batchID);
  const canResolve = useCan('complaint.resolve', batchID);
  const [managers, setManagers] = useState<Member[]>([]);

  const load = useCallback(() => api.get(`/v1/batches/${batchID}/complaints${canManage ? '' : '/mine'}`, { limit: 100 }).then((res) => setItems(toList<Complaint>(res.data))).catch(() => {}), [batchID, canManage]);
  useEffect(() => {
    load();
    if (canManage) {
      api.get(`/v1/batches/${batchID}/members`, { limit: 100 }).then((response) => {
        setManagers(toList<Member>(response.data).filter((member) => member.roles.includes('COMPLAINT_MANAGER')));
      }).catch(() => {});
    }
  }, [batchID, canManage, load]);

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.post(`/v1/batches/${batchID}/complaints`, form);
      toast.success(form.is_anonymous ? 'Anonymous complaint submitted. It will not appear in your personal history.' : 'Complaint submitted');
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

  const setComplaintStatus = async (id: string, status: string) => {
    try {
      await api.patch(`/v1/batches/${batchID}/complaints/${id}/status`, { status });
      toast.success(`Complaint marked ${status.replaceAll('_', ' ').toLowerCase()}`);
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Status change failed'));
    }
  };

  const assignComplaint = async (id: string, userID: string) => {
    if (!userID) return;
    try {
      await api.patch(`/v1/batches/${batchID}/complaints/${id}/assignee`, { user_id: userID });
      toast.success('Complaint assigned');
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Assignment failed'));
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
          {canManage && (
            <div className="mt-3 flex flex-wrap items-center gap-2 text-xs">
              <select aria-label={`Assign manager for complaint ${c.subject}`} value={c.assigned_to || ''} onChange={(event) => assignComplaint(c.id, event.target.value)} className="px-2 py-1.5 rounded-lg border border-line bg-charcoal-card text-ink text-xs">
                <option value="">Assign manager…</option>
                {managers.map((manager) => <option key={manager.user_id} value={manager.user_id}>{manager.display_name}</option>)}
              </select>
              {c.status === 'OPEN' && <button onClick={() => setComplaintStatus(c.id, 'IN_REVIEW')} className="text-primary font-medium">Start review</button>}
              {c.status === 'IN_REVIEW' && canResolve && <button onClick={() => setComplaintStatus(c.id, 'RESOLVED')} className="text-emerald-700 font-medium">Resolve</button>}
              {c.status === 'RESOLVED' && <button onClick={() => setComplaintStatus(c.id, 'IN_REVIEW')} className="text-primary font-medium">Reopen</button>}
              {c.status === 'RESOLVED' && canResolve && <button onClick={() => setComplaintStatus(c.id, 'CLOSED')} className="text-muted font-medium">Close</button>}
            </div>
          )}
          {thread?.id === c.id && (
            <div className="mt-4 rounded-xl bg-surface p-4 space-y-2">
              {thread.messages.map((m, i) => (
                <div key={i} className="text-sm"><span className="text-ink">{m.message}</span> <span className="text-xs text-muted">· {timeAgo(m.created_at)}</span></div>
              ))}
              {thread.messages.length === 0 && <p className="text-sm text-muted">No messages yet.</p>}
              <div className="flex gap-2 pt-1">
                <input aria-label="Write a reply" className={inputCls} placeholder="Write a reply…" value={reply} onChange={(e) => setReply(e.target.value)} />
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
  const elevated = useCan('feedback.view', batchID);
  const [show, setShow] = useState(false);

  const load = useCallback(() => {
    if (!elevated) return;
    api.get(`/v1/batches/${batchID}/feedback`, { limit: 100 }).then((res) => setItems(toList<FeedbackItem>(res.data))).catch(() => {});
  }, [batchID, elevated]);
  useEffect(() => {
    load();
  }, [load]);

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
      {!elevated && <Card className="p-6"><p className="text-sm text-muted">Feedback submissions are private. Representatives can review them without exposing anonymous authors.</p></Card>}
      {elevated && items.length === 0 && <Card className="p-8"><Empty message="No feedback yet." /></Card>}
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
