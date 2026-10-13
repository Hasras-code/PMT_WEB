import { useCallback, useEffect, useState } from 'react';
import { api, toList, errMsg, uploadFile } from '../api/client';
import toast from 'react-hot-toast';
import type { AdminBatch, AdminUser, AdminUserPage, GalleryImage, PlatformRole } from '../types';
import { Card, Badge, statusTone, Empty, Field, inputCls, fmtDate, Tabs } from '../components/ui';
import { useCan } from '../hooks/useRole';
import { usePatchEditor } from '../hooks/usePatchEditor';

const USER_PAGE_SIZE = 20;

export default function Administration() {
  const canUsers = useCan('platform_user.manage');
  const canGallery = useCan('gallery.manage');
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [usersTotal, setUsersTotal] = useState(0);
  const [usersPage, setUsersPage] = useState(0);
  const [userSearch, setUserSearch] = useState('');
  const [searchInput, setSearchInput] = useState('');
  const [usersLoading, setUsersLoading] = useState(false);
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
  const [actingID, setActingID] = useState('');

  const loadUsers = useCallback(async (query = userSearch, page = usersPage) => {
    setUsersLoading(true);
    try {
      const response = await api.get('/v1/admin/users', { query, limit: USER_PAGE_SIZE, offset: page * USER_PAGE_SIZE });
      const result = response.data as AdminUserPage;
      const items = toList<AdminUser>(result.items);
      setUsers(items);
      setUsersTotal(Number(result.total) || 0);
      return items;
    } catch (err) {
      toast.error(errMsg(err, 'Could not load users'));
      return [];
    } finally {
      setUsersLoading(false);
    }
  }, [userSearch, usersPage]);
  const loadPlatformRoles = () => api.get('/v1/admin/roles').then((res) => setPlatformRoles(toList<PlatformRole>(res.data))).catch((e) => toast.error(errMsg(e, 'Could not load platform roles')));
  const loadBatches = () => api.get('/v1/admin/batches').then((res) => setBatches(toList<AdminBatch>(res.data))).catch((e) => toast.error(errMsg(e, 'Could not load cohorts')));
  const loadGallery = () => api.get('/v1/admin/gallery', { limit: 100 }).then((res) => setGallery(toList<GalleryImage>(res.data))).catch(() => {});
  const editor = usePatchEditor(loadGallery, 'Gallery image updated');
  useEffect(() => {
    if (canUsers) {
      loadUsers();
    }
  }, [canUsers, loadUsers]);

  useEffect(() => {
    if (canUsers) {
      loadPlatformRoles();
      loadBatches();
    }
    if (canGallery) loadGallery();
    if (!canUsers && canGallery) setTab('gallery');
  }, [canGallery, canUsers]);

  const userStatus = (id: string, action: 'suspend' | 'reactivate' | 'archive') => {
    if (actingID) return;
    setActingID(id);
    api.post(`/v1/admin/users/${id}/${action}`).then(() => { toast.success(`User ${action}d`); loadUsers(); }).catch((e) => toast.error(errMsg(e, 'Action failed'))).finally(() => setActingID(''));
  };

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
      const updatedUsers = await loadUsers();
      setEditingUser(updatedUsers.find((user) => user.id === editingUser.id) || null);
      setSelectedRole('');
      setSelectedBatch('');
    } catch (err) {
      toast.error(errMsg(err, 'Role change failed'));
    }
  };

  const authorizeImage = async (f: File) => {
    const init = await api.post('/v1/admin/gallery/uploads', { file_name: f.name, mime_type: f.type, size_bytes: f.size });
		await uploadFile(init.data, f);
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

  const editGallery = (image: GalleryImage) => {
    editor.open('Edit gallery image', `/v1/admin/gallery/${image.id}`, [
      { key: 'title', label: 'Image title', value: image.title || '' },
      { key: 'caption', label: 'Caption', value: image.caption || '', multiline: true },
      { key: 'alt_text', label: 'Alternative text', value: image.alt_text, required: true },
    ]);
  };

  if (canUsers === false && canGallery === false) return <Card className="p-8"><Empty message="You do not have platform administration permission." /></Card>;

  return (
    <div className="space-y-6">
      <Card className="px-4 pt-2">
        <Tabs
          tabs={(['users', 'gallery'] as const).filter((item) => (item === 'users' ? canUsers : canGallery))}
          active={tab}
          onChange={setTab}
          labels={{ users: 'Platform users', gallery: 'Gallery' }}
        />
      </Card>

      {tab === 'users' && (
        <Card className="p-7">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <h2 className="text-lg font-semibold text-ink">Platform users ({usersTotal})</h2>
            <form onSubmit={(event) => { event.preventDefault(); setUsersPage(0); setUserSearch(searchInput.trim()); }} className="flex w-full gap-2 sm:max-w-xl">
              <input aria-label="Search platform users" className={inputCls} placeholder="Search name, email, or student number…" value={searchInput} onChange={(event) => setSearchInput(event.target.value)} />
              <button type="submit" className="shrink-0 rounded-xl bg-primary px-4 py-2.5 text-sm font-medium text-white hover:bg-primary-dark">Search</button>
              {(searchInput || userSearch) && <button type="button" onClick={() => { setSearchInput(''); setUserSearch(''); setUsersPage(0); }} className="shrink-0 rounded-xl border border-line px-4 py-2.5 text-sm text-muted hover:text-ink">Clear</button>}
            </form>
          </div>
          {usersLoading ? (
            <p className="py-8 text-sm text-muted" role="status">Loading users…</p>
          ) : users.length === 0 ? (
            <div className="py-6"><Empty message={userSearch ? 'No users match your search.' : 'No users found.'} /></div>
          ) : (
            <div className="mt-4 overflow-x-auto">
              <table className="w-full text-left text-[15px]">
                <caption className="sr-only">Platform users</caption>
                <thead>
                  <tr className="text-muted text-sm border-b border-line">
                    <th scope="col" className="py-3 pr-4 font-medium">User</th>
                    <th scope="col" className="py-3 pr-4 font-medium">Student #</th>
                    <th scope="col" className="py-3 pr-4 font-medium">Combination</th>
                    <th scope="col" className="py-3 pr-4 font-medium">Status</th>
                    <th scope="col" className="py-3 pr-4 font-medium">Joined</th>
                    <th scope="col" className="py-3 font-medium">Actions</th>
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
                          <button disabled={actingID === u.id} onClick={() => userStatus(u.id, 'suspend')} className="text-orange-600 hover:underline disabled:opacity-50">Suspend</button>
                          <button disabled={actingID === u.id} onClick={() => userStatus(u.id, 'reactivate')} className="text-emerald-600 hover:underline disabled:opacity-50">Reactivate</button>
                          <button disabled={actingID === u.id} onClick={() => userStatus(u.id, 'archive')} className="text-red-600 hover:underline disabled:opacity-50">Archive</button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          <div className="mt-5 flex flex-col gap-3 border-t border-line pt-4 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-sm text-muted">{usersTotal === 0 ? 'No users to show' : `Showing ${usersPage * USER_PAGE_SIZE + 1}–${Math.min((usersPage + 1) * USER_PAGE_SIZE, usersTotal)} of ${usersTotal}`}</p>
            <div className="flex items-center gap-2">
              <button type="button" disabled={usersLoading || usersPage === 0} onClick={() => setUsersPage((page) => Math.max(0, page - 1))} className="rounded-lg border border-line px-3 py-2 text-sm text-ink disabled:opacity-40">Previous</button>
              <span className="px-2 text-sm text-muted">Page {usersTotal === 0 ? 0 : usersPage + 1} of {Math.max(1, Math.ceil(usersTotal / USER_PAGE_SIZE))}</span>
              <button type="button" disabled={usersLoading || (usersPage + 1) * USER_PAGE_SIZE >= usersTotal} onClick={() => setUsersPage((page) => page + 1)} className="rounded-lg border border-line px-3 py-2 text-sm text-ink disabled:opacity-40">Next</button>
            </div>
          </div>
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
              <form onSubmit={uploadGallery} className="grid grid-cols-1 sm:grid-cols-2 gap-4">
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
                  <img src={g.thumbnail_url || g.display_url} alt={g.alt_text} className="w-full h-44 object-cover" loading="lazy" decoding="async" />
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
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink/50 p-4" onMouseDown={(event) => { if (event.target === event.currentTarget) setEditingUser(null); }}>
          <div
            className="w-full max-w-2xl overflow-hidden rounded-2xl border border-line bg-charcoal-card shadow-2xl"
            role="dialog"
            aria-modal="true"
            aria-labelledby="edit-user-title"
            onKeyDown={(event) => { if (event.key === 'Escape') setEditingUser(null); }}
            tabIndex={-1}
            ref={(el) => { el?.focus(); }}
          >
            <div className="flex items-center justify-between border-b border-line px-6 py-5">
              <div>
                <h2 id="edit-user-title" className="text-xl font-semibold text-ink">Edit User: {editingUser.display_name}</h2>
                <p className="mt-1 text-sm text-muted">Manage this user&apos;s platform and cohort roles.</p>
              </div>
              <button onClick={() => setEditingUser(null)} className="text-2xl leading-none text-muted hover:text-ink" aria-label="Close user editor"><span aria-hidden="true">×</span></button>
            </div>
            <div className="grid gap-5 px-6 py-6 sm:grid-cols-2">
              <Field label="Full Name"><input className={inputCls} value={editingUser.display_name} readOnly /></Field>
              <Field label="Reg No"><input className={inputCls} value={editingUser.student_number} readOnly /></Field>
              <Field label="Email"><input className={inputCls} value={editingUser.email} readOnly /></Field>
              <Field label="Account Status"><input className={inputCls} value={editingUser.status.toLowerCase()} readOnly /></Field>
              <div className="sm:col-span-2">
                <Field label="Assign role">
                  <div className="flex gap-3">
                    <select className={inputCls} value={selectedRole} onChange={(event) => setSelectedRole(event.target.value)}>
                      <option value="">Select role…</option>
                      {platformRoles.filter((role) => role.scope === 'PLATFORM'
                        ? !editingUser.platform_roles.includes(role.code)
                        : !selectedBatch || !editingUser.batch_roles.some((assigned) => assigned.batch_id === selectedBatch && assigned.code === role.code))
                        .map((role) => (
                        <option key={`${role.scope}-${role.code}`} value={role.code}>
                          {role.name} ({role.scope.toLowerCase()})
                        </option>
                      ))}
                    </select>
                    {platformRoles.find((role) => role.code === selectedRole)?.scope === 'BATCH' && (
                      <select aria-label="Select cohort for batch role" className="min-w-48 rounded-xl border border-line bg-charcoal-card px-3.5 py-2.5 text-[15px] text-ink" value={selectedBatch} onChange={(event) => setSelectedBatch(event.target.value)}>
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
                      <button onClick={() => changePlatformRole(role, true)} className="text-base leading-none hover:text-red-600" title={`Remove ${role}`} aria-label={`Remove ${role}`}><span aria-hidden="true">×</span></button>
                    </span>
                  ))}
                  {(editingUser.batch_roles || []).map((role) => (
                    <span key={`${role.batch_id}-${role.code}`} className="inline-flex items-center gap-2 rounded-full bg-primary-light px-3 py-1.5 text-sm font-medium text-primary">
                      {role.code} · {role.batch_name}
                      <button onClick={() => changePlatformRole(role.code, true, role.batch_id)} className="text-base leading-none hover:text-red-600" title={`Remove ${role.code}`} aria-label={`Remove ${role.code}`}><span aria-hidden="true">×</span></button>
                    </span>
                  ))}
                  {editingUser.platform_roles.length === 0 && editingUser.batch_roles.length === 0 && <p className="text-sm text-muted">No roles assigned.</p>}
                </div>
              </div>
            </div>
            <div className="flex justify-end gap-3 border-t border-line bg-surface px-6 py-4">
              <button onClick={() => setEditingUser(null)} className="rounded-xl border border-line bg-charcoal-card px-5 py-2.5 text-[15px] font-medium text-ink hover:bg-surface">Cancel</button>
              <button onClick={() => setEditingUser(null)} className="rounded-xl bg-primary px-5 py-2.5 text-[15px] font-medium text-white hover:bg-primary-dark">Done</button>
            </div>
          </div>
        </div>
      )}
      {editor.dialog}
    </div>
  );
}
