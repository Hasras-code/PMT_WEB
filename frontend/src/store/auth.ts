import { create } from 'zustand';
import type { AccessContext, User } from '../types';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  loading: boolean;
  access: AccessContext | null;
  setUser: (user: User | null) => void;
  setLoading: (loading: boolean) => void;
  setAccess: (access: AccessContext | null) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: JSON.parse(localStorage.getItem('user') || 'null'),
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
}));
