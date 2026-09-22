import { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { api, toList, errMsg, uploadFile } from '../api/client';
import toast from 'react-hot-toast';
import type { Batch, BatchProfile, Semester, Module, Lesson, LinkItem, Member, Position } from '../types';
import { useCan } from '../hooks/useRole';
import { Card, Badge, statusTone, Empty, Field, inputCls, fmtDate, coverColor, initials } from '../components/ui';

const TABS = ['Overview', 'Semesters', 'Modules', 'Lessons', 'Links', 'Committee'] as const;

async function promptEdit(url: string, fields: { key: string; label: string; value: string }[], reload: () => void) {
  const body: Record<string, string> = {};
  for (const field of fields) {
    const value = window.prompt(field.label, field.value);
    if (value === null) return;
    body[field.key] = value;
  }
  try {
    await api.patch(url, body);
    toast.success('Updated');
    reload();
  } catch (err) {
    toast.error(errMsg(err, 'Update failed'));
  }
}

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
      {tab === 'Committee' && <CommitteeTab batchID={batchID} />}
    </div>
  );
}

function Overview({ batchID, batch, onUpdate }: { batchID: string; batch: Batch; onUpdate: (b: Batch) => void }) {
  const [profile, setProfile] = useState<BatchProfile>({ headline: '', about_text: '', mission_text: '', contact_email: '' });
  const canManageBatch = useCan('batch.manage', batchID);
  const canManageProfile = useCan('batch.profile.manage', batchID);
  const elevated = canManageBatch || canManageProfile;
  const [edit, setEdit] = useState({ name: batch.name, description: batch.description });
  const [heroBusy, setHeroBusy] = useState(false);
  const [heroAvailable, setHeroAvailable] = useState(true);
  const [heroVersion, setHeroVersion] = useState(0);

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

  const setHeroImage = async (file?: File) => {
    if (!file) return;
    setHeroBusy(true);
    try {
      const init = await api.post(`/v1/batches/${batchID}/profile/uploads`, { file_name: file.name, mime_type: file.type, size_bytes: file.size });
      await uploadFile(init.data.upload_url, file);
      await api.patch(`/v1/batches/${batchID}/profile`, { upload_id: init.data.upload_id });
      setHeroAvailable(true);
      setHeroVersion((version) => version + 1);
      toast.success('Public cover image updated');
    } catch (err) {
      toast.error(errMsg(err, 'Cover image upload failed'));
    } finally {
      setHeroBusy(false);
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
      {canManageBatch && <Card className="p-7 space-y-4">
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
      </Card>}
      {canManageProfile && <Card className="p-7 space-y-4">
        <h2 className="text-lg font-semibold text-ink">Public profile</h2>
        {heroAvailable && <img src={`/v1/public/batches/${batch.slug}/image?v=${heroVersion}`} alt="Public course cover" className="w-full h-32 rounded-xl object-cover bg-surface" onError={() => setHeroAvailable(false)} />}
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
        <label className="inline-flex cursor-pointer text-sm text-primary font-medium hover:underline">
          {heroBusy ? 'Uploading cover…' : 'Change public cover image'}
          <input type="file" accept="image/jpeg,image/png,image/webp" disabled={heroBusy} className="hidden" onChange={(event) => setHeroImage(event.target.files?.[0])} />
        </label>
      </Card>}
    </div>
  );
}

function Semesters({ batchID }: { batchID: string }) {
  const [items, setItems] = useState<Semester[]>([]);
  const elevated = useCan('semester.manage', batchID);
  const [form, setForm] = useState({ semester_number: 1, name: '', academic_year: '', starts_at: '', ends_at: '' });
  const [show, setShow] = useState(false);

  const load = useCallback(() => api.get(`/v1/batches/${batchID}/semesters`).then((res) => setItems(toList<Semester>(res.data))).catch(() => {}), [batchID]);
  useEffect(() => {
    load();
  }, [load]);

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
              {elevated && <button onClick={() => promptEdit(`/v1/batches/${batchID}/semesters/${s.id}`, [{ key: 'name', label: 'Semester name', value: s.name }, { key: 'academic_year', label: 'Academic year', value: s.academic_year }], load)} className="text-xs text-primary hover:underline">Edit</button>}
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
}

function ModulesTab({ batchID }: { batchID: string }) {
  const [items, setItems] = useState<Module[]>([]);
  const elevated = useCan('module.manage', batchID);
  const [semesters, setSemesters] = useState<Semester[]>([]);
  const [form, setForm] = useState({ semester_id: '', module_code: '', name: '', description: '', lecturer_name: '' });
  const [show, setShow] = useState(false);

  const load = useCallback(() => {
    api.get(`/v1/batches/${batchID}/modules`, { limit: 100 }).then((res) => setItems(toList<Module>(res.data))).catch(() => {});
    api.get(`/v1/batches/${batchID}/semesters`, { limit: 100 }).then((res) => setSemesters(toList<Semester>(res.data))).catch(() => {});
  }, [batchID]);
  useEffect(() => {
    load();
  }, [load]);

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
              {elevated && <button onClick={() => promptEdit(`/v1/batches/${batchID}/modules/${m.id}`, [{ key: 'name', label: 'Module name', value: m.name }, { key: 'description', label: 'Description', value: m.description || '' }, { key: 'lecturer_name', label: 'Lecturer', value: m.lecturer_name || '' }], load)} className="text-xs text-primary hover:underline">Edit</button>}
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
  const elevated = useCan('lesson.manage', batchID);
  const [modules, setModules] = useState<Module[]>([]);
  const [form, setForm] = useState({ module_id: '', title: '', description: '', youtube_video_id: '', lesson_date: '', duration_seconds: '' });
  const [show, setShow] = useState(false);

  const load = useCallback(() => {
    api.get(`/v1/batches/${batchID}/lessons`, { limit: 100 }).then((res) => setItems(toList<Lesson>(res.data))).catch(() => {});
    api.get(`/v1/batches/${batchID}/modules`, { limit: 100 }).then((res) => setModules(toList<Module>(res.data))).catch(() => {});
  }, [batchID]);
  useEffect(() => {
    load();
  }, [load]);

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
              {elevated && <button onClick={() => promptEdit(`/v1/batches/${batchID}/lessons/${l.id}`, [{ key: 'title', label: 'Lesson title', value: l.title }, { key: 'description', label: 'Description', value: l.description || '' }, { key: 'youtube_video_id', label: 'YouTube video ID', value: l.youtube_video_id }], load)} className="text-xs text-primary hover:underline">Edit</button>}
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
  const elevated = useCan('link.manage', batchID);
  const [modules, setModules] = useState<Module[]>([]);
  const [form, setForm] = useState({ module_id: '', title: '', url: '', description: '', category: '' });
  const [show, setShow] = useState(false);

  const load = useCallback(() => {
    api.get(`/v1/batches/${batchID}/links`, { limit: 100 }).then((res) => setItems(toList<LinkItem>(res.data))).catch(() => {});
    api.get(`/v1/batches/${batchID}/modules`, { limit: 100 }).then((res) => setModules(toList<Module>(res.data))).catch(() => {});
  }, [batchID]);
  useEffect(() => {
    load();
  }, [load]);

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
              {elevated && <button onClick={() => promptEdit(`/v1/batches/${batchID}/links/${l.id}`, [{ key: 'title', label: 'Link title', value: l.title }, { key: 'url', label: 'URL', value: l.url }, { key: 'description', label: 'Description', value: l.description || '' }], load)} className="text-xs text-primary hover:underline">Edit</button>}
              {elevated && l.status !== 'PUBLISHED' && <button onClick={() => api.post(`/v1/batches/${batchID}/links/${l.id}/publish`).then(() => { toast.success('Published'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-xs text-primary font-medium hover:underline">Publish</button>}
              {elevated && (<button onClick={() => api.delete(`/v1/batches/${batchID}/links/${l.id}`).then(() => { toast.success('Archived'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-xs text-red-600 hover:underline">Archive</button>)}
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
}

interface PositionAssignment {
  id: string;
  user_id: string;
  display_name: string;
  starts_at: string;
  ends_at: string | null;
}

function CommitteeTab({ batchID }: { batchID: string }) {
  const canManage = useCan('position.manage', batchID);
  const [positions, setPositions] = useState<Position[]>([]);
  const [members, setMembers] = useState<Member[]>([]);
  const [assignments, setAssignments] = useState<Record<string, PositionAssignment[]>>({});
  const [show, setShow] = useState(false);
  const [form, setForm] = useState({ title: '', description: '', sort_order: 0, is_public: true });
  const [selectedMembers, setSelectedMembers] = useState<Record<string, string>>({});

  const load = useCallback(() => api.get(`/v1/batches/${batchID}/positions`, { limit: 100 }).then((response) => setPositions(toList<Position>(response.data))).catch(() => {}), [batchID]);
  useEffect(() => {
    load();
    if (canManage) api.get(`/v1/batches/${batchID}/members`, { limit: 100 }).then((response) => setMembers(toList<Member>(response.data).filter((member) => member.status === 'ACTIVE'))).catch(() => {});
  }, [batchID, canManage, load]);

  const create = async (event: React.FormEvent) => {
    event.preventDefault();
    try {
      await api.post(`/v1/batches/${batchID}/positions`, form);
      setForm({ title: '', description: '', sort_order: 0, is_public: true });
      setShow(false);
      toast.success('Committee position created');
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Create failed'));
    }
  };

  const toggleAssignments = async (positionID: string) => {
    if (assignments[positionID]) {
      setAssignments((current) => {
        const next = { ...current };
        delete next[positionID];
        return next;
      });
      return;
    }
    try {
      const response = await api.get(`/v1/batches/${batchID}/positions/${positionID}/assignments`, { limit: 100 });
      setAssignments((current) => ({ ...current, [positionID]: toList<PositionAssignment>(response.data) }));
    } catch (err) {
      toast.error(errMsg(err, 'Could not load assignments'));
    }
  };

  const assign = async (positionID: string) => {
    const target = selectedMembers[positionID];
    if (!target) return;
    try {
      await api.post(`/v1/batches/${batchID}/positions/${positionID}/assignments`, { user_id: target, starts_at: new Date().toISOString() });
      const response = await api.get(`/v1/batches/${batchID}/positions/${positionID}/assignments`, { limit: 100 });
      setAssignments((current) => ({ ...current, [positionID]: toList<PositionAssignment>(response.data) }));
      toast.success('Committee member assigned');
    } catch (err) {
      toast.error(errMsg(err, 'Assignment failed'));
    }
  };

  const endAssignment = async (positionID: string, assignmentID: string) => {
    try {
      await api.post(`/v1/batches/${batchID}/positions/${positionID}/assignments/${assignmentID}/end`);
      const response = await api.get(`/v1/batches/${batchID}/positions/${positionID}/assignments`, { limit: 100 });
      setAssignments((current) => ({ ...current, [positionID]: toList<PositionAssignment>(response.data) }));
      toast.success('Assignment ended');
    } catch (err) {
      toast.error(errMsg(err, 'Could not end assignment'));
    }
  };

  return (
    <div className="space-y-5">
      {canManage && <div className="flex justify-end"><button onClick={() => setShow(!show)} className="px-4 py-2 rounded-xl bg-primary text-white text-sm font-medium">+ Position</button></div>}
      {show && canManage && (
        <Card className="p-6">
          <form onSubmit={create} className="grid grid-cols-2 gap-4">
            <Field label="Title"><input required className={inputCls} value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} /></Field>
            <Field label="Sort order"><input type="number" className={inputCls} value={form.sort_order} onChange={(event) => setForm({ ...form, sort_order: Number(event.target.value) })} /></Field>
            <div className="col-span-2"><Field label="Description"><textarea className={inputCls} value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} /></Field></div>
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={form.is_public} onChange={(event) => setForm({ ...form, is_public: event.target.checked })} /> Show publicly</label>
            <div className="col-span-2"><button className="px-5 py-2.5 rounded-xl bg-emerald-600 text-white text-sm font-medium">Create</button></div>
          </form>
        </Card>
      )}
      {positions.length === 0 && <Card className="p-8"><Empty message="No committee positions yet." /></Card>}
      {positions.map((position) => (
        <Card key={position.id} className="p-6">
          <div className="flex items-start justify-between gap-4">
            <div><h3 className="font-semibold text-ink">{position.title}</h3><p className="text-sm text-muted">{position.description || 'No description.'}</p></div>
            <div className="flex gap-2"><Badge tone={position.is_public ? 'green' : 'gray'}>{position.is_public ? 'PUBLIC' : 'PRIVATE'}</Badge><button onClick={() => toggleAssignments(position.id)} className="text-xs text-primary">Assignments</button>{canManage && <button onClick={() => promptEdit(`/v1/batches/${batchID}/positions/${position.id}`, [{ key: 'title', label: 'Position title', value: position.title }, { key: 'description', label: 'Description', value: position.description || '' }], load)} className="text-xs text-primary">Edit</button>}{canManage && position.is_public && <button onClick={() => api.delete(`/v1/batches/${batchID}/positions/${position.id}`).then(() => { toast.success('Position hidden'); load(); }).catch((err) => toast.error(errMsg(err, 'Update failed')))} className="text-xs text-red-600">Hide</button>}</div>
          </div>
          {assignments[position.id] && (
            <div className="mt-4 p-4 rounded-xl bg-surface space-y-2">
              {(assignments[position.id] ?? []).map((assignment) => (
                <div key={assignment.id} className="flex justify-between text-sm"><span>{assignment.display_name}{assignment.ends_at ? ' · ended' : ''}</span>{canManage && !assignment.ends_at && <button onClick={() => endAssignment(position.id, assignment.id)} className="text-red-600 text-xs">End</button>}</div>
              ))}
              {(assignments[position.id] ?? []).length === 0 && <p className="text-sm text-muted">No assignments.</p>}
              {canManage && <div className="flex gap-2 pt-2"><select className={inputCls} value={selectedMembers[position.id] || ''} onChange={(event) => setSelectedMembers({ ...selectedMembers, [position.id]: event.target.value })}><option value="">Select member…</option>{members.map((member) => <option key={member.user_id} value={member.user_id}>{member.display_name}</option>)}</select><button onClick={() => assign(position.id)} className="px-4 rounded-xl bg-primary text-white text-sm">Assign</button></div>}
            </div>
          )}
        </Card>
      ))}
    </div>
  );
}
