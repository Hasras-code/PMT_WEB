import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { api } from '../api/client';
import { useAuthStore } from '../store/auth';
import type { AccessContext } from '../types';

export function useAccess(): AccessContext | null {
  const userID = useAuthStore((state) => state.user?.id);
  const access = useAuthStore((state) => state.access);
  const setAccess = useAuthStore((state) => state.setAccess);

  useEffect(() => {
    let cancelled = false;
    if (!userID) {
      setAccess(null);
      return;
    }
    if (access) return;
    api.get('/v1/me/access')
      .then((response) => {
        if (!cancelled) setAccess(response.data as AccessContext);
      })
      .catch(() => {
        if (!cancelled) setAccess({ platform_roles: [], platform_permissions: [], memberships: [] });
      });
    return () => { cancelled = true; };
  }, [access, setAccess, userID]);
  return access;
}

/** Returns true for platform-role holders or users with a non-student batch role. */
export function useElevated(): boolean | null {
  const access = useAccess();
  if (!access) return null;
  return access.platform_roles.length > 0 || access.memberships.some((membership) => membership.roles.some((role) => role !== 'STUDENT'));
}

export function useCan(permission: string, batchID?: string): boolean | null {
  const access = useAccess();
  if (!access) return null;
  if (access.platform_permissions.includes(permission)) return true;
  if (!batchID) return access.memberships.some((membership) => membership.permissions.includes(permission));
  return access.memberships.find((membership) => membership.batch_id === batchID)?.permissions.includes(permission) ?? false;
}

/** Returns the active portal base path: '/student' or '/admin'. */
export function usePortalBase(): string {
  const { pathname } = useLocation();
  return pathname.startsWith('/student') ? '/student' : '/admin';
}
