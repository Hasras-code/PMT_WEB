import { useCallback, useEffect, useState } from 'react';
import { api, toList, errMsg } from '../api/client';
import toast from 'react-hot-toast';
import type { AuditLog } from '../types';
import { Card, Badge, Empty, fmtDateTime } from '../components/ui';
import { useAccess, useCan } from '../hooks/useRole';

export default function Monitoring() {
  const access = useAccess();
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(false);
  const canPlatformAudit = useCan('platform_audit.view');
  const platformAdmin = access?.platform_roles.includes('PLATFORM_ADMIN') ?? false;

  const load = useCallback(async () => {
    if (!platformAdmin || !canPlatformAudit) return;
    setLoading(true);
    try {
      const res = await api.get('/v1/admin/audit-logs', { limit: 100 });
      setLogs(toList<AuditLog>(res.data));
    } catch (err) {
      toast.error(errMsg(err, 'Could not load system logs'));
      setLogs([]);
    } finally {
      setLoading(false);
    }
  }, [canPlatformAudit, platformAdmin]);

  useEffect(() => {
    load();
  }, [load]);

  if (!platformAdmin || canPlatformAudit === false) return <Card className="p-8"><Empty message="You do not have permission to view audit logs." /></Card>;

  return (
    <Card className="p-7">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-ink">System Logs</h2>
          <p className="text-sm text-muted mt-0.5">Platform-wide audit trail</p>
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
