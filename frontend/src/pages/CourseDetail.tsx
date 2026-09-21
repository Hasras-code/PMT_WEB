import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { api, toList, errMsg } from '../api/client';
import toast from 'react-hot-toast';
import type { Batch, BatchProfile, Semester, Module, Lesson, LinkItem } from '../types';
import { useElevated } from '../hooks/useRole';
import { Card, Badge, statusTone, Empty, Field, inputCls, fmtDate, coverColor, initials } from '../components/ui';

const TABS = ['Overview', 'Semesters', 'Modules', 'Lessons', 'Links'] as const;

export default function CourseDetail() {
  const { batchID = '' } = useParams();
  const [batch, setBatch] = useState<Batch | null>(null);
  const [tab, setTab] = useState<(typeof TABS)[number]>('Overview');

  useEffect(() => {
    api.get(`/v1/batches/${batchID}`).then((res) => setBatch(res.data)).catch(() => toast.error('Course not found'));
  }, [batchID]);

  if (!batch) return <p className="text-muted text-sm">Loading…</p>;

  return (
    <div className="space-y-6">
      <Card className="overflow-hidden">
        <div className="h-32 flex items-center px-8 gap-5" style={{ background: coverColor(batch.slug) }}>
          <span className="w-16 h-16 rounded-2xl bg-white/20 text-white text-2xl font-bold flex items-center justify-center">
            {initials(batch.name)}
          </span>
          <div>
            <h1 className="text-2xl font-bold text-white">{batch.name}</h1>
            <p className="text-white/80 text-sm">
              {batch.slug} · Class of {batch.entry_year} · {batch.status}
            </p>
          </div>
        </div>
        <div className="flex gap-1 px-4 pt-3 border-b border-line overflow-x-auto">
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
        </div>
      </Card>

      {tab === 'Overview' && <Overview batchID={batchID} batch={batch} onUpdate={setBatch} />}
      {tab === 'Semesters' && <Semesters batchID={batchID} />}
      {tab === 'Modules' && <ModulesTab batchID={batchID} />}
      {tab === 'Lessons' && <LessonsTab batchID={batchID} />}
      {tab === 'Links' && <LinksTab batchID={batchID} />}
    </div>
  );
}

function Overview({ batchID, batch, onUpdate }: { batchID: string; batch: Batch; onUpdate: (b: Batch) => void }) {
  const [profile, setProfile] = useState<BatchProfile>({ headline: '', about_text: '', mission_text: '', contact_email: '' });
  const elevated = useElevated();
  const [edit, setEdit] = useState({ name: batch.name, description: batch.description });

  useEffect(() => {
    api.get(`/v1/batches/${batchID}/profile`).then((res) => setProfile(res.data)).catch(() => {});
  }, [batchID]);

  const saveBatch = async () => {
    try {
      await api.patch(`/v1/batches/${batchID}`, { name: edit.name, description: edit.description });
      onUpdate({ ...batch, name: edit.name, description: edit.description });
      toast.success('Course updated');
    } catch (err) {
      toast.error(errMsg(err, 'Update failed'));
    }
  };

  const saveProfile = async () => {
    try {
      await api.patch(`/v1/batches/${batchID}/profile`, {
        headline: profile.headline || null,
        about_text: profile.about_text || null,
        mission_text: profile.mission_text || null,
        contact_email: profile.contact_email || null,
      });
      toast.success('Public profile updated');
    } catch (err) {
      toast.error(errMsg(err, 'Profile update failed'));
    }
  };

  const set = (k: keyof BatchProfile) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
    setProfile({ ...profile, [k]: e.target.value });

  if (!elevated) {
    return (
      <Card className="p-7 space-y-3">
        <h2 className="text-lg font-semibold text-ink">{batch.name}</h2>
        <p className="text-[15px] text-muted">{batch.description || 'No description.'}</p>
        {profile.headline && <p className="text-[15px] text-ink font-medium">{profile.headline}</p>}
        {profile.about_text && <p className="text-sm text-muted whitespace-pre-wrap">{profile.about_text}</p>}
        {profile.mission_text && <p className="text-sm text-muted">Mission: {profile.mission_text}</p>}
        {profile.contact_email && <p className="text-sm text-muted">Contact: {profile.contact_email}</p>}
      </Card>
    );
  }

  return (
    <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
      <Card className="p-7 space-y-4">
        <h2 className="text-lg font-semibold text-ink">Course details</h2>
        <Field label="Name">
          <input className={inputCls} value={edit.name} onChange={(e) => setEdit({ ...edit, name: e.target.value })} />
        </Field>
        <Field label="Description">
          <textarea rows={4} className={inputCls} value={edit.description} onChange={(e) => setEdit({ ...edit, description: e.target.value })} />
        </Field>
        <button onClick={saveBatch} className="px-5 py-2.5 rounded-xl bg-primary text-white text-[15px] font-medium hover:bg-primary-dark">
          Save changes
        </button>
      </Card>
      <Card className="p-7 space-y-4">
        <h2 className="text-lg font-semibold text-ink">Public profile</h2>
        <Field label="Headline">
          <input className={inputCls} value={profile.headline || ''} onChange={set('headline')} />
        </Field>
        <Field label="About">
          <textarea rows={3} className={inputCls} value={profile.about_text || ''} onChange={set('about_text')} />
        </Field>
        <Field label="Mission">
          <textarea rows={2} className={inputCls} value={profile.mission_text || ''} onChange={set('mission_text')} />
        </Field>
        <Field label="Contact email">
          <input className={inputCls} value={profile.contact_email || ''} onChange={set('contact_email')} />
        </Field>
        <button onClick={saveProfile} className="px-5 py-2.5 rounded-xl bg-primary text-white text-[15px] font-medium hover:bg-primary-dark">
          Save profile
        </button>
      </Card>
    </div>
  );
}

function Semesters({ batchID }: { batchID: string }) {
  const [items, setItems] = useState<Semester[]>([]);
  const elevated = useElevated();
  const [form, setForm] = useState({ semester_number: 1, name: '', academic_year: '', starts_at: '', ends_at: '' });
  const [show, setShow] = useState(false);

  const load = () => api.get(`/v1/batches/${batchID}/semesters`).then((res) => setItems(toList<Semester>(res.data))).catch(() => {});
  useEffect(() => {
    load();
  }, [batchID]);

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    const body: Record<string, unknown> = { semester_number: Number(form.semester_number), name: form.name, academic_year: form.academic_year };
    if (form.starts_at) body.starts_at = new Date(form.starts_at).toISOString();
    if (form.ends_at) body.ends_at = new Date(form.ends_at).toISOString();
    try {
      await api.post(`/v1/batches/${batchID}/semesters`, body);
      toast.success('Semester created');
      setShow(false);
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Create failed'));
    }
  };

  return (
    <Card className="p-7">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-ink">Semesters</h2>
        {elevated && (<button onClick={() => setShow(!show)} className="px-4 py-2 rounded-xl bg-primary text-white text-sm font-medium">+ New</button>)}
      </div>
      {show && (
        <form onSubmit={create} className="mt-4 grid grid-cols-2 gap-4 p-4 rounded-xl bg-surface">
          <Field label="Number"><input required type="number" min={1} className={inputCls} value={form.semester_number} onChange={(e) => setForm({ ...form, semester_number: Number(e.target.value) })} /></Field>
          <Field label="Name"><input required className={inputCls} value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></Field>
          <Field label="Academic year"><input required placeholder="2026/2027" className={inputCls} value={form.academic_year} onChange={(e) => setForm({ ...form, academic_year: e.target.value })} /></Field>
          <div className="grid grid-cols-2 gap-4">
            <Field label="Starts"><input type="datetime-local" className={inputCls} value={form.starts_at} onChange={(e) => setForm({ ...form, starts_at: e.target.value })} /></Field>
            <Field label="Ends"><input type="datetime-local" className={inputCls} value={form.ends_at} onChange={(e) => setForm({ ...form, ends_at: e.target.value })} /></Field>
          </div>
          <div className="col-span-2"><button className="px-5 py-2.5 rounded-xl bg-emerald-600 text-white text-sm font-medium">Create</button></div>
        </form>
      )}
      <div className="mt-4 divide-y divide-line">
        {items.length === 0 && <Empty message="No semesters yet." />}
        {items.map((s) => (
          <div key={s.id} className="py-3.5 flex items-center justify-between">
            <div>
              <p className="font-medium text-ink">Semester {s.semester_number} — {s.name}</p>
              <p className="text-sm text-muted">{s.academic_year} · {fmtDate(s.starts_at)} → {fmtDate(s.ends_at)}</p>
            </div>
            <div className="flex items-center gap-2">
              {s.is_current ? <Badge tone="green">CURRENT</Badge> : elevated ? (
                <button onClick={() => api.post(`/v1/batches/${batchID}/semesters/${s.id}/set-current`).then(() => { toast.success('Current semester set'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-xs text-primary font-medium hover:underline">
                  Set current
                </button>
              ) : null}
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
}

function ModulesTab({ batchID }: { batchID: string }) {
  const [items, setItems] = useState<Module[]>([]);
  const elevated = useElevated();
  const [semesters, setSemesters] = useState<Semester[]>([]);
  const [form, setForm] = useState({ semester_id: '', module_code: '', name: '', description: '', lecturer_name: '' });
  const [show, setShow] = useState(false);

  const load = () => {
    api.get(`/v1/batches/${batchID}/modules`, { limit: 100 }).then((res) => setItems(toList<Module>(res.data))).catch(() => {});
    api.get(`/v1/batches/${batchID}/semesters`, { limit: 100 }).then((res) => setSemesters(toList<Semester>(res.data))).catch(() => {});
  };
  useEffect(() => {
    load();
  }, [batchID]);

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.post(`/v1/batches/${batchID}/modules`, {
        semester_id: form.semester_id,
        module_code: form.module_code,
        name: form.name,
        description: form.description,
        lecturer_name: form.lecturer_name || null,
      });
      toast.success('Module created');
      setShow(false);
      setForm({ semester_id: '', module_code: '', name: '', description: '', lecturer_name: '' });
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Create failed'));
    }
  };

  return (
    <Card className="p-7">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-ink">Modules</h2>
        {elevated && (<button onClick={() => setShow(!show)} className="px-4 py-2 rounded-xl bg-primary text-white text-sm font-medium">+ New</button>)}
      </div>
      {show && (
        <form onSubmit={create} className="mt-4 grid grid-cols-2 gap-4 p-4 rounded-xl bg-surface">
          <Field label="Semester">
            <select required className={inputCls} value={form.semester_id} onChange={(e) => setForm({ ...form, semester_id: e.target.value })}>
              <option value="">Select…</option>
              {semesters.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </select>
          </Field>
          <Field label="Module code"><input required placeholder="CS101" className={inputCls} value={form.module_code} onChange={(e) => setForm({ ...form, module_code: e.target.value })} /></Field>
          <Field label="Name"><input required className={inputCls} value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></Field>
          <Field label="Lecturer"><input className={inputCls} value={form.lecturer_name} onChange={(e) => setForm({ ...form, lecturer_name: e.target.value })} /></Field>
          <div className="col-span-2"><Field label="Description"><textarea rows={2} className={inputCls} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></Field></div>
          <div className="col-span-2"><button className="px-5 py-2.5 rounded-xl bg-emerald-600 text-white text-sm font-medium">Create</button></div>
        </form>
      )}
      <div className="mt-4 space-y-3">
        {items.length === 0 && <Empty message="No modules yet." />}
        {items.map((m) => (
          <div key={m.id} className="p-4 rounded-xl border border-line flex items-start justify-between gap-4">
            <div>
              <p className="font-medium text-ink">{m.name} <span className="ml-2 text-xs font-mono text-muted">{m.module_code}</span></p>
              <p className="text-sm text-muted mt-0.5">{m.description || 'No description.'}</p>
              {m.lecturer_name && <p className="text-xs text-muted mt-1">Lecturer: {m.lecturer_name}</p>}
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Badge tone={statusTone(m.status)}>{m.status}</Badge>
              {elevated && (<button onClick={() => api.delete(`/v1/batches/${batchID}/modules/${m.id}`).then(() => { toast.success('Archived'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-xs text-red-600 hover:underline">Archive</button>)}
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
}

function LessonsTab({ batchID }: { batchID: string }) {
  const [items, setItems] = useState<Lesson[]>([]);
  const elevated = useElevated();
  const [modules, setModules] = useState<Module[]>([]);
  const [form, setForm] = useState({ module_id: '', title: '', description: '', youtube_video_id: '', lesson_date: '', duration_seconds: '' });
  const [show, setShow] = useState(false);

  const load = () => {
    api.get(`/v1/batches/${batchID}/lessons`, { limit: 100 }).then((res) => setItems(toList<Lesson>(res.data))).catch(() => {});
    api.get(`/v1/batches/${batchID}/modules`, { limit: 100 }).then((res) => setModules(toList<Module>(res.data))).catch(() => {});
  };
  useEffect(() => {
    load();
  }, [batchID]);

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    const body: Record<string, unknown> = { module_id: form.module_id, title: form.title, description: form.description, youtube_video_id: form.youtube_video_id };
    if (form.lesson_date) body.lesson_date = form.lesson_date;
    if (form.duration_seconds) body.duration_seconds = Number(form.duration_seconds);
    try {
      await api.post(`/v1/batches/${batchID}/lessons`, body);
      toast.success('Lesson created');
      setShow(false);
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Create failed'));
    }
  };

  return (
    <Card className="p-7">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-ink">Recorded lessons</h2>
        {elevated && (<button onClick={() => setShow(!show)} className="px-4 py-2 rounded-xl bg-primary text-white text-sm font-medium">+ New</button>)}
      </div>
      {show && (
        <form onSubmit={create} className="mt-4 grid grid-cols-2 gap-4 p-4 rounded-xl bg-surface">
          <Field label="Module">
            <select required className={inputCls} value={form.module_id} onChange={(e) => setForm({ ...form, module_id: e.target.value })}>
              <option value="">Select…</option>
              {modules.map((m) => <option key={m.id} value={m.id}>{m.module_code} — {m.name}</option>)}
            </select>
          </Field>
          <Field label="Title"><input required className={inputCls} value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} /></Field>
          <Field label="YouTube video ID"><input required placeholder="dQw4w9WgXcQ" className={inputCls} value={form.youtube_video_id} onChange={(e) => setForm({ ...form, youtube_video_id: e.target.value })} /></Field>
          <div className="grid grid-cols-2 gap-4">
            <Field label="Date"><input type="date" className={inputCls} value={form.lesson_date} onChange={(e) => setForm({ ...form, lesson_date: e.target.value })} /></Field>
            <Field label="Duration (sec)"><input type="number" min={1} className={inputCls} value={form.duration_seconds} onChange={(e) => setForm({ ...form, duration_seconds: e.target.value })} /></Field>
          </div>
          <div className="col-span-2"><Field label="Description"><textarea rows={2} className={inputCls} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></Field></div>
          <div className="col-span-2"><button className="px-5 py-2.5 rounded-xl bg-emerald-600 text-white text-sm font-medium">Create</button></div>
        </form>
      )}
      <div className="mt-4 divide-y divide-line">
        {items.length === 0 && <Empty message="No lessons yet." />}
        {items.map((l) => (
          <div key={l.id} className="py-3.5 flex items-center justify-between gap-4">
            <div>
              <p className="font-medium text-ink">{l.title}</p>
              <p className="text-sm text-muted">
                {fmtDate(l.lesson_date)}{l.duration_seconds ? ` · ${Math.round(l.duration_seconds / 60)} min` : ''} ·{' '}
                <a className="text-primary hover:underline" href={`https://www.youtube.com/watch?v=${l.youtube_video_id}`} target="_blank" rel="noreferrer">Watch</a>
              </p>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Badge tone={statusTone(l.status)}>{l.status}</Badge>
              {elevated && l.status !== 'PUBLISHED' && <button onClick={() => api.post(`/v1/batches/${batchID}/lessons/${l.id}/publish`).then(() => { toast.success('Published'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-xs text-primary font-medium hover:underline">Publish</button>}
              {elevated && (<button onClick={() => api.delete(`/v1/batches/${batchID}/lessons/${l.id}`).then(() => { toast.success('Archived'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-xs text-red-600 hover:underline">Archive</button>)}
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
}

function LinksTab({ batchID }: { batchID: string }) {
  const [items, setItems] = useState<LinkItem[]>([]);
  const elevated = useElevated();
  const [modules, setModules] = useState<Module[]>([]);
  const [form, setForm] = useState({ module_id: '', title: '', url: '', description: '', category: '' });
  const [show, setShow] = useState(false);

  const load = () => {
    api.get(`/v1/batches/${batchID}/links`, { limit: 100 }).then((res) => setItems(toList<LinkItem>(res.data))).catch(() => {});
    api.get(`/v1/batches/${batchID}/modules`, { limit: 100 }).then((res) => setModules(toList<Module>(res.data))).catch(() => {});
  };
  useEffect(() => {
    load();
  }, [batchID]);

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    const body: Record<string, unknown> = { title: form.title, url: form.url };
    if (form.module_id) body.module_id = form.module_id;
    if (form.description) body.description = form.description;
    if (form.category) body.category = form.category;
    try {
      await api.post(`/v1/batches/${batchID}/links`, body);
      toast.success('Link created');
      setShow(false);
      setForm({ module_id: '', title: '', url: '', description: '', category: '' });
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Create failed'));
    }
  };

  return (
    <Card className="p-7">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-ink">Useful links</h2>
        {elevated && (<button onClick={() => setShow(!show)} className="px-4 py-2 rounded-xl bg-primary text-white text-sm font-medium">+ New</button>)}
      </div>
      {show && (
        <form onSubmit={create} className="mt-4 grid grid-cols-2 gap-4 p-4 rounded-xl bg-surface">
          <Field label="Title"><input required className={inputCls} value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} /></Field>
          <Field label="URL"><input required type="url" placeholder="https://…" className={inputCls} value={form.url} onChange={(e) => setForm({ ...form, url: e.target.value })} /></Field>
          <Field label="Module (optional)">
            <select className={inputCls} value={form.module_id} onChange={(e) => setForm({ ...form, module_id: e.target.value })}>
              <option value="">None</option>
              {modules.map((m) => <option key={m.id} value={m.id}>{m.module_code} — {m.name}</option>)}
            </select>
          </Field>
          <Field label="Category"><input className={inputCls} value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })} /></Field>
          <div className="col-span-2"><Field label="Description"><input className={inputCls} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></Field></div>
          <div className="col-span-2"><button className="px-5 py-2.5 rounded-xl bg-emerald-600 text-white text-sm font-medium">Create</button></div>
        </form>
      )}
      <div className="mt-4 divide-y divide-line">
        {items.length === 0 && <Empty message="No links yet." />}
        {items.map((l) => (
          <div key={l.id} className="py-3.5 flex items-center justify-between gap-4">
            <div>
              <a href={l.url} target="_blank" rel="noreferrer" className="font-medium text-primary hover:underline">{l.title}</a>
              <p className="text-sm text-muted">{l.category && `${l.category} · `}{l.description}</p>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Badge tone={statusTone(l.status)}>{l.status}</Badge>
              {elevated && l.status !== 'PUBLISHED' && <button onClick={() => api.post(`/v1/batches/${batchID}/links/${l.id}/publish`).then(() => { toast.success('Published'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-xs text-primary font-medium hover:underline">Publish</button>}
              {elevated && (<button onClick={() => api.delete(`/v1/batches/${batchID}/links/${l.id}`).then(() => { toast.success('Archived'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-xs text-red-600 hover:underline">Archive</button>)}
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
}
