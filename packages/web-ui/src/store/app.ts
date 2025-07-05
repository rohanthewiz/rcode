// Global app state management
import { create } from 'zustand';
import type { AppInfo, Provider } from '@/types/api';
import { api } from '@/api/client';

interface AppState {
  // State
  appInfo: AppInfo | null;
  providers: Provider[];
  defaultModels: Record<string, string>;
  isInitialized: boolean;
  isLoading: boolean;
  error: string | null;

  // Actions
  initialize: () => Promise<void>;
  loadProviders: () => Promise<void>;
  setError: (error: string | null) => void;
  clearError: () => void;
}

export const useAppStore = create<AppState>((set) => ({
  // Initial state
  appInfo: null,
  providers: [],
  defaultModels: {},
  isInitialized: false,
  isLoading: false,
  error: null,

  // Actions
  initialize: async () => {
    set({ isLoading: true, error: null });
    try {
      const info = await api.app.info();
      set({ appInfo: info, isInitialized: true });
      
      // Load providers after getting app info
      const { providers, default: defaultModels } = await api.config.providers();
      set({ providers, defaultModels });
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to initialize app' });
      throw error;
    } finally {
      set({ isLoading: false });
    }
  },

  loadProviders: async () => {
    try {
      const { providers, default: defaultModels } = await api.config.providers();
      set({ providers, defaultModels });
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to load providers' });
      throw error;
    }
  },

  setError: (error) => set({ error }),
  clearError: () => set({ error: null }),
}));