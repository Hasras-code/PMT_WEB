import { useEffect, useState } from 'react';
import { api, toList, errMsg } from '../api/client';
import { useAppStore } from '../store/app';
import toast from 'react-hot-toast';
import type { Member, RoleCatalogEntry } from '../types';
import { Card, Badge, statusTone, Empty, inputCls, fmtDate } from '../components/ui';

const LIFECYCLE = ['suspend', 'reactivate', 'graduate', 'leave'] as const;

export default function Users() {
  const { currentBatchID, currentBatch } = useAppStore();
  const [members, setMembers] = useState<Member[]>([]);
  const [roles, setRoles] = useState<RoleCatalogEntry[]>([]);
  const [userID, setUserID] = useState('');
  const [assign, setAssign] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);

  const load = async () => {
    if (!currentBatchID) return;
    setLoading(true);
    try {
      const [m, r] = await Promise.all([
        api.get(`/v1/batches/${currentBatchID}/members`, { limit: 100 }),
        api.get(`/v1/batches/${currentBatchID}/roles`).catch(() => ({ data: [] })),
      ]);
      setMembers(toList<Member>(m.data));
      setRoles(toList<RoleCatalogEntry>(r.data));
    } catch (err) {
      toast.error(errMsg(err, 'Could not load members'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [currentBatchID]);

  const approve = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.post(`/v1/batches/${currentBatchID}/members`, { user_id: userID });
      toast.success('Member approved');
      setUserID('');
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Approval failed'));
    }
  };

  const setRole = async (membershipID: string, role: string, remove: boolean) => {
    try {
      if (remove) {
        await api.delete(`/v1/batches/${currentBatchID}/members/${membershipID}/roles/${role}`);
      } else {
        await api.post(`/v1/batches/${currentBatchID}/members/${membershipID}/roles`, { role });
      }
      toast.success(remove ? 'Role removed' : 'Role assigned');
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Role change failed'));
    }
  };

  const lifecycle = async (membershipID: string, action: string) => {
    try {
      await api.post(`/v1/batches/${currentBatchID}/members/${membershipID}/${action}`);
      toast.success(`Member ${action}d`);
      load();
    } catch (err) {
      toast.error(errMsg(err, 'Action failed'));
    }
  };

  if (!currentBatchID) {
    return (
      <Card className="p-8">
        <Empty message="Select a cohort from the dropdown above to manage its users." />
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <Card className="p-7">
        <h2 className="text-lg font-semibold text-ink">Add New User</h2>
        <p className="text-sm text-muted mt-1">
          Approve a verified account into {currentBatch()?.name || 'this cohort'} as a student.
        </p>
        <form onSubmit={approve} className="mt-4 flex gap-3">
          <input required placeholder="User UUID" className={inputCls} value={userID} onChange={(e) => setUserID(e.target.value)} />
          <button type="submit" className="px-6 rounded-xl bg-primary text-white text-[15px] font-medium hover:bg-primary-dark whitespace-nowrap">
            Approve
          </button>
        </form>
      </Card>

      <Card className="p-7">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-ink">Members ({members.length})</h2>
          <button onClick={load} className="text-sm text-primary font-medium hover:underline">
            Refresh
          </button>
        </div>
        {loading ? (
          <p className="py-6 text-sm text-muted">Loading…</p>
        ) : members.length === 0 ? (
          <Empty message="No members yet." />
        ) : (
          <div className="mt-4 overflow-x-auto">
            <table className="w-full text-left text-[15px]">
              <thead>
                <tr className="text-muted text-sm border-b border-line">
                  <th className="py-3 pr-4 font-medium">Name</th>
                  <th className="py-3 pr-4 font-medium">Combination</th>
                  <th className="py-3 pr-4 font-medium">Status</th>
                  <th className="py-3 pr-4 font-medium">Roles</th>
                  <th className="py-3 pr-4 font-medium">Joined</th>
                  <th className="py-3 pr-4 font-medium">Assign role</th>
                  <th className="py-3 font-medium">Lifecycle</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line">
                {members.map((m) => (
                  <tr key={m.id}>
                    <td className="py-3.5 pr-4">
                      <p className="font-medium text-ink">{m.display_name}</p>
                      <p className="text-xs text-muted font-mono">{m.user_id.slice(0, 8)}…</p>
                    </td>
                    <td className="py-3.5 pr-4 text-sm text-muted">{m.combination || '—'}</td>
                    <td className="py-3.5 pr-4">
                      <Badge tone={statusTone(m.status)}>{m.status}</Badge>
                    </td>
                    <td className="py-3.5 pr-4">
                      <div className="flex flex-wrap gap-1.5">
                        {m.roles.map((r) => (
                          <span key={r} className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium bg-primary-light text-primary">
                            {r}
                            {r !== 'STUDENT' && (
                              <button onClick={() => setRole(m.id, r, true)} className="hover:text-red-600" title={`Remove ${r}`}>
                                ×
                              </button>
                            )}
                          </span>
                        ))}
                      </div>
                    </td>
                    <td className="py-3.5 pr-4 text-muted text-sm">{fmtDate(m.joined_at)}</td>
                    <td className="py-3.5 pr-4">
                      <div className="flex gap-2">
                        <select
                          className="px-3 py-1.5 rounded-lg border border-line bg-white text-sm"
                          value={assign[m.id] || ''}
                          onChange={(e) => setAssign({ ...assign, [m.id]: e.target.value })}
                        >
                          <option value="">Select…</option>
                          {roles.filter((r) => r.code !== 'STUDENT').map((r) => (
                            <option key={r.code} value={r.code}>
                              {r.name || r.code}
                            </option>
                          ))}
                        </select>
                        <button
                          disabled={!assign[m.id]}
                          onClick={() => setRole(m.id, assign[m.id], false)}
                          className="px-3 py-1.5 rounded-lg bg-primary text-white text-sm disabled:opacity-40"
                        >
                          Add
                        </button>
                      </div>
                    </td>
                    <td className="py-3.5">
                      <div className="flex flex-wrap gap-1.5">
                        {LIFECYCLE.map((a) => (
                          <button key={a} onClick={() => lifecycle(m.id, a)} className="px-2.5 py-1 rounded-lg border border-line text-xs text-muted hover:text-ink hover:border-muted">
                            {a}
                          </button>
                        ))}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}
