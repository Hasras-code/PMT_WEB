import { useEffect, useState } from 'react';
import { api, toList, errMsg, uploadFile } from '../api/client';
import toast from 'react-hot-toast';
import type { AdminBatch, AdminUser, GalleryImage, PlatformRole } from '../types';
import { Card, Badge, statusTone, Empty, Field, inputCls, fmtDate } from '../components/ui';
import { useCan } from '../hooks/useRole';

export default function Administration() {
  const canUsers = useCan('platform_user.manage');
  const canGallery = useCan('gallery.manage');
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [platformRoles, setPlatformRoles] = useState<PlatformRole[]>([]);
  const [batches, setBatches] = useState<AdminBatch[]>([]);
  const [editingUser, setEditingUser] = useState<AdminUser | null>(null);
  const [selectedRole, setSelectedRole] = useState('');
  const [selectedBatch, setSelectedBatch] = useState('');
  const [gallery, setGallery] = useState<GalleryImage[]>([]);
  const [tab, setTab] = useState<'users' | 'gallery'>('users');
  const [showUpload, setShowUpload] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [thumb, setThumb] = useState<File | null>(null);
  const [gform, setGform] = useState({ title: '', caption: '', alt_text: '' });
  const [busy, setBusy] = useState(false);

  const loadUsers = () => api.get('/v1/admin/users', { limit: 100 }).then((res) => setUsers(toList<AdminUser>(res.data))).catch(() => {});
  const loadPlatformRoles = () => api.get('/v1/admin/roles').then((res) => setPlatformRoles(toList<PlatformRole>(res.data))).catch((e) => toast.error(errMsg(e, 'Could not load platform roles')));
  const loadBatches = () => api.get('/v1/admin/batches').then((res) => setBatches(toList<AdminBatch>(res.data))).catch((e) => toast.error(errMsg(e, 'Could not load cohorts')));
  const loadGallery = () => api.get('/v1/admin/gallery', { limit: 100 }).then((res) => setGallery(toList<GalleryImage>(res.data))).catch(() => {});
  useEffect(() => {
    if (canUsers) {
      loadUsers();
      loadPlatformRoles();
      loadBatches();
    }
    if (canGallery) loadGallery();
    if (!canUsers && canGallery) setTab('gallery');
  }, [canGallery, canUsers]);

  const userStatus = (id: string, action: 'suspend' | 'reactivate' | 'archive') =>
    api.post(`/v1/admin/users/${id}/${action}`).then(() => { toast.success(`User ${action}d`); loadUsers(); }).catch((e) => toast.error(errMsg(e, 'Action failed')));

  const openUserEditor = (user: AdminUser) => {
    setEditingUser(user);
    setSelectedRole('');
    setSelectedBatch('');
  };

  const changePlatformRole = async (role: string, remove = false, batchID = '') => {
    if (!editingUser) return;
    try {
      if (remove) {
        const query = batchID ? `?batch_id=${batchID}` : '';
        await api.delete(`/v1/admin/users/${editingUser.id}/roles/${role}${query}`);
      } else {
        await api.post(`/v1/admin/users/${editingUser.id}/roles`, { role, batch_id: batchID || undefined });
      }
      toast.success(remove ? 'Role removed' : 'Role assigned');
      const response = await api.get('/v1/admin/users', { limit: 100 });
      const updatedUsers = toList<AdminUser>(response.data);
      setUsers(updatedUsers);
      setEditingUser(updatedUsers.find((user) => user.id === editingUser.id) || null);
      setSelectedRole('');
      setSelectedBatch('');
    } catch (err) {
      toast.error(errMsg(err, 'Role change failed'));
    }
  };

  const authorizeImage = async (f: File) => {
    const init = await api.post('/v1/admin/gallery/uploads', { file_name: f.name, mime_type: f.type, size_bytes: f.size });
    await uploadFile(init.data.upload_url, f);
    return init.data.upload_id as string;
  };

  const uploadGallery = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!file || !thumb) return;
    setBusy(true);
    try {
      const [displayId, thumbId] = await Promise.all([authorizeImage(file), authorizeImage(thumb)]);
      await api.post('/v1/admin/gallery', { ...gform, display_upload_id: displayId, thumbnail_upload_id: thumbId });
      toast.success('Gallery image added as draft');
      setShowUpload(false);
      setFile(null);
      setThumb(null);
      loadGallery();
    } catch (err) {
      toast.error(errMsg(err, 'Gallery upload failed'));
    } finally {
      setBusy(false);
    }
  };

  const editGallery = async (image: GalleryImage) => {
    const title = window.prompt('Image title', image.title || '');
    if (title === null) return;
    const caption = window.prompt('Caption', image.caption || '');
    if (caption === null) return;
    const altText = window.prompt('Alternative text', image.alt_text);
    if (altText === null || !altText.trim()) return;
    try {
      await api.patch(`/v1/admin/gallery/${image.id}`, { title, caption, alt_text: altText });
      toast.success('Gallery image updated');
      loadGallery();
    } catch (err) {
      toast.error(errMsg(err, 'Update failed'));
    }
  };

  if (canUsers === false && canGallery === false) return <Card className="p-8"><Empty message="You do not have platform administration permission." /></Card>;

  return (
    <div className="space-y-6">
      <Card className="px-4 pt-2 flex gap-1">
        {(['users', 'gallery'] as const).filter((item) => item === 'users' ? canUsers : canGallery).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-4 py-3 text-[15px] capitalize border-b-2 -mb-px ${tab === t ? 'border-primary text-primary font-medium' : 'border-transparent text-muted hover:text-ink'}`}
          >
            {t === 'users' ? 'Platform users' : 'Gallery'}
          </button>
        ))}
      </Card>

      {tab === 'users' && (
        <Card className="p-7">
          <h2 className="text-lg font-semibold text-ink">Platform users {users.length > 0 && `(${users.length})`}</h2>
          {users.length === 0 ? (
            <Empty message="Requires the platform_user.manage permission." />
          ) : (
            <div className="mt-4 overflow-x-auto">
              <table className="w-full text-left text-[15px]">
                <thead>
                  <tr className="text-muted text-sm border-b border-line">
                    <th className="py-3 pr-4 font-medium">User</th>
                    <th className="py-3 pr-4 font-medium">Student #</th>
                    <th className="py-3 pr-4 font-medium">Combination</th>
                    <th className="py-3 pr-4 font-medium">Status</th>
                    <th className="py-3 pr-4 font-medium">Joined</th>
                    <th className="py-3 font-medium">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-line">
                  {users.map((u) => (
                    <tr key={u.id}>
                      <td className="py-3 pr-4"><p className="font-medium text-ink">{u.display_name}</p><p className="text-xs text-muted">{u.email}</p></td>
                      <td className="py-3 pr-4 text-muted text-sm font-mono">{u.student_number}</td>
                      <td className="py-3 pr-4 text-muted text-sm">{u.combination || '—'}</td>
                      <td className="py-3 pr-4"><Badge tone={statusTone(u.status)}>{u.status}</Badge></td>
                      <td className="py-3 pr-4 text-muted text-sm">{fmtDate(u.created_at)}</td>
                      <td className="py-3">
                        <div className="flex gap-2 text-xs font-medium">
                          <button onClick={() => openUserEditor(u)} className="text-primary hover:underline">Edit</button>
                          <button onClick={() => userStatus(u.id, 'suspend')} className="text-orange-600 hover:underline">Suspend</button>
                          <button onClick={() => userStatus(u.id, 'reactivate')} className="text-emerald-600 hover:underline">Reactivate</button>
                          <button onClick={() => userStatus(u.id, 'archive')} className="text-red-600 hover:underline">Archive</button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>
      )}

      {tab === 'gallery' && (
        <div className="space-y-6">
          <div className="flex justify-end">
            <button onClick={() => setShowUpload(!showUpload)} className="px-5 py-2.5 rounded-xl bg-primary text-white text-[15px] font-medium hover:bg-primary-dark">
              + Add image
            </button>
          </div>
          {showUpload && (
            <Card className="p-7">
              <form onSubmit={uploadGallery} className="grid grid-cols-2 gap-4">
                <Field label="Title"><input className={inputCls} value={gform.title} onChange={(e) => setGform({ ...gform, title: e.target.value })} /></Field>
                <Field label="Alt text (required)"><input required className={inputCls} value={gform.alt_text} onChange={(e) => setGform({ ...gform, alt_text: e.target.value })} /></Field>
                <div className="col-span-2"><Field label="Caption"><input className={inputCls} value={gform.caption} onChange={(e) => setGform({ ...gform, caption: e.target.value })} /></Field></div>
                <Field label="Display image (JPEG/PNG/WebP ≤10MB)"><input required type="file" accept="image/jpeg,image/png,image/webp" onChange={(e) => setFile(e.target.files?.[0] || null)} className="text-sm" /></Field>
                <Field label="Thumbnail (≤1MB)"><input required type="file" accept="image/jpeg,image/png,image/webp" onChange={(e) => setThumb(e.target.files?.[0] || null)} className="text-sm" /></Field>
                <div className="col-span-2"><button disabled={busy} className="px-5 py-2.5 rounded-xl bg-emerald-600 text-white text-sm font-medium disabled:opacity-50">{busy ? 'Uploading…' : 'Upload'}</button></div>
              </form>
            </Card>
          )}
          {gallery.length === 0 ? (
            <Card className="p-8"><Empty message="No gallery images. Requires the gallery.manage permission." /></Card>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
              {gallery.map((g) => (
                <Card key={g.id} className="overflow-hidden">
                  <img src={g.thumbnail_url || g.display_url} alt={g.alt_text} className="w-full h-44 object-cover" />
                  <div className="p-4">
                    <div className="flex items-center justify-between">
                      <p className="font-medium text-ink">{g.title || g.alt_text}</p>
                      {g.status && <Badge tone={statusTone(g.status)}>{g.status}</Badge>}
                    </div>
                    <div className="mt-2 flex gap-3 text-xs font-medium">
                      <button onClick={() => editGallery(g)} className="text-primary hover:underline">Edit</button>
                      <button onClick={() => api.post(`/v1/admin/gallery/${g.id}/publish`).then(() => { toast.success('Published'); loadGallery(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-emerald-600 hover:underline">Publish</button>
                      <button onClick={() => api.delete(`/v1/admin/gallery/${g.id}`).then(() => { toast.success('Archived'); loadGallery(); }).catch((e) => toast.error(errMsg(e, 'Failed')))} className="text-red-600 hover:underline">Archive</button>
                    </div>
                  </div>
                </Card>
              ))}
            </div>
          )}
        </div>
      )}

      {editingUser && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink/50 p-4" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) setEditingUser(null); }}>
          <div className="w-full max-w-2xl overflow-hidden rounded-2xl bg-white shadow-2xl" role="dialog" aria-modal="true" aria-labelledby="edit-user-title">
            <div className="flex items-center justify-between border-b border-line px-6 py-5">
              <div>
                <h2 id="edit-user-title" className="text-xl font-semibold text-ink">Edit User: {editingUser.display_name}</h2>
                <p className="mt-1 text-sm text-muted">Manage this user&apos;s platform administration roles.</p>
              </div>
              <button onClick={() => setEditingUser(null)} className="text-2xl leading-none text-muted hover:text-ink" aria-label="Close">×</button>
            </div>
            <div className="grid gap-5 px-6 py-6 sm:grid-cols-2">
              <Field label="Full Name"><input className={inputCls} value={editingUser.display_name} readOnly /></Field>
              <Field label="Reg No"><input className={inputCls} value={editingUser.student_number} readOnly /></Field>
              <Field label="Email"><input className={inputCls} value={editingUser.email} readOnly /></Field>
              <Field label="Account Status"><input className={inputCls} value={editingUser.status.toLowerCase()} readOnly /></Field>
              <div className="sm:col-span-2">
                <Field label="Platform Role">
                  <div className="flex gap-3">
                    <select className={inputCls} value={selectedRole} onChange={(event) => setSelectedRole(event.target.value)}>
                      <option value="">Select role…</option>
                      {platformRoles.filter((role) => !editingUser.platform_roles.includes(role.code)).map((role) => (
                        <option key={`${role.scope}-${role.code}`} value={role.code}>
                          {role.name} ({role.scope.toLowerCase()})
                        </option>
                      ))}
                    </select>
                    {platformRoles.find((role) => role.code === selectedRole)?.scope === 'BATCH' && (
                      <select className="min-w-48 rounded-xl border border-line bg-white px-3.5 py-2.5 text-[15px] text-ink" value={selectedBatch} onChange={(event) => setSelectedBatch(event.target.value)}>
                        <option value="">Select cohort…</option>
                        {batches.map((batch) => <option key={batch.id} value={batch.id}>{batch.name}</option>)}
                      </select>
                    )}
                    <button disabled={!selectedRole || (platformRoles.find((role) => role.code === selectedRole)?.scope === 'BATCH' && !selectedBatch)} onClick={() => changePlatformRole(selectedRole, false, selectedBatch)} className="shrink-0 rounded-xl bg-primary px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-dark disabled:opacity-40">Assign</button>
                  </div>
                </Field>
                <div className="mt-3 flex flex-wrap gap-2">
                  {(editingUser.platform_roles || []).map((role) => (
                    <span key={role} className="inline-flex items-center gap-2 rounded-full bg-primary-light px-3 py-1.5 text-sm font-medium text-primary">
                      {platformRoles.find((item) => item.code === role)?.name || role}
                      <button onClick={() => changePlatformRole(role, true)} className="text-base leading-none hover:text-red-600" title={`Remove ${role}`} aria-label={`Remove ${role}`}>×</button>
                    </span>
                  ))}
                  {(editingUser.batch_roles || []).map((role) => (
                    <span key={`${role.batch_id}-${role.code}`} className="inline-flex items-center gap-2 rounded-full bg-primary-light px-3 py-1.5 text-sm font-medium text-primary">
                      {role.code} · {role.batch_name}
                      <button onClick={() => changePlatformRole(role.code, true, role.batch_id)} className="text-base leading-none hover:text-red-600" title={`Remove ${role.code}`} aria-label={`Remove ${role.code}`}>×</button>
                    </span>
                  ))}
                  {editingUser.platform_roles.length === 0 && editingUser.batch_roles.length === 0 && <p className="text-sm text-muted">No roles assigned.</p>}
                </div>
              </div>
            </div>
            <div className="flex justify-end gap-3 border-t border-line bg-surface px-6 py-4">
              <button onClick={() => setEditingUser(null)} className="rounded-xl border border-line bg-white px-5 py-2.5 text-[15px] font-medium text-ink hover:bg-surface">Cancel</button>
              <button onClick={() => setEditingUser(null)} className="rounded-xl bg-primary px-5 py-2.5 text-[15px] font-medium text-white hover:bg-primary-dark">Done</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
