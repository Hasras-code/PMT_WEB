import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { UsersIcon, BookOpenIcon, AcademicCapIcon, ArrowTrendingUpIcon } from '@heroicons/react/24/outline';
import { api, toList } from '../api/client';
import { useAuthStore } from '../store/auth';
import { useAppStore } from '../store/app';
import type { AuditLog, Batch } from '../types';
import { Card, CardTitle, PrimaryButton, OutlineButton, timeAgo } from '../components/ui';
import CreateCourse from '../components/CreateCourse';
import { useAccess, useCan } from '../hooks/useRole';

const DOTS = ['bg-primary', 'bg-orange-500', 'bg-emerald-500', 'bg-purple-500'];

/** Admin portal dashboard (Figma admin view). RequireAdmin guarantees the viewer is elevated. */
export default function Dashboard() {
  const navigate = useNavigate();
  const { user } = useAuthStore();
  const { batches, currentBatchID, setBatches } = useAppStore();
  const [stats, setStats] = useState({ users: '—', courses: '—', assessments: '—', uptime: '—' });
  const [activity, setActivity] = useState<AuditLog[]>([]);
  const [showCreate, setShowCreate] = useState(false);
  const canCreateBatch = useCan('batch.create');
  const canManageMembers = useCan('membership.manage', currentBatchID);
  const canViewPlatformAudit = useCan('platform_audit.view');
  const access = useAccess();
  const platformAdmin = access?.platform_roles.includes('PLATFORM_ADMIN') ?? false;

  useEffect(() => {
    api.get('/v1/batches').then((res) => setBatches(res.data as Batch[])).catch(() => {});
  }, [setBatches]);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      const next = { users: '—', courses: String(batches.length) || '—', assessments: '—', uptime: '—' };
      try {
        // The admin dashboard uses the existing application session. Protected
        // infrastructure probes are checked explicitly from the system page.
        await api.get('/v1/me');
        next.uptime = 'Operational';
      } catch {
        next.uptime = 'Down';
      }
      try {
        const platformStats = await api.get('/v1/admin/stats');
        next.users = String(platformStats.data.users ?? '—');
      } catch {
        /* Non-platform admins use the selected cohort count below. */
      }
      if (currentBatchID) {
        try {
          const summary = await api.get(`/v1/batches/${currentBatchID}/summary`);
          if (next.users === '—') next.users = String(summary.data.members ?? '—');
          next.assessments = String(summary.data.resources ?? '—');
        } catch {
          /* ignore */
        }
        try {
          const a = await api.get(`/v1/batches/${currentBatchID}/audit-logs`, { limit: 8 });
          if (!cancelled) setActivity(toList<AuditLog>(a.data).slice(0, 6));
        } catch {
          if (!cancelled) setActivity([]);
        }
      }
      if (!cancelled) setStats(next);
    };
    load();
    return () => {
      cancelled = true;
    };
  }, [currentBatchID, batches.length]);

  const tiles = [
    { label: 'Total Users', value: stats.users, Icon: UsersIcon, tile: 'bg-blue-50 text-primary' },
    { label: 'Active Courses', value: stats.courses, Icon: BookOpenIcon, tile: 'bg-emerald-50 text-emerald-600' },
    { label: 'Resources', value: stats.assessments, Icon: AcademicCapIcon, tile: 'bg-purple-50 text-purple-600' },
    { label: 'System Uptime', value: stats.uptime, Icon: ArrowTrendingUpIcon, tile: 'bg-orange-50 text-orange-600' },
  ];

  return (
    <div className="space-y-7">
      <div className="bg-primary rounded-2xl px-8 py-9">
        <h1 className="text-[26px] font-bold text-white">Welcome back, {user?.display_name || 'Admin User'}!</h1>
        <p className="mt-2 text-white/85 text-[16px]">Monitor and manage your learning platform.</p>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-6">
        {tiles.map(({ label, value, Icon, tile }) => (
          <Card key={label} className="p-6">
            <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${tile}`}>
              <Icon className="w-6 h-6" strokeWidth={1.8} />
            </div>
            <p className="mt-5 text-[15px] text-muted">{label}</p>
            <p className="mt-1 text-[26px] font-semibold text-ink">{value}</p>
          </Card>
        ))}
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
        <Card className="p-7">
          <CardTitle>Recent Activity</CardTitle>
          <div className="mt-4 divide-y divide-line">
            {activity.length === 0 && <p className="py-6 text-sm text-muted">No recent activity in this cohort.</p>}
            {activity.map((a, i) => (
              <div key={a.id} className="py-4 first:pt-2">
                <div className="flex items-start gap-3">
                  <span className={`mt-1.5 w-2.5 h-2.5 rounded-full shrink-0 ${DOTS[i % DOTS.length]}`} />
                  <p className="text-[15px] text-ink leading-snug">
                    {a.action.replaceAll('_', ' ').toLowerCase()} · {a.entity_type}
                  </p>
                </div>
                <p className="ml-[22px] mt-1 text-sm text-muted">{timeAgo(a.created_at)}</p>
              </div>
            ))}
          </div>
        </Card>
        <Card className="p-7">
          <CardTitle>Quick Actions</CardTitle>
          <div className="mt-5 space-y-4">
            {canManageMembers && <PrimaryButton onClick={() => navigate('/admin/users')}>Add New User</PrimaryButton>}
            {canCreateBatch && <OutlineButton onClick={() => setShowCreate(true)}>Create Course</OutlineButton>}
            {platformAdmin && canViewPlatformAudit && <OutlineButton onClick={() => navigate('/admin/monitoring')}>View System Logs</OutlineButton>}
          </div>
        </Card>
      </div>

      <CreateCourse open={showCreate} onClose={() => setShowCreate(false)} />
    </div>
  );
}
