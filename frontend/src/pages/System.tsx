import { useCallback, useEffect, useState } from 'react';
import { CheckCircleIcon, XCircleIcon, LockClosedIcon } from '@heroicons/react/24/outline';
import { api, basicAuthHeader } from '../api/client';
import { useAppStore } from '../store/app';
import { Card, CardTitle, Field, inputCls } from '../components/ui';

export default function System() {
  const { currentBatchID } = useAppStore();
  const [live, setLive] = useState<boolean | null>(null);
  const [ready, setReady] = useState<boolean | null>(null);
  const [protectedHealth, setProtectedHealth] = useState(false);
  const [basicUser, setBasicUser] = useState(sessionStorage.getItem('basic_user') || '');
  const [basicPass, setBasicPass] = useState(sessionStorage.getItem('basic_pass') || '');
  const [counts, setCounts] = useState({ members: '—', resources: '—', announcements: '—', events: '—' });

  const check = useCallback(async () => {
    setLive(null);
    setReady(null);
    setProtectedHealth(false);
    const headers = basicAuthHeader();
    try {
      await api.get('/health/live', undefined, { headers });
      setLive(true);
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status;
      if (status === 401) setProtectedHealth(true);
      setLive(false);
    }
    try {
      await api.get('/health/ready', undefined, { headers });
      setReady(true);
    } catch {
      setReady(false);
    }
  }, []);

  useEffect(() => {
    check();
    if (!currentBatchID) return;
    api.get(`/v1/batches/${currentBatchID}/summary`).then((response) => setCounts({
      members: String(response.data.members),
      resources: String(response.data.resources),
      announcements: String(response.data.announcements),
      events: String(response.data.events),
    })).catch(() => {});
  }, [check, currentBatchID]);

  const services = [
    { name: 'API service', ok: live, detail: 'HTTP liveness probe' },
    { name: 'PostgreSQL database', ok: ready, detail: 'Readiness probe (pool ping)' },
  ];

  return (
    <div className="space-y-6">
      <Card className="p-7">
        <CardTitle>Service status</CardTitle>
        <p className="mt-1 text-sm text-muted">Health endpoints require HTTP Basic Auth (see AUTH_BASIC_USER / AUTH_BASIC_PASS).</p>
        <form
          className="mt-4 grid grid-cols-3 gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            sessionStorage.setItem('basic_user', basicUser);
            sessionStorage.setItem('basic_pass', basicPass);
            check();
          }}
        >
          <Field label="Basic-auth user">
            <input className={inputCls} value={basicUser} onChange={(e) => setBasicUser(e.target.value)} autoComplete="username" />
          </Field>
          <Field label="Basic-auth password">
            <input type="password" className={inputCls} value={basicPass} onChange={(e) => setBasicPass(e.target.value)} autoComplete="current-password" />
          </Field>
          <div className="flex items-end">
            <button className="px-5 py-2.5 rounded-xl bg-primary text-white text-sm font-medium">Check status</button>
          </div>
        </form>
        {protectedHealth && (
          <p className="mt-3 flex items-center gap-2 text-sm text-orange-700">
            <LockClosedIcon className="w-4 h-4" /> Health endpoints rejected the credentials — enter the basic-auth user and password above.
          </p>
        )}
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
