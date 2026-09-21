import { Fragment, useEffect, useState } from 'react';
import { api, toList, errMsg, uploadFile } from '../api/client';
import { useAppStore } from '../store/app';
import toast from 'react-hot-toast';
import type { Module, Resource, ResourceType } from '../types';
import { useCan } from '../hooks/useRole';
import { Card, Badge, statusTone, Empty, Field, inputCls, fmtDate } from '../components/ui';

const TYPES: { code: ResourceType; label: string }[] = [
  { code: 'LECTURE_NOTE', label: 'Lecture note' },
  { code: 'HANDWRITTEN_NOTE', label: 'Handwritten note' },
  { code: 'PAST_PAPER', label: 'Past paper' },
  { code: 'TUTORIAL', label: 'Tutorial' },
  { code: 'ASSIGNMENT', label: 'Assignment' },
  { code: 'OTHER', label: 'Other' },
];

function fmtSize(bytes: number): string {
  if (bytes >= 1 << 20) return `${(bytes / (1 << 20)).toFixed(1)} MB`;
  return `${Math.max(1, Math.round(bytes / 1024))} KB`;
}

export default function Assessments() {
  const { currentBatchID } = useAppStore();
  const [items, setItems] = useState<Resource[]>([]);
  const [modules, setModules] = useState<Module[]>([]);
  const [versions, setVersions] = useState<Record<string, unknown[]>>({});
  const [saved, setSaved] = useState<Set<string>>(new Set());
  const [show, setShow] = useState(false);
  const elevated = useCan('resource.create', currentBatchID);
  const canUpdate = useCan('resource.update', currentBatchID);
  const [busy, setBusy] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [form, setForm] = useState({ module_id: '', type: 'LECTURE_NOTE' as ResourceType, title: '', description: '', academic_year: '', exam_type: '' });

  const load = () => {
    if (!currentBatchID) return;
    api.get(`/v1/batches/${currentBatchID}/resources`, { limit: 100 }).then((res) => setItems(toList<Resource>(res.data))).catch(() => {});
    api.get(`/v1/batches/${currentBatchID}/modules`, { limit: 100 }).then((res) => setModules(toList<Module>(res.data))).catch(() => {});
    api.get('/v1/me/bookmarks', { limit: 100 }).then((res) => setSaved(new Set(toList<{ id: string }>(res.data).map((b) => b.id)))).catch(() => {});
  };
  useEffect(load, [currentBatchID]);

  const modName = (id: string) => modules.find((m) => m.id === id)?.module_code || '—';

  const upload = async (e: React.FormEvent) => {
    e.preventDefault();
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
      toast.success('Assessment material uploaded');
      setShow(false);
      setFile(null);
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Upload failed'));
    } finally {
      setBusy(false);
    }
  };

  const download = async (r: Resource) => {
    try {
      const res = await api.get(`/v1/batches/${currentBatchID}/resources/${r.id}/download`);
      window.open(res.data.url, '_blank');
    } catch (err) {
      toast.error(errMsg(err, 'Download failed'));
    }
  };

  const toggleBookmark = async (r: Resource) => {
    try {
      if (saved.has(r.id)) {
        await api.delete(`/v1/batches/${currentBatchID}/resources/${r.id}/bookmark`);
      } else {
        await api.put(`/v1/batches/${currentBatchID}/resources/${r.id}/bookmark`);
      }
      const next = new Set(saved);
      if (next.has(r.id)) next.delete(r.id); else next.add(r.id);
      setSaved(next);
      toast.success(saved.has(r.id) ? 'Bookmark removed' : 'Bookmarked');
    } catch (err) {
      toast.error(errMsg(err, 'Bookmark failed'));
    }
  };

  const toggleVersions = async (r: Resource) => {
    if (versions[r.id]) {
      const next = { ...versions };
      delete next[r.id];
      setVersions(next);
      return;
    }
    try {
      const res = await api.get(`/v1/batches/${currentBatchID}/resources/${r.id}/versions`);
      setVersions({ ...versions, [r.id]: toList(res.data) });
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
      window.open(response.data.url, '_blank');
    } catch (err) {
      toast.error(errMsg(err, 'Download failed'));
    }
  };

  const editResource = async (item: Resource) => {
    const title = window.prompt('Material title', item.title);
    if (title === null || !title.trim()) return;
    const description = window.prompt('Description', item.description || '');
    if (description === null) return;
    try {
      await api.patch(`/v1/batches/${currentBatchID}/resources/${item.id}`, { title, description });
      toast.success('Material updated');
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Update failed'));
    }
  };

  if (!currentBatchID) {
    return (
      <Card className="p-8">
        <Empty message="Select a cohort from the dropdown above to view its assessments." />
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <p className="text-muted text-[15px]">{items.length} file{items.length === 1 ? '' : 's'} in this cohort</p>
        {elevated && (<button onClick={() => setShow(!show)} className="px-5 py-2.5 rounded-xl bg-primary text-white text-[15px] font-medium hover:bg-primary-dark">
          + Upload material
        </button>)}
      </div>

      {show && elevated && (
        <Card className="p-7">
          <h2 className="text-lg font-semibold text-ink">Upload PDF material</h2>
          <form onSubmit={upload} className="mt-4 grid grid-cols-2 gap-4">
            <Field label="Module">
              <select required className={inputCls} value={form.module_id} onChange={(e) => setForm({ ...form, module_id: e.target.value })}>
                <option value="">Select…</option>
                {modules.map((m) => <option key={m.id} value={m.id}>{m.module_code} — {m.name}</option>)}
              </select>
            </Field>
            <Field label="Type">
              <select className={inputCls} value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value as ResourceType })}>
                {TYPES.map((t) => <option key={t.code} value={t.code}>{t.label}</option>)}
              </select>
            </Field>
            <div className="col-span-2">
              <Field label="Title">
                <input required className={inputCls} value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} />
              </Field>
            </div>
            <div className="col-span-2">
              <Field label="Description">
                <textarea rows={2} className={inputCls} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
              </Field>
            </div>
            <Field label="Academic year (optional)"><input className={inputCls} value={form.academic_year} onChange={(e) => setForm({ ...form, academic_year: e.target.value })} /></Field>
            <Field label="Exam type (optional)"><input className={inputCls} value={form.exam_type} onChange={(e) => setForm({ ...form, exam_type: e.target.value })} /></Field>
            <div className="col-span-2">
              <Field label="PDF file (max 50 MB)">
                <input required type="file" accept="application/pdf,.pdf" onChange={(e) => setFile(e.target.files?.[0] || null)} className="text-sm text-ink" />
              </Field>
            </div>
            <div className="col-span-2">
              <button disabled={busy} className="px-5 py-2.5 rounded-xl bg-emerald-600 text-white text-sm font-medium disabled:opacity-50">
                {busy ? 'Uploading…' : 'Upload'}
              </button>
            </div>
          </form>
        </Card>
      )}

      <Card className="p-7">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-[15px]">
            <thead>
              <tr className="text-muted text-sm border-b border-line">
                <th className="py-3 pr-4 font-medium">Title</th>
                <th className="py-3 pr-4 font-medium">Type</th>
                <th className="py-3 pr-4 font-medium">Module</th>
                <th className="py-3 pr-4 font-medium">Version</th>
                <th className="py-3 pr-4 font-medium">Size</th>
                <th className="py-3 pr-4 font-medium">Status</th>
                <th className="py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {items.map((r) => (
                <Fragment key={r.id}>
                  <tr>
                    <td className="py-3.5 pr-4">
                      <p className="font-medium text-ink">{r.title}</p>
                      <p className="text-xs text-muted font-mono">{r.file_name} · {fmtDate(r.created_at)}</p>
                    </td>
                    <td className="py-3.5 pr-4"><Badge tone="purple">{r.type.replaceAll('_', ' ')}</Badge></td>
                    <td className="py-3.5 pr-4 text-muted text-sm font-mono">{modName(r.module_id)}</td>
                    <td className="py-3.5 pr-4 text-muted text-sm">v{r.version_number}</td>
                    <td className="py-3.5 pr-4 text-muted text-sm">{fmtSize(r.size_bytes)}</td>
                    <td className="py-3.5 pr-4"><Badge tone={statusTone(r.status)}>{r.status}</Badge></td>
                    <td className="py-3.5">
                      <div className="flex flex-wrap gap-2 text-xs font-medium">
                        <button onClick={() => download(r)} className="text-primary hover:underline">Download</button>
                        <button onClick={() => toggleBookmark(r)} className="text-muted hover:text-ink hover:underline">{saved.has(r.id) ? 'Saved ✓' : 'Bookmark'}</button>
                        <button onClick={() => toggleVersions(r)} className="text-muted hover:text-ink hover:underline">Versions</button>
                        {canUpdate && (
                          <label className="cursor-pointer text-primary hover:underline">
                            New version
                            <input type="file" accept="application/pdf,.pdf" disabled={busy} className="hidden" onChange={(event) => addVersion(r.id, event.target.files?.[0])} />
                          </label>
                        )}
                        {canUpdate && <button onClick={() => editResource(r)} className="text-primary hover:underline">Edit</button>}
                        {elevated && r.status !== 'PUBLISHED' && (
                          <button onClick={() => api.post(`/v1/batches/${currentBatchID}/resources/${r.id}/publish`).then(() => { toast.success('Published'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-emerald-600 hover:underline">Publish</button>
                        )}
                        {elevated && (<button onClick={() => api.delete(`/v1/batches/${currentBatchID}/resources/${r.id}`).then(() => { toast.success('Archived'); load(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-red-600 hover:underline">Archive</button>)}
                      </div>
                    </td>
                  </tr>
                  {versions[r.id] && (
                    <tr>
                      <td colSpan={7} className="pb-4">
                        <div className="rounded-xl bg-surface p-4 text-sm">
                          {versions[r.id].map((v: unknown) => {
                            const ver = v as Record<string, unknown>;
                            return (
                              <div key={String(ver.id)} className="flex justify-between py-1.5 border-b border-line last:border-0">
                                <span className="text-ink">v{String(ver.version_number)} — {String(ver.file_name)}</span>
                                <span className="flex items-center gap-3 text-muted">
                                  {fmtSize(Number(ver.size_bytes))}{ver.change_note ? ` · ${String(ver.change_note)}` : ''}
                                  <button onClick={() => downloadVersion(r.id, String(ver.id))} className="text-primary hover:underline">Download</button>
                                </span>
                              </div>
                            );
                          })}
                        </div>
                      </td>
                    </tr>
                  )}
                </Fragment>
              ))}
            </tbody>
          </table>
          {items.length === 0 && <Empty message="No assessment materials yet." />}
        </div>
      </Card>
    </div>
  );
}
