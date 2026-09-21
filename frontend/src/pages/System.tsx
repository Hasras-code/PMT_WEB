import { useEffect, useState } from 'react';
import { CheckCircleIcon, XCircleIcon } from '@heroicons/react/24/outline';
import { api, toList } from '../api/client';
import { useAppStore } from '../store/app';
import { Card, CardTitle } from '../components/ui';

export default function System() {
  const { currentBatchID } = useAppStore();
  const [live, setLive] = useState<boolean | null>(null);
  const [ready, setReady] = useState<boolean | null>(null);
  const [counts, setCounts] = useState({ members: '—', resources: '—', announcements: '—', events: '—' });

  useEffect(() => {
    api.get('/health/live').then(() => setLive(true)).catch(() => setLive(false));
    api.get('/health/ready').then(() => setReady(true)).catch(() => setReady(false));
    if (!currentBatchID) return;
    const jobs: [string, (v: string) => void][] = [
      [`/v1/batches/${currentBatchID}/members`, (v) => setCounts((c) => ({ ...c, members: v }))],
      [`/v1/batches/${currentBatchID}/resources`, (v) => setCounts((c) => ({ ...c, resources: v }))],
      [`/v1/batches/${currentBatchID}/announcements`, (v) => setCounts((c) => ({ ...c, announcements: v }))],
      [`/v1/batches/${currentBatchID}/events`, (v) => setCounts((c) => ({ ...c, events: v }))],
    ];
    jobs.forEach(([url, set]) =>
      api.get(url, { limit: 1 }).then((res) => set(String(toList(res.data).length))).catch(() => {}),
    );
  }, [currentBatchID]);

  const services = [
    { name: 'API service', ok: live, detail: 'HTTP liveness probe' },
    { name: 'PostgreSQL database', ok: ready, detail: 'Readiness probe (pool ping)' },
  ];

  return (
    <div className="space-y-6">
      <Card className="p-7">
        <CardTitle>Service status</CardTitle>
        <div className="mt-4 space-y-3">
          {services.map((s) => (
            <div key={s.name} className="flex items-center justify-between p-4 rounded-xl border border-line">
              <div className="flex items-center gap-3">
                {s.ok === null ? (
                  <span className="w-6 h-6 rounded-full border-2 border-line border-t-primary animate-spin" />
                ) : s.ok ? (
                  <CheckCircleIcon className="w-6 h-6 text-emerald-600" />
                ) : (
                  <XCircleIcon className="w-6 h-6 text-red-600" />
                )}
                <div>
                  <p className="font-medium text-ink">{s.name}</p>
                  <p className="text-sm text-muted">{s.detail}</p>
                </div>
              </div>
              <span className={`text-sm font-medium ${s.ok ? 'text-emerald-600' : s.ok === false ? 'text-red-600' : 'text-muted'}`}>
                {s.ok === null ? 'Checking…' : s.ok ? 'Operational' : 'Down'}
              </span>
            </div>
          ))}
        </div>
      </Card>

      <Card className="p-7">
        <CardTitle>Data overview</CardTitle>
        <div className="mt-4 grid grid-cols-2 xl:grid-cols-4 gap-4">
          {[
            ['Members', counts.members],
            ['Resources', counts.resources],
            ['Announcements', counts.announcements],
            ['Meetings', counts.events],
          ].map(([label, value]) => (
            <div key={label} className="p-4 rounded-xl bg-surface text-center">
              <p className="text-2xl font-semibold text-ink">{value}</p>
              <p className="text-sm text-muted mt-0.5">{label}</p>
            </div>
          ))}
        </div>
      </Card>

      <Card className="p-7">
        <CardTitle>Backup &amp; recovery</CardTitle>
        <p className="mt-2 text-[15px] text-muted leading-relaxed">
          Database backups, file-store snapshots and restore procedures are handled at the infrastructure level. See{' '}
          <code className="px-1.5 py-0.5 rounded bg-surface text-ink text-sm">docs/operations.md</code> in the project
          repository for the backup schedule, retention policy and disaster-recovery runbook.
        </p>
      </Card>
    </div>
  );
}
