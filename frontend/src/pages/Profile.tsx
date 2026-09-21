import { useEffect, useState } from 'react';
import { api, toList, errMsg } from '../api/client';
import { useAuthStore } from '../store/auth';
import toast from 'react-hot-toast';
import type { Session } from '../types';
import { Card, Badge, statusTone, Empty, Field, inputCls, fmtDateTime } from '../components/ui';

export default function Profile() {
  const { user, setUser } = useAuthStore();
  const [form, setForm] = useState({ first_name: '', last_name: '', display_name: '', phone_number: '' });
  const [editing, setEditing] = useState(false);
  const [sessions, setSessions] = useState<Session[]>([]);

  const loadSessions = () => api.get('/v1/me/sessions').then((res) => setSessions(toList<Session>(res.data))).catch(() => {});

  useEffect(() => {
    if (user) setForm({ first_name: user.first_name || '', last_name: user.last_name || '', display_name: user.display_name || '', phone_number: user.phone_number || '' });
    loadSessions();
  }, []);

  const save = async () => {
    try {
      await api.patch('/v1/me/profile', { first_name: form.first_name, last_name: form.last_name, display_name: form.display_name, phone_number: form.phone_number || null });
      const me = await api.get('/v1/me');
      setUser(me.data);
      setEditing(false);
      toast.success('Profile updated');
    } catch (err) {
      toast.error(errMsg(err, 'Update failed'));
    }
  };

  const revokeSession = (id: string) =>
    api.delete(`/v1/me/sessions/${id}`).then(() => { toast.success('Session revoked'); loadSessions(); }).catch((e) => toast.error(errMsg(e, 'Failed')));

  const logoutAll = () =>
    api.post('/v1/auth/logout-all').then(() => { toast.success('Signed out everywhere'); loadSessions(); }).catch((e) => toast.error(errMsg(e, 'Failed')));

  if (!user) return <p className="text-sm text-muted">Loading…</p>;

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <Card className="p-7">
        <div className="flex items-center gap-4">
          <div className="w-16 h-16 rounded-2xl bg-primary-light text-primary text-2xl font-bold flex items-center justify-center">
            {user.display_name[0]?.toUpperCase()}
          </div>
          <div className="flex-1">
            <h2 className="text-xl font-semibold text-ink">{user.display_name}</h2>
            <p className="text-sm text-muted">{user.email} · {user.student_number}{user.combination ? ` · ${user.combination}` : ''}</p>
          </div>
          <Badge tone={statusTone(user.status)}>{user.status}</Badge>
        </div>
        {editing ? (
          <div className="mt-6 grid grid-cols-2 gap-4">
            <Field label="First name"><input className={inputCls} value={form.first_name} onChange={(e) => setForm({ ...form, first_name: e.target.value })} /></Field>
            <Field label="Last name"><input className={inputCls} value={form.last_name} onChange={(e) => setForm({ ...form, last_name: e.target.value })} /></Field>
            <Field label="Display name"><input className={inputCls} value={form.display_name} onChange={(e) => setForm({ ...form, display_name: e.target.value })} /></Field>
            <Field label="Phone number"><input className={inputCls} value={form.phone_number} onChange={(e) => setForm({ ...form, phone_number: e.target.value })} /></Field>
            <div className="col-span-2 flex gap-3">
              <button onClick={save} className="px-5 py-2.5 rounded-xl bg-primary text-white text-sm font-medium">Save</button>
              <button onClick={() => setEditing(false)} className="px-5 py-2.5 rounded-xl border border-line text-sm font-medium">Cancel</button>
            </div>
          </div>
        ) : (
          <div className="mt-6 flex items-center justify-between">
            <p className="text-sm text-muted">Phone: {user.phone_number || '—'}</p>
            <button onClick={() => setEditing(true)} className="px-5 py-2.5 rounded-xl bg-primary text-white text-sm font-medium">Edit profile</button>
          </div>
        )}
      </Card>

      <Card className="p-7">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-ink">Active sessions</h2>
          <button onClick={logoutAll} className="text-sm text-red-600 font-medium hover:underline">Sign out everywhere</button>
        </div>
        {sessions.length === 0 ? <Empty message="No active sessions." /> : (
          <div className="mt-3 divide-y divide-line">
            {sessions.map((s) => (
              <div key={s.id} className="py-3 flex items-center justify-between gap-4">
                <div className="min-w-0">
                  <p className="text-sm font-medium text-ink truncate">{s.user_agent || 'Unknown device'}</p>
                  <p className="text-xs text-muted">Last used {fmtDateTime(s.last_used_at)} · expires {fmtDateTime(s.expires_at)}</p>
                </div>
                <button onClick={() => revokeSession(s.id)} className="text-xs text-red-600 font-medium hover:underline shrink-0">Revoke</button>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
