// Session state management
import { create } from 'zustand';
import type { SessionInfo } from '@/types/api';
import { api } from '@/api/client';

interface SessionState {
  // State
  sessions: SessionInfo[];
  activeSessionId: string | null;
  isLoading: boolean;
  error: string | null;

  // Computed
  activeSession: SessionInfo | null;

  // Actions
  loadSessions: () => Promise<void>;
  createSession: () => Promise<SessionInfo>;
  deleteSession: (id: string) => Promise<void>;
  setActiveSession: (id: string | null) => void;
  updateSession: (session: SessionInfo) => void;
  shareSession: (id: string) => Promise<void>;
  unshareSession: (id: string) => Promise<void>;
}

export const useSessionStore = create<SessionState>((set, get) => ({
  // Initial state
  sessions: [],
  activeSessionId: null,
  isLoading: false,
  error: null,

  // Computed
  get activeSession() {
    const state = get();
    return state.sessions.find(s => s.id === state.activeSessionId) || null;
  },

  // Actions
  loadSessions: async () => {
    set({ isLoading: true, error: null });
    try {
      const sessions = await api.session.list();
      // Sort by updated time, newest first
      sessions.sort((a, b) => b.time.updated - a.time.updated);
      set({ sessions });
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to load sessions' });
      throw error;
    } finally {
      set({ isLoading: false });
    }
  },

  createSession: async () => {
    set({ error: null });
    try {
      const newSession = await api.session.create();
      set(state => ({
        sessions: [newSession, ...state.sessions],
        activeSessionId: newSession.id,
      }));
      return newSession;
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to create session' });
      throw error;
    }
  },

  deleteSession: async (id: string) => {
    set({ error: null });
    try {
      await api.session.delete(id);
      set(state => {
        const sessions = state.sessions.filter(s => s.id !== id);
        const activeSessionId = state.activeSessionId === id 
          ? (sessions[0]?.id || null)
          : state.activeSessionId;
        return { sessions, activeSessionId };
      });
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to delete session' });
      throw error;
    }
  },

  setActiveSession: (id) => {
    set({ activeSessionId: id });
  },

  updateSession: (session) => {
    set(state => ({
      sessions: state.sessions.map(s => s.id === session.id ? session : s),
    }));
  },

  shareSession: async (id) => {
    try {
      const updated = await api.session.share(id);
      get().updateSession(updated);
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to share session' });
      throw error;
    }
  },

  unshareSession: async (id) => {
    try {
      const updated = await api.session.unshare(id);
      get().updateSession(updated);
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to unshare session' });
      throw error;
    }
  },
}));