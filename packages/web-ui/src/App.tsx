import { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { MainLayout } from './components/MainLayout';
import { ChatView } from './components/chat/ChatView';
import { useAppStore } from './store/app';
import { useSessionStore } from './store/session';
import { useChatStore } from './store/chat';
import { eventClient } from './api/events';
import type { Event } from './types/api';

function App() {
  const { initialize, isInitialized } = useAppStore();
  const { updateSession, sessions } = useSessionStore();
  const { updateMessage } = useChatStore();

  // Initialize app and connect to SSE
  useEffect(() => {
    initialize().catch(console.error);
    
    // Connect to SSE
    eventClient.connect();

    // Subscribe to events
    const unsubscribe = eventClient.subscribe((event: Event) => {
      switch (event.type) {
        case 'session.updated':
          updateSession(event.properties.info);
          break;
        case 'session.deleted':
          // Session deletion is handled by the session store
          break;
        case 'message.updated':
          updateMessage(event.properties.info.metadata.sessionID, event.properties.info);
          break;
        case 'message.part.updated':
          // Handle part updates if needed
          break;
        case 'session.idle':
          // Update streaming status
          break;
        case 'session.error':
          // Handle session errors
          console.error('Session error:', event.properties.error);
          break;
      }
    });

    return () => {
      unsubscribe();
      eventClient.disconnect();
    };
  }, []);

  if (!isInitialized) {
    return (
      <div className="flex items-center justify-center h-screen bg-background">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">Initializing OpenCode...</p>
        </div>
      </div>
    );
  }

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<MainLayout />}>
          <Route index element={<Navigate to="/chat" replace />} />
          <Route path="chat" element={<ChatView />} />
          <Route path="chat/:sessionId" element={<ChatView />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;