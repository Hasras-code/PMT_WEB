import { useCallback, useEffect, useState } from 'react';
import { api, toList, errMsg } from '../api/client';
import { useAppStore } from '../store/app';
import toast from 'react-hot-toast';
import type { Member, RoleCatalogEntry } from '../types';
import { Card, Badge, statusTone, Empty, inputCls, fmtDate } from '../components/ui';
import { useAccess, useCan } from '../hooks/useRole';

const LIFECYCLE = ['suspend', 'reactivate', 'graduate', 'leave'] as const;

interface MembershipCandidate {
  id: string;
  student_number: string;
  combination: string | null;
  display_name: string;
  email: string;
}

export default function Users() {
  const { currentBatchID, currentBatch } = useAppStore();
  const [members, setMembers] = useState<Member[]>([]);
  const [roles, setRoles] = useState<RoleCatalogEntry[]>([]);
  const [userID, setUserID] = useState('');
  const [query, setQuery] = useState('');
  const [candidates, setCandidates] = useState<MembershipCandidate[]>([]);
  const [searching, setSearching] = useState(false);
  const [assign, setAssign] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const canManage = useCan('membership.manage', currentBatchID);
  const canAssignBatchRoles = useCan('role.assign', currentBatchID);
  const access = useAccess();
  const platformAdmin = access?.platform_roles.includes('PLATFORM_ADMIN') ?? false;
  const currentRoles = access?.memberships.find((membership) => membership.batch_id === currentBatchID)?.roles || [];
  const batchRep = currentRoles.includes('BATCH_REP');
  const academicRep = currentRoles.includes('ACADEMIC_REP');
  const canAssignRoles = platformAdmin || canAssignBatchRoles === true;
  const canChangeRole = (role: string) => platformAdmin || batchRep || (academicRep && role !== 'BATCH_REP');

  const load = useCallback(async () => {
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
  }, [currentBatchID]);

  useEffect(() => {
    load();
  }, [load]);

  const searchCandidates = async (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim().length < 2) return;
    setSearching(true);
    try {
      const response = await api.get(`/v1/batches/${currentBatchID}/members/candidates`, { query: query.trim(), limit: 20 });
      setCandidates(toList<MembershipCandidate>(response.data));
    } catch (err) {
      toast.error(errMsg(err, 'Search failed'));
    } finally {
      setSearching(false);
    }
  };

  const approve = async (candidateID: string) => {
    setUserID(candidateID);
    try {
      await api.post(`/v1/batches/${currentBatchID}/members`, { user_id: candidateID });
      toast.success('Member approved');
      setUserID('');
      setCandidates((items) => items.filter((candidate) => candidate.id !== candidateID));
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
      {canManage && <Card className="p-7">
        <h2 className="text-lg font-semibold text-ink">Add New User</h2>
        <p className="text-sm text-muted mt-1">
          Approve a verified account into {currentBatch()?.name || 'this cohort'} as a student.
        </p>
        <form onSubmit={searchCandidates} className="mt-4 flex gap-3">
          <input required minLength={2} placeholder="Search by student number, name, or email" className={inputCls} value={query} onChange={(e) => setQuery(e.target.value)} />
          <button type="submit" className="px-6 rounded-xl bg-primary text-white text-[15px] font-medium hover:bg-primary-dark whitespace-nowrap">
            {searching ? 'Searching…' : 'Search'}
          </button>
        </form>
        {candidates.length > 0 && (
          <div className="mt-4 divide-y divide-line rounded-xl border border-line">
            {candidates.map((candidate) => (
              <div key={candidate.id} className="p-4 flex items-center justify-between gap-4">
                <div>
                  <p className="font-medium text-ink">{candidate.display_name}</p>
                  <p className="text-sm text-muted">{candidate.student_number} · {candidate.combination || '—'} · {candidate.email}</p>
                </div>
                <button disabled={userID === candidate.id} onClick={() => approve(candidate.id)} className="px-4 py-2 rounded-xl bg-primary text-white text-sm font-medium disabled:opacity-50">
                  {userID === candidate.id ? 'Approving…' : 'Approve'}
                </button>
              </div>
            ))}
          </div>
        )}
        {query.length >= 2 && !searching && candidates.length === 0 && <p className="mt-3 text-sm text-muted">Search for a verified account that is not already in this cohort.</p>}
      </Card>}

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
                  {canAssignRoles && <th className="py-3 pr-4 font-medium">Assign role</th>}
                  {canManage && <th className="py-3 font-medium">Lifecycle</th>}
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
                            {canAssignRoles && r !== 'STUDENT' && canChangeRole(r) && (
                              <button onClick={() => setRole(m.id, r, true)} className="hover:text-red-600" title={`Remove ${r}`}>
                                ×
                              </button>
                            )}
                          </span>
                        ))}
                      </div>
                    </td>
                    <td className="py-3.5 pr-4 text-muted text-sm">{fmtDate(m.joined_at)}</td>
                    {canAssignRoles && <td className="py-3.5 pr-4">
                      <div className="flex gap-2">
                        <select
                          className="px-3 py-1.5 rounded-lg border border-line bg-white text-sm"
                          value={assign[m.id] || ''}
                          onChange={(e) => setAssign({ ...assign, [m.id]: e.target.value })}
                        >
                          <option value="">Select…</option>
                          {roles.filter((r) => r.code !== 'STUDENT' && canChangeRole(r.code)).map((r) => (
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
                    </td>}
                    {canManage && <td className="py-3.5">
                      <div className="flex flex-wrap gap-1.5">
                        {LIFECYCLE.map((a) => (
                          <button key={a} onClick={() => lifecycle(m.id, a)} className="px-2.5 py-1 rounded-lg border border-line text-xs text-muted hover:text-ink hover:border-muted">
                            {a}
                          </button>
                        ))}
                      </div>
                    </td>}
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
