import { create } from 'zustand';
import { clearSession } from '../api/client';
import type { AccessContext, User } from '../types';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  loading: boolean;
  access: AccessContext | null;
  setUser: (user: User | null) => void;
  setLoading: (loading: boolean) => void;
  setAccess: (access: AccessContext | null) => void;
  logout: () => void;
}

function readStoredUser(): User | null {
  try {
    const raw = localStorage.getItem('user');
    return raw ? (JSON.parse(raw) as User) : null;
  } catch {
    localStorage.removeItem('user');
    return null;
  }
}

export const useAuthStore = create<AuthState>((set) => ({
  user: readStoredUser(),
  isAuthenticated: !!localStorage.getItem('access_token'),
  loading: false,
  access: null,
  setUser: (user) => {
    set((state) => ({ user, isAuthenticated: !!user, access: state.user?.id === user?.id ? state.access : null }));
    if (user) {
      localStorage.setItem('user', JSON.stringify(user));
    } else {
      localStorage.removeItem('user');
    }
  },
  setLoading: (loading) => set({ loading }),
  setAccess: (access) => set({ access }),
  logout: () => {
    clearSession();
    set({ user: null, isAuthenticated: false, access: null });
  },
}));
