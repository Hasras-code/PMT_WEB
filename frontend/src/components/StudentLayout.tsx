import { useEffect, useState } from 'react';
import { Outlet, NavLink, useNavigate, useLocation } from 'react-router-dom';
import {
  ChartBarIcon,
  BookOpenIcon,
  ShieldCheckIcon,
  VideoCameraIcon,
  ChatBubbleLeftIcon,
  PhotoIcon,
  BellIcon,
  BanknotesIcon,
} from '@heroicons/react/24/outline';
import { useAuthStore } from '../store/auth';
import { useAppStore } from '../store/app';
import { api, toList } from '../api/client';
import type { Batch, NotificationItem } from '../types';

const NAV = [
  { to: '/student/dashboard', label: 'Dashboard', Icon: ChartBarIcon, title: 'Dashboard' },
  { to: '/student/courses', label: 'My Courses', Icon: BookOpenIcon, title: 'My Courses' },
  { to: '/student/assessments', label: 'Resources', Icon: ShieldCheckIcon, title: 'Resource Library' },
  { to: '/student/funds', label: 'Funds', Icon: BanknotesIcon, title: 'Fund Transparency' },
  { to: '/student/meetings', label: 'Video Meetings', Icon: VideoCameraIcon, title: 'Video Meetings' },
  { to: '/student/communication', label: 'Communication', Icon: ChatBubbleLeftIcon, title: 'Communication' },
  { to: '/student/gallery', label: 'Gallery', Icon: PhotoIcon, title: 'Gallery' },
];

const TITLES: Record<string, string> = {
  '/student/profile': 'Profile',
  '/student/notifications': 'Notifications',
};

export default function StudentLayout() {
  const { user, logout } = useAuthStore();
  const { batches, currentBatchID, setBatches, setCurrentBatchID } = useAppStore();
  const navigate = useNavigate();
  const location = useLocation();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [unread, setUnread] = useState(0);

  useEffect(() => {
    if (!sidebarOpen) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setSidebarOpen(false);
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [sidebarOpen]);

  useEffect(() => {
    api.get('/v1/batches').then((res) => setBatches(res.data as Batch[])).catch(() => {});
    api.get('/v1/me/notifications', { limit: 100 })
      .then((res) => setUnread(toList<NotificationItem>(res.data).filter((n) => !n.read_at).length))
      .catch(() => {});
  }, [setBatches]);

  const activeTitle =
    NAV.find((n) => location.pathname === n.to || location.pathname.startsWith(n.to + '/'))?.title ||
    TITLES[location.pathname] ||
    (location.pathname.startsWith('/public/') ? 'Public View' : 'Dashboard');

  const handleLogout = async () => {
    try {
      const refreshToken = localStorage.getItem('refresh_token');
      if (refreshToken) await api.post('/v1/auth/logout', { refresh_token: refreshToken });
    } catch {
      /* ignore */
    }
    logout();
    navigate('/login');
  };

  return (
    <div className="flex min-h-screen bg-obsidian-dark">
      <aside
        id="student-sidebar"
        className={`fixed inset-y-0 left-0 z-50 w-[300px] bg-charcoal-card border-r border-line flex flex-col transform transition-transform duration-200 ${
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        } lg:translate-x-0`}
      >
        <div className="px-7 pt-8 pb-6 border-b border-line">
          <p className="text-[32px] tracking-tight leading-tight font-bold text-gold-gradient">
            PMT Family
          </p>
          <p className="mt-1 text-xs font-medium text-muted uppercase tracking-widest">Student Portal</p>
        </div>
        <nav className="flex-1 overflow-y-auto px-4 py-5 space-y-1">
          {NAV.map(({ to, label, Icon }) => (
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
                onClick={() => navigate('/student/profile')}
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

      {sidebarOpen && <button type="button" aria-label="Close menu" className="fixed inset-0 z-40 bg-ink/30 lg:hidden cursor-default" onClick={() => setSidebarOpen(false)} />}

      <div className="flex-1 flex flex-col min-w-0 lg:pl-[300px]">
        <header className="sticky top-0 z-30 bg-charcoal-card/90 backdrop-blur-md border-b border-line">
          <div className="flex items-center justify-between px-4 sm:px-8 h-[104px]">
            <div className="flex items-center gap-4">
              <button onClick={() => setSidebarOpen(true)} className="lg:hidden text-ink" aria-label="Open menu" aria-expanded={sidebarOpen} aria-controls="student-sidebar">
                <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                </svg>
              </button>
              <h1 className="text-[22px] font-medium text-ink">{activeTitle}</h1>
            </div>
            <div className="flex items-center gap-4">
              <button onClick={() => navigate('/student/notifications')} className="relative p-2 text-ink hover:text-primary" aria-label={unread > 0 ? `Notifications, ${unread} unread` : 'Notifications'}>
                <BellIcon className="w-6 h-6" strokeWidth={1.7} aria-hidden="true" />
                {unread > 0 && <span className="absolute top-1.5 right-1.5 w-2.5 h-2.5 rounded-full bg-red-500" aria-hidden="true" />}
              </button>
              <select
                value={currentBatchID}
                onChange={(e) => setCurrentBatchID(e.target.value)}
                className="px-4 py-2.5 rounded-xl border border-line bg-slate-950/80 text-[15px] text-ink focus:outline-none focus:ring-2 focus:ring-amber-500/50 max-w-[220px]"
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
        <main className="flex-1 px-4 sm:px-8 py-8">
          <div className="max-w-[1400px] mx-auto">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
