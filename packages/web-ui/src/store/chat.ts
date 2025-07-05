// Chat state management
import { create } from 'zustand';
import type { MessageInfo, TextPart, FilePart } from '@/types/api';
import { api } from '@/api/client';

interface ChatState {
  // State
  messages: Record<string, MessageInfo[]>; // sessionId -> messages
  isStreaming: Record<string, boolean>; // sessionId -> streaming status
  selectedProvider: string | null;
  selectedModel: string | null;
  error: string | null;

  // Actions
  loadMessages: (sessionId: string) => Promise<void>;
  sendMessage: (sessionId: string, parts: (TextPart | FilePart)[]) => Promise<void>;
  updateMessage: (sessionId: string, message: MessageInfo) => void;
  setProvider: (providerId: string, modelId: string) => void;
  abortSession: (sessionId: string) => Promise<void>;
  clearMessages: (sessionId: string) => void;
}

export const useChatStore = create<ChatState>((set, get) => ({
  // Initial state
  messages: {},
  isStreaming: {},
  selectedProvider: null,
  selectedModel: null,
  error: null,

  // Actions
  loadMessages: async (sessionId) => {
    try {
      const messages = await api.message.list(sessionId);
      set(state => ({
        messages: { ...state.messages, [sessionId]: messages },
      }));
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to load messages' });
      throw error;
    }
  },

  sendMessage: async (sessionId, parts) => {
    const { selectedProvider, selectedModel } = get();
    if (!selectedProvider || !selectedModel) {
      throw new Error('No provider or model selected');
    }

    set(state => ({
      isStreaming: { ...state.isStreaming, [sessionId]: true },
      error: null,
    }));

    try {
      const message = await api.message.send(sessionId, {
        providerID: selectedProvider,
        modelID: selectedModel,
        parts,
      });

      // Add the message to the store
      set(state => ({
        messages: {
          ...state.messages,
          [sessionId]: [...(state.messages[sessionId] || []), message],
        },
      }));
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to send message' });
      throw error;
    } finally {
      set(state => ({
        isStreaming: { ...state.isStreaming, [sessionId]: false },
      }));
    }
  },

  updateMessage: (sessionId, message) => {
    set(state => {
      const messages = state.messages[sessionId] || [];
      const index = messages.findIndex(m => m.id === message.id);
      
      if (index === -1) {
        // Add new message
        return {
          messages: {
            ...state.messages,
            [sessionId]: [...messages, message],
          },
        };
      } else {
        // Update existing message
        const updated = [...messages];
        updated[index] = message;
        return {
          messages: {
            ...state.messages,
            [sessionId]: updated,
          },
        };
      }
    });

    // Check if streaming is complete
    if (message.metadata.time.completed) {
      set(state => ({
        isStreaming: { ...state.isStreaming, [sessionId]: false },
      }));
    }
  },

  setProvider: (providerId, modelId) => {
    set({ selectedProvider: providerId, selectedModel: modelId });
  },

  abortSession: async (sessionId) => {
    try {
      await api.session.abort(sessionId);
      set(state => ({
        isStreaming: { ...state.isStreaming, [sessionId]: false },
      }));
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Failed to abort session' });
      throw error;
    }
  },

  clearMessages: (sessionId) => {
    set(state => {
      const messages = { ...state.messages };
      delete messages[sessionId];
      return { messages };
    });
  },
}));