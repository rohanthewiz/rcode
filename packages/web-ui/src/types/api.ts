// API types based on the server schema
// These are derived from the server's zod schemas

export interface AppInfo {
  version: string;
  git: boolean;
  path: {
    root: string;
    cwd: string;
    home: string;
  };
  permissions: Record<string, any>;
  installation: string;
  time: {
    opened: number;
    initialized: number;
  };
}

export interface SessionInfo {
  id: string;
  parentID?: string;
  share?: {
    url: string;
  };
  title: string;
  version: string;
  time: {
    created: number;
    updated: number;
  };
}

export interface Provider {
  id: string;
  name: string;
  icon: string;
  models: Record<string, Model>;
}

export interface Model {
  id: string;
  name: string;
  description?: string;
  cost: {
    input: number;
    output: number;
    cache_read?: number;
    cache_write?: number;
  };
  limit: {
    context?: number;
    output?: number;
  };
  temperature?: boolean;
  tool_call?: boolean;
  options?: Record<string, any>;
}

export type MessageRole = 'user' | 'assistant';

export interface MessagePart {
  type: 'text' | 'file' | 'tool-invocation' | 'step-start';
}

export interface TextPart extends MessagePart {
  type: 'text';
  text: string;
}

export interface FilePart extends MessagePart {
  type: 'file';
  mediaType: string;
  filename?: string;
  url: string;
}

export interface ToolInvocationPart extends MessagePart {
  type: 'tool-invocation';
  toolInvocation: ToolInvocation;
}

export interface StepStartPart extends MessagePart {
  type: 'step-start';
}

export type ToolInvocation = ToolCall | ToolPartialCall | ToolResult;

export interface ToolCall {
  state: 'call';
  step?: number;
  toolCallId: string;
  toolName: string;
  args: any;
}

export interface ToolPartialCall {
  state: 'partial-call';
  step?: number;
  toolCallId: string;
  toolName: string;
  args: any;
}

export interface ToolResult {
  state: 'result';
  step?: number;
  toolCallId: string;
  toolName: string;
  args: any;
  result: string;
}

export interface MessageInfo {
  id: string;
  role: MessageRole;
  parts: (TextPart | FilePart | ToolInvocationPart | StepStartPart)[];
  metadata: {
    time: {
      created: number;
      completed?: number;
    };
    error?: {
      name: string;
      data: Record<string, any>;
    };
    sessionID: string;
    tool: Record<string, {
      title: string;
      time: {
        start: number;
        end: number;
      };
      error?: boolean;
      message?: string;
      [key: string]: any;
    }>;
    assistant?: {
      system: string[];
      modelID: string;
      providerID: string;
      path: {
        cwd: string;
        root: string;
      };
      cost: number;
      summary?: boolean;
      tokens: {
        input: number;
        output: number;
        reasoning: number;
        cache: {
          read: number;
          write: number;
        };
      };
    };
  };
}

export interface ConfigInfo {
  [key: string]: any;
}

// Event types
export interface EventPayload {
  type: string;
  properties: any;
}

export interface SessionUpdatedEvent extends EventPayload {
  type: 'session.updated';
  properties: {
    info: SessionInfo;
  };
}

export interface SessionDeletedEvent extends EventPayload {
  type: 'session.deleted';
  properties: {
    info: SessionInfo;
  };
}

export interface SessionIdleEvent extends EventPayload {
  type: 'session.idle';
  properties: {
    sessionID: string;
  };
}

export interface SessionErrorEvent extends EventPayload {
  type: 'session.error';
  properties: {
    error: {
      name: string;
      data: Record<string, any>;
    };
  };
}

export interface MessageUpdatedEvent extends EventPayload {
  type: 'message.updated';
  properties: {
    info: MessageInfo;
  };
}

export interface MessagePartUpdatedEvent extends EventPayload {
  type: 'message.part.updated';
  properties: {
    part: MessagePart;
    sessionID: string;
    messageID: string;
  };
}

export type Event = 
  | SessionUpdatedEvent
  | SessionDeletedEvent
  | SessionIdleEvent
  | SessionErrorEvent
  | MessageUpdatedEvent
  | MessagePartUpdatedEvent;