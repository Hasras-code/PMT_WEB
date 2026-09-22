import { useCallback, useEffect, useState } from 'react';
import { api, toList, errMsg } from '../api/client';
import { useAppStore } from '../store/app';
import toast from 'react-hot-toast';
import type { AuditLog } from '../types';
import { Card, Badge, Empty, fmtDateTime } from '../components/ui';
import { useCan } from '../hooks/useRole';

export default function Monitoring() {
  const { currentBatchID, currentBatch } = useAppStore();
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [scope, setScope] = useState<'cohort' | 'platform'>('cohort');
  const [loading, setLoading] = useState(false);
  const canBatchAudit = useCan('audit.view', currentBatchID);
  const canPlatformAudit = useCan('platform_audit.view');

  const load = useCallback(async () => {
    if ((scope === 'cohort' && !canBatchAudit) || (scope === 'platform' && !canPlatformAudit)) return;
    setLoading(true);
    try {
      const url = scope === 'cohort' && currentBatchID ? `/v1/batches/${currentBatchID}/audit-logs` : '/v1/admin/audit-logs';
      const res = await api.get(url, { limit: 100 });
      setLogs(toList<AuditLog>(res.data));
    } catch (err) {
      toast.error(errMsg(err, 'Could not load system logs'));
      setLogs([]);
    } finally {
      setLoading(false);
    }
  }, [canBatchAudit, canPlatformAudit, currentBatchID, scope]);

  useEffect(() => {
    if (!canBatchAudit && canPlatformAudit) setScope('platform');
  }, [canBatchAudit, canPlatformAudit]);

  useEffect(() => {
    load();
  }, [load]);

  if (canBatchAudit === false && canPlatformAudit === false) return <Card className="p-8"><Empty message="You do not have permission to view audit logs." /></Card>;

  return (
    <Card className="p-7">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-ink">System Logs</h2>
          <p className="text-sm text-muted mt-0.5">
            {scope === 'cohort' ? `Audit trail for ${currentBatch()?.name || 'this cohort'}` : 'Platform-wide audit trail'}
          </p>
        </div>
        <div className="flex gap-2">
          {canBatchAudit && <button onClick={() => setScope('cohort')} className={`px-4 py-2 rounded-xl text-sm font-medium ${scope === 'cohort' ? 'bg-primary text-white' : 'border border-line text-muted'}`}>
            Cohort
          </button>}
          {canPlatformAudit && <button onClick={() => setScope('platform')} className={`px-4 py-2 rounded-xl text-sm font-medium ${scope === 'platform' ? 'bg-primary text-white' : 'border border-line text-muted'}`}>
            Platform
          </button>}
        </div>
      </div>
      {loading ? (
          <p className="py-6 text-sm text-muted" role="status">Loading…</p>
      ) : logs.length === 0 ? (
        <Empty message="No log entries." />
      ) : (
        <div className="mt-4 overflow-x-auto">
          <table className="w-full text-left text-[15px]">
            <caption className="sr-only">Audit log entries</caption>
            <thead>
              <tr className="text-muted text-sm border-b border-line">
                <th scope="col" className="py-3 pr-4 font-medium">Time</th>
                <th scope="col" className="py-3 pr-4 font-medium">Action</th>
                <th scope="col" className="py-3 pr-4 font-medium">Entity</th>
                <th scope="col" className="py-3 pr-4 font-medium">Actor</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {logs.map((l) => (
                <tr key={l.id}>
                  <td className="py-3 pr-4 text-muted text-sm whitespace-nowrap">{fmtDateTime(l.created_at)}</td>
                  <td className="py-3 pr-4"><Badge tone="blue">{l.action.replaceAll('_', ' ')}</Badge></td>
                  <td className="py-3 pr-4 text-sm text-ink font-mono" title={l.entity_id || undefined}>{l.entity_type}{l.entity_id ? ` · ${l.entity_id.slice(0, 8)}…` : ''}</td>
                  <td className="py-3 text-sm text-muted font-mono" title={l.actor_user_id || undefined}>{l.actor_user_id ? l.actor_user_id.slice(0, 8) + '…' : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}
