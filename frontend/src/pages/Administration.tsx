import { useEffect, useState } from 'react';
import { api, toList, errMsg } from '../api/client';
import toast from 'react-hot-toast';
import type { AdminUser, GalleryImage } from '../types';
import { Card, Badge, statusTone, Empty, Field, inputCls, fmtDate } from '../components/ui';

export default function Administration() {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [gallery, setGallery] = useState<GalleryImage[]>([]);
  const [tab, setTab] = useState<'users' | 'gallery'>('users');
  const [showUpload, setShowUpload] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [thumb, setThumb] = useState<File | null>(null);
  const [gform, setGform] = useState({ title: '', caption: '', alt_text: '' });
  const [busy, setBusy] = useState(false);

  const loadUsers = () => api.get('/v1/admin/users', { limit: 100 }).then((res) => setUsers(toList<AdminUser>(res.data))).catch(() => {});
  const loadGallery = () => api.get('/v1/admin/gallery', { limit: 100 }).then((res) => setGallery(toList<GalleryImage>(res.data))).catch(() => {});
  useEffect(() => {
    loadUsers();
    loadGallery();
  }, []);

  const userStatus = (id: string, action: 'suspend' | 'reactivate' | 'archive') =>
    api.post(`/v1/admin/users/${id}/${action}`).then(() => { toast.success(`User ${action}d`); loadUsers(); }).catch((e) => toast.error(errMsg(e, 'Action failed')));

  const authorizeImage = async (f: File) => {
    const init = await api.post('/v1/admin/gallery/uploads', { file_name: f.name, mime_type: f.type, size_bytes: f.size });
    await fetch(init.data.upload_url, { method: 'PUT', body: f, headers: { 'Content-Type': f.type } });
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

  return (
    <div className="space-y-6">
      <Card className="px-4 pt-2 flex gap-1">
        {(['users', 'gallery'] as const).map((t) => (
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
                      <td className="py-3 pr-4"><Badge tone={statusTone(u.status)}>{u.status}</Badge></td>
                      <td className="py-3 pr-4 text-muted text-sm">{fmtDate(u.created_at)}</td>
                      <td className="py-3">
                        <div className="flex gap-2 text-xs font-medium">
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
    </div>
  );
}
