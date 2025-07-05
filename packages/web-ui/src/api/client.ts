// API client for communicating with the OpenCode server
import type {
  AppInfo,
  SessionInfo,
  MessageInfo,
  Provider,
  ConfigInfo,
  TextPart,
  FilePart,
} from '@/types/api';

const BASE_URL = '/api';

class ApiError extends Error {
  constructor(public status: number, message: string, public data?: any) {
    super(message);
    this.name = 'ApiError';
  }
}

async function request<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  const response = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Unknown error' }));
    throw new ApiError(response.status, error.message || 'Request failed', error);
  }

  return response.json();
}

export const api = {
  // App endpoints
  app: {
    info: () => request<AppInfo>('/app'),
    init: () => request<boolean>('/app/init', { method: 'POST' }),
  },

  // Config endpoints
  config: {
    get: () => request<ConfigInfo>('/config'),
    providers: () => request<{
      providers: Provider[];
      default: Record<string, string>;
    }>('/config/providers'),
  },

  // Session endpoints
  session: {
    list: () => request<SessionInfo[]>('/session'),
    create: () => request<SessionInfo>('/session', { method: 'POST' }),
    delete: (id: string) => request<boolean>(`/session/${id}`, { method: 'DELETE' }),
    share: (id: string) => request<SessionInfo>(`/session/${id}/share`, { method: 'POST' }),
    unshare: (id: string) => request<SessionInfo>(`/session/${id}/share`, { method: 'DELETE' }),
    abort: (id: string) => request<boolean>(`/session/${id}/abort`, { method: 'POST' }),
    
    init: (id: string, providerID: string, modelID: string) =>
      request<boolean>(`/session/${id}/init`, {
        method: 'POST',
        body: JSON.stringify({ providerID, modelID }),
      }),
    
    summarize: (id: string, providerID: string, modelID: string) =>
      request<boolean>(`/session/${id}/summarize`, {
        method: 'POST',
        body: JSON.stringify({ providerID, modelID }),
      }),
  },

  // Message endpoints
  message: {
    list: (sessionID: string) => request<MessageInfo[]>(`/session/${sessionID}/message`),
    
    send: (sessionID: string, params: {
      providerID: string;
      modelID: string;
      parts: (TextPart | FilePart)[];
    }) =>
      request<MessageInfo>(`/session/${sessionID}/message`, {
        method: 'POST',
        body: JSON.stringify(params),
      }),
  },

  // File search
  file: {
    search: (query: string) => 
      request<string[]>(`/file?query=${encodeURIComponent(query)}`),
  },
};