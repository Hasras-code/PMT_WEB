import { useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { api, toList } from '../api/client';
import { useAuthStore } from '../store/auth';
import { useAppStore } from '../store/app';
import type { Batch, Member } from '../types';

/** Returns true for platform admins and batch-role holders, false for plain students, null while checking. */
export function useElevated(): boolean | null {
  const { user } = useAuthStore();
  const [elevated, setElevated] = useState<boolean | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (useAppStore.getState().batches.length === 0) {
        try {
          const r = await api.get('/v1/batches');
          if (!cancelled) useAppStore.getState().setBatches(r.data as Batch[]);
        } catch {
          /* ignore */
        }
      }
      const list = useAppStore.getState().batches;
      try {
        await api.get('/v1/admin/users', { limit: 1 });
        if (!cancelled) setElevated(true);
        return;
      } catch {
        /* not a platform admin — check batch roles */
      }
      const me = useAuthStore.getState().user || user;
      if (!me) {
        if (!cancelled) setElevated(false);
        return;
      }
      for (const b of list) {
        try {
          const res = await api.get(`/v1/batches/${b.id}/members`, { limit: 100 });
          const mine = toList<Member>(res.data).find((m) => m.user_id === me.id);
          if (mine && mine.roles.some((r) => r !== 'STUDENT')) {
            if (!cancelled) setElevated(true);
            return;
          }
        } catch {
          /* no membership.view — keep checking */
        }
      }
      if (!cancelled) setElevated(false);
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  return elevated;
}

/** Returns the active portal base path: '/student' or '/admin'. */
export function usePortalBase(): string {
  const { pathname } = useLocation();
  return pathname.startsWith('/student') ? '/student' : '/admin';
}
