import { create } from 'zustand';
import type { Batch } from '../types';

interface AppState {
  batches: Batch[];
  currentBatchID: string;
  setBatches: (batches: Batch[]) => void;
  setCurrentBatchID: (id: string) => void;
  currentBatch: () => Batch | undefined;
}

export const useAppStore = create<AppState>((set, get) => ({
  batches: [],
  currentBatchID: localStorage.getItem('current_batch_id') || '',
  setBatches: (batches) => {
    set({ batches });
    const { currentBatchID } = get();
    if (!currentBatchID && batches.length > 0) {
      localStorage.setItem('current_batch_id', batches[0].id);
      set({ currentBatchID: batches[0].id });
    }
    if (currentBatchID && !batches.some((b) => b.id === currentBatchID)) {
      const fallback = batches.length > 0 ? batches[0].id : '';
      localStorage.setItem('current_batch_id', fallback);
      set({ currentBatchID: fallback });
    }
  },
  setCurrentBatchID: (id) => {
    localStorage.setItem('current_batch_id', id);
    set({ currentBatchID: id });
  },
  currentBatch: () => get().batches.find((b) => b.id === get().currentBatchID),
}));
