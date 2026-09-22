import { useEffect, useState } from 'react';
import { Outlet, NavLink, useNavigate, useLocation } from 'react-router-dom';
import {
  ChartBarIcon,
  UsersIcon,
  BookOpenIcon,
  ShieldCheckIcon,
  ChatBubbleLeftIcon,
  VideoCameraIcon,
  PresentationChartLineIcon,
  CircleStackIcon,
  Cog6ToothIcon,
  BellIcon,
} from '@heroicons/react/24/outline';
import { useAuthStore } from '../store/auth';
import { useAppStore } from '../store/app';
import { api, clearSession, toList } from '../api/client';
import type { Batch, NotificationItem } from '../types';
import { useAccess } from '../hooks/useRole';

const NAV = [
  { to: '/admin/dashboard', label: 'Dashboard', Icon: ChartBarIcon, title: 'Dashboard' },
  { to: '/admin/users', label: 'User Management', Icon: UsersIcon, title: 'User Management' },
  { to: '/admin/courses', label: 'Course Management', Icon: BookOpenIcon, title: 'Course Management' },
  { to: '/admin/assessments', label: 'Resources', Icon: ShieldCheckIcon, title: 'Resource Library' },
  { to: '/admin/communication', label: 'Communication', Icon: ChatBubbleLeftIcon, title: 'Communication' },
  { to: '/admin/meetings', label: 'Video Meetings', Icon: VideoCameraIcon, title: 'Video Meetings' },
  { to: '/admin/monitoring', label: 'Monitoring', Icon: PresentationChartLineIcon, title: 'Monitoring' },
  { to: '/admin/system', label: 'Backup & Recovery', Icon: CircleStackIcon, title: 'Backup & Recovery' },
  { to: '/admin/administration', label: 'Administration', Icon: Cog6ToothIcon, title: 'Administration' },
];

const TITLES: Record<string, string> = {
  '/admin/gallery': 'Gallery',
  '/admin/profile': 'Profile',
  '/admin/notifications': 'Notifications',
};

export default function Layout() {
  const { user, setUser } = useAuthStore();
  const { batches, currentBatchID, setBatches, setCurrentBatchID } = useAppStore();
  const navigate = useNavigate();
  const location = useLocation();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [unread, setUnread] = useState(0);
  const access = useAccess();

  const has = (permission: string) => Boolean(access?.platform_permissions.includes(permission) || access?.memberships.some((membership) => membership.permissions.includes(permission)));
  const visibleNav = NAV.filter((item) => {
    if (item.to.endsWith('/users')) return has('membership.manage') || has('role.assign') || has('platform_user.manage');
    if (item.to.endsWith('/assessments')) return has('resource.create');
    if (item.to.endsWith('/communication')) return has('announcement.create') || has('feedback.view') || has('complaint.view_all');
    if (item.to.endsWith('/meetings')) return has('event.manage');
    if (item.to.endsWith('/monitoring')) return has('audit.view') || has('platform_audit.view');
    if (item.to.endsWith('/administration')) return has('platform_user.manage') || has('gallery.manage');
    return true;
  });

  useEffect(() => {
    api
      .get('/v1/batches')
      .then((res) => setBatches(res.data as Batch[]))
      .catch(() => {});
    api
      .get('/v1/me/notifications', { limit: 100 })
      .then((res) => setUnread(toList<NotificationItem>(res.data).filter((n) => !n.read_at).length))
      .catch(() => {});
  }, [setBatches]);

  const activeTitle =
    visibleNav.find((n) => location.pathname === n.to || location.pathname.startsWith(n.to + '/'))?.title ||
    TITLES[location.pathname] ||
    'Dashboard';

  const handleLogout = async () => {
    try {
      const refreshToken = localStorage.getItem('refresh_token');
      if (refreshToken) await api.post('/v1/auth/logout', { refresh_token: refreshToken });
    } catch {
      /* ignore */
    }
    setUser(null);
    clearSession();
    navigate('/login');
  };

  return (
    <div className="flex min-h-screen bg-surface">
      <aside
        className={`fixed inset-y-0 left-0 z-50 w-[300px] bg-white border-r border-line flex flex-col transform transition-transform duration-200 ${
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        } lg:translate-x-0`}
      >
        <div className="px-7 pt-8 pb-6 border-b border-line">
          <p className="text-[26px] leading-[1.25] font-semibold text-primary">
            Learning
            <br />
            Management
            <br />
            System
          </p>
        </div>
        <nav className="flex-1 overflow-y-auto px-4 py-5 space-y-1">
          {visibleNav.map(({ to, label, Icon }) => (
            <NavLink
              key={to}
              to={to}
              onClick={() => setSidebarOpen(false)}
              className={({ isActive }) =>
                `flex items-center gap-3.5 px-4 py-3 rounded-xl text-[15px] transition-colors ${
                  isActive ? 'bg-primary-light text-primary font-medium' : 'text-ink hover:bg-surface font-normal'
                }`
              }
            >
              <Icon className="w-[22px] h-[22px] shrink-0" strokeWidth={1.7} />
              <span className="leading-snug">{label}</span>
            </NavLink>
          ))}
        </nav>
        <div className="p-4 border-t border-line">
          {user ? (
            <div className="flex items-center gap-3 px-2">
              <button
                onClick={() => navigate('/admin/profile')}
                className="w-10 h-10 rounded-full bg-primary-light text-primary font-semibold flex items-center justify-center shrink-0"
              >
                {user.display_name[0]?.toUpperCase()}
              </button>
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium text-ink truncate">{user.display_name}</p>
                <button onClick={handleLogout} className="text-xs text-muted hover:text-red-600">
                  Log out
                </button>
              </div>
            </div>
          ) : (
            <NavLink to="/login" className="text-sm text-primary font-medium px-2">
              Sign in
            </NavLink>
          )}
        </div>
      </aside>

      {sidebarOpen && (
        <div className="fixed inset-0 z-40 bg-ink/30 lg:hidden" onClick={() => setSidebarOpen(false)} />
      )}

      <div className="flex-1 flex flex-col min-w-0 lg:pl-[300px]">
        <header className="sticky top-0 z-30 bg-white border-b border-line">
          <div className="flex items-center justify-between px-8 h-[104px]">
            <div className="flex items-center gap-4">
              <button onClick={() => setSidebarOpen(true)} className="lg:hidden text-ink" aria-label="Open menu">
                <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                </svg>
              </button>
              <h1 className="text-[22px] font-medium text-ink">{activeTitle}</h1>
            </div>
            <div className="flex items-center gap-4">
              <button
                onClick={() => navigate('/admin/notifications')}
                className="relative p-2 text-ink hover:text-primary"
                aria-label="Notifications"
              >
                <BellIcon className="w-6 h-6" strokeWidth={1.7} />
                {unread > 0 && <span className="absolute top-1.5 right-1.5 w-2.5 h-2.5 rounded-full bg-red-500" />}
              </button>
              <select
                value={currentBatchID}
                onChange={(e) => setCurrentBatchID(e.target.value)}
                className="px-4 py-2.5 rounded-xl border border-line bg-white text-[15px] text-ink focus:outline-none focus:ring-2 focus:ring-primary/40 max-w-[220px]"
                aria-label="Select cohort"
              >
                {batches.length === 0 && <option value="">No cohort</option>}
                {batches.map((b) => (
                  <option key={b.id} value={b.id}>
                    {b.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </header>
        <main className="flex-1 px-8 py-8">
          <div className="max-w-[1400px] mx-auto">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
