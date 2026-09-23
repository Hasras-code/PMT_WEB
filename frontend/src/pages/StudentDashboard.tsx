import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  BookOpenIcon,
  VideoCameraIcon,
  BellIcon,
  MegaphoneIcon,
  BookmarkIcon,
} from '@heroicons/react/24/outline';
import { api, toList } from '../api/client';
import { useAuthStore } from '../store/auth';
import { useAppStore } from '../store/app';
import type { Announcement, Kuppi, NotificationItem } from '../types';
import { Card, CardTitle, PrimaryButton, OutlineButton, Badge, timeAgo, coverColor, initials, Skeleton } from '../components/ui';

const DOTS = ['bg-primary', 'bg-orange-500', 'bg-emerald-500', 'bg-purple-500'];

interface Bookmark {
  id: string;
  batch_id: string;
  title: string;
  type: string;
  created_at: string;
}

/** Student portal dashboard: personal learning overview. */
export default function StudentDashboard() {
  const navigate = useNavigate();
  const { user } = useAuthStore();
  const { batches } = useAppStore();
  const [kuppis, setKuppis] = useState<(Kuppi & { batchName: string })[]>([]);
  const [announcements, setAnnouncements] = useState<(Announcement & { batchName: string })[]>([]);
  const [bookmarks, setBookmarks] = useState<Bookmark[]>([]);
  const [unread, setUnread] = useState(0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    (async () => {
      const recordings: (Kuppi & { batchName: string })[] = [];
      const anns: (Announcement & { batchName: string })[] = [];
      await Promise.all(
        batches.map(async (b) => {
          try {
            const r = await api.get(`/v1/batches/${b.id}/kuppis`, { limit: 20 });
            toList<Kuppi>(r.data).forEach((kuppi) => recordings.push({ ...kuppi, batchName: b.name }));
          } catch { /* ignore */ }
          try {
            const r = await api.get(`/v1/batches/${b.id}/announcements`, { limit: 20 });
            toList<Announcement>(r.data).forEach((a) => anns.push({ ...a, batchName: b.name }));
          } catch { /* ignore */ }
        }),
      );
      try {
        const r = await api.get('/v1/me/bookmarks', { limit: 20 });
        if (!cancelled) setBookmarks(toList<Bookmark>(r.data));
      } catch { /* ignore */ }
      try {
        const r = await api.get('/v1/me/notifications', { limit: 100 });
        if (!cancelled) setUnread(toList<NotificationItem>(r.data).filter((n) => !n.read_at).length);
      } catch { /* ignore */ }
      if (cancelled) return;
      setLoading(false);
      setKuppis(
        recordings
          .filter((kuppi) => kuppi.status === 'PUBLISHED')
          .sort((a, b) => +new Date(b.recorded_at || b.published_at || b.created_at) - +new Date(a.recorded_at || a.published_at || a.created_at))
          .slice(0, 5),
      );
      setAnnouncements(anns.slice(0, 5));
    })();
    return () => {
      cancelled = true;
    };
  }, [batches]);

  const tiles = [
    { label: 'My Courses', value: String(batches.length), Icon: BookOpenIcon, tile: 'bg-blue-50 text-primary' },
    { label: 'Kuppis', value: String(kuppis.length), Icon: VideoCameraIcon, tile: 'bg-emerald-50 text-emerald-600' },
    { label: 'Announcements', value: String(announcements.length), Icon: MegaphoneIcon, tile: 'bg-purple-50 text-purple-600' },
    { label: 'Unread Notifications', value: String(unread), Icon: BellIcon, tile: 'bg-orange-50 text-orange-600' },
  ];

  return (
    <div className="space-y-7">
      <div className="bg-primary rounded-2xl px-8 py-9">
        <h1 className="text-[26px] font-bold text-white">Welcome back, {user?.display_name || 'Student'}!</h1>
        <p className="mt-2 text-white/85 text-[16px]">Continue learning — here is what is happening in your cohorts.</p>
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
          <CardTitle>My Courses</CardTitle>
          {batches.length === 0 ? (
            <p className="py-6 text-sm text-muted">You are not enrolled in any cohort yet. Ask a representative to approve your membership.</p>
          ) : (
            <div className="mt-4 space-y-3">
              {batches.slice(0, 4).map((b) => (
                <button key={b.id} onClick={() => navigate(`/student/courses/${b.id}`)} className="w-full flex items-center gap-4 p-3 rounded-xl border border-line hover:border-primary text-left transition-colors">
                  <span className="w-11 h-11 rounded-xl text-white font-bold flex items-center justify-center shrink-0" style={{ background: coverColor(b.slug) }}>
                    {initials(b.name)}
                  </span>
                  <span className="min-w-0">
                    <span className="block font-medium text-ink truncate">{b.name}</span>
                    <span className="block text-xs text-muted">Class of {b.entry_year}</span>
                  </span>
                </button>
              ))}
            </div>
          )}
        </Card>
        <Card className="p-7">
          <CardTitle>Latest Kuppis</CardTitle>
          <div className="mt-4 divide-y divide-line" role="status" aria-live="polite" aria-label="Latest Kuppis">
            {loading ? (
              <div className="space-y-3 py-2" aria-hidden="true">
                <Skeleton className="h-4 w-3/4" />
                <Skeleton className="h-4 w-1/2" />
                <Skeleton className="h-4 w-2/3" />
              </div>
            ) : (
              <>
                {kuppis.length === 0 && <p className="py-6 text-sm text-muted">No published Kuppis.</p>}
                {kuppis.map((m, i) => (
                  <div key={m.id} className="py-3.5 first:pt-1">
                    <div className="flex items-start gap-3">
                      <span className={`mt-1.5 w-2.5 h-2.5 rounded-full shrink-0 ${DOTS[i % DOTS.length]}`} />
                      <p className="text-[15px] text-ink leading-snug">{m.title}</p>
                    </div>
                    <p className="ml-[22px] mt-1 text-sm text-muted">{m.module.code} · {m.batchName}</p>
                  </div>
                ))}
              </>
            )}
          </div>
        </Card>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
        <Card className="p-7">
          <CardTitle>Latest Announcements</CardTitle>
          <div className="mt-4 divide-y divide-line" role="status" aria-live="polite" aria-label="Latest announcements">
            {loading ? (
              <div className="space-y-3 py-2" aria-hidden="true">
                <Skeleton className="h-4 w-2/3" />
                <Skeleton className="h-4 w-1/2" />
              </div>
            ) : (
              <>
                {announcements.length === 0 && <p className="py-6 text-sm text-muted">No announcements.</p>}
                {announcements.map((a) => (
                  <div key={a.id} className="py-3.5 first:pt-1">
                    <p className="text-[15px] font-medium text-ink">{a.title}</p>
                    <p className="text-sm text-muted line-clamp-1">{a.body}</p>
                    <p className="mt-1 text-xs text-muted">{a.batchName} · {timeAgo(a.created_at)}</p>
                  </div>
                ))}
              </>
            )}
          </div>
        </Card>
        <Card className="p-7">
          <div className="flex items-center gap-2">
            <BookmarkIcon className="w-5 h-5 text-muted" />
            <CardTitle>My Bookmarks</CardTitle>
          </div>
          <div className="mt-4 divide-y divide-line" role="status" aria-live="polite" aria-label="Bookmarked materials">
            {loading ? (
              <div className="space-y-3 py-2" aria-hidden="true">
                <Skeleton className="h-4 w-3/4" />
                <Skeleton className="h-4 w-1/2" />
              </div>
            ) : (
              <>
                {bookmarks.length === 0 && <p className="py-6 text-sm text-muted">No bookmarked materials. Open any assessment file and bookmark it.</p>}
                {bookmarks.slice(0, 5).map((b) => (
                  <div key={b.id} className="py-3 first:pt-1 flex items-center justify-between gap-3">
                    <p className="text-[15px] text-ink truncate">{b.title}</p>
                    <Badge tone="purple">{b.type.replaceAll('_', ' ')}</Badge>
                  </div>
                ))}
              </>
            )}
          </div>
          <div className="mt-5 space-y-4">
            <PrimaryButton onClick={() => navigate('/student/courses')}>Browse Courses</PrimaryButton>
            <OutlineButton onClick={() => navigate('/student/meetings')}>View Kuppis</OutlineButton>
            <OutlineButton onClick={() => navigate('/student/communication')}>Get Support</OutlineButton>
          </div>
        </Card>
      </div>
    </div>
  );
}
