import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  ArrowDownTrayIcon,
  BookmarkIcon,
  CloudArrowUpIcon,
  DocumentTextIcon,
  MagnifyingGlassIcon,
  PencilSquareIcon,
  TrashIcon,
} from '@heroicons/react/24/outline';
import { api, toList, errMsg, uploadFile } from '../api/client';
import { useAppStore } from '../store/app';
import toast from 'react-hot-toast';
import type { Module, Resource, ResourceType } from '../types';
import { useAccess, useCan } from '../hooks/useRole';
import { Badge, Card, CardTitle, Empty, Field, inputCls, statusTone, fmtDate } from '../components/ui';

const TYPES: { code: ResourceType; label: string; plural: string }[] = [
  { code: 'LECTURE_NOTE', label: 'Lecture note', plural: 'Lecture Notes' },
  { code: 'PAST_PAPER', label: 'Past paper', plural: 'Past Papers' },
  { code: 'ASSIGNMENT', label: 'Assignment', plural: 'Assignments' },
  { code: 'REFERENCE', label: 'Reference', plural: 'References' },
];

function typeLabel(type: string): string {
  return TYPES.find((item) => item.code === type)?.label || type.replaceAll('_', ' ').toLowerCase();
}

function fmtSize(bytes: number): string {
  if (bytes >= 1 << 20) return `${(bytes / (1 << 20)).toFixed(1)} MB`;
  return `${Math.max(1, Math.round(bytes / 1024))} KB`;
}

export default function Assessments() {
  const { currentBatchID, currentBatch } = useAppStore();
  const access = useAccess();
  const platformAdmin = access?.platform_roles.includes('PLATFORM_ADMIN') ?? false;
  const createPermission = useCan('resource.create', currentBatchID);
  const updatePermission = useCan('resource.update', currentBatchID);
  const canCreate = platformAdmin || createPermission === true;
  const canUpdate = platformAdmin || updatePermission === true;
  const [items, setItems] = useState<Resource[]>([]);
  const [modules, setModules] = useState<Module[]>([]);
  const [versions, setVersions] = useState<Record<string, unknown[]>>({});
  const [saved, setSaved] = useState<Set<string>>(new Set());
  const [showUpload, setShowUpload] = useState(false);
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [moduleFilter, setModuleFilter] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [query, setQuery] = useState('');
  const [form, setForm] = useState({ module_id: '', type: 'LECTURE_NOTE' as ResourceType, title: '', description: '', academic_year: '', exam_type: '' });

  const load = useCallback(async () => {
    if (!currentBatchID) return;
    setLoading(true);
    const params: Record<string, string | number> = { limit: 100 };
    if (moduleFilter) params.module_id = moduleFilter;
    if (typeFilter) params.type = typeFilter;
    if (query.trim()) params.query = query.trim();
    try {
      const [resources, moduleResponse, bookmarks] = await Promise.all([
        api.get(`/v1/batches/${currentBatchID}/resources`, params),
        api.get(`/v1/batches/${currentBatchID}/modules`, { limit: 100 }),
        api.get('/v1/me/bookmarks', { limit: 100 }),
      ]);
      setItems(toList<Resource>(resources.data));
      setModules(toList<Module>(moduleResponse.data));
      setSaved(new Set(toList<{ id: string }>(bookmarks.data).map((bookmark) => bookmark.id)));
    } catch (err) {
      toast.error(errMsg(err, 'Could not load resources'));
    } finally {
      setLoading(false);
    }
  }, [currentBatchID, moduleFilter, query, typeFilter]);

  useEffect(() => {
    const timer = window.setTimeout(load, query ? 250 : 0);
    return () => window.clearTimeout(timer);
  }, [load, query]);

  const grouped = useMemo(() => {
    const groups = new Map<string, { code: string; name: string; items: Resource[] }>();
    for (const item of items) {
      const group = groups.get(item.module_id) || { code: item.module_code || '—', name: item.module_name || 'Course resources', items: [] };
      group.items.push(item);
      groups.set(item.module_id, group);
    }
    return [...groups.entries()];
  }, [items]);

  const upload = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!file || !currentBatchID) return;
    if (file.type !== 'application/pdf' || !file.name.toLowerCase().endsWith('.pdf')) {
      toast.error('Only PDF files are accepted');
      return;
    }
    setBusy(true);
    try {
      const init = await api.post(`/v1/batches/${currentBatchID}/resources/uploads`, {
        file_name: file.name,
        mime_type: 'application/pdf',
        size_bytes: file.size,
      });
      await uploadFile(init.data.upload_url, file);
      const body: Record<string, unknown> = {
        module_id: form.module_id,
        upload_id: init.data.upload_id,
        type: form.type,
        title: form.title,
        description: form.description,
      };
      if (form.academic_year) body.academic_year = form.academic_year;
      if (form.exam_type) body.exam_type = form.exam_type;
      await api.post(`/v1/batches/${currentBatchID}/resources`, body);
      toast.success('Resource uploaded as a draft');
      setShowUpload(false);
      setFile(null);
      setForm({ module_id: '', type: 'LECTURE_NOTE', title: '', description: '', academic_year: '', exam_type: '' });
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Upload failed'));
    } finally {
      setBusy(false);
    }
  };

  const download = async (item: Resource) => {
    try {
      const response = await api.get(`/v1/batches/${currentBatchID}/resources/${item.id}/download`);
      window.open(response.data.url, '_blank', 'noopener,noreferrer');
    } catch (err) {
      toast.error(errMsg(err, 'Download failed'));
    }
  };

  const toggleBookmark = async (item: Resource) => {
    const wasSaved = saved.has(item.id);
    try {
      if (wasSaved) await api.delete(`/v1/batches/${currentBatchID}/resources/${item.id}/bookmark`);
      else await api.put(`/v1/batches/${currentBatchID}/resources/${item.id}/bookmark`);
      setSaved((current) => {
        const next = new Set(current);
        if (wasSaved) next.delete(item.id); else next.add(item.id);
        return next;
      });
      toast.success(wasSaved ? 'Bookmark removed' : 'Resource bookmarked');
    } catch (err) {
      toast.error(errMsg(err, 'Bookmark failed'));
    }
  };

  const toggleVersions = async (item: Resource) => {
    if (versions[item.id]) {
      setVersions((current) => {
        const next = { ...current };
        delete next[item.id];
        return next;
      });
      return;
    }
    try {
      const response = await api.get(`/v1/batches/${currentBatchID}/resources/${item.id}/versions`);
      setVersions((current) => ({ ...current, [item.id]: toList(response.data) }));
    } catch (err) {
      toast.error(errMsg(err, 'Could not load versions'));
    }
  };

  const addVersion = async (resourceID: string, nextFile?: File) => {
    if (!nextFile || nextFile.type !== 'application/pdf') return;
    setBusy(true);
    try {
      const init = await api.post(`/v1/batches/${currentBatchID}/resources/${resourceID}/versions/uploads`, {
        file_name: nextFile.name,
        mime_type: 'application/pdf',
        size_bytes: nextFile.size,
      });
      await uploadFile(init.data.upload_url, nextFile);
      await api.post(`/v1/batches/${currentBatchID}/resources/${resourceID}/versions`, {
        upload_id: init.data.upload_id,
        original_size_bytes: nextFile.size,
        change_note: 'Uploaded from the web portal',
      });
      toast.success('New version uploaded');
      setVersions((current) => {
        const next = { ...current };
        delete next[resourceID];
        return next;
      });
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Version upload failed'));
    } finally {
      setBusy(false);
    }
  };

  const downloadVersion = async (resourceID: string, versionID: string) => {
    try {
      const response = await api.get(`/v1/batches/${currentBatchID}/resources/${resourceID}/versions/${versionID}/download`);
      window.open(response.data.url, '_blank', 'noopener,noreferrer');
    } catch (err) {
      toast.error(errMsg(err, 'Download failed'));
    }
  };

  const editResource = async (item: Resource) => {
    const title = window.prompt('Resource title', item.title);
    if (title === null || !title.trim()) return;
    const description = window.prompt('Description', item.description || '');
    if (description === null) return;
    try {
      await api.patch(`/v1/batches/${currentBatchID}/resources/${item.id}`, { title, description });
      toast.success('Resource updated');
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Update failed'));
    }
  };

  const publish = (item: Resource) => api.post(`/v1/batches/${currentBatchID}/resources/${item.id}/publish`)
    .then(() => { toast.success('Resource published'); load(); })
    .catch((err) => toast.error(errMsg(err, 'Publish failed')));

  const archive = (item: Resource) => api.delete(`/v1/batches/${currentBatchID}/resources/${item.id}`)
    .then(() => { toast.success('Resource archived'); load(); })
    .catch((err) => toast.error(errMsg(err, 'Archive failed')));

  if (!currentBatchID) return <Card className="p-8"><Empty message="Select a cohort to open its resource library." /></Card>;

  return (
    <div className="space-y-7">
      <div className="flex flex-col gap-5 xl:flex-row xl:items-end xl:justify-between">
        <div className="flex items-center gap-4">
          <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-primary-light text-primary"><DocumentTextIcon className="h-7 w-7" /></div>
          <div>
            <h1 className="text-2xl font-semibold text-ink">Resource Library</h1>
            <p className="mt-1 text-[15px] text-muted">Academic materials for {currentBatch()?.name || 'your selected cohort'}.</p>
          </div>
        </div>
        {canCreate && <button onClick={() => setShowUpload((open) => !open)} className="inline-flex items-center justify-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-[15px] font-medium text-white hover:bg-primary-dark"><CloudArrowUpIcon className="h-5 w-5" /> Upload Resource</button>}
      </div>

      <Card className="p-4">
        <div className="grid gap-3 lg:grid-cols-[minmax(180px,0.8fr)_minmax(180px,0.8fr)_minmax(240px,1.4fr)]">
          <select className={inputCls} value={moduleFilter} onChange={(event) => setModuleFilter(event.target.value)}><option value="">All subjects</option>{modules.map((module) => <option key={module.id} value={module.id}>{module.module_code} — {module.name}</option>)}</select>
          <select className={inputCls} value={typeFilter} onChange={(event) => setTypeFilter(event.target.value)}><option value="">All types</option>{TYPES.map((type) => <option key={type.code} value={type.code}>{type.plural}</option>)}</select>
          <div className="relative"><MagnifyingGlassIcon className="pointer-events-none absolute left-3.5 top-1/2 h-5 w-5 -translate-y-1/2 text-muted" /><input className={`${inputCls} pl-11`} placeholder="Search title, description, or subject…" value={query} onChange={(event) => setQuery(event.target.value)} /></div>
        </div>
        <div className="mt-4 flex flex-wrap gap-2 border-t border-line pt-4">
          <button onClick={() => setTypeFilter('')} className={`rounded-lg px-3.5 py-2 text-sm font-medium ${!typeFilter ? 'bg-primary text-white' : 'bg-surface text-muted hover:text-ink'}`}>All Resources</button>
          {TYPES.map((type) => <button key={type.code} onClick={() => setTypeFilter(type.code)} className={`rounded-lg px-3.5 py-2 text-sm font-medium ${typeFilter === type.code ? 'bg-primary text-white' : 'bg-surface text-muted hover:text-ink'}`}>{type.plural}</button>)}
        </div>
      </Card>

      {showUpload && canCreate && (
        <Card className="p-7">
          <CardTitle>Upload PDF resource</CardTitle>
          <p className="mt-1 text-sm text-muted">Resources remain drafts until you publish them.</p>
          <form onSubmit={upload} className="mt-5 grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Subject"><select required className={inputCls} value={form.module_id} onChange={(event) => setForm({ ...form, module_id: event.target.value })}><option value="">Select subject…</option>{modules.map((module) => <option key={module.id} value={module.id}>{module.module_code} — {module.name}</option>)}</select></Field>
            <Field label="Resource type"><select className={inputCls} value={form.type} onChange={(event) => setForm({ ...form, type: event.target.value as ResourceType })}>{TYPES.map((type) => <option key={type.code} value={type.code}>{type.label}</option>)}</select></Field>
            <div className="sm:col-span-2"><Field label="Title"><input required className={inputCls} value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} /></Field></div>
            <div className="sm:col-span-2"><Field label="Description"><textarea rows={3} className={inputCls} value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} /></Field></div>
            <Field label="Academic year (optional)"><input className={inputCls} value={form.academic_year} onChange={(event) => setForm({ ...form, academic_year: event.target.value })} /></Field>
            <Field label="Exam type (optional)"><input className={inputCls} value={form.exam_type} onChange={(event) => setForm({ ...form, exam_type: event.target.value })} /></Field>
            <div className="sm:col-span-2"><Field label="PDF file (maximum 50 MB)"><input required type="file" accept="application/pdf,.pdf" onChange={(event) => setFile(event.target.files?.[0] || null)} className="block w-full rounded-xl border border-line bg-white px-3.5 py-2.5 text-sm text-ink" /></Field></div>
            <div className="sm:col-span-2 flex justify-end gap-3"><button type="button" onClick={() => setShowUpload(false)} className="rounded-xl border border-line bg-white px-5 py-2.5 text-sm font-medium text-ink hover:bg-surface">Cancel</button><button disabled={busy} className="rounded-xl bg-primary px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-dark disabled:opacity-50">{busy ? 'Uploading…' : 'Upload resource'}</button></div>
          </form>
        </Card>
      )}

      {loading ? <Card className="p-10 text-center text-sm text-muted">Loading resources…</Card> : grouped.length === 0 ? <Card className="p-10"><Empty message="No resources match the selected filters." /></Card> : grouped.map(([moduleID, group]) => (
        <section key={moduleID} className="space-y-4">
          <div className="flex items-center gap-3 border-b border-line pb-3"><span className="h-8 w-1 rounded-full bg-primary" /><div><h2 className="text-lg font-semibold text-ink">{group.code}</h2><p className="text-sm text-muted">{group.name}</p></div><Badge tone="purple">{group.items.length} {group.items.length === 1 ? 'item' : 'items'}</Badge></div>
          <div className="grid gap-5 md:grid-cols-2 2xl:grid-cols-3">
            {group.items.map((item) => (
              <Card key={item.id} className="overflow-hidden">
                <div className="flex h-32 items-center justify-center border-b border-line bg-surface"><div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-white text-primary shadow-sm"><DocumentTextIcon className="h-9 w-9" /></div></div>
                <div className="p-5">
                  <div className="flex items-start justify-between gap-3"><div className="min-w-0"><h3 className="truncate font-semibold text-ink">{item.title}</h3><p className="mt-1 line-clamp-2 min-h-10 text-sm leading-5 text-muted">{item.description || 'No description provided.'}</p></div>{canUpdate && <Badge tone={statusTone(item.status)}>{item.status}</Badge>}</div>
                  <div className="mt-4 flex flex-wrap items-center gap-2 text-xs text-muted"><span className="rounded-md bg-primary-light px-2 py-1 font-medium text-primary">{typeLabel(item.type)}</span><span>{fmtSize(item.size_bytes)}</span><span>v{item.version_number}</span><span>{fmtDate(item.created_at)}</span></div>
                  <div className="mt-5 flex items-center gap-1 border-t border-line pt-4">
                    <button onClick={() => download(item)} className="rounded-lg p-2 text-primary hover:bg-primary-light" title="Download"><ArrowDownTrayIcon className="h-5 w-5" /></button>
                    <button onClick={() => toggleBookmark(item)} className={`rounded-lg p-2 hover:bg-primary-light ${saved.has(item.id) ? 'text-primary' : 'text-muted hover:text-primary'}`} title={saved.has(item.id) ? 'Remove bookmark' : 'Bookmark'}><BookmarkIcon className="h-5 w-5" /></button>
                    <button onClick={() => toggleVersions(item)} className="rounded-lg px-2 py-1.5 text-xs font-medium text-muted hover:bg-surface hover:text-ink">Versions</button><div className="flex-1" />
                    {canUpdate && <label className="cursor-pointer rounded-lg p-2 text-muted hover:bg-primary-light hover:text-primary" title="Upload new version"><CloudArrowUpIcon className="h-5 w-5" /><input type="file" accept="application/pdf,.pdf" disabled={busy} className="hidden" onChange={(event) => addVersion(item.id, event.target.files?.[0])} /></label>}
                    {canUpdate && <button onClick={() => editResource(item)} className="rounded-lg p-2 text-muted hover:bg-primary-light hover:text-primary" title="Edit"><PencilSquareIcon className="h-5 w-5" /></button>}
                    {canCreate && item.status !== 'PUBLISHED' && <button onClick={() => publish(item)} className="rounded-lg px-2 py-1.5 text-xs font-medium text-emerald-600 hover:bg-emerald-50">Publish</button>}
                    {canCreate && <button onClick={() => archive(item)} className="rounded-lg p-2 text-muted hover:bg-red-50 hover:text-red-600" title="Archive"><TrashIcon className="h-5 w-5" /></button>}
                  </div>
                  {versions[item.id] && <div className="mt-4 space-y-2 rounded-xl bg-surface p-3 text-xs">{versions[item.id].map((value) => { const version = value as Record<string, unknown>; return <div key={String(version.id)} className="flex items-center justify-between gap-3 border-b border-line pb-2 last:border-0 last:pb-0"><span className="truncate text-ink">v{String(version.version_number)} · {String(version.file_name)}</span><button onClick={() => downloadVersion(item.id, String(version.id))} className="font-medium text-primary hover:underline">Download</button></div>; })}</div>}
                </div>
              </Card>
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}
